// Package ogmiosutil holds Ogmios v6 JSON encoders and decoders shared by the
// backends that talk to Ogmios directly or through a provider proxy.
package ogmiosutil

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strings"

	"github.com/SundaeSwap-finance/ogmigo/v6/ouroboros/chainsync/num"
	"github.com/SundaeSwap-finance/ogmigo/v6/ouroboros/shared"
	"github.com/blinklabs-io/gouroboros/ledger/common"

	"github.com/Salvionied/apollo/v2/backend"
	"github.com/Salvionied/apollo/v2/internal/backendutil"
)

// UtxosToShared converts resolved gouroboros UTxOs into the ogmigo
// shared.Utxo wire form expected by EvaluateTxWithAdditionalUtxos.
func UtxosToShared(utxos []common.Utxo) ([]shared.Utxo, error) {
	result := make([]shared.Utxo, 0, len(utxos))
	for _, utxo := range utxos {
		su, err := UtxoToShared(utxo)
		if err != nil {
			return nil, err
		}
		result = append(result, su)
	}
	return result, nil
}

// UtxoToShared converts a single resolved gouroboros UTxO into an
// ogmigo shared.Utxo. The value is encoded as Ogmios expects: the outer key is
// "ada" (with inner key "lovelace") for the coin, and the policy ID hex (with
// inner asset-name hex) for native assets.
func UtxoToShared(utxo common.Utxo) (shared.Utxo, error) {
	if err := backend.ValidateAdditionalUtxo(utxo); err != nil {
		return shared.Utxo{}, err
	}

	out := utxo.Output

	coin, err := bigIntToNum(out.Amount())
	if err != nil {
		return shared.Utxo{}, fmt.Errorf("invalid lovelace amount: %w", err)
	}
	value := shared.Value{
		shared.AdaPolicy: {
			shared.AdaAsset: coin,
		},
	}
	if assets := out.Assets(); assets != nil {
		for _, policyId := range assets.Policies() {
			policyHex := hex.EncodeToString(policyId.Bytes())
			for _, assetName := range assets.Assets(policyId) {
				qty, err := bigIntToNum(assets.Asset(policyId, assetName))
				if err != nil {
					return shared.Utxo{}, fmt.Errorf("invalid asset quantity for policy %s: %w", policyHex, err)
				}
				if value[policyHex] == nil {
					value[policyHex] = map[string]num.Int{}
				}
				value[policyHex][hex.EncodeToString(assetName)] = qty
			}
		}
	}

	su := shared.Utxo{
		Transaction: shared.UtxoTxID{ID: hex.EncodeToString(utxo.Id.Id().Bytes())},
		Index:       utxo.Id.Index(),
		Address:     out.Address().String(),
		Value:       value,
	}

	// Datum: inline datum CBOR hex goes in Datum, a bare datum hash in DatumHash.
	if datum := out.Datum(); datum != nil {
		datumCbor, err := datum.MarshalCBOR()
		if err != nil {
			return shared.Utxo{}, fmt.Errorf("failed to encode inline datum: %w", err)
		}
		su.Datum = hex.EncodeToString(datumCbor)
	} else if datumHash := out.DatumHash(); datumHash != nil {
		su.DatumHash = hex.EncodeToString(datumHash.Bytes())
	}

	// Reference script: Ogmios expects {"language": ..., "cbor": ...}.
	if script := out.ScriptRef(); script != nil {
		scriptJSON, err := ScriptRefJSON(script)
		if err != nil {
			return shared.Utxo{}, err
		}
		su.Script = scriptJSON
	}

	return su, nil
}

// bigIntToNum converts a big.Int quantity into the ogmigo num.Int used by
// shared.Value, preserving the full magnitude (no int64 truncation).
func bigIntToNum(v *big.Int) (num.Int, error) {
	if v == nil {
		return num.Int64(0), nil
	}
	n, ok := num.New(v.String())
	if !ok {
		return num.Int{}, fmt.Errorf("cannot represent quantity %s", v.String())
	}
	return n, nil
}

// ScriptRefJSON encodes a reference script as the Ogmios script JSON
// object ({"language": "plutus:vN"|"native", "cbor": "<hex>"}) matching the
// shape consumed by ogmiosScriptToScriptRef.
func ScriptRefJSON(script common.Script) (json.RawMessage, error) {
	var language string
	switch script.(type) {
	case common.PlutusV1Script:
		language = "plutus:v1"
	case common.PlutusV2Script:
		language = "plutus:v2"
	case common.PlutusV3Script:
		language = "plutus:v3"
	case common.PlutusV4Script:
		language = "plutus:v4"
	case common.NativeScript:
		language = "native"
	default:
		return nil, fmt.Errorf("unsupported reference script type %T", script)
	}
	payload := struct {
		Language string `json:"language"`
		Cbor     string `json:"cbor"`
	}{
		Language: language,
		Cbor:     hex.EncodeToString(script.RawScriptBytes()),
	}
	return json.Marshal(payload)
}

// ParseEvaluationResult parses the Ogmios v6 evaluateTransaction result
// array: [{"validator":{"purpose":...,"index":...},"budget":{"memory":...,"cpu":...}}].
func ParseEvaluationResult(raw json.RawMessage) (map[common.RedeemerKey]common.ExUnits, error) {
	var items []struct {
		Validator struct {
			Purpose string `json:"purpose"`
			Index   uint64 `json:"index"`
		} `json:"validator"`
		Budget struct {
			Memory uint64 `json:"memory"`
			Cpu    uint64 `json:"cpu"`
		} `json:"budget"`
		Error json.RawMessage `json:"error"`
	}
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, fmt.Errorf("failed to parse evaluation result: %w", err)
	}
	if len(items) == 0 {
		return nil, errors.New("script evaluation returned no results")
	}
	result := make(map[common.RedeemerKey]common.ExUnits, len(items))
	for _, item := range items {
		if jsonValuePresent(item.Error) {
			return nil, fmt.Errorf("script evaluation failed for validator %s:%d: %s",
				item.Validator.Purpose, item.Validator.Index, string(item.Error))
		}
		if item.Validator.Purpose == "" {
			return nil, fmt.Errorf("malformed evaluation result entry: %s", evalErrorSnippet(raw))
		}
		tag, err := backendutil.ParseRedeemerTag(item.Validator.Purpose)
		if err != nil {
			return nil, fmt.Errorf("invalid redeemer purpose %q: %w", item.Validator.Purpose, err)
		}
		if item.Validator.Index > math.MaxUint32 {
			return nil, fmt.Errorf("redeemer index %d exceeds uint32 range", item.Validator.Index)
		}
		if item.Budget.Memory > math.MaxInt64 || item.Budget.Cpu > math.MaxInt64 {
			return nil, fmt.Errorf("ExUnits overflow for validator %s:%d: memory=%d cpu=%d",
				item.Validator.Purpose, item.Validator.Index, item.Budget.Memory, item.Budget.Cpu)
		}
		key := common.RedeemerKey{Tag: tag, Index: uint32(item.Validator.Index)}
		result[key] = common.ExUnits{Memory: int64(item.Budget.Memory), Steps: int64(item.Budget.Cpu)}
	}
	return result, nil
}

// maxErrorSnippetSize bounds how much of a provider payload is quoted in an
// error message.
const maxErrorSnippetSize = 512

// jsonValuePresent reports whether a raw JSON field was present and not null.
func jsonValuePresent(raw json.RawMessage) bool {
	trimmed := strings.TrimSpace(string(raw))
	return trimmed != "" && trimmed != "null"
}

// evalErrorSnippet bounds a response payload for inclusion in error messages.
func evalErrorSnippet(data []byte) string {
	if len(data) > maxErrorSnippetSize {
		data = data[:maxErrorSnippetSize]
	}
	return string(data)
}

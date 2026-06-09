package maestro

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strconv"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/babbage"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/mary"
	"github.com/blinklabs-io/gouroboros/ledger/shelley"
	maestroClient "github.com/maestro-org/go-sdk/client"
	"github.com/maestro-org/go-sdk/models"
	"github.com/maestro-org/go-sdk/utils"

	"github.com/Salvionied/apollo/v2/backend"
)

// MaestroChainContext implements backend.ChainContext using the Maestro API.
type MaestroChainContext struct {
	client    *maestroClient.Client
	networkId uint8
}

// NewMaestroChainContext creates a new Maestro chain context.
func NewMaestroChainContext(networkId uint8, projectId string) (*MaestroChainContext, error) {
	networkStr := networkString(networkId)
	client := maestroClient.NewClient(projectId, networkStr)
	return &MaestroChainContext{
		client:    client,
		networkId: networkId,
	}, nil
}

// networkString maps Cardano network ID to Maestro API network name.
// In Cardano, network ID 1 = mainnet, network ID 0 = testnet.
// For testnet, defaults to "preprod". Use NewMaestroChainContextWithNetwork
// for other testnet variants (e.g. "preview").
func networkString(networkId uint8) string {
	if networkId == 1 {
		return "mainnet"
	}
	return "preprod"
}

// NewMaestroChainContextWithNetwork creates a Maestro chain context with an explicit network name.
// Use this for testnet variants like "preview" where the network ID alone (0) is ambiguous.
func NewMaestroChainContextWithNetwork(networkId uint8, projectId string, network string) (*MaestroChainContext, error) {
	client := maestroClient.NewClient(projectId, network)
	return &MaestroChainContext{
		client:    client,
		networkId: networkId,
	}, nil
}

func (m *MaestroChainContext) ProtocolParams() (backend.ProtocolParameters, error) {
	resp, err := m.client.ProtocolParameters()
	if err != nil {
		return backend.ProtocolParameters{}, err
	}

	data := resp.Data
	priceMem, err := backend.ParseFraction(data.ScriptExecutionPrices.Memory)
	if err != nil {
		return backend.ProtocolParameters{}, fmt.Errorf("invalid memory price: %w", err)
	}
	priceStep, err := backend.ParseFraction(data.ScriptExecutionPrices.Steps)
	if err != nil {
		return backend.ProtocolParameters{}, fmt.Errorf("invalid step price: %w", err)
	}

	pp := backend.ProtocolParameters{
		MinFeeCoefficient:   data.MinFeeCoefficient,
		MinFeeConstant:      data.MinFeeConstant.LovelaceAmount.Lovelace,
		MaxBlockSize:        int(data.MaxBlockBodySize.Bytes),
		MaxTxSize:           int(data.MaxTransactionSize.Bytes),
		MaxBlockHeaderSize:  int(data.MaxBlockHeaderSize.Bytes),
		KeyDeposits:         strconv.FormatInt(data.StakeCredentialDeposit.LovelaceAmount.Lovelace, 10),
		PoolDeposits:        strconv.FormatInt(data.StakePoolDeposit.LovelaceAmount.Lovelace, 10),
		MinPoolCost:         strconv.FormatInt(data.MinStakePoolCost.LovelaceAmount.Lovelace, 10),
		MaxTxExMem:          strconv.FormatInt(data.MaxExecutionUnitsPerTransaction.Memory, 10),
		MaxTxExSteps:        strconv.FormatInt(data.MaxExecutionUnitsPerTransaction.Steps, 10),
		MaxBlockExMem:       strconv.FormatInt(data.MaxExecutionUnitsPerBlock.Memory, 10),
		MaxBlockExSteps:     strconv.FormatInt(data.MaxExecutionUnitsPerBlock.Steps, 10),
		MaxValSize:          strconv.FormatInt(data.MaxValueSize.Bytes, 10),
		CollateralPercent:   int(data.CollateralPercentage),
		MaxCollateralInputs: int(data.MaxCollateralInputs),
		CoinsPerUtxoByte:    strconv.FormatInt(data.MinUtxoDepositCoefficient, 10),
		PriceMem:            priceMem,
		PriceStep:           priceStep,
	}

	// Parse cost models from Maestro response.
	// PlutusCostModels is typed as `any`; when unmarshaled from JSON it is
	// map[string]interface{} with keys like "plutus:v1", "plutus:v2", "plutus:v3"
	// and values that are []interface{} of float64.
	// ComputeScriptDataHash expects keys "PlutusV1", "PlutusV2", "PlutusV3".
	if rawModels, ok := data.PlutusCostModels.(map[string]any); ok {
		pp.CostModels = make(map[string][]int64, len(rawModels))
		for key, val := range rawModels {
			costs, ok := val.([]any)
			if !ok {
				return backend.ProtocolParameters{}, fmt.Errorf("unexpected cost model format for %s: expected []any, got %T", key, val)
			}
			int64Costs := make([]int64, 0, len(costs))
			for i, c := range costs {
				f, ok := c.(float64)
				if !ok {
					return backend.ProtocolParameters{}, fmt.Errorf("cost model %q element %d: expected float64, got %T", key, i, c)
				}
				int64Costs = append(int64Costs, int64(f))
			}
			pp.CostModels[maestroCostModelKey(key)] = int64Costs
		}
	}

	return pp, nil
}

func (m *MaestroChainContext) GenesisParams() (backend.GenesisParameters, error) {
	return backend.GenesisParameters{}, errors.New("genesis params not available via Maestro API")
}

func (m *MaestroChainContext) NetworkId() uint8 {
	return m.networkId
}

func (m *MaestroChainContext) CurrentEpoch() (uint64, error) {
	resp, err := m.client.CurrentEpoch()
	if err != nil {
		return 0, err
	}
	if resp.Data.EpochNo < 0 {
		return 0, fmt.Errorf("invalid epoch value: %d", resp.Data.EpochNo)
	}
	return uint64(resp.Data.EpochNo), nil
}

func (m *MaestroChainContext) MaxTxFee() (uint64, error) {
	pp, err := m.ProtocolParams()
	if err != nil {
		return 0, err
	}
	return backend.ComputeMaxTxFee(pp)
}

func (m *MaestroChainContext) Tip() (uint64, error) {
	resp, err := m.client.ChainTip()
	if err != nil {
		return 0, err
	}
	if resp.Data.Slot < 0 {
		return 0, fmt.Errorf("invalid slot value: %d", resp.Data.Slot)
	}
	return uint64(resp.Data.Slot), nil
}

func (m *MaestroChainContext) Utxos(address common.Address) ([]common.Utxo, error) {
	const maxPages = 1000
	var allUtxos []common.Utxo
	params := utils.NewParameters()
	var lastCursor string

	for range maxPages {
		resp, err := m.client.UtxosAtAddress(address.String(), params)
		if err != nil {
			return nil, err
		}

		for _, raw := range resp.Data {
			utxo, err := maestroUtxoToCommon(raw, address)
			if err != nil {
				return nil, fmt.Errorf("failed to parse UTxO: %w", err)
			}
			allUtxos = append(allUtxos, utxo)
		}

		lastCursor = resp.NextCursor
		if lastCursor == "" {
			break
		}
		params = utils.NewParameters()
		params.Cursor(lastCursor)
	}

	if lastCursor != "" {
		return nil, fmt.Errorf("UTxO pagination exceeded %d pages; results may be incomplete", maxPages)
	}

	return allUtxos, nil
}

func (m *MaestroChainContext) SubmitTx(txCbor []byte) (common.Blake2b256, error) {
	txHex := hex.EncodeToString(txCbor)
	resp, err := m.client.SubmitTx(txHex)
	if err != nil {
		return common.Blake2b256{}, err
	}
	hashBytes, err := hex.DecodeString(resp.Data)
	if err != nil {
		return common.Blake2b256{}, err
	}
	if len(hashBytes) != common.Blake2b256Size {
		return common.Blake2b256{}, fmt.Errorf("invalid tx hash length: expected %d bytes, got %d", common.Blake2b256Size, len(hashBytes))
	}
	var result common.Blake2b256
	copy(result[:], hashBytes)
	return result, nil
}

func (m *MaestroChainContext) EvaluateTx(txCbor []byte) (map[common.RedeemerKey]common.ExUnits, error) {
	txHex := hex.EncodeToString(txCbor)
	evalResp, err := m.client.EvaluateTx(txHex)
	if err != nil {
		return nil, err
	}

	result := make(map[common.RedeemerKey]common.ExUnits)
	for _, eval := range evalResp {
		if eval.RedeemerIndex < 0 {
			return nil, fmt.Errorf("negative redeemer index: %d", eval.RedeemerIndex)
		}
		if eval.RedeemerIndex > math.MaxUint32 {
			return nil, fmt.Errorf("redeemer index %d exceeds uint32 range", eval.RedeemerIndex)
		}
		tag, err := backend.ParseRedeemerTag(eval.RedeemerTag)
		if err != nil {
			return nil, fmt.Errorf("invalid redeemer tag %q: %w", eval.RedeemerTag, err)
		}
		key := common.RedeemerKey{Tag: tag, Index: uint32(eval.RedeemerIndex)}
		result[key] = common.ExUnits{Memory: eval.ExUnits.Mem, Steps: eval.ExUnits.Steps}
	}
	return result, nil
}

func (m *MaestroChainContext) UtxoByRef(txHash common.Blake2b256, index uint32) (*common.Utxo, error) {
	hashHex := hex.EncodeToString(txHash.Bytes())
	resp, err := m.client.TransactionOutputFromReference(hashHex, int(index), nil)
	if err != nil {
		return nil, err
	}

	addr, err := common.NewAddress(resp.Data.Address)
	if err != nil {
		return nil, err
	}
	utxo, err := maestroUtxoToCommon(resp.Data, addr)
	if err != nil {
		return nil, err
	}
	return &utxo, nil
}

func (m *MaestroChainContext) ScriptCbor(scriptHash common.Blake2b224) ([]byte, error) {
	hashHex := hex.EncodeToString(scriptHash.Bytes())
	resp, err := m.client.ScriptByHash(hashHex)
	if err != nil {
		return nil, err
	}
	if resp.Data.Bytes == "" {
		return nil, errors.New("no script CBOR available")
	}
	return hex.DecodeString(resp.Data.Bytes)
}

func maestroUtxoToCommon(raw models.Utxo, address common.Address) (common.Utxo, error) {
	hashBytes, err := hex.DecodeString(raw.TxHash)
	if err != nil {
		return common.Utxo{}, err
	}
	if len(hashBytes) != common.Blake2b256Size {
		return common.Utxo{}, fmt.Errorf("invalid tx hash length: expected %d bytes, got %d", common.Blake2b256Size, len(hashBytes))
	}
	var txId common.Blake2b256
	copy(txId[:], hashBytes)

	if raw.Index < 0 {
		return common.Utxo{}, fmt.Errorf("negative output index: %d", raw.Index)
	}
	if raw.Index > math.MaxUint32 {
		return common.Utxo{}, fmt.Errorf("output index %d exceeds uint32 range", raw.Index)
	}
	input := shelley.ShelleyTransactionInput{
		TxId:        txId,
		OutputIndex: uint32(raw.Index),
	}

	var lovelace uint64
	assetData := make(map[common.Blake2b224]map[cbor.ByteString]*big.Int)

	for _, asset := range raw.Assets {
		if asset.Unit == "lovelace" {
			if asset.Amount < 0 {
				return common.Utxo{}, fmt.Errorf("negative lovelace amount: %d", asset.Amount)
			}
			lovelace = uint64(asset.Amount) //nolint:gosec // validated non-negative above
		} else {
			if asset.Amount < 0 {
				return common.Utxo{}, fmt.Errorf("negative asset amount %d for unit %s", asset.Amount, asset.Unit)
			}
			policyId, assetName, err := backend.ParseAssetUnit(asset.Unit)
			if err != nil {
				return common.Utxo{}, fmt.Errorf("invalid asset unit %q: %w", asset.Unit, err)
			}

			if _, ok := assetData[policyId]; !ok {
				assetData[policyId] = make(map[cbor.ByteString]*big.Int)
			}
			assetData[policyId][assetName] = big.NewInt(asset.Amount)
		}
	}

	var assets *common.MultiAsset[common.MultiAssetTypeOutput]
	if len(assetData) > 0 {
		ma := common.NewMultiAsset[common.MultiAssetTypeOutput](assetData)
		assets = &ma
	}

	output := babbage.BabbageTransactionOutput{
		OutputAddress: address,
		OutputAmount: mary.MaryTransactionOutputValue{
			Amount: lovelace,
			Assets: assets,
		},
	}

	// Map datum to output's DatumOption.
	// Maestro returns the datum field as a JSON object with keys "type", "hash",
	// "bytes", "json". When unmarshaled into `any` it becomes map[string]interface{}.
	if datumMap, ok := raw.Datum.(map[string]any); ok {
		if datumCborHex, ok := datumMap["bytes"].(string); ok && datumCborHex != "" {
			// Inline datum: "bytes" field contains the CBOR hex of the datum.
			datumBytes, err := hex.DecodeString(datumCborHex)
			if err != nil {
				return common.Utxo{}, fmt.Errorf("invalid inline datum CBOR hex %q: %w", datumCborHex, err)
			}
			cborBytes, err := cbor.Encode([]any{1, cbor.Tag{Number: 24, Content: datumBytes}})
			if err != nil {
				return common.Utxo{}, fmt.Errorf("failed to encode inline datum option: %w", err)
			}
			var opt babbage.BabbageTransactionOutputDatumOption
			if err := opt.UnmarshalCBOR(cborBytes); err != nil {
				return common.Utxo{}, fmt.Errorf("failed to unmarshal inline datum option: %w", err)
			}
			output.DatumOption = &opt
		} else if hashHex, ok := datumMap["hash"].(string); ok && hashHex != "" {
			// Datum hash reference only.
			hashBytes, err := hex.DecodeString(hashHex)
			if err != nil {
				return common.Utxo{}, fmt.Errorf("invalid datum hash hex %q: %w", hashHex, err)
			}
			if len(hashBytes) != common.Blake2b256Size {
				return common.Utxo{}, fmt.Errorf("invalid datum hash length: expected %d bytes, got %d", common.Blake2b256Size, len(hashBytes))
			}
			var hash common.Blake2b256
			copy(hash[:], hashBytes)
			cborBytes, err := cbor.Encode([]any{0, hash})
			if err != nil {
				return common.Utxo{}, fmt.Errorf("failed to encode datum option hash: %w", err)
			}
			var opt babbage.BabbageTransactionOutputDatumOption
			if err := opt.UnmarshalCBOR(cborBytes); err != nil {
				return common.Utxo{}, fmt.Errorf("failed to unmarshal datum option: %w", err)
			}
			output.DatumOption = &opt
		}
	}

	// Parse reference script if present.
	if raw.ReferenceScript.Bytes != "" {
		scriptBytes, err := hex.DecodeString(raw.ReferenceScript.Bytes)
		if err != nil {
			return common.Utxo{}, fmt.Errorf("invalid reference script hex: %w", err)
		}
		ref, err := maestroScriptRef(raw.ReferenceScript.Type, scriptBytes)
		if err != nil {
			return common.Utxo{}, fmt.Errorf("failed to parse reference script: %w", err)
		}
		output.TxOutScriptRef = ref
	}

	return common.Utxo{
		Id:     input,
		Output: &output,
	}, nil
}

// maestroScriptRef builds a ScriptRef from the Maestro script type and CBOR bytes.
func maestroScriptRef(scriptType string, scriptCbor []byte) (*common.ScriptRef, error) {
	switch scriptType {
	case "native":
		var native common.NativeScript
		if _, err := cbor.Decode(scriptCbor, &native); err != nil {
			return nil, fmt.Errorf("failed to decode native script: %w", err)
		}
		return &common.ScriptRef{Type: common.ScriptRefTypeNativeScript, Script: native}, nil
	case "plutusv1":
		return &common.ScriptRef{Type: common.ScriptRefTypePlutusV1, Script: common.PlutusV1Script(scriptCbor)}, nil
	case "plutusv2":
		return &common.ScriptRef{Type: common.ScriptRefTypePlutusV2, Script: common.PlutusV2Script(scriptCbor)}, nil
	case "plutusv3":
		return &common.ScriptRef{Type: common.ScriptRefTypePlutusV3, Script: common.PlutusV3Script(scriptCbor)}, nil
	default:
		return nil, fmt.Errorf("unknown script type %q", scriptType)
	}
}

// maestroCostModelKey translates Maestro cost model keys to the canonical form
// expected by ComputeScriptDataHash ("PlutusV1", "PlutusV2", "PlutusV3").
func maestroCostModelKey(key string) string {
	switch key {
	case "plutus:v1":
		return "PlutusV1"
	case "plutus:v2":
		return "PlutusV2"
	case "plutus:v3":
		return "PlutusV3"
	default:
		return key
	}
}

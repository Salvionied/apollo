package ogmios

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strconv"

	"github.com/SundaeSwap-finance/kugo"
	ogmigo "github.com/SundaeSwap-finance/ogmigo/v6"
	"github.com/SundaeSwap-finance/ogmigo/v6/ouroboros/chainsync"
	"github.com/SundaeSwap-finance/ogmigo/v6/ouroboros/shared"
	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/babbage"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/mary"
	"github.com/blinklabs-io/gouroboros/ledger/shelley"

	"github.com/Salvionied/apollo/v2/backend"
)

// OgmiosChainContext implements backend.ChainContext using Ogmios + Kupo.
type OgmiosChainContext struct {
	ogmios    *ogmigo.Client
	kupo      *kugo.Client
	networkId uint8
}

// NewOgmiosChainContext creates a new Ogmios chain context.
func NewOgmiosChainContext(ogmiosClient *ogmigo.Client, kupoClient *kugo.Client, networkId uint8) *OgmiosChainContext {
	return &OgmiosChainContext{
		ogmios:    ogmiosClient,
		kupo:      kupoClient,
		networkId: networkId,
	}
}

func (o *OgmiosChainContext) ProtocolParams() (backend.ProtocolParameters, error) {
	ctx := context.Background()
	raw, err := o.ogmios.CurrentProtocolParameters(ctx)
	if err != nil {
		return backend.ProtocolParameters{}, err
	}

	var params ogmiosProtocolParams
	if err := json.Unmarshal(raw, &params); err != nil {
		return backend.ProtocolParameters{}, fmt.Errorf("failed to parse protocol params: %w", err)
	}

	return params.toProtocolParams()
}

func (o *OgmiosChainContext) GenesisParams() (backend.GenesisParameters, error) {
	ctx := context.Background()
	raw, err := o.ogmios.GenesisConfig(ctx, "shelley")
	if err != nil {
		return backend.GenesisParameters{}, err
	}

	var genesis ogmiosGenesisConfig
	if err := json.Unmarshal(raw, &genesis); err != nil {
		return backend.GenesisParameters{}, err
	}

	return genesis.toGenesisParams(), nil
}

func (o *OgmiosChainContext) NetworkId() uint8 {
	return o.networkId
}

func (o *OgmiosChainContext) CurrentEpoch() (uint64, error) {
	ctx := context.Background()
	return o.ogmios.CurrentEpoch(ctx)
}

func (o *OgmiosChainContext) MaxTxFee() (uint64, error) {
	pp, err := o.ProtocolParams()
	if err != nil {
		return 0, err
	}
	return backend.ComputeMaxTxFee(pp)
}

func (o *OgmiosChainContext) Tip() (uint64, error) {
	ctx := context.Background()
	point, err := o.ogmios.ChainTip(ctx)
	if err != nil {
		return 0, err
	}
	ps, ok := point.PointStruct()
	if !ok || ps == nil {
		return 0, errors.New("chain tip is origin")
	}
	return ps.Slot, nil
}

func (o *OgmiosChainContext) Utxos(address common.Address) ([]common.Utxo, error) {
	if o.kupo == nil {
		return nil, errors.New("kupo client required for UTxO lookup")
	}
	ctx := context.Background()
	matches, err := o.kupo.Matches(ctx, kugo.OnlyUnspent(), kugo.Address(address.String()))
	if err != nil {
		return nil, err
	}

	var utxos []common.Utxo
	for _, match := range matches {
		utxo, err := matchToUtxo(ctx, match, address, o.kupo)
		if err != nil {
			return nil, fmt.Errorf("failed to parse UTxO match: %w", err)
		}
		utxos = append(utxos, utxo)
	}
	return utxos, nil
}

func (o *OgmiosChainContext) SubmitTx(txCbor []byte) (common.Blake2b256, error) {
	ctx := context.Background()
	txHex := hex.EncodeToString(txCbor)
	resp, err := o.ogmios.SubmitTx(ctx, txHex)
	if err != nil {
		return common.Blake2b256{}, err
	}
	if resp.Error != nil {
		return common.Blake2b256{}, fmt.Errorf("submit tx error: %s", resp.Error.Message)
	}
	hashBytes, err := hex.DecodeString(resp.ID)
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

func (o *OgmiosChainContext) EvaluateTx(txCbor []byte) (map[common.RedeemerKey]common.ExUnits, error) {
	ctx := context.Background()
	txHex := hex.EncodeToString(txCbor)
	resp, err := o.ogmios.EvaluateTx(ctx, txHex)
	if err != nil {
		return nil, err
	}
	return evaluateResponseToExUnits(resp)
}

// evaluateResponseToExUnits converts an ogmigo EvaluateTxResponse into a
// redeemer ExUnits map. A response with zero evaluation results is an error:
// returning an empty map with a nil error would let callers silently keep
// zero execution budgets for their redeemers.
func evaluateResponseToExUnits(resp *ogmigo.EvaluateTxResponse) (map[common.RedeemerKey]common.ExUnits, error) {
	if resp == nil {
		return nil, errors.New("empty evaluate response")
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("evaluate tx error: %s", resp.Error.Message)
	}
	if len(resp.ExUnits) == 0 {
		return nil, errors.New("script evaluation returned no results")
	}

	result := make(map[common.RedeemerKey]common.ExUnits, len(resp.ExUnits))
	for _, eu := range resp.ExUnits {
		tag, err := backend.ParseRedeemerTag(eu.Validator.Purpose)
		if err != nil {
			return nil, fmt.Errorf("invalid redeemer purpose %q: %w", eu.Validator.Purpose, err)
		}
		if eu.Validator.Index > math.MaxUint32 {
			return nil, fmt.Errorf("redeemer index %d exceeds uint32 range", eu.Validator.Index)
		}
		key := common.RedeemerKey{Tag: tag, Index: uint32(eu.Validator.Index)}
		if eu.Budget.Memory > math.MaxInt64 || eu.Budget.Cpu > math.MaxInt64 {
			return nil, fmt.Errorf("ExUnits overflow: memory=%d cpu=%d", eu.Budget.Memory, eu.Budget.Cpu)
		}
		result[key] = common.ExUnits{Memory: int64(eu.Budget.Memory), Steps: int64(eu.Budget.Cpu)}
	}
	return result, nil
}

func (o *OgmiosChainContext) UtxoByRef(txHash common.Blake2b256, index uint32) (*common.Utxo, error) {
	ctx := context.Background()
	hashHex := hex.EncodeToString(txHash.Bytes())
	query := chainsync.TxInQuery{
		Transaction: shared.UtxoTxID{ID: hashHex},
		Index:       index,
	}
	utxos, err := o.ogmios.UtxosByTxIn(ctx, query)
	if err != nil {
		return nil, err
	}
	if len(utxos) == 0 {
		return nil, errors.New("utxo not found")
	}

	raw := utxos[0]
	addr, err := common.NewAddress(raw.Address)
	if err != nil {
		return nil, err
	}
	result, err := ogmiosUtxoToCommon(raw, addr)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (o *OgmiosChainContext) ScriptCbor(scriptHash common.Blake2b224) ([]byte, error) {
	if o.kupo == nil {
		return nil, errors.New("kupo client required for script lookup")
	}
	ctx := context.Background()
	hashHex := hex.EncodeToString(scriptHash.Bytes())
	script, err := o.kupo.Script(ctx, hashHex)
	if err != nil {
		return nil, err
	}
	return hex.DecodeString(script.Script)
}

// --- Ogmios response types and conversion ---

type ogmiosProtocolParams struct {
	MinFeeCoefficient  int64           `json:"minFeeCoefficient"`
	MinFeeConstant     ogmiosLovelace  `json:"minFeeConstant"`
	MaxBlockBodySize   ogmiosBytes     `json:"maxBlockBodySize"`
	MaxBlockHeaderSize ogmiosBytes     `json:"maxBlockHeaderSize"`
	MaxTxSize          ogmiosBytes     `json:"maxTransactionSize"`
	StakeKeyDeposit    ogmiosLovelace  `json:"stakeCredentialDeposit"`
	PoolDeposit        ogmiosLovelace  `json:"stakePoolDeposit"`
	MinPoolCost        ogmiosLovelace  `json:"minStakePoolCost"`
	CollateralPercent  int             `json:"collateralPercentage"`
	MaxCollateral      int             `json:"maxCollateralInputs"`
	MaxValSize         ogmiosBytes     `json:"maxValueSize"`
	ScriptPrices       ogmiosPrices    `json:"scriptExecutionPrices"`
	MaxTxExUnits       ogmiosExUnits   `json:"maxExecutionUnitsPerTransaction"`
	MaxBlockExUnits    ogmiosExUnits   `json:"maxExecutionUnitsPerBlock"`
	MinUtxoDeposit     int64           `json:"minUtxoDepositCoefficient"`
	CostModels         json.RawMessage `json:"plutusCostModels"`
}

type ogmiosLovelace struct {
	Lovelace int64 `json:"lovelace"`
}

type ogmiosBytes struct {
	Bytes int `json:"bytes"`
}

type ogmiosPrices struct {
	Memory string `json:"memory"`
	CPU    string `json:"cpu"`
}

type ogmiosExUnits struct {
	Memory int64 `json:"memory"`
	CPU    int64 `json:"cpu"`
}

func (p *ogmiosProtocolParams) toProtocolParams() (backend.ProtocolParameters, error) {
	priceMem, err := backend.ParseFraction(p.ScriptPrices.Memory)
	if err != nil {
		return backend.ProtocolParameters{}, fmt.Errorf("invalid memory price: %w", err)
	}
	priceStep, err := backend.ParseFraction(p.ScriptPrices.CPU)
	if err != nil {
		return backend.ProtocolParameters{}, fmt.Errorf("invalid CPU price: %w", err)
	}

	pp := backend.ProtocolParameters{
		MinFeeConstant:      p.MinFeeConstant.Lovelace,
		MinFeeCoefficient:   p.MinFeeCoefficient,
		MaxBlockSize:        p.MaxBlockBodySize.Bytes,
		MaxTxSize:           p.MaxTxSize.Bytes,
		MaxBlockHeaderSize:  p.MaxBlockHeaderSize.Bytes,
		KeyDeposits:         strconv.FormatInt(p.StakeKeyDeposit.Lovelace, 10),
		PoolDeposits:        strconv.FormatInt(p.PoolDeposit.Lovelace, 10),
		MinPoolCost:         strconv.FormatInt(p.MinPoolCost.Lovelace, 10),
		PriceMem:            priceMem,
		PriceStep:           priceStep,
		MaxTxExMem:          strconv.FormatInt(p.MaxTxExUnits.Memory, 10),
		MaxTxExSteps:        strconv.FormatInt(p.MaxTxExUnits.CPU, 10),
		MaxBlockExMem:       strconv.FormatInt(p.MaxBlockExUnits.Memory, 10),
		MaxBlockExSteps:     strconv.FormatInt(p.MaxBlockExUnits.CPU, 10),
		MaxValSize:          strconv.Itoa(p.MaxValSize.Bytes),
		CollateralPercent:   p.CollateralPercent,
		MaxCollateralInputs: p.MaxCollateral,
		CoinsPerUtxoByte:    strconv.FormatInt(p.MinUtxoDeposit, 10),
	}

	// Parse cost models from Ogmios JSON.
	// Ogmios uses keys like "plutus:v1", "plutus:v2", "plutus:v3".
	// ComputeScriptDataHash expects "PlutusV1", "PlutusV2", "PlutusV3".
	if len(p.CostModels) > 0 {
		var rawModels map[string][]int64
		if err := json.Unmarshal(p.CostModels, &rawModels); err != nil {
			return backend.ProtocolParameters{}, fmt.Errorf("failed to parse cost models: %w", err)
		}
		pp.CostModels = make(map[string][]int64, len(rawModels))
		for key, costs := range rawModels {
			pp.CostModels[ogmiosCostModelKey(key)] = costs
		}
	}

	return pp, nil
}

// ogmiosCostModelKey translates Ogmios cost model keys to the canonical form
// expected by ComputeScriptDataHash ("PlutusV1", "PlutusV2", "PlutusV3").
func ogmiosCostModelKey(key string) string {
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

type ogmiosGenesisConfig struct {
	NetworkMagic      int     `json:"networkMagic"`
	EpochLength       int     `json:"epochLength"`
	SlotLength        int     `json:"slotLength"`
	SlotsPerKesPeriod int     `json:"slotsPerKesPeriod"`
	MaxKesEvolutions  int     `json:"maxKESEvolutions"`
	SecurityParam     int     `json:"securityParameter"`
	UpdateQuorum      int     `json:"updateQuorum"`
	ActiveSlots       float64 `json:"activeSlotsCoefficient"`
	MaxLovelaceSupply int64   `json:"maxLovelaceSupply"`
}

func (g *ogmiosGenesisConfig) toGenesisParams() backend.GenesisParameters {
	return backend.GenesisParameters{
		ActiveSlotsCoefficient: g.ActiveSlots,
		UpdateQuorum:           g.UpdateQuorum,
		NetworkMagic:           g.NetworkMagic,
		EpochLength:            g.EpochLength,
		MaxLovelaceSupply:      strconv.FormatInt(g.MaxLovelaceSupply, 10),
		SlotLength:             g.SlotLength,
		SlotsPerKesPeriod:      g.SlotsPerKesPeriod,
		MaxKesEvolutions:       g.MaxKesEvolutions,
		SecurityParam:          g.SecurityParam,
	}
}

// datumFetcher resolves a datum's CBOR (hex-encoded) by its hash. It is
// implemented by *kugo.Client via the Kupo /v1/datums/{hash} endpoint.
type datumFetcher interface {
	Datum(ctx context.Context, datumHash string) (string, error)
}

func matchToUtxo(ctx context.Context, match kugo.Match, address common.Address, datums datumFetcher) (common.Utxo, error) {
	hashBytes, err := hex.DecodeString(match.TransactionID)
	if err != nil {
		return common.Utxo{}, err
	}
	if len(hashBytes) != common.Blake2b256Size {
		return common.Utxo{}, fmt.Errorf("invalid tx hash length: expected %d bytes, got %d", common.Blake2b256Size, len(hashBytes))
	}
	var txId common.Blake2b256
	copy(txId[:], hashBytes)
	if match.OutputIndex < 0 {
		return common.Utxo{}, fmt.Errorf("negative output index: %d", match.OutputIndex)
	}
	if match.OutputIndex > math.MaxUint32 {
		return common.Utxo{}, fmt.Errorf("output index %d exceeds uint32 range", match.OutputIndex)
	}
	utxo, err := sharedValueToUtxo(txId, uint32(match.OutputIndex), shared.Value(match.Value), address)
	if err != nil {
		return common.Utxo{}, err
	}
	output, ok := utxo.Output.(*babbage.BabbageTransactionOutput)
	if !ok {
		return common.Utxo{}, fmt.Errorf("unexpected UTxO output type: %T", utxo.Output)
	}

	// Set datum option from kupo match data. Kupo only returns the datum hash
	// in matches; its datum_type discriminator says whether the on-chain
	// output carried an inline datum or just the hash.
	if match.DatumHash != "" {
		switch match.DatumType {
		case "inline":
			opt, err := fetchInlineDatumOption(ctx, datums, match.DatumHash)
			if err != nil {
				return common.Utxo{}, err
			}
			output.DatumOption = opt
		case "hash":
			opt, err := parseDatumOption(match.DatumHash)
			if err != nil {
				return common.Utxo{}, fmt.Errorf("failed to parse datum option: %w", err)
			}
			output.DatumOption = opt
		default:
			return common.Utxo{}, fmt.Errorf("unsupported kupo datum type %q for datum hash %s", match.DatumType, match.DatumHash)
		}
	}

	// Set script reference from kupo match data, verifying the script bytes
	// against the script hash claimed by kupo.
	if match.Script.Script != "" {
		ref, err := kupoScriptToScriptRef(match.Script, match.ScriptHash)
		if err != nil {
			return common.Utxo{}, fmt.Errorf("failed to parse script ref: %w", err)
		}
		output.TxOutScriptRef = ref
	}

	return utxo, nil
}

// fetchInlineDatumOption fetches the inline datum bytes for the given datum
// hash from Kupo and builds an inline datum option. The fetched bytes are
// verified against the datum hash before use; a mismatch fails closed.
func fetchInlineDatumOption(ctx context.Context, datums datumFetcher, datumHashHex string) (*babbage.BabbageTransactionOutputDatumOption, error) {
	if datums == nil {
		return nil, fmt.Errorf("kupo client required to resolve inline datum %s", datumHashHex)
	}
	expectedBytes, err := hex.DecodeString(datumHashHex)
	if err != nil {
		return nil, fmt.Errorf("invalid datum hash hex %q: %w", datumHashHex, err)
	}
	if len(expectedBytes) != common.Blake2b256Size {
		return nil, fmt.Errorf("invalid datum hash length: expected %d bytes, got %d", common.Blake2b256Size, len(expectedBytes))
	}
	var expected common.Blake2b256
	copy(expected[:], expectedBytes)

	datumCborHex, err := datums.Datum(ctx, datumHashHex)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch inline datum %s: %w", datumHashHex, err)
	}
	if datumCborHex == "" {
		return nil, fmt.Errorf("kupo returned no datum for inline datum hash %s", datumHashHex)
	}
	datumBytes, err := hex.DecodeString(datumCborHex)
	if err != nil {
		return nil, fmt.Errorf("invalid inline datum CBOR hex %q: %w", datumCborHex, err)
	}
	if computed := common.Blake2b256Hash(datumBytes); computed != expected {
		return nil, fmt.Errorf("inline datum hash mismatch for %s: fetched datum hashes to %s",
			datumHashHex, hex.EncodeToString(computed.Bytes()))
	}
	return parseInlineDatumCbor(datumCborHex)
}

func ogmiosUtxoToCommon(raw shared.Utxo, addr common.Address) (common.Utxo, error) {
	hashBytes, err := hex.DecodeString(raw.Transaction.ID)
	if err != nil {
		return common.Utxo{}, err
	}
	if len(hashBytes) != common.Blake2b256Size {
		return common.Utxo{}, fmt.Errorf("invalid tx hash length: expected %d bytes, got %d", common.Blake2b256Size, len(hashBytes))
	}
	var txId common.Blake2b256
	copy(txId[:], hashBytes)
	utxo, err := sharedValueToUtxo(txId, raw.Index, raw.Value, addr)
	if err != nil {
		return common.Utxo{}, err
	}
	output, ok := utxo.Output.(*babbage.BabbageTransactionOutput)
	if !ok {
		return common.Utxo{}, fmt.Errorf("unexpected UTxO output type: %T", utxo.Output)
	}

	// Set datum option from ogmios UTxO data.
	// Ogmios provides inline datum CBOR hex in the Datum field,
	// and the datum hash in DatumHash.
	if raw.Datum != "" {
		// Inline datum: Datum field contains the CBOR hex of the datum.
		opt, err := parseInlineDatumCbor(raw.Datum)
		if err != nil {
			return common.Utxo{}, fmt.Errorf("failed to parse inline datum: %w", err)
		}
		output.DatumOption = opt
	} else if raw.DatumHash != "" {
		// Datum hash only.
		opt, err := parseDatumOption(raw.DatumHash)
		if err != nil {
			return common.Utxo{}, fmt.Errorf("failed to parse datum hash: %w", err)
		}
		output.DatumOption = opt
	}

	// Set script reference from ogmios UTxO data.
	if len(raw.Script) > 0 && string(raw.Script) != "null" {
		ref, err := ogmiosScriptToScriptRef(raw.Script)
		if err != nil {
			return common.Utxo{}, fmt.Errorf("failed to parse script ref: %w", err)
		}
		if ref != nil {
			output.TxOutScriptRef = ref
		}
	}

	return utxo, nil
}

func sharedValueToUtxo(txId common.Blake2b256, outputIndex uint32, value shared.Value, addr common.Address) (common.Utxo, error) {
	input := shelley.ShelleyTransactionInput{
		TxId:        txId,
		OutputIndex: outputIndex,
	}

	// Require int64 range (not just uint64) to match the other backends and
	// keep downstream signed lovelace arithmetic safe.
	lovelaceBig := value.AdaLovelace().BigInt()
	if lovelaceBig.Sign() < 0 || !lovelaceBig.IsInt64() {
		return common.Utxo{}, fmt.Errorf("invalid lovelace quantity %s", lovelaceBig.String())
	}
	lovelace := lovelaceBig.Uint64()
	assetData := make(map[common.Blake2b224]map[cbor.ByteString]*big.Int)

	for policyIdStr, assets := range value {
		if policyIdStr == "ada" {
			continue
		}
		policyBytes, err := hex.DecodeString(policyIdStr)
		if err != nil {
			return common.Utxo{}, fmt.Errorf("invalid policy ID hex %q: %w", policyIdStr, err)
		}
		if len(policyBytes) != common.Blake2b224Size {
			return common.Utxo{}, fmt.Errorf("invalid policy ID length for %q: expected %d bytes, got %d", policyIdStr, common.Blake2b224Size, len(policyBytes))
		}
		var policyId common.Blake2b224
		copy(policyId[:], policyBytes)

		for assetName, qty := range assets {
			qtyBig := qty.BigInt()
			if qtyBig.Sign() < 0 {
				return common.Utxo{}, fmt.Errorf("negative asset quantity %s for policy %s asset %s", qtyBig.String(), policyIdStr, assetName)
			}
			nameBytes, err := hex.DecodeString(assetName)
			if err != nil {
				return common.Utxo{}, fmt.Errorf("invalid asset name hex %q: %w (asset names must be hex-encoded)", assetName, err)
			}
			if _, ok := assetData[policyId]; !ok {
				assetData[policyId] = make(map[cbor.ByteString]*big.Int)
			}
			assetData[policyId][cbor.NewByteString(nameBytes)] = new(big.Int).Set(qtyBig)
		}
	}

	var assets *common.MultiAsset[common.MultiAssetTypeOutput]
	if len(assetData) > 0 {
		ma := common.NewMultiAsset[common.MultiAssetTypeOutput](assetData)
		assets = &ma
	}

	output := babbage.BabbageTransactionOutput{
		OutputAddress: addr,
		OutputAmount: mary.MaryTransactionOutputValue{
			Amount: lovelace,
			Assets: assets,
		},
	}

	return common.Utxo{
		Id:     input,
		Output: &output,
	}, nil
}

// parseDatumOption constructs a BabbageTransactionOutputDatumOption from a datum hash hex string.
// It always creates a datum hash reference (type 0). For inline datums, use parseInlineDatumCbor
// which requires the full datum CBOR.
func parseDatumOption(datumHashHex string) (*babbage.BabbageTransactionOutputDatumOption, error) {
	hashBytes, err := hex.DecodeString(datumHashHex)
	if err != nil {
		return nil, fmt.Errorf("invalid datum hash hex %q: %w", datumHashHex, err)
	}
	if len(hashBytes) != common.Blake2b256Size {
		return nil, fmt.Errorf("invalid datum hash length: expected %d bytes, got %d", common.Blake2b256Size, len(hashBytes))
	}
	var hash common.Blake2b256
	copy(hash[:], hashBytes)

	cborBytes, err := cbor.Encode([]any{0, hash})
	if err != nil {
		return nil, fmt.Errorf("failed to encode datum option: %w", err)
	}
	var opt babbage.BabbageTransactionOutputDatumOption
	if err := opt.UnmarshalCBOR(cborBytes); err != nil {
		return nil, fmt.Errorf("failed to unmarshal datum option: %w", err)
	}
	return &opt, nil
}

// parseInlineDatumCbor constructs a BabbageTransactionOutputDatumOption for an inline datum
// from its CBOR hex representation (as returned by Ogmios).
func parseInlineDatumCbor(datumCborHex string) (*babbage.BabbageTransactionOutputDatumOption, error) {
	datumBytes, err := hex.DecodeString(datumCborHex)
	if err != nil {
		return nil, fmt.Errorf("invalid datum CBOR hex %q: %w", datumCborHex, err)
	}
	// Inline datum option: [1, #6.24(datum_cbor)]
	cborBytes, err := cbor.Encode([]any{1, cbor.Tag{Number: 24, Content: datumBytes}})
	if err != nil {
		return nil, fmt.Errorf("failed to encode inline datum option: %w", err)
	}
	var opt babbage.BabbageTransactionOutputDatumOption
	if err := opt.UnmarshalCBOR(cborBytes); err != nil {
		return nil, fmt.Errorf("failed to unmarshal inline datum option: %w", err)
	}
	return &opt, nil
}

// kupoScriptToScriptRef converts a kugo Script to a common.ScriptRef. When
// kupo supplies the script hash (expectedHashHex non-empty), the script bytes
// are verified against it rather than trusted as-is.
func kupoScriptToScriptRef(script kugo.Script, expectedHashHex string) (*common.ScriptRef, error) {
	scriptBytes, err := hex.DecodeString(script.Script)
	if err != nil {
		return nil, fmt.Errorf("invalid script hex %q: %w", script.Script, err)
	}

	var scriptType uint
	switch script.Language {
	case kugo.ScriptLanguageNative:
		scriptType = common.ScriptRefTypeNativeScript
	case kugo.ScriptLanguagePlutusV1:
		scriptType = common.ScriptRefTypePlutusV1
	case kugo.ScriptLanguagePlutusV2:
		scriptType = common.ScriptRefTypePlutusV2
	case kugo.ScriptLanguagePlutusV3:
		scriptType = common.ScriptRefTypePlutusV3
	default:
		return nil, fmt.Errorf("unsupported kupo script language: %d", script.Language)
	}

	return backend.ScriptRefFromBytes(scriptType, scriptBytes, expectedHashHex)
}

// ogmiosScriptToScriptRef converts an Ogmios script JSON to a common.ScriptRef.
// Ogmios v6 uses: {"language": "plutus:v1"|"plutus:v2"|"plutus:v3"|"native", "cbor": "hex"}
// Ogmios does not include the script hash in UTxO responses, so no hash
// verification is possible here.
func ogmiosScriptToScriptRef(scriptJSON json.RawMessage) (*common.ScriptRef, error) {
	var raw struct {
		Language string `json:"language"`
		Cbor     string `json:"cbor"`
	}
	if err := json.Unmarshal(scriptJSON, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse script JSON: %w", err)
	}
	if raw.Cbor == "" {
		// Native scripts may use "json" field instead of "cbor"; skip these for now.
		return nil, nil
	}

	scriptBytes, err := hex.DecodeString(raw.Cbor)
	if err != nil {
		return nil, fmt.Errorf("invalid script CBOR hex %q: %w", raw.Cbor, err)
	}

	var scriptType uint
	switch raw.Language {
	case "native":
		scriptType = common.ScriptRefTypeNativeScript
	case "plutus:v1":
		scriptType = common.ScriptRefTypePlutusV1
	case "plutus:v2":
		scriptType = common.ScriptRefTypePlutusV2
	case "plutus:v3":
		scriptType = common.ScriptRefTypePlutusV3
	default:
		return nil, fmt.Errorf("unsupported ogmios script language %q", raw.Language)
	}

	return backend.ScriptRefFromBytes(scriptType, scriptBytes, "")
}

package OgmiosChainContext

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/SundaeSwap-finance/apollo/serialization"
	"github.com/SundaeSwap-finance/apollo/serialization/Address"
	"github.com/SundaeSwap-finance/apollo/serialization/Amount"
	"github.com/SundaeSwap-finance/apollo/serialization/Asset"
	"github.com/SundaeSwap-finance/apollo/serialization/AssetName"
	"github.com/SundaeSwap-finance/apollo/serialization/MultiAsset"
	"github.com/SundaeSwap-finance/apollo/serialization/PlutusData"
	"github.com/SundaeSwap-finance/apollo/serialization/Policy"
	"github.com/SundaeSwap-finance/apollo/serialization/Redeemer"
	"github.com/SundaeSwap-finance/apollo/serialization/Transaction"
	"github.com/SundaeSwap-finance/apollo/serialization/TransactionInput"
	"github.com/SundaeSwap-finance/apollo/serialization/TransactionOutput"
	"github.com/SundaeSwap-finance/apollo/serialization/UTxO"
	"github.com/SundaeSwap-finance/apollo/serialization/Value"
	"github.com/SundaeSwap-finance/apollo/txBuilding/Backend/Base"
	"github.com/SundaeSwap-finance/kugo"
	"github.com/SundaeSwap-finance/ogmigo/v6"
	"github.com/SundaeSwap-finance/ogmigo/v6/ouroboros/chainsync"
	"github.com/SundaeSwap-finance/ogmigo/v6/ouroboros/chainsync/num"
	"github.com/SundaeSwap-finance/ogmigo/v6/ouroboros/shared"

	"github.com/Salvionied/cbor/v2"
)

type OgmiosChainContext struct {
	_epoch_info     Base.Epoch
	_epoch          int
	_Network        int
	_genesis_param  Base.GenesisParameters
	_protocol_param Base.ProtocolParameters
	ogmigo          ogmigo.Client
	kugo            kugo.Client
}

func NewOgmiosChainContext(ogmigoClient ogmigo.Client, kugoClient kugo.Client) OgmiosChainContext {
	occ := OgmiosChainContext{
		ogmigo: ogmigoClient,
		kugo:   kugoClient,
	}
	return occ
}

func (occ *OgmiosChainContext) Init() {
	latest_epochs := occ.LatestEpoch()
	occ._epoch_info = latest_epochs
	//Init Genesis
	params := occ.GenesisParams()
	occ._genesis_param = params
	//init epoch
	latest_params := occ.LatestEpochParams()
	occ._protocol_param = latest_params
}

func multiAsset_OgmigoToApollo(m map[string]map[string]num.Int) MultiAsset.MultiAsset[int64] {
	if len(m) == 0 {
		return nil
	}
	assetMap := make(map[Policy.PolicyId]Asset.Asset[int64])
	for policy, tokens := range m {
		tokensMap := make(map[AssetName.AssetName]int64)
		for token, amt := range tokens {
			tok := *AssetName.NewAssetNameFromHexString(token)
			tokensMap[tok] = amt.Int64()
		}
		pol := Policy.PolicyId{
			Value: policy,
		}
		assetMap[pol] = make(map[AssetName.AssetName]int64)
		assetMap[pol] = tokensMap
	}
	return assetMap
}

func multiAsset_ApolloToOgmigo(ma MultiAsset.MultiAsset[int64]) map[string]map[string]num.Int {
	assetMap := make(map[string]map[string]num.Int)
	for policy, asset := range ma {
		tokensMap := make(map[string]num.Int)
		for assetName, amount := range asset {
			tokensMap[assetName.HexString()] = num.Int64(amount)
		}
		assetMap[policy.Value] = tokensMap
	}
	return assetMap
}

func value_OgmigoToApollo(v shared.Value) Value.AlonzoValue {
	val := v.AssetsExceptAda()
	ass := multiAsset_OgmigoToApollo(val)
	if ass == nil {
		return Value.AlonzoValue{
			Am:        Amount.AlonzoAmount{},
			Coin:      v.AdaLovelace().Int64(),
			HasAssets: false,
		}
	}
	return Value.AlonzoValue{
		Am: Amount.AlonzoAmount{
			Coin:  v.AdaLovelace().Int64(),
			Value: ass,
		},
		Coin:      0,
		HasAssets: true,
	}
}

func value_ApolloToOgmigo(v Value.AlonzoValue) shared.Value {
	if v.HasAssets {
		result := multiAsset_ApolloToOgmigo(v.Am.Value)
		result["ada"] = make(map[string]num.Int)
		result["ada"]["lovelace"] = num.Int64(v.Am.Coin)
		return result
	} else {
		result := make(map[string]map[string]num.Int)
		result["ada"] = make(map[string]num.Int)
		result["ada"]["lovelace"] = num.Int64(v.Coin)
		return result
	}
}

func datum_OgmigoToApollo(d string, dh string) *PlutusData.DatumOption {
	if d != "" {
		datumBytes, err := hex.DecodeString(d)
		if err != nil {
			log.Fatal(err, "OgmiosChainContext: Failed to decode datum from hex: %v", d)
		}
		var pd PlutusData.PlutusData
		err = cbor.Unmarshal(datumBytes, &pd)
		if err != nil {
			log.Fatal(err, "OgmiosChainContext: datum is not valid plutus data: %v", d)
		}
		res := PlutusData.DatumOptionInline(&pd)
		return &res
	}
	if dh != "" {
		datumHashBytes, err := hex.DecodeString(dh)
		if err != nil {
			log.Fatal(err, "OgmiosChainContext: Failed to decode datum hash from hex: %v", dh)
		}
		res := PlutusData.DatumOptionHash(datumHashBytes)
		return &res
	}
	return nil
}

func datum_ApolloToOgmigo(pd *PlutusData.DatumOption) (string, string, error) {
	switch pd.DatumType {
	case PlutusData.DatumTypeHash:
		return "", hex.EncodeToString(pd.Hash), nil
	case PlutusData.DatumTypeInline:
		bytes, err := cbor.Marshal(pd.Inline)
		if err != nil {
			return "", "", err
		}
		return hex.EncodeToString(bytes), "", nil
	default:
		return "", "", fmt.Errorf("datum_ApolloToOgmigo: unknown type tag for DatumOption: %v", pd.DatumType)
	}
}

func scriptRef_OgmigoToApollo(script json.RawMessage) (*PlutusData.ScriptRef, error) {
	if len(script) == 0 {
		return nil, nil
	}
	var ref struct {
		Language string
		Cbor     string
	}
	if err := json.Unmarshal(script, &ref); err != nil {
		return nil, err
	}
	if ref.Language == "" {
		return nil, fmt.Errorf("can't parse script ref (missing 'language') '%s'", string(script))
	}
	if ref.Cbor == "" {
		return nil, fmt.Errorf("can't parse script ref (missing 'cbor') '%s'", string(script))
	}
	cborRaw, err := hex.DecodeString(ref.Cbor)
	if err != nil {
		return nil, err
	}
	return &PlutusData.ScriptRef{
		Script: PlutusData.InnerScript{
			Script: cborRaw,
		},
	}, nil
}

func scriptRef_ApolloToOgmigo(script *PlutusData.ScriptRef) (json.RawMessage, error) {
	if script == nil {
		return nil, nil
	}
	enc, err := json.Marshal(script)
	if err != nil {
		return nil, err
	}
	return enc, nil
}

func Utxo_OgmigoToApollo(u shared.Utxo) UTxO.UTxO {
	txHashRaw, err := hex.DecodeString(u.Transaction.ID)
	if err != nil {
		log.Fatal(err, "Failed to decode ogmigo transaction ID")
	}
	addr, err := Address.DecodeAddress(u.Address)
	if err != nil {
		log.Fatal(err, "Failed to decode ogmigo address")
	}
	datum := datum_OgmigoToApollo(u.Datum, u.DatumHash)
	v := value_OgmigoToApollo(u.Value)
	scriptRef, err := scriptRef_OgmigoToApollo(u.Script)
	if err != nil {
		log.Fatal(err, "Failed to convert script ref from ogmigo")
	}
	return UTxO.UTxO{
		Input: TransactionInput.TransactionInput{
			TransactionId: txHashRaw,
			Index:         int(u.Index),
		},
		Output: TransactionOutput.TransactionOutput{
			PostAlonzo: TransactionOutput.TransactionOutputAlonzo{
				Address:   addr,
				Amount:    v,
				Datum:     datum,
				ScriptRef: scriptRef,
			},
			PreAlonzo:    TransactionOutput.TransactionOutputShelley{},
			IsPostAlonzo: true,
		},
	}
}

func Utxo_ApolloToOgmigo(u UTxO.UTxO) shared.Utxo {
	amount := value_ApolloToOgmigo(u.Output.GetValue().ToAlonzoValue())
	datum, datumHash, err := datum_ApolloToOgmigo(u.Output.GetDatumOption())
	if err != nil {
		log.Fatal(err, "Failed to convert apollo datum object to ogmigo format")
	}
	scriptRef, err := scriptRef_ApolloToOgmigo(u.Output.GetScriptRef())
	if err != nil {
		log.Fatal(err, "Failed to convert apollo script ref to ogmigo format")
	}
	return shared.Utxo{
		Transaction: shared.UtxoTxID{
			ID: hex.EncodeToString(u.Input.TransactionId),
		},
		Index:     uint32(u.Input.Index),
		Address:   u.Output.GetAddress().String(),
		Value:     amount,
		Datum:     datum,
		DatumHash: datumHash,
		Script:    scriptRef,
	}
}

func (occ *OgmiosChainContext) GetUtxoFromRef(txHash string, index int) (UTxO.UTxO, error) {
	ctx := context.Background()
	utxos, err := occ.ogmigo.UtxosByTxIn(ctx, chainsync.TxInQuery{
		Transaction: shared.UtxoTxID{
			ID: txHash,
		},
		Index: uint32(index),
	})
	if err != nil {
		log.Fatal(err, "REQUEST PROTOCOL")
	}
	if len(utxos) == 0 {
		return UTxO.UTxO{}, fmt.Errorf("Could not fetch utxo: %v#%v", txHash, index)
	} else {
		apolloUtxo := Utxo_OgmigoToApollo(utxos[0])
		return apolloUtxo, nil
	}
}

func ogmiosValue_toAddressAmount(v shared.Value) []Base.AddressAmount {
	amts := make([]Base.AddressAmount, 0)
	amts = append(amts, Base.AddressAmount{
		Unit:     "lovelace",
		Quantity: strconv.FormatInt(v.AdaLovelace().Int64(), 10),
	})
	val := v.AssetsExceptAda()
	for policyId, tokenMap := range val {
		for tokenName, quantity := range tokenMap {
			amts = append(amts, Base.AddressAmount{
				Unit:     policyId + tokenName,
				Quantity: strconv.FormatInt(quantity.Int64(), 10),
			})
		}
	}
	return amts
}

func (occ *OgmiosChainContext) TxOuts(txHash string) []Base.Output {
	ctx := context.Background()
	outs := make([]Base.Output, 1)
	more_utxos := true
	chunk_size := 10
	for more_utxos {
		queries := make([]chainsync.TxInQuery, chunk_size)
		for ix, _ := range queries {
			queries[ix] = chainsync.TxInQuery{
				Transaction: shared.UtxoTxID{
					ID: txHash,
				},
				Index: uint32(ix),
			}
		}
		us, err := occ.ogmigo.UtxosByTxIn(ctx, queries...)
		if len(us) < chunk_size || err != nil {
			more_utxos = false
		}
		for _, u := range us {
			am := ogmiosValue_toAddressAmount(u.Value)
			apolloUtxo := Base.Output{
				Address:             u.Address,
				Amount:              am,
				OutputIndex:         int(u.Index),
				DataHash:            u.DatumHash,
				InlineDatum:         u.Datum,
				Collateral:          false, // Can querying ogmios return collateral outputs?
				ReferenceScriptHash: "",    // TODO
			}
			outs = append(outs, apolloUtxo)
		}
	}
	return outs
}

// Seems unused
func (occ *OgmiosChainContext) LatestBlock() Base.Block {
	ctx := context.Background()
	point, err := occ.ogmigo.ChainTip(ctx)
	if err != nil {
		log.Fatal("OgmiosChainContext: LatestBlock: failed to request chain tip", err)
	}
	s, ok := point.PointStruct()
	if !ok {
		log.Fatal("OgmiosChainContext: LatestBlock: expected a struct")
	}
	return Base.Block{
		Hash: s.ID,
		Slot: int(s.Slot),
	}
}

// Given an era history, find the unix timestamp for the end of the current
// epoch
func computeEndTime(epoch uint64, networkStartTime time.Time, history *ogmigo.EraHistory) (uint64, error) {
	for _, summary := range history.Summaries {
		// Skip ahead to current era
		if !(summary.Start.Epoch <= epoch && summary.End.Epoch > epoch) {
			continue
		}
		eraStartOffset := summary.Start.Time.Seconds
		eraStartTime := networkStartTime.Add(time.Second * time.Duration(eraStartOffset.Int64()))
		slotDuration := time.Millisecond * time.Duration(summary.Parameters.SlotLength.Milliseconds.Int64())
		// We have to count epochs from the era start to the current
		// epoch, and add 1 since we want the end of this one
		eraElapsed := slotDuration * time.Duration((epoch+1-summary.Start.Epoch)*summary.Parameters.EpochLength)
		eraEndTime := eraStartTime.Add(eraElapsed)
		return uint64(eraEndTime.Unix()), nil
	}
	return 0, fmt.Errorf("Epoch %v not found in history: %v", epoch, *history)
}

// Because ogmios does not return the end time when querying the current epoch,
// we have to dig through the era summaries and query the network start time
func (occ *OgmiosChainContext) LatestEpoch() Base.Epoch {
	ctx := context.Background()
	current, err := occ.ogmigo.CurrentEpoch(ctx)
	if err != nil {
		log.Fatal(err, "OgmiosChainContext: LatestEpoch: failed to request current epoch")
	}
	genesisConfig, err := occ.ogmigo.GenesisConfig(ctx, "byron")
	if err != nil {
		log.Fatal(err, "OgmiosChainContext: LatestEpoch: failed to request genesis config")
	}
	var genesisInfo struct {
		StartTime string
	}
	err = json.Unmarshal(genesisConfig, &genesisInfo)
	if err != nil {
		log.Fatal(err, "OgmiosChainContext: LatestEpoch: failed to parse genesis config")
	}
	startTime, err := time.Parse(time.RFC3339, genesisInfo.StartTime)
	if err != nil {
		log.Fatal(err, "OgmiosChainContext: LatestEpoch: failed to parse genesis config startTime")
	}
	eraSummaries, err := occ.ogmigo.EraSummaries(ctx)
	if err != nil {
		log.Fatal(err, "OgmiosChainContext: LatestEpoch: failed to request era summaries")
	}
	endTime, err := computeEndTime(current, startTime, eraSummaries)
	if err != nil {
		log.Fatal(err, "OgmiosChainContext: LatestEpoch: failed to compute end time for epoch")
	}
	return Base.Epoch{
		Epoch:   int(current),
		EndTime: int(endTime),
	}
}

func (occ *OgmiosChainContext) KupoToUtxo(m kugo.Match) UTxO.UTxO {
	ctx := context.Background()
	addr, au := occ.kupoToAddressUtxo(ctx, m)
	return occ.addressUtxoToUtxo(ctx, addr, au)
}

func (occ *OgmiosChainContext) kupoToAddressUtxo(ctx context.Context, match kugo.Match) (Address.Address, Base.AddressUTXO) {
	datum := ""
	var err error
	if match.DatumType == "inline" {
		datum, err = occ.kugo.Datum(ctx, match.DatumHash)
		if err != nil {
			log.Fatal(err, "OgmiosChainContext: AddressUtxos: kupo datum request failed")
		}
	}
	am := ogmiosValue_toAddressAmount(shared.Value(match.Value))
	addr, _ := Address.DecodeAddress(match.Address)
	return addr, Base.AddressUTXO{
		TxHash:      match.TransactionID,
		OutputIndex: match.OutputIndex,
		Amount:      am,
		// We probably don't need this info and kupo doesn't provide it in this query
		Block:               "",
		DataHash:            match.DatumHash,
		InlineDatum:         datum,
		ReferenceScriptHash: match.ScriptHash,
	}
}

func (occ *OgmiosChainContext) AddressUtxos(address string, gather bool) []Base.AddressUTXO {
	ctx := context.Background()
	addressUtxos := make([]Base.AddressUTXO, 0)
	matches, err := occ.kugo.Matches(ctx, kugo.OnlyUnspent(), kugo.Address(address))
	if err != nil {
		log.Fatal(err, "OgmiosChainContext: AddressUtxos: kupo request failed")
	}
	for _, match := range matches {
		_, addrUtxo := occ.kupoToAddressUtxo(ctx, match)
		addressUtxos = append(addressUtxos, addrUtxo)
	}
	return addressUtxos

}

type AdaLovelace struct {
	Ada Lovelace `json:"ada"`
}

type Lovelace struct {
	Lovelace uint64 `json:"lovelace"`
}

type Bytes struct {
	Bytes uint64 `json:"bytes"`
}

type Prices struct {
	Memory float64 `json:"memory"`
	Cpu    float64 `json:"cpu"`
}

func parseFraction(s string) (int64, int64, error) {
	before, after, found := strings.Cut(s, "/")
	if !found {
		return 0, 0, fmt.Errorf("parseFraction: Not a fraction: %s", s)
	}
	n, err := strconv.ParseInt(before, 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("parseFraction: %w", err)
	}
	d, err := strconv.ParseInt(after, 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("parseFraction: %w", err)
	}
	return n, d, nil
}

func (p *Prices) UnmarshalJSON(b []byte) error {
	var x struct {
		Memory string `json:"memory"`
		Cpu    string `json:"cpu"`
	}
	err := json.Unmarshal(b, &x)
	if err != nil {
		return err
	}
	mn, md, err := parseFraction(x.Memory)
	if err != nil {
		return err
	}
	p.Memory = float64(mn) / float64(md)
	cn, cd, err := parseFraction(x.Cpu)
	if err != nil {
		return err
	}
	p.Cpu = float64(cn) / float64(cd)
	return nil
}

type Version struct {
	Major uint64 `json:"major"`
	Minor uint64 `json:"minor"`
	Patch uint64 `json:"patch"`
}

type ExUnits struct {
	Cpu    uint64 `json:"cpu"`
	Memory uint64 `json:"memory"`
}

type ReferenceScriptsFees struct {
	Base       float64 `json:"base"`
	Range      uint64  `json:"range"`
	Multiplier float64 `json:"multiplier"`
}

type OgmiosCostModels struct {
	V1 []int
	V2 []int
	V3 []int
}

func (ocm *OgmiosCostModels) UnmarshalJSON(bytes []byte) error {
	x := make(map[string][]int)
	err := json.Unmarshal(bytes, &x)
	if err != nil {
		return err
	}
	v1, ok := x["plutus:v1"]
	if !ok {
		return fmt.Errorf("OgmiosCostModels: UnmarshalJSON: missing 'plutus:v1': %v", string(bytes))
	}
	v2, ok := x["plutus:v2"]
	if !ok {
		return fmt.Errorf("OgmiosCostModels: UnmarshalJSON: missing 'plutus:v2': %v", string(bytes))
	}
	v3, ok := x["plutus:v3"]
	if !ok {
		return fmt.Errorf("OgmiosCostModels: UnmarshalJSON: missing 'plutus:v3': %v", string(bytes))
	}
	ocm.V1 = v1
	ocm.V2 = v2
	ocm.V3 = v3
	return nil
}

type OgmiosProtocolParameters struct {
	MinFeeConstant                  AdaLovelace          `json:"minFeeConstant"`
	MinFeeCoefficient               uint64               `json:"minFeeCoefficient"`
	MaxBlockSize                    Bytes                `json:"maxBlockBodySize"`
	MaxTxSize                       Bytes                `json:"maxTransactionSize"`
	MaxBlockHeaderSize              Bytes                `json:"maxBlockHeaderSize"`
	KeyDeposits                     AdaLovelace          `json:"stakeCredentialDeposit"`
	PoolDeposits                    AdaLovelace          `json:"stakePoolDeposit"`
	PoolInfluence                   string               `json:"stakePoolPledgeInfluence"`
	MonetaryExpansion               string               `json:"monetaryExpansion"`
	TreasuryExpansion               string               `json:"treasuryExpansion"`
	ExtraEntropy                    string               `json:"extraEntropy"`
	MaxValSize                      Bytes                `json:"maxValueSize"`
	ScriptExecutionPrices           Prices               `json:"scriptExecutionPrices"`
	MinUtxoDepositCoefficient       uint64               `json:"minUtxoDepositCoefficient"`
	MinUtxoDepositConstant          AdaLovelace          `json:"minUtxoDepositConstant"`
	MinStakePoolCost                AdaLovelace          `json:"minStakePoolCost"`
	MaxExecutionUnitsPerTransaction ExUnits              `json:"maxExecutionUnitsPerTransaction"`
	MaxExecutionUnitsPerBlock       ExUnits              `json:"maxExecutionUnitsPerBlock"`
	CollateralPercentage            uint64               `json:"collateralPercentage"`
	MaxCollateralInputs             uint64               `json:"maxCollateralInputs"`
	MinFeeReferenceScripts          ReferenceScriptsFees `json:"minFeeReferenceScripts"`
	CostModels                      OgmiosCostModels     `json:"plutusCostModels"`
	Version                         Version              `json:"version"`
}

func ratio(s string) float32 {
	n, d, ok := strings.Cut(s, "/")
	if !ok {
		return 0
	}
	num, err := strconv.Atoi(n)
	if err != nil {
		return 0
	}
	den, err := strconv.Atoi(d)
	if err != nil {
		return 0
	}
	return float32(num) / float32(den)
}

func (occ *OgmiosChainContext) LatestEpochParams() Base.ProtocolParameters {
	ctx := context.Background()
	pparams, err := occ.ogmigo.CurrentProtocolParameters(ctx)
	if err != nil {
		log.Fatal(err, "OgmiosChainContext: LatestEpochParams: protocol parameters request failed")
	}

	var ogmiosParams OgmiosProtocolParameters
	if err := json.Unmarshal(pparams, &ogmiosParams); err != nil {
		log.Fatal(err, "OgmiosChainContext: LatestEpochParams: failed to parse protocol parameters")
	}

	cm := map[Base.CostModelsPlutusVersion]PlutusData.CostModel{
		Base.CostModelsPlutusV1: ogmiosParams.CostModels.V1,
		Base.CostModelsPlutusV2: ogmiosParams.CostModels.V2,
		Base.CostModelsPlutusV3: ogmiosParams.CostModels.V3,
	}

	return Base.ProtocolParameters{
		MinFeeConstant:     int(ogmiosParams.MinFeeConstant.Ada.Lovelace),
		MinFeeCoefficient:  int(ogmiosParams.MinFeeCoefficient),
		MaxBlockSize:       int(ogmiosParams.MaxBlockSize.Bytes),
		MaxTxSize:          int(ogmiosParams.MaxTxSize.Bytes),
		MaxBlockHeaderSize: int(ogmiosParams.MaxBlockHeaderSize.Bytes),
		KeyDeposits:        strconv.FormatUint(ogmiosParams.KeyDeposits.Ada.Lovelace, 10),
		PoolDeposits:       strconv.FormatUint(ogmiosParams.PoolDeposits.Ada.Lovelace, 10),
		PooolInfluence:     ratio(ogmiosParams.PoolInfluence),
		MonetaryExpansion:  ratio(ogmiosParams.MonetaryExpansion),
		TreasuryExpansion:  ratio(ogmiosParams.TreasuryExpansion),
		// Unsure if ogmios reports this, but it's 0 on mainnet and
		// preview
		DecentralizationParam: 0,
		ExtraEntropy:          ogmiosParams.ExtraEntropy,
		MinUtxo:               strconv.FormatUint(ogmiosParams.MinUtxoDepositConstant.Ada.Lovelace, 10),
		ProtocolMajorVersion:  int(ogmiosParams.Version.Major),
		ProtocolMinorVersion:  int(ogmiosParams.Version.Minor),
		MinPoolCost:           strconv.FormatUint(ogmiosParams.MinStakePoolCost.Ada.Lovelace, 10),
		PriceMem:              float32(ogmiosParams.ScriptExecutionPrices.Memory),
		PriceStep:             float32(ogmiosParams.ScriptExecutionPrices.Cpu),
		MaxTxExMem:            strconv.FormatUint(ogmiosParams.MaxExecutionUnitsPerTransaction.Memory, 10),
		MaxTxExSteps:          strconv.FormatUint(ogmiosParams.MaxExecutionUnitsPerTransaction.Cpu, 10),
		MaxBlockExMem:         strconv.FormatUint(ogmiosParams.MaxExecutionUnitsPerBlock.Memory, 10),
		MaxBlockExSteps:       strconv.FormatUint(ogmiosParams.MaxExecutionUnitsPerBlock.Cpu, 10),
		MaxValSize:            strconv.FormatUint(ogmiosParams.MaxValSize.Bytes, 10),
		CollateralPercent:     int(ogmiosParams.CollateralPercentage),
		MaxCollateralInuts:    int(ogmiosParams.MaxCollateralInputs),
		CoinsPerUtxoByte:      strconv.FormatUint(ogmiosParams.MinUtxoDepositCoefficient, 10),
		// PerUtxoWord is deprecated https://cips.cardano.org/cips/cip55/
		CoinsPerUtxoWord:       strconv.FormatUint(ogmiosParams.MinUtxoDepositCoefficient, 10),
		MinFeeReferenceScripts: int(ogmiosParams.MinFeeReferenceScripts.Base),
		CostModels:             cm,
	}
}

func (occ *OgmiosChainContext) GenesisParams() Base.GenesisParameters {
	genesisParams := Base.GenesisParameters{}
	//TODO
	return genesisParams
}
func (occ *OgmiosChainContext) _CheckEpochAndUpdate() bool {
	if occ._epoch_info.EndTime <= int(time.Now().Unix()) {
		latest_epochs := occ.LatestEpoch()
		occ._epoch_info = latest_epochs
		return true
	}
	return false
}

func (occ *OgmiosChainContext) Network() int {
	return occ._Network
}

func (occ *OgmiosChainContext) Epoch() int {
	if occ._CheckEpochAndUpdate() {
		new_epoch := occ.LatestEpoch()
		occ._epoch = new_epoch.Epoch
	}
	return occ._epoch
}

// Seems unused
func (occ *OgmiosChainContext) LastBlockSlot() int {
	block := occ.LatestBlock()
	return block.Slot
}

func (occ *OgmiosChainContext) GetGenesisParams() Base.GenesisParameters {
	if occ._CheckEpochAndUpdate() {
		params := occ.GenesisParams()
		occ._genesis_param = params
	}
	return occ._genesis_param
}

func (occ *OgmiosChainContext) GetProtocolParams() Base.ProtocolParameters {
	if occ._CheckEpochAndUpdate() {
		latest_params := occ.LatestEpochParams()
		occ._protocol_param = latest_params
	}
	return occ._protocol_param
}

func (occ *OgmiosChainContext) MaxTxFee() int {
	protocol_param := occ.GetProtocolParams()
	maxTxExSteps, _ := strconv.Atoi(protocol_param.MaxTxExSteps)
	maxTxExMem, _ := strconv.Atoi(protocol_param.MaxTxExMem)
	return Base.Fee(occ, protocol_param.MaxTxSize, maxTxExSteps, maxTxExMem)
}

func (occ *OgmiosChainContext) addressUtxoToUtxo(ctx context.Context, address Address.Address, result Base.AddressUTXO) UTxO.UTxO {
	decodedTxId, _ := hex.DecodeString(result.TxHash)
	tx_in := TransactionInput.TransactionInput{TransactionId: decodedTxId, Index: result.OutputIndex}
	amount := result.Amount
	lovelace_amount := 0
	multi_assets := MultiAsset.MultiAsset[int64]{}
	for _, item := range amount {
		if item.Unit == "lovelace" {
			amount, err := strconv.Atoi(item.Quantity)
			if err != nil {
				log.Fatal(err)
			}
			lovelace_amount += amount
		} else {
			asset_quantity, err := strconv.ParseInt(item.Quantity, 10, 64)
			if err != nil {
				log.Fatal(err)
			}
			policy_id := Policy.PolicyId{Value: item.Unit[:56]}
			asset_name := *AssetName.NewAssetNameFromHexString(item.Unit[56:])
			_, ok := multi_assets[policy_id]
			if !ok {
				multi_assets[policy_id] = Asset.Asset[int64]{}
			}
			multi_assets[policy_id][asset_name] = int64(asset_quantity)
		}
	}
	final_amount := Value.Value{}
	if len(multi_assets) > 0 {
		final_amount = Value.Value{Am: Amount.Amount{Coin: int64(lovelace_amount), Value: multi_assets}, HasAssets: true}
	} else {
		final_amount = Value.Value{Coin: int64(lovelace_amount), HasAssets: false}
	}
	datum_hash := serialization.DatumHash{}
	if result.DataHash != "" && result.InlineDatum == "" {

		datum_hash = serialization.DatumHash{}
		copy(datum_hash.Payload[:], result.DataHash[:])
	}

	// Get reference script
	var refScript []byte
	if result.ReferenceScriptHash != "" {
		ref, err := occ.kugo.Script(ctx, result.ReferenceScriptHash)
		if err != nil {
			log.Fatal(fmt.Errorf("OgmiosChainContext: failed to query reference script hash: '%v': %w", result.ReferenceScriptHash, err))
		}
		raw, err := hex.DecodeString(ref.Script)
		if err != nil {
			log.Fatal(fmt.Errorf("OgmiosChainContext: failed to decode reference script bytes from hex: %w", err))
		}
		refScript = raw
	}

	// Get inline datum
	var inlineDatum *PlutusData.DatumOption
	if result.InlineDatum != "" {
		decoded, err := hex.DecodeString(result.InlineDatum)
		if err != nil {
			log.Fatal(err)
		}
		var x PlutusData.PlutusData
		err = cbor.Unmarshal(decoded, &x)
		if err != nil {
			log.Fatal(err)
		}
		option := PlutusData.DatumOptionInline(&x)
		inlineDatum = &option
	}

	has_alonzo_datum := inlineDatum != nil
	has_ref_script := refScript != nil

	var tx_out TransactionOutput.TransactionOutput
	if has_alonzo_datum || has_ref_script {
		tx_out = TransactionOutput.TransactionOutput{IsPostAlonzo: true,
			PostAlonzo: TransactionOutput.TransactionOutputAlonzo{
				Address: address,
				Amount:  final_amount.ToAlonzoValue(),
			},
		}
		if inlineDatum != nil {
			tx_out.PostAlonzo.Datum = inlineDatum
		}
		if refScript != nil {
			tx_out.PostAlonzo.ScriptRef = &PlutusData.ScriptRef{
				Script: PlutusData.InnerScript{
					Script: refScript,
				},
			}
		}
	} else {
		tx_out = TransactionOutput.TransactionOutput{PreAlonzo: TransactionOutput.TransactionOutputShelley{
			Address:   address,
			Amount:    final_amount,
			DatumHash: datum_hash,
			HasDatum:  len(datum_hash.Payload) > 0}, IsPostAlonzo: false}
	}
	return UTxO.UTxO{
		Input:  tx_in,
		Output: tx_out,
	}
}

// Copied from blockfrost context def since it just calls AddressUtxos and then
// converts
func (occ *OgmiosChainContext) Utxos(address Address.Address) []UTxO.UTxO {
	ctx := context.Background()
	results := occ.AddressUtxos(address.String(), true)
	utxos := make([]UTxO.UTxO, 0)
	for _, result := range results {
		utxos = append(utxos, occ.addressUtxoToUtxo(ctx, address, result))
	}
	return utxos
}

func (occ *OgmiosChainContext) SubmitTx(tx Transaction.Transaction) (serialization.TransactionId, error) {
	ctx := context.Background()
	bytes := tx.Bytes()
	result, err := occ.ogmigo.SubmitTx(ctx, hex.EncodeToString(bytes))
	if err != nil {
		return serialization.TransactionId{}, fmt.Errorf("OgmiosChainContext: SubmitTx: %v", err)
	}
	if result.Error != nil {
		return serialization.TransactionId{}, OgmiosError{
			Code:    result.Error.Code,
			Message: result.Error.Message,
			Data:    result.Error.Data,
		}
	}
	return tx.TransactionBody.Id(), nil
}

func convertOgmiosRedeemerTag(tag string) (string, error) {
	switch tag {
	case "spend":
		return Redeemer.RedeemerTagNames[0], nil
	case "mint":
		return Redeemer.RedeemerTagNames[1], nil
	case "publish":
		return Redeemer.RedeemerTagNames[2], nil
	case "withdraw":
		return Redeemer.RedeemerTagNames[3], nil
	default:
		return "", fmt.Errorf("Unexpected ogmios redeemer tag: %s", tag)
	}
}

type OgmiosError struct {
	Code    int
	Message string
	Data    []byte
}

func (o OgmiosError) Error() string {
	return fmt.Sprintf("%v %v %v", o.Code, o.Message, string(o.Data))
}

func (occ *OgmiosChainContext) evaluateTx(tx []byte, additionalUtxos []UTxO.UTxO) (map[string]Redeemer.ExecutionUnits, error) {
	final_result := make(map[string]Redeemer.ExecutionUnits)
	ctx := context.Background()
	var additionalUtxosOgmigo []shared.Utxo
	for _, u := range additionalUtxos {
		additionalUtxosOgmigo = append(additionalUtxosOgmigo, Utxo_ApolloToOgmigo(u))
	}
	eval, err := occ.ogmigo.EvaluateTxWithAdditionalUtxos(ctx, hex.EncodeToString(tx), additionalUtxosOgmigo)
	if err != nil {
		return nil, fmt.Errorf(
			"OgmiosChainContext: EvaluateTx: Error evaluating tx: %v",
			err,
		)
	}
	if eval.Error != nil {
		return nil, OgmiosError{
			Code:    eval.Error.Code,
			Message: eval.Error.Message,
			Data:    eval.Error.Data,
		}
	}
	for _, e := range eval.ExUnits {
		purpose, err := convertOgmiosRedeemerTag(e.Validator.Purpose)
		if err != nil {
			return nil, fmt.Errorf("OgmiosChainContext: EvaluateTx: %w", err)
		}
		val := fmt.Sprintf("%v:%v", purpose, e.Validator.Index)
		final_result[val] = Redeemer.ExecutionUnits{
			Mem:   int64(e.Budget.Memory),
			Steps: int64(e.Budget.Cpu),
		}
	}
	return final_result, nil
}

func (occ *OgmiosChainContext) CostModelsV1() PlutusData.CostModel {
	pparams := occ.GetProtocolParams()
	return pparams.CostModels[Base.CostModelsPlutusV1]
}

func (occ *OgmiosChainContext) CostModelsV2() PlutusData.CostModel {
	pparams := occ.GetProtocolParams()
	return pparams.CostModels[Base.CostModelsPlutusV2]
}

func (occ *OgmiosChainContext) CostModelsV3() PlutusData.CostModel {
	pparams := occ.GetProtocolParams()
	return pparams.CostModels[Base.CostModelsPlutusV3]
}

func (occ *OgmiosChainContext) EvaluateTx(tx []byte) (map[string]Redeemer.ExecutionUnits, error) {
	return occ.evaluateTx(tx, nil)
}

func (occ *OgmiosChainContext) EvaluateTxWithAdditionalUtxos(tx []byte, additionalUtxos []UTxO.UTxO) (map[string]Redeemer.ExecutionUnits, error) {
	return occ.evaluateTx(tx, additionalUtxos)
}

// This is unused
func (occ *OgmiosChainContext) GetContractCbor(scriptHash string) string {
	//TODO
	return ""
}

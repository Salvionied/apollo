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

	"github.com/Salvionied/apollo/serialization"
	"github.com/Salvionied/apollo/serialization/Address"
	"github.com/Salvionied/apollo/serialization/Amount"
	"github.com/Salvionied/apollo/serialization/Asset"
	"github.com/Salvionied/apollo/serialization/AssetName"
	"github.com/Salvionied/apollo/serialization/MultiAsset"
	"github.com/Salvionied/apollo/serialization/PlutusData"
	"github.com/Salvionied/apollo/serialization/Policy"
	"github.com/Salvionied/apollo/serialization/Redeemer"
	"github.com/Salvionied/apollo/serialization/Transaction"
	"github.com/Salvionied/apollo/serialization/TransactionInput"
	"github.com/Salvionied/apollo/serialization/TransactionOutput"
	"github.com/Salvionied/apollo/serialization/UTxO"
	"github.com/Salvionied/apollo/serialization/Value"
	"github.com/Salvionied/apollo/txBuilding/Backend/Base"
	"github.com/SundaeSwap-finance/kugo"
	"github.com/SundaeSwap-finance/ogmigo/v6"
	"github.com/SundaeSwap-finance/ogmigo/v6/ouroboros/chainsync"
	"github.com/SundaeSwap-finance/ogmigo/v6/ouroboros/chainsync/num"
	"github.com/SundaeSwap-finance/ogmigo/v6/ouroboros/shared"

	"github.com/fxamacker/cbor/v2"
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

func value_OgmigoToApollo(v shared.Value) Value.AlonzoValue {
	ass := multiAsset_OgmigoToApollo(v.AssetsExceptAda())
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

func scriptRef_OgmigoToApollo(script json.RawMessage) (*PlutusData.ScriptRef, error) {
	if len(script) == 0 {
		return nil, nil
	}
	var tmpData struct {
		Language string `json:"language"`
		Cbor     string `json:"cbor"`
	}
	if err := json.Unmarshal(script, &tmpData); err != nil {
		return nil, err
	}
	scriptBytes, err := hex.DecodeString(tmpData.Cbor)
	if err != nil {
		return nil, err
	}
	ref := PlutusData.ScriptRef(scriptBytes)
	return &ref, nil
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

func (occ *OgmiosChainContext) GetUtxoFromRef(txHash string, index int) (*UTxO.UTxO, error) {
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
		return nil, nil
	} else {
		apolloUtxo := Utxo_OgmigoToApollo(utxos[0])
		return &apolloUtxo, nil
	}
}

func statequeryValue_toAddressAmount(v shared.Value) []Base.AddressAmount {
	amts := make([]Base.AddressAmount, 0)
	amts = append(amts, Base.AddressAmount{
		Unit:     "lovelace",
		Quantity: strconv.FormatInt(v.AdaLovelace().Int64(), 10),
	})
	for policyId, tokenMap := range v.AssetsExceptAda() {
		for tokenName, quantity := range tokenMap {
			amts = append(amts, Base.AddressAmount{
				Unit:     policyId + tokenName,
				Quantity: strconv.FormatInt(quantity.Int64(), 10),
			})
		}
	}
	return amts
}

func kugoValue_toSharedValue(v kugo.Value) shared.Value {
	result := shared.Value{}
	for policyId, assets := range v {
		for assetName, amt := range assets {
			if _, ok := result[policyId]; !ok {
				result[policyId] = map[string]num.Int{}
			}
			result[policyId][assetName] = amt
		}
	}
	return result
}

func chainsyncValue_toAddressAmount(v shared.Value) []Base.AddressAmount {
	// same as above
	return statequeryValue_toAddressAmount(v)
}

func (occ *OgmiosChainContext) TxOuts(txHash string) []Base.Output {
	ctx := context.Background()
	outs := make([]Base.Output, 1)
	more_utxos := true
	chunk_size := 10
	for more_utxos {
		queries := make([]chainsync.TxInQuery, chunk_size)
		for ix := range queries {
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
			am := statequeryValue_toAddressAmount(u.Value)
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

func (occ *OgmiosChainContext) LatestEpoch() Base.Epoch {
	ctx := context.Background()
	current, err := occ.ogmigo.CurrentEpoch(ctx)
	if err != nil {
		log.Fatal(err, "OgmiosChainContext: LatestEpoch: failed to request current epoch")
	}
	return Base.Epoch{
		Epoch: int(current),
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
		datum := ""
		if match.DatumType == "inline" {
			datum, err = occ.kugo.Datum(ctx, match.DatumHash)
			if err != nil {
				log.Fatal(err, "OgmiosChainContext: AddressUtxos: kupo datum request failed")
			}
		}
		addressUtxos = append(addressUtxos, Base.AddressUTXO{
			TxHash:      match.TransactionID,
			OutputIndex: match.OutputIndex,
			Amount:      chainsyncValue_toAddressAmount(kugoValue_toSharedValue(match.Value)),
			// We probably don't need this info and kupo doesn't provide it in this query
			Block:       "",
			DataHash:    match.DatumHash,
			InlineDatum: datum,
		})
	}
	return addressUtxos

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

type OgmiosProtocolParameters struct {
	MinFeeConstant                  Lovelace               `json:"minFeeConstant"`
	MinFeeCoefficient               uint64                 `json:"minFeeCoefficient"`
	MaxBlockSize                    Bytes                  `json:"maxBlockBodySize"`
	MaxTxSize                       Bytes                  `json:"maxTransactionSize"`
	MaxBlockHeaderSize              Bytes                  `json:"maxBlockHeaderSize"`
	KeyDeposits                     Lovelace               `json:"stakeCredentialDeposit"`
	PoolDeposits                    Lovelace               `json:"stakePoolDeposit"`
	PoolInfluence                   string                 `json:"stakePoolPledgeInfluence"`
	MonetaryExpansion               string                 `json:"monetaryExpansion"`
	TreasuryExpansion               string                 `json:"treasuryExpansion"`
	ExtraEntropy                    string                 `json:"extraEntropy"`
	MaxValSize                      Bytes                  `json:"maxValueSize"`
	ScriptExecutionPrices           Prices                 `json:"scriptExecutionPrices"`
	MinUtxoDepositCoefficient       uint64                 `json:"minUtxoDepositCoefficient"`
	MinUtxoDepositConstant          Lovelace               `json:"minUtxoDepositConstant"`
	MinStakePoolCost                Lovelace               `json:"minStakePoolCost"`
	MaxExecutionUnitsPerTransaction ExUnits                `json:"maxExecutionUnitsPerTransaction"`
	MaxExecutionUnitsPerBlock       ExUnits                `json:"maxExecutionUnitsPerBlock"`
	CollateralPercentage            uint64                 `json:"collateralPercentage"`
	MaxCollateralInputs             uint64                 `json:"maxCollateralInputs"`
	MaximumReferenceScriptsSize     uint64                 `json:"maximumReferenceScriptsSize"`
	MinFeeReferenceScripts          MinFeeReferenceScripts `json:"minFeeReferenceScripts"`
	Version                         Version                `json:"version"`
	CostModels                      map[string][]int64     `json:"plutusCostModels"`
}

type MinFeeReferenceScripts struct {
	Range      float64 `json:"range"`
	Base       float64 `json:"base"`
	Multiplier float64 `json:"multiplier"`
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

	return Base.ProtocolParameters{
		MinFeeConstant:     int(ogmiosParams.MinFeeConstant.Lovelace),
		MinFeeCoefficient:  int(ogmiosParams.MinFeeCoefficient),
		MaxBlockSize:       int(ogmiosParams.MaxBlockSize.Bytes),
		MaxTxSize:          int(ogmiosParams.MaxTxSize.Bytes),
		MaxBlockHeaderSize: int(ogmiosParams.MaxBlockHeaderSize.Bytes),
		KeyDeposits:        strconv.FormatUint(ogmiosParams.KeyDeposits.Lovelace, 10),
		PoolDeposits:       strconv.FormatUint(ogmiosParams.PoolDeposits.Lovelace, 10),
		PooolInfluence:     ratio(ogmiosParams.PoolInfluence),
		MonetaryExpansion:  ratio(ogmiosParams.MonetaryExpansion),
		TreasuryExpansion:  ratio(ogmiosParams.TreasuryExpansion),
		// Unsure if ogmios reports this, but it's 0 on mainnet and
		// preview
		DecentralizationParam: 0,
		ExtraEntropy:          ogmiosParams.ExtraEntropy,
		MinUtxo:               strconv.FormatUint(ogmiosParams.MinUtxoDepositConstant.Lovelace, 10),
		ProtocolMajorVersion:  int(ogmiosParams.Version.Major),
		ProtocolMinorVersion:  int(ogmiosParams.Version.Minor),
		MinPoolCost:           strconv.FormatUint(ogmiosParams.MinStakePoolCost.Lovelace, 10),
		PriceMem:              float32(ogmiosParams.ScriptExecutionPrices.Memory),
		PriceStep:             float32(ogmiosParams.ScriptExecutionPrices.Cpu),
		MaxTxExMem: strconv.FormatUint(
			ogmiosParams.MaxExecutionUnitsPerTransaction.Memory,
			10,
		),
		MaxTxExSteps: strconv.FormatUint(
			ogmiosParams.MaxExecutionUnitsPerTransaction.Cpu,
			10,
		),
		MaxBlockExMem: strconv.FormatUint(
			ogmiosParams.MaxExecutionUnitsPerBlock.Memory,
			10,
		),
		MaxBlockExSteps:    strconv.FormatUint(ogmiosParams.MaxExecutionUnitsPerBlock.Cpu, 10),
		MaxValSize:         strconv.FormatUint(ogmiosParams.MaxValSize.Bytes, 10),
		CollateralPercent:  int(ogmiosParams.CollateralPercentage),
		MaxCollateralInuts: int(ogmiosParams.MaxCollateralInputs),
		CoinsPerUtxoByte:   strconv.FormatUint(ogmiosParams.MinUtxoDepositCoefficient, 10),
		// PerUtxoWord is deprecated https://cips.cardano.org/cips/cip55/
		CoinsPerUtxoWord:                 strconv.FormatUint(ogmiosParams.MinUtxoDepositCoefficient, 10),
		MaximumReferenceScriptsSize:      int(ogmiosParams.MaximumReferenceScriptsSize),
		MinFeeReferenceScriptsRange:      int(ogmiosParams.MinFeeReferenceScripts.Range),
		MinFeeReferenceScriptsBase:       int(ogmiosParams.MinFeeReferenceScripts.Base),
		MinFeeReferenceScriptsMultiplier: int(ogmiosParams.MinFeeReferenceScripts.Multiplier),
		CostModels:                       ogmiosParams.CostModels,
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

func (occ *OgmiosChainContext) Epoch() (int, error) {
	if occ._CheckEpochAndUpdate() {
		new_epoch := occ.LatestEpoch()
		occ._epoch = new_epoch.Epoch
	}
	return occ._epoch, nil
}

// Seems unused
func (occ *OgmiosChainContext) LastBlockSlot() (int, error) {
	block := occ.LatestBlock()
	return block.Slot, nil
}

func (occ *OgmiosChainContext) GetGenesisParams() (Base.GenesisParameters, error) {
	if occ._CheckEpochAndUpdate() {
		params := occ.GenesisParams()
		occ._genesis_param = params
	}
	return occ._genesis_param, nil
}

func (occ *OgmiosChainContext) GetProtocolParams() (Base.ProtocolParameters, error) {
	if occ._CheckEpochAndUpdate() {
		latest_params := occ.LatestEpochParams()
		occ._protocol_param = latest_params
	}
	return occ._protocol_param, nil
}

func (occ *OgmiosChainContext) MaxTxFee() (int, error) {
	protocol_param, _ := occ.GetProtocolParams()
	maxTxExSteps, _ := strconv.Atoi(protocol_param.MaxTxExSteps)
	maxTxExMem, _ := strconv.Atoi(protocol_param.MaxTxExMem)
	return Base.Fee(occ, protocol_param.MaxTxSize, maxTxExSteps, maxTxExMem)
}

// Copied from blockfrost context def since it just calls AddressUtxos and then
// converts
func (occ *OgmiosChainContext) Utxos(address Address.Address) ([]UTxO.UTxO, error) {
	results := occ.AddressUtxos(address.String(), true)
	utxos := make([]UTxO.UTxO, 0)
	for _, result := range results {
		decodedTxId, _ := hex.DecodeString(result.TxHash)
		tx_in := TransactionInput.TransactionInput{
			TransactionId: decodedTxId,
			Index:         result.OutputIndex,
		}
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
		var final_amount Value.Value
		if len(multi_assets) > 0 {
			final_amount = Value.Value{
				Am:        Amount.Amount{Coin: int64(lovelace_amount), Value: multi_assets},
				HasAssets: true,
			}
		} else {
			final_amount = Value.Value{Coin: int64(lovelace_amount), HasAssets: false}
		}
		datum_hash := serialization.DatumHash{}
		if result.DataHash != "" && result.InlineDatum == "" {

			datum_hash = serialization.DatumHash{}
			copy(datum_hash.Payload[:], result.DataHash[:])
		}
		var tx_out TransactionOutput.TransactionOutput
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
			l := PlutusData.DatumOptionInline(&x)
			tx_out = TransactionOutput.TransactionOutput{IsPostAlonzo: true,
				PostAlonzo: TransactionOutput.TransactionOutputAlonzo{
					Address: address,
					Amount:  final_amount.ToAlonzoValue(),
					Datum:   &l},
			}
		} else {
			tx_out = TransactionOutput.TransactionOutput{PreAlonzo: TransactionOutput.TransactionOutputShelley{
				Address:   address,
				Amount:    final_amount,
				DatumHash: datum_hash,
				HasDatum:  len(datum_hash.Payload) > 0}, IsPostAlonzo: false}
		}
		utxos = append(utxos, UTxO.UTxO{Input: tx_in, Output: tx_out})
	}
	return utxos, nil
}

func (occ *OgmiosChainContext) SubmitTx(
	tx Transaction.Transaction,
) (serialization.TransactionId, error) {
	ctx := context.Background()
	bytes, err := tx.Bytes()
	if err != nil {
		log.Fatal(err, "OgmiosChainContext: SubmitTx: Error getting tx bytes")
	}
	_, err = occ.ogmigo.SubmitTx(ctx, hex.EncodeToString(bytes))
	if err != nil {
		log.Fatal(err, "OgmiosChainContext: SubmitTx: Error submitting tx")
	}
	txId, _ := tx.TransactionBody.Id()
	return txId, nil

}

func (occ *OgmiosChainContext) EvaluateTx(tx []uint8) (map[string]Redeemer.ExecutionUnits, error) {
	final_result := make(map[string]Redeemer.ExecutionUnits)
	ctx := context.Background()
	eval, err := occ.ogmigo.EvaluateTx(ctx, hex.EncodeToString(tx))
	if err != nil {
		log.Fatal(err, "OgmiosChainContext: EvaluateTx: Error evaluating tx")
	}
	for _, e := range eval.ExUnits {
		final_result[e.Validator.Purpose] = Redeemer.ExecutionUnits{
			Mem:   int64(e.Budget.Memory),
			Steps: int64(e.Budget.Cpu),
		}
	}
	return final_result, nil
}

// This is unused
func (occ *OgmiosChainContext) GetContractCbor(scriptHash string) (string, error) {
	//TODO
	return "", nil
}

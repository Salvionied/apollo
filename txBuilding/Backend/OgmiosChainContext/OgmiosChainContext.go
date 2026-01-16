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

	"github.com/Salvionied/apollo/v2/serialization"
	"github.com/Salvionied/apollo/v2/serialization/Address"
	"github.com/Salvionied/apollo/v2/serialization/Amount"
	"github.com/Salvionied/apollo/v2/serialization/Asset"
	"github.com/Salvionied/apollo/v2/serialization/AssetName"
	"github.com/Salvionied/apollo/v2/serialization/MultiAsset"
	"github.com/Salvionied/apollo/v2/serialization/PlutusData"
	"github.com/Salvionied/apollo/v2/serialization/Policy"
	"github.com/Salvionied/apollo/v2/serialization/Redeemer"
	"github.com/Salvionied/apollo/v2/serialization/Transaction"
	"github.com/Salvionied/apollo/v2/serialization/TransactionInput"
	"github.com/Salvionied/apollo/v2/serialization/TransactionOutput"
	"github.com/Salvionied/apollo/v2/serialization/UTxO"
	"github.com/Salvionied/apollo/v2/serialization/Value"
	"github.com/Salvionied/apollo/v2/txBuilding/Backend/Base"
	"github.com/Salvionied/apollo/v2/txBuilding/Utils"
	"github.com/SundaeSwap-finance/kugo"
	"github.com/SundaeSwap-finance/ogmigo/v6"
	"github.com/SundaeSwap-finance/ogmigo/v6/ouroboros/chainsync"
	"github.com/SundaeSwap-finance/ogmigo/v6/ouroboros/chainsync/num"
	"github.com/SundaeSwap-finance/ogmigo/v6/ouroboros/shared"

	"github.com/blinklabs-io/gouroboros/cbor"
)

type OgmiosChainContext struct {
	_epoch_info     Base.Epoch
	_epoch          int
	_Network        int
	_genesis_param  Base.GenesisParameters
	_protocol_param Base.ProtocolParameters
	ogmigo          *ogmigo.Client
	kugo            *kugo.Client
	// Optional base context used for requests. If nil, context.Background() is used.
	BaseContext context.Context
	// Optional per-request timeout. If zero, no timeout is applied.
	RequestTimeout time.Duration
}

func NewOgmiosChainContext(
	ogmigoClient *ogmigo.Client,
	kugoClient *kugo.Client,
) OgmiosChainContext {
	occ := OgmiosChainContext{
		ogmigo: ogmigoClient,
		kugo:   kugoClient,
	}
	return occ
}

// requestContext returns a context and cancel function for making requests.
// It uses BaseContext if set, otherwise context.Background().
// If RequestTimeout > 0, it wraps the context with a timeout.
func (occ *OgmiosChainContext) requestContext() (context.Context, func()) {
	ctx := occ.BaseContext
	if ctx == nil {
		ctx = context.Background()
	}
	if occ.RequestTimeout > 0 {
		reqCtx, cancel := context.WithTimeout(ctx, occ.RequestTimeout)
		return reqCtx, cancel
	}
	return ctx, func() {}
}

func (occ *OgmiosChainContext) Init() error {
	latestEpochs, err := occ.LatestEpoch()
	if err != nil {
		return fmt.Errorf("failed to get latest epoch: %w", err)
	}
	occ._epoch_info = latestEpochs
	//Init Genesis
	params := occ.GenesisParams()
	occ._genesis_param = params
	//init epoch
	latestParams, err := occ.LatestEpochParams()
	if err != nil {
		return fmt.Errorf("failed to get protocol params: %w", err)
	}
	occ._protocol_param = latestParams
	return nil
}

func multiAsset_OgmigoToApollo(
	m map[string]map[string]num.Int,
) MultiAsset.MultiAsset[int64] {
	if len(m) == 0 {
		return nil
	}
	assetMap := make(map[Policy.PolicyId]Asset.Asset[int64])
	for policy, tokens := range m {
		tokensMap := make(map[AssetName.AssetName]int64)
		for token, amt := range tokens {
			tokPtr := AssetName.NewAssetNameFromHexString(token)
			if tokPtr == nil {
				log.Printf(
					"multiAsset_OgmigoToApollo: skipping invalid asset name %q for policy %q",
					token,
					policy,
				)
				continue
			}
			tokensMap[*tokPtr] = amt.Int64()
		}
		pol := Policy.PolicyId{
			Value: policy,
		}
		assetMap[pol] = make(map[AssetName.AssetName]int64)
		assetMap[pol] = tokensMap
	}
	return assetMap
}

func valueOgmigoToApollo(v shared.Value) Value.AlonzoValue {
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

func datumOgmigoToApollo(
	d string,
	dh string,
) (*PlutusData.DatumOption, error) {
	if d != "" {
		datumBytes, err := hex.DecodeString(d)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to decode datum from hex %q: %w",
				d,
				err,
			)
		}
		var pd PlutusData.PlutusData
		_, err = cbor.Decode(datumBytes, &pd)
		if err != nil {
			return nil, fmt.Errorf(
				"datum is not valid plutus data %q: %w",
				d,
				err,
			)
		}
		res := PlutusData.DatumOptionInline(&pd)
		return &res, nil
	}
	if dh != "" {
		datumHashBytes, err := hex.DecodeString(dh)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to decode datum hash from hex %q: %w",
				dh,
				err,
			)
		}
		res := PlutusData.DatumOptionHash(datumHashBytes)
		return &res, nil
	}
	return nil, nil
}

func scriptRefOgmigoToApollo(
	script json.RawMessage,
) (*PlutusData.ScriptRef, error) {
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

func utxoOgmigoToApollo(u shared.Utxo) (UTxO.UTxO, error) {
	txHashRaw, err := hex.DecodeString(u.Transaction.ID)
	if err != nil {
		return UTxO.UTxO{}, fmt.Errorf(
			"failed to decode transaction ID: %w",
			err,
		)
	}
	addr, err := Address.DecodeAddress(u.Address)
	if err != nil {
		return UTxO.UTxO{}, fmt.Errorf(
			"failed to decode address: %w",
			err,
		)
	}
	datum, err := datumOgmigoToApollo(u.Datum, u.DatumHash)
	if err != nil {
		return UTxO.UTxO{}, fmt.Errorf("failed to convert datum: %w", err)
	}
	v := valueOgmigoToApollo(u.Value)
	scriptRef, err := scriptRefOgmigoToApollo(u.Script)
	if err != nil {
		return UTxO.UTxO{}, fmt.Errorf(
			"failed to convert script ref: %w",
			err,
		)
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
	}, nil
}

func (occ *OgmiosChainContext) GetUtxoFromRef(
	txHash string,
	index int,
) (*UTxO.UTxO, error) {
	reqCtx, cancel := occ.requestContext()
	defer cancel()
	utxos, err := occ.ogmigo.UtxosByTxIn(reqCtx, chainsync.TxInQuery{
		Transaction: shared.UtxoTxID{
			ID: txHash,
		},
		Index: uint32(index),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get utxo from ref: %w", err)
	}
	if len(utxos) == 0 {
		return nil, nil
	}
	apolloUtxo, err := utxoOgmigoToApollo(utxos[0])
	if err != nil {
		return nil, fmt.Errorf("failed to convert utxo: %w", err)
	}
	return &apolloUtxo, nil
}

func statequeryValue_toAddressAmount(v shared.Value) []Base.AddressAmount {
	// Start with capacity for lovelace + some assets
	amts := make([]Base.AddressAmount, 0, 1+len(v.AssetsExceptAda()))
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

func (occ *OgmiosChainContext) TxOuts(txHash string) []Base.Output {
	outs := make([]Base.Output, 1)
	moreUtxos := true
	chunkSize := 10
	for moreUtxos {
		queries := make([]chainsync.TxInQuery, chunkSize)
		for ix := range queries {
			queries[ix] = chainsync.TxInQuery{
				Transaction: shared.UtxoTxID{
					ID: txHash,
				},
				Index: uint32(ix),
			}
		}
		reqCtx, cancel := occ.requestContext()
		us, err := occ.ogmigo.UtxosByTxIn(reqCtx, queries...)
		cancel()
		if len(us) < chunkSize || err != nil {
			moreUtxos = false
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
func (occ *OgmiosChainContext) LatestBlock() (Base.Block, error) {
	reqCtx, cancel := occ.requestContext()
	defer cancel()
	point, err := occ.ogmigo.ChainTip(reqCtx)
	if err != nil {
		return Base.Block{}, fmt.Errorf(
			"failed to request chain tip: %w",
			err,
		)
	}
	s, ok := point.PointStruct()
	if !ok {
		return Base.Block{}, fmt.Errorf("expected a struct from chain tip")
	}
	return Base.Block{
		Hash: s.ID,
		Slot: int(s.Slot),
	}, nil
}

func (occ *OgmiosChainContext) LatestEpoch() (Base.Epoch, error) {
	reqCtx, cancel := occ.requestContext()
	defer cancel()
	current, err := occ.ogmigo.CurrentEpoch(reqCtx)
	if err != nil {
		return Base.Epoch{}, fmt.Errorf(
			"failed to request current epoch: %w",
			err,
		)
	}
	return Base.Epoch{
		Epoch: int(current),
	}, nil
}

func (occ *OgmiosChainContext) AddressUtxos(
	address string,
	gather bool,
) ([]Base.AddressUTXO, error) {
	reqCtx, cancel := occ.requestContext()
	defer cancel()
	addressUtxos := make([]Base.AddressUTXO, 0, 100)
	matches, err := occ.kugo.Matches(
		reqCtx,
		kugo.OnlyUnspent(),
		kugo.Address(address),
	)
	if err != nil {
		return nil, fmt.Errorf("kupo request failed: %w", err)
	}
	for _, match := range matches {
		datum := ""
		if match.DatumType == "inline" {
			datumCtx, datumCancel := occ.requestContext()
			datum, err = occ.kugo.Datum(datumCtx, match.DatumHash)
			datumCancel()
			if err != nil {
				return nil, fmt.Errorf(
					"kupo datum request failed: %w",
					err,
				)
			}
		}
		addressUtxos = append(addressUtxos, Base.AddressUTXO{
			TxHash:      match.TransactionID,
			OutputIndex: match.OutputIndex,
			Amount: statequeryValue_toAddressAmount(
				kugoValue_toSharedValue(match.Value),
			),
			// We probably don't need this info and kupo doesn't provide it in this query
			Block:       "",
			DataHash:    match.DatumHash,
			InlineDatum: datum,
		})
	}
	return addressUtxos, nil
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

func (occ *OgmiosChainContext) LatestEpochParams() (Base.ProtocolParameters, error) {
	reqCtx, cancel := occ.requestContext()
	defer cancel()
	pparams, err := occ.ogmigo.CurrentProtocolParameters(reqCtx)
	if err != nil {
		return Base.ProtocolParameters{}, fmt.Errorf(
			"protocol parameters request failed: %w",
			err,
		)
	}
	var ogmiosParams OgmiosProtocolParameters
	if err := json.Unmarshal(pparams, &ogmiosParams); err != nil {
		return Base.ProtocolParameters{}, fmt.Errorf(
			"failed to parse protocol parameters: %w",
			err,
		)
	}

	return Base.ProtocolParameters{
		MinFeeConstant:     int64(ogmiosParams.MinFeeConstant.Lovelace),
		MinFeeCoefficient:  int64(ogmiosParams.MinFeeCoefficient),
		MaxBlockSize:       int(ogmiosParams.MaxBlockSize.Bytes),
		MaxTxSize:          int(ogmiosParams.MaxTxSize.Bytes),
		MaxBlockHeaderSize: int(ogmiosParams.MaxBlockHeaderSize.Bytes),
		KeyDeposits: strconv.FormatUint(
			ogmiosParams.KeyDeposits.Lovelace,
			10,
		),
		PoolDeposits: strconv.FormatUint(
			ogmiosParams.PoolDeposits.Lovelace,
			10,
		),
		PooolInfluence:    ratio(ogmiosParams.PoolInfluence),
		MonetaryExpansion: ratio(ogmiosParams.MonetaryExpansion),
		TreasuryExpansion: ratio(ogmiosParams.TreasuryExpansion),
		// Unsure if ogmios reports this, but it's 0 on mainnet and
		// preview
		DecentralizationParam: 0,
		ExtraEntropy:          ogmiosParams.ExtraEntropy,
		MinUtxo: strconv.FormatUint(
			ogmiosParams.MinUtxoDepositConstant.Lovelace,
			10,
		),
		ProtocolMajorVersion: int(ogmiosParams.Version.Major),
		ProtocolMinorVersion: int(ogmiosParams.Version.Minor),
		MinPoolCost: strconv.FormatUint(
			ogmiosParams.MinStakePoolCost.Lovelace,
			10,
		),
		PriceMem: float32(
			ogmiosParams.ScriptExecutionPrices.Memory,
		),
		PriceStep: float32(ogmiosParams.ScriptExecutionPrices.Cpu),
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
		MaxBlockExSteps: strconv.FormatUint(
			ogmiosParams.MaxExecutionUnitsPerBlock.Cpu,
			10,
		),
		MaxValSize: strconv.FormatUint(
			ogmiosParams.MaxValSize.Bytes,
			10,
		),
		CollateralPercent:  int(ogmiosParams.CollateralPercentage),
		MaxCollateralInuts: int(ogmiosParams.MaxCollateralInputs),
		CoinsPerUtxoByte: strconv.FormatUint(
			ogmiosParams.MinUtxoDepositCoefficient,
			10,
		),
		// PerUtxoWord is deprecated https://cips.cardano.org/cips/cip55/
		CoinsPerUtxoWord: strconv.FormatUint(
			ogmiosParams.MinUtxoDepositCoefficient,
			10,
		),
		MaximumReferenceScriptsSize: int(
			ogmiosParams.MaximumReferenceScriptsSize,
		),
		MinFeeReferenceScriptsRange: int(
			ogmiosParams.MinFeeReferenceScripts.Range,
		),
		MinFeeReferenceScriptsBase: int(
			ogmiosParams.MinFeeReferenceScripts.Base,
		),
		MinFeeReferenceScriptsMultiplier: int(
			ogmiosParams.MinFeeReferenceScripts.Multiplier,
		),
		CostModels: ogmiosParams.CostModels,
	}, nil
}

func (occ *OgmiosChainContext) GenesisParams() Base.GenesisParameters {
	genesisParams := Base.GenesisParameters{}
	//TODO
	return genesisParams
}
func (occ *OgmiosChainContext) checkEpochAndUpdate() (bool, error) {
	if occ._epoch_info.EndTime <= int(time.Now().Unix()) {
		latestEpochs, err := occ.LatestEpoch()
		if err != nil {
			return false, err
		}
		occ._epoch_info = latestEpochs
		return true, nil
	}
	return false, nil
}

func (occ *OgmiosChainContext) Network() int {
	return occ._Network
}

func (occ *OgmiosChainContext) Epoch() (int, error) {
	updated, err := occ.checkEpochAndUpdate()
	if err != nil {
		return 0, fmt.Errorf("failed to check epoch: %w", err)
	}
	if updated {
		newEpoch, err := occ.LatestEpoch()
		if err != nil {
			return 0, fmt.Errorf("failed to get latest epoch: %w", err)
		}
		occ._epoch = newEpoch.Epoch
	}
	return occ._epoch, nil
}

// Seems unused
func (occ *OgmiosChainContext) LastBlockSlot() (int, error) {
	block, err := occ.LatestBlock()
	if err != nil {
		return 0, fmt.Errorf("failed to get latest block: %w", err)
	}
	return block.Slot, nil
}

func (occ *OgmiosChainContext) GetGenesisParams() (Base.GenesisParameters, error) {
	_, err := occ.checkEpochAndUpdate()
	if err != nil {
		return Base.GenesisParameters{}, fmt.Errorf(
			"failed to check epoch: %w",
			err,
		)
	}
	params := occ.GenesisParams()
	occ._genesis_param = params
	return occ._genesis_param, nil
}

func (occ *OgmiosChainContext) GetProtocolParams() (Base.ProtocolParameters, error) {
	updated, err := occ.checkEpochAndUpdate()
	if err != nil {
		return Base.ProtocolParameters{}, fmt.Errorf(
			"failed to check epoch: %w",
			err,
		)
	}
	if updated {
		latestParams, err := occ.LatestEpochParams()
		if err != nil {
			return Base.ProtocolParameters{}, fmt.Errorf(
				"failed to get protocol params: %w",
				err,
			)
		}
		occ._protocol_param = latestParams
	}
	return occ._protocol_param, nil
}

func (occ *OgmiosChainContext) MaxTxFee() (int, error) {
	protocol_param, err := occ.GetProtocolParams()
	if err != nil {
		return 0, err
	}
	maxTxExSteps, err := strconv.Atoi(protocol_param.MaxTxExSteps)
	if err != nil {
		return 0, fmt.Errorf("failed to parse MaxTxExSteps: %w", err)
	}
	maxTxExMem, err := strconv.Atoi(protocol_param.MaxTxExMem)
	if err != nil {
		return 0, fmt.Errorf("failed to parse MaxTxExMem: %w", err)
	}
	return Base.Fee(occ, protocol_param.MaxTxSize, maxTxExSteps, maxTxExMem)
}

// Copied from blockfrost context def since it just calls AddressUtxos and then
// converts
func (occ *OgmiosChainContext) Utxos(
	address Address.Address,
) ([]UTxO.UTxO, error) {
	results, err := occ.AddressUtxos(address.String(), true)
	if err != nil {
		return nil, fmt.Errorf("failed to get address utxos: %w", err)
	}
	utxos := make([]UTxO.UTxO, 0, len(results))
	for _, result := range results {
		decodedTxId, err := Utils.DecodeTxHash(result.TxHash)
		if err != nil {
			return nil, err
		}
		txIn := TransactionInput.TransactionInput{
			TransactionId: decodedTxId,
			Index:         result.OutputIndex,
		}
		lovelaceAmount, multiAssets, err := Utils.ParseAddressAmounts(
			result.Amount,
		)
		if err != nil {
			return nil, err
		}
		var finalAmount Value.Value
		if len(multiAssets) > 0 {
			finalAmount = Value.Value{
				Am: Amount.Amount{
					Coin:  lovelaceAmount,
					Value: multiAssets,
				},
				HasAssets: true,
			}
		} else {
			finalAmount = Value.Value{
				Coin:      lovelaceAmount,
				HasAssets: false,
			}
		}
		txOut, err := Utils.BuildTransactionOutput(
			address,
			finalAmount,
			result.DataHash,
			result.InlineDatum,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to build transaction output: %w",
				err,
			)
		}
		utxos = append(utxos, UTxO.UTxO{Input: txIn, Output: txOut})
	}
	return utxos, nil
}

func (occ *OgmiosChainContext) SubmitTx(
	tx Transaction.Transaction,
) (serialization.TransactionId, error) {
	txBytes, err := tx.Bytes()
	if err != nil {
		return serialization.TransactionId{}, fmt.Errorf(
			"failed to get tx bytes: %w",
			err,
		)
	}
	reqCtx, cancel := occ.requestContext()
	defer cancel()
	_, err = occ.ogmigo.SubmitTx(reqCtx, hex.EncodeToString(txBytes))
	if err != nil {
		return serialization.TransactionId{}, fmt.Errorf(
			"failed to submit tx: %w",
			err,
		)
	}
	txId, _ := tx.TransactionBody.Id()
	return txId, nil
}

func (occ *OgmiosChainContext) EvaluateTx(
	tx []uint8,
) (map[string]Redeemer.ExecutionUnits, error) {
	reqCtx, cancel := occ.requestContext()
	defer cancel()
	eval, err := occ.ogmigo.EvaluateTx(reqCtx, hex.EncodeToString(tx))
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate tx: %w", err)
	}
	finalResult := make(map[string]Redeemer.ExecutionUnits)
	for _, e := range eval.ExUnits {
		finalResult[e.Validator.Purpose] = Redeemer.ExecutionUnits{
			Mem:   uint64(e.Budget.Memory),
			Steps: uint64(e.Budget.Cpu),
		}
	}
	return finalResult, nil
}

// This is unused
func (occ *OgmiosChainContext) GetContractCbor(
	scriptHash string,
) (string, error) {
	//TODO
	return "", nil
}

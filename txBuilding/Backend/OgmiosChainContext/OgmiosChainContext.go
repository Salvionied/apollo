package OgmiosChainContext

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
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

// requestContext returns a context for making requests, applying the optional
// timeout if configured. The returned cancel function must always be called
// to release resources.
func (occ *OgmiosChainContext) requestContext() (context.Context, context.CancelFunc) {
	ctx := occ.BaseContext
	if ctx == nil {
		ctx = context.Background()
	}
	if occ.RequestTimeout > 0 {
		return context.WithTimeout(ctx, occ.RequestTimeout)
	}
	return ctx, func() {}
}

func (occ *OgmiosChainContext) Init() error {
	latestEpochs, err := occ.LatestEpoch()
	if err != nil {
		return fmt.Errorf("OgmiosChainContext: Init: %w", err)
	}
	occ._epoch_info = latestEpochs
	//Init Genesis
	params := occ.GenesisParams()
	occ._genesis_param = params
	//init epoch
	latestParams, err := occ.LatestEpochParams()
	if err != nil {
		return fmt.Errorf("OgmiosChainContext: Init: %w", err)
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

func datum_OgmigoToApollo(
	d string,
	dh string,
) (*PlutusData.DatumOption, error) {
	if d != "" {
		datumBytes, err := hex.DecodeString(d)
		if err != nil {
			return nil, fmt.Errorf(
				"OgmiosChainContext: failed to decode datum from hex %q: %w",
				d,
				err,
			)
		}
		var pd PlutusData.PlutusData
		err = cbor.Unmarshal(datumBytes, &pd)
		if err != nil {
			return nil, fmt.Errorf(
				"OgmiosChainContext: datum is not valid plutus data %q: %w",
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
				"OgmiosChainContext: failed to decode datum hash from hex %q: %w",
				dh,
				err,
			)
		}
		res := PlutusData.DatumOptionHash(datumHashBytes)
		return &res, nil
	}
	return nil, nil
}

func scriptRef_OgmigoToApollo(
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

func Utxo_OgmigoToApollo(u shared.Utxo) (UTxO.UTxO, error) {
	txHashRaw, err := hex.DecodeString(u.Transaction.ID)
	if err != nil {
		return UTxO.UTxO{}, fmt.Errorf(
			"OgmiosChainContext: failed to decode transaction ID %q: %w",
			u.Transaction.ID,
			err,
		)
	}
	addr, err := Address.DecodeAddress(u.Address)
	if err != nil {
		return UTxO.UTxO{}, fmt.Errorf(
			"OgmiosChainContext: failed to decode address %q: %w",
			u.Address,
			err,
		)
	}
	datum, err := datum_OgmigoToApollo(u.Datum, u.DatumHash)
	if err != nil {
		return UTxO.UTxO{}, err
	}
	v := value_OgmigoToApollo(u.Value)
	scriptRef, err := scriptRef_OgmigoToApollo(u.Script)
	if err != nil {
		return UTxO.UTxO{}, fmt.Errorf(
			"OgmiosChainContext: failed to convert script ref: %w",
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
	ctx, cancel := occ.requestContext()
	defer cancel()

	utxos, err := occ.ogmigo.UtxosByTxIn(ctx, chainsync.TxInQuery{
		Transaction: shared.UtxoTxID{
			ID: txHash,
		},
		Index: uint32(index),
	})
	if err != nil {
		return nil, fmt.Errorf(
			"OgmiosChainContext: GetUtxoFromRef: failed to query UTxO: %w",
			err,
		)
	}
	if len(utxos) == 0 {
		return nil, nil
	}
	apolloUtxo, err := Utxo_OgmigoToApollo(utxos[0])
	if err != nil {
		return nil, err
	}
	return &apolloUtxo, nil
}

func statequeryValue_toAddressAmount(v shared.Value) []Base.AddressAmount {
	assets := v.AssetsExceptAda()
	amts := make([]Base.AddressAmount, 0, 1+len(assets))
	amts = append(amts, Base.AddressAmount{
		Unit:     "lovelace",
		Quantity: strconv.FormatInt(v.AdaLovelace().Int64(), 10),
	})
	for policyId, tokenMap := range assets {
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
		ctx, cancel := occ.requestContext()
		us, err := occ.ogmigo.UtxosByTxIn(ctx, queries...)
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
	ctx, cancel := occ.requestContext()
	defer cancel()

	point, err := occ.ogmigo.ChainTip(ctx)
	if err != nil {
		return Base.Block{}, fmt.Errorf(
			"OgmiosChainContext: LatestBlock: failed to request chain tip: %w",
			err,
		)
	}
	s, ok := point.PointStruct()
	if !ok {
		return Base.Block{}, errors.New(
			"OgmiosChainContext: LatestBlock: expected a struct, got origin point",
		)
	}
	return Base.Block{
		Hash: s.ID,
		Slot: int(s.Slot),
	}, nil
}

func (occ *OgmiosChainContext) LatestEpoch() (Base.Epoch, error) {
	ctx, cancel := occ.requestContext()
	defer cancel()

	current, err := occ.ogmigo.CurrentEpoch(ctx)
	if err != nil {
		return Base.Epoch{}, fmt.Errorf(
			"OgmiosChainContext: LatestEpoch: failed to request current epoch: %w",
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
	ctx, cancel := occ.requestContext()
	defer cancel()

	matches, err := occ.kugo.Matches(
		ctx,
		kugo.OnlyUnspent(),
		kugo.Address(address),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"OgmiosChainContext: AddressUtxos: kupo request failed: %w",
			err,
		)
	}
	addressUtxos := make([]Base.AddressUTXO, 0, len(matches))
	for _, match := range matches {
		datum := ""
		if match.DatumType == "inline" {
			datum, err = occ.kugo.Datum(ctx, match.DatumHash)
			if err != nil {
				return nil, fmt.Errorf(
					"OgmiosChainContext: AddressUtxos: kupo datum request failed: %w",
					err,
				)
			}
		}
		addressUtxos = append(addressUtxos, Base.AddressUTXO{
			TxHash:      match.TransactionID,
			OutputIndex: match.OutputIndex,
			Amount: chainsyncValue_toAddressAmount(
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
	ctx, cancel := occ.requestContext()
	defer cancel()

	pparams, err := occ.ogmigo.CurrentProtocolParameters(ctx)
	if err != nil {
		return Base.ProtocolParameters{}, fmt.Errorf(
			"OgmiosChainContext: LatestEpochParams: protocol parameters request failed: %w",
			err,
		)
	}
	var ogmiosParams OgmiosProtocolParameters
	if err := json.Unmarshal(pparams, &ogmiosParams); err != nil {
		return Base.ProtocolParameters{}, fmt.Errorf(
			"OgmiosChainContext: LatestEpochParams: failed to parse protocol parameters: %w",
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
		CostModelsRaw: ogmiosParams.CostModels,
	}, nil
}

// ogmiosGenesisConfig represents the Ogmios v6 shelley genesis response.
type ogmiosGenesisConfig struct {
	ActiveSlotsCoefficient json.RawMessage `json:"activeSlotsCoefficient"`
	UpdateQuorum           int             `json:"updateQuorum"`
	MaxLovelaceSupply      json.Number     `json:"maxLovelaceSupply"`
	NetworkMagic           int             `json:"networkMagic"`
	EpochLength            int             `json:"epochLength"`
	StartTime              string          `json:"startTime"`
	SlotsPerKESPeriod      int             `json:"slotsPerKESPeriod"`
	SlotLength             slotLengthObj   `json:"slotLength"`
	MaxKESEvolutions       int             `json:"maxKESEvolutions"`
	SecurityParameter      int             `json:"securityParameter"`
}

// slotLengthObj represents the slot length object from Ogmios.
type slotLengthObj struct {
	Milliseconds int `json:"milliseconds"`
}

// fractionObj represents a fraction as an object with numerator and denominator.
type fractionObj struct {
	Numerator   int `json:"numerator"`
	Denominator int `json:"denominator"`
}

// parseActiveSlotsCoefficient parses the activeSlotsCoefficient field which
// can be a fraction object, a string fraction, or a plain float.
func parseActiveSlotsCoefficient(raw json.RawMessage) float32 {
	// Try parsing as fraction object {"numerator": N, "denominator": D}
	var frac fractionObj
	if err := json.Unmarshal(raw, &frac); err == nil && frac.Denominator != 0 {
		return float32(frac.Numerator) / float32(frac.Denominator)
	}

	// Try parsing as string fraction "N/D" using the existing ratio helper
	var strVal string
	if err := json.Unmarshal(raw, &strVal); err == nil {
		if result := ratio(strVal); result != 0 {
			return result
		}
	}

	// Try parsing as plain float
	var floatVal float64
	if err := json.Unmarshal(raw, &floatVal); err == nil {
		return float32(floatVal)
	}

	return 0
}

func (occ *OgmiosChainContext) GenesisParams() Base.GenesisParameters {
	ctx, cancel := occ.requestContext()
	defer cancel()

	rawConfig, err := occ.ogmigo.GenesisConfig(ctx, "shelley")
	if err != nil {
		fmt.Printf(
			"OgmiosChainContext: GenesisParams: failed to get genesis config: %v\n",
			err,
		)
		return Base.GenesisParameters{}
	}

	var config ogmiosGenesisConfig
	if err := json.Unmarshal(rawConfig, &config); err != nil {
		fmt.Printf(
			"OgmiosChainContext: GenesisParams: failed to parse genesis config: %v\n",
			err,
		)
		return Base.GenesisParameters{}
	}

	// Parse startTime from ISO 8601 to Unix timestamp
	systemStart := 0
	if config.StartTime != "" {
		t, err := time.Parse(time.RFC3339, config.StartTime)
		if err == nil {
			systemStart = int(t.Unix())
		}
	}

	// Convert slot length from milliseconds to seconds
	slotLength := config.SlotLength.Milliseconds / 1000

	return Base.GenesisParameters{
		ActiveSlotsCoefficient: parseActiveSlotsCoefficient(
			config.ActiveSlotsCoefficient,
		),
		UpdateQuorum:      config.UpdateQuorum,
		MaxLovelaceSupply: config.MaxLovelaceSupply.String(),
		NetworkMagic:      config.NetworkMagic,
		EpochLength:       config.EpochLength,
		SystemStart:       systemStart,
		SlotsPerKesPeriod: config.SlotsPerKESPeriod,
		SlotLength:        slotLength,
		MaxKesEvolutions:  config.MaxKESEvolutions,
		SecurityParam:     config.SecurityParameter,
	}
}

func (occ *OgmiosChainContext) _CheckEpochAndUpdate() (bool, error) {
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
	updated, err := occ._CheckEpochAndUpdate()
	if err != nil {
		return 0, err
	}
	if updated {
		newEpoch, err := occ.LatestEpoch()
		if err != nil {
			return 0, err
		}
		occ._epoch = newEpoch.Epoch
	}
	return occ._epoch, nil
}

// Seems unused
func (occ *OgmiosChainContext) LastBlockSlot() (int, error) {
	block, err := occ.LatestBlock()
	if err != nil {
		return 0, err
	}
	return block.Slot, nil
}

func (occ *OgmiosChainContext) GetGenesisParams() (Base.GenesisParameters, error) {
	updated, err := occ._CheckEpochAndUpdate()
	if err != nil {
		return Base.GenesisParameters{}, err
	}
	if updated {
		params := occ.GenesisParams()
		occ._genesis_param = params
	}
	return occ._genesis_param, nil
}

func (occ *OgmiosChainContext) GetProtocolParams() (Base.ProtocolParameters, error) {
	updated, err := occ._CheckEpochAndUpdate()
	if err != nil {
		return Base.ProtocolParameters{}, err
	}
	if updated {
		latestParams, err := occ.LatestEpochParams()
		if err != nil {
			return Base.ProtocolParameters{}, err
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
		return 0, fmt.Errorf(
			"MaxTxFee: invalid MaxTxExSteps %q: %w",
			protocol_param.MaxTxExSteps,
			err,
		)
	}
	maxTxExMem, err := strconv.Atoi(protocol_param.MaxTxExMem)
	if err != nil {
		return 0, fmt.Errorf(
			"MaxTxFee: invalid MaxTxExMem %q: %w",
			protocol_param.MaxTxExMem,
			err,
		)
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
		return nil, err
	}
	utxos := make([]UTxO.UTxO, 0, len(results))
	for _, result := range results {
		decodedTxId, err := hex.DecodeString(result.TxHash)
		if err != nil {
			return nil, fmt.Errorf(
				"OgmiosChainContext: Utxos: failed to decode tx hash %q: %w",
				result.TxHash,
				err,
			)
		}
		tx_in := TransactionInput.TransactionInput{
			TransactionId: decodedTxId,
			Index:         result.OutputIndex,
		}
		amount := result.Amount
		lovelace_amount := 0
		multi_assets := MultiAsset.MultiAsset[int64]{}
		for _, item := range amount {
			if item.Unit == "lovelace" {
				amt, err := strconv.Atoi(item.Quantity)
				if err != nil {
					return nil, fmt.Errorf(
						"OgmiosChainContext: Utxos: failed to parse lovelace quantity %q: %w",
						item.Quantity,
						err,
					)
				}
				lovelace_amount += amt
			} else {
				asset_quantity, err := strconv.ParseInt(item.Quantity, 10, 64)
				if err != nil {
					return nil, fmt.Errorf(
						"OgmiosChainContext: Utxos: failed to parse asset quantity %q: %w",
						item.Quantity,
						err,
					)
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
				Am: Amount.Amount{
					Coin:  int64(lovelace_amount),
					Value: multi_assets,
				},
				HasAssets: true,
			}
		} else {
			final_amount = Value.Value{Coin: int64(lovelace_amount), HasAssets: false}
		}
		datum_hash := serialization.DatumHash{}
		if result.DataHash != "" && result.InlineDatum == "" {
			decoded_hash, err := hex.DecodeString(result.DataHash)
			if err != nil {
				return nil, fmt.Errorf(
					"OgmiosChainContext: Utxos: invalid data hash %q: %w",
					result.DataHash,
					err,
				)
			}
			datum_hash = serialization.DatumHash{Payload: decoded_hash}
		}
		var tx_out TransactionOutput.TransactionOutput
		if result.InlineDatum != "" {
			decoded, err := hex.DecodeString(result.InlineDatum)
			if err != nil {
				return nil, fmt.Errorf(
					"OgmiosChainContext: Utxos: failed to decode inline datum: %w",
					err,
				)
			}
			var x PlutusData.PlutusData
			err = cbor.Unmarshal(decoded, &x)
			if err != nil {
				return nil, fmt.Errorf(
					"OgmiosChainContext: Utxos: failed to unmarshal inline datum: %w",
					err,
				)
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
	txBytes, err := tx.Bytes()
	if err != nil {
		return serialization.TransactionId{}, fmt.Errorf(
			"OgmiosChainContext: SubmitTx: failed to get tx bytes: %w",
			err,
		)
	}

	ctx, cancel := occ.requestContext()
	defer cancel()

	_, err = occ.ogmigo.SubmitTx(ctx, hex.EncodeToString(txBytes))
	if err != nil {
		return serialization.TransactionId{}, fmt.Errorf(
			"OgmiosChainContext: SubmitTx: failed to submit tx: %w",
			err,
		)
	}
	txId, _ := tx.TransactionBody.Id()
	return txId, nil
}

func (occ *OgmiosChainContext) EvaluateTx(
	tx []uint8,
) (map[string]Redeemer.ExecutionUnits, error) {
	ctx, cancel := occ.requestContext()
	defer cancel()

	eval, err := occ.ogmigo.EvaluateTx(ctx, hex.EncodeToString(tx))
	if err != nil {
		return nil, fmt.Errorf(
			"OgmiosChainContext: EvaluateTx: failed to evaluate tx: %w",
			err,
		)
	}

	result := make(map[string]Redeemer.ExecutionUnits)
	for _, e := range eval.ExUnits {
		result[e.Validator.Purpose] = Redeemer.ExecutionUnits{
			Mem:   int64(e.Budget.Memory),
			Steps: int64(e.Budget.Cpu),
		}
	}
	return result, nil
}

// EvaluateTxWithAdditionalUtxos is not supported by Ogmios.
// Returns an error if non-empty additional UTxOs are provided.
func (occ *OgmiosChainContext) EvaluateTxWithAdditionalUtxos(
	tx []uint8,
	utxos []UTxO.UTxO,
) (map[string]Redeemer.ExecutionUnits, error) {
	if len(utxos) > 0 {
		return nil, fmt.Errorf(
			"OgmiosChainContext does not support" +
				" additional UTxOs for evaluation",
		)
	}
	return occ.EvaluateTx(tx)
}

func int64sToInts(vals []int64) PlutusData.CostModel {
	cm := make(PlutusData.CostModel, len(vals))
	for i, v := range vals {
		if v > int64(math.MaxInt) {
			cm[i] = math.MaxInt
		} else if v < int64(math.MinInt) {
			cm[i] = math.MinInt
		} else {
			cm[i] = int(v)
		}
	}
	return cm
}

// CostModelsV1 returns the Plutus V1 cost model from
// protocol parameters, or nil if unavailable.
func (occ *OgmiosChainContext) CostModelsV1() PlutusData.CostModel {
	pp, err := occ.GetProtocolParams()
	if err != nil {
		return nil
	}
	if pp.CostModels != nil {
		if cm, ok := pp.CostModels[Base.CostModelsPlutusV1]; ok {
			return cm
		}
	}
	if pp.CostModelsRaw != nil {
		if v, ok := pp.CostModelsRaw["plutus:v1"]; ok {
			return int64sToInts(v)
		}
	}
	return nil
}

// CostModelsV2 returns the Plutus V2 cost model from
// protocol parameters, or nil if unavailable.
func (occ *OgmiosChainContext) CostModelsV2() PlutusData.CostModel {
	pp, err := occ.GetProtocolParams()
	if err != nil {
		return nil
	}
	if pp.CostModels != nil {
		if cm, ok := pp.CostModels[Base.CostModelsPlutusV2]; ok {
			return cm
		}
	}
	if pp.CostModelsRaw != nil {
		if v, ok := pp.CostModelsRaw["plutus:v2"]; ok {
			return int64sToInts(v)
		}
	}
	return nil
}

// CostModelsV3 returns the Plutus V3 cost model from
// protocol parameters, or nil if unavailable.
func (occ *OgmiosChainContext) CostModelsV3() PlutusData.CostModel {
	pp, err := occ.GetProtocolParams()
	if err != nil {
		return nil
	}
	if pp.CostModels != nil {
		if cm, ok := pp.CostModels[Base.CostModelsPlutusV3]; ok {
			return cm
		}
	}
	if pp.CostModelsRaw != nil {
		if v, ok := pp.CostModelsRaw["plutus:v3"]; ok {
			return int64sToInts(v)
		}
	}
	return nil
}

// This is unused
func (occ *OgmiosChainContext) GetContractCbor(
	scriptHash string,
) (string, error) {
	//TODO
	return "", nil
}

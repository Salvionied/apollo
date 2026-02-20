package UtxorpcChainContext

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"time"

	"connectrpc.com/connect"
	"github.com/Salvionied/apollo/serialization"
	"github.com/Salvionied/apollo/serialization/Address"
	"github.com/Salvionied/apollo/serialization/Redeemer"
	"github.com/Salvionied/apollo/serialization/Transaction"
	"github.com/Salvionied/apollo/serialization/TransactionInput"
	"github.com/Salvionied/apollo/serialization/TransactionOutput"
	"github.com/Salvionied/apollo/serialization/UTxO"
	"github.com/Salvionied/apollo/txBuilding/Backend/Base"
	cardanopb "github.com/utxorpc/go-codegen/utxorpc/v1alpha/cardano"
	"github.com/utxorpc/go-codegen/utxorpc/v1alpha/query"
	utxorpc "github.com/utxorpc/go-sdk"
	"github.com/utxorpc/go-sdk/cardano"
)

// bigIntToInt64 extracts an int64 value from a protobuf BigInt.
// BigInt can contain: Int (int64), BigUInt (unsigned bytes), or BigNInt (negative bytes).
func bigIntToInt64(b *cardanopb.BigInt) (int64, error) {
	if b == nil {
		return 0, errors.New("BigInt is nil")
	}
	switch v := b.GetBigInt().(type) {
	case *cardanopb.BigInt_Int:
		return v.Int, nil
	case *cardanopb.BigInt_BigUInt:
		bi := new(big.Int).SetBytes(v.BigUInt)
		if !bi.IsInt64() {
			return 0, fmt.Errorf("BigUInt value %s overflows int64", bi.String())
		}
		return bi.Int64(), nil
	case *cardanopb.BigInt_BigNInt:
		bi := new(big.Int).SetBytes(v.BigNInt)
		bi.Neg(bi)
		if !bi.IsInt64() {
			return 0, fmt.Errorf("BigNInt value %s overflows int64", bi.String())
		}
		return bi.Int64(), nil
	default:
		return 0, errors.New("BigInt has no value set")
	}
}

// bigIntToUint64 extracts a uint64 value from a protobuf BigInt.
func bigIntToUint64(b *cardanopb.BigInt) (uint64, error) {
	if b == nil {
		return 0, errors.New("BigInt is nil")
	}
	switch v := b.GetBigInt().(type) {
	case *cardanopb.BigInt_Int:
		if v.Int < 0 {
			return 0, fmt.Errorf("negative Int value %d cannot convert to uint64", v.Int)
		}
		return uint64(v.Int), nil
	case *cardanopb.BigInt_BigUInt:
		bi := new(big.Int).SetBytes(v.BigUInt)
		if !bi.IsUint64() {
			return 0, fmt.Errorf("BigUInt value %s overflows uint64", bi.String())
		}
		return bi.Uint64(), nil
	case *cardanopb.BigInt_BigNInt:
		return 0, errors.New("negative BigNInt cannot convert to uint64")
	default:
		return 0, errors.New("BigInt has no value set")
	}
}

// bigIntToString converts a protobuf BigInt to its string representation.
func bigIntToString(b *cardanopb.BigInt) (string, error) {
	if b == nil {
		return "", errors.New("BigInt is nil")
	}
	switch v := b.GetBigInt().(type) {
	case *cardanopb.BigInt_Int:
		return strconv.FormatInt(v.Int, 10), nil
	case *cardanopb.BigInt_BigUInt:
		bi := new(big.Int).SetBytes(v.BigUInt)
		return bi.String(), nil
	case *cardanopb.BigInt_BigNInt:
		bi := new(big.Int).SetBytes(v.BigNInt)
		bi.Neg(bi)
		return bi.String(), nil
	default:
		return "", errors.New("BigInt has no value set")
	}
}

// Protocol parameter parsing helpers that wrap conversion with consistent error formatting
func parseParamInt64(b *cardanopb.BigInt, field string) (int64, error) {
	val, err := bigIntToInt64(b)
	if err != nil {
		return 0, fmt.Errorf("GetProtocolParams: %s: %w", field, err)
	}
	return val, nil
}

func parseParamUint64(b *cardanopb.BigInt, field string) (uint64, error) {
	val, err := bigIntToUint64(b)
	if err != nil {
		return 0, fmt.Errorf("GetProtocolParams: %s: %w", field, err)
	}
	return val, nil
}

func parseParamString(b *cardanopb.BigInt, field string) (string, error) {
	val, err := bigIntToString(b)
	if err != nil {
		return "", fmt.Errorf("GetProtocolParams: %s: %w", field, err)
	}
	return val, nil
}

type UtxorpcChainContext struct {
	_Network        int
	_protocol_param Base.ProtocolParameters
	_genesis_param  Base.GenesisParameters
	client          *cardano.Client
	latestUpdate    time.Time
}

// Interface requirements (no UTxO RPC equivalent, yet)
func (u *UtxorpcChainContext) GetContractCbor(
	scriptHash string,
) (string, error) {
	return "", nil
}

func NewUtxorpcChainContext(
	baseUrl string,
	network int,
	dmtrApiKey ...string,
) (UtxorpcChainContext, error) {
	var _dmtrApiKey string
	if len(dmtrApiKey) > 0 {
		_dmtrApiKey = dmtrApiKey[0]
	}
	// Check config
	if baseUrl == "" && _dmtrApiKey == "" {
		return UtxorpcChainContext{}, errors.New(
			"provide either a URL or a Demeter API key",
		)
	}

	var networkString string
	switch network {
	case 0:
		networkString = "mainnet"
	case 1:
		networkString = "testnet" // ?
	case 2:
		networkString = "preview"
	case 3:
		networkString = "preprod"
	default:
		return UtxorpcChainContext{}, errors.New("invalid network")
	}
	if baseUrl == "" {
		baseUrl = fmt.Sprintf(
			"https://%s.utxorpc-v0.demeter.run",
			networkString,
		)
	}
	cardanoclient := cardano.NewClient(utxorpc.WithBaseUrl(baseUrl))
	if _dmtrApiKey != "" {
		cardanoclient.UtxorpcClient.SetHeader("dmtr-api-key", _dmtrApiKey)
	}
	u := UtxorpcChainContext{
		client: cardanoclient, _Network: network,
	}
	err := u.init()
	return u, err
}

func (u *UtxorpcChainContext) init() error {
	_, err := u.GetProtocolParams()
	return err
}

func (u *UtxorpcChainContext) Network() int {
	return u._Network
}

func (u *UtxorpcChainContext) GetGenesisParams() (Base.GenesisParameters, error) {
	if time.Since(u.latestUpdate) > time.Minute*5 {
		genesisParams := Base.GenesisParameters{}
		gpFromApi, err := u.client.UtxorpcClient.ReadGenesis(
			connect.NewRequest(&query.ReadGenesisRequest{}),
		)
		if err != nil {
			return genesisParams, err
		}
		gpCardano := gpFromApi.Msg.GetCardano()
		// Map the fields
		if gpCardano.GetActiveSlotsCoeff() != nil {
			genesisParams.ActiveSlotsCoefficient = float32(
				gpCardano.GetActiveSlotsCoeff().GetNumerator(),
			) / float32(
				gpCardano.GetActiveSlotsCoeff().GetDenominator(),
			)
		}
		genesisParams.UpdateQuorum = int(gpCardano.GetUpdateQuorum())
		maxLovelaceSupply, err := bigIntToString(gpCardano.GetMaxLovelaceSupply())
		if err != nil {
			return genesisParams, fmt.Errorf("GetGenesisParams: MaxLovelaceSupply: %w", err)
		}
		genesisParams.MaxLovelaceSupply = maxLovelaceSupply
		genesisParams.NetworkMagic = int(gpCardano.GetNetworkMagic())
		genesisParams.EpochLength = int(gpCardano.GetEpochLength())
		// SystemStart is a string timestamp, need to parse to unix timestamp
		// For now, leave as 0 or parse if needed
		genesisParams.SlotsPerKesPeriod = int(gpCardano.GetSlotsPerKesPeriod())
		genesisParams.SlotLength = int(gpCardano.GetSlotLength())
		genesisParams.MaxKesEvolutions = int(gpCardano.GetMaxKesEvolutions())
		genesisParams.SecurityParam = int(gpCardano.GetSecurityParam())

		u._genesis_param = genesisParams
		u.latestUpdate = time.Now()
		return genesisParams, nil
	}
	return u._genesis_param, nil
}

func (u *UtxorpcChainContext) GetProtocolParams() (Base.ProtocolParameters, error) {
	if time.Since(u.latestUpdate) > time.Minute*5 {
		protocolParams := Base.ProtocolParameters{}
		ppFromApi, err := u.client.UtxorpcClient.ReadParams(
			connect.NewRequest(&query.ReadParamsRequest{}),
		)
		if err != nil {
			return protocolParams, err
		}
		ppCardano := ppFromApi.Msg.GetValues().GetCardano()
		if ppCardano == nil {
			return protocolParams, errors.New("GetProtocolParams: API returned nil Cardano parameters")
		}
		// Map ALL the fields
		minFeeConstant, err := parseParamInt64(ppCardano.GetMinFeeConstant(), "MinFeeConstant")
		if err != nil {
			return protocolParams, err
		}
		minFeeCoefficient, err := parseParamInt64(ppCardano.GetMinFeeCoefficient(), "MinFeeCoefficient")
		if err != nil {
			return protocolParams, err
		}
		protocolParams.MinFeeConstant = minFeeConstant
		protocolParams.MinFeeCoefficient = minFeeCoefficient
		protocolParams.MaxTxSize = int(ppCardano.GetMaxTxSize())
		protocolParams.MaxBlockSize = int(ppCardano.GetMaxBlockBodySize())
		protocolParams.MaxBlockHeaderSize = int(
			ppCardano.GetMaxBlockHeaderSize(),
		)
		stakeKeyDeposit, err := parseParamUint64(ppCardano.GetStakeKeyDeposit(), "StakeKeyDeposit")
		if err != nil {
			return protocolParams, err
		}
		poolDeposit, err := parseParamUint64(ppCardano.GetPoolDeposit(), "PoolDeposit")
		if err != nil {
			return protocolParams, err
		}
		protocolParams.KeyDeposits = strconv.FormatUint(stakeKeyDeposit, 10)
		protocolParams.PoolDeposits = strconv.FormatUint(poolDeposit, 10)
		if ppCardano.GetPoolInfluence().GetDenominator() != 0 {
			protocolParams.PooolInfluence = float32(
				uint32(
					ppCardano.GetPoolInfluence().GetNumerator(),
				) / ppCardano.GetPoolInfluence().
					GetDenominator(),
			)
		} else {
			protocolParams.PooolInfluence = 0
		}
		if ppCardano.GetMonetaryExpansion().GetDenominator() != 0 {
			protocolParams.MonetaryExpansion = float32(
				uint32(
					ppCardano.GetMonetaryExpansion().GetNumerator(),
				) / ppCardano.GetMonetaryExpansion().
					GetDenominator(),
			)
		} else {
			protocolParams.MonetaryExpansion = 0
		}
		if ppCardano.GetTreasuryExpansion().GetDenominator() != 0 {
			protocolParams.TreasuryExpansion = float32(
				uint32(
					ppCardano.GetTreasuryExpansion().GetNumerator(),
				) / ppCardano.GetTreasuryExpansion().
					GetDenominator(),
			)
		} else {
			protocolParams.TreasuryExpansion = 0
		}
		protocolParams.DecentralizationParam = 0
		protocolParams.ExtraEntropy = ""
		protocolParams.ProtocolMajorVersion = int(
			ppCardano.GetProtocolVersion().GetMajor(),
		)
		protocolParams.ProtocolMinorVersion = int(
			ppCardano.GetProtocolVersion().GetMinor(),
		)
		//CHECK HERE
		//protocolParams.MinUtxo = ppFromApi.Data.
		minPoolCost, err := parseParamString(ppCardano.GetMinPoolCost(), "MinPoolCost")
		if err != nil {
			return protocolParams, err
		}
		protocolParams.MinPoolCost = minPoolCost
		if ppCardano.GetPrices().GetMemory().GetDenominator() != 0 {
			protocolParams.PriceMem = float32(
				uint32(
					ppCardano.GetPrices().GetMemory().GetNumerator(),
				) / ppCardano.GetPrices().
					GetMemory().
					GetDenominator(),
			)
		} else {
			protocolParams.PriceMem = 0
		}
		if ppCardano.GetPrices().GetSteps().GetDenominator() != 0 {
			protocolParams.PriceStep = float32(
				uint32(
					ppCardano.GetPrices().GetSteps().GetNumerator(),
				) / ppCardano.GetPrices().
					GetSteps().
					GetDenominator(),
			)
		} else {
			protocolParams.PriceStep = 0
		}
		protocolParams.MaxTxExMem = strconv.FormatUint(
			ppCardano.GetMaxExecutionUnitsPerTransaction().GetMemory(),
			10,
		)
		protocolParams.MaxTxExSteps = strconv.FormatUint(
			ppCardano.GetMaxExecutionUnitsPerTransaction().GetSteps(),
			10,
		)
		protocolParams.MaxBlockExMem = strconv.FormatUint(
			ppCardano.GetMaxExecutionUnitsPerBlock().GetMemory(),
			10,
		)
		protocolParams.MaxBlockExSteps = strconv.FormatUint(
			ppCardano.GetMaxExecutionUnitsPerBlock().GetSteps(),
			10,
		)
		protocolParams.MaxValSize = strconv.FormatUint(
			ppCardano.GetMaxValueSize(),
			10,
		)
		protocolParams.CollateralPercent = int(
			ppCardano.GetCollateralPercentage(),
		)
		protocolParams.MaxCollateralInuts = int(
			ppCardano.GetMaxCollateralInputs(),
		)
		coinsPerUtxoByte, err := parseParamString(ppCardano.GetCoinsPerUtxoByte(), "CoinsPerUtxoByte")
		if err != nil {
			return protocolParams, err
		}
		protocolParams.CoinsPerUtxoByte = coinsPerUtxoByte
		protocolParams.CoinsPerUtxoWord = "0"
		protocolParams.CostModels = map[string][]int64{
			"PlutusV1": ppCardano.GetCostModels().GetPlutusV1().GetValues(),
			"PlutusV2": ppCardano.GetCostModels().GetPlutusV2().GetValues(),
			"PlutusV3": ppCardano.GetCostModels().GetPlutusV3().GetValues(),
		}
		u._protocol_param = protocolParams
		u.latestUpdate = time.Now()
	}
	return u._protocol_param, nil
}

func (u *UtxorpcChainContext) GetUtxoFromRef(
	txHash string,
	txIndex int,
) (*UTxO.UTxO, error) {
	resp, err := u.client.GetUtxoByRef(txHash, uint32(txIndex))
	if err != nil {
		return nil, fmt.Errorf("failed to read UTxO: %w", err)
	}

	if len(resp.Msg.Items) == 0 {
		return nil, nil // Not found
	}

	item := resp.Msg.Items[0]
	tmpUtxo := UTxO.UTxO{}
	tmpUtxo.Input = TransactionInput.TransactionInput{
		TransactionId: item.TxoRef.Hash,
		Index:         int(item.TxoRef.Index),
	}
	tmpOutput := TransactionOutput.TransactionOutput{}
	err = tmpOutput.UnmarshalCBOR(item.NativeBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal output: %w", err)
	}
	tmpUtxo.Output = tmpOutput
	return &tmpUtxo, nil
}

func (u *UtxorpcChainContext) EvaluateTx(
	txBytes []byte,
) (map[string]Redeemer.ExecutionUnits, error) {
	resp, err := u.client.EvaluateTransaction(hex.EncodeToString(txBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate transaction: %w", err)
	}

	// Parse the response
	result := make(map[string]Redeemer.ExecutionUnits)
	report := resp.Msg.GetReport()
	if report != nil {
		if cardanoEval := report.GetCardano(); cardanoEval != nil {
			tagStrings := []string{"spend", "mint", "cert", "wdrl"}
			for _, r := range cardanoEval.Redeemers {
				tagStr := tagStrings[r.Purpose]
				key := fmt.Sprintf("%s:%d", tagStr, r.Index)
				result[key] = Redeemer.ExecutionUnits{
					Steps: int64(r.ExUnits.Steps),
					Mem:   int64(r.ExUnits.Memory),
				}
			}
		}
	}
	return result, nil
}

func (u *UtxorpcChainContext) Epoch() (int, error) {
	resp, err := u.client.GetTip()
	if err != nil {
		return 0, fmt.Errorf("failed to read tip: %w", err)
	}
	slot := resp.Msg.Tip.Slot
	epochLength := uint64(432000) // Mainnet epoch length
	if u._Network != 0 {
		epochLength = 86400 // Testnet
	}
	epoch := int(slot / epochLength)
	return epoch, nil
}

func (u *UtxorpcChainContext) MaxTxFee() (int, error) {
	protocol_param, err := u.GetProtocolParams()
	if err != nil {
		return 0, err
	}
	maxTxExSteps, err := strconv.Atoi(protocol_param.MaxTxExSteps)
	if err != nil {
		return 0, err
	}
	maxTxExMem, err := strconv.Atoi(protocol_param.MaxTxExMem)
	if err != nil {
		return 0, err
	}
	return Base.Fee(u, protocol_param.MaxTxSize, maxTxExSteps, maxTxExMem)
}

func (u *UtxorpcChainContext) LastBlockSlot() (int, error) {
	ppFromApi, err := u.client.UtxorpcClient.ReadParams(
		connect.NewRequest(&query.ReadParamsRequest{}),
	)
	if err != nil {
		return 0, err
	}
	return int(ppFromApi.Msg.GetLedgerTip().GetSlot()), nil
}

func (u *UtxorpcChainContext) Utxos(
	address Address.Address,
) ([]UTxO.UTxO, error) {
	resp, err := u.client.GetUtxosByAddress(address.Bytes())
	if err != nil {
		return nil, fmt.Errorf("failed to get UTxOs: %w", err)
	}

	ret := make([]UTxO.UTxO, 0, len(resp.Msg.Items))
	for _, item := range resp.Msg.Items {
		tmpUtxo := UTxO.UTxO{}
		tmpUtxo.Input = TransactionInput.TransactionInput{
			TransactionId: item.TxoRef.Hash,
			Index:         int(item.TxoRef.Index),
		}
		tmpOutput := TransactionOutput.TransactionOutput{}
		err = tmpOutput.UnmarshalCBOR(item.NativeBytes)
		if err != nil {
			return ret, fmt.Errorf("failed to unmarshal output: %w", err)
		}
		tmpUtxo.Output = tmpOutput
		ret = append(ret, tmpUtxo)
	}
	return ret, nil
}

func (u *UtxorpcChainContext) SubmitTx(
	tx Transaction.Transaction,
) (serialization.TransactionId, error) {
	txBytes, err := tx.Bytes()
	if err != nil {
		return serialization.TransactionId{}, fmt.Errorf(
			"failed to encode transaction: %w",
			err,
		)
	}

	resp, err := u.client.SubmitTransaction(hex.EncodeToString(txBytes))
	if err != nil {
		return serialization.TransactionId{}, fmt.Errorf(
			"failed to submit transaction: %w",
			err,
		)
	}

	return serialization.TransactionId{Payload: resp.Msg.GetRef()}, nil
}

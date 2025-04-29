package UtxorpcChainContext

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"time"

	"github.com/Salvionied/apollo/serialization"
	"github.com/Salvionied/apollo/serialization/Address"
	"github.com/Salvionied/apollo/serialization/Redeemer"
	"github.com/Salvionied/apollo/serialization/Transaction"
	"github.com/Salvionied/apollo/serialization/TransactionInput"
	"github.com/Salvionied/apollo/serialization/TransactionOutput"
	"github.com/Salvionied/apollo/serialization/UTxO"
	"github.com/Salvionied/apollo/txBuilding/Backend/Base"
	utxorpc "github.com/utxorpc/go-sdk"
)

type UtxorpcChainContext struct {
	_epoch_info     Base.Epoch
	_epoch          int
	_Network        int
	_genesis_param  Base.GenesisParameters
	_protocol_param Base.ProtocolParameters
	client          *utxorpc.UtxorpcClient
	latestUpdate    time.Time
}

// Interface requirements
func (u *UtxorpcChainContext) EvaluateTx([]uint8) (map[string]Redeemer.ExecutionUnits, error) {}
func (u *UtxorpcChainContext) GetUtxoFromRef(txHash string, txIndex int) (*UTxO.UTxO, error)  {}
func (u *UtxorpcChainContext) GetContractCbor(scriptHash string) (string, error)              {}

func NewUtxorpcChainContext(baseUrl string, network int, dmtrApiKey ...string) (UtxorpcChainContext, error) {
	var _dmtrApiKey string
	if len(dmtrApiKey) > 0 {
		_dmtrApiKey = dmtrApiKey[0]
	}
	// Check config
	if baseUrl == "" && _dmtrApiKey == "" {
		return UtxorpcChainContext{}, fmt.Errorf("You must provide either a URL or a Demeter API key")
	}

	var networkString string
	if network == 0 {
		networkString = "mainnet"
	} else if network == 1 {
		networkString = "testnet" // ?
	} else if network == 2 {
		networkString = "preview"
	} else if network == 3 {
		networkString = "preprod"
	} else {
		return UtxorpcChainContext{}, fmt.Errorf("Invalid network")
	}
	if baseUrl == "" {
		baseUrl = fmt.Sprintf("https://%s.utxorpc-v0.demeter.run", networkString)
	}
	utxorpcClient := utxorpc.NewClient(utxorpc.WithBaseUrl(baseUrl))
	if _dmtrApiKey != "" {
		utxorpcClient.SetHeader("dmtr-api-key", _dmtrApiKey)
	}
	u := UtxorpcChainContext{
		client: utxorpcClient, _Network: network,
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
	genesisParams := Base.GenesisParameters{}
	// NO GENESIS PARAMS IN UTXORPC
	return genesisParams, nil
}

func (u *UtxorpcChainContext) GetProtocolParams() (Base.ProtocolParameters, error) {
	if time.Since(u.latestUpdate) > time.Minute*5 {
		protocolParams := Base.ProtocolParameters{}
		ppFromApi, err := u.client.ReadParams()
		if err != nil {
			return protocolParams, err
		}
		ppCardano := ppFromApi.Msg.GetValues().GetCardano()
		// Map ALL the fields
		protocolParams.MinFeeConstant = int(ppCardano.GetMinFeeConstant())
		protocolParams.MinFeeCoefficient = int(ppCardano.GetMinFeeCoefficient())
		protocolParams.MaxTxSize = int(ppCardano.GetMaxTxSize())
		protocolParams.MaxBlockSize = int(ppCardano.GetMaxBlockBodySize())
		protocolParams.MaxBlockHeaderSize = int(ppCardano.GetMaxBlockHeaderSize())
		protocolParams.KeyDeposits = fmt.Sprintf("%d", ppCardano.GetStakeKeyDeposit())
		protocolParams.PoolDeposits = fmt.Sprintf("%d", ppCardano.GetPoolDeposit())
		protocolParams.PooolInfluence = float32(
			uint32(ppCardano.GetPoolInfluence().GetNumerator()) / ppCardano.GetPoolInfluence().GetDenominator(),
		)
		protocolParams.MonetaryExpansion = float32(
			uint32(ppCardano.GetMonetaryExpansion().GetNumerator()) / ppCardano.GetMonetaryExpansion().GetDenominator(),
		)
		protocolParams.TreasuryExpansion = float32(
			uint32(ppCardano.GetTreasuryExpansion().GetNumerator()) / ppCardano.GetTreasuryExpansion().GetDenominator(),
		)
		protocolParams.DecentralizationParam = 0
		protocolParams.ExtraEntropy = ""
		protocolParams.ProtocolMajorVersion = int(ppCardano.GetProtocolVersion().GetMajor())
		protocolParams.ProtocolMinorVersion = int(ppCardano.GetProtocolVersion().GetMinor())
		//CHECK HERE
		//protocolParams.MinUtxo = ppFromApi.Data.
		protocolParams.MinPoolCost = fmt.Sprint("%d", ppCardano.GetMinPoolCost())
		protocolParams.PriceMem = float32(
			uint32(ppCardano.GetPrices().GetMemory().GetNumerator()) / ppCardano.GetPrices().GetMemory().GetDenominator(),
		)
		protocolParams.PriceStep = float32(
			uint32(ppCardano.GetPrices().GetSteps().GetNumerator()) / ppCardano.GetPrices().GetSteps().GetDenominator(),
		)
		protocolParams.MaxTxExMem = fmt.Sprintf("%d", ppCardano.GetMaxExecutionUnitsPerTransaction().GetMemory())
		protocolParams.MaxTxExSteps = fmt.Sprint("%d", ppCardano.GetMaxExecutionUnitsPerTransaction().GetSteps())
		protocolParams.MaxBlockExMem = fmt.Sprintf("%d", ppCardano.GetMaxExecutionUnitsPerBlock().GetMemory())
		protocolParams.MaxBlockExSteps = fmt.Sprintf("%d", ppCardano.GetMaxExecutionUnitsPerBlock().GetSteps())
		protocolParams.MaxValSize = fmt.Sprintf("%d", ppCardano.GetMaxValueSize())
		protocolParams.CollateralPercent = int(ppCardano.GetCollateralPercentage())
		protocolParams.MaxCollateralInuts = int(ppCardano.GetMaxCollateralInputs())
		protocolParams.CoinsPerUtxoByte = fmt.Sprint(ppCardano.GetCoinsPerUtxoByte())
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

func (u *UtxorpcChainContext) Epoch() (int, error) {
	// TODO
	return 0, nil
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
	ppFromApi, err := u.client.ReadParams()
	if err != nil {
		return 0, err
	}
	return int(ppFromApi.Msg.GetLedgerTip().GetSlot()), nil
}

func (u *UtxorpcChainContext) Utxos(address Address.Address) ([]UTxO.UTxO, error) {
	addrCbor, err := address.MarshalCBOR()
	ret := []UTxO.UTxO{}
	var tmpUtxo UTxO.UTxO
	resp, err := u.client.GetUtxosByAddress(addrCbor)
	if err != nil {
		return ret, err
	}
	for _, item := range resp.Msg.GetItems() {
		tmpUtxo.Input = TransactionInput.TransactionInput{
			TransactionId: item.GetTxoRef().GetHash(),
			Index:         int(item.GetTxoRef().GetIndex()),
		}
		tmpOutput := TransactionOutput.TransactionOutput{}
		err = tmpOutput.UnmarshalCBOR(item.GetNativeBytes())
		if err != nil {
			return ret, err
		}
		tmpUtxo.Output = tmpOutput
		ret = append(ret, tmpUtxo)
	}
	return ret, nil
}

func (u *UtxorpcChainContext) SubmitTx(tx Transaction.Transaction) (serialization.TransactionId, error) {
	txBytes, err := tx.Bytes()
	if err != nil {
		return serialization.TransactionId{Payload: []byte{}}, err
	}
	_, err = u.client.SubmitTx(hex.EncodeToString(txBytes))
	if err != nil {
		return serialization.TransactionId{Payload: []byte{}}, err
	}
	return tx.Id(), nil
}

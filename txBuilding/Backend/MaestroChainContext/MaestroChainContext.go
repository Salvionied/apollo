package MaestroChainContext

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Salvionied/apollo/serialization"
	"github.com/Salvionied/apollo/serialization/Address"
	"github.com/Salvionied/apollo/serialization/Redeemer"
	"github.com/Salvionied/apollo/serialization/Transaction"
	"github.com/Salvionied/apollo/serialization/TransactionInput"
	"github.com/Salvionied/apollo/serialization/TransactionOutput"
	"github.com/Salvionied/apollo/serialization/UTxO"
	"github.com/Salvionied/apollo/txBuilding/Backend/Base"
	"github.com/fxamacker/cbor/v2"
	"github.com/maestro-org/go-sdk/client"
	"github.com/maestro-org/go-sdk/utils"
)

type MaestroChainContext struct {
	_epoch_info     Base.Epoch
	_epoch          int
	_Network        int
	_genesis_param  Base.GenesisParameters
	_protocol_param Base.ProtocolParameters
	client          *client.Client
	latestUpdate    time.Time
}

func NewMaestroChainContext(
	network int,
	projectId string,
) (MaestroChainContext, error) {
	var networkString string
	switch network {
	case 0:
		networkString = "mainnet"
	case 1:
		networkString = "testnet"
	case 2:
		networkString = "preview"
	case 3:
		networkString = "preprod"
	default:
		return MaestroChainContext{}, errors.New("invalid network")
	}
	maestroClient := client.NewClient(projectId, networkString)
	mcc := MaestroChainContext{
		client: maestroClient, _Network: network,
	}
	err := mcc.Init()
	return mcc, err
}
func (mcc *MaestroChainContext) Init() error {
	latest_epochs, err := mcc.LatestEpoch()
	if err != nil {
		return err
	}
	mcc._epoch_info = latest_epochs
	params, err := mcc.GenesisParams()
	if err != nil {
		return err
	}
	mcc._genesis_param = params
	latest_params, err := mcc.LatestEpochParams()
	if err != nil {
		return err
	}
	mcc._protocol_param = latest_params
	return nil
}

func (mcc *MaestroChainContext) LatestBlock() (Base.Block, error) {
	latestBlock := Base.Block{}
	latestBlockFromApi, err := mcc.client.LatestBlock()
	if err != nil {
		return latestBlock, err
	}
	if latestBlockFromApi == nil {
		return latestBlock, nil
	} else {
		tmpTime, _ := time.Parse("2006-01-02 15:04:05", latestBlockFromApi.Data.Timestamp)
		latestBlock.Time = int(tmpTime.Unix())
		latestBlock.Height = int(latestBlockFromApi.Data.Height)
		latestBlock.Hash = latestBlockFromApi.Data.Hash
		latestBlock.Slot = int(latestBlockFromApi.Data.AbsoluteSlot)
		latestBlock.Epoch = int(latestBlockFromApi.Data.Epoch)
		latestBlock.EpochSlot = int(latestBlockFromApi.Data.EpochSlot)
		latestBlock.SlotLeader = latestBlockFromApi.Data.BlockProducer
		latestBlock.Size = int(latestBlockFromApi.Data.Size)
		latestBlock.TxCount = len(latestBlockFromApi.Data.TxHashes)
		latestBlock.Output = latestBlockFromApi.Data.TotalOutputLovelace
		latestBlock.Fees = strconv.FormatInt(latestBlockFromApi.Data.TotalFees, 10)
		latestBlock.BlockVRF = latestBlockFromApi.Data.VrfKey
		latestBlock.PreviousBlock = latestBlockFromApi.Data.PreviousBlock
		latestBlock.NextBlock = latestBlockFromApi.Data.Hash
		latestBlock.Confirmations = int(latestBlockFromApi.Data.Confirmations)
	}
	return latestBlock, nil
}

func (mcc *MaestroChainContext) LatestEpoch() (Base.Epoch, error) {
	epoch := Base.Epoch{}
	latestEpoch, err := mcc.client.CurrentEpoch()
	if err != nil {
		return epoch, err
	}
	epoch.ActiveStake = ""
	epoch.BlockCount = int(latestEpoch.Data.BlkCount)
	epoch.EndTime = int(latestEpoch.LastUpdated.BlockSlot)
	epoch.Fees = latestEpoch.Data.Fees
	epoch.FirstBlockTime = int(latestEpoch.Data.StartTime)
	epoch.StartTime = int(latestEpoch.Data.StartTime)
	epoch.TxCount = int(latestEpoch.Data.TxCount)
	return epoch, nil

}

func parseMaestroFloat(floatString string) float32 {
	if floatString == "" {
		return 0
	}
	splitString := strings.Split(floatString, "/")
	top := splitString[0]
	bottom := splitString[1]
	topFloat, _ := strconv.ParseFloat(top, 32)
	bottomFloat, _ := strconv.ParseFloat(bottom, 32)
	return float32(topFloat / bottomFloat)
}

func (mcc *MaestroChainContext) LatestEpochParams() (Base.ProtocolParameters, error) {
	protocolParams := Base.ProtocolParameters{}
	ppFromApi, err := mcc.client.ProtocolParameters()
	if err != nil {
		return protocolParams, err
	}
	// Map ALL the fields
	protocolParams.MinFeeConstant = int64(
		ppFromApi.Data.MinFeeConstant.LovelaceAmount.Lovelace,
	)
	protocolParams.MinFeeCoefficient = int64(ppFromApi.Data.MinFeeCoefficient)
	protocolParams.MaxTxSize = int(ppFromApi.Data.MaxTransactionSize.Bytes)
	protocolParams.MaxBlockSize = int(ppFromApi.Data.MaxBlockBodySize.Bytes)
	protocolParams.MaxBlockHeaderSize = int(
		ppFromApi.Data.MaxBlockHeaderSize.Bytes,
	)
	protocolParams.KeyDeposits = strconv.FormatInt(
		ppFromApi.Data.StakeCredentialDeposit.LovelaceAmount.Lovelace,
		10,
	)
	protocolParams.PoolDeposits = strconv.FormatInt(
		ppFromApi.Data.StakePoolDeposit.LovelaceAmount.Lovelace,
		10,
	)
	parsedPoolInfl, _ := strconv.ParseFloat(
		ppFromApi.Data.StakePoolPledgeInfluence,
		32,
	)
	protocolParams.PooolInfluence = float32(parsedPoolInfl)
	monExp, _ := strconv.ParseFloat(ppFromApi.Data.MonetaryExpansion, 32)
	protocolParams.MonetaryExpansion = float32(monExp)
	tresExp, _ := strconv.ParseFloat(ppFromApi.Data.TreasuryExpansion, 32)
	protocolParams.TreasuryExpansion = float32(tresExp)
	protocolParams.DecentralizationParam = 0
	protocolParams.ExtraEntropy = ""
	protocolParams.ProtocolMajorVersion = int(
		ppFromApi.Data.ProtocolVersion.Major,
	)
	protocolParams.ProtocolMinorVersion = int(
		ppFromApi.Data.ProtocolVersion.Minor,
	)
	//CHECK HERE
	//protocolParams.MinUtxo = ppFromApi.Data.
	protocolParams.MinPoolCost = strconv.FormatInt(
		ppFromApi.Data.MinStakePoolCost.LovelaceAmount.Lovelace,
		10,
	)
	protocolParams.PriceMem = parseMaestroFloat(
		ppFromApi.Data.ScriptExecutionPrices.Memory,
	)
	protocolParams.PriceStep = parseMaestroFloat(
		ppFromApi.Data.ScriptExecutionPrices.Steps,
	)
	protocolParams.MaxTxExMem = strconv.FormatInt(
		ppFromApi.Data.MaxExecutionUnitsPerTransaction.Memory,
		10,
	)
	protocolParams.MaxTxExSteps = strconv.FormatInt(
		ppFromApi.Data.MaxExecutionUnitsPerTransaction.Steps,
		10,
	)
	protocolParams.MaxBlockExMem = strconv.FormatInt(
		ppFromApi.Data.MaxExecutionUnitsPerBlock.Memory,
		10,
	)
	protocolParams.MaxBlockExSteps = strconv.FormatInt(
		ppFromApi.Data.MaxExecutionUnitsPerBlock.Steps,
		10,
	)
	protocolParams.MaxValSize = strconv.FormatInt(
		ppFromApi.Data.MaxValueSize.Bytes,
		10,
	)
	protocolParams.CollateralPercent = int(ppFromApi.Data.CollateralPercentage)
	protocolParams.MaxCollateralInuts = int(ppFromApi.Data.MaxCollateralInputs)
	protocolParams.CoinsPerUtxoByte = strconv.FormatInt(
		ppFromApi.Data.MinUtxoDepositCoefficient,
		10,
	)
	protocolParams.CoinsPerUtxoWord = "0"
	//protocolParams.CostModels = ppFromApi.Data.CostModels
	return protocolParams, nil
}

func (mcc *MaestroChainContext) GenesisParams() (Base.GenesisParameters, error) {
	genesisParams := Base.GenesisParameters{}
	// NO GENESIS PARAMS IN MAESTRO
	return genesisParams, nil
}

func (mcc *MaestroChainContext) Network() int {
	return mcc._Network
}

func (mcc *MaestroChainContext) Epoch() (int, error) {
	if time.Since(mcc.latestUpdate) > time.Minute*5 {
		new_epoch, err := mcc.LatestEpoch()
		if err != nil {
			return 0, err
		}

		mcc._epoch = new_epoch.Epoch
	}
	return mcc._epoch, nil
}

func (mcc *MaestroChainContext) LastBlockSlot() (int, error) {
	block, err := mcc.LatestBlock()
	if err != nil {
		return 0, err
	}
	return block.Slot, nil
}

func (mcc *MaestroChainContext) GetGenesisParams() (Base.GenesisParameters, error) {
	if time.Since(mcc.latestUpdate) > time.Minute*5 {
		params, err := mcc.GenesisParams()
		if err != nil {
			return Base.GenesisParameters{}, err
		}
		mcc._genesis_param = params
	}
	return mcc._genesis_param, nil
}

func (mcc *MaestroChainContext) GetProtocolParams() (Base.ProtocolParameters, error) {
	if time.Since(mcc.latestUpdate) > time.Minute*5 {
		latest_params, err := mcc.LatestEpochParams()
		if err != nil {
			return Base.ProtocolParameters{}, err
		}
		mcc._protocol_param = latest_params
		mcc.latestUpdate = time.Now()
	}
	return mcc._protocol_param, nil
}

func (mcc *MaestroChainContext) MaxTxFee() (int, error) {
	protocol_param, err := mcc.GetProtocolParams()
	if err != nil {
		return 0, err
	}
	maxTxExSteps, err := strconv.Atoi(protocol_param.MaxTxExSteps)
	if err != nil {
		return 0, fmt.Errorf("MaxTxFee: invalid MaxTxExSteps %q: %w", protocol_param.MaxTxExSteps, err)
	}
	maxTxExMem, err := strconv.Atoi(protocol_param.MaxTxExMem)
	if err != nil {
		return 0, fmt.Errorf("MaxTxFee: invalid MaxTxExMem %q: %w", protocol_param.MaxTxExMem, err)
	}
	return Base.Fee(mcc, protocol_param.MaxTxSize, maxTxExSteps, maxTxExMem)
}
func (mcc *MaestroChainContext) TxOuts(txHash string) ([]Base.Output, error) {
	tx, err := mcc.client.TransactionDetails(txHash)
	if err != nil {
		return nil, err
	}
	outputs := make([]Base.Output, 0)
	for idx, txOut := range tx.Data.Outputs {
		amount := []Base.AddressAmount{}
		for _, addrAmount := range txOut.Assets {
			amount = append(amount, Base.AddressAmount{
				Unit:     addrAmount.Unit,
				Quantity: strconv.FormatInt(addrAmount.Amount, 10),
			})
		}
		output := Base.Output{
			Address:             txOut.Address,
			OutputIndex:         idx,
			ReferenceScriptHash: txOut.ReferenceScript.Hash,
			Amount:              amount,
		}
		outputs = append(outputs, output)
	}
	return outputs, nil
}

func (mcc *MaestroChainContext) GetUtxoFromRef(
	txHash string,
	index int,
) (*UTxO.UTxO, error) {
	var utxo *UTxO.UTxO
	params := utils.NewParameters()
	params.WithCbor()
	txOutputByRef, err := mcc.client.TransactionOutputFromReference(
		txHash,
		index,
		params,
	)
	if err != nil {
		return utxo, err
	}
	decodedCbor, err := hex.DecodeString(txOutputByRef.Data.TxOutCbor)
	if err != nil {
		return nil, fmt.Errorf("GetUtxoFromRef: invalid CBOR hex: %w", err)
	}
	output := TransactionOutput.TransactionOutput{}
	err = cbor.Unmarshal(decodedCbor, &output)
	if err != nil {
		return nil, fmt.Errorf("GetUtxoFromRef: failed to unmarshal CBOR: %w", err)
	}
	decodedHash, err := hex.DecodeString(txHash)
	if err != nil {
		return nil, fmt.Errorf("GetUtxoFromRef: invalid tx hash %q: %w", txHash, err)
	}
	utxo = &UTxO.UTxO{
		Input: TransactionInput.TransactionInput{
			TransactionId: decodedHash,
			Index:         index,
		},
		Output: output,
	}
	return utxo, nil
}

func (mcc *MaestroChainContext) AddressUtxos(
	address string,
	gather bool,
) ([]Base.AddressUTXO, error) {
	addressUtxos := make([]Base.AddressUTXO, 0)
	params := utils.NewParameters()
	params.ResolveDatums()
	utxosAtAddressAtApi, err := mcc.client.UtxosAtAddress(address, params)
	if err != nil {
		return addressUtxos, err
	}

	for _, maestroUtxo := range utxosAtAddressAtApi.Data {
		assets := make([]Base.AddressAmount, 0)
		for _, asset := range maestroUtxo.Assets {
			assets = append(assets, Base.AddressAmount{
				Unit:     asset.Unit,
				Quantity: strconv.FormatInt(asset.Amount, 10),
			})
		}
		utxo := Base.AddressUTXO{
			Amount:      assets,
			OutputIndex: int(maestroUtxo.Index),
			TxHash:      maestroUtxo.TxHash,
			InlineDatum: fmt.Sprint(maestroUtxo.Datum),
		}
		addressUtxos = append(addressUtxos, utxo)
	}

	if gather {
		for utxosAtAddressAtApi.NextCursor != "" {
			params.Cursor(utxosAtAddressAtApi.NextCursor)
			utxosAtAddressAtApi, err = mcc.client.UtxosAtAddress(
				address,
				params,
			)
			if err != nil {
				return addressUtxos, err
			}
			for _, maestroUtxo := range utxosAtAddressAtApi.Data {
				assets := make([]Base.AddressAmount, 0)
				for _, asset := range maestroUtxo.Assets {
					assets = append(assets, Base.AddressAmount{
						Unit:     asset.Unit,
						Quantity: strconv.FormatInt(asset.Amount, 10),
					})
				}
				utxo := Base.AddressUTXO{
					Amount:      assets,
					OutputIndex: int(maestroUtxo.Index),
					TxHash:      maestroUtxo.TxHash,
					InlineDatum: fmt.Sprint(maestroUtxo.Datum),
				}
				addressUtxos = append(addressUtxos, utxo)
			}
		}
	}

	return addressUtxos, nil

}

func (mcc *MaestroChainContext) Utxos(
	address Address.Address,
) ([]UTxO.UTxO, error) {
	utxos := make([]UTxO.UTxO, 0)
	params := utils.NewParameters()
	params.WithCbor()
	params.ResolveDatums()
	utxosAtAddressAtApi, err := mcc.client.UtxosAtAddress(
		address.String(),
		params,
	)
	if err != nil {
		return utxos, err
	}

	for _, maestroUtxo := range utxosAtAddressAtApi.Data {
		utxo := UTxO.UTxO{}
		decodedHash, err := hex.DecodeString(maestroUtxo.TxHash)
		if err != nil {
			return nil, fmt.Errorf("Utxos: invalid tx hash %q: %w", maestroUtxo.TxHash, err)
		}
		utxo.Input = TransactionInput.TransactionInput{
			TransactionId: decodedHash,
			Index:         int(maestroUtxo.Index),
		}
		output := TransactionOutput.TransactionOutput{}
		decodedCbor, err := hex.DecodeString(maestroUtxo.TxOutCbor)
		if err != nil {
			return nil, fmt.Errorf("Utxos: invalid CBOR hex: %w", err)
		}
		err = cbor.Unmarshal(decodedCbor, &output)
		if err != nil {
			return nil, fmt.Errorf("Utxos: failed to unmarshal CBOR: %w", err)
		}
		utxo.Output = output
		utxos = append(utxos, utxo)
	}

	for utxosAtAddressAtApi.NextCursor != "" {
		params.Cursor(utxosAtAddressAtApi.NextCursor)
		utxosAtAddressAtApi, err = mcc.client.UtxosAtAddress(
			address.String(),
			params,
		)
		if err != nil {
			return utxos, err
		}
		for _, maestroUtxo := range utxosAtAddressAtApi.Data {
			utxo := UTxO.UTxO{}
			decodedHash, err := hex.DecodeString(maestroUtxo.TxHash)
			if err != nil {
				return nil, fmt.Errorf("Utxos: invalid tx hash %q: %w", maestroUtxo.TxHash, err)
			}
			utxo.Input = TransactionInput.TransactionInput{
				TransactionId: decodedHash,
				Index:         int(maestroUtxo.Index),
			}
			output := TransactionOutput.TransactionOutput{}
			decodedCbor, err := hex.DecodeString(maestroUtxo.TxOutCbor)
			if err != nil {
				return nil, fmt.Errorf("Utxos: invalid CBOR hex: %w", err)
			}
			err = cbor.Unmarshal(decodedCbor, &output)
			if err != nil {
				return nil, fmt.Errorf("Utxos: failed to unmarshal CBOR: %w", err)
			}
			utxo.Output = output
			utxos = append(utxos, utxo)
		}
	}

	return utxos, nil
}

func (mcc *MaestroChainContext) SubmitTx(
	tx Transaction.Transaction,
) (serialization.TransactionId, error) {
	txBytes, err := tx.Bytes()
	if err != nil {
		return serialization.TransactionId{}, err
	}
	txHex := hex.EncodeToString(txBytes)
	resp, err := mcc.client.SubmitTx(txHex)
	if err != nil {

		return serialization.TransactionId{}, err
	}
	decodedResponseHash, err := hex.DecodeString(resp.Data)
	if err != nil {
		return serialization.TransactionId{}, fmt.Errorf("SubmitTx: invalid response hash: %w", err)
	}
	return serialization.TransactionId{
		Payload: decodedResponseHash,
	}, nil
}

type EvalResult struct {
	Result map[string]map[string]int `json:"EvaluationResult"`
}

type ExecutionResult struct {
	Result EvalResult `json:"result"`
}

func (mcc *MaestroChainContext) EvaluateTx(
	tx []byte,
) (map[string]Redeemer.ExecutionUnits, error) {
	final_result := make(map[string]Redeemer.ExecutionUnits)
	encodedTx := hex.EncodeToString(tx)
	evaluation, err := mcc.client.EvaluateTx(encodedTx)
	if err != nil {
		return final_result, err
	}
	for _, eval := range evaluation {
		final_result[eval.RedeemerTag+":"+strconv.Itoa(eval.RedeemerIndex)] = Redeemer.ExecutionUnits{
			Mem:   eval.ExUnits.Mem,
			Steps: eval.ExUnits.Steps,
		}
	}
	return final_result, nil
}

func (mcc *MaestroChainContext) GetContractCbor(
	scriptHash string,
) (string, error) {
	res, err := mcc.client.ScriptByHash(scriptHash)
	if err != nil {
		return "", err
	}
	scCborBytes := res.Data.Bytes
	bytes := []byte{}
	decodedBytes, _ := hex.DecodeString(scCborBytes)
	_ = cbor.Unmarshal(decodedBytes, &bytes)
	return hex.EncodeToString(bytes), nil

}

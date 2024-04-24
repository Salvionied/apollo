package MaestroChainContext

import (
	"encoding/hex"
	"fmt"
	"log"
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
	"github.com/Salvionied/cbor/v2"
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

func NewMaestroChainContext(network int, projectId string) (MaestroChainContext, error) {
	networkString := "mainnet"
	if network == 0 {
		networkString = "mainnet"
	} else if network == 1 {
		networkString = "testnet"
	} else if network == 2 {
		networkString = "preview"
	} else if network == 3 {
		networkString = "pre-prod"
	} else {
		return MaestroChainContext{}, fmt.Errorf("Invalid network")
	}
	maestroClient := client.NewClient(projectId, networkString)
	mcc := MaestroChainContext{
		client: maestroClient, _Network: network,
	}
	mcc.Init()
	return mcc, nil
}
func (mcc *MaestroChainContext) Init() {
	latest_epochs := mcc.LatestEpoch()
	mcc._epoch_info = latest_epochs
	params := mcc.GenesisParams()
	mcc._genesis_param = params
	latest_params := mcc.LatestEpochParams()
	mcc._protocol_param = latest_params
}

func (mcc *MaestroChainContext) LatestBlock() Base.Block {
	latestBlock := Base.Block{}
	latestBlockFromApi, _ := mcc.client.LatestBlock()
	if latestBlockFromApi == nil {
		return latestBlock
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
		latestBlock.Fees = fmt.Sprint(latestBlockFromApi.Data.TotalFees)
		latestBlock.BlockVRF = latestBlockFromApi.Data.VrfKey
		latestBlock.PreviousBlock = latestBlockFromApi.Data.PreviousBlock
		latestBlock.NextBlock = latestBlockFromApi.Data.Hash
		latestBlock.Confirmations = int(latestBlockFromApi.Data.Confirmations)
	}
	return latestBlock
}

func (mcc *MaestroChainContext) LatestEpoch() Base.Epoch {
	epoch := Base.Epoch{}
	latestEpoch, err := mcc.client.CurrentEpoch()
	if err != nil {
		return epoch
	}
	epoch.ActiveStake = ""
	epoch.BlockCount = int(latestEpoch.Data.BlkCount)
	epoch.EndTime = int(latestEpoch.LastUpdated.BlockSlot)
	epoch.Fees = latestEpoch.Data.Fees
	epoch.FirstBlockTime = int(latestEpoch.Data.StartTime)
	epoch.StartTime = int(latestEpoch.Data.StartTime)
	epoch.TxCount = int(latestEpoch.Data.TxCount)
	return epoch

}

func parseMaestroFloat(floatString string) float32 {
	splitString := strings.Split(floatString, "/")
	top := splitString[0]
	bottom := splitString[1]
	topFloat, _ := strconv.ParseFloat(top, 32)
	bottomFloat, _ := strconv.ParseFloat(bottom, 32)
	return float32(topFloat / bottomFloat)
}

func (mcc *MaestroChainContext) LatestEpochParams() Base.ProtocolParameters {
	protocolParams := Base.ProtocolParameters{}
	ppFromApi, err := mcc.client.ProtocolParameters()
	if err != nil {
		log.Fatal(err)
	}
	// Map ALL the fields
	protocolParams.MinFeeConstant = int(ppFromApi.Data.MinFeeConstant)
	protocolParams.MinFeeCoefficient = int(ppFromApi.Data.MinFeeCoefficient)
	protocolParams.MaxTxSize = int(ppFromApi.Data.MaxTxSize)
	protocolParams.MaxBlockSize = int(ppFromApi.Data.MaxBlockBodySize)
	protocolParams.MaxBlockHeaderSize = int(ppFromApi.Data.MaxBlockHeaderSize)
	protocolParams.KeyDeposits = fmt.Sprint(ppFromApi.Data.StakeKeyDeposit)
	protocolParams.PoolDeposits = fmt.Sprint(ppFromApi.Data.PoolDeposit)
	parsedPoolInfl, _ := strconv.ParseFloat(ppFromApi.Data.PoolInfluence, 32)
	protocolParams.PooolInfluence = float32(parsedPoolInfl)
	monExp, _ := strconv.ParseFloat(ppFromApi.Data.MonetaryExpansion, 32)
	protocolParams.MonetaryExpansion = float32(monExp)
	tresExp, _ := strconv.ParseFloat(ppFromApi.Data.TreasuryExpansion, 32)
	protocolParams.TreasuryExpansion = float32(tresExp)
	protocolParams.DecentralizationParam = 0
	protocolParams.ExtraEntropy = ""
	protocolParams.ProtocolMajorVersion = int(ppFromApi.Data.ProtocolVersion.Major)
	protocolParams.ProtocolMinorVersion = int(ppFromApi.Data.ProtocolVersion.Minor)
	//CHECK HERE
	//protocolParams.MinUtxo = ppFromApi.Data.
	protocolParams.MinPoolCost = fmt.Sprint(ppFromApi.Data.MinPoolCost)
	protocolParams.PriceMem = parseMaestroFloat(ppFromApi.Data.Prices.Memory)
	protocolParams.PriceStep = parseMaestroFloat(ppFromApi.Data.Prices.Steps)
	protocolParams.MaxTxExMem = fmt.Sprint(ppFromApi.Data.MaxExecutionUnitsPerTransaction.Memory)
	protocolParams.MaxTxExSteps = fmt.Sprint(ppFromApi.Data.MaxExecutionUnitsPerTransaction.Steps)
	protocolParams.MaxBlockExMem = fmt.Sprint(ppFromApi.Data.MaxExecutionUnitsPerBlock.Memory)
	protocolParams.MaxBlockExSteps = fmt.Sprint(ppFromApi.Data.MaxExecutionUnitsPerBlock.Steps)
	protocolParams.MaxValSize = fmt.Sprint(ppFromApi.Data.MaxValueSize)
	protocolParams.CollateralPercent = int(ppFromApi.Data.CollateralPercentage)
	protocolParams.MaxCollateralInuts = int(ppFromApi.Data.MaxCollateralInputs)
	protocolParams.CoinsPerUtxoByte = fmt.Sprint(ppFromApi.Data.CoinsPerUtxoByte)
	protocolParams.CoinsPerUtxoWord = "0"
	//protocolParams.CostModels = ppFromApi.Data.CostModels
	return protocolParams
}

func (mcc *MaestroChainContext) GenesisParams() Base.GenesisParameters {
	genesisParams := Base.GenesisParameters{}
	// NO GENESIS PARAMS IN MAESTRO
	return genesisParams
}

func (mcc *MaestroChainContext) Network() int {
	return mcc._Network
}

func (mcc *MaestroChainContext) Epoch() int {
	if time.Since(mcc.latestUpdate) > time.Minute*5 {
		new_epoch := mcc.LatestEpoch()
		mcc._epoch = new_epoch.Epoch
	}
	return mcc._epoch
}

func (mcc *MaestroChainContext) LastBlockSlot() int {
	block := mcc.LatestBlock()
	return block.Slot
}

func (mcc *MaestroChainContext) GetGenesisParams() Base.GenesisParameters {
	if time.Since(mcc.latestUpdate) > time.Minute*5 {
		params := mcc.GenesisParams()
		mcc._genesis_param = params
	}
	return mcc._genesis_param
}

func (mcc *MaestroChainContext) GetProtocolParams() Base.ProtocolParameters {
	if time.Since(mcc.latestUpdate) > time.Minute*5 {
		latest_params := mcc.LatestEpochParams()
		mcc._protocol_param = latest_params
		mcc.latestUpdate = time.Now()
	}
	return mcc._protocol_param
}

func (mcc *MaestroChainContext) MaxTxFee() int {
	protocol_param := mcc.GetProtocolParams()
	maxTxExSteps, _ := strconv.Atoi(protocol_param.MaxTxExSteps)
	maxTxExMem, _ := strconv.Atoi(protocol_param.MaxTxExMem)
	return Base.Fee(mcc, protocol_param.MaxTxSize, maxTxExSteps, maxTxExMem)
}
func (mcc *MaestroChainContext) TxOuts(txHash string) []Base.Output {
	tx, err := mcc.client.TransactionDetails(txHash)
	if err != nil {
		log.Fatal(err)
	}
	outputs := make([]Base.Output, 0)
	for idx, txOut := range tx.Data.Outputs {
		amount := []Base.AddressAmount{}
		for _, addrAmount := range txOut.Assets {
			amount = append(amount, Base.AddressAmount{
				Unit:     addrAmount.Unit,
				Quantity: fmt.Sprint(addrAmount.Amount),
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
	return outputs
}
func (mcc *MaestroChainContext) GetUtxoFromRef(txHash string, index int) *UTxO.UTxO {
	var utxo *UTxO.UTxO
	params := utils.NewParameters()
	params.WithCbor()
	txOutputByRef, err := mcc.client.TransactionOutputFromReference(txHash, index, params)
	if err != nil {
		return utxo
	}
	decodedCbor, _ := hex.DecodeString(txOutputByRef.Data.TxOutCbor)
	output := TransactionOutput.TransactionOutput{}
	err = cbor.Unmarshal(decodedCbor, &output)
	if err != nil {
		log.Fatal(err)
		return nil
	}
	decodedHash, _ := hex.DecodeString(txHash)
	utxo = &UTxO.UTxO{
		Input: TransactionInput.TransactionInput{
			TransactionId: decodedHash,
			Index:         index,
		},
		Output: output,
	}
	return utxo
}
func (mcc *MaestroChainContext) AddressUtxos(address string, gather bool) []Base.AddressUTXO {
	addressUtxos := make([]Base.AddressUTXO, 0)
	params := utils.NewParameters()
	params.ResolveDatums()
	utxosAtAddressAtApi, err := mcc.client.UtxosAtAddress(address, params)
	if err != nil {
		return addressUtxos
	}

	for _, maestroUtxo := range utxosAtAddressAtApi.Data {
		assets := make([]Base.AddressAmount, 0)
		for _, asset := range maestroUtxo.Assets {
			assets = append(assets, Base.AddressAmount{
				Unit:     asset.Unit,
				Quantity: fmt.Sprint(asset.Amount),
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
			utxosAtAddressAtApi, err = mcc.client.UtxosAtAddress(address, params)
			if err != nil {
				return addressUtxos
			}
			for _, maestroUtxo := range utxosAtAddressAtApi.Data {
				assets := make([]Base.AddressAmount, 0)
				for _, asset := range maestroUtxo.Assets {
					assets = append(assets, Base.AddressAmount{
						Unit:     asset.Unit,
						Quantity: fmt.Sprint(asset.Amount),
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

	return addressUtxos

}
func (mcc *MaestroChainContext) Utxos(address Address.Address) []UTxO.UTxO {
	utxos := make([]UTxO.UTxO, 0)
	params := utils.NewParameters()
	params.WithCbor()
	params.ResolveDatums()
	utxosAtAddressAtApi, err := mcc.client.UtxosAtAddress(address.String(), params)
	if err != nil {
		return utxos
	}

	for _, maestroUtxo := range utxosAtAddressAtApi.Data {
		utxo := UTxO.UTxO{}
		decodedHash, _ := hex.DecodeString(maestroUtxo.TxHash)
		utxo.Input = TransactionInput.TransactionInput{
			TransactionId: decodedHash,
			Index:         int(maestroUtxo.Index),
		}
		output := TransactionOutput.TransactionOutput{}
		decodedCbor, _ := hex.DecodeString(maestroUtxo.TxOutCbor)
		err = cbor.Unmarshal(decodedCbor, &output)
		if err != nil {
			log.Fatal(err)
			return nil
		}
		utxo.Output = output
		utxos = append(utxos, utxo)
	}

	for utxosAtAddressAtApi.NextCursor != "" {
		params.Cursor(utxosAtAddressAtApi.NextCursor)
		utxosAtAddressAtApi, err = mcc.client.UtxosAtAddress(address.String(), params)
		if err != nil {
			return utxos
		}
		for _, maestroUtxo := range utxosAtAddressAtApi.Data {
			utxo := UTxO.UTxO{}
			decodedHash, _ := hex.DecodeString(maestroUtxo.TxHash)
			utxo.Input = TransactionInput.TransactionInput{
				TransactionId: decodedHash,
				Index:         int(maestroUtxo.Index),
			}
			output := TransactionOutput.TransactionOutput{}
			decodedCbor, _ := hex.DecodeString(maestroUtxo.TxOutCbor)
			err = cbor.Unmarshal(decodedCbor, &output)
			if err != nil {
				log.Fatal(err)
				return nil
			}
			utxo.Output = output
			utxos = append(utxos, utxo)
		}
	}

	return utxos
}

func (mcc *MaestroChainContext) SubmitTx(tx Transaction.Transaction) (serialization.TransactionId, error) {
	txBytes, err := tx.Bytes()
	if err != nil {
		return serialization.TransactionId{}, err
	}
	txHex := hex.EncodeToString(txBytes)
	resp, err := mcc.client.SubmitTx(txHex)
	if err != nil {
		log.Fatal(err)
		return serialization.TransactionId{}, err
	}
	decodedResponseHash, _ := hex.DecodeString(resp.Data)
	return serialization.TransactionId{Payload: []byte(decodedResponseHash)}, nil
}

type EvalResult struct {
	Result map[string]map[string]int `json:"EvaluationResult"`
}

type ExecutionResult struct {
	Result EvalResult `json:"result"`
}

func (mcc *MaestroChainContext) EvaluateTx(tx []byte) map[string]Redeemer.ExecutionUnits {
	final_result := make(map[string]Redeemer.ExecutionUnits)
	encodedTx := hex.EncodeToString(tx)
	evaluation, err := mcc.client.EvaluateTx(encodedTx)
	if err != nil {
		return final_result
	}
	for _, eval := range evaluation {
		final_result[eval.RedeemerTag+":"+fmt.Sprint(eval.RedeemerIndex)] = Redeemer.ExecutionUnits{
			Mem:   eval.ExUnits.Mem,
			Steps: eval.ExUnits.Steps,
		}
	}
	return final_result
}

func (mcc *MaestroChainContext) GetContractCbor(scriptHash string) string {
	res, err := mcc.client.ScriptByHash(scriptHash)
	if err != nil {
		return ""
	}
	scCborBytes := res.Data.Bytes
	bytes := []byte{}
	decodedBytes, _ := hex.DecodeString(scCborBytes)
	_ = cbor.Unmarshal(decodedBytes, &bytes)
	return hex.EncodeToString(bytes)

}

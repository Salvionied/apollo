package MaestroChainContext

import (
	"strconv"
	"time"

	"github.com/SundaeSwap-finance/apollo/serialization"
	"github.com/SundaeSwap-finance/apollo/serialization/Address"
	"github.com/SundaeSwap-finance/apollo/serialization/Redeemer"
	"github.com/SundaeSwap-finance/apollo/serialization/Transaction"
	"github.com/SundaeSwap-finance/apollo/serialization/UTxO"
	"github.com/SundaeSwap-finance/apollo/txBuilding/Backend/Base"
)

type MaestroChainContext struct {
	_epoch_info     Base.Epoch
	_epoch          int
	_Network        int
	_genesis_param  Base.GenesisParameters
	_protocol_param Base.ProtocolParameters
}

func NewBlockfrostChainContext(baseUrl string, network int, projectId string) MaestroChainContext {
	mcc := MaestroChainContext{}
	return mcc
}
func (mcc *MaestroChainContext) Init() {
	//TODO
}

func (mcc *MaestroChainContext) GetUtxoFromRef(txHash string, index int) *UTxO.UTxO {
	var utxo *UTxO.UTxO
	return utxo
}

func (mcc *MaestroChainContext) TxOuts(txHash string) []Base.Output {
	//TODO
	return nil

}

func (mcc *MaestroChainContext) LatestBlock() Base.Block {
	latestBlock := Base.Block{}
	//TODO
	return latestBlock
}

func (mcc *MaestroChainContext) LatestEpoch() Base.Epoch {
	epoch := Base.Epoch{}
	//TODO
	return epoch

}
func (mcc *MaestroChainContext) AddressUtxos(address string, gather bool) []Base.AddressUTXO {
	addressUtxos := make([]Base.AddressUTXO, 0)
	//TODO
	return addressUtxos

}

func (mcc *MaestroChainContext) LatestEpochParams() Base.ProtocolParameters {
	protocolParams := Base.ProtocolParameters{}
	//TODO
	return protocolParams
}

func (mcc *MaestroChainContext) GenesisParams() Base.GenesisParameters {
	genesisParams := Base.GenesisParameters{}
	//TODO
	return genesisParams
}
func (mcc *MaestroChainContext) _CheckEpochAndUpdate() bool {
	if mcc._epoch_info.EndTime <= int(time.Now().Unix()) {
		latest_epochs := mcc.LatestEpoch()
		mcc._epoch_info = latest_epochs
		return true
	}
	return false
}

func (mcc *MaestroChainContext) Network() int {
	return mcc._Network
}

func (mcc *MaestroChainContext) Epoch() int {
	if mcc._CheckEpochAndUpdate() {
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
	if mcc._CheckEpochAndUpdate() {
		params := mcc.GenesisParams()
		mcc._genesis_param = params
	}
	return mcc._genesis_param
}

func (mcc *MaestroChainContext) GetProtocolParams() Base.ProtocolParameters {
	if mcc._CheckEpochAndUpdate() {
		latest_params := mcc.LatestEpochParams()
		mcc._protocol_param = latest_params
	}
	return mcc._protocol_param
}

func (mcc *MaestroChainContext) MaxTxFee() int {
	protocol_param := mcc.GetProtocolParams()
	maxTxExSteps, _ := strconv.Atoi(protocol_param.MaxTxExSteps)
	maxTxExMem, _ := strconv.Atoi(protocol_param.MaxTxExMem)
	return Base.Fee(mcc, protocol_param.MaxTxSize, maxTxExSteps, maxTxExMem)
}

func (mcc *MaestroChainContext) Utxos(address Address.Address) []UTxO.UTxO {
	utxos := make([]UTxO.UTxO, 0)
	//TODO
	return utxos
}

func (mcc *MaestroChainContext) SubmitTx(tx Transaction.Transaction) (serialization.TransactionId, error) {
	//TODO
	return serialization.TransactionId{Payload: tx.TransactionBody.Hash()}, nil
}

type EvalResult struct {
	Result map[string]map[string]int `json:"EvaluationResult"`
}

type ExecutionResult struct {
	Result EvalResult `json:"result"`
}

func (mcc *MaestroChainContext) EvaluateTx(tx []byte) (map[string]Redeemer.ExecutionUnits, error) {
	final_result := make(map[string]Redeemer.ExecutionUnits)
	//TODO
	return final_result, nil
}

type BlockfrostContractCbor struct {
	Cbor string `json:"cbor"`
}

func (mcc *MaestroChainContext) GetContractCbor(scriptHash string) string {
	//TODO
	return ""
}

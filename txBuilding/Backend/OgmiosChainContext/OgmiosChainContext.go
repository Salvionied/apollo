package OgmiosChainContext

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

//TOOD

type OgmiosChainContext struct {
	_epoch_info     Base.Epoch
	_epoch          int
	_Network        int
	_genesis_param  Base.GenesisParameters
	_protocol_param Base.ProtocolParameters
}

func NewBlockfrostChainContext(baseUrl string, network int, projectId string) OgmiosChainContext {
	occ := OgmiosChainContext{}
	return occ
}
func (occ *OgmiosChainContext) Init() {
	//TODO
}

func (occ *OgmiosChainContext) GetUtxoFromRef(txHash string, index int) *UTxO.UTxO {
	var utxo *UTxO.UTxO
	return utxo
}

func (occ *OgmiosChainContext) TxOuts(txHash string) []Base.Output {
	//TODO
	return nil

}

func (occ *OgmiosChainContext) LatestBlock() Base.Block {
	latestBlock := Base.Block{}
	//TODO
	return latestBlock
}

func (occ *OgmiosChainContext) LatestEpoch() Base.Epoch {
	epoch := Base.Epoch{}
	//TODO
	return epoch

}
func (occ *OgmiosChainContext) AddressUtxos(address string, gather bool) []Base.AddressUTXO {
	addressUtxos := make([]Base.AddressUTXO, 0)
	//TODO
	return addressUtxos

}

func (occ *OgmiosChainContext) LatestEpochParams() Base.ProtocolParameters {
	protocolParams := Base.ProtocolParameters{}
	//TODO
	return protocolParams
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

func (occ *OgmiosChainContext) Utxos(address Address.Address) []UTxO.UTxO {
	utxos := make([]UTxO.UTxO, 0)
	//TODO
	return utxos
}

func (occ *OgmiosChainContext) SubmitTx(tx Transaction.Transaction) (serialization.TransactionId, error) {
	//TODO
	return serialization.TransactionId{Payload: tx.TransactionBody.Hash()}, nil
}

type EvalResult struct {
	Result map[string]map[string]int `json:"EvaluationResult"`
}

type ExecutionResult struct {
	Result EvalResult `json:"result"`
}

func (occ *OgmiosChainContext) EvaluateTx(tx []byte) map[string]Redeemer.ExecutionUnits {
	final_result := make(map[string]Redeemer.ExecutionUnits)
	//TODO
	return final_result
}

type BlockfrostContractCbor struct {
	Cbor string `json:"cbor"`
}

func (occ *OgmiosChainContext) GetContractCbor(scriptHash string) string {
	//TODO
	return ""
}

package BlockFrostChainContext

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
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
	"github.com/SundaeSwap-finance/apollo/txBuilding/Backend/Cache"

	"github.com/Salvionied/cbor/v2"
)

type BlockFrostChainContext struct {
	client                    *http.Client
	_epoch_info               Base.Epoch
	_epoch                    int
	_Network                  int
	_genesis_param            Base.GenesisParameters
	_protocol_param           Base.ProtocolParameters
	_baseUrl                  string
	_projectId                string
	ctx                       context.Context
	CustomSubmissionEndpoints []string
}

func NewBlockfrostChainContext(baseUrl string, network int, projectId string) BlockFrostChainContext {
	ctx := context.Background()
	file, err := ioutil.ReadFile("config.ini")
	var cse []string
	if err == nil {
		cse = strings.Split(string(file), "\n")
	} else {
		cse = []string{}
	}
	// latest_epochs, err := api.Epoch(ctx)
	// if err != nil {
	// 	log.Fatal(err, "LATEST EPOCH")
	// }

	bfc := BlockFrostChainContext{client: &http.Client{}, _Network: network, _baseUrl: baseUrl, _projectId: projectId, ctx: ctx, CustomSubmissionEndpoints: cse}
	bfc.Init()
	return bfc
}
func (bfc *BlockFrostChainContext) Init() {
	latest_epochs := bfc.LatestEpoch()
	bfc._epoch_info = latest_epochs
	//Init Genesis
	params := bfc.GenesisParams()
	bfc._genesis_param = params
	//init epoch
	latest_params := bfc.LatestEpochParams()
	bfc._protocol_param = latest_params
}

func (bfc *BlockFrostChainContext) GetUtxoFromRef(txHash string, index int) *UTxO.UTxO {
	txOuts := bfc.TxOuts(txHash)
	for _, txOut := range txOuts {
		if txOut.OutputIndex == index {
			return txOut.ToUTxO(txHash)
		}
	}
	return nil
}

func (bfc *BlockFrostChainContext) TxOuts(txHash string) []Base.Output {
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/v0/txs/%s/utxos", bfc._baseUrl, txHash), nil)
	req.Header.Set("project_id", bfc._projectId)
	res, err := bfc.client.Do(req)
	if err != nil {
		log.Fatal(err, "REQUEST PROTOCOL")
	}
	body, err := ioutil.ReadAll(res.Body)
	var response Base.TxUtxos
	err = json.Unmarshal(body, &response)
	if err != nil {
		log.Fatal(err, "UNMARSHAL PROTOCOL")
	}
	return response.Outputs

}

func (bfc *BlockFrostChainContext) LatestBlock() Base.Block {
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/v0/blocks/latest", bfc._baseUrl), nil)
	req.Header.Set("project_id", bfc._projectId)
	res, err := bfc.client.Do(req)
	if err != nil {
		log.Fatal(err, "REQUEST PROTOCOL")
	}
	body, err := ioutil.ReadAll(res.Body)
	var response Base.Block
	err = json.Unmarshal(body, &response)
	if err != nil {
		log.Fatal(err, "UNMARSHAL PROTOCOL")
	}
	return response
}

func (bfc *BlockFrostChainContext) LatestEpoch() Base.Epoch {
	res := Base.Epoch{}
	found := Cache.Get[Base.Epoch]("latest_epoch", &res)
	timest := time.Time{}
	foundTime := Cache.Get[time.Time]("latest_epoch_time", &timest)
	if !found || !foundTime || time.Since(timest) > 5*time.Minute {
		req, _ := http.NewRequest("GET", fmt.Sprintf("%s/v0/epochs/latest", bfc._baseUrl), nil)
		req.Header.Set("project_id", bfc._projectId)
		res, err := bfc.client.Do(req)
		if err != nil {
			log.Fatal(err, "REQUEST PROTOCOL")
		}
		body, err := ioutil.ReadAll(res.Body)
		var response Base.Epoch
		err = json.Unmarshal(body, &response)
		if err != nil {
			log.Fatal(err, "UNMARSHAL PROTOCOL")
		}
		Cache.Set("latest_epoch", response)
		now := time.Now()
		Cache.Set("latest_epoch_time", now)
		return response
	} else {
		return res
	}
}
func (bfc *BlockFrostChainContext) AddressUtxos(address string, gather bool) []Base.AddressUTXO {
	if gather {
		var i = 1
		result := make([]Base.AddressUTXO, 0)
		for {
			req, _ := http.NewRequest("GET", fmt.Sprintf("%s/v0/addresses/%s/utxos?page=%s", bfc._baseUrl, address, fmt.Sprint(i)), nil)
			req.Header.Set("project_id", bfc._projectId)
			res, err := bfc.client.Do(req)
			if err != nil {
				log.Fatal(err, "REQUEST PROTOCOL")
			}
			body, err := ioutil.ReadAll(res.Body)
			var response []Base.AddressUTXO
			err = json.Unmarshal(body, &response)
			if len(response) == 0 {
				break
			}
			if err != nil {
				log.Fatal(err, "UNMARSHAL PROTOCOL")
			}
			result = append(result, response...)
			i++
		}
		return result
	} else {
		req, _ := http.NewRequest("GET", fmt.Sprintf("%s/v0/addresses/%s/utxos", bfc._baseUrl, address), nil)
		req.Header.Set("project_id", bfc._projectId)
		res, err := bfc.client.Do(req)
		if err != nil {
			log.Fatal(err, "REQUEST PROTOCOL")
		}
		body, err := ioutil.ReadAll(res.Body)
		var response []Base.AddressUTXO
		err = json.Unmarshal(body, &response)
		if err != nil {
			log.Fatal(err, "UNMARSHAL PROTOCOL")
		}
		return response
	}
}

func (bfc *BlockFrostChainContext) LatestEpochParams() Base.ProtocolParameters {
	pm := Base.ProtocolParameters{}
	found := Cache.Get[Base.ProtocolParameters]("latest_epoch_params", &pm)
	timest := time.Time{}
	foundTime := Cache.Get[time.Time]("latest_epoch_params_time", &timest)
	if !found || !foundTime || time.Since(timest) > 5*time.Minute {
		req, _ := http.NewRequest("GET", fmt.Sprintf("%s/v0/epochs/latest/parameters", bfc._baseUrl), nil)
		req.Header.Set("project_id", bfc._projectId)
		res, err := bfc.client.Do(req)
		if err != nil {
			log.Fatal(err, "REQUEST PROTOCOL")
		}
		body, err := ioutil.ReadAll(res.Body)
		var response = Base.ProtocolParameters{}
		err = json.Unmarshal(body, &response)
		if err != nil {
			log.Fatal(err, "UNMARSHAL PROTOCOL")
		}
		Cache.Set("latest_epoch_params", response)
		now := time.Now()
		Cache.Set("latest_epoch_params_time", now)
		return response
	} else {
		return pm
	}
}

func (bfc *BlockFrostChainContext) GenesisParams() Base.GenesisParameters {
	gp := Base.GenesisParameters{}
	found := Cache.Get[Base.GenesisParameters]("genesis_params", &gp)
	timestamp := ""
	foundTime := Cache.Get[string]("genesis_params_time", &timestamp)
	timest := time.Time{}
	if timestamp != "" {
		timest, _ = time.Parse(time.RFC3339, timestamp)
	}
	if !found || !foundTime || time.Since(timest) > 5*time.Minute {
		req, _ := http.NewRequest("GET", fmt.Sprintf("%s/v0/Genesis", bfc._baseUrl), nil)
		req.Header.Set("project_id", bfc._projectId)
		res, err := bfc.client.Do(req)
		if err != nil {
			log.Fatal(err, "REQUEST PROTOCOL")
		}
		body, err := ioutil.ReadAll(res.Body)
		var response = Base.GenesisParameters{}
		err = json.Unmarshal(body, &response)
		if err != nil {
			log.Fatal(err, "UNMARSHAL PROTOCOL")
		}
		Cache.Set("genesis_params", response)
		now := time.Now()
		Cache.Set("genesis_params_time", now)
		return response
	} else {

		return gp
	}
}
func (bfc *BlockFrostChainContext) _CheckEpochAndUpdate() bool {
	if bfc._epoch_info.EndTime <= int(time.Now().Unix()) {
		latest_epochs := bfc.LatestEpoch()
		bfc._epoch_info = latest_epochs
		return true
	}
	return false
}

func (bfc *BlockFrostChainContext) Network() int {
	return bfc._Network
}

func (bfc *BlockFrostChainContext) Epoch() int {
	if bfc._CheckEpochAndUpdate() {
		new_epoch := bfc.LatestEpoch()
		bfc._epoch = new_epoch.Epoch
	}
	return bfc._epoch
}

func (bfc *BlockFrostChainContext) LastBlockSlot() int {
	block := bfc.LatestBlock()
	return block.Slot
}

func (bfc *BlockFrostChainContext) GetGenesisParams() Base.GenesisParameters {
	if bfc._CheckEpochAndUpdate() {
		params := bfc.GenesisParams()
		bfc._genesis_param = params
	}
	return bfc._genesis_param
}

func (bfc *BlockFrostChainContext) GetProtocolParams() Base.ProtocolParameters {
	if bfc._CheckEpochAndUpdate() {
		latest_params := bfc.LatestEpochParams()
		bfc._protocol_param = latest_params
	}
	return bfc._protocol_param
}

func (bfc *BlockFrostChainContext) MaxTxFee() int {
	protocol_param := bfc.GetProtocolParams()
	maxTxExSteps, _ := strconv.Atoi(protocol_param.MaxTxExSteps)
	maxTxExMem, _ := strconv.Atoi(protocol_param.MaxTxExMem)
	return Base.Fee(bfc, protocol_param.MaxTxSize, maxTxExSteps, maxTxExMem)
}

func (bfc *BlockFrostChainContext) Utxos(address Address.Address) []UTxO.UTxO {
	results := bfc.AddressUtxos(address.String(), true)
	utxos := make([]UTxO.UTxO, 0)
	for _, result := range results {
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
	return utxos
}
func (bfc *BlockFrostChainContext) SpecialSubmitTx(tx Transaction.Transaction, logger chan string) serialization.TransactionId {
	txBytes, _ := cbor.Marshal(tx)
	if bfc.CustomSubmissionEndpoints != nil {
		logger <- ("Custom Submission Endpoints Found, submitting...")
		for _, endpoint := range bfc.CustomSubmissionEndpoints {
			logger <- fmt.Sprint("TRYING WITH:", endpoint)
			req, _ := http.NewRequest("POST", endpoint, bytes.NewBuffer(txBytes))
			req.Header.Set("project_id", bfc._projectId)
			req.Header.Set("Content-Type", "application/cbor")
			res, err := bfc.client.Do(req)
			if err != nil {
				log.Fatal(err, "REQUEST PROTOCOL")
			}
			body, err := ioutil.ReadAll(res.Body)
			var response any
			err = json.Unmarshal(body, &response)
			if err != nil {
				log.Fatal(err, "UNMARSHAL PROTOCOL")
			}
			logger <- fmt.Sprint("RESPONSE:", response)
		}
	}
	req, _ := http.NewRequest("POST", fmt.Sprintf("%s/v0/tx/submit", bfc._baseUrl), bytes.NewBuffer(txBytes))
	req.Header.Set("project_id", bfc._projectId)
	req.Header.Set("Content-Type", "application/cbor")
	res, err := bfc.client.Do(req)
	if err != nil {
		log.Fatal(err, "REQUEST PROTOCOL")
	}
	body, err := ioutil.ReadAll(res.Body)
	var response any
	err = json.Unmarshal(body, &response)
	if err != nil {
		log.Fatal(err, "UNMARSHAL PROTOCOL")
	}
	return serialization.TransactionId{Payload: tx.TransactionBody.Hash()}
}
func (bfc *BlockFrostChainContext) SubmitTx(tx Transaction.Transaction) (serialization.TransactionId, error) {
	txBytes, _ := cbor.Marshal(tx)
	if bfc.CustomSubmissionEndpoints != nil {
		for _, endpoint := range bfc.CustomSubmissionEndpoints {
			req, _ := http.NewRequest("POST", endpoint, bytes.NewBuffer(txBytes))
			req.Header.Set("project_id", bfc._projectId)
			req.Header.Set("Content-Type", "application/cbor")
			res, err := bfc.client.Do(req)
			if err != nil {
				log.Fatal(err, "REQUEST PROTOCOL")
			}
			body, err := ioutil.ReadAll(res.Body)
			var response any
			err = json.Unmarshal(body, &response)
			if err != nil {
				log.Fatal(err, "UNMARSHAL PROTOCOL")
			}
		}
	}
	req, _ := http.NewRequest("POST", fmt.Sprintf("%s/v0/tx/submit", bfc._baseUrl), bytes.NewBuffer(txBytes))
	req.Header.Set("project_id", bfc._projectId)
	req.Header.Set("Content-Type", "application/cbor")
	res, err := bfc.client.Do(req)
	if err != nil {
		log.Fatal(err, "REQUEST PROTOCOL")
	}
	body, err := ioutil.ReadAll(res.Body)
	var response any
	err = json.Unmarshal(body, &response)
	if err != nil {
		return serialization.TransactionId{}, err
	}
	return serialization.TransactionId{Payload: tx.TransactionBody.Hash()}, nil
}

type EvalResult struct {
	Result map[string]map[string]int `json:"EvaluationResult"`
}

type ExecutionResult struct {
	Result EvalResult `json:"result"`
}

func (bfc *BlockFrostChainContext) EvaluateTx(tx []byte) (map[string]Redeemer.ExecutionUnits, error) {
	encoded := hex.EncodeToString(tx)
	req, _ := http.NewRequest("POST", fmt.Sprintf("%s/v0/utils/txs/evaluate", bfc._baseUrl), strings.NewReader(encoded))
	req.Header.Set("project_id", bfc._projectId)
	req.Header.Set("Content-Type", "application/cbor")
	res, err := bfc.client.Do(req)
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(res.Body)
	var x any
	err = json.Unmarshal(body, &x)
	if err != nil {
		return nil, err
	}
	var response ExecutionResult
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}
	final_result := make(map[string]Redeemer.ExecutionUnits, 0)
	for k, v := range response.Result.Result {

		final_result[k] = Redeemer.ExecutionUnits{Steps: int64(v["steps"]), Mem: int64(v["memory"])}
	}
	return final_result, nil
}

func (bfc *BlockFrostChainContext) EvaluateTxWithAdditionalUtxos(tx []byte, additionalUtxos []UTxO.UTxO) (map[string]Redeemer.ExecutionUnits, error) {
	return nil, fmt.Errorf("EvaluateTxWithAdditionalUtxos: Not implemented")
}

type BlockfrostContractCbor struct {
	Cbor string `json:"cbor"`
}

func (bfc *BlockFrostChainContext) GetContractCbor(scriptHash string) string {
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/v0/scripts/%s/cbor", bfc._baseUrl, scriptHash), nil)
	req.Header.Set("project_id", bfc._projectId)
	res, err := bfc.client.Do(req)
	if err != nil {
		log.Fatal(err, "REQUEST PROTOCOL")
	}
	body, err := ioutil.ReadAll(res.Body)
	var response BlockfrostContractCbor
	err = json.Unmarshal(body, &response)
	if err != nil {
		log.Fatal(err, "UNMARSHAL PROTOCOL")
	}
	return response.Cbor
}

func (bfc *BlockFrostChainContext) CostModelsV1() PlutusData.CostModel {
	log.Fatal("BlockFrostChainContext: CostModelsV1: unimplemented")
	return PlutusData.CostModel(nil)
}

func (bfc *BlockFrostChainContext) CostModelsV2() PlutusData.CostModel {
	log.Fatal("BlockFrostChainContext: CostModelsV2: unimplemented")
	return PlutusData.CostModel(nil)
}

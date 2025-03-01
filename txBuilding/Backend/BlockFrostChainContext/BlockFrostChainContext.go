package BlockFrostChainContext

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
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
	"github.com/Salvionied/apollo/txBuilding/Backend/Cache"

	"github.com/fxamacker/cbor/v2"
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

func NewBlockfrostChainContext(baseUrl string, network int, projectId ...string) (BlockFrostChainContext, error) {
	ctx := context.Background()
	file, err := os.ReadFile("config.ini")
	var cse []string
	if err == nil {
		cse = strings.Split(string(file), "\n")
	} else {
		cse = []string{}
	}

	var _projectId string
	if len(projectId) > 0 {
		_projectId = projectId[0]
	}

	var _baseUrl string
	if strings.Contains(baseUrl, "blockfrost.io") {
		_baseUrl = baseUrl + "/v0"
	} else {
		_baseUrl = baseUrl
	}

	bfc := BlockFrostChainContext{client: &http.Client{}, _Network: network, _baseUrl: _baseUrl, _projectId: _projectId, ctx: ctx, CustomSubmissionEndpoints: cse}
	err = bfc.Init()
	if err != nil {
		return bfc, err
	}
	return bfc, nil
}
func (bfc *BlockFrostChainContext) Init() error {
	latest_epochs, err := bfc.LatestEpoch()
	if err != nil {
		return err
	}
	bfc._epoch_info = latest_epochs
	//Init Genesis
	params, err := bfc.GenesisParams()
	if err != nil {
		return err
	}
	bfc._genesis_param = params
	//init epoch
	latest_params, err := bfc.LatestEpochParams()
	if err != nil {
		return err
	}
	bfc._protocol_param = latest_params
	return nil
}

func (bfc *BlockFrostChainContext) GetUtxoFromRef(txHash string, index int) (*UTxO.UTxO, error) {
	txOuts, err := bfc.TxOuts(txHash)
	if err != nil {
		return nil, err
	}
	for _, txOut := range txOuts {
		if txOut.OutputIndex == index {
			return txOut.ToUTxO(txHash), nil
		}
	}
	return nil, errors.New("UTXO doesn't exist")
}

func (bfc *BlockFrostChainContext) TxOuts(txHash string) ([]Base.Output, error) {
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/txs/%s/utxos", bfc._baseUrl, txHash), nil)
	if bfc._projectId != "" {
		req.Header.Set("project_id", bfc._projectId)
	}
	res, err := bfc.client.Do(req)
	if err != nil {
		return nil, err
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	var response Base.TxUtxos
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}
	return response.Outputs, nil

}

func (bfc *BlockFrostChainContext) LatestBlock() (Base.Block, error) {
	bb := Base.Block{}
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/blocks/latest", bfc._baseUrl), nil)
	if bfc._projectId != "" {
		req.Header.Set("project_id", bfc._projectId)
	}
	res, err := bfc.client.Do(req)
	if err != nil {
		return bb, err
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return bb, err
	}
	var response Base.Block
	err = json.Unmarshal(body, &response)
	if err != nil {
		return bb, err
	}
	return response, nil
}

func (bfc *BlockFrostChainContext) LatestEpoch() (Base.Epoch, error) {
	resultingEpoch := Base.Epoch{}
	found := Cache.Get[Base.Epoch]("latest_epoch", &resultingEpoch)
	timest := time.Time{}
	foundTime := Cache.Get[time.Time]("latest_epoch_time", &timest)
	if !found || !foundTime || time.Since(timest) > 5*time.Minute {
		req, _ := http.NewRequest("GET", fmt.Sprintf("%s/epochs/latest", bfc._baseUrl), nil)
		if bfc._projectId != "" {
			req.Header.Set("project_id", bfc._projectId)
		}
		res, err := bfc.client.Do(req)
		if err != nil {
			return resultingEpoch, err
		}
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return resultingEpoch, err
		}
		var response Base.Epoch
		err = json.Unmarshal(body, &response)
		if err != nil {
			return resultingEpoch, err
		}
		Cache.Set("latest_epoch", response)
		now := time.Now()
		Cache.Set("latest_epoch_time", now)
		return response, nil
	} else {
		return resultingEpoch, nil
	}
}
func (bfc *BlockFrostChainContext) AddressUtxos(address string, gather bool) ([]Base.AddressUTXO, error) {
	if gather {
		var i = 1
		result := make([]Base.AddressUTXO, 0)
		for {
			req, _ := http.NewRequest("GET", fmt.Sprintf("%s/addresses/%s/utxos?page=%s", bfc._baseUrl, address, fmt.Sprint(i)), nil)
			if bfc._projectId != "" {
				req.Header.Set("project_id", bfc._projectId)
			}
			res, err := bfc.client.Do(req)
			if err != nil {
				return nil, err
			}
			body, err := io.ReadAll(res.Body)
			if err != nil {
				return nil, err
			}
			var response []Base.AddressUTXO
			err = json.Unmarshal(body, &response)
			if len(response) == 0 {
				break
			}
			if err != nil {
				return nil, err
			}
			result = append(result, response...)
			i++
		}
		return result, nil
	} else {
		req, _ := http.NewRequestWithContext(bfc.ctx, "GET", fmt.Sprintf("%s/addresses/%s/utxos", bfc._baseUrl, address), nil)
		if bfc._projectId != "" {
			req.Header.Set("project_id", bfc._projectId)
		}
		res, err := bfc.client.Do(req)
		if err != nil {
			return nil, err
		}
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, err
		}
		var response []Base.AddressUTXO
		err = json.Unmarshal(body, &response)
		if err != nil {
			return nil, err
		}
		return response, nil
	}
}

func (bfc *BlockFrostChainContext) LatestEpochParams() (Base.ProtocolParameters, error) {
	pm := Base.ProtocolParameters{}
	found := Cache.Get[Base.ProtocolParameters]("latest_epoch_params", &pm)
	timest := time.Time{}
	foundTime := Cache.Get[time.Time]("latest_epoch_params_time", &timest)
	if !found || !foundTime || time.Since(timest) > 5*time.Minute {
		req, _ := http.NewRequestWithContext(bfc.ctx, "GET", fmt.Sprintf("%s/epochs/latest/parameters", bfc._baseUrl), nil)
		if bfc._projectId != "" {
			req.Header.Set("project_id", bfc._projectId)
		}
		res, err := bfc.client.Do(req)
		if err != nil {
			return pm, err
		}
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return pm, err
		}
		var response = Base.BlockfrostProtocolParams{}
		err = json.Unmarshal(body, &response)
		if err != nil {
			return pm, err
		}
		Cache.Set("latest_epoch_params", response)
		now := time.Now()
		Cache.Set("latest_epoch_params_time", now)
		return response.ToBaseParams(), nil
	} else {
		return pm, nil
	}
}

func (bfc *BlockFrostChainContext) GenesisParams() (Base.GenesisParameters, error) {
	gp := Base.GenesisParameters{}
	found := Cache.Get[Base.GenesisParameters]("genesis_params", &gp)
	timestamp := ""
	foundTime := Cache.Get[string]("genesis_params_time", &timestamp)
	timest := time.Time{}
	if timestamp != "" {
		timest, _ = time.Parse(time.RFC3339, timestamp)
	}
	if !found || !foundTime || time.Since(timest) > 5*time.Minute {
		req, _ := http.NewRequest("GET", fmt.Sprintf("%s/Genesis", bfc._baseUrl), nil)
		if bfc._projectId != "" {
			req.Header.Set("project_id", bfc._projectId)
		}
		res, err := bfc.client.Do(req)
		if err != nil {
			return gp, err
		}
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return gp, err
		}
		var response = Base.GenesisParameters{}
		err = json.Unmarshal(body, &response)
		if err != nil {
			return gp, err
		}
		Cache.Set("genesis_params", response)
		now := time.Now()
		Cache.Set("genesis_params_time", now)
		return response, nil
	} else {

		return gp, nil
	}
}
func (bfc *BlockFrostChainContext) _CheckEpochAndUpdate() error {
	if bfc._epoch_info.EndTime <= int(time.Now().Unix()) {
		latest_epochs, err := bfc.LatestEpoch()
		if err != nil {
			return err
		}
		bfc._epoch_info = latest_epochs
		bfc._epoch = latest_epochs.Epoch
		latestParams, err := bfc.GetProtocolParams()
		if err != nil {
			return err
		}
		bfc._protocol_param = latestParams
		latestGensParams, err := bfc.GetGenesisParams()
		if err != nil {
			return err
		}
		bfc._genesis_param = latestGensParams

	}
	return nil
}

func (bfc *BlockFrostChainContext) Network() int {
	return bfc._Network
}

func (bfc *BlockFrostChainContext) Epoch() (int, error) {
	err := bfc._CheckEpochAndUpdate()
	if err != nil {
		return 0, err
	}
	return bfc._epoch, nil
}

func (bfc *BlockFrostChainContext) LastBlockSlot() (int, error) {
	block, err := bfc.LatestBlock()
	if err != nil {
		return 0, err
	}
	return block.Slot, nil
}

func (bfc *BlockFrostChainContext) GetGenesisParams() (Base.GenesisParameters, error) {
	gp := Base.GenesisParameters{}
	err := bfc._CheckEpochAndUpdate()
	if err != nil {
		return gp, err
	}
	return bfc._genesis_param, nil
}

func (bfc *BlockFrostChainContext) GetProtocolParams() (Base.ProtocolParameters, error) {
	pps := Base.ProtocolParameters{}
	err := bfc._CheckEpochAndUpdate()
	if err != nil {
		return pps, err
	}
	return bfc._protocol_param, nil
}

func (bfc *BlockFrostChainContext) MaxTxFee() (int, error) {
	protocol_param, err := bfc.GetProtocolParams()
	if err != nil {
		return 0, err
	}
	maxTxExSteps, _ := strconv.Atoi(protocol_param.MaxTxExSteps)
	maxTxExMem, _ := strconv.Atoi(protocol_param.MaxTxExMem)
	return Base.Fee(bfc, protocol_param.MaxTxSize, maxTxExSteps, maxTxExMem)
}

func (bfc *BlockFrostChainContext) Utxos(address Address.Address) ([]UTxO.UTxO, error) {
	results, err := bfc.AddressUtxos(address.String(), true)
	if err != nil {
		return nil, err
	}
	utxos := make([]UTxO.UTxO, 0)
	for _, result := range results {
		decodedTxId, _ := hex.DecodeString(result.TxHash)
		tx_in := TransactionInput.TransactionInput{TransactionId: decodedTxId, Index: result.OutputIndex}
		amount := result.Amount
		lovelace_amount := 0
		multi_assets := MultiAsset.MultiAsset[int64]{}
		for _, item := range amount {
			if item.Unit == "lovelace" {
				amount, _ := strconv.Atoi(item.Quantity)

				lovelace_amount += amount
			} else {
				asset_quantity, _ := strconv.ParseInt(item.Quantity, 10, 64)
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
			decoded, _ := hex.DecodeString(result.InlineDatum)
			var x PlutusData.PlutusData
			err := cbor.Unmarshal(decoded, &x)
			if err != nil {
				continue
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
func (bfc *BlockFrostChainContext) SpecialSubmitTx(tx Transaction.Transaction, logger chan string) (serialization.TransactionId, error) {
	resId := serialization.TransactionId{}
	txBytes, _ := cbor.Marshal(tx)
	if bfc.CustomSubmissionEndpoints != nil {
		logger <- ("Custom Submission Endpoints Found, submitting...")
		for _, endpoint := range bfc.CustomSubmissionEndpoints {
			logger <- fmt.Sprint("TRYING WITH:", endpoint)
			req, _ := http.NewRequest("POST", endpoint, bytes.NewBuffer(txBytes))
			if bfc._projectId != "" {
				req.Header.Set("project_id", bfc._projectId)
			}
			req.Header.Set("Content-Type", "application/cbor")
			res, err := bfc.client.Do(req)
			if err != nil {
				return resId, err
			}
			body, err := io.ReadAll(res.Body)
			if err != nil {
				return resId, err
			}
			var response any
			err = json.Unmarshal(body, &response)
			if err != nil {
				return resId, err
			}
			logger <- fmt.Sprint("RESPONSE:", response)
		}
	}
	req, _ := http.NewRequestWithContext(bfc.ctx, "POST", fmt.Sprintf("%s/tx/submit", bfc._baseUrl), bytes.NewBuffer(txBytes))
	if bfc._projectId != "" {
		req.Header.Set("project_id", bfc._projectId)
	}
	req.Header.Set("Content-Type", "application/cbor")
	res, err := bfc.client.Do(req)
	if err != nil {
		return resId, err
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return resId, err
	}
	var response any
	err = json.Unmarshal(body, &response)
	if err != nil {
		return resId, err
	}
	hash, _ := tx.TransactionBody.Hash()
	return serialization.TransactionId{Payload: hash}, nil
}
func (bfc *BlockFrostChainContext) SubmitTx(tx Transaction.Transaction) (serialization.TransactionId, error) {
	resId := serialization.TransactionId{}
	txBytes, _ := cbor.Marshal(tx)
	if bfc.CustomSubmissionEndpoints != nil {
		for _, endpoint := range bfc.CustomSubmissionEndpoints {
			req, _ := http.NewRequest("POST", endpoint, bytes.NewBuffer(txBytes))
			if bfc._projectId != "" {
				req.Header.Set("project_id", bfc._projectId)
			}
			req.Header.Set("Content-Type", "application/cbor")
			res, err := bfc.client.Do(req)
			if err != nil {
				return resId, err
			}
			body, err := io.ReadAll(res.Body)
			if err != nil {
				return resId, err
			}
			var response any
			err = json.Unmarshal(body, &response)
			if err != nil {
				return resId, err
			}
		}
	}
	req, _ := http.NewRequest("POST", fmt.Sprintf("%s/tx/submit", bfc._baseUrl), bytes.NewBuffer(txBytes))
	if bfc._projectId != "" {
		req.Header.Set("project_id", bfc._projectId)
	}
	req.Header.Set("Content-Type", "application/cbor")
	res, err := bfc.client.Do(req)
	if err != nil {
		return serialization.TransactionId{}, err
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return serialization.TransactionId{}, err
	}
	var response any
	err = json.Unmarshal(body, &response)
	if err != nil {
		return serialization.TransactionId{}, err
	}
	if res.Status != "200 OK" {
		return serialization.TransactionId{}, fmt.Errorf("error submitting tx: %v", response)
	}
	hash, err := tx.TransactionBody.Hash()
	if err != nil {
		return serialization.TransactionId{}, err
	}
	return serialization.TransactionId{Payload: hash}, nil
}

type EvalResult struct {
	Result map[string]map[string]int `json:"EvaluationResult"`
}

type ExecutionResult struct {
	Result EvalResult `json:"result"`
}

func (bfc *BlockFrostChainContext) EvaluateTx(tx []byte) (map[string]Redeemer.ExecutionUnits, error) {
	encoded := hex.EncodeToString(tx)
	req, _ := http.NewRequest("POST", fmt.Sprintf("%s/utils/txs/evaluate", bfc._baseUrl), strings.NewReader(encoded))
	if bfc._projectId != "" {
		req.Header.Set("project_id", bfc._projectId)
	}
	req.Header.Set("Content-Type", "application/cbor")
	res, err := bfc.client.Do(req)
	if err != nil {
		return nil, err
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
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

type BlockfrostContractCbor struct {
	Cbor string `json:"cbor"`
}

func (bfc *BlockFrostChainContext) GetContractCbor(scriptHash string) (string, error) {
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/scripts/%s/cbor", bfc._baseUrl, scriptHash), nil)
	if bfc._projectId != "" {
		req.Header.Set("project_id", bfc._projectId)
	}
	res, err := bfc.client.Do(req)
	if err != nil {
		return "", err
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	var response BlockfrostContractCbor
	err = json.Unmarshal(body, &response)
	if err != nil {
		return "", err
	}
	return response.Cbor, nil
}

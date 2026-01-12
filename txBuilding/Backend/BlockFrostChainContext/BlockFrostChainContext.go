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

	"github.com/Salvionied/apollo/v2/serialization"
	"github.com/Salvionied/apollo/v2/serialization/Address"
	"github.com/Salvionied/apollo/v2/serialization/Amount"
	"github.com/Salvionied/apollo/v2/serialization/Redeemer"
	"github.com/Salvionied/apollo/v2/serialization/Transaction"
	"github.com/Salvionied/apollo/v2/serialization/TransactionInput"
	"github.com/Salvionied/apollo/v2/serialization/UTxO"
	"github.com/Salvionied/apollo/v2/serialization/Value"
	"github.com/Salvionied/apollo/v2/txBuilding/Backend/Base"
	"github.com/Salvionied/apollo/v2/txBuilding/Backend/Cache"
	"github.com/Salvionied/apollo/v2/txBuilding/Utils"

	"github.com/blinklabs-io/gouroboros/cbor"
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

func NewBlockfrostChainContext(
	baseUrl string,
	network int,
	projectId ...string,
) (BlockFrostChainContext, error) {
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

	bfc := BlockFrostChainContext{
		client:                    &http.Client{},
		_Network:                  network,
		_baseUrl:                  _baseUrl,
		_projectId:                _projectId,
		ctx:                       ctx,
		CustomSubmissionEndpoints: cse,
	}
	err = bfc.Init()
	if err != nil {
		return bfc, err
	}
	return bfc, nil
}

// doRequest is a helper method for making HTTP requests to the BlockFrost API.
// It handles setting the project_id header and reading the response body.
func (bfc *BlockFrostChainContext) doRequest(
	method, path string,
	body io.Reader,
) ([]byte, error) {
	req, err := http.NewRequestWithContext(
		bfc.ctx,
		method,
		bfc._baseUrl+path,
		body,
	)
	if err != nil {
		return nil, err
	}
	if bfc._projectId != "" {
		req.Header.Set("project_id", bfc._projectId)
	}
	res, err := bfc.client.Do(req)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, errors.New("nil response from BlockFrost API")
	}
	defer res.Body.Close()
	return io.ReadAll(res.Body)
}

// doRequestWithHeaders is a helper for HTTP requests that need custom headers.
// It is useful for POST requests with Content-Type headers.
func (bfc *BlockFrostChainContext) doRequestWithHeaders(
	method, url string,
	body io.Reader,
	headers map[string]string,
) (*http.Response, []byte, error) {
	req, err := http.NewRequestWithContext(bfc.ctx, method, url, body)
	if err != nil {
		return nil, nil, err
	}
	if bfc._projectId != "" {
		req.Header.Set("project_id", bfc._projectId)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	res, err := bfc.client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	if res == nil {
		return nil, nil, errors.New("nil response from BlockFrost API")
	}
	defer res.Body.Close()
	respBody, err := io.ReadAll(res.Body)
	if err != nil {
		return res, nil, err
	}
	return res, respBody, nil
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

func (bfc *BlockFrostChainContext) GetUtxoFromRef(
	txHash string,
	index int,
) (*UTxO.UTxO, error) {
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

func (bfc *BlockFrostChainContext) TxOuts(
	txHash string,
) ([]Base.Output, error) {
	body, err := bfc.doRequest(
		"GET",
		fmt.Sprintf("/txs/%s/utxos", txHash),
		nil,
	)
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
	body, err := bfc.doRequest("GET", "/blocks/latest", nil)
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
		body, err := bfc.doRequest("GET", "/epochs/latest", nil)
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

func (bfc *BlockFrostChainContext) AddressUtxos(
	address string,
	gather bool,
) ([]Base.AddressUTXO, error) {
	if gather {
		var i = 1
		result := make([]Base.AddressUTXO, 0)
		for {
			body, err := bfc.doRequest(
				"GET",
				fmt.Sprintf(
					"/addresses/%s/utxos?page=%s",
					address,
					strconv.Itoa(i),
				),
				nil,
			)
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
		body, err := bfc.doRequest(
			"GET",
			fmt.Sprintf("/addresses/%s/utxos", address),
			nil,
		)
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
		body, err := bfc.doRequest("GET", "/epochs/latest/parameters", nil)
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
		body, err := bfc.doRequest("GET", "/Genesis", nil)
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
	maxTxExSteps, err := strconv.Atoi(protocol_param.MaxTxExSteps)
	if err != nil {
		return 0, fmt.Errorf("failed to parse MaxTxExSteps: %w", err)
	}
	maxTxExMem, err := strconv.Atoi(protocol_param.MaxTxExMem)
	if err != nil {
		return 0, fmt.Errorf("failed to parse MaxTxExMem: %w", err)
	}
	return Base.Fee(bfc, protocol_param.MaxTxSize, maxTxExSteps, maxTxExMem)
}

func (bfc *BlockFrostChainContext) Utxos(
	address Address.Address,
) ([]UTxO.UTxO, error) {
	results, err := bfc.AddressUtxos(address.String(), true)
	if err != nil {
		return nil, err
	}
	utxos := make([]UTxO.UTxO, 0)
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

func (bfc *BlockFrostChainContext) SpecialSubmitTx(
	tx Transaction.Transaction,
	logger chan string,
) (serialization.TransactionId, error) {
	resId := serialization.TransactionId{}
	txBytes, _ := cbor.Encode(tx)
	headers := map[string]string{"Content-Type": "application/cbor"}
	if bfc.CustomSubmissionEndpoints != nil {
		logger <- ("Custom Submission Endpoints Found, submitting...")
		for _, endpoint := range bfc.CustomSubmissionEndpoints {
			logger <- fmt.Sprint("TRYING WITH:", endpoint)
			_, body, err := bfc.doRequestWithHeaders(
				"POST",
				endpoint,
				bytes.NewBuffer(txBytes),
				headers,
			)
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
	_, body, err := bfc.doRequestWithHeaders(
		"POST",
		bfc._baseUrl+"/tx/submit",
		bytes.NewBuffer(txBytes),
		headers,
	)
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

func (bfc *BlockFrostChainContext) SubmitTx(
	tx Transaction.Transaction,
) (serialization.TransactionId, error) {
	resId := serialization.TransactionId{}
	txBytes, _ := cbor.Encode(tx)
	headers := map[string]string{"Content-Type": "application/cbor"}
	if bfc.CustomSubmissionEndpoints != nil {
		for _, endpoint := range bfc.CustomSubmissionEndpoints {
			_, body, err := bfc.doRequestWithHeaders(
				"POST",
				endpoint,
				bytes.NewBuffer(txBytes),
				headers,
			)
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
	res, body, err := bfc.doRequestWithHeaders(
		"POST",
		bfc._baseUrl+"/tx/submit",
		bytes.NewBuffer(txBytes),
		headers,
	)
	if err != nil {
		return serialization.TransactionId{}, err
	}
	var response any
	err = json.Unmarshal(body, &response)
	if err != nil {
		return serialization.TransactionId{}, err
	}
	if res.Status != "200 OK" {
		return serialization.TransactionId{}, fmt.Errorf(
			"error submitting tx: %v",
			response,
		)
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

func (bfc *BlockFrostChainContext) EvaluateTx(
	tx []byte,
) (map[string]Redeemer.ExecutionUnits, error) {
	encoded := hex.EncodeToString(tx)
	headers := map[string]string{"Content-Type": "application/cbor"}
	_, body, err := bfc.doRequestWithHeaders(
		"POST",
		bfc._baseUrl+"/utils/txs/evaluate",
		strings.NewReader(encoded),
		headers,
	)
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
		final_result[k] = Redeemer.ExecutionUnits{
			Mem:   uint64(v["memory"]),
			Steps: uint64(v["steps"]),
		}
	}
	return final_result, nil
}

type BlockfrostContractCbor struct {
	Cbor string `json:"cbor"`
}

func (bfc *BlockFrostChainContext) GetContractCbor(
	scriptHash string,
) (string, error) {
	body, err := bfc.doRequest(
		"GET",
		fmt.Sprintf("/scripts/%s/cbor", scriptHash),
		nil,
	)
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

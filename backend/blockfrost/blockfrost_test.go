package blockfrost

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"math"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/babbage"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/mary"
	"github.com/blinklabs-io/gouroboros/ledger/shelley"
)

func testAddress(t *testing.T) common.Address {
	t.Helper()
	var raw [57]byte
	raw[0] = 0x00
	raw[1] = 0xAA
	raw[29] = 0xBB
	addr, err := common.NewAddressFromBytes(raw[:])
	if err != nil {
		t.Fatal(err)
	}
	return addr
}

func TestHydrateUtxoResolvesInlineDatumAndReferenceScript(t *testing.T) {
	script := common.PlutusV2Script([]byte{0x01, 0x02})
	scriptHashHex := hex.EncodeToString(script.Hash().Bytes())
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v0/scripts/"+scriptHashHex+"/cbor" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{
			"cbor": hex.EncodeToString(script),
		})
	}))
	defer server.Close()

	addr := testAddress(t)
	ctx := NewBlockFrostChainContext(server.URL, 0, "")
	// BlockFrost returns inline_datum as a CBOR-encoded hex string.
	// 0x182a is the CBOR encoding of the integer 42.
	wantDatumCbor := []byte{0x18, 0x2a}
	raw := bfAddressUTxO{
		TxHash:              "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		OutputIndex:         0,
		Address:             addr.String(),
		Amount:              []bfAddressAmount{{Unit: "lovelace", Quantity: "1000000"}},
		InlineDatum:         json.RawMessage(`"182a"`),
		ReferenceScriptHash: scriptHashHex,
	}

	utxo, err := ctx.hydrateUtxo(raw, addr)
	if err != nil {
		t.Fatal(err)
	}
	output, ok := utxo.Output.(*babbage.BabbageTransactionOutput)
	if !ok {
		t.Fatalf("unexpected output type %T", utxo.Output)
	}
	if output.Datum() == nil {
		t.Fatal("expected inline datum to be populated")
	}
	// The exact on-chain CBOR bytes must be preserved so the datum hash is unchanged.
	if got := output.Datum().Cbor(); !bytes.Equal(got, wantDatumCbor) {
		t.Fatalf("inline datum CBOR = %x, want %x", got, wantDatumCbor)
	}
	scriptRef := output.ScriptRef()
	if scriptRef == nil {
		t.Fatal("expected reference script to be populated")
	}
	if _, ok := scriptRef.(common.PlutusV2Script); !ok {
		t.Fatalf("expected PlutusV2 reference script, got %T", scriptRef)
	}
}

func TestUtxoByRefFillsMissingTxHashOnTxUtxosOutputs(t *testing.T) {
	addr := testAddress(t)
	const txHashHex = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v0/txs/"+txHashHex+"/utxos" {
			http.NotFound(w, r)
			return
		}
		// Mirror Blockfrost /txs/{hash}/utxos: top-level hash only; outputs omit tx_hash.
		_ = json.NewEncoder(w).Encode(map[string]any{
			"hash":   txHashHex,
			"inputs": []any{},
			"outputs": []map[string]any{
				{
					"address":      addr.String(),
					"amount":       []map[string]string{{"unit": "lovelace", "quantity": "2000000"}},
					"output_index": 1,
				},
			},
		})
	}))
	defer server.Close()

	ctx := NewBlockFrostChainContext(server.URL, 0, "test-project")
	hashBytes, err := hex.DecodeString(txHashHex)
	if err != nil {
		t.Fatal(err)
	}
	var txHash common.Blake2b256
	copy(txHash[:], hashBytes)

	utxo, err := ctx.UtxoByRef(txHash, 1)
	if err != nil {
		t.Fatalf("UtxoByRef: %v", err)
	}
	if utxo == nil {
		t.Fatal("expected utxo")
	}
	if got := hex.EncodeToString(utxo.Id.Id().Bytes()); got != txHashHex {
		t.Fatalf("utxo tx id = %s, want %s", got, txHashHex)
	}
	if utxo.Id.Index() != 1 {
		t.Fatalf("utxo index = %d, want 1", utxo.Id.Index())
	}
	out, ok := utxo.Output.(*babbage.BabbageTransactionOutput)
	if !ok {
		t.Fatalf("unexpected output type %T", utxo.Output)
	}
	if out.OutputAmount.Amount != 2_000_000 {
		t.Fatalf("lovelace = %d, want 2000000", out.OutputAmount.Amount)
	}
}

func TestEvaluateTxRejectsRedeemerIndexOverflow(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v0/utils/txs/evaluate" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"result": map[string]any{
				"EvaluationResult": map[string]any{
					"spend:4294967296": map[string]uint64{
						"memory": 1,
						"steps":  1,
					},
				},
				"EvaluationFailure": nil,
			},
		})
	}))
	defer server.Close()

	ctx := NewBlockFrostChainContext(server.URL, 0, "")
	_, err := ctx.EvaluateTx([]byte{0x84}, nil)
	if err == nil {
		t.Fatal("expected redeemer index overflow error")
	}
}

func TestEvaluateTxSendsHexEncodedBody(t *testing.T) {
	txCbor := []byte{0x84, 0xa3, 0x00}
	var gotBody string
	var gotContentType string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v0/utils/txs/evaluate" {
			http.NotFound(w, r)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("failed to read request body: %v", err)
		}
		gotBody = string(body)
		gotContentType = r.Header.Get("Content-Type")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","method":"evaluateTransaction","result":[` +
			`{"validator":{"purpose":"spend","index":0},"budget":{"memory":1700,"cpu":476468}}` +
			`],"id":null}`))
	}))
	defer server.Close()

	ctx := NewBlockFrostChainContext(server.URL, 0, "")
	result, err := ctx.EvaluateTx(txCbor, nil)
	if err != nil {
		t.Fatal(err)
	}
	if gotBody != hex.EncodeToString(txCbor) {
		t.Fatalf("request body = %q, want hex-encoded tx CBOR %q", gotBody, hex.EncodeToString(txCbor))
	}
	if gotContentType != "application/cbor" {
		t.Fatalf("Content-Type = %q, want application/cbor", gotContentType)
	}
	key := common.RedeemerKey{Tag: common.RedeemerTagSpend, Index: 0}
	if eu, ok := result[key]; !ok || eu.Memory != 1700 || eu.Steps != 476468 {
		t.Fatalf("unexpected result %v", result)
	}
}

func TestParseEvaluateTxResponseOgmiosV5(t *testing.T) {
	data := []byte(`{"type":"jsonwsp/response","result":{"EvaluationResult":{` +
		`"spend:0":{"memory":1700,"steps":476468},` +
		`"mint:1":{"memory":250,"steps":1000}` +
		`}}}`)
	result, err := parseEvaluateTxResponse(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result))
	}
	spendKey := common.RedeemerKey{Tag: common.RedeemerTagSpend, Index: 0}
	if eu := result[spendKey]; eu.Memory != 1700 || eu.Steps != 476468 {
		t.Fatalf("unexpected spend budget %+v", eu)
	}
	mintKey := common.RedeemerKey{Tag: common.RedeemerTagMint, Index: 1}
	if eu := result[mintKey]; eu.Memory != 250 || eu.Steps != 1000 {
		t.Fatalf("unexpected mint budget %+v", eu)
	}
}

func TestParseEvaluateTxResponseOgmiosV6(t *testing.T) {
	data := []byte(`{"jsonrpc":"2.0","method":"evaluateTransaction","result":[` +
		`{"validator":{"purpose":"spend","index":0},"budget":{"memory":1700,"cpu":476468}},` +
		`{"validator":{"purpose":"withdraw","index":2},"budget":{"memory":250,"cpu":1000}}` +
		`],"id":null}`)
	result, err := parseEvaluateTxResponse(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result))
	}
	spendKey := common.RedeemerKey{Tag: common.RedeemerTagSpend, Index: 0}
	if eu := result[spendKey]; eu.Memory != 1700 || eu.Steps != 476468 {
		t.Fatalf("unexpected spend budget %+v", eu)
	}
	withdrawKey := common.RedeemerKey{Tag: common.RedeemerTagReward, Index: 2}
	if eu := result[withdrawKey]; eu.Memory != 250 || eu.Steps != 1000 {
		t.Fatalf("unexpected withdraw budget %+v", eu)
	}
}

func TestParseEvaluateTxResponseOgmiosV6Error(t *testing.T) {
	data := []byte(`{"jsonrpc":"2.0","method":"evaluateTransaction",` +
		`"error":{"code":3010,"message":"Some scripts of the transaction terminated with error(s).",` +
		`"data":[{"validator":{"purpose":"spend","index":0},"error":{"code":3011,"message":"boom"}}]},"id":null}`)
	_, err := parseEvaluateTxResponse(data)
	if err == nil {
		t.Fatal("expected evaluation error")
	}
	if !strings.Contains(err.Error(), "3010") {
		t.Fatalf("error should include Ogmios error code, got: %v", err)
	}
}

func TestParseEvaluateTxResponseOgmiosV6PerValidatorError(t *testing.T) {
	data := []byte(`{"jsonrpc":"2.0","result":[` +
		`{"validator":{"purpose":"spend","index":0},"error":{"code":3011,"message":"boom"}}` +
		`],"id":null}`)
	if _, err := parseEvaluateTxResponse(data); err == nil {
		t.Fatal("expected per-validator evaluation error")
	}
}

func TestParseEvaluateTxResponseV5Failure(t *testing.T) {
	data := []byte(`{"result":{"EvaluationFailure":{"ScriptFailures":{"spend:0":["validator failed"]}}}}`)
	if _, err := parseEvaluateTxResponse(data); err == nil {
		t.Fatal("expected evaluation failure error")
	}
}

func TestParseEvaluateTxResponseRejectsUnknownShape(t *testing.T) {
	for _, data := range []string{
		`{"foo":"bar"}`,
		`{}`,
		`{"result":{"SomethingElse":{}}}`,
		`{"result":"oops"}`,
	} {
		t.Run(data, func(t *testing.T) {
			if _, err := parseEvaluateTxResponse([]byte(data)); err == nil {
				t.Fatalf("expected error for unrecognized response %s", data)
			}
		})
	}
}

func TestParseEvaluateTxResponseRejectsEmptyResults(t *testing.T) {
	for _, data := range []string{
		`{"result":[]}`,
		`{"result":{"EvaluationResult":{}}}`,
	} {
		t.Run(data, func(t *testing.T) {
			if _, err := parseEvaluateTxResponse([]byte(data)); err == nil {
				t.Fatalf("expected error for empty evaluation results %s", data)
			}
		})
	}
}

func TestParseEvaluateTxResponseV6RejectsIndexOverflow(t *testing.T) {
	data := []byte(`{"result":[{"validator":{"purpose":"spend","index":4294967296},"budget":{"memory":1,"cpu":1}}]}`)
	if _, err := parseEvaluateTxResponse(data); err == nil {
		t.Fatal("expected redeemer index overflow error")
	}
}

func TestAddressUTxOToUtxoRejectsInvalidAssetUnit(t *testing.T) {
	addr := testAddress(t)
	raw := bfAddressUTxO{
		TxHash:      "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		OutputIndex: 0,
		Address:     addr.String(),
		Amount: []bfAddressAmount{
			{Unit: "lovelace", Quantity: "1000000"},
			{Unit: "abcd", Quantity: "1"},
		},
	}
	if _, err := raw.toUtxo(addr); err == nil {
		t.Fatal("expected invalid asset unit error")
	}
}

func TestAddressUTxOToUtxoRejectsNegativeAssetQuantity(t *testing.T) {
	addr := testAddress(t)
	raw := bfAddressUTxO{
		TxHash:      "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		OutputIndex: 0,
		Address:     addr.String(),
		Amount: []bfAddressAmount{
			{Unit: "lovelace", Quantity: "1000000"},
			{Unit: "00000000000000000000000000000000000000000000000000000001", Quantity: "-1"},
		},
	}
	if _, err := raw.toUtxo(addr); err == nil {
		t.Fatal("expected negative asset quantity error")
	}
}

func TestAddressUTxOToUtxoRejectsOutputIndexOverflow(t *testing.T) {
	addr := testAddress(t)
	raw := bfAddressUTxO{
		TxHash:      "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		OutputIndex: int(math.MaxUint32) + 1,
		Address:     addr.String(),
		Amount:      []bfAddressAmount{{Unit: "lovelace", Quantity: "1000000"}},
	}
	if _, err := raw.toUtxo(addr); err == nil {
		t.Fatal("expected output index overflow error")
	}
}

// TestProtocolParamsParsesRefScriptCostPerByte verifies that the BlockFrost
// min_fee_ref_script_cost_per_byte field is parsed into MinFeeRefScriptCostPerByte
// and surfaced via RefScriptFeePerByte(), so the Conway reference-script fee is
// actually charged (not silently zero) when BlockFrost supplies the price.
func TestProtocolParamsParsesRefScriptCostPerByte(t *testing.T) {
	const body = `{
		"min_fee_a": 44,
		"min_fee_b": 155381,
		"max_tx_size": 16384,
		"coins_per_utxo_size": "4310",
		"collateral_percent": 150,
		"max_collateral_inputs": 3,
		"min_fee_ref_script_cost_per_byte": 15
	}`

	var raw bfProtocolParams
	if err := json.Unmarshal([]byte(body), &raw); err != nil {
		t.Fatal(err)
	}
	pp, err := raw.toProtocolParams()
	if err != nil {
		t.Fatal(err)
	}
	if got := pp.MinFeeRefScriptCostPerByte; got != 15 {
		t.Fatalf("MinFeeRefScriptCostPerByte = %v, want 15", got)
	}
	if got := pp.RefScriptFeePerByte(); got != 15 {
		t.Fatalf("RefScriptFeePerByte() = %v, want 15", got)
	}
	if got, want := pp.RefScriptFeePerByteRational(), big.NewRat(15, 1); got.Cmp(want) != 0 {
		t.Fatalf("RefScriptFeePerByteRational() = %s, want %s", got, want)
	}
}

func TestProtocolParamsPreservesFractionalRefScriptCostPerByte(t *testing.T) {
	const body = `{
		"min_fee_a": 44,
		"min_fee_b": 155381,
		"max_tx_size": 16384,
		"coins_per_utxo_size": "4310",
		"collateral_percent": 150,
		"max_collateral_inputs": 3,
		"min_fee_ref_script_cost_per_byte": 15.125
	}`

	var raw bfProtocolParams
	if err := json.Unmarshal([]byte(body), &raw); err != nil {
		t.Fatal(err)
	}
	pp, err := raw.toProtocolParams()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := pp.RefScriptFeePerByteRational(), big.NewRat(121, 8); got.Cmp(want) != 0 {
		t.Fatalf("RefScriptFeePerByteRational() = %s, want %s", got, want)
	}
}

func TestProtocolParamsPrefersCostModelsRaw(t *testing.T) {
	const body = `{
		"min_fee_a": 44,
		"min_fee_b": 155381,
		"max_tx_size": 16384,
		"coins_per_utxo_size": "4310",
		"collateral_percent": 150,
		"max_collateral_inputs": 3,
		"cost_models": {
			"PlutusV3": {"addInteger-cpu-arguments-intercept": 1, "addInteger-cpu-arguments-slope": 2}
		},
		"cost_models_raw": {
			"PlutusV3": [100, 200, 300, 400]
		}
	}`

	var raw bfProtocolParams
	if err := json.Unmarshal([]byte(body), &raw); err != nil {
		t.Fatal(err)
	}
	pp, err := raw.toProtocolParams()
	if err != nil {
		t.Fatal(err)
	}
	got := pp.CostModels["PlutusV3"]
	if len(got) != 4 || got[0] != 100 || got[3] != 400 {
		t.Fatalf("expected cost_models_raw values, got %v", got)
	}
}

func TestProtocolParamsFallsBackToNamedCostModels(t *testing.T) {
	const body = `{
		"min_fee_a": 44,
		"min_fee_b": 155381,
		"max_tx_size": 16384,
		"coins_per_utxo_size": "4310",
		"collateral_percent": 150,
		"max_collateral_inputs": 3,
		"cost_models": {
			"PlutusV2": [9, 8, 7]
		}
	}`

	var raw bfProtocolParams
	if err := json.Unmarshal([]byte(body), &raw); err != nil {
		t.Fatal(err)
	}
	pp, err := raw.toProtocolParams()
	if err != nil {
		t.Fatal(err)
	}
	got := pp.CostModels["PlutusV2"]
	if len(got) != 3 || got[0] != 9 {
		t.Fatalf("expected named cost_models fallback, got %v", got)
	}
}

// sampleCommonUtxo builds a resolved gouroboros UTxO for additional-UTxO
// request-shaping tests: a known tx ref, address, lovelace coin, one native
// asset, and a PlutusV2 reference script.
func sampleCommonUtxo(t *testing.T) common.Utxo {
	t.Helper()
	var txId common.Blake2b256
	for i := range txId {
		txId[i] = 0x11
	}
	input := shelley.ShelleyTransactionInput{TxId: txId, OutputIndex: 3}

	var policyId common.Blake2b224
	for i := range policyId {
		policyId[i] = 0xAB
	}
	assetName := []byte("TOKEN")
	assetData := map[common.Blake2b224]map[cbor.ByteString]*big.Int{
		policyId: {cbor.NewByteString(assetName): big.NewInt(42)},
	}
	ma := common.NewMultiAsset[common.MultiAssetTypeOutput](assetData)

	output := babbage.BabbageTransactionOutput{
		OutputAddress: testAddress(t),
		OutputAmount: mary.MaryTransactionOutputValue{
			Amount: 1_500_000,
			Assets: &ma,
		},
		TxOutScriptRef: &common.ScriptRef{
			Type:   common.ScriptRefTypePlutusV2,
			Script: common.PlutusV2Script([]byte{0x49, 0x48, 0x01, 0x00}),
		},
	}
	return common.Utxo{Id: input, Output: &output}
}

func TestBuildEvalUtxosRequestShape(t *testing.T) {
	txCbor := []byte{0x84, 0xa3, 0x00}
	body, err := buildEvalUtxosRequest(txCbor, []common.Utxo{sampleCommonUtxo(t)})
	if err != nil {
		t.Fatal(err)
	}

	// Decode generically to assert exact JSON keys and casing.
	var req map[string]json.RawMessage
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}
	var cborField string
	if err := json.Unmarshal(req["cbor"], &cborField); err != nil {
		t.Fatalf("cbor field: %v", err)
	}
	if cborField != hex.EncodeToString(txCbor) {
		t.Fatalf("cbor = %q, want %q", cborField, hex.EncodeToString(txCbor))
	}

	var set []json.RawMessage
	if err := json.Unmarshal(req["additionalUtxoSet"], &set); err != nil {
		t.Fatalf("additionalUtxoSet field: %v", err)
	}
	if len(set) != 1 {
		t.Fatalf("additionalUtxoSet len = %d, want 1", len(set))
	}

	// Each item is a [txIn, txOut] pair.
	var pair []json.RawMessage
	if err := json.Unmarshal(set[0], &pair); err != nil {
		t.Fatalf("pair: %v", err)
	}
	if len(pair) != 2 {
		t.Fatalf("pair len = %d, want 2", len(pair))
	}

	var txIn map[string]any
	if err := json.Unmarshal(pair[0], &txIn); err != nil {
		t.Fatalf("txIn: %v", err)
	}
	if txIn["txId"] != strings.Repeat("11", 32) {
		t.Fatalf("txIn.txId = %v", txIn["txId"])
	}
	if txIn["index"] != float64(3) {
		t.Fatalf("txIn.index = %v, want 3", txIn["index"])
	}

	var txOut map[string]json.RawMessage
	if err := json.Unmarshal(pair[1], &txOut); err != nil {
		t.Fatalf("txOut: %v", err)
	}
	if _, ok := txOut["address"]; !ok {
		t.Fatal("txOut missing address")
	}
	// datum_hash / datum must be absent (omitempty) for a script-only output.
	if _, ok := txOut["datumHash"]; ok {
		t.Fatal("txOut should not contain camelCase datumHash")
	}
	if _, ok := txOut["datum_hash"]; ok {
		t.Fatal("txOut should not contain datum_hash")
	}
	if _, ok := txOut["datum"]; ok {
		t.Fatal("txOut should not contain datum")
	}

	var value struct {
		Coins  int64            `json:"coins"`
		Assets map[string]int64 `json:"assets"`
	}
	if err := json.Unmarshal(txOut["value"], &value); err != nil {
		t.Fatalf("value: %v", err)
	}
	if value.Coins != 1_500_000 {
		t.Fatalf("coins = %d, want 1500000", value.Coins)
	}
	assetKey := strings.Repeat("ab", 28) + "." + hex.EncodeToString([]byte("TOKEN"))
	if value.Assets[assetKey] != 42 {
		t.Fatalf("assets[%q] = %d, want 42", assetKey, value.Assets[assetKey])
	}

	// Script must be tagged under "plutus:v2" with the raw script bytes (base16).
	var script map[string]string
	if err := json.Unmarshal(txOut["script"], &script); err != nil {
		t.Fatalf("script: %v", err)
	}
	wantScriptHex := hex.EncodeToString([]byte{0x49, 0x48, 0x01, 0x00})
	if script["plutus:v2"] != wantScriptHex {
		t.Fatalf("script[plutus:v2] = %q, want %q", script["plutus:v2"], wantScriptHex)
	}
	if _, ok := script["plutus:v1"]; ok {
		t.Fatal("script should not contain plutus:v1")
	}
	if _, ok := script["plutus:v3"]; ok {
		t.Fatal("script should not contain plutus:v3")
	}
}

func TestBuildEvalUtxosRequestHashOnlyDatum(t *testing.T) {
	var txId common.Blake2b256
	for i := range txId {
		txId[i] = 0x22
	}
	var datumHash common.Blake2b256
	for i := range datumHash {
		datumHash[i] = 0xCD
	}
	// Encode a hash-only datum option ([0, hash]).
	datumCbor, err := cbor.Encode([]any{0, datumHash})
	if err != nil {
		t.Fatal(err)
	}
	var opt babbage.BabbageTransactionOutputDatumOption
	if err := opt.UnmarshalCBOR(datumCbor); err != nil {
		t.Fatal(err)
	}
	output := babbage.BabbageTransactionOutput{
		OutputAddress: testAddress(t),
		OutputAmount:  mary.MaryTransactionOutputValue{Amount: 2_000_000},
		DatumOption:   &opt,
	}
	utxo := common.Utxo{
		Id:     shelley.ShelleyTransactionInput{TxId: txId, OutputIndex: 0},
		Output: &output,
	}

	body, err := buildEvalUtxosRequest([]byte{0x00}, []common.Utxo{utxo})
	if err != nil {
		t.Fatal(err)
	}
	txOut := decodeSingleTxOut(t, body)

	// The hash-only datum must surface under snake_case "datum_hash" and there
	// must be no inline "datum".
	var datumHashField string
	if err := json.Unmarshal(txOut["datum_hash"], &datumHashField); err != nil {
		t.Fatalf("datum_hash field: %v", err)
	}
	if datumHashField != strings.Repeat("cd", 32) {
		t.Fatalf("datum_hash = %q, want %q", datumHashField, strings.Repeat("cd", 32))
	}
	if _, ok := txOut["datum"]; ok {
		t.Fatal("hash-only datum must not emit inline datum")
	}
	if _, ok := txOut["datumHash"]; ok {
		t.Fatal("must not emit camelCase datumHash")
	}
}

func TestBuildEvalUtxosRequestInlineDatum(t *testing.T) {
	var txId common.Blake2b256
	for i := range txId {
		txId[i] = 0x33
	}
	// Inline datum option [1, #6.24(<datum cbor>)]; datum is a simple integer 1.
	innerDatumCbor := []byte{0x01}
	optCbor, err := cbor.Encode([]any{1, cbor.Tag{Number: 24, Content: innerDatumCbor}})
	if err != nil {
		t.Fatal(err)
	}
	var opt babbage.BabbageTransactionOutputDatumOption
	if err := opt.UnmarshalCBOR(optCbor); err != nil {
		t.Fatal(err)
	}
	output := babbage.BabbageTransactionOutput{
		OutputAddress: testAddress(t),
		OutputAmount:  mary.MaryTransactionOutputValue{Amount: 2_000_000},
		DatumOption:   &opt,
	}
	utxo := common.Utxo{
		Id:     shelley.ShelleyTransactionInput{TxId: txId, OutputIndex: 0},
		Output: &output,
	}

	body, err := buildEvalUtxosRequest([]byte{0x00}, []common.Utxo{utxo})
	if err != nil {
		t.Fatal(err)
	}
	txOut := decodeSingleTxOut(t, body)

	// An inline datum must surface under "datum" as the datum CBOR hex, and not
	// as "datum_hash".
	var datumField string
	if err := json.Unmarshal(txOut["datum"], &datumField); err != nil {
		t.Fatalf("datum field: %v", err)
	}
	if datumField != hex.EncodeToString(innerDatumCbor) {
		t.Fatalf("datum = %q, want %q", datumField, hex.EncodeToString(innerDatumCbor))
	}
	if _, ok := txOut["datumHash"]; ok {
		t.Fatal("inline datum must not emit camelCase datumHash")
	}
	if _, ok := txOut["datum_hash"]; ok {
		t.Fatal("inline datum must not emit datum_hash")
	}
}

func TestBfScriptRefFromScriptRejectsNativeScript(t *testing.T) {
	if _, err := bfScriptRefFromScript(common.NativeScript{}); err == nil {
		t.Fatal("expected unsupported native script error")
	}
}

func TestBfScriptRefFromScriptLanguageDetection(t *testing.T) {
	raw := []byte{0x49, 0x48, 0x01, 0x00}
	wantHex := hex.EncodeToString(raw)

	v1, err := bfScriptRefFromScript(common.PlutusV1Script(raw))
	if err != nil {
		t.Fatal(err)
	}
	if v1.PlutusV1 == nil || *v1.PlutusV1 != wantHex || v1.PlutusV2 != nil || v1.PlutusV3 != nil {
		t.Fatalf("v1 mis-tagged: %+v", v1)
	}
	v3, err := bfScriptRefFromScript(common.PlutusV3Script(raw))
	if err != nil {
		t.Fatal(err)
	}
	if v3.PlutusV3 == nil || *v3.PlutusV3 != wantHex || v3.PlutusV1 != nil || v3.PlutusV2 != nil || v3.PlutusV4 != nil {
		t.Fatalf("v3 mis-tagged: %+v", v3)
	}
	v4, err := bfScriptRefFromScript(common.PlutusV4Script(raw))
	if err != nil {
		t.Fatal(err)
	}
	if v4.PlutusV4 == nil || *v4.PlutusV4 != wantHex || v4.PlutusV1 != nil || v4.PlutusV2 != nil || v4.PlutusV3 != nil {
		t.Fatalf("v4 mis-tagged: %+v", v4)
	}
}

func TestBuildEvalUtxosRequestRejectsOverflowQuantity(t *testing.T) {
	var txId common.Blake2b256
	var policyId common.Blake2b224
	for i := range policyId {
		policyId[i] = 0x01
	}
	overflow := new(big.Int).Add(new(big.Int).SetUint64(math.MaxInt64), big.NewInt(1))
	assetData := map[common.Blake2b224]map[cbor.ByteString]*big.Int{
		policyId: {cbor.NewByteString(nil): overflow},
	}
	ma := common.NewMultiAsset[common.MultiAssetTypeOutput](assetData)
	output := babbage.BabbageTransactionOutput{
		OutputAddress: testAddress(t),
		OutputAmount:  mary.MaryTransactionOutputValue{Amount: 1, Assets: &ma},
	}
	utxo := common.Utxo{
		Id:     shelley.ShelleyTransactionInput{TxId: txId, OutputIndex: 0},
		Output: &output,
	}
	if _, err := buildEvalUtxosRequest([]byte{0x00}, []common.Utxo{utxo}); err == nil {
		t.Fatal("expected overflow asset quantity to be rejected")
	}
}

func TestEvaluateTxPrefersSimpleEndpointWhenAdditionalUtxosProvided(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		if r.URL.Path != "/api/v0/utils/txs/evaluate" {
			t.Errorf("unexpected path %q; simple evaluate should succeed without /evaluate/utxos", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"result":{"EvaluationResult":{"spend:0":{"memory":1700,"steps":476468}}}}`))
	}))
	defer server.Close()

	ctx := NewBlockFrostChainContext(server.URL, 0, "")
	result, err := ctx.EvaluateTx([]byte{0x84}, []common.Utxo{sampleCommonUtxo(t)})
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 1 || paths[0] != "/api/v0/utils/txs/evaluate" {
		t.Fatalf("paths = %v, want single /api/v0/utils/txs/evaluate", paths)
	}
	key := common.RedeemerKey{Tag: common.RedeemerTagSpend, Index: 0}
	if eu := result[key]; eu.Memory != 1700 || eu.Steps != 476468 {
		t.Fatalf("unexpected ExUnits %+v", eu)
	}
}

func TestEvaluateTxFallsBackToUtxosOnMissingInputs(t *testing.T) {
	prevRetries, prevWait := evaluateSimpleRetries, evaluateSimpleRetryWait
	evaluateSimpleRetries, evaluateSimpleRetryWait = 2, 0
	defer func() {
		evaluateSimpleRetries, evaluateSimpleRetryWait = prevRetries, prevWait
	}()

	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		switch r.URL.Path {
		case "/api/v0/utils/txs/evaluate":
			// Mimic Ogmios/Blockfrost reporting unknown inputs on the simple path.
			_, _ = w.Write([]byte(`{"result":{"EvaluationFailure":{"UnknownInputs":["abc#0"]}}}`))
		case "/api/v0/utils/txs/evaluate/utxos":
			_, _ = w.Write([]byte(`{"result":{"EvaluationResult":{"spend:0":{"memory":1700,"steps":476468}}}}`))
		default:
			t.Errorf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	ctx := NewBlockFrostChainContext(server.URL, 0, "")
	// ADA-only additional UTxO: fallback to /evaluate/utxos is allowed.
	result, err := ctx.EvaluateTx([]byte{0x84}, []common.Utxo{sampleAdaOnlyUtxo(t)})
	if err != nil {
		t.Fatal(err)
	}
	wantPaths := []string{
		"/api/v0/utils/txs/evaluate",
		"/api/v0/utils/txs/evaluate",
		"/api/v0/utils/txs/evaluate/utxos",
	}
	if len(paths) != len(wantPaths) {
		t.Fatalf("paths = %v, want %v", paths, wantPaths)
	}
	for i := range wantPaths {
		if paths[i] != wantPaths[i] {
			t.Fatalf("paths = %v, want %v", paths, wantPaths)
		}
	}
	key := common.RedeemerKey{Tag: common.RedeemerTagSpend, Index: 0}
	if eu := result[key]; eu.Memory != 1700 || eu.Steps != 476468 {
		t.Fatalf("unexpected ExUnits %+v", eu)
	}
}

func TestEvaluateTxRetriesSimpleWhenAdditionalUtxosHaveAssets(t *testing.T) {
	prevRetries, prevWait := evaluateSimpleRetries, evaluateSimpleRetryWait
	evaluateSimpleRetries, evaluateSimpleRetryWait = 3, 0
	defer func() {
		evaluateSimpleRetries, evaluateSimpleRetryWait = prevRetries, prevWait
	}()

	var paths []string
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		if r.URL.Path != "/api/v0/utils/txs/evaluate" {
			t.Errorf("unexpected path %q; asset-bearing UTxOs must not use /evaluate/utxos", r.URL.Path)
			return
		}
		attempts++
		if attempts < 3 {
			_, _ = w.Write([]byte(`{"result":{"EvaluationFailure":{"UnknownInputs":["abc#0"]}}}`))
			return
		}
		_, _ = w.Write([]byte(`{"result":{"EvaluationResult":{"spend:0":{"memory":1700,"steps":476468}}}}`))
	}))
	defer server.Close()

	ctx := NewBlockFrostChainContext(server.URL, 0, "")
	result, err := ctx.EvaluateTx([]byte{0x84}, []common.Utxo{sampleCommonUtxo(t)})
	if err != nil {
		t.Fatal(err)
	}
	if attempts != 3 {
		t.Fatalf("simple attempts = %d, want 3", attempts)
	}
	for _, p := range paths {
		if p != "/api/v0/utils/txs/evaluate" {
			t.Fatalf("paths = %v, want only simple evaluate", paths)
		}
	}
	key := common.RedeemerKey{Tag: common.RedeemerTagSpend, Index: 0}
	if eu := result[key]; eu.Memory != 1700 || eu.Steps != 476468 {
		t.Fatalf("unexpected ExUnits %+v", eu)
	}
}

func TestParseEvaluateTxResponseOgmiosFault(t *testing.T) {
	data := []byte(`{"type":"jsonwsp/fault","version":"1.0","servicename":"ogmios",` +
		`"fault":{"code":"client","string":"Invalid request: failed to decode payload from base64 or base16."}}`)
	_, err := parseEvaluateTxResponse(data)
	if err == nil {
		t.Fatal("expected fault error")
	}
	if !strings.Contains(err.Error(), "failed to decode payload") {
		t.Fatalf("error should include fault string, got: %v", err)
	}
}

func TestIsMissingInputsEvalError(t *testing.T) {
	if !isMissingInputsEvalError(errors.New(`script evaluation failed: {"UnknownInputs":[]}`)) {
		t.Fatal("expected UnknownInputs to match")
	}
	if isMissingInputsEvalError(errors.New(`ogmios evaluate fault (client): failed to decode payload from base64 or base16`)) {
		t.Fatal("decode fault must not trigger utxos fallback")
	}
}

// sampleAdaOnlyUtxo builds a resolved ADA-only UTxO suitable for the
// /evaluate/utxos fallback path (no native assets).
func sampleAdaOnlyUtxo(t *testing.T) common.Utxo {
	t.Helper()
	var txId common.Blake2b256
	for i := range txId {
		txId[i] = 0x44
	}
	output := babbage.BabbageTransactionOutput{
		OutputAddress: testAddress(t),
		OutputAmount:  mary.MaryTransactionOutputValue{Amount: 2_000_000},
	}
	return common.Utxo{
		Id:     shelley.ShelleyTransactionInput{TxId: txId, OutputIndex: 0},
		Output: &output,
	}
}

// decodeSingleTxOut decodes an eval request body and returns the txOut of its
// single additional-UTxO item.
func decodeSingleTxOut(t *testing.T, body []byte) map[string]json.RawMessage {
	t.Helper()
	var req map[string]json.RawMessage
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}
	var set []json.RawMessage
	if err := json.Unmarshal(req["additionalUtxoSet"], &set); err != nil {
		t.Fatalf("additionalUtxoSet: %v", err)
	}
	if len(set) != 1 {
		t.Fatalf("additionalUtxoSet len = %d, want 1", len(set))
	}
	var pair []json.RawMessage
	if err := json.Unmarshal(set[0], &pair); err != nil {
		t.Fatalf("pair: %v", err)
	}
	if len(pair) != 2 {
		t.Fatalf("pair len = %d, want 2", len(pair))
	}
	var txOut map[string]json.RawMessage
	if err := json.Unmarshal(pair[1], &txOut); err != nil {
		t.Fatalf("txOut: %v", err)
	}
	return txOut
}

package blockfrost

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/blinklabs-io/gouroboros/ledger/babbage"
	"github.com/blinklabs-io/gouroboros/ledger/common"
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
	_, err := ctx.EvaluateTx([]byte{0x84})
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
	result, err := ctx.EvaluateTx(txCbor)
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
}

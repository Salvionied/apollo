package ogmios

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SundaeSwap-finance/kugo"
	ogmigo "github.com/SundaeSwap-finance/ogmigo/v6"
	"github.com/SundaeSwap-finance/ogmigo/v6/ouroboros/chainsync/num"
	"github.com/SundaeSwap-finance/ogmigo/v6/ouroboros/shared"
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

func TestSharedValueToUtxoRejectsNegativeLovelace(t *testing.T) {
	value := shared.Value{
		shared.AdaPolicy: {
			shared.AdaAsset: num.Int64(-1),
		},
	}
	if _, err := sharedValueToUtxo(common.Blake2b256{}, 0, value, testAddress(t)); err == nil {
		t.Fatal("expected negative lovelace error")
	}
}

func TestSharedValueToUtxoRejectsNegativeAssetQuantity(t *testing.T) {
	value := shared.Value{
		shared.AdaPolicy: {
			shared.AdaAsset: num.Int64(1000000),
		},
		"00000000000000000000000000000000000000000000000000000001": {
			"544f4b454e": num.Int64(-1),
		},
	}
	if _, err := sharedValueToUtxo(common.Blake2b256{}, 0, value, testAddress(t)); err == nil {
		t.Fatal("expected negative asset quantity error")
	}
}

const testTxHashHex = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func testMatch(datumType, datumHashHex string) kugo.Match {
	return kugo.Match{
		TransactionID: testTxHashHex,
		OutputIndex:   0,
		DatumType:     datumType,
		DatumHash:     datumHashHex,
		Value: kugo.Value{
			shared.AdaPolicy: {
				shared.AdaAsset: num.Int64(1000000),
			},
		},
	}
}

// kupoDatumServer serves the Kupo /v1/datums/{hash} endpoint, returning the
// given datum CBOR hex for any requested hash.
func kupoDatumServer(t *testing.T, datumCborHex string) (*kugo.Client, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"datum": datumCborHex})
	}))
	t.Cleanup(server.Close)
	return kugo.New(kugo.WithEndpoint(server.URL)), server
}

func TestMatchToUtxoFetchesAndVerifiesInlineDatum(t *testing.T) {
	datumCbor := []byte{0x18, 0x2a} // CBOR encoding of integer 42
	datumHash := common.Blake2b256Hash(datumCbor)
	client, _ := kupoDatumServer(t, hex.EncodeToString(datumCbor))

	match := testMatch("inline", hex.EncodeToString(datumHash.Bytes()))
	utxo, err := matchToUtxo(t.Context(), match, testAddress(t), client)
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
	if got := output.Datum().Cbor(); !bytes.Equal(got, datumCbor) {
		t.Fatalf("inline datum CBOR = %x, want %x", got, datumCbor)
	}
}

func TestMatchToUtxoRejectsInlineDatumHashMismatch(t *testing.T) {
	// Kupo returns datum bytes that do not hash to the claimed datum hash.
	client, _ := kupoDatumServer(t, "182b")
	datumHash := common.Blake2b256Hash([]byte{0x18, 0x2a})

	match := testMatch("inline", hex.EncodeToString(datumHash.Bytes()))
	if _, err := matchToUtxo(t.Context(), match, testAddress(t), client); err == nil {
		t.Fatal("expected inline datum hash mismatch error")
	}
}

func TestMatchToUtxoRejectsMissingInlineDatum(t *testing.T) {
	client, _ := kupoDatumServer(t, "")
	datumHash := common.Blake2b256Hash([]byte{0x18, 0x2a})

	match := testMatch("inline", hex.EncodeToString(datumHash.Bytes()))
	if _, err := matchToUtxo(t.Context(), match, testAddress(t), client); err == nil {
		t.Fatal("expected missing inline datum error")
	}
}

func TestMatchToUtxoHashDatumProducesHashOption(t *testing.T) {
	datumHash := common.Blake2b256Hash([]byte{0x18, 0x2a})
	datumHashHex := hex.EncodeToString(datumHash.Bytes())

	// No datum fetcher needed for hash datums; nil must not be dereferenced.
	match := testMatch("hash", datumHashHex)
	utxo, err := matchToUtxo(t.Context(), match, testAddress(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	output, ok := utxo.Output.(*babbage.BabbageTransactionOutput)
	if !ok {
		t.Fatalf("unexpected output type %T", utxo.Output)
	}
	if output.Datum() != nil {
		t.Fatal("hash datum must not produce an inline datum")
	}
	gotHash := output.DatumHash()
	if gotHash == nil {
		t.Fatal("expected datum hash to be populated")
	}
	if got := hex.EncodeToString(gotHash.Bytes()); got != datumHashHex {
		t.Fatalf("datum hash = %s, want %s", got, datumHashHex)
	}
}

func TestMatchToUtxoRejectsUnknownDatumType(t *testing.T) {
	datumHash := common.Blake2b256Hash([]byte{0x18, 0x2a})
	match := testMatch("bogus", hex.EncodeToString(datumHash.Bytes()))
	if _, err := matchToUtxo(t.Context(), match, testAddress(t), nil); err == nil {
		t.Fatal("expected unsupported datum type error")
	}
}

func TestKupoScriptToScriptRefVerifiesHash(t *testing.T) {
	scriptBytes := []byte{0x01, 0x02, 0x03}
	script := kugo.Script{
		Language: kugo.ScriptLanguagePlutusV2,
		Script:   hex.EncodeToString(scriptBytes),
	}
	correctHash := hex.EncodeToString(common.PlutusV2Script(scriptBytes).Hash().Bytes())

	ref, err := kupoScriptToScriptRef(script, correctHash)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := ref.Script.(common.PlutusV2Script); !ok {
		t.Fatalf("expected PlutusV2 script, got %T", ref.Script)
	}

	wrongHash := hex.EncodeToString(common.PlutusV1Script(scriptBytes).Hash().Bytes())
	if _, err := kupoScriptToScriptRef(script, wrongHash); err == nil {
		t.Fatal("expected script hash mismatch error")
	}
}

func TestEvaluateResponseToExUnitsRejectsZeroResults(t *testing.T) {
	if _, err := evaluateResponseToExUnits(nil); err == nil {
		t.Fatal("expected error for nil response")
	}
	if _, err := evaluateResponseToExUnits(&ogmigo.EvaluateTxResponse{}); err == nil {
		t.Fatal("expected error for zero evaluation results")
	}
}

func TestEvaluateResponseToExUnitsReportsErrors(t *testing.T) {
	resp := &ogmigo.EvaluateTxResponse{
		Error: &ogmigo.EvaluateTxError{Code: 3010, Message: "script failure"},
	}
	if _, err := evaluateResponseToExUnits(resp); err == nil {
		t.Fatal("expected evaluation error")
	}
}

func TestEvaluateResponseToExUnitsConvertsResults(t *testing.T) {
	resp := &ogmigo.EvaluateTxResponse{
		ExUnits: []ogmigo.ExUnits{
			{
				Validator: ogmigo.Validator{Purpose: "spend", Index: 0},
				Budget:    ogmigo.ExUnitsBudget{Memory: 1700, Cpu: 476468},
			},
		},
	}
	result, err := evaluateResponseToExUnits(resp)
	if err != nil {
		t.Fatal(err)
	}
	key := common.RedeemerKey{Tag: common.RedeemerTagSpend, Index: 0}
	if eu := result[key]; eu.Memory != 1700 || eu.Steps != 476468 {
		t.Fatalf("unexpected budget %+v", eu)
	}
}

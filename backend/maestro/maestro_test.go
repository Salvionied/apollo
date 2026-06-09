package maestro

import (
	"bytes"
	"encoding/hex"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/maestro-org/go-sdk/models"
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

func TestMaestroUtxoToCommonRejectsInvalidAssetUnit(t *testing.T) {
	addr := testAddress(t)
	raw := models.Utxo{
		TxHash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Index:  0,
		Assets: []models.Asset{
			{Unit: "lovelace", Amount: 1000000},
			{Unit: "abcd", Amount: 1},
		},
	}
	if _, err := maestroUtxoToCommon(raw, addr); err == nil {
		t.Fatal("expected invalid asset unit error")
	}
}

func TestMaestroUtxoToCommonRejectsNegativeAssetQuantity(t *testing.T) {
	addr := testAddress(t)
	raw := models.Utxo{
		TxHash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Index:  0,
		Assets: []models.Asset{
			{Unit: "lovelace", Amount: 1000000},
			{Unit: "00000000000000000000000000000000000000000000000000000001", Amount: -1},
		},
	}
	if _, err := maestroUtxoToCommon(raw, addr); err == nil {
		t.Fatal("expected negative asset quantity error")
	}
}

func TestMaestroUtxoToCommonRejectsOutputIndexOverflow(t *testing.T) {
	addr := testAddress(t)
	raw := models.Utxo{
		TxHash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Index:  int64(math.MaxUint32) + 1,
		Assets: []models.Asset{{Unit: "lovelace", Amount: 1000000}},
	}
	if _, err := maestroUtxoToCommon(raw, addr); err == nil {
		t.Fatal("expected output index overflow error")
	}
}

const testTxHashHex = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func testUtxoWithDatum(datum any) models.Utxo {
	return models.Utxo{
		TxHash: testTxHashHex,
		Index:  0,
		Assets: []models.Asset{{Unit: "lovelace", Amount: 1000000}},
		Datum:  datum,
	}
}

func TestMaestroUtxoToCommonHashDatumWithResolvedBytes(t *testing.T) {
	// Maestro can return resolved datum bytes even for type "hash" datums;
	// these must produce a hash datum option, never an inline datum.
	datumCbor := []byte{0x18, 0x2a}
	datumHash := common.Blake2b256Hash(datumCbor)
	datumHashHex := hex.EncodeToString(datumHash.Bytes())
	raw := testUtxoWithDatum(map[string]any{
		"type":  "hash",
		"hash":  datumHashHex,
		"bytes": hex.EncodeToString(datumCbor),
	})

	utxo, err := maestroUtxoToCommon(raw, testAddress(t))
	if err != nil {
		t.Fatal(err)
	}
	output := utxo.Output
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

func TestMaestroUtxoToCommonInlineDatum(t *testing.T) {
	datumCbor := []byte{0x18, 0x2a}
	raw := testUtxoWithDatum(map[string]any{
		"type":  "inline",
		"hash":  hex.EncodeToString(common.Blake2b256Hash(datumCbor).Bytes()),
		"bytes": hex.EncodeToString(datumCbor),
	})

	utxo, err := maestroUtxoToCommon(raw, testAddress(t))
	if err != nil {
		t.Fatal(err)
	}
	datum := utxo.Output.Datum()
	if datum == nil {
		t.Fatal("expected inline datum to be populated")
	}
	if got := datum.Cbor(); !bytes.Equal(got, datumCbor) {
		t.Fatalf("inline datum CBOR = %x, want %x", got, datumCbor)
	}
}

func TestMaestroUtxoToCommonRejectsInlineDatumWithoutBytes(t *testing.T) {
	raw := testUtxoWithDatum(map[string]any{
		"type": "inline",
		"hash": hex.EncodeToString(common.Blake2b256Hash([]byte{0x18, 0x2a}).Bytes()),
	})
	if _, err := maestroUtxoToCommon(raw, testAddress(t)); err == nil {
		t.Fatal("expected missing inline datum bytes error")
	}
}

func TestMaestroUtxoToCommonRejectsUnknownDatumType(t *testing.T) {
	raw := testUtxoWithDatum(map[string]any{
		"type":  "bogus",
		"bytes": "182a",
	})
	if _, err := maestroUtxoToCommon(raw, testAddress(t)); err == nil {
		t.Fatal("expected unsupported datum type error")
	}
}

func TestMaestroScriptRefVerifiesHash(t *testing.T) {
	scriptBytes := []byte{0x01, 0x02, 0x03}
	correctHash := hex.EncodeToString(common.PlutusV2Script(scriptBytes).Hash().Bytes())

	ref, err := maestroScriptRef("plutusv2", scriptBytes, correctHash)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := ref.Script.(common.PlutusV2Script); !ok {
		t.Fatalf("expected PlutusV2 script, got %T", ref.Script)
	}

	wrongHash := hex.EncodeToString(common.PlutusV1Script(scriptBytes).Hash().Bytes())
	if _, err := maestroScriptRef("plutusv2", scriptBytes, wrongHash); err == nil {
		t.Fatal("expected script hash mismatch error")
	}
}

func TestMaestroScriptRefRejectsUnknownType(t *testing.T) {
	if _, err := maestroScriptRef("plutusv9", []byte{0x01}, ""); err == nil {
		t.Fatal("expected unknown script type error")
	}
}

func TestNewMaestroChainContextWithNetworkAllowlist(t *testing.T) {
	for _, network := range []string{"mainnet", "preprod", "preview", "Mainnet"} {
		if _, err := NewMaestroChainContextWithNetwork(0, "project-id", network); err != nil {
			t.Fatalf("expected network %q to be accepted: %v", network, err)
		}
	}
	for _, network := range []string{"", "sancho", "evil.example.com", "mainnet.attacker"} {
		if _, err := NewMaestroChainContextWithNetwork(0, "project-id", network); err == nil {
			t.Fatalf("expected network %q to be rejected", network)
		}
	}
}

func TestSubmitTxPostsCborToTxManager(t *testing.T) {
	txCbor := []byte{0x84, 0xa3, 0x00}
	wantHash := bytes.Repeat([]byte{0xab}, common.Blake2b256Size)
	var gotPath, gotContentType string
	var gotBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotContentType = r.Header.Get("Content-Type")
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("failed to read request body: %v", err)
		}
		gotBody = body
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(hex.EncodeToString(wantHash)))
	}))
	defer server.Close()

	ctx, err := NewMaestroChainContextWithNetwork(0, "project-id", "preprod")
	if err != nil {
		t.Fatal(err)
	}
	ctx.client.BaseUrl = server.URL

	txHash, err := ctx.SubmitTx(txCbor)
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/txmanager" {
		t.Fatalf("submit path = %q, want /txmanager", gotPath)
	}
	if gotContentType != "application/cbor" {
		t.Fatalf("Content-Type = %q, want application/cbor", gotContentType)
	}
	if !bytes.Equal(gotBody, txCbor) {
		t.Fatalf("request body = %x, want raw tx CBOR %x", gotBody, txCbor)
	}
	if !bytes.Equal(txHash.Bytes(), wantHash) {
		t.Fatalf("tx hash = %x, want %x", txHash.Bytes(), wantHash)
	}
}

func TestEvaluationsToExUnitsRejectsZeroResults(t *testing.T) {
	if _, err := evaluationsToExUnits(nil); err == nil {
		t.Fatal("expected error for zero evaluation results")
	}
	if _, err := evaluationsToExUnits(models.EvaluateTxResponse{}); err == nil {
		t.Fatal("expected error for zero evaluation results")
	}
}

func TestEvaluationsToExUnitsConvertsResults(t *testing.T) {
	evals := models.EvaluateTxResponse{
		{
			RedeemerTag:   "spend",
			RedeemerIndex: 0,
			ExUnits:       models.ExecutionUnits{Mem: 1700, Steps: 476468},
		},
		{
			RedeemerTag:   "mint",
			RedeemerIndex: 1,
			ExUnits:       models.ExecutionUnits{Mem: 250, Steps: 1000},
		},
	}
	result, err := evaluationsToExUnits(evals)
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

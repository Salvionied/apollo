package blockfrost

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
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

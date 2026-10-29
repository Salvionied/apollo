package apollo

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"testing"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/dijkstra"
)

// signAndVerifyDijkstra signs a with a fixed key and checks the resulting
// transaction against gouroboros: the witness must be present in the encoded
// transaction and verify against the ledger transaction hash.
func signAndVerifyDijkstra(t *testing.T, a *Apollo) {
	t.Helper()
	seed := bytes.Repeat([]byte{0x42}, ed25519.SeedSize)
	if _, err := a.SignWithSkey(seed); err != nil {
		t.Fatal(err)
	}
	txCbor, err := a.GetTxCbor()
	if err != nil {
		t.Fatal(err)
	}
	tx, err := dijkstra.NewDijkstraTransactionFromCbor(txCbor)
	if err != nil {
		t.Fatalf("decode signed Dijkstra transaction: %v", err)
	}
	witnesses := tx.WitnessSet.VkeyWitnesses.Items()
	if len(witnesses) != 1 {
		t.Fatalf("vkey witnesses = %d, want 1", len(witnesses))
	}
	txHash := tx.Hash()
	if !ed25519.Verify(witnesses[0].Vkey, txHash.Bytes(), witnesses[0].Signature) {
		t.Fatal("signature does not verify against the ledger transaction hash")
	}
}

func completedDijkstraPayment(t *testing.T) *Apollo {
	t.Helper()
	cc := dijkstraTestContext(t, dijkstra.MinProtocolVersionDijkstra)
	addr := testAddress(t)
	addTestUtxo(cc, addr, 10_000_000, 0x01, 0)
	payment, err := NewPayment(validTestAddrBech32, 2_000_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	a, err := newTestApollo(cc).
		SetWallet(NewExternalWallet(addr)).
		SetTargetEra(EraDijkstra).
		AddPayment(payment).
		Complete()
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func TestSignDijkstraTransaction(t *testing.T) {
	signAndVerifyDijkstra(t, completedDijkstraPayment(t))
}

func TestSignLoadedDijkstraTransaction(t *testing.T) {
	built := completedDijkstraPayment(t)
	txCbor, err := built.GetTxCbor()
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := newTestApollo(dijkstraTestContext(t, dijkstra.MinProtocolVersionDijkstra)).LoadTxCbor(hex.EncodeToString(txCbor))
	if err != nil {
		t.Fatal(err)
	}
	signAndVerifyDijkstra(t, loaded)
}

// TestSignLoadedDijkstraTransactionPreservesBodyBytes signs a loaded
// transaction whose body uses a valid but non-canonical indefinite-length
// map. The witness must sign the original body bytes, which define the
// transaction hash, rather than a canonical re-encoding.
func TestSignLoadedDijkstraTransactionPreservesBodyBytes(t *testing.T) {
	txCbor, err := completedDijkstraPayment(t).GetTxCbor()
	if err != nil {
		t.Fatal(err)
	}
	var envelope []cbor.RawMessage
	if _, err := cbor.Decode(txCbor, &envelope); err != nil {
		t.Fatal(err)
	}
	body := []byte(envelope[0])
	if body[0] < 0xa1 || body[0] > 0xb7 {
		t.Fatalf("body header 0x%02x is not a small definite-length map", body[0])
	}
	indefinite := append([]byte{0xbf}, body[1:]...)
	indefinite = append(indefinite, 0xff)
	rewritten := dijkstraEnvelope(indefinite, envelope[1], envelope[2])

	loaded, err := newTestApollo(dijkstraTestContext(t, dijkstra.MinProtocolVersionDijkstra)).LoadTxCbor(hex.EncodeToString(rewritten))
	if err != nil {
		t.Fatal(err)
	}
	signAndVerifyDijkstra(t, loaded)
	signedCbor, err := loaded.GetTxCbor()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(signedCbor, indefinite) {
		t.Fatal("signed transaction must keep the original body bytes")
	}
}

func TestLoadTxCborRejectsMalformedDijkstraTransaction(t *testing.T) {
	txCbor, err := completedDijkstraPayment(t).GetTxCbor()
	if err != nil {
		t.Fatal(err)
	}
	var envelope []cbor.RawMessage
	if _, err := cbor.Decode(txCbor, &envelope); err != nil {
		t.Fatal(err)
	}
	// A Dijkstra witness set must not carry Plutus V4 scripts (field 8).
	v4Witness, err := cbor.Encode(map[uint]any{8: []any{[]byte{0x01}}})
	if err != nil {
		t.Fatal(err)
	}
	withV4Witness := dijkstraEnvelope(envelope[0], v4Witness, envelope[2])

	tests := map[string][]byte{
		"truncated":         txCbor[:len(txCbor)-1],
		"trailing bytes":    append(append([]byte(nil), txCbor...), 0x00),
		"v4 witness script": withV4Witness,
	}
	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := New(setupFixedContext()).LoadTxCbor(hex.EncodeToString(input)); err == nil {
				t.Fatal("LoadTxCbor accepted a malformed transaction")
			}
		})
	}
}

// dijkstraEnvelope assembles a three-component Dijkstra transaction array from
// already-encoded body, witness set, and auxiliary data.
func dijkstraEnvelope(components ...[]byte) []byte {
	size := 1
	for _, component := range components {
		size += len(component)
	}
	out := make([]byte, 0, size)
	out = append(out, 0x83)
	for _, component := range components {
		out = append(out, component...)
	}
	return out
}

func TestWalletSignLoadedDijkstraTransaction(t *testing.T) {
	w, err := NewBursaWallet(signingTestMnemonic)
	if err != nil {
		t.Fatal(err)
	}
	txCbor, err := completedDijkstraPayment(t).GetTxCbor()
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := newTestApollo(dijkstraTestContext(t, dijkstra.MinProtocolVersionDijkstra)).
		SetWallet(w).
		LoadTxCbor(hex.EncodeToString(txCbor))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := loaded.Sign(); err != nil {
		t.Fatal(err)
	}
	signedCbor, err := loaded.GetTxCbor()
	if err != nil {
		t.Fatal(err)
	}
	tx, err := dijkstra.NewDijkstraTransactionFromCbor(signedCbor)
	if err != nil {
		t.Fatal(err)
	}
	witnesses := tx.WitnessSet.VkeyWitnesses.Items()
	if len(witnesses) != 1 {
		t.Fatalf("vkey witnesses = %d, want 1", len(witnesses))
	}
	txHash := tx.Hash()
	if !ed25519.Verify(witnesses[0].Vkey, txHash.Bytes(), witnesses[0].Signature) {
		t.Fatal("signature does not verify against the ledger transaction hash")
	}
}

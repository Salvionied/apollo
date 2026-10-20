package apollo

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"testing"

	"github.com/blinklabs-io/gouroboros/ledger/conway"
)

func TestSignLoadedConwayTransaction(t *testing.T) {
	cc := setupFixedContext()
	addr := testAddress(t)
	addTestUtxo(cc, addr, 10_000_000, 0x01, 0)
	payment, err := NewPayment(validTestAddrBech32, 2_000_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	built, err := New(cc).SetWallet(NewExternalWallet(addr)).AddPayment(payment).Complete()
	if err != nil {
		t.Fatal(err)
	}
	txCbor, err := built.GetTxCbor()
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := New(setupFixedContext()).LoadTxCbor(hex.EncodeToString(txCbor))
	if err != nil {
		t.Fatal(err)
	}
	seed := bytes.Repeat([]byte{0x42}, ed25519.SeedSize)
	if _, err := loaded.SignWithSkey(seed); err != nil {
		t.Fatal(err)
	}
	signedCbor, err := loaded.GetTxCbor()
	if err != nil {
		t.Fatal(err)
	}
	tx, err := conway.NewConwayTransactionFromCbor(signedCbor)
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
	trailing := hex.EncodeToString(append(append([]byte(nil), txCbor...), 0x00))
	if _, err := New(setupFixedContext()).LoadTxCbor(trailing); err == nil {
		t.Fatal("LoadTxCbor accepted a Conway transaction with trailing bytes")
	}
}

func TestWalletSignLoadedConwayTransaction(t *testing.T) {
	w, err := NewBursaWallet(signingTestMnemonic)
	if err != nil {
		t.Fatal(err)
	}
	cc := setupFixedContext()
	addr := testAddress(t)
	addTestUtxo(cc, addr, 10_000_000, 0x01, 0)
	payment, err := NewPayment(validTestAddrBech32, 2_000_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	built, err := New(cc).SetWallet(NewExternalWallet(addr)).AddPayment(payment).Complete()
	if err != nil {
		t.Fatal(err)
	}
	txCbor, err := built.GetTxCbor()
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := New(setupFixedContext()).SetWallet(w).LoadTxCbor(hex.EncodeToString(txCbor))
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
	tx, err := conway.NewConwayTransactionFromCbor(signedCbor)
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

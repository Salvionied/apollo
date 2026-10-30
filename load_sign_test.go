package apollo

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/blinklabs-io/gouroboros/ledger/common"
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
	built, err := newTestApollo(cc).SetWallet(NewExternalWallet(addr)).AddPayment(payment).Complete()
	if err != nil {
		t.Fatal(err)
	}
	txCbor, err := built.GetTxCbor()
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := newTestApollo(setupFixedContext()).LoadTxCbor(hex.EncodeToString(txCbor))
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
	if _, err := newTestApollo(setupFixedContext()).LoadTxCbor(trailing); err == nil {
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
	built, err := newTestApollo(cc).SetWallet(NewExternalWallet(addr)).AddPayment(payment).Complete()
	if err != nil {
		t.Fatal(err)
	}
	txCbor, err := built.GetTxCbor()
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := newTestApollo(setupFixedContext()).SetWallet(w).LoadTxCbor(hex.EncodeToString(txCbor))
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

func TestLoadedConwayBodyMutationIsRejected(t *testing.T) {
	cc := setupFixedContext()
	addr := testAddress(t)
	addTestUtxo(cc, addr, 10_000_000, 0x01, 0)
	payment, err := NewPayment(validTestAddrBech32, 2_000_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	built, err := newTestApollo(cc).SetWallet(NewExternalWallet(addr)).AddPayment(payment).Complete()
	if err != nil {
		t.Fatal(err)
	}
	txCbor, err := built.GetTxCbor()
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := newTestApollo(setupFixedContext()).LoadTxCbor(hex.EncodeToString(txCbor))
	if err != nil {
		t.Fatal(err)
	}
	loaded.GetTx().Body.TxOutputs[0].OutputAmount.Amount++
	seed := bytes.Repeat([]byte{0x42}, ed25519.SeedSize)
	if _, err := loaded.SignWithSkey(seed); err == nil || !strings.Contains(err.Error(), "body differs from preserved CBOR") {
		t.Fatalf("SignWithSkey accepted a changed decoded body: %v", err)
	}
	if _, err := loaded.GetTxCbor(); err == nil || !strings.Contains(err.Error(), "body differs from preserved CBOR") {
		t.Fatalf("GetTxCbor accepted a changed decoded body: %v", err)
	}
	wallet, err := NewBursaWallet(signingTestMnemonic)
	if err != nil {
		t.Fatal(err)
	}
	loaded.SetWallet(wallet)
	if _, err := loaded.Sign(); err == nil || !strings.Contains(err.Error(), "body differs from preserved CBOR") {
		t.Fatalf("Sign accepted a changed decoded body: %v", err)
	}
	if _, err := loaded.Submit(); err == nil || !strings.Contains(err.Error(), "body differs from preserved CBOR") {
		t.Fatalf("Submit accepted a changed decoded body: %v", err)
	}
	cloned := loaded.Clone()
	if _, err := cloned.SignWithSkey(seed); err == nil || !strings.Contains(err.Error(), "body differs from preserved CBOR") {
		t.Fatalf("Clone discarded the changed body and signed the original: %v", err)
	}
	loaded.GetTx().SetCbor(nil)
	loaded.GetTx().Body.SetCbor(nil)
	if _, err := loaded.SignWithSkey(seed); err == nil || !strings.Contains(err.Error(), "body differs from preserved CBOR") {
		t.Fatalf("clearing the raw CBOR bypassed the changed-body check: %v", err)
	}
}

func TestSignLoadedConwayTransactionWithWithdrawalMap(t *testing.T) {
	cc := setupFixedContext()
	addr := testAddress(t)
	addTestUtxo(cc, addr, 10_000_000, 0x01, 0)
	payment, err := NewPayment(validTestAddrBech32, 2_000_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	built, err := newTestApollo(cc).SetWallet(NewExternalWallet(addr)).AddPayment(payment).Complete()
	if err != nil {
		t.Fatal(err)
	}
	stakeHash := common.Blake2b224Hash([]byte("test withdrawal key"))
	stakeAddress, err := common.NewAddressFromParts(
		common.AddressTypeNoneKey,
		common.AddressNetworkTestnet,
		nil,
		stakeHash.Bytes(),
	)
	if err != nil {
		t.Fatal(err)
	}
	built.GetTx().Body.TxWithdrawals = map[*common.Address]uint64{&stakeAddress: 1_000_000}
	txCbor, err := built.GetTxCbor()
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := newTestApollo(setupFixedContext()).LoadTxCbor(hex.EncodeToString(txCbor))
	if err != nil {
		t.Fatal(err)
	}
	seed := bytes.Repeat([]byte{0x42}, ed25519.SeedSize)
	if _, err := loaded.SignWithSkey(seed); err != nil {
		t.Fatalf("unchanged loaded withdrawal transaction was rejected: %v", err)
	}
}

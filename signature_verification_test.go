package apollo

import (
	"bytes"
	"crypto/ed25519"
	"fmt"
	"testing"

	"github.com/blinklabs-io/bursa"
	"github.com/blinklabs-io/bursa/bip32"
	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/common"

	"github.com/Salvionied/apollo/v2/backend"
	"github.com/Salvionied/apollo/v2/backend/fixed"
)

// signingTestMnemonic is the canonical all-zero-entropy BIP39 test vector. It
// is published in the BIP39 specification itself, so it can never hold real
// funds and is safe to hardcode.
const signingTestMnemonic = "abandon abandon abandon abandon abandon " +
	"abandon abandon abandon abandon abandon abandon about"

// bodyFeeKey is the transaction_body map key for the fee, per the Conway CDDL
// (0 inputs, 1 outputs, 2 fee, 3 ttl, ...).
const bodyFeeKey = 2

// submitCaptureContext records the exact bytes handed to SubmitTx. That is the
// only place the transaction that would go on the wire can be observed, so the
// signature checks below run against those bytes rather than against builder
// state. It performs no network I/O.
type submitCaptureContext struct {
	*fixed.FixedChainContext
	submitted []byte
}

// Capabilities adds submission to the fixed context's in-memory operations.
func (c *submitCaptureContext) Capabilities() backend.CapabilitySet {
	return c.FixedChainContext.Capabilities() |
		backend.CapabilitySet(backend.CapabilitySubmitTx)
}

func (c *submitCaptureContext) SubmitTx(
	txCbor []byte,
) (common.Blake2b256, error) {
	c.submitted = bytes.Clone(txCbor)
	var elements []cbor.RawMessage
	if _, err := cbor.Decode(txCbor, &elements); err != nil {
		return common.Blake2b256{}, err
	}
	if len(elements) == 0 {
		return common.Blake2b256{}, fmt.Errorf("submitted transaction has no body")
	}
	return common.Blake2b256Hash(elements[0]), nil
}

// signAndSubmit builds a single-output transaction spending the wallet's own
// UTxO, signs it, submits it, and returns the submitted CBOR.
func signAndSubmit(t *testing.T, w Wallet) []byte {
	t.Helper()
	cc := &submitCaptureContext{
		FixedChainContext: networkTestContext(
			t,
			common.AddressNetworkMainnet,
		),
	}
	addTestUtxo(cc.FixedChainContext, w.Address(), 10_000_000, 0x01, 0)
	payee := syntheticAddress(
		t,
		common.AddressTypeKeyKey,
		common.AddressNetworkMainnet,
	)
	a, err := New(cc).
		SetWallet(w).
		SetTtl(50_000_000).
		PayToAddress(payee, 2_000_000).
		Complete()
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}
	if a, err = a.Sign(); err != nil {
		t.Fatalf("Sign failed: %v", err)
	}
	if _, err := a.Submit(); err != nil {
		t.Fatalf("Submit failed: %v", err)
	}
	if len(cc.submitted) == 0 {
		t.Fatal("no transaction CBOR reached SubmitTx")
	}
	return cc.submitted
}

// txTopLevelElements splits a transaction into its raw top-level CBOR elements
// ([body, witness_set, is_valid, auxiliary_data]) without interpreting them,
// so element 0 is exactly the byte string the ledger hashes to get the tx id.
func txTopLevelElements(t *testing.T, txCbor []byte) []cbor.RawMessage {
	t.Helper()
	var elements []cbor.RawMessage
	if _, err := cbor.Decode(txCbor, &elements); err != nil {
		t.Fatalf("failed to split transaction into elements: %v", err)
	}
	if len(elements) != 4 {
		t.Fatalf("expected 4 top-level tx elements, got %d", len(elements))
	}
	return elements
}

// submittedVkeyWitnesses decodes the vkey witnesses (witness set key 0) out of
// the raw witness-set element.
func submittedVkeyWitnesses(
	t *testing.T,
	witnessSetCbor []byte,
) []common.VkeyWitness {
	t.Helper()
	var witnessSet map[uint64]cbor.RawMessage
	if _, err := cbor.Decode(witnessSetCbor, &witnessSet); err != nil {
		t.Fatalf("failed to decode submitted witness set: %v", err)
	}
	raw, ok := witnessSet[0]
	if !ok {
		t.Fatal("submitted witness set has no vkey witnesses (key 0)")
	}
	var witnesses cbor.SetType[common.VkeyWitness]
	if _, err := cbor.Decode(raw, &witnesses); err != nil {
		t.Fatalf("failed to decode submitted vkey witnesses: %v", err)
	}
	items := witnesses.Items()
	if len(items) == 0 {
		t.Fatal("submitted transaction carries no vkey witnesses")
	}
	return items
}

// assertSignsSubmittedBodyHash verifies every submitted witness independently
// of Apollo: the signature must be a valid Ed25519 signature over
// blake2b-256(body element), the vkey must hash to wantKeyHash, and the
// signature must not verify over anything else. The negative controls matter:
// without them the test would only prove that something was signed, not that
// the transaction body was.
func assertSignsSubmittedBodyHash(
	t *testing.T,
	txCbor []byte,
	wantKeyHash common.Blake2b224,
) {
	t.Helper()
	elements := txTopLevelElements(t, txCbor)
	body := []byte(elements[0])
	bodyHash := common.Blake2b256Hash(body)

	for i, witness := range submittedVkeyWitnesses(t, elements[1]) {
		if len(witness.Vkey) != ed25519.PublicKeySize {
			t.Fatalf("witness %d vkey is %d bytes, want %d",
				i, len(witness.Vkey), ed25519.PublicKeySize)
		}
		if len(witness.Signature) != ed25519.SignatureSize {
			t.Fatalf("witness %d signature is %d bytes, want %d",
				i, len(witness.Signature), ed25519.SignatureSize)
		}
		vkey := ed25519.PublicKey(witness.Vkey)
		if !ed25519.Verify(vkey, bodyHash.Bytes(), witness.Signature) {
			t.Fatalf(
				"witness %d does not sign blake2b-256 of the submitted body (%s)",
				i, bodyHash,
			)
		}
		// A wallet handing back the stake key as the payment vkey produces a
		// witness that verifies but that no input can be spent with.
		if got := common.Blake2b224Hash(witness.Vkey); got != wantKeyHash {
			t.Fatalf("witness %d vkey hashes to %s, want %s",
				i, got, wantKeyHash)
		}
		if ed25519.Verify(vkey, txCbor, witness.Signature) {
			t.Fatalf("witness %d signs the whole transaction, not the body hash",
				i)
		}
		if ed25519.Verify(vkey, body, witness.Signature) {
			t.Fatalf("witness %d signs the un-hashed body, not its hash", i)
		}
	}
}

// testPaymentKey derives a CIP-1852 payment key from the public test mnemonic.
func testPaymentKey(t *testing.T, addressID uint32) bip32.XPrv {
	t.Helper()
	rootKey, err := bursa.GetRootKeyFromMnemonic(signingTestMnemonic, "")
	if err != nil {
		t.Fatal(err)
	}
	accountKey, err := bursa.GetAccountKey(rootKey, 0)
	if err != nil {
		t.Fatal(err)
	}
	paymentKey, err := bursa.GetPaymentKey(accountKey, addressID)
	if err != nil {
		t.Fatal(err)
	}
	return paymentKey
}

// TestBursaWalletSignsSubmittedBodyHash covers BursaWallet.SignTxBody end to
// end: build, sign, submit, then verify the witness against the bytes on the
// wire.
func TestBursaWalletSignsSubmittedBodyHash(t *testing.T) {
	w, err := NewBursaWallet(signingTestMnemonic)
	if err != nil {
		t.Fatal(err)
	}
	// The vkey assertion below only catches a stake-for-payment key mix-up if
	// the two hashes actually differ for this wallet.
	if w.PubKeyHash() == w.StakePubKeyHash() {
		t.Fatal("test wallet payment and stake key hashes must differ")
	}
	assertSignsSubmittedBodyHash(t, signAndSubmit(t, w), w.PubKeyHash())
}

// TestKeyPairWalletSignsSubmittedBodyHash covers KeyPairWallet.SignTxBody with
// an enterprise address controlled by the raw extended key.
func TestKeyPairWalletSignsSubmittedBodyHash(t *testing.T) {
	key := testPaymentKey(t, 1)
	keyHash := common.Blake2b224Hash(key.Public().PublicKey())
	addr, err := common.NewAddressFromParts(
		common.AddressTypeKeyNone,
		common.AddressNetworkMainnet,
		keyHash.Bytes(),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	w, err := NewKeyPairWallet(addr, key)
	if err != nil {
		t.Fatal(err)
	}
	if w.PubKeyHash() != keyHash {
		t.Fatalf("wallet key hash %s does not control address %s",
			w.PubKeyHash(), addr)
	}
	assertSignsSubmittedBodyHash(t, signAndSubmit(t, w), w.PubKeyHash())
}

// TestSignatureDoesNotSurviveBodyTampering re-encodes the submitted body and
// bumps the fee, the classic post-signing tamper. The unmodified re-encode is
// checked first so a failure of the tampered case cannot be explained by the
// re-encoding itself.
func TestSignatureDoesNotSurviveBodyTampering(t *testing.T) {
	w, err := NewBursaWallet(signingTestMnemonic)
	if err != nil {
		t.Fatal(err)
	}
	txCbor := signAndSubmit(t, w)
	elements := txTopLevelElements(t, txCbor)
	witness := submittedVkeyWitnesses(t, elements[1])[0]
	vkey := ed25519.PublicKey(witness.Vkey)

	signedHash := common.Blake2b256Hash(elements[0])
	if !ed25519.Verify(vkey, signedHash.Bytes(), witness.Signature) {
		t.Fatalf("witness does not sign the submitted body hash (%s)",
			signedHash)
	}

	var body map[uint64]cbor.RawMessage
	if _, err := cbor.Decode(elements[0], &body); err != nil {
		t.Fatalf("failed to decode submitted body fields: %v", err)
	}
	reencoded, err := cbor.Encode(body)
	if err != nil {
		t.Fatal(err)
	}
	hash := common.Blake2b256Hash(reencoded)
	if !ed25519.Verify(vkey, hash.Bytes(), witness.Signature) {
		t.Fatal("re-encoding the untouched body changed the signed hash")
	}

	originalFee, ok := body[bodyFeeKey]
	if !ok {
		t.Fatal("submitted body has no fee field")
	}
	tamperedFee, err := cbor.Encode(uint64(9_999_999))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(originalFee, tamperedFee) {
		t.Fatal("tampered fee matches the original fee")
	}
	body[bodyFeeKey] = tamperedFee
	tampered, err := cbor.Encode(body)
	if err != nil {
		t.Fatal(err)
	}
	tamperedHash := common.Blake2b256Hash(tampered)
	if ed25519.Verify(vkey, tamperedHash.Bytes(), witness.Signature) {
		t.Fatal("signature still verifies after the fee was tampered with")
	}
}

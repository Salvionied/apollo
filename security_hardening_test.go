package apollo

import (
	"crypto/ed25519"
	"fmt"
	"math"
	"math/big"
	"strings"
	"testing"

	"github.com/blinklabs-io/bursa"
	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/common"

	"github.com/Salvionied/apollo/v2/backend/fixed"
)

func TestBufferExUnits(t *testing.T) {
	tests := []struct {
		name   string
		value  int64
		factor float64
		want   int64
	}{
		{"zero", 0, 1.2, 0},
		{"negative clamped to zero", -100, 1.2, 0},
		{"normal value buffered", 1000, 1.2, 1200},
		{"near MaxInt64 saturates instead of wrapping negative", math.MaxInt64, 1.2, math.MaxInt64},
		{"just above overflow threshold saturates", math.MaxInt64/6*5 + 1, 1.2, math.MaxInt64},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := bufferExUnits(tt.value, tt.factor)
			if got != tt.want {
				t.Errorf("bufferExUnits(%d, %v) = %d, want %d", tt.value, tt.factor, got, tt.want)
			}
			if got < 0 {
				t.Errorf("bufferExUnits(%d, %v) = %d, must never be negative", tt.value, tt.factor, got)
			}
		})
	}
}

func TestMintNormalizesPolicyIdCase(t *testing.T) {
	cc := setupFixedContext()
	a := New(cc)

	// Mixed-case hex sorts differently as a string ('A' < 'a') than as bytes
	// (0xab < 0xac), which would misbind mint redeemer indexes. Mint must
	// normalize policy IDs to lowercase.
	lower := "ab012345678901234567890123456789012345678901234567890123"
	upper := "AC012345678901234567890123456789012345678901234567890123"
	redeemer := common.Datum{}
	a = a.Mint(NewUnit(lower, "746f6b656e", 1), &redeemer, nil)
	a = a.Mint(NewUnit(upper, "746f6b656e", 1), &redeemer, nil)

	sorted := a.sortedMintPolicyIds()
	if len(sorted) != 2 {
		t.Fatalf("expected 2 policies, got %d", len(sorted))
	}
	if sorted[0] != "ab012345678901234567890123456789012345678901234567890123" ||
		sorted[1] != "ac012345678901234567890123456789012345678901234567890123" {
		t.Errorf("policies not normalized to lowercase byte order: %v", sorted)
	}
	for policy := range a.mintRedeemers {
		if policy != "ab012345678901234567890123456789012345678901234567890123" &&
			policy != "ac012345678901234567890123456789012345678901234567890123" {
			t.Errorf("mintRedeemers key not lowercased: %q", policy)
		}
	}
}

func TestMintDedupsPolicyIdAcrossCase(t *testing.T) {
	cc := setupFixedContext()
	a := New(cc)

	// The same policy supplied in two cases must collapse to one entry, not
	// produce an extra redeemer index.
	policy := "ab012345678901234567890123456789012345678901234567890123"
	a = a.Mint(NewUnit(policy, "61", 1), nil, nil)
	a = a.Mint(NewUnit("AB012345678901234567890123456789012345678901234567890123", "62", 1), nil, nil)

	sorted := a.sortedMintPolicyIds()
	if len(sorted) != 1 {
		t.Fatalf("expected 1 unique policy, got %d: %v", len(sorted), sorted)
	}
	if sorted[0] != policy {
		t.Errorf("expected %q, got %q", policy, sorted[0])
	}
}

func TestPaymentFromTxOutRejectsOversizedLovelace(t *testing.T) {
	addr, err := common.NewAddress(validTestAddrBech32)
	if err != nil {
		t.Fatalf("failed to parse address: %v", err)
	}
	out := NewBabbageOutput(addr, NewSimpleValue(uint64(math.MaxInt64)+1), nil, nil)
	if _, err := PaymentFromTxOut(&out); err == nil {
		t.Error("expected error for lovelace above MaxInt64, got nil")
	}
	out = NewBabbageOutput(addr, NewSimpleValue(uint64(math.MaxInt64)), nil, nil)
	if _, err := PaymentFromTxOut(&out); err != nil {
		t.Errorf("expected MaxInt64 lovelace to be accepted, got: %v", err)
	}
}

func TestNewPaymentFromValueRejectsOversizedLovelace(t *testing.T) {
	addr, err := common.NewAddress(validTestAddrBech32)
	if err != nil {
		t.Fatalf("failed to parse address: %v", err)
	}
	if _, err := NewPaymentFromValue(addr, NewSimpleValue(uint64(math.MaxInt64)+1)); err == nil {
		t.Error("expected error for lovelace above MaxInt64, got nil")
	}
}

// --- Change output asset normalization ---

func TestNormalizeChangeAssets(t *testing.T) {
	var policyId common.Blake2b224
	for i := range policyId {
		policyId[i] = 0xaa
	}

	t.Run("zero quantities are pruned", func(t *testing.T) {
		assets := common.NewMultiAsset[common.MultiAssetTypeOutput](
			map[common.Blake2b224]map[cbor.ByteString]common.MultiAssetTypeOutput{
				policyId: {cbor.NewByteString([]byte("tok")): big.NewInt(0)},
			})
		got, err := normalizeChangeAssets(&assets)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != nil {
			t.Errorf("expected nil after pruning zero quantities, got %v", got)
		}
	})

	t.Run("negative quantities are an error", func(t *testing.T) {
		assets := common.NewMultiAsset[common.MultiAssetTypeOutput](
			map[common.Blake2b224]map[cbor.ByteString]common.MultiAssetTypeOutput{
				policyId: {cbor.NewByteString([]byte("tok")): big.NewInt(-5)},
			})
		if _, err := normalizeChangeAssets(&assets); err == nil {
			t.Error("expected error for negative change asset quantity, got nil")
		}
	})

	t.Run("positive quantities are preserved", func(t *testing.T) {
		assets := common.NewMultiAsset[common.MultiAssetTypeOutput](
			map[common.Blake2b224]map[cbor.ByteString]common.MultiAssetTypeOutput{
				policyId: {
					cbor.NewByteString([]byte("tok")):  big.NewInt(7),
					cbor.NewByteString([]byte("gone")): big.NewInt(0),
				},
			})
		got, err := normalizeChangeAssets(&assets)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got == nil {
			t.Fatal("expected assets to remain")
		}
		if qty := got.Asset(policyId, []byte("tok")); qty == nil || qty.Cmp(big.NewInt(7)) != 0 {
			t.Errorf("expected tok=7 preserved, got %v", qty)
		}
		if qty := got.Asset(policyId, []byte("gone")); qty != nil {
			t.Errorf("expected zero-quantity asset pruned, got %v", qty)
		}
	})

	t.Run("nil passes through", func(t *testing.T) {
		got, err := normalizeChangeAssets(nil)
		if err != nil || got != nil {
			t.Errorf("expected nil/nil, got %v/%v", got, err)
		}
	})
}

func TestCompleteSpendAllOfTokenOmitsZeroChangeEntry(t *testing.T) {
	cc := setupFixedContext()
	addr := testAddress(t)

	policyHex := strings.Repeat("aa", 28)
	var policyId common.Blake2b224
	for i := range policyId {
		policyId[i] = 0xaa
	}
	assets := common.NewMultiAsset[common.MultiAssetTypeOutput](
		map[common.Blake2b224]map[cbor.ByteString]common.MultiAssetTypeOutput{
			policyId: {cbor.NewByteString([]byte("tok")): big.NewInt(5)},
		})
	var txHash common.Blake2b256
	txHash[0] = 0x01
	cc.AddUtxo(addr, makeAssetTestUtxo(t, txHash, 0, 10_000_000, &assets))

	// Send the entire token balance; the change output must not contain a
	// zero-quantity entry for the token (invalid in Conway-era CDDL).
	p, err := NewPayment(validTestAddrBech32, 2_000_000, []Unit{NewUnit(policyHex, "746f6b", 5)})
	if err != nil {
		t.Fatal(err)
	}
	a := New(cc).SetWallet(NewExternalWallet(addr)).AddPayment(p).SetTtl(50000000)
	a, err = a.Complete()
	if err != nil {
		t.Fatal(err)
	}
	for i, out := range a.GetTx().Body.TxOutputs {
		if out.OutputAmount.Assets == nil {
			continue
		}
		for _, pid := range out.OutputAmount.Assets.Policies() {
			for _, name := range out.OutputAmount.Assets.Assets(pid) {
				qty := out.OutputAmount.Assets.Asset(pid, name)
				if qty == nil || qty.Sign() <= 0 {
					t.Errorf("output %d has non-positive asset quantity %v for %x", i, qty, name)
				}
			}
		}
	}
}

func TestCompleteBurnConsumingFullBalanceOmitsZeroChangeEntry(t *testing.T) {
	cc := setupFixedContext()
	addr := testAddress(t)

	policyHex := strings.Repeat("aa", 28)
	var policyId common.Blake2b224
	for i := range policyId {
		policyId[i] = 0xaa
	}
	assets := common.NewMultiAsset[common.MultiAssetTypeOutput](
		map[common.Blake2b224]map[cbor.ByteString]common.MultiAssetTypeOutput{
			policyId: {cbor.NewByteString([]byte("tok")): big.NewInt(5)},
		})
	var txHash common.Blake2b256
	txHash[0] = 0x01
	cc.AddUtxo(addr, makeAssetTestUtxo(t, txHash, 0, 10_000_000, &assets))

	p, err := NewPayment(validTestAddrBech32, 2_000_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	a := New(cc).SetWallet(NewExternalWallet(addr)).AddPayment(p).SetTtl(50000000)
	a = a.Mint(NewUnit(policyHex, "746f6b", -5), nil, nil)
	a, err = a.Complete()
	if err != nil {
		t.Fatal(err)
	}
	for i, out := range a.GetTx().Body.TxOutputs {
		if out.OutputAmount.Assets == nil {
			continue
		}
		for _, pid := range out.OutputAmount.Assets.Policies() {
			for _, name := range out.OutputAmount.Assets.Assets(pid) {
				qty := out.OutputAmount.Assets.Asset(pid, name)
				if qty == nil || qty.Sign() <= 0 {
					t.Errorf("output %d has non-positive asset quantity %v for %x", i, qty, name)
				}
			}
		}
	}
}

func TestCompleteBurnWithoutTokensFailsClosed(t *testing.T) {
	cc := setupFixedContext()
	addr := testAddress(t)
	addTestUtxo(cc, addr, 10_000_000, 0x01, 0)

	policyHex := strings.Repeat("aa", 28)
	p, err := NewPayment(validTestAddrBech32, 2_000_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	a := New(cc).SetWallet(NewExternalWallet(addr)).AddPayment(p).SetTtl(50000000)
	// Burn tokens that are not present in any available UTxO: the build must
	// fail at selection time instead of signing a non-conserving transaction.
	a = a.Mint(NewUnit(policyHex, "746f6b", -5), nil, nil)
	if _, err := a.Complete(); err == nil {
		t.Error("expected error when burning tokens absent from inputs, got nil")
	}
}

// --- Execution-unit evaluation fail-closed behavior ---

type fakeEvalContext struct {
	*fixed.FixedChainContext
	result map[common.RedeemerKey]common.ExUnits
}

func (f *fakeEvalContext) EvaluateTx(_ []byte, _ []common.Utxo) (map[common.RedeemerKey]common.ExUnits, error) {
	return f.result, nil
}

func TestEstimateExecutionUnitsRejectsEmptyEvalResult(t *testing.T) {
	cc := &fakeEvalContext{
		FixedChainContext: setupFixedContext(),
		result:            map[common.RedeemerKey]common.ExUnits{},
	}
	addr := testAddress(t)
	addTestUtxo(cc.FixedChainContext, addr, 10_000_000, 0x01, 0)
	addTestUtxo(cc.FixedChainContext, addr, 10_000_000, 0x02, 0)

	policyHex := strings.Repeat("aa", 28)
	redeemer := common.Datum{}
	p, err := NewPayment(validTestAddrBech32, 2_000_000, []Unit{NewUnit(policyHex, "746f6b", 5)})
	if err != nil {
		t.Fatal(err)
	}
	a := New(cc).SetWallet(NewExternalWallet(addr)).AddPayment(p).SetTtl(50000000)
	a = a.Mint(NewUnit(policyHex, "746f6b", 5), &redeemer, nil)
	_, err = a.Complete()
	if err == nil {
		t.Fatal("expected error for empty evaluation result, got nil")
	}
	if !strings.Contains(err.Error(), "no results") {
		t.Errorf("expected empty-result error, got: %v", err)
	}
}

func TestEstimateExecutionUnitsRejectsOutOfRangeIndex(t *testing.T) {
	cc := &fakeEvalContext{
		FixedChainContext: setupFixedContext(),
		result: map[common.RedeemerKey]common.ExUnits{
			{Tag: common.RedeemerTagMint, Index: 7}: {Memory: 1000, Steps: 1000},
		},
	}
	addr := testAddress(t)
	addTestUtxo(cc.FixedChainContext, addr, 10_000_000, 0x01, 0)
	addTestUtxo(cc.FixedChainContext, addr, 10_000_000, 0x02, 0)

	policyHex := strings.Repeat("aa", 28)
	redeemer := common.Datum{}
	p, err := NewPayment(validTestAddrBech32, 2_000_000, []Unit{NewUnit(policyHex, "746f6b", 5)})
	if err != nil {
		t.Fatal(err)
	}
	a := New(cc).SetWallet(NewExternalWallet(addr)).AddPayment(p).SetTtl(50000000)
	a = a.Mint(NewUnit(policyHex, "746f6b", 5), &redeemer, nil)
	_, err = a.Complete()
	if err == nil {
		t.Fatal("expected error for out-of-range redeemer index, got nil")
	}
	if !strings.Contains(err.Error(), "out of range") {
		t.Errorf("expected out-of-range error, got: %v", err)
	}
}

// --- Backend-supplied UTxO amount validation ---

type badAmountOutput struct {
	common.TransactionOutput
	amount *big.Int
}

func (b badAmountOutput) Amount() *big.Int {
	return b.amount
}

func TestSumUtxoValuesRejectsInvalidAmount(t *testing.T) {
	a := New(setupFixedContext())
	var txHash common.Blake2b256
	txHash[0] = 0x01
	base := makeAssetTestUtxo(t, txHash, 0, 1_000_000, nil)

	tooBig := new(big.Int).Lsh(big.NewInt(1), 64) // 2^64, outside uint64
	utxo := common.Utxo{Id: base.Id, Output: badAmountOutput{base.Output, tooBig}}
	if _, err := a.sumUtxoValues([]common.Utxo{utxo}); err == nil {
		t.Error("expected error for out-of-range UTxO amount, got nil")
	}

	utxo = common.Utxo{Id: base.Id, Output: badAmountOutput{base.Output, nil}}
	if _, err := a.sumUtxoValues([]common.Utxo{utxo}); err == nil {
		t.Error("expected error for nil UTxO amount, got nil")
	}
}

// --- Ed25519 signing-key consistency ---

func TestNewVkeyWitnessFromSkeyRejectsMismatchedPublicKey(t *testing.T) {
	seed := make([]byte, ed25519.SeedSize)
	for i := range seed {
		seed[i] = byte(i + 1)
	}
	edKey := ed25519.NewKeyFromSeed(seed)
	var txHash common.Blake2b256

	if _, err := NewVkeyWitnessFromSkey(txHash, []byte(edKey)); err != nil {
		t.Fatalf("expected valid 64-byte key to be accepted, got: %v", err)
	}

	// Corrupt the embedded public-key half: signing with it would allow
	// private-scalar recovery from two signatures over the same message.
	corrupted := append([]byte(nil), edKey...)
	corrupted[ed25519.SeedSize] ^= 0xFF
	if _, err := NewVkeyWitnessFromSkey(txHash, corrupted); err == nil {
		t.Error("expected error for mismatched public-key half, got nil")
	}
}

// --- Min-UTxO parameter validation ---

func TestMinLovelacePostAlonzoRejectsInvalidParams(t *testing.T) {
	addr, err := common.NewAddress(validTestAddrBech32)
	if err != nil {
		t.Fatal(err)
	}
	out := NewBabbageOutput(addr, NewSimpleValue(2_000_000), nil, nil)

	if _, err := MinLovelacePostAlonzo(&out, 0); err == nil {
		t.Error("expected error for zero coins_per_utxo_byte, got nil")
	}
	if _, err := MinLovelacePostAlonzo(&out, -1); err == nil {
		t.Error("expected error for negative coins_per_utxo_byte, got nil")
	}
	if _, err := MinLovelacePostAlonzo(&out, math.MaxInt64); err == nil {
		t.Error("expected overflow error for huge coins_per_utxo_byte, got nil")
	}
	got, err := MinLovelacePostAlonzo(&out, 4310)
	if err != nil {
		t.Fatalf("expected valid params to succeed, got: %v", err)
	}
	if got <= 0 {
		t.Errorf("expected positive min lovelace, got %d", got)
	}
}

// --- Metadata bounds on the direct SetShelleyMetadata path ---

func TestToMetadatumEnforcesBounds(t *testing.T) {
	if _, err := toMetadatum(strings.Repeat("a", 65)); err == nil {
		t.Error("expected error for 65-byte metadata text, got nil")
	}
	if _, err := toMetadatum(strings.Repeat("a", 64)); err != nil {
		t.Errorf("expected 64-byte metadata text to be accepted, got: %v", err)
	}
	if _, err := toMetadatum(make([]byte, 65)); err == nil {
		t.Error("expected error for 65-byte metadata bytes, got nil")
	}
	if _, err := toMetadatum(make([]byte, 64)); err != nil {
		t.Errorf("expected 64-byte metadata bytes to be accepted, got: %v", err)
	}

	overMax := new(big.Int).Add(new(big.Int).SetUint64(^uint64(0)), big.NewInt(1))
	if _, err := toMetadatum(overMax); err == nil {
		t.Error("expected error for metadata integer above 2^64-1, got nil")
	}
	underMin := new(big.Int).Neg(overMax)
	if _, err := toMetadatum(underMin); err == nil {
		t.Error("expected error for metadata integer below -(2^64-1), got nil")
	}
	if _, err := toMetadatum(new(big.Int).SetUint64(^uint64(0))); err != nil {
		t.Errorf("expected max metadata integer to be accepted, got: %v", err)
	}
}

// --- Wallet key-derivation and redaction hardening ---

func testMnemonic(t *testing.T) string {
	t.Helper()
	mnemonic, err := bursa.GenerateMnemonic()
	if err != nil {
		t.Fatal(err)
	}
	return mnemonic
}

func TestBursaWalletHonorsPasswordOption(t *testing.T) {
	mnemonic := testMnemonic(t)

	viaOption, err := NewBursaWallet(mnemonic, bursa.WithPassword("s3cret"))
	if err != nil {
		t.Fatal(err)
	}
	viaArg, err := NewBursaWalletWithPassphrase(mnemonic, "s3cret")
	if err != nil {
		t.Fatal(err)
	}
	noPass, err := NewBursaWallet(mnemonic)
	if err != nil {
		t.Fatal(err)
	}

	if viaOption.PubKeyHash() != viaArg.PubKeyHash() {
		t.Error("bursa.WithPassword option must derive the same keys as the passphrase argument")
	}
	if viaOption.PubKeyHash() == noPass.PubKeyHash() {
		t.Error("bursa.WithPassword option was silently ignored")
	}
}

func TestBursaWalletRejectsConflictingPassphrases(t *testing.T) {
	mnemonic := testMnemonic(t)
	if _, err := NewBursaWalletWithPassphrase(mnemonic, "one", bursa.WithPassword("two")); err == nil {
		t.Error("expected error for conflicting passphrases, got nil")
	}
}

func TestBursaWalletDerivationIndices(t *testing.T) {
	mnemonic := testMnemonic(t)

	// Account and address indices must affect both the address and the
	// signing keys consistently (verified internally by the constructor).
	base, err := NewBursaWallet(mnemonic)
	if err != nil {
		t.Fatal(err)
	}
	acct1, err := NewBursaWallet(mnemonic, bursa.WithAccountID(1))
	if err != nil {
		t.Fatal(err)
	}
	if base.PubKeyHash() == acct1.PubKeyHash() {
		t.Error("WithAccountID was silently ignored for signing keys")
	}
	acct1Addr := acct1.Address()
	if acct1.PubKeyHash() != acct1Addr.PaymentKeyHash() {
		t.Error("signing key does not control the wallet address")
	}

	// bursa derives the address from AddressID, so a conflicting PaymentID
	// would bind on-chain artifacts to a key that does not control the
	// displayed address. Must be rejected.
	if _, err := NewBursaWallet(mnemonic, bursa.WithPaymentID(1)); err == nil {
		t.Error("expected error for PaymentID conflicting with AddressID, got nil")
	}

	// Consistent indices are fine.
	addr1, err := NewBursaWallet(mnemonic,
		bursa.WithAddressID(1), bursa.WithPaymentID(1), bursa.WithStakeID(1))
	if err != nil {
		t.Fatal(err)
	}
	addr1Addr := addr1.Address()
	if addr1.PubKeyHash() != addr1Addr.PaymentKeyHash() {
		t.Error("signing key does not control the wallet address for AddressID=1")
	}
}

func TestWalletStringRedactsKeyMaterialByValue(t *testing.T) {
	mnemonic := testMnemonic(t)
	w, err := NewBursaWallet(mnemonic)
	if err != nil {
		t.Fatal(err)
	}

	// Formatting the dereferenced value must not dump struct fields: the
	// String/GoString methods need value receivers to cover this case.
	for _, format := range []string{"%v", "%+v", "%#v", "%s"} {
		out := fmt.Sprintf(format, *w)
		if strings.Contains(out, mnemonic) {
			t.Errorf("format %q leaked the mnemonic", format)
		}
		if !strings.Contains(out, "BursaWallet{address:") {
			t.Errorf("format %q did not use the redacting String method: %q", format, out)
		}
	}
}

func TestNewKeyPairWalletValidatesKeyLength(t *testing.T) {
	addr := testAddress(t)
	if _, err := NewKeyPairWallet(addr, make([]byte, 32)); err == nil {
		t.Error("expected error for 32-byte key, got nil")
	}
	if _, err := NewKeyPairWallet(addr, nil); err == nil {
		t.Error("expected error for nil key, got nil")
	}
	if _, err := NewKeyPairWallet(addr, make([]byte, 96)); err != nil {
		t.Errorf("expected 96-byte key to be accepted, got: %v", err)
	}
}

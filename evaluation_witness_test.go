package apollo

import (
	"crypto/ed25519"
	"errors"
	"strings"
	"testing"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/conway"

	"github.com/Salvionied/apollo/v2/backend/fixed"
)

type evaluationTestWallet struct {
	address common.Address
	key     ed25519.PrivateKey
}

func (w *evaluationTestWallet) Address() common.Address { return w.address }
func (w *evaluationTestWallet) SignTxBody(hash common.Blake2b256) (common.VkeyWitness, error) {
	return common.VkeyWitness{
		Vkey:      w.key.Public().(ed25519.PublicKey),
		Signature: ed25519.Sign(w.key, hash.Bytes()),
	}, nil
}
func (w *evaluationTestWallet) PubKeyHash() common.Blake2b224 {
	return common.Blake2b224Hash(w.key.Public().(ed25519.PublicKey))
}
func (w *evaluationTestWallet) StakePubKeyHash() common.Blake2b224 { return common.Blake2b224{} }

type evaluationTestProvider struct {
	witnesses []common.VkeyWitness
	err       error
	calls     int
	requested [][]common.Blake2b224
}

type signingEvaluationProvider struct {
	key ed25519.PrivateKey
}

func (p signingEvaluationProvider) EvaluationWitnesses(
	hash common.Blake2b256,
	_ []common.Blake2b224,
) ([]common.VkeyWitness, error) {
	return []common.VkeyWitness{{
		Vkey:      p.key.Public().(ed25519.PublicKey),
		Signature: ed25519.Sign(p.key, hash.Bytes()),
	}}, nil
}

func (p *evaluationTestProvider) EvaluationWitnesses(
	_ common.Blake2b256,
	required []common.Blake2b224,
) ([]common.VkeyWitness, error) {
	p.calls++
	p.requested = append(p.requested, append([]common.Blake2b224(nil), required...))
	return p.witnesses, p.err
}

func evaluationKey(seed byte) ed25519.PrivateKey {
	seedBytes := make([]byte, ed25519.SeedSize)
	for i := range seedBytes {
		seedBytes[i] = seed
	}
	return ed25519.NewKeyFromSeed(seedBytes)
}

func evaluationWitness(t *testing.T, key ed25519.PrivateKey, body conway.ConwayTransactionBody) common.VkeyWitness {
	t.Helper()
	body.SetCbor(nil)
	bodyCbor, err := cbor.Encode(&body)
	if err != nil {
		t.Fatal(err)
	}
	hash := common.Blake2b256Hash(bodyCbor)
	return common.VkeyWitness{
		Vkey:      key.Public().(ed25519.PublicKey),
		Signature: ed25519.Sign(key, hash.Bytes()),
	}
}

func evaluationBody(signers ...common.Blake2b224) conway.ConwayTransactionBody {
	body := conway.ConwayTransactionBody{TxFee: 2_000_000}
	if len(signers) > 0 {
		body.TxRequiredSigners = cbor.NewSetType(signers, true)
	}
	return body
}

func TestEvaluationUsesPrimaryPaymentWitness(t *testing.T) {
	key := evaluationKey(1)
	wallet := &evaluationTestWallet{address: testAddress(t), key: key}
	body := evaluationBody(wallet.PubKeyHash())

	witnesses, err := New(setupFixedContext()).SetWallet(wallet).evaluationWitnesses(&body)
	if err != nil {
		t.Fatal(err)
	}
	if len(witnesses) != 1 || string(witnesses[0].Vkey) != string(key.Public().(ed25519.PublicKey)) {
		t.Fatalf("expected primary payment witness, got %#v", witnesses)
	}
}

func TestEvaluationUsesBursaStakeWitness(t *testing.T) {
	wallet, err := NewBursaWallet(testMnemonic(t))
	if err != nil {
		t.Fatal(err)
	}
	body := evaluationBody(wallet.StakePubKeyHash())

	witnesses, err := New(setupFixedContext()).SetWallet(wallet).evaluationWitnesses(&body)
	if err != nil {
		t.Fatal(err)
	}
	if len(witnesses) != 1 || common.Blake2b224Hash(witnesses[0].Vkey) != wallet.StakePubKeyHash() {
		t.Fatalf("expected Bursa stake witness, got %#v", witnesses)
	}
}

func TestEvaluationExternalWalletRequiresProvider(t *testing.T) {
	key := evaluationKey(2)
	hash := common.Blake2b224Hash(key.Public().(ed25519.PublicKey))
	body := evaluationBody(hash)

	_, err := New(setupFixedContext()).SetWallet(NewExternalWallet(testAddress(t))).evaluationWitnesses(&body)
	if err == nil || !strings.Contains(err.Error(), hash.String()) {
		t.Fatalf("expected missing signer %s, got %v", hash, err)
	}
}

func TestEvaluationExternalWalletUsesExplicitProvider(t *testing.T) {
	key := evaluationKey(3)
	hash := common.Blake2b224Hash(key.Public().(ed25519.PublicKey))
	body := evaluationBody(hash)
	provider := &evaluationTestProvider{witnesses: []common.VkeyWitness{evaluationWitness(t, key, body)}}

	witnesses, err := New(setupFixedContext()).SetWallet(NewExternalWallet(testAddress(t))).AddEvaluationWitnessProvider(provider).evaluationWitnesses(&body)
	if err != nil {
		t.Fatal(err)
	}
	if len(witnesses) != 1 || provider.calls != 1 {
		t.Fatalf("expected provider witness, got %d witnesses and %d calls", len(witnesses), provider.calls)
	}
}

func TestEvaluationCombinesMultipleProviders(t *testing.T) {
	firstKey := evaluationKey(31)
	secondKey := evaluationKey(32)
	firstHash := common.Blake2b224Hash(firstKey.Public().(ed25519.PublicKey))
	secondHash := common.Blake2b224Hash(secondKey.Public().(ed25519.PublicKey))
	body := evaluationBody(firstHash, secondHash)
	first := &evaluationTestProvider{witnesses: []common.VkeyWitness{evaluationWitness(t, firstKey, body)}}
	second := &evaluationTestProvider{witnesses: []common.VkeyWitness{evaluationWitness(t, secondKey, body)}}

	witnesses, err := New(setupFixedContext()).AddEvaluationWitnessProvider(first).AddEvaluationWitnessProvider(second).evaluationWitnesses(&body)
	if err != nil {
		t.Fatal(err)
	}
	if len(witnesses) != 2 || first.calls != 1 || second.calls != 1 {
		t.Fatalf("expected both providers to contribute, got %d witnesses, %d and %d calls", len(witnesses), first.calls, second.calls)
	}
	if len(second.requested[0]) != 1 || second.requested[0][0] != secondHash {
		t.Fatalf("second provider received wrong missing signers: %#v", second.requested)
	}
}

func TestEvaluationRejectsUnexpectedWitness(t *testing.T) {
	requiredKey := evaluationKey(4)
	unexpectedKey := evaluationKey(44)
	body := evaluationBody(common.Blake2b224Hash(requiredKey.Public().(ed25519.PublicKey)))
	provider := &evaluationTestProvider{witnesses: []common.VkeyWitness{evaluationWitness(t, unexpectedKey, body)}}

	_, err := New(setupFixedContext()).AddEvaluationWitnessProvider(provider).evaluationWitnesses(&body)
	if err == nil || !strings.Contains(err.Error(), "unexpected") {
		t.Fatalf("expected unexpected witness error, got %v", err)
	}
}

func TestEvaluationRejectsDuplicateWitness(t *testing.T) {
	key := evaluationKey(5)
	hash := common.Blake2b224Hash(key.Public().(ed25519.PublicKey))
	body := evaluationBody(hash)
	witness := evaluationWitness(t, key, body)
	provider := &evaluationTestProvider{witnesses: []common.VkeyWitness{witness, witness}}

	_, err := New(setupFixedContext()).AddEvaluationWitnessProvider(provider).evaluationWitnesses(&body)
	if err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("expected duplicate witness error, got %v", err)
	}
}

func TestEvaluationRejectsMalformedVkey(t *testing.T) {
	key := evaluationKey(6)
	hash := common.Blake2b224Hash(key.Public().(ed25519.PublicKey))
	body := evaluationBody(hash)
	provider := &evaluationTestProvider{witnesses: []common.VkeyWitness{{Vkey: make([]byte, 31), Signature: make([]byte, 64)}}}

	_, err := New(setupFixedContext()).AddEvaluationWitnessProvider(provider).evaluationWitnesses(&body)
	if err == nil || !strings.Contains(err.Error(), "vkey") {
		t.Fatalf("expected malformed vkey error, got %v", err)
	}
}

func TestEvaluationRejectsMalformedSignature(t *testing.T) {
	key := evaluationKey(7)
	hash := common.Blake2b224Hash(key.Public().(ed25519.PublicKey))
	body := evaluationBody(hash)
	provider := &evaluationTestProvider{witnesses: []common.VkeyWitness{{Vkey: key.Public().(ed25519.PublicKey), Signature: make([]byte, 63)}}}

	_, err := New(setupFixedContext()).AddEvaluationWitnessProvider(provider).evaluationWitnesses(&body)
	if err == nil || !strings.Contains(err.Error(), "signature") {
		t.Fatalf("expected malformed signature error, got %v", err)
	}
}

func TestEvaluationRejectsInvalidSignature(t *testing.T) {
	key := evaluationKey(8)
	hash := common.Blake2b224Hash(key.Public().(ed25519.PublicKey))
	body := evaluationBody(hash)
	witness := evaluationWitness(t, key, body)
	witness.Signature[0] ^= 0xff
	provider := &evaluationTestProvider{witnesses: []common.VkeyWitness{witness}}

	_, err := New(setupFixedContext()).AddEvaluationWitnessProvider(provider).evaluationWitnesses(&body)
	if err == nil || !strings.Contains(err.Error(), "signature") {
		t.Fatalf("expected invalid signature error, got %v", err)
	}
}

func TestEvaluationReportsAllMissingSignerHashes(t *testing.T) {
	first := common.Blake2b224Hash(evaluationKey(9).Public().(ed25519.PublicKey))
	second := common.Blake2b224Hash(evaluationKey(10).Public().(ed25519.PublicKey))
	body := evaluationBody(second, first)

	_, err := New(setupFixedContext()).evaluationWitnesses(&body)
	if err == nil || !strings.Contains(err.Error(), first.String()) || !strings.Contains(err.Error(), second.String()) {
		t.Fatalf("expected both missing hashes, got %v", err)
	}
}

func TestEvaluationWitnessOrderDeterministic(t *testing.T) {
	firstKey := evaluationKey(41)
	secondKey := evaluationKey(42)
	firstHash := common.Blake2b224Hash(firstKey.Public().(ed25519.PublicKey))
	secondHash := common.Blake2b224Hash(secondKey.Public().(ed25519.PublicKey))
	body := evaluationBody(secondHash, firstHash)
	provider := &evaluationTestProvider{witnesses: []common.VkeyWitness{
		evaluationWitness(t, firstKey, body),
		evaluationWitness(t, secondKey, body),
	}}

	witnesses, err := New(setupFixedContext()).AddEvaluationWitnessProvider(provider).evaluationWitnesses(&body)
	if err != nil {
		t.Fatal(err)
	}
	first := common.Blake2b224Hash(witnesses[0].Vkey)
	second := common.Blake2b224Hash(witnesses[1].Vkey)
	if string(first[:]) > string(second[:]) {
		t.Fatalf("witnesses were not sorted by key hash: %#v", witnesses)
	}
}

func TestEvaluationNoRequiredSignersDoesNotRequireProvider(t *testing.T) {
	provider := &evaluationTestProvider{err: errors.New("should not be called")}
	witnesses, err := New(setupFixedContext()).AddEvaluationWitnessProvider(provider).evaluationWitnesses(&conway.ConwayTransactionBody{})
	if err != nil {
		t.Fatal(err)
	}
	if len(witnesses) != 0 || provider.calls != 0 {
		t.Fatalf("expected no witnesses or provider calls, got %d and %d", len(witnesses), provider.calls)
	}
}

type captureEvalContext struct {
	*fixed.FixedChainContext
	txCbor []byte
}

func (c *captureEvalContext) EvaluateTx(txCbor []byte, _ []common.Utxo) (map[common.RedeemerKey]common.ExUnits, error) {
	c.txCbor = append([]byte(nil), txCbor...)
	return map[common.RedeemerKey]common.ExUnits{}, nil
}

func TestEvaluationWitnessesSignPostBalanceBody(t *testing.T) {
	key := evaluationKey(51)
	hash := common.Blake2b224Hash(key.Public().(ed25519.PublicKey))
	context := &captureEvalContext{FixedChainContext: setupFixedContext()}
	provider := signingEvaluationProvider{key: key}
	a := New(context).SetWallet(NewExternalWallet(testAddress(t))).AddRequiredSigner(hash).AddEvaluationWitnessProvider(provider)

	if err := a.estimateExecutionUnits(nil, nil); err == nil || !strings.Contains(err.Error(), "no results") {
		t.Fatalf("expected evaluation result error after capture, got %v", err)
	}
	var tx conway.ConwayTransaction
	if _, err := cbor.Decode(context.txCbor, &tx); err != nil {
		t.Fatal(err)
	}
	witnesses := tx.WitnessSet.VkeyWitnesses.Items()
	if len(witnesses) != 1 {
		t.Fatalf("expected captured evaluation witness, got %d", len(witnesses))
	}
	tx.Body.SetCbor(nil)
	bodyCbor, err := cbor.Encode(&tx.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !ed25519.Verify(ed25519.PublicKey(witnesses[0].Vkey), common.Blake2b256Hash(bodyCbor).Bytes(), witnesses[0].Signature) {
		t.Fatal("captured evaluation witness does not sign the encoded evaluation body")
	}
}

func TestCloneCopiesEvaluationProvidersWithoutAliasing(t *testing.T) {
	first := &evaluationTestProvider{}
	second := &evaluationTestProvider{}
	original := New(setupFixedContext()).AddEvaluationWitnessProvider(first)
	clone := original.Clone()
	clone.AddEvaluationWitnessProvider(second)

	if len(original.evaluationWitnessProviders) != 1 || len(clone.evaluationWitnessProviders) != 2 {
		t.Fatalf("provider slices aliased: original %d clone %d", len(original.evaluationWitnessProviders), len(clone.evaluationWitnessProviders))
	}
}

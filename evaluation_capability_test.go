package apollo

import (
	"errors"
	"math/big"
	"strings"
	"testing"

	"github.com/blinklabs-io/gouroboros/ledger/common"
	plutigoData "github.com/blinklabs-io/plutigo/data"

	"github.com/Salvionied/apollo/v2/backend"
	"github.com/Salvionied/apollo/v2/backend/fixed"
)

// capabilityEvalContext is a deterministic evaluation backend whose reported
// capabilities and additionalUtxos handling are configurable, so a build can be
// driven through both sides of the CapabilityEvaluateTxAdditionalUtxos gate.
// It performs no network I/O.
type capabilityEvalContext struct {
	*fixed.FixedChainContext

	// supportsAdditionalUtxos adds CapabilityEvaluateTxAdditionalUtxos to the
	// reported set, as Maestro and Ogmios do.
	supportsAdditionalUtxos bool
	// rejectAdditionalUtxos fails on a non-empty set, as backend/utxorpc does.
	rejectAdditionalUtxos bool

	// result is returned for every accepted call.
	result map[common.RedeemerKey]common.ExUnits

	// received records the additionalUtxos of each call, preserving nil.
	received [][]common.Utxo
}

func (c *capabilityEvalContext) Capabilities() backend.CapabilitySet {
	caps := c.FixedChainContext.Capabilities() |
		backend.CapabilitySet(backend.CapabilityEvaluateTx)
	if c.supportsAdditionalUtxos {
		caps |= backend.CapabilitySet(
			backend.CapabilityEvaluateTxAdditionalUtxos,
		)
	}
	return caps
}

func (c *capabilityEvalContext) EvaluateTx(
	_ []byte,
	additionalUtxos []common.Utxo,
) (map[common.RedeemerKey]common.ExUnits, error) {
	c.received = append(c.received, additionalUtxos)
	if c.rejectAdditionalUtxos && len(additionalUtxos) > 0 {
		return nil, backend.NewUnsupportedError(
			"UTxO RPC", backend.CapabilityEvaluateTxAdditionalUtxos)
	}
	if c.result == nil {
		return nil, errors.New("capabilityEvalContext: result not configured")
	}
	return c.result, nil
}

// setupCapabilityMintBuilder builds a Plutus minting transaction, the smallest
// script build that still forwards resolved spending inputs to the evaluator.
func setupCapabilityMintBuilder(
	t *testing.T,
	cc *capabilityEvalContext,
) *Apollo {
	t.Helper()
	addr := testAddress(t)
	addTestUtxo(cc.FixedChainContext, addr, 50_000_000, 0x31, 0)
	addTestUtxo(cc.FixedChainContext, addr, 20_000_000, 0x32, 0)

	policyHex := strings.Repeat("ab", 28)
	redeemer := testRedeemerDatum()
	p, err := NewPayment(validTestAddrBech32, 2_000_000,
		[]Unit{NewUnit(policyHex, "746f6b656e", 5)})
	if err != nil {
		t.Fatal(err)
	}
	return New(cc).
		SetWallet(NewExternalWallet(addr)).
		AddPayment(p).
		SetTtl(50_000_000).
		Mint(NewUnit(policyHex, "746f6b656e", 5), &redeemer, nil)
}

// TestPlutusBuildSucceedsWhenBackendRejectsAdditionalUtxos is the regression
// test for the release blocker: Apollo forwarded every resolved spending input
// as additionalUtxos, and a backend that rejects a non-empty set (UTxO RPC)
// could therefore not build any Plutus transaction at all.
func TestPlutusBuildSucceedsWhenBackendRejectsAdditionalUtxos(t *testing.T) {
	cc := &capabilityEvalContext{
		FixedChainContext:     setupFixedContext(),
		rejectAdditionalUtxos: true,
		result:                mintRedeemerUnits(1_000, 1_000),
	}
	a, err := setupCapabilityMintBuilder(t, cc).Complete()
	if err != nil {
		t.Fatalf("Complete on additional-UTxO-rejecting backend: %v", err)
	}
	if a.GetTx() == nil {
		t.Fatal("expected a built transaction")
	}
	if len(cc.received) == 0 {
		t.Fatal("expected EvaluateTx to be called")
	}
}

// TestPlutusSpendSucceedsWhenBackendRejectsAdditionalUtxos covers the script
// spend path, where the redeemer index is resolved against the same inputs
// slice that used to be forwarded as additionalUtxos.
func TestPlutusSpendSucceedsWhenBackendRejectsAdditionalUtxos(t *testing.T) {
	cc := &capabilityEvalContext{
		FixedChainContext:     setupFixedContext(),
		rejectAdditionalUtxos: true,
	}
	addr := testAddress(t)
	addTestUtxo(cc.FixedChainContext, addr, 30_000_000, 0x41, 0)

	var scriptHash common.Blake2b256
	scriptHash[0] = 0x42
	scriptUtxo := makeTestUtxo(t, scriptHash, 0, 10_000_000)
	cc.AddUtxo(addr, scriptUtxo)

	// The spend redeemer index follows the canonical input ordering, so resolve
	// it from the sorted inputs rather than assuming index 0.
	inputs := SortInputs([]common.Utxo{scriptUtxo})
	spendIndex := uint32(0)
	for i, utxo := range inputs {
		if utxoRef(utxo) == utxoRef(scriptUtxo) {
			spendIndex = uint32(i) //nolint:gosec // bounded by len(inputs)
		}
	}
	cc.result = map[common.RedeemerKey]common.ExUnits{
		{Tag: common.RedeemerTagSpend, Index: spendIndex}: {
			Memory: 1_000, Steps: 1_000,
		},
	}

	redeemer := common.Datum{Data: plutigoData.NewInteger(big.NewInt(1))}
	p, err := NewPayment(validTestAddrBech32, 2_000_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	a := New(cc).
		SetWallet(NewExternalWallet(addr)).
		AttachScript(common.PlutusV2Script([]byte{0x01, 0x02})).
		CollectFrom(scriptUtxo, redeemer, common.ExUnits{}).
		AddPayment(p).
		SetTtl(50_000_000)

	if _, err := a.Complete(); err != nil {
		t.Fatalf("Complete on additional-UTxO-rejecting backend: %v", err)
	}
	for i, utxos := range cc.received {
		if utxos != nil {
			t.Fatalf("call %d received %d additional UTxOs, want nil",
				i, len(utxos))
		}
	}
}

// TestEvaluationOmitsAdditionalUtxosWhenCapabilityAbsent asserts the gate
// itself: a context that declines the capability must be handed nil, not an
// argument the ChainContext contract permits it to reject.
func TestEvaluationOmitsAdditionalUtxosWhenCapabilityAbsent(t *testing.T) {
	cc := &capabilityEvalContext{
		FixedChainContext: setupFixedContext(),
		result:            mintRedeemerUnits(1_000, 1_000),
	}
	if backend.Supports(cc, backend.CapabilityEvaluateTxAdditionalUtxos) {
		t.Fatal("stub must not report additional-UTxO support")
	}
	if _, err := setupCapabilityMintBuilder(t, cc).Complete(); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if len(cc.received) == 0 {
		t.Fatal("expected EvaluateTx to be called")
	}
	for i, utxos := range cc.received {
		if utxos != nil {
			t.Fatalf("call %d received %d additional UTxOs, want nil",
				i, len(utxos))
		}
	}
}

// TestEvaluationSendsAdditionalUtxosWhenCapabilityPresent is the other half of
// the gate: a reporting context must still receive the full resolved input set,
// which is what off-chain and chained inputs depend on.
func TestEvaluationSendsAdditionalUtxosWhenCapabilityPresent(t *testing.T) {
	cc := &capabilityEvalContext{
		FixedChainContext:       setupFixedContext(),
		supportsAdditionalUtxos: true,
		result:                  mintRedeemerUnits(1_000, 1_000),
	}
	a, err := setupCapabilityMintBuilder(t, cc).Complete()
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if len(cc.received) == 0 {
		t.Fatal("expected EvaluateTx to be called")
	}
	wanted := a.GetTx().Body.Inputs()
	for i, utxos := range cc.received {
		if len(utxos) != len(wanted) {
			t.Fatalf("call %d received %d additional UTxOs, want %d",
				i, len(utxos), len(wanted))
		}
		refs := make(map[string]bool, len(utxos))
		for _, utxo := range utxos {
			refs[utxoRef(utxo)] = true
		}
		for _, in := range wanted {
			ref := utxoRef(common.Utxo{Id: in})
			if !refs[ref] {
				t.Fatalf("call %d omitted resolved input %s", i, ref)
			}
		}
	}
}

// unreportedEvalContext embeds the ChainContext interface rather than the stub
// type, so the stub's Capabilities method is not promoted and the wrapper does
// not satisfy backend.CapabilityReporter. It stands in for a third-party
// backend written before capability reporting existed.
type unreportedEvalContext struct {
	backend.ChainContext
	inner *capabilityEvalContext
}

// TestEvaluationSendsAdditionalUtxosForUnreportedContext pins the documented
// fallback for third-party backends: backend.CapabilitiesOf reports
// AllCapabilities for a ChainContext that does not implement
// CapabilityReporter, so such a context keeps the historic behavior of
// receiving the resolved input set.
func TestEvaluationSendsAdditionalUtxosForUnreportedContext(t *testing.T) {
	stub := &capabilityEvalContext{
		FixedChainContext: setupFixedContext(),
		result:            mintRedeemerUnits(1_000, 1_000),
	}
	cc := &unreportedEvalContext{ChainContext: stub, inner: stub}
	if _, ok := backend.ChainContext(cc).(backend.CapabilityReporter); ok {
		t.Fatal("wrapper must not satisfy backend.CapabilityReporter")
	}
	if !backend.Supports(cc, backend.CapabilityEvaluateTxAdditionalUtxos) {
		t.Fatal("a context without CapabilityReporter must report everything")
	}

	addr := testAddress(t)
	addTestUtxo(stub.FixedChainContext, addr, 50_000_000, 0x51, 0)
	policyHex := strings.Repeat("ab", 28)
	redeemer := testRedeemerDatum()
	p, err := NewPayment(validTestAddrBech32, 2_000_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	a := New(cc).
		SetWallet(NewExternalWallet(addr)).
		AddPayment(p).
		SetTtl(50_000_000).
		Mint(NewUnit(policyHex, "746f6b656e", 1), &redeemer, nil)
	if _, err := a.Complete(); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if len(cc.inner.received) == 0 {
		t.Fatal("expected EvaluateTx to be called")
	}
	for i, utxos := range cc.inner.received {
		if len(utxos) == 0 {
			t.Fatalf("call %d received no additional UTxOs", i)
		}
	}
}

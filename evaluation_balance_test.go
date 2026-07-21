package apollo

import (
	"math"
	"math/big"
	"testing"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/common"
)

func evaluationAsset(t *testing.T, quantity int64) *common.MultiAsset[common.MultiAssetTypeOutput] {
	t.Helper()
	var policy common.Blake2b224
	policy[0] = 1
	assets := common.NewMultiAsset[common.MultiAssetTypeOutput](
		map[common.Blake2b224]map[cbor.ByteString]common.MultiAssetTypeOutput{
			policy: {cbor.NewByteString([]byte("token")): big.NewInt(quantity)},
		},
	)
	return &assets
}

func TestBalancedOutputsAbsorbsAdaDustIntoFee(t *testing.T) {
	a := New(setupFixedContext())
	got, err := a.buildBalancedOutputs(nil, 2_000_000, balanceContext{
		totalInput:    NewSimpleValue(2_000_100),
		totalRequired: Value{},
		changeAddress: testAddress(t),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Outputs) != 0 {
		t.Fatalf("expected no dust change output, got %d outputs", len(got.Outputs))
	}
	if got.Fee != 2_000_100 {
		t.Fatalf("fee = %d, want dust absorbed into 2000100", got.Fee)
	}
}

func TestBalancedOutputsNeverDropsAssetChange(t *testing.T) {
	a := New(setupFixedContext())
	got, err := a.buildBalancedOutputs(nil, 2_000_000, balanceContext{
		totalInput:    NewValue(5_000_000, evaluationAsset(t, 1)),
		totalRequired: Value{},
		changeAddress: testAddress(t),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Outputs) != 1 || !ValueFromMaryValue(got.Outputs[0].OutputAmount).HasAssets() {
		t.Fatal("asset-bearing change was not retained")
	}
}

func TestBalancedOutputsRejectsNegativeFee(t *testing.T) {
	a := New(setupFixedContext())
	_, err := a.buildBalancedOutputs(nil, -1, balanceContext{changeAddress: testAddress(t)})
	if err == nil {
		t.Fatal("expected negative fee error")
	}
}

func TestBalancedOutputsRejectsOverflow(t *testing.T) {
	a := New(setupFixedContext())
	_, err := a.buildBalancedOutputs(nil, 1, balanceContext{
		totalInput:    NewSimpleValue(math.MaxUint64),
		totalRequired: NewSimpleValue(math.MaxUint64),
		changeAddress: testAddress(t),
	})
	if err == nil {
		t.Fatal("expected required value overflow")
	}
}

func TestBalancedOutputsIncludesWithdrawal(t *testing.T) {
	// Withdrawals are modeled as additional totalInput before balancing.
	a := New(setupFixedContext())
	const withdrawal = uint64(2_000_000)
	got, err := a.buildBalancedOutputs(nil, 2_000_000, balanceContext{
		totalInput:    NewSimpleValue(3_000_000 + withdrawal),
		totalRequired: NewSimpleValue(1_000_000),
		changeAddress: testAddress(t),
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Fee != 2_000_000 || len(got.Outputs) != 1 || got.Outputs[0].OutputAmount.Amount != 2_000_000 {
		t.Fatalf("withdrawal-derived input did not become balanced change: %#v", got)
	}
}

func TestBalancedOutputsIncludesCertificateDeposit(t *testing.T) {
	// Certificate deposits are part of totalRequired (as Complete() does).
	a := New(setupFixedContext())
	const payment, fee, deposit = uint64(1_000_000), int64(2_000_000), uint64(StakeDeposit)
	got, err := a.buildBalancedOutputs(nil, fee, balanceContext{
		totalInput:    NewSimpleValue(10_000_000),
		totalRequired: NewSimpleValue(payment + deposit),
		stakeDeposit:  StakeDeposit,
		changeAddress: testAddress(t),
	})
	if err != nil {
		t.Fatal(err)
	}
	wantChange := 10_000_000 - payment - deposit - uint64(fee)
	if len(got.Outputs) != 1 || got.Outputs[0].OutputAmount.Amount != wantChange {
		t.Fatalf("deposit was not reserved from change: got %#v, want change %d", got, wantChange)
	}
}

func TestBalancedOutputsIncludesCertificateRefund(t *testing.T) {
	// Deregistration refunds are part of totalInput (as Complete() does).
	a := New(setupFixedContext())
	const payment, fee, refund = uint64(1_000_000), int64(2_000_000), uint64(StakeDeposit)
	got, err := a.buildBalancedOutputs(nil, fee, balanceContext{
		totalInput:    NewSimpleValue(5_000_000 + refund),
		totalRequired: NewSimpleValue(payment),
		stakeDeposit:  StakeDeposit,
		changeAddress: testAddress(t),
	})
	if err != nil {
		t.Fatal(err)
	}
	wantChange := 5_000_000 + refund - payment - uint64(fee)
	if len(got.Outputs) != 1 || got.Outputs[0].OutputAmount.Amount != wantChange {
		t.Fatalf("refund was not added to change: got %#v, want change %d", got, wantChange)
	}
}

func TestBalancedOutputsIncludesGovernanceValues(t *testing.T) {
	// Governance must be applied via governanceRequired, not double-counted in
	// totalRequired (Complete keeps those fields separate).
	a := New(setupFixedContext())
	const payment, fee, donation = uint64(1_000_000), int64(2_000_000), uint64(5_000_000)
	got, err := a.buildBalancedOutputs(nil, fee, balanceContext{
		totalInput:         NewSimpleValue(20_000_000),
		totalRequired:      NewSimpleValue(payment),
		governanceRequired: NewSimpleValue(donation),
		changeAddress:      testAddress(t),
	})
	if err != nil {
		t.Fatal(err)
	}
	wantChange := 20_000_000 - payment - donation - uint64(fee)
	if len(got.Outputs) != 1 || got.Outputs[0].OutputAmount.Amount != wantChange {
		t.Fatalf("governance was not reserved from change: got %#v, want change %d", got, wantChange)
	}

	// If governance were also folded into totalRequired, change would under-count.
	doubleCounted, err := a.buildBalancedOutputs(nil, fee, balanceContext{
		totalInput:         NewSimpleValue(20_000_000),
		totalRequired:      NewSimpleValue(payment + donation),
		governanceRequired: NewSimpleValue(donation),
		changeAddress:      testAddress(t),
	})
	if err != nil {
		t.Fatal(err)
	}
	if doubleCounted.Outputs[0].OutputAmount.Amount == wantChange {
		t.Fatal("governance double-count check failed: separate fields must change the result")
	}
}

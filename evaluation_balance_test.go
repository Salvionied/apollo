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
	a := New(setupFixedContext())
	got, err := a.buildBalancedOutputs(nil, 2_000_000, balanceContext{
		totalInput:    NewSimpleValue(5_000_000),
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

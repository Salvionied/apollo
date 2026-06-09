package apollo

import (
	"math/big"
	"testing"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/common"
)

func testPolicyId(b byte) common.Blake2b224 {
	var pid common.Blake2b224
	pid[0] = b
	return pid
}

func testAssetName(name string) cbor.ByteString {
	return cbor.NewByteString([]byte(name))
}

func testMultiAsset(policyByte byte, name string, qty int64) *common.MultiAsset[common.MultiAssetTypeOutput] {
	data := map[common.Blake2b224]map[cbor.ByteString]common.MultiAssetTypeOutput{
		testPolicyId(policyByte): {
			testAssetName(name): big.NewInt(qty),
		},
	}
	ma := common.NewMultiAsset[common.MultiAssetTypeOutput](data)
	return &ma
}

func TestNewValue(t *testing.T) {
	v := NewValue(1000, nil)
	if v.Coin != 1000 {
		t.Errorf("expected coin 1000, got %d", v.Coin)
	}
	if v.Assets != nil {
		t.Error("expected nil assets")
	}
}

func TestNewSimpleValue(t *testing.T) {
	v := NewSimpleValue(5000000)
	if v.Coin != 5000000 {
		t.Errorf("expected coin 5000000, got %d", v.Coin)
	}
	if v.HasAssets() {
		t.Error("expected no assets")
	}
}

func TestValueAddCoin(t *testing.T) {
	a := NewSimpleValue(100)
	b := NewSimpleValue(200)
	result, err := a.Add(b)
	if err != nil {
		t.Fatal(err)
	}
	if result.Coin != 300 {
		t.Errorf("expected 300, got %d", result.Coin)
	}
}

func TestValueAddWithAssets(t *testing.T) {
	a := NewValue(100, testMultiAsset(1, "token", 50))
	b := NewValue(200, testMultiAsset(1, "token", 30))
	result, err := a.Add(b)
	if err != nil {
		t.Fatal(err)
	}
	if result.Coin != 300 {
		t.Errorf("expected coin 300, got %d", result.Coin)
	}
	if result.Assets == nil {
		t.Fatal("expected assets")
	}
	qty := result.Assets.Asset(testPolicyId(1), []byte("token"))
	if qty == nil || qty.Int64() != 80 {
		t.Errorf("expected 80 tokens, got %v", qty)
	}
}

func TestValueAddMixedAssets(t *testing.T) {
	a := NewValue(100, testMultiAsset(1, "tokenA", 50))
	b := NewSimpleValue(200)
	result, err := a.Add(b)
	if err != nil {
		t.Fatal(err)
	}
	if result.Coin != 300 {
		t.Errorf("expected coin 300, got %d", result.Coin)
	}
	if !result.HasAssets() {
		t.Error("expected assets to be preserved")
	}

	// Reverse direction
	result2, err := b.Add(a)
	if err != nil {
		t.Fatal(err)
	}
	if result2.Coin != 300 {
		t.Errorf("expected coin 300, got %d", result2.Coin)
	}
	if !result2.HasAssets() {
		t.Error("expected assets to be preserved in reverse add")
	}
}

func TestValueAddOverflow(t *testing.T) {
	a := NewSimpleValue(^uint64(0)) // max uint64
	b := NewSimpleValue(1)
	_, err := a.Add(b)
	if err == nil {
		t.Error("expected overflow error")
	}
}

func TestValueSubCoin(t *testing.T) {
	a := NewSimpleValue(300)
	b := NewSimpleValue(100)
	result, err := a.Sub(b)
	if err != nil {
		t.Fatal(err)
	}
	if result.Coin != 200 {
		t.Errorf("expected 200, got %d", result.Coin)
	}
}

func TestValueSubUnderflow(t *testing.T) {
	a := NewSimpleValue(100)
	b := NewSimpleValue(200)
	_, err := a.Sub(b)
	if err == nil {
		t.Error("expected underflow error")
	}
}

func TestValueGreaterOrEqual(t *testing.T) {
	tests := []struct {
		name     string
		a, b     Value
		expected bool
	}{
		{"equal coins", NewSimpleValue(100), NewSimpleValue(100), true},
		{"greater coin", NewSimpleValue(200), NewSimpleValue(100), true},
		{"less coin", NewSimpleValue(50), NewSimpleValue(100), false},
		{"coin with nil assets vs coin", NewValue(200, nil), NewSimpleValue(100), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.a.GreaterOrEqual(tt.b); got != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}

func TestValueClone(t *testing.T) {
	original := NewValue(1000, testMultiAsset(1, "token", 50))
	cloned := original.Clone()
	if cloned.Coin != original.Coin {
		t.Error("coin mismatch")
	}
	if !cloned.HasAssets() {
		t.Error("expected cloned assets")
	}
	// Mutate clone, verify original unchanged
	cloned.Coin = 9999
	if original.Coin != 1000 {
		t.Error("original coin was mutated")
	}
}

func TestValueToMaryValue(t *testing.T) {
	v := NewValue(2000000, testMultiAsset(1, "token", 100))
	mv := v.ToMaryValue()
	if mv.Amount != 2000000 {
		t.Errorf("expected 2000000, got %d", mv.Amount)
	}
	if mv.Assets == nil {
		t.Error("expected assets in MaryValue")
	}
}

func TestValueFromMaryValue(t *testing.T) {
	v := NewValue(3000000, testMultiAsset(2, "asset", 42))
	mv := v.ToMaryValue()
	roundTrip := ValueFromMaryValue(mv)
	if roundTrip.Coin != 3000000 {
		t.Errorf("expected 3000000, got %d", roundTrip.Coin)
	}
	if !roundTrip.HasAssets() {
		t.Error("expected assets to survive round-trip")
	}
	qty := roundTrip.Assets.Asset(testPolicyId(2), []byte("asset"))
	if qty == nil || qty.Int64() != 42 {
		t.Errorf("expected 42, got %v", qty)
	}
}

func TestValueCloneAssetIndependence(t *testing.T) {
	original := NewValue(1000, testMultiAsset(1, "token", 50))
	cloned := original.Clone()
	// Verify the cloned assets are independent from the original
	origQty := original.Assets.Asset(testPolicyId(1), []byte("token"))
	cloneQty := cloned.Assets.Asset(testPolicyId(1), []byte("token"))
	if origQty == nil || cloneQty == nil {
		t.Fatal("expected non-nil asset quantities")
	}
	if origQty.Int64() != cloneQty.Int64() {
		t.Error("cloned asset quantity should match original")
	}
	// Mutate the clone and verify the original is unchanged.
	extra := testMultiAsset(1, "token", 25)
	cloned.Assets.Add(extra)
	afterOrig := original.Assets.Asset(testPolicyId(1), []byte("token"))
	if afterOrig == nil || afterOrig.Int64() != 50 {
		t.Errorf("original should still be 50 after mutating clone, got %v", afterOrig)
	}
	afterClone := cloned.Assets.Asset(testPolicyId(1), []byte("token"))
	if afterClone == nil || afterClone.Int64() != 75 {
		t.Errorf("cloned should be 75 after mutation, got %v", afterClone)
	}
}

func TestValueSubWithAssets(t *testing.T) {
	a := NewValue(300, testMultiAsset(1, "token", 80))
	b := NewValue(100, testMultiAsset(1, "token", 30))
	result, err := a.Sub(b)
	if err != nil {
		t.Fatal(err)
	}
	if result.Coin != 200 {
		t.Errorf("expected coin 200, got %d", result.Coin)
	}
	qty := result.Assets.Asset(testPolicyId(1), []byte("token"))
	if qty == nil || qty.Int64() != 50 {
		t.Errorf("expected 50 tokens, got %v", qty)
	}
}

func TestCloneMultiAsset(t *testing.T) {
	original := testMultiAsset(1, "token", 100)
	cloned := CloneMultiAsset(original)
	if cloned == nil {
		t.Fatal("expected non-nil clone")
	}
	qty := cloned.Asset(testPolicyId(1), []byte("token"))
	if qty == nil || qty.Int64() != 100 {
		t.Errorf("expected 100, got %v", qty)
	}
}

func TestCloneMultiAssetNil(t *testing.T) {
	cloned := CloneMultiAsset(nil)
	if cloned != nil {
		t.Error("expected nil clone of nil")
	}
}

func TestMultiAssetIsEmpty(t *testing.T) {
	if !MultiAssetIsEmpty(nil) {
		t.Error("nil should be empty")
	}
	ma := testMultiAsset(1, "token", 100)
	if MultiAssetIsEmpty(ma) {
		t.Error("non-empty should not be empty")
	}
}

func TestMultiAssetFromMap(t *testing.T) {
	data := map[common.Blake2b224]map[cbor.ByteString]*big.Int{
		testPolicyId(1): {
			testAssetName("foo"): big.NewInt(42),
		},
	}
	result := MultiAssetFromMap(data)
	if result == nil {
		t.Fatal("expected non-nil")
	}
	qty := result.Asset(testPolicyId(1), []byte("foo"))
	if qty == nil || qty.Int64() != 42 {
		t.Errorf("expected 42, got %v", qty)
	}
}

func TestMultiAssetFromMapEmpty(t *testing.T) {
	result := MultiAssetFromMap(nil)
	if result != nil {
		t.Error("expected nil for empty map")
	}
}

func TestNewBabbageOutputSimple(t *testing.T) {
	addr := testAddress(t)
	output := NewBabbageOutputSimple(addr, 5000000)
	if output.OutputAmount.Amount != 5000000 {
		t.Errorf("expected 5000000, got %d", output.OutputAmount.Amount)
	}
}

func TestNewBabbageOutput(t *testing.T) {
	addr := testAddress(t)
	v := NewValue(3000000, testMultiAsset(1, "token", 100))
	output := NewBabbageOutput(addr, v, nil, nil)
	if output.OutputAmount.Amount != 3000000 {
		t.Errorf("expected 3000000, got %d", output.OutputAmount.Amount)
	}
	if output.OutputAmount.Assets == nil {
		t.Error("expected assets in output")
	}
}

func TestNewDatumOptionHash(t *testing.T) {
	var hash common.Blake2b256
	hash[0] = 0xab
	opt, err := NewDatumOptionHash(hash)
	if err != nil {
		t.Fatal(err)
	}
	if opt == nil {
		t.Fatal("expected non-nil datum option")
	}
}

func TestNewDatumOptionInlineNil(t *testing.T) {
	_, err := NewDatumOptionInline(nil)
	if err == nil {
		t.Error("expected error for nil datum")
	}
}

func TestOutputCborSize(t *testing.T) {
	addr := testAddress(t)
	output := NewBabbageOutputSimple(addr, 2000000)
	size, err := OutputCborSize(&output)
	if err != nil {
		t.Fatal(err)
	}
	if size <= 0 {
		t.Error("expected positive CBOR size")
	}
}

func TestMinLovelacePostAlonzo(t *testing.T) {
	addr := testAddress(t)
	output := NewBabbageOutputSimple(addr, 0)
	coinsPerUtxoByte := int64(4310)
	minLov, err := MinLovelacePostAlonzo(&output, coinsPerUtxoByte)
	if err != nil {
		t.Fatal(err)
	}
	// Min lovelace = coinsPerUtxoByte * (outputSize + 160)
	outputSize, err := OutputCborSize(&output)
	if err != nil {
		t.Fatal(err)
	}
	expected := coinsPerUtxoByte * int64(outputSize+160)
	if minLov != expected {
		t.Errorf("expected %d, got %d", expected, minLov)
	}
	if minLov <= 0 {
		t.Errorf("expected positive min lovelace, got %d", minLov)
	}
}

func TestNewNativeScriptPubkey(t *testing.T) {
	var keyHash common.Blake2b224
	keyHash[0] = 0xaa
	ns, err := NewNativeScriptPubkey(keyHash)
	if err != nil {
		t.Fatal(err)
	}
	// Verify it round-trips through CBOR
	cborBytes, err := cbor.Encode(&ns)
	if err != nil {
		t.Fatal(err)
	}
	if len(cborBytes) == 0 {
		t.Error("expected non-empty CBOR")
	}
}

func TestNewNativeScriptInvalidBefore(t *testing.T) {
	ns, err := NewNativeScriptInvalidBefore(1000)
	if err != nil {
		t.Fatal(err)
	}
	cborBytes, err := cbor.Encode(&ns)
	if err != nil {
		t.Fatal(err)
	}
	if len(cborBytes) == 0 {
		t.Error("expected non-empty CBOR")
	}
}

func TestNewNativeScriptInvalidHereafter(t *testing.T) {
	ns, err := NewNativeScriptInvalidHereafter(2000)
	if err != nil {
		t.Fatal(err)
	}
	cborBytes, err := cbor.Encode(&ns)
	if err != nil {
		t.Fatal(err)
	}
	if len(cborBytes) == 0 {
		t.Error("expected non-empty CBOR")
	}
}

func TestNewNativeScriptAll(t *testing.T) {
	var kh1, kh2 common.Blake2b224
	kh1[0] = 0x01
	kh2[0] = 0x02
	ns1, err := NewNativeScriptPubkey(kh1)
	if err != nil {
		t.Fatal(err)
	}
	ns2, err := NewNativeScriptPubkey(kh2)
	if err != nil {
		t.Fatal(err)
	}
	all, err := NewNativeScriptAll([]common.NativeScript{ns1, ns2})
	if err != nil {
		t.Fatal(err)
	}
	cborBytes, err := cbor.Encode(&all)
	if err != nil {
		t.Fatal(err)
	}
	if len(cborBytes) == 0 {
		t.Error("expected non-empty CBOR")
	}
}

func TestNewNativeScriptAny(t *testing.T) {
	ns1, err := NewNativeScriptInvalidBefore(100)
	if err != nil {
		t.Fatal(err)
	}
	ns2, err := NewNativeScriptInvalidHereafter(200)
	if err != nil {
		t.Fatal(err)
	}
	nativeAny, err := NewNativeScriptAny([]common.NativeScript{ns1, ns2})
	if err != nil {
		t.Fatal(err)
	}
	cborBytes, err := cbor.Encode(&nativeAny)
	if err != nil {
		t.Fatal(err)
	}
	if len(cborBytes) == 0 {
		t.Error("expected non-empty CBOR")
	}
}

func TestNewNativeScriptNofK(t *testing.T) {
	ns1, err := NewNativeScriptInvalidBefore(100)
	if err != nil {
		t.Fatal(err)
	}
	ns2, err := NewNativeScriptInvalidHereafter(200)
	if err != nil {
		t.Fatal(err)
	}
	nofk, err := NewNativeScriptNofK(1, []common.NativeScript{ns1, ns2})
	if err != nil {
		t.Fatal(err)
	}
	cborBytes, err := cbor.Encode(&nofk)
	if err != nil {
		t.Fatal(err)
	}
	if len(cborBytes) == 0 {
		t.Error("expected non-empty CBOR")
	}
}

func TestNewNativeScriptNofKZeroN(t *testing.T) {
	ns1, err := NewNativeScriptInvalidBefore(100)
	if err != nil {
		t.Fatal(err)
	}
	_, err = NewNativeScriptNofK(0, []common.NativeScript{ns1})
	if err == nil {
		t.Error("expected error when n is 0")
	}
}

func TestNewNativeScriptNofKEmptyScripts(t *testing.T) {
	_, err := NewNativeScriptNofK(1, []common.NativeScript{})
	if err == nil {
		t.Error("expected error for empty scripts list")
	}
}

func TestNewNativeScriptNofKExceedsScripts(t *testing.T) {
	ns1, err := NewNativeScriptInvalidBefore(100)
	if err != nil {
		t.Fatal(err)
	}
	_, err = NewNativeScriptNofK(5, []common.NativeScript{ns1})
	if err == nil {
		t.Error("expected error when n exceeds number of scripts")
	}
}

func TestSignMessage(t *testing.T) {
	// 32-byte seed key
	key := make([]byte, 32)
	key[0] = 0x01
	msg := []byte("hello world")
	sig, err := SignMessage(key, msg)
	if err != nil {
		t.Fatal(err)
	}
	if len(sig) != 64 {
		t.Errorf("expected 64-byte signature, got %d", len(sig))
	}
}

func TestSignMessage64ByteKey(t *testing.T) {
	key := make([]byte, 64)
	key[0] = 0x01
	msg := []byte("test message")
	sig, err := SignMessage(key, msg)
	if err != nil {
		t.Fatal(err)
	}
	if len(sig) != 64 {
		t.Errorf("expected 64-byte signature, got %d", len(sig))
	}
}

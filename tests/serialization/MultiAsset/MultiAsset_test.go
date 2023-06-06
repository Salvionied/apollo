package serialization_test

import (
	"testing"

	"github.com/Salvionied/apollo/serialization/Asset"
	"github.com/Salvionied/apollo/serialization/AssetName"
	"github.com/Salvionied/apollo/serialization/MultiAsset"
	"github.com/Salvionied/apollo/serialization/Policy"
	"github.com/Salvionied/apollo/serialization/Value"
)

func TestMultiAssetAddition(t *testing.T) {
	policy_id := Policy.PolicyId{Value: "ec8b7d1dd0b124e8333d3fa8d818f6eac068231a287554e9ceae490e"}
	policy_id2 := Policy.PolicyId{Value: "ec8b7d1dd0b124e8333d3fa8d818f6eac068231a287554e9ceae490f"}
	asset_name1 := AssetName.NewAssetNameFromString("token1")
	asset_name2 := AssetName.NewAssetNameFromString("token2")

	a := MultiAsset.MultiAsset[int64]{
		policy_id: Asset.Asset[int64]{asset_name1: 1, asset_name2: 2},
	}
	a_clone := MultiAsset.MultiAsset[int64]{
		policy_id: Asset.Asset[int64]{asset_name1: 1, asset_name2: 2},
	}

	b := MultiAsset.MultiAsset[int64]{
		policy_id:  Asset.Asset[int64]{asset_name1: 10, asset_name2: 20},
		policy_id2: Asset.Asset[int64]{asset_name1: 1, asset_name2: 2}}
	b_clone := MultiAsset.MultiAsset[int64]{
		policy_id:  Asset.Asset[int64]{asset_name1: 10, asset_name2: 20},
		policy_id2: Asset.Asset[int64]{asset_name1: 1, asset_name2: 2}}
	c := a.Add(b)
	if c[policy_id][asset_name1] != 11 {
		t.Errorf("Expected 11, got %d", c[policy_id][asset_name1])
	}
	if c[policy_id][asset_name2] != 22 {
		t.Errorf("Expected 22, got %d", c[policy_id][asset_name2])
	}
	if c[policy_id2][asset_name1] != 1 {
		t.Errorf("Expected 1, got %d", c[policy_id2][asset_name1])
	}
	if c[policy_id2][asset_name2] != 2 {
		t.Errorf("Expected 2, got %d", c[policy_id2][asset_name2])
	}
	if !a.Equal(a_clone) {
		t.Errorf("Expected true, got false")
	}

	if a.Equal(b) {
		t.Errorf("Expected false, got true")
	}
	if !b.Equal(b_clone) {
		t.Errorf("Expected true, got false")
	}

	d := b.Sub(a)
	if d[policy_id][asset_name1] != 9 {
		t.Errorf("Expected 9, got %d", d[policy_id][asset_name1])
	}
	if d[policy_id][asset_name2] != 18 {
		t.Errorf("Expected 18, got %d", d[policy_id][asset_name2])
	}
	if d[policy_id2][asset_name1] != 1 {
		t.Errorf("Expected 1, got %d", d[policy_id2][asset_name1])
	}
	if d[policy_id2][asset_name2] != 2 {
		t.Errorf("Expected 2, got %d", d[policy_id2][asset_name2])
	}
	if !a.Equal(a_clone) {
		t.Errorf("Expected true, got false")
	}
	if !b.Equal(b_clone) {
		t.Errorf("Expected true, got false")
	}

}

func TestMultiAssetComparison(t *testing.T) {
	policy_id := Policy.PolicyId{Value: "ec8b7d1dd0b124e8333d3fa8d818f6eac068231a287554e9ceae490e"}
	policy_id2 := Policy.PolicyId{Value: "ec8b7d1dd0b124e8333d3fa8d818f6eac068231a287554e9ceae490f"}
	asset_name1 := AssetName.NewAssetNameFromString("token1")
	asset_name2 := AssetName.NewAssetNameFromString("token2")
	asset_name3 := AssetName.NewAssetNameFromString("token3")

	a := MultiAsset.MultiAsset[int64]{
		policy_id: Asset.Asset[int64]{asset_name1: 1, asset_name2: 2},
	}

	b := MultiAsset.MultiAsset[int64]{
		policy_id: Asset.Asset[int64]{asset_name1: 1, asset_name2: 2, asset_name3: 3},
	}

	c := MultiAsset.MultiAsset[int64]{
		policy_id:  Asset.Asset[int64]{asset_name1: 1, asset_name2: 3},
		policy_id2: Asset.Asset[int64]{asset_name1: 1, asset_name2: 2}}

	d := MultiAsset.MultiAsset[int64]{
		policy_id2: Asset.Asset[int64]{asset_name1: 1, asset_name2: 2}}

	if a.Equal(b) {
		t.Errorf("Expected false, got true")
	}
	if !(a.Greater(b) || a.Equal(b)) {
		t.Errorf("Expected true, got false")
	}
	if b.Greater(a) || b.Equal(a) {
		t.Errorf("Expected false, got true")
	}

	if a.Equal(c) {
		t.Errorf("Expected false, got true")
	}

	if !(a.Less(c) || a.Equal(c)) {
		t.Errorf("Expected true, got false")
	}

	if c.Less(a) || c.Equal(a) {
		t.Errorf("Expected false, got true")
	}

	if a.Equal(d) {
		t.Errorf("Expected false, got true")
	}

	if a.Less(d) || a.Equal(d) {
		t.Errorf("Expected false, got true")
	}

	if d.Less(a) || d.Equal(a) {
		t.Errorf("Expected false, got true")
	}

}

func TestValues(t *testing.T) {
	policy_id := Policy.PolicyId{Value: "ec8b7d1dd0b124e8333d3fa8d818f6eac068231a287554e9ceae490e"}
	policy_id2 := Policy.PolicyId{Value: "ec8b7d1dd0b124e8333d3fa8d818f6eac068231a287554e9ceae490f"}
	asset_name1 := AssetName.NewAssetNameFromString("token1")
	asset_name2 := AssetName.NewAssetNameFromString("token2")

	multiasset1 := MultiAsset.MultiAsset[int64]{
		policy_id: Asset.Asset[int64]{asset_name1: 1, asset_name2: 2},
	}

	multiasset2 := MultiAsset.MultiAsset[int64]{
		policy_id: Asset.Asset[int64]{asset_name1: 11, asset_name2: 22}}

	multiasset3 := MultiAsset.MultiAsset[int64]{
		policy_id:  Asset.Asset[int64]{asset_name1: 11, asset_name2: 22},
		policy_id2: Asset.Asset[int64]{asset_name1: 11, asset_name2: 22}}

	a := Value.SimpleValue(int64(1), multiasset1)

	b := Value.SimpleValue(int64(11), multiasset2)

	c := Value.SimpleValue(int64(11), multiasset3)

	if a.Equal(b) {
		t.Errorf("Expected false, got true")
	}

	if a.GreaterOrEqual(b) {
		t.Errorf("Expected true, got false")
	}

	if b.Less(a) || b.Equal(a) {
		t.Errorf("Expected false, got true")
	}

	if c.LessOrEqual(a) {
		t.Errorf("Expected true, got false")
	}

	if a.Greater(c) || a.Equal(c) {
		t.Errorf("Expected false, got true")
	}

	if !b.GreaterOrEqual(c) {
		t.Errorf("Expected true, got false")
	}

	if c.LessOrEqual(b) {
		t.Errorf("Expected false, got true")
	}

	if !a.Equal(a) {
		t.Errorf("Expected true, got false")
	}

	e := b.Sub(a)
	if e.GetCoin() != 10 {
		t.Errorf("Expected 10, got %d", e.GetCoin())
	}
	if e.GetAssets()[policy_id][asset_name1] != 10 {
		t.Errorf("Expected 10, got %d", e.GetAssets()[policy_id][asset_name1])
	}
	if e.GetAssets()[policy_id][asset_name2] != 20 {
		t.Errorf("Expected 20, got %d", e.GetAssets()[policy_id][asset_name2])
	}

	f := c.Sub(a)

	if f.GetCoin() != 10 {
		t.Errorf("Expected 10, got %d", f.GetCoin())
	}
	if f.GetAssets()[policy_id][asset_name1] != 10 {
		t.Errorf("Expected 10, got %d", f.GetAssets()[policy_id][asset_name1])
	}
	if f.GetAssets()[policy_id][asset_name2] != 20 {
		t.Errorf("Expected 20, got %d", f.GetAssets()[policy_id][asset_name2])
	}
	if f.GetAssets()[policy_id2][asset_name1] != 11 {
		t.Errorf("Expected 10, got %d", f.GetAssets()[policy_id2][asset_name1])
	}
	if f.GetAssets()[policy_id2][asset_name2] != 22 {
		t.Errorf("Expected 20, got %d", f.GetAssets()[policy_id2][asset_name2])
	}

	g := a.Add(Value.PureLovelaceValue(100))
	if g.GetCoin() != 101 {
		t.Errorf("Expected 101, got %d", g.GetCoin())
	}

	if !a.Equal(Value.SimpleValue(int64(1), multiasset1)) {
		t.Errorf("Expected true, got false")
	}

}

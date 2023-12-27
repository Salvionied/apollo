package Asset_test

import (
	"testing"

	"github.com/Salvionied/apollo/serialization/Asset"
	"github.com/Salvionied/apollo/serialization/AssetName"
)

var assetNamet1 = AssetName.NewAssetNameFromString("test")
var assetNamet2 = AssetName.NewAssetNameFromString("test2")

func TestEquality(t *testing.T) {
	val0 := Asset.Asset[int64]{
		assetNamet1: 100,
	}
	val1 := Asset.Asset[int64]{
		assetNamet1: 100,
	}
	if !val0.Equal(val1) {
		t.Errorf("Amounts should be equal")
	}
	val2 := Asset.Asset[int64]{
		assetNamet1: 99,
	}
	if val0.Equal(val2) {
		t.Errorf("Amounts should not be equal")
	}
	val3 := Asset.Asset[int64]{
		assetNamet1: 100,
		assetNamet2: 100,
	}
	if val0.Equal(val3) {
		t.Errorf("Amounts should not be equal")
	}
}

func TestClone(t *testing.T) {
	val0 := Asset.Asset[int64]{
		assetNamet1: 100,
	}
	val1 := val0.Clone()
	if !val0.Equal(val1) {
		t.Errorf("Amounts should be equal")
	}
	if &val0 == &val1 {
		t.Errorf("Amounts should not be equal")
	}
}

func TestLess(t *testing.T) {
	val0 := Asset.Asset[int64]{
		assetNamet1: 50,
	}
	val1 := Asset.Asset[int64]{
		assetNamet1: 100,
	}
	if !val0.Less(val1) {
		t.Errorf("Amounts should be equal")
	}
	if val1.Less(val0) {
		t.Errorf("Amounts should not be equal")
	}
	val2 := Asset.Asset[int64]{}
	if !val2.Less(val1) {
		t.Errorf("Amounts should be equal")
	}
	if val1.Less(val2) {
		t.Errorf("Amounts should not be equal")
	}
}

func TestGreater(t *testing.T) {
	val0 := Asset.Asset[int64]{
		assetNamet1: 50,
	}
	val1 := Asset.Asset[int64]{
		assetNamet1: 100,
	}
	if !val1.Greater(val0) {
		t.Errorf("Amounts should be equal")
	}
	if val0.Greater(val1) {
		t.Errorf("Amounts should not be equal")
	}
	val2 := Asset.Asset[int64]{}
	if !val1.Greater(val2) {
		t.Errorf("Amounts should be equal")
	}
	if !val2.Greater(val1) {
		t.Errorf("Amounts should not be equal")
	}
}

func TestSub(t *testing.T) {
	val0 := Asset.Asset[int64]{}
	val0[assetNamet1] = 100
	val1 := Asset.Asset[int64]{}
	val1[assetNamet1] = 50
	val2 := val0.Sub(val1)
	if val2[assetNamet1] != 50 {
		t.Errorf("Amounts should be equal")
	}
	val3 := Asset.Asset[int64]{}
	val3 = val3.Sub(val1)
	if val3[assetNamet1] != -50 {
		t.Errorf("Amounts should be equal")
	}
}

func TestAdd(t *testing.T) {
	val0 := Asset.Asset[int64]{}
	val0[assetNamet1] = 100
	val1 := Asset.Asset[int64]{}
	val1[assetNamet1] = 50
	val2 := val0.Add(val1)
	if val2[assetNamet1] != 150 {
		t.Errorf("Amounts should be equal")
	}
	val3 := Asset.Asset[int64]{}
	val3 = val3.Add(val1)
	if val3[assetNamet1] != 50 {
		t.Errorf("Amounts should be equal")
	}
}

func TestInverted(t *testing.T) {
	val0 := Asset.Asset[int64]{}
	val0[assetNamet1] = 100
	val1 := val0.Inverted()
	if val1[assetNamet1] != -100 {
		t.Errorf("Amounts should be equal")
	}
}

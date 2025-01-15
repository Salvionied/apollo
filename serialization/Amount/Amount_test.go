package Amount_test

import (
	"encoding/hex"
	"testing"

	"github.com/Salvionied/apollo/serialization/Amount"
	"github.com/Salvionied/apollo/serialization/AssetName"
	"github.com/Salvionied/apollo/serialization/MultiAsset"
	"github.com/Salvionied/apollo/serialization/Policy"
	"github.com/Salvionied/cbor/v2"
)

var policy = Policy.PolicyId{Value: "fc11a9ef431f81b837736be5f53e4da29b9469c983d07f321262ce61"}
var assetNamet1 = AssetName.NewAssetNameFromString("test")
var assetNamet2 = AssetName.NewAssetNameFromString("test2")

func TestEqualityNoAssets(t *testing.T) {
	val0 := Amount.Amount{
		Coin:  100,
		Value: MultiAsset.MultiAsset[int64]{},
	}
	val1 := Amount.Amount{
		Coin:  100,
		Value: MultiAsset.MultiAsset[int64]{},
	}
	if !val0.Equal(val1) {
		t.Errorf("Amounts should be equal")
	}
}

func TestEqualityAssets(t *testing.T) {
	val0 := Amount.Amount{
		Coin: 100,
		Value: MultiAsset.MultiAsset[int64]{
			policy: {assetNamet1: 100},
		}}
	val1 := Amount.Amount{
		Coin: 100,
		Value: MultiAsset.MultiAsset[int64]{
			policy: {assetNamet1: 100},
		}}
	if !val0.Equal(val1) {
		t.Errorf("Amounts should be equal")
	}
	val2 := Amount.Amount{
		Coin: 100,
		Value: MultiAsset.MultiAsset[int64]{
			policy: {assetNamet1: 99},
		}}
	if val0.Equal(val2) {
		t.Errorf("Amounts should not be equal")
	}

}

func TestNoAssetPreAlonzoMarshaling(t *testing.T) {
	val0 := Amount.Amount{
		Coin:  100,
		Value: MultiAsset.MultiAsset[int64]{},
	}
	marshaled, _ := cbor.Marshal(val0)
	if hex.EncodeToString(marshaled) != "821864a0" {
		t.Errorf("Marshaling failed. Expected: 821864a0, Got: %x", marshaled)
	}

	var unmarshaled Amount.Amount
	err := cbor.Unmarshal(marshaled, &unmarshaled)
	if err != nil {
		t.Error("Failed unmarshaling", err)
	}
	if !val0.Equal(unmarshaled) {
		t.Errorf("Unmarshaling failed. Expected: %v, Got: %v", val0, unmarshaled)
	}
}

func TestPreAlonzoMarshalingWAssets(t *testing.T) {
	val0 := Amount.Amount{
		Coin: 100,
		Value: MultiAsset.MultiAsset[int64]{
			policy: {assetNamet1: 100},
		}}
	marshaled, _ := cbor.Marshal(val0)
	if hex.EncodeToString(marshaled) != "821864a1581cfc11a9ef431f81b837736be5f53e4da29b9469c983d07f321262ce61a144746573741864" {
		t.Errorf("Marshaling failed. Expected: 821864a1581cfc11a9ef431f81b837736be5f53e4da29b9469c983d07f321262ce61a144746573741864, Got: %x", marshaled)
	}
	var unmarshaled Amount.Amount
	err := cbor.Unmarshal(marshaled, &unmarshaled)
	if err != nil {
		t.Error("Failed unmarshaling", err)
	}
	if !val0.Equal(unmarshaled) {
		t.Errorf("Unmarshaling failed. Expected: %v, Got: %v", val0, unmarshaled)
	}
}

func TestToAlonzoAmount(t *testing.T) {
	val0 := Amount.Amount{
		Coin: 100,
		Value: MultiAsset.MultiAsset[int64]{
			policy: {assetNamet1: 100},
		}}
	alonzoAmount := val0.ToAlonzo()
	if alonzoAmount.Coin != 100 {
		t.Errorf("Coin value should be 100, got %d", alonzoAmount.Coin)
	}
	if alonzoAmount.Value[policy][assetNamet1] != 100 {
		t.Errorf("Asset value should be 100, got %d", alonzoAmount.Value[policy][assetNamet1])
	}
}

func TestFromAlonzoAmount(t *testing.T) {
	alonzoAmount := Amount.AlonzoAmount{
		Coin: 100,
		Value: MultiAsset.MultiAsset[int64]{
			policy: {assetNamet1: 100},
		}}
	amount := alonzoAmount.ToShelley()
	if amount.Coin != 100 {
		t.Errorf("Coin value should be 100, got %d", amount.Coin)
	}
	if amount.Value[policy][assetNamet1] != 100 {
		t.Errorf("Asset value should be 100, got %d", amount.Value[policy][assetNamet1])
	}
}

func TestCloneShelley(t *testing.T) {
	val0 := Amount.Amount{
		Coin: 100,
		Value: MultiAsset.MultiAsset[int64]{
			policy: {assetNamet1: 100},
		}}
	val1 := val0.Clone()
	if !val0.Equal(val1) {
		t.Errorf("Amounts should be equal")
	}
	if &val0 == &val1 {
		t.Errorf("Amounts should not be the same object")
	}
	if &val0.Value == &val1.Value {
		t.Errorf("MultiAssets should not be the same object")
	}
}

func TestCloneAlonzo(t *testing.T) {
	val0 := Amount.AlonzoAmount{
		Coin: 100,
		Value: MultiAsset.MultiAsset[int64]{
			policy: {assetNamet1: 100},
		}}
	val1 := val0.Clone()
	if val0.Coin != val1.Coin {
		t.Errorf("Coins should be equal")
	}
	if val0.Value[policy][assetNamet1] != val1.Value[policy][assetNamet1] {
		t.Errorf("Assets should be equal")
	}
	if &val0 == &val1 {
		t.Errorf("Amounts should not be the same object")
	}
	if &val0.Value == &val1.Value {
		t.Errorf("MultiAssets should not be the same object")
	}
}

func TestZeroAssetRemoval(t *testing.T) {
	val0 := Amount.Amount{
		Coin: 100,
		Value: MultiAsset.MultiAsset[int64]{
			policy: {assetNamet1: 0},
		}}
	val1 := val0.RemoveZeroAssets()
	if val1.Value[policy][assetNamet1] != 0 {
		t.Errorf("Asset should be 0")
	}
}

func TestStrictLess(t *testing.T) {
	val0 := Amount.Amount{
		Coin: 100,
		Value: MultiAsset.MultiAsset[int64]{
			policy: {assetNamet1: 100},
		}}
	val1 := Amount.Amount{
		Coin: 100,
		Value: MultiAsset.MultiAsset[int64]{
			policy: {assetNamet1: 99},
		}}
	if val1.Less(val0) {
		t.Errorf("val1 should be less than val0")
	}
	if val0.Less(val1) {
		t.Errorf("val0 should not be less than val1")
	}
}

func TestStrictLessCoin(t *testing.T) {
	val0 := Amount.Amount{
		Coin: 100,
		Value: MultiAsset.MultiAsset[int64]{
			policy: {assetNamet1: 100},
		}}
	val1 := Amount.Amount{
		Coin: 99,
		Value: MultiAsset.MultiAsset[int64]{
			policy: {assetNamet1: 100},
		}}
	if !val1.Less(val0) {
		t.Errorf("val1 should be less than val0")
	}
	if val0.Less(val1) {
		t.Errorf("val0 should not be less than val1")
	}
}

func TestStrictGreater(t *testing.T) {
	val0 := Amount.Amount{
		Coin: 100,
		Value: MultiAsset.MultiAsset[int64]{
			policy: {assetNamet1: 100},
		}}
	val1 := Amount.Amount{
		Coin: 100,
		Value: MultiAsset.MultiAsset[int64]{
			policy: {assetNamet1: 99},
		}}
	if val1.Greater(val0) {
		t.Errorf("val1 should not be greater than val0")
	}
	if val0.Greater(val1) {
		t.Errorf("val0 should be greater than val1")
	}
}

func TestStrictGreaterCoin(t *testing.T) {
	val0 := Amount.Amount{
		Coin: 100,
		Value: MultiAsset.MultiAsset[int64]{
			policy: {assetNamet1: 100},
		}}
	val1 := Amount.Amount{
		Coin: 99,
		Value: MultiAsset.MultiAsset[int64]{
			policy: {assetNamet1: 100},
		}}
	if val1.Greater(val0) {
		t.Errorf("val1 should not be greater than val0")
	}
	if !val0.Greater(val1) {
		t.Errorf("val0 should be greater than val1")
	}
}

func TestAddCoin(t *testing.T) {
	val0 := Amount.Amount{
		Coin: 100,
		Value: MultiAsset.MultiAsset[int64]{
			policy: {assetNamet1: 100},
		}}
	val1 := Amount.Amount{
		Coin:  99,
		Value: MultiAsset.MultiAsset[int64]{}}
	val0 = val0.Add(val1)
	if val0.Coin != 199 {
		t.Errorf("Coin should be 199, got %d", val0.Coin)
	}
	if len(val0.Value) != 1 {
		t.Errorf("Asset should be 1, got %d", len(val0.Value))
	}
}

func TestSubCoin(t *testing.T) {
	val := Amount.Amount{
		Coin: 100,
		Value: MultiAsset.MultiAsset[int64]{
			policy: {assetNamet1: 100, assetNamet2: 100},
		}}
	val1 := Amount.Amount{
		Coin:  99,
		Value: MultiAsset.MultiAsset[int64]{}}
	val = val.Sub(val1)
	if val.Coin != 1 {
		t.Errorf("Coin should be 1, got %d", val.Coin)
	}
	if len(val.Value) != 1 {
		t.Errorf("Asset should be 1, got %d", len(val.Value))
	}
}

func TestSubCoinAndAsset(t *testing.T) {
	val := Amount.Amount{
		Coin: 100,
		Value: MultiAsset.MultiAsset[int64]{
			policy: {assetNamet1: 100, assetNamet2: 100},
		}}
	val1 := Amount.Amount{
		Coin: 99,
		Value: MultiAsset.MultiAsset[int64]{
			policy: {assetNamet1: 99, assetNamet2: 99},
		}}
	val = val.Sub(val1)
	if val.Coin != 1 {
		t.Errorf("Coin should be 1, got %d", val.Coin)
	}
	if len(val.Value) != 1 {
		t.Errorf("Asset should be 1, got %d", len(val.Value))
	}
	if val.Value[policy][assetNamet1] != 1 {
		t.Errorf("Asset value should be 1, got %d", val.Value[policy][assetNamet1])
	}
	if val.Value[policy][assetNamet2] != 1 {
		t.Errorf("Asset value should be 1, got %d", val.Value[policy][assetNamet2])
	}
}

func TestAdd(t *testing.T) {
	val0 := Amount.Amount{
		Coin:  100,
		Value: MultiAsset.MultiAsset[int64]{}}
	val1 := Amount.Amount{
		Coin: 99,
		Value: MultiAsset.MultiAsset[int64]{
			policy: {assetNamet1: 100, assetNamet2: 100},
		}}
	val0 = val0.Add(val1)
	if val0.Coin != 199 {
		t.Errorf("Coin should be 199, got %d", val0.Coin)
	}
	if len(val0.Value) != 1 {
		t.Errorf("Asset should be 1, got %d", len(val0.Value))
	}
	if val0.Value[policy][assetNamet1] != 100 {
		t.Errorf("Asset value should be 100, got %d", val0.Value[policy][assetNamet1])
	}
	if val0.Value[policy][assetNamet2] != 100 {
		t.Errorf("Asset value should be 100, got %d", val0.Value[policy][assetNamet2])
	}

}

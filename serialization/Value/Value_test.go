package Value_test

import (
	"encoding/hex"
	"testing"

	"github.com/Salvionied/apollo/serialization/Amount"
	"github.com/Salvionied/apollo/serialization/Asset"
	"github.com/Salvionied/apollo/serialization/AssetName"
	"github.com/Salvionied/apollo/serialization/MultiAsset"
	"github.com/Salvionied/apollo/serialization/Policy"
	"github.com/Salvionied/apollo/serialization/Value"
	"github.com/fxamacker/cbor/v2"
)

func TestMarshalCBOR(t *testing.T) {

	type expectedResult struct {
		Error   string
		Result  string
		IsError bool
	}
	type test struct {
		input    Value.Value
		expected expectedResult
	}

	vectors := map[string]test{
		"No Assets": {
			input: Value.Value{
				Coin:      15000000,
				HasAssets: false,
			},
			expected: expectedResult{
				Result:  hex.EncodeToString([]byte{26, 0, 228, 225, 192}),
				IsError: false,
			},
		},
		"Test from pycardano": {
			input: Value.Value{
				Am: Amount.Amount{
					Value: MultiAsset.MultiAsset[int64]{
						Policy.PolicyId{Value: "ec8b7d1dd0b124e8333d3fa8d818f6eac068231a287554e9ceae490e"}: Asset.Asset[int64]{
							*AssetName.NewAssetNameFromHexString("5365636f6e6454657374746f6b656e"): 10000000,
							*AssetName.NewAssetNameFromHexString("54657374746f6b656e"):             10000000,
						},
					},
				},
				HasAssets: true,
			},
			expected: expectedResult{
				Result:  "8200a1581cec8b7d1dd0b124e8333d3fa8d818f6eac068231a287554e9ceae490ea24954657374746f6b656e1a009896804f5365636f6e6454657374746f6b656e1a00989680",
				IsError: false,
			},
		},
		"Empty Asset name": {
			input: Value.Value{
				Am: Amount.Amount{
					Coin: 2000000,
					Value: MultiAsset.MultiAsset[int64]{
						Policy.PolicyId{
							Value: "725BA16E744ABF2074C951C320FCC92EA0158ED7BB325B092A58245D",
						}: Asset.Asset[int64]{
							*AssetName.NewAssetNameFromHexString(""): 1,
						},
					},
				},
				HasAssets: true,
			}, expected: expectedResult{
				Result: hex.EncodeToString(
					[]byte{
						0x82,
						0x1A,
						0x00,
						0x01E,
						0x84,
						0x80,
						0xA1,
						0x58,
						0x1C,
						0x72,
						0x5B,
						0xA1,
						0x6E,
						0x74,
						0x4A,
						0xBF,
						0x20,
						0x74,
						0xC9,
						0x51,
						0xC3,
						0x20,
						0xFC,
						0xC9,
						0x2E,
						0xA0,
						0x15,
						0x8E,
						0xD7,
						0xBB,
						0x32,
						0x5B,
						0x09,
						0x2A,
						0x58,
						0x24,
						0x5D,
						0xA1,
						0x40,
						0x01,
					},
				),
				IsError: false,
			},
		},
	}

	for name, testCase := range vectors {
		t.Run(name, func(t *testing.T) {
			res, err := testCase.input.MarshalCBOR()
			result := hex.EncodeToString(res)
			if testCase.expected.IsError {
				if err == nil {
					t.Errorf(
						"\ntest: %v\ninput: %v\nexpected: %v\nresult: %v",
						name,
						testCase.input,
						testCase.expected.Error,
						result,
					)
				} else if err.Error() != testCase.expected.Error {
					t.Errorf("\ntest: %v\ninput: %v\nexpected: %v\nresult: %v", name, testCase.input, testCase.expected.Error, err.Error())
				}
			} else {
				if testCase.expected.Result != result {
					t.Errorf("\ntest: %v\ninput: %v\nexpected: %v\nresult: %v", name, testCase.input, testCase.expected.Result, result)
				}
			}
		})
	}
}

func TestValueComparations(t *testing.T) {
	val := Value.PureLovelaceValue(10_000_000)
	val2 := Value.PureLovelaceValue(15_000_000)

	if !val.Less(val2) {
		t.Errorf("val should be less than val2")
	}
	if !val2.Greater(val) {
		t.Errorf("val2 should be greater than val")
	}
	if !val.LessOrEqual(val2) {
		t.Errorf("val should be less or equal than val2")
	}
	if !val2.GreaterOrEqual(val) {
		t.Errorf("val2 should be greater or equal than val")
	}

}

func TestEquality(t *testing.T) {
	val := Value.PureLovelaceValue(1_000_000)
	clonedVal := val.Clone()
	if !val.Equal(clonedVal) {
		t.Errorf("val should be equal to clonedVal")
	}

}

var testValue = Value.PureLovelaceValue(10_000_000)

var testPolicy = Policy.PolicyId{
	Value: "ec8b7d1dd0b124e8333d3fa8d818f6eac068231a287554e9ceae490e",
}

var testAssetName = AssetName.NewAssetNameFromHexString(
	"5365636f6e6454657374746f6b656e",
)
var testAsset = Asset.Asset[int64]{
	*testAssetName: 10000000,
	*AssetName.NewAssetNameFromHexString("54657374746f6b656e"): 10000000,
}
var testMultiAsset = MultiAsset.MultiAsset[int64]{
	testPolicy: testAsset,
}
var testValueWithAssets = Value.SimpleValue(10_000_000, testMultiAsset)

func TestMarshalAndUnmarshalAlonzoValue(t *testing.T) {
	noTokenAlValue := testValue.ToAlonzoValue()
	marshaled, err := cbor.Marshal(noTokenAlValue)
	if err != nil {
		t.Errorf("error while marshaling: %v", err)
	}
	if hex.EncodeToString(marshaled) != "1a00989680" {
		t.Error(
			"marshaled value should be equal to the expected one",
			hex.EncodeToString(marshaled),
		)
	}
	unmarshaled := Value.AlonzoValue{}
	err = cbor.Unmarshal(marshaled, &unmarshaled)
	if err != nil {
		t.Errorf("error while unmarshaling: %v", err)
	}
	if !unmarshaled.ToValue().Equal(noTokenAlValue.ToValue()) {
		t.Errorf("unmarshaled value should be equal to the marshaled one")
	}
	withTokenAlValue := testValueWithAssets.ToAlonzoValue()
	marshaled, err = cbor.Marshal(withTokenAlValue)
	if err != nil {
		t.Errorf("error while marshaling: %v", err)
	}
	if hex.EncodeToString(
		marshaled,
	) != "821a00989680a1581cec8b7d1dd0b124e8333d3fa8d818f6eac068231a287554e9ceae490ea24954657374746f6b656e1a009896804f5365636f6e6454657374746f6b656e1a00989680" {
		t.Error(
			"marshaled value should be equal to the expected one",
			hex.EncodeToString(marshaled),
		)
	}
	unmarshaled = Value.AlonzoValue{}
	err = cbor.Unmarshal(marshaled, &unmarshaled)
	if err != nil {
		t.Errorf("error while unmarshaling: %v", err)
	}
	if !unmarshaled.ToValue().Equal(withTokenAlValue.ToValue()) {
		t.Errorf("unmarshaled value should be equal to the marshaled one")
	}
}

func TestAlonzoValueClone(t *testing.T) {
	original := testValueWithAssets.ToAlonzoValue()
	cloned := original.Clone()
	if !original.ToValue().Equal(cloned.ToValue()) {
		t.Errorf("cloned value should be equal to the original one")
	}
	ogNoToken := testValue.ToAlonzoValue()
	clonedNoToken := ogNoToken.Clone()
	if !ogNoToken.ToValue().Equal(clonedNoToken.ToValue()) {
		t.Errorf("cloned value should be equal to the original one")
	}

}

func TestToAlonzo(t *testing.T) {
	original := testValueWithAssets
	alonzo := original.ToAlonzoValue()
	if !original.Equal(alonzo.ToValue()) {
		t.Errorf("original value should be equal to the alonzo one")
	}
}

func TestRemoveZeroAssets(t *testing.T) {
	original := Value.SimpleValue(1_000_000, MultiAsset.MultiAsset[int64]{
		Policy.PolicyId{Value: "ec8b7d1dd0b124e8333d3fa8d818f6eac068231a287554e9ceae490e"}: Asset.Asset[int64]{
			*AssetName.NewAssetNameFromHexString("5365636f6e6454657374746f6b656e"): 0,
		},
	})
	removed := original.RemoveZeroAssets()
	if len(removed.ToAlonzoValue().ToValue().Am.Value) != 0 {
		t.Errorf("removed value should have no assets")
	}
}

func TestValueClone(t *testing.T) {
	original := testValueWithAssets
	cloned := original.Clone()
	if !original.Equal(cloned) {
		t.Errorf("cloned value should be equal to the original one")
	}
	ogNoToken := testValue
	clonedNoToken := ogNoToken.Clone()
	if !ogNoToken.Equal(clonedNoToken) {
		t.Errorf("cloned value should be equal to the original one")
	}
}

func TestAddAssets(t *testing.T) {
	original := testValue
	original.AddAssets(testMultiAsset)
	if !original.Equal(testValueWithAssets) {
		t.Errorf("original value should be equal to the testValueWithAssets")
	}

	original.AddAssets(testMultiAsset)
	if !original.Equal(
		Value.SimpleValue(10_000_000, MultiAsset.MultiAsset[int64]{
			Policy.PolicyId{Value: "ec8b7d1dd0b124e8333d3fa8d818f6eac068231a287554e9ceae490e"}: Asset.Asset[int64]{
				*testAssetName: 20000000,
				*AssetName.NewAssetNameFromHexString("54657374746f6b656e"): 20000000,
			},
		}),
	) {
		t.Error(
			"og value should be equal to the testValueWithAssets",
			original.String(),
		)
	}
}

func TestSimpleValue(t *testing.T) {
	newValNoToken := Value.SimpleValue(
		10_000_000,
		MultiAsset.MultiAsset[int64]{},
	)
	if !newValNoToken.Equal(testValue) {
		t.Errorf("newValNoToken should be equal to testValue")
	}
	newValWithToken := Value.SimpleValue(10_000_000, testMultiAsset)
	if !newValWithToken.Equal(testValueWithAssets) {
		t.Errorf("newValWithToken should be equal to testValueWithAssets")
	}
}

func TestSubLovelace(t *testing.T) {
	original := testValueWithAssets
	original.SubLovelace(5_000_000)
	if !original.Equal(Value.SimpleValue(5_000_000, testMultiAsset)) {
		t.Errorf("original value should be equal to the testValueWithAssets")
	}

	ogNoToken := testValue
	ogNoToken.SubLovelace(5_000_000)
	if !ogNoToken.Equal(Value.PureLovelaceValue(5_000_000)) {
		t.Errorf("ogNoToken value should be equal to the testValueWithAssets")
	}

}

func TestAddLovelace(t *testing.T) {
	original := testValueWithAssets
	original.AddLovelace(5_000_000)
	if !original.Equal(Value.SimpleValue(15_000_000, testMultiAsset)) {
		t.Errorf("original value should be equal to the testValueWithAssets")
	}

	ogNoToken := testValue
	ogNoToken.AddLovelace(5_000_000)
	if !ogNoToken.Equal(Value.PureLovelaceValue(15_000_000)) {
		t.Errorf("ogNoToken value should be equal to the testValueWithAssets")
	}
}

func TestSetLovelace(t *testing.T) {
	original := testValueWithAssets
	original.SetLovelace(5_000_000)
	if !original.Equal(Value.SimpleValue(5_000_000, testMultiAsset)) {
		t.Errorf("original value should be equal to the testValueWithAssets")
	}

	ogNoToken := testValue
	ogNoToken.SetLovelace(5_000_000)
	if !ogNoToken.Equal(Value.PureLovelaceValue(5_000_000)) {
		t.Errorf("ogNoToken value should be equal to the testValueWithAssets")
	}
}

func TestSetMultiAsset(t *testing.T) {
	original := testValueWithAssets
	ma := MultiAsset.MultiAsset[int64]{
		Policy.PolicyId{Value: "ec8b7d1dd0b124e8333d3fa8d818f6eac068231a287554e9ceae490e"}: Asset.Asset[int64]{
			*AssetName.NewAssetNameFromHexString("5365636f6e6454657374746f6b656e"): 12,
		},
	}
	original.SetMultiAsset(ma)
	if !original.Equal(Value.SimpleValue(10_000_000, ma)) {
		t.Errorf("original value should be equal to the testValueWithAssets")
	}

	ogNoToken := testValue
	ogNoToken.SetMultiAsset(testMultiAsset)
	if !ogNoToken.Equal(Value.SimpleValue(10_000_000, testMultiAsset)) {
		t.Errorf("ogNoToken value should be equal to the testValueWithAssets")
	}
}

func TestGetCoin(t *testing.T) {
	original := testValueWithAssets
	if original.GetCoin() != 10_000_000 {
		t.Errorf("original value should be equal to the testValueWithAssets")
	}

	ogNoToken := testValue
	if ogNoToken.GetCoin() != 10_000_000 {
		t.Errorf("ogNoToken value should be equal to the testValueWithAssets")
	}
}

func TestGetMultiAsset(t *testing.T) {
	original := testValueWithAssets
	if !original.GetAssets().Equal(testMultiAsset) {
		t.Errorf("original value should be equal to the testValueWithAssets")
	}

	ogNoToken := testValue
	if len(ogNoToken.GetAssets()) != 0 {
		t.Errorf("ogNoToken value should be equal to the testValueWithAssets")
	}
}

func TestAdd(t *testing.T) {
	original := testValueWithAssets
	original = original.Add(testValue)
	if !original.Equal(Value.SimpleValue(20_000_000, testMultiAsset)) {
		t.Errorf("original value should be equal to the testValueWithAssets")
	}

	ogNoToken := testValue
	ogNoToken = ogNoToken.Add(testValue)
	if !ogNoToken.Equal(Value.PureLovelaceValue(20_000_000)) {
		t.Errorf("ogNoToken value should be equal to the testValueWithAssets")
	}

	// Test adding assets
	original = testValueWithAssets
	original = original.Add(testValueWithAssets)
	if !original.Equal(
		Value.SimpleValue(20_000_000, MultiAsset.MultiAsset[int64]{
			Policy.PolicyId{Value: "ec8b7d1dd0b124e8333d3fa8d818f6eac068231a287554e9ceae490e"}: Asset.Asset[int64]{
				*AssetName.NewAssetNameFromHexString("5365636f6e6454657374746f6b656e"): 20000000,
				*AssetName.NewAssetNameFromHexString("54657374746f6b656e"):             20000000,
			},
		}),
	) {
		t.Errorf("original value should be equal to the testValueWithAssets")
	}

}

func TestSub(t *testing.T) {
	original := testValueWithAssets
	original = original.Sub(testValue)
	if !original.Equal(Value.SimpleValue(0, testMultiAsset)) {
		t.Errorf("original value should be equal to the testValueWithAssets")
	}

	ogNoToken := testValue
	ogNoToken = ogNoToken.Sub(testValue)
	if !ogNoToken.Equal(Value.PureLovelaceValue(0)) {
		t.Errorf("ogNoToken value should be equal to the testValueWithAssets")
	}

	// Test adding assets
	original = testValueWithAssets
	original = original.Sub(testValueWithAssets)
	if !original.Equal(Value.SimpleValue(0, MultiAsset.MultiAsset[int64]{
		Policy.PolicyId{Value: "ec8b7d1dd0b124e8333d3fa8d818f6eac068231a287554e9ceae490e"}: Asset.Asset[int64]{
			*AssetName.NewAssetNameFromHexString("5365636f6e6454657374746f6b656e"): 0,
			*AssetName.NewAssetNameFromHexString("54657374746f6b656e"):             0,
		},
	})) {
		t.Errorf("original value should be equal to the testValueWithAssets")
	}
}

func TestMarshalAndUmnarshalNormalValue(t *testing.T) {
	marshaled, err := cbor.Marshal(testValue)
	if err != nil {
		t.Errorf("error while marshaling: %v", err)
	}
	if hex.EncodeToString(marshaled) != "1a00989680" {
		t.Error(
			"marshaled value should be equal to the expected one",
			hex.EncodeToString(marshaled),
		)
	}
	unmarshaled := Value.Value{}
	err = cbor.Unmarshal(marshaled, &unmarshaled)
	if err != nil {
		t.Errorf("error while unmarshaling: %v", err)
	}
	if !unmarshaled.Equal(testValue) {
		t.Errorf("unmarshaled value should be equal to the marshaled one")
	}
}

package value_test

import (
	"encoding/hex"
	"testing"

	"github.com/salvionied/apollo/serialization/Amount"
	"github.com/salvionied/apollo/serialization/Asset"
	"github.com/salvionied/apollo/serialization/AssetName"
	"github.com/salvionied/apollo/serialization/MultiAsset"
	"github.com/salvionied/apollo/serialization/Policy"
	"github.com/salvionied/apollo/serialization/Value"
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
				Result:  hex.EncodeToString([]byte{0x82, 0x1A, 0x00, 0x01E, 0x84, 0x80, 0xA1, 0x58, 0x1C, 0x72, 0x5B, 0xA1, 0x6E, 0x74, 0x4A, 0xBF, 0x20, 0x74, 0xC9, 0x51, 0xC3, 0x20, 0xFC, 0xC9, 0x2E, 0xA0, 0x15, 0x8E, 0xD7, 0xBB, 0x32, 0x5B, 0x09, 0x2A, 0x58, 0x24, 0x5D, 0xA1, 0x40, 0x01}),
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
					t.Errorf("\ntest: %v\ninput: %v\nexpected: %v\nresult: %v", name, testCase.input, testCase.expected.Error, result)
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

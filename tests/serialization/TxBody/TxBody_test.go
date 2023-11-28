package txbody_test

import (
	"encoding/hex"
	"testing"

	"github.com/Salvionied/cbor/v2"
	"github.com/SundaeSwap-finance/apollo/serialization/Address"
	"github.com/SundaeSwap-finance/apollo/serialization/TransactionBody"
	"github.com/SundaeSwap-finance/apollo/serialization/TransactionInput"
	"github.com/SundaeSwap-finance/apollo/serialization/TransactionOutput"
	"github.com/SundaeSwap-finance/apollo/serialization/Value"
)

func mustDecodeHexString(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}

func mustDecodeAddress(s string) Address.Address {
	a, err := Address.DecodeAddress(s)
	if err != nil {
		panic(err)
	}
	return a
}

func TestTxBodyMarshalCBOR(t *testing.T) {

	type expectedResult struct {
		Error   string
		Result  string
		IsError bool
	}
	type test struct {
		input    TransactionBody.TransactionBody
		expected expectedResult
	}
	vectors := map[string]test{
		"Simple": {
			input: TransactionBody.TransactionBody{
				Inputs: []TransactionInput.TransactionInput{
					{
						TransactionId: mustDecodeHexString("e8a7a0b0e5b883e5bda9e5b1b1e5b88be8aeba"),
						Index:         0,
					},
				},
				Outputs: []TransactionOutput.TransactionOutput{
					TransactionOutput.SimpleTransactionOutput(
						mustDecodeAddress("addr_test1vrj2asywelxue68wlz84g6xpjfv69vn9arknsgxvtlg2uusqey860"),
						Value.Value{
							Coin: 1000000,
						},
					),
				},
				Fee:          1000000,
				Ttl:          1000,
				Withdrawals:  nil,
				Certificates: nil,
			},
			expected: expectedResult{
				Result:  "a400818253e8a7a0b0e5b883e5bda9e5b1b1e5b88be8aeba00018182581d60e4aec08ecfcdcce8eef88f5468c19259a2b265e8ed3820cc5fd0ae721a000f4240021a000f4240031903e8",
				IsError: false,
			},
		},
	}

	for name, testCase := range vectors {
		t.Run(name, func(t *testing.T) {
			em, _ := cbor.CanonicalEncOptions().EncMode()
			res, err := em.Marshal(testCase.input)
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

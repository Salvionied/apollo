package Address_test

import (
	"encoding/hex"
	"fmt"
	"reflect"
	"testing"

	"github.com/Salvionied/apollo/serialization/Address"
)

func TestDecodeAddress(t *testing.T) {
	type expectedResult struct {
		Error   string
		Result  Address.Address
		IsError bool
	}

	type testDecodeCase struct {
		input    string
		expected expectedResult
	}
	/**
	  The following test vectors use the following payment key, script key, script and pointer: (From CIP-0019)
	  addr_vk1w0l2sr2zgfm26ztc6nl9xy8ghsk5sh6ldwemlpmp9xylzy4dtf7st80zhd
	  stake_vk1px4j0r2fk7ux5p23shz8f3y5y2qam7s954rgf3lg5merqcj6aetsft99wu
	  script1cda3khwqv60360rp5m7akt50m6ttapacs8rqhn5w342z7r35m37
	  (2498243, 27, 3)

	  There is no pointer implementation in the decoding function therefore there are no test cases for it
	  **/
	cases := map[string]testDecodeCase{
		"Valid KEY_KEY mainnet address": {
			input: "addr1qx2fxv2umyhttkxyxp8x0dlpdt3k6cwng5pxj3jhsydzer3n0d3vllmyqwsx5wktcd8cc3sq835lu7drv2xwl2wywfgse35a3x",
			expected: expectedResult{
				Result: Address.Address{
					PaymentPart: []byte{148, 147, 49, 92, 217, 46, 181, 216, 196, 48, 78, 103, 183, 225, 106, 227, 109, 97, 211, 69, 2, 105, 70, 87, 129, 26, 44, 142},
					StakingPart: []byte{51, 123, 98, 207, 255, 100, 3, 160, 106, 58, 203, 195, 79, 140, 70, 0, 60, 105, 254, 121, 163, 98, 140, 239, 169, 196, 114, 81},
					Network:     Address.MAINNET,
					AddressType: Address.KEY_KEY,
					HeaderByte:  0b00000001,
					Hrp:         "addr",
				},
				IsError: false,
			},
		},
		"Valid SCRIPT_KEY mainnet Address": {
			input: "addr1z8phkx6acpnf78fuvxn0mkew3l0fd058hzquvz7w36x4gten0d3vllmyqwsx5wktcd8cc3sq835lu7drv2xwl2wywfgs9yc0hh",
			expected: expectedResult{
				Result: Address.Address{
					PaymentPart: []byte{195, 123, 27, 93, 192, 102, 159, 29, 60, 97, 166, 253, 219, 46, 143, 222, 150, 190, 135, 184, 129, 198, 11, 206, 142, 141, 84, 47},
					StakingPart: []byte{51, 123, 98, 207, 255, 100, 3, 160, 106, 58, 203, 195, 79, 140, 70, 0, 60, 105, 254, 121, 163, 98, 140, 239, 169, 196, 114, 81},
					Network:     Address.MAINNET,
					AddressType: Address.SCRIPT_KEY,
					HeaderByte:  0b00010001,
					Hrp:         "addr",
				},
				IsError: false,
			},
		},
		"Valid KEY_SCRIPT mainnet Address": {
			input: "addr1yx2fxv2umyhttkxyxp8x0dlpdt3k6cwng5pxj3jhsydzerkr0vd4msrxnuwnccdxlhdjar77j6lg0wypcc9uar5d2shs2z78ve",
			expected: expectedResult{
				Result: Address.Address{
					PaymentPart: []byte{148, 147, 49, 92, 217, 46, 181, 216, 196, 48, 78, 103, 183, 225, 106, 227, 109, 97, 211, 69, 2, 105, 70, 87, 129, 26, 44, 142},
					StakingPart: []byte{195, 123, 27, 93, 192, 102, 159, 29, 60, 97, 166, 253, 219, 46, 143, 222, 150, 190, 135, 184, 129, 198, 11, 206, 142, 141, 84, 47},
					Network:     Address.MAINNET,
					AddressType: Address.KEY_SCRIPT,
					HeaderByte:  0b00100001,
					Hrp:         "addr",
				},
				IsError: false,
			},
		},
		"Valid SCRIPT_SCRIPT mainnet Address": {
			input: "addr1x8phkx6acpnf78fuvxn0mkew3l0fd058hzquvz7w36x4gt7r0vd4msrxnuwnccdxlhdjar77j6lg0wypcc9uar5d2shskhj42g",
			expected: expectedResult{
				Result: Address.Address{
					PaymentPart: []byte{195, 123, 27, 93, 192, 102, 159, 29, 60, 97, 166, 253, 219, 46, 143, 222, 150, 190, 135, 184, 129, 198, 11, 206, 142, 141, 84, 47},
					StakingPart: []byte{195, 123, 27, 93, 192, 102, 159, 29, 60, 97, 166, 253, 219, 46, 143, 222, 150, 190, 135, 184, 129, 198, 11, 206, 142, 141, 84, 47},
					Network:     Address.MAINNET,
					AddressType: Address.SCRIPT_SCRIPT,
					HeaderByte:  0b00110001,
					Hrp:         "addr",
				},
				IsError: false,
			},
		},
		"Valid KEY_NONE mainnet Address": {
			input: "addr1vx2fxv2umyhttkxyxp8x0dlpdt3k6cwng5pxj3jhsydzers66hrl8",
			expected: expectedResult{
				Result: Address.Address{
					PaymentPart: []byte{148, 147, 49, 92, 217, 46, 181, 216, 196, 48, 78, 103, 183, 225, 106, 227, 109, 97, 211, 69, 2, 105, 70, 87, 129, 26, 44, 142},
					StakingPart: make([]byte, 0),
					Network:     Address.MAINNET,
					AddressType: Address.KEY_NONE,
					HeaderByte:  0b01100001,
					Hrp:         "addr",
				},
				IsError: false,
			},
		},
		"Valid SCRIPT_NONE mainnet Address": {
			input: "addr1w8phkx6acpnf78fuvxn0mkew3l0fd058hzquvz7w36x4gtcyjy7wx",
			expected: expectedResult{
				Result: Address.Address{
					PaymentPart: []byte{195, 123, 27, 93, 192, 102, 159, 29, 60, 97, 166, 253, 219, 46, 143, 222, 150, 190, 135, 184, 129, 198, 11, 206, 142, 141, 84, 47},
					StakingPart: make([]byte, 0),
					Network:     Address.MAINNET,
					AddressType: Address.SCRIPT_NONE,
					HeaderByte:  0b01110001,
					Hrp:         "addr",
				},
				IsError: false,
			},
		},
		"Valid NONE_KEY mainnet Address": {
			input: "stake1uyehkck0lajq8gr28t9uxnuvgcqrc6070x3k9r8048z8y5gh6ffgw",
			expected: expectedResult{
				Result: Address.Address{
					PaymentPart: make([]byte, 0),
					StakingPart: []byte{51, 123, 98, 207, 255, 100, 3, 160, 106, 58, 203, 195, 79, 140, 70, 0, 60, 105, 254, 121, 163, 98, 140, 239, 169, 196, 114, 81},
					Network:     Address.MAINNET,
					AddressType: Address.NONE_KEY,
					HeaderByte:  0b11100001,
					Hrp:         "stake",
				},
				IsError: false,
			},
		},
		"Valid NONE_SCRIPT mainnet Address": {
			input: "stake178phkx6acpnf78fuvxn0mkew3l0fd058hzquvz7w36x4gtcccycj5",
			expected: expectedResult{
				Result: Address.Address{
					PaymentPart: make([]byte, 0),
					StakingPart: []byte{195, 123, 27, 93, 192, 102, 159, 29, 60, 97, 166, 253, 219, 46, 143, 222, 150, 190, 135, 184, 129, 198, 11, 206, 142, 141, 84, 47},
					Network:     Address.MAINNET,
					AddressType: Address.NONE_SCRIPT,
					HeaderByte:  0b11110001,
					Hrp:         "stake",
				},
				IsError: false,
			},
		},
		"Valid KEY_KEY testnet address": {
			input: "addr_test1qz2fxv2umyhttkxyxp8x0dlpdt3k6cwng5pxj3jhsydzer3n0d3vllmyqwsx5wktcd8cc3sq835lu7drv2xwl2wywfgs68faae",
			expected: expectedResult{
				Result: Address.Address{
					PaymentPart: []byte{148, 147, 49, 92, 217, 46, 181, 216, 196, 48, 78, 103, 183, 225, 106, 227, 109, 97, 211, 69, 2, 105, 70, 87, 129, 26, 44, 142},
					StakingPart: []byte{51, 123, 98, 207, 255, 100, 3, 160, 106, 58, 203, 195, 79, 140, 70, 0, 60, 105, 254, 121, 163, 98, 140, 239, 169, 196, 114, 81},
					Network:     Address.TESTNET,
					AddressType: Address.KEY_KEY,
					HeaderByte:  0b00000000,
					Hrp:         "addr_test",
				},
				IsError: false,
			},
		},
		"Valid SCRIPT_KEY testnet Address": {
			input: "addr_test1zrphkx6acpnf78fuvxn0mkew3l0fd058hzquvz7w36x4gten0d3vllmyqwsx5wktcd8cc3sq835lu7drv2xwl2wywfgsxj90mg",
			expected: expectedResult{
				Result: Address.Address{
					PaymentPart: []byte{195, 123, 27, 93, 192, 102, 159, 29, 60, 97, 166, 253, 219, 46, 143, 222, 150, 190, 135, 184, 129, 198, 11, 206, 142, 141, 84, 47},
					StakingPart: []byte{51, 123, 98, 207, 255, 100, 3, 160, 106, 58, 203, 195, 79, 140, 70, 0, 60, 105, 254, 121, 163, 98, 140, 239, 169, 196, 114, 81},
					Network:     Address.TESTNET,
					AddressType: Address.SCRIPT_KEY,
					HeaderByte:  0b00010000,
					Hrp:         "addr_test",
				},
				IsError: false,
			},
		},
		"Valid KEY_SCRIPT testnet Address": {
			input: "addr_test1yz2fxv2umyhttkxyxp8x0dlpdt3k6cwng5pxj3jhsydzerkr0vd4msrxnuwnccdxlhdjar77j6lg0wypcc9uar5d2shsf5r8qx",
			expected: expectedResult{
				Result: Address.Address{
					PaymentPart: []byte{148, 147, 49, 92, 217, 46, 181, 216, 196, 48, 78, 103, 183, 225, 106, 227, 109, 97, 211, 69, 2, 105, 70, 87, 129, 26, 44, 142},
					StakingPart: []byte{195, 123, 27, 93, 192, 102, 159, 29, 60, 97, 166, 253, 219, 46, 143, 222, 150, 190, 135, 184, 129, 198, 11, 206, 142, 141, 84, 47},
					Network:     Address.TESTNET,
					AddressType: Address.KEY_SCRIPT,
					HeaderByte:  0b00100000,
					Hrp:         "addr_test",
				},
				IsError: false,
			},
		},
		"Valid SCRIPT_SCRIPT testnet Address": {
			input: "addr_test1xrphkx6acpnf78fuvxn0mkew3l0fd058hzquvz7w36x4gt7r0vd4msrxnuwnccdxlhdjar77j6lg0wypcc9uar5d2shs4p04xh",
			expected: expectedResult{
				Result: Address.Address{
					PaymentPart: []byte{195, 123, 27, 93, 192, 102, 159, 29, 60, 97, 166, 253, 219, 46, 143, 222, 150, 190, 135, 184, 129, 198, 11, 206, 142, 141, 84, 47},
					StakingPart: []byte{195, 123, 27, 93, 192, 102, 159, 29, 60, 97, 166, 253, 219, 46, 143, 222, 150, 190, 135, 184, 129, 198, 11, 206, 142, 141, 84, 47},
					Network:     Address.TESTNET,
					AddressType: Address.SCRIPT_SCRIPT,
					HeaderByte:  0b00110000,
					Hrp:         "addr_test",
				},
				IsError: false,
			},
		},
		"Valid KEY_NONE testnet Address": {
			input: "addr_test1vz2fxv2umyhttkxyxp8x0dlpdt3k6cwng5pxj3jhsydzerspjrlsz",
			expected: expectedResult{
				Result: Address.Address{
					PaymentPart: []byte{148, 147, 49, 92, 217, 46, 181, 216, 196, 48, 78, 103, 183, 225, 106, 227, 109, 97, 211, 69, 2, 105, 70, 87, 129, 26, 44, 142},
					StakingPart: make([]byte, 0),
					Network:     Address.TESTNET,
					AddressType: Address.KEY_NONE,
					HeaderByte:  0b01100000,
					Hrp:         "addr_test",
				},
				IsError: false,
			},
		},
		"Valid SCRIPT_NONE testnet Address": {
			input: "addr_test1wrphkx6acpnf78fuvxn0mkew3l0fd058hzquvz7w36x4gtcl6szpr",
			expected: expectedResult{
				Result: Address.Address{
					PaymentPart: []byte{195, 123, 27, 93, 192, 102, 159, 29, 60, 97, 166, 253, 219, 46, 143, 222, 150, 190, 135, 184, 129, 198, 11, 206, 142, 141, 84, 47},
					StakingPart: make([]byte, 0),
					Network:     Address.TESTNET,
					AddressType: Address.SCRIPT_NONE,
					HeaderByte:  0b01110000,
					Hrp:         "addr_test",
				},
				IsError: false,
			},
		},
		"Valid NONE_KEY testnet Address": {
			input: "stake_test1uqehkck0lajq8gr28t9uxnuvgcqrc6070x3k9r8048z8y5gssrtvn",
			expected: expectedResult{
				Result: Address.Address{
					PaymentPart: make([]byte, 0),
					StakingPart: []byte{51, 123, 98, 207, 255, 100, 3, 160, 106, 58, 203, 195, 79, 140, 70, 0, 60, 105, 254, 121, 163, 98, 140, 239, 169, 196, 114, 81},
					Network:     Address.TESTNET,
					AddressType: Address.NONE_KEY,
					HeaderByte:  0b11100000,
					Hrp:         "stake_test",
				},
				IsError: false,
			},
		},
		"Valid NONE_SCRIPT testnet Address": {
			input: "stake_test17rphkx6acpnf78fuvxn0mkew3l0fd058hzquvz7w36x4gtcljw6kf",
			expected: expectedResult{
				Result: Address.Address{
					PaymentPart: make([]byte, 0),
					StakingPart: []byte{195, 123, 27, 93, 192, 102, 159, 29, 60, 97, 166, 253, 219, 46, 143, 222, 150, 190, 135, 184, 129, 198, 11, 206, 142, 141, 84, 47},
					Network:     Address.TESTNET,
					AddressType: Address.NONE_SCRIPT,
					HeaderByte:  0b11110000,
					Hrp:         "stake_test",
				},
				IsError: false,
			},
		},

		"Invalid bech32 address": {
			input:    "TEST",
			expected: expectedResult{Error: "invalid index of 1", IsError: true},
		},
		"Old Address Format": {
			input:    "DdzFFzCqrhsqohJ5SJXSmtmXWb19MosWJpgbJSK17GnTto1E13YrYqYfTMpzYV4ft2xt5WFqAkbxPZv63pjL3mGW1e299kcqhewLNSvB",
			expected: expectedResult{Error: "string not all lowercase or all uppercase", IsError: true},
		},
		"Invalid Network": {
			input:    "addr1llhwamhwamhq5hyc50",
			expected: expectedResult{Error: "invalid network tag", IsError: true},
		},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			result, err := Address.DecodeAddress(testCase.input)

			if testCase.expected.IsError {
				if err == nil || fmt.Sprint(err) != testCase.expected.Error {
					t.Errorf("\ntest: %v, \nexpected: %v, \nresult: %v", name, testCase.expected.Error, err)
				}
			} else {
				if err != nil || !reflect.DeepEqual(result, testCase.expected.Result) || result.Debug() != testCase.expected.Result.Debug() {
					t.Errorf("\ntest: %v\nexpected: %v\nresult: %v\nerror: %v", name, testCase.expected.Result.Debug(), result.Debug(), err)
				}
			}
		})
	}
}

func TestToCbor(t *testing.T) {
	type testToCborCase struct {
		input    Address.Address
		expected string
	}

	cases := map[string]testToCborCase{
		"Valid KEY_KEY mainnet address": {
			input: Address.Address{
				PaymentPart: []byte{148, 147, 49, 92, 217, 46, 181, 216, 196, 48, 78, 103, 183, 225, 106, 227, 109, 97, 211, 69, 2, 105, 70, 87, 129, 26, 44, 142},
				StakingPart: []byte{51, 123, 98, 207, 255, 100, 3, 160, 106, 58, 203, 195, 79, 140, 70, 0, 60, 105, 254, 121, 163, 98, 140, 239, 169, 196, 114, 81},
				Network:     Address.MAINNET,
				AddressType: Address.KEY_KEY,
				HeaderByte:  0b00000001,
				Hrp:         "addr",
			},
			expected: "5839019493315cd92eb5d8c4304e67b7e16ae36d61d34502694657811a2c8e337b62cfff6403a06a3acbc34f8c46003c69fe79a3628cefa9c47251",
		},
		"Valid KEY_NONE mainnet Address": {
			input: Address.Address{
				PaymentPart: []byte{148, 147, 49, 92, 217, 46, 181, 216, 196, 48, 78, 103, 183, 225, 106, 227, 109, 97, 211, 69, 2, 105, 70, 87, 129, 26, 44, 142},
				StakingPart: make([]byte, 0),
				Network:     Address.MAINNET,
				AddressType: Address.KEY_NONE,
				HeaderByte:  0b01100001,
				Hrp:         "addr",
			},
			expected: "581d619493315cd92eb5d8c4304e67b7e16ae36d61d34502694657811a2c8e",
		},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			result, _ := testCase.input.ToCbor()
			if result != testCase.expected {
				t.Errorf("\ntest: %v\nexpected: %v\nresult: %v", name, testCase.expected, result)
			}
		})
	}
}

func TestString(t *testing.T) {
	type testStringCase struct {
		input    Address.Address
		expected string
	}

	cases := map[string]testStringCase{
		"Valid KEY_KEY mainnet address": {
			expected: "addr1qx2fxv2umyhttkxyxp8x0dlpdt3k6cwng5pxj3jhsydzer3n0d3vllmyqwsx5wktcd8cc3sq835lu7drv2xwl2wywfgse35a3x",
			input: Address.Address{
				PaymentPart: []byte{148, 147, 49, 92, 217, 46, 181, 216, 196, 48, 78, 103, 183, 225, 106, 227, 109, 97, 211, 69, 2, 105, 70, 87, 129, 26, 44, 142},
				StakingPart: []byte{51, 123, 98, 207, 255, 100, 3, 160, 106, 58, 203, 195, 79, 140, 70, 0, 60, 105, 254, 121, 163, 98, 140, 239, 169, 196, 114, 81},
				Network:     Address.MAINNET,
				AddressType: Address.KEY_KEY,
				HeaderByte:  0b00000001,
				Hrp:         "addr",
			},
		},
		"Valid SCRIPT_KEY mainnet Address": {
			expected: "addr1z8phkx6acpnf78fuvxn0mkew3l0fd058hzquvz7w36x4gten0d3vllmyqwsx5wktcd8cc3sq835lu7drv2xwl2wywfgs9yc0hh",
			input: Address.Address{
				PaymentPart: []byte{195, 123, 27, 93, 192, 102, 159, 29, 60, 97, 166, 253, 219, 46, 143, 222, 150, 190, 135, 184, 129, 198, 11, 206, 142, 141, 84, 47},
				StakingPart: []byte{51, 123, 98, 207, 255, 100, 3, 160, 106, 58, 203, 195, 79, 140, 70, 0, 60, 105, 254, 121, 163, 98, 140, 239, 169, 196, 114, 81},
				Network:     Address.MAINNET,
				AddressType: Address.SCRIPT_KEY,
				HeaderByte:  0b00010001,
				Hrp:         "addr",
			},
		},
		"Valid KEY_SCRIPT mainnet Address": {
			expected: "addr1yx2fxv2umyhttkxyxp8x0dlpdt3k6cwng5pxj3jhsydzerkr0vd4msrxnuwnccdxlhdjar77j6lg0wypcc9uar5d2shs2z78ve",
			input: Address.Address{
				PaymentPart: []byte{148, 147, 49, 92, 217, 46, 181, 216, 196, 48, 78, 103, 183, 225, 106, 227, 109, 97, 211, 69, 2, 105, 70, 87, 129, 26, 44, 142},
				StakingPart: []byte{195, 123, 27, 93, 192, 102, 159, 29, 60, 97, 166, 253, 219, 46, 143, 222, 150, 190, 135, 184, 129, 198, 11, 206, 142, 141, 84, 47},
				Network:     Address.MAINNET,
				AddressType: Address.KEY_SCRIPT,
				HeaderByte:  0b00100001,
				Hrp:         "addr",
			},
		},
		"Valid SCRIPT_SCRIPT mainnet Address": {
			expected: "addr1x8phkx6acpnf78fuvxn0mkew3l0fd058hzquvz7w36x4gt7r0vd4msrxnuwnccdxlhdjar77j6lg0wypcc9uar5d2shskhj42g",
			input: Address.Address{
				PaymentPart: []byte{195, 123, 27, 93, 192, 102, 159, 29, 60, 97, 166, 253, 219, 46, 143, 222, 150, 190, 135, 184, 129, 198, 11, 206, 142, 141, 84, 47},
				StakingPart: []byte{195, 123, 27, 93, 192, 102, 159, 29, 60, 97, 166, 253, 219, 46, 143, 222, 150, 190, 135, 184, 129, 198, 11, 206, 142, 141, 84, 47},
				Network:     Address.MAINNET,
				AddressType: Address.SCRIPT_SCRIPT,
				HeaderByte:  0b00110001,
				Hrp:         "addr",
			},
		},
		"Valid KEY_NONE mainnet Address": {
			expected: "addr1vx2fxv2umyhttkxyxp8x0dlpdt3k6cwng5pxj3jhsydzers66hrl8",
			input: Address.Address{
				PaymentPart: []byte{148, 147, 49, 92, 217, 46, 181, 216, 196, 48, 78, 103, 183, 225, 106, 227, 109, 97, 211, 69, 2, 105, 70, 87, 129, 26, 44, 142},
				StakingPart: make([]byte, 0),
				Network:     Address.MAINNET,
				AddressType: Address.KEY_NONE,
				HeaderByte:  0b01100001,
				Hrp:         "addr",
			},
		},
		"Valid SCRIPT_NONE mainnet Address": {
			expected: "addr1w8phkx6acpnf78fuvxn0mkew3l0fd058hzquvz7w36x4gtcyjy7wx",
			input: Address.Address{
				PaymentPart: []byte{195, 123, 27, 93, 192, 102, 159, 29, 60, 97, 166, 253, 219, 46, 143, 222, 150, 190, 135, 184, 129, 198, 11, 206, 142, 141, 84, 47},
				StakingPart: make([]byte, 0),
				Network:     Address.MAINNET,
				AddressType: Address.SCRIPT_NONE,
				HeaderByte:  0b01110001,
				Hrp:         "addr",
			},
		},
		"Valid NONE_KEY mainnet Address": {
			expected: "stake1uyehkck0lajq8gr28t9uxnuvgcqrc6070x3k9r8048z8y5gh6ffgw",
			input: Address.Address{
				PaymentPart: make([]byte, 0),
				StakingPart: []byte{51, 123, 98, 207, 255, 100, 3, 160, 106, 58, 203, 195, 79, 140, 70, 0, 60, 105, 254, 121, 163, 98, 140, 239, 169, 196, 114, 81},
				Network:     Address.MAINNET,
				AddressType: Address.NONE_KEY,
				HeaderByte:  0b11100001,
				Hrp:         "stake",
			},
		},
		"Valid NONE_SCRIPT mainnet Address": {
			expected: "stake178phkx6acpnf78fuvxn0mkew3l0fd058hzquvz7w36x4gtcccycj5",
			input: Address.Address{
				PaymentPart: make([]byte, 0),
				StakingPart: []byte{195, 123, 27, 93, 192, 102, 159, 29, 60, 97, 166, 253, 219, 46, 143, 222, 150, 190, 135, 184, 129, 198, 11, 206, 142, 141, 84, 47},
				Network:     Address.MAINNET,
				AddressType: Address.NONE_SCRIPT,
				HeaderByte:  0b11110001,
				Hrp:         "stake",
			},
		},
		"Valid KEY_KEY testnet address": {
			expected: "addr_test1qz2fxv2umyhttkxyxp8x0dlpdt3k6cwng5pxj3jhsydzer3n0d3vllmyqwsx5wktcd8cc3sq835lu7drv2xwl2wywfgs68faae",
			input: Address.Address{
				PaymentPart: []byte{148, 147, 49, 92, 217, 46, 181, 216, 196, 48, 78, 103, 183, 225, 106, 227, 109, 97, 211, 69, 2, 105, 70, 87, 129, 26, 44, 142},
				StakingPart: []byte{51, 123, 98, 207, 255, 100, 3, 160, 106, 58, 203, 195, 79, 140, 70, 0, 60, 105, 254, 121, 163, 98, 140, 239, 169, 196, 114, 81},
				Network:     Address.TESTNET,
				AddressType: Address.KEY_KEY,
				HeaderByte:  0b00000000,
				Hrp:         "addr_test",
			},
		},
		"Valid SCRIPT_KEY testnet Address": {
			expected: "addr_test1zrphkx6acpnf78fuvxn0mkew3l0fd058hzquvz7w36x4gten0d3vllmyqwsx5wktcd8cc3sq835lu7drv2xwl2wywfgsxj90mg",
			input: Address.Address{
				PaymentPart: []byte{195, 123, 27, 93, 192, 102, 159, 29, 60, 97, 166, 253, 219, 46, 143, 222, 150, 190, 135, 184, 129, 198, 11, 206, 142, 141, 84, 47},
				StakingPart: []byte{51, 123, 98, 207, 255, 100, 3, 160, 106, 58, 203, 195, 79, 140, 70, 0, 60, 105, 254, 121, 163, 98, 140, 239, 169, 196, 114, 81},
				Network:     Address.TESTNET,
				AddressType: Address.SCRIPT_KEY,
				HeaderByte:  0b00010000,
				Hrp:         "addr_test",
			},
		},
		"Valid KEY_SCRIPT testnet Address": {
			expected: "addr_test1yz2fxv2umyhttkxyxp8x0dlpdt3k6cwng5pxj3jhsydzerkr0vd4msrxnuwnccdxlhdjar77j6lg0wypcc9uar5d2shsf5r8qx",
			input: Address.Address{
				PaymentPart: []byte{148, 147, 49, 92, 217, 46, 181, 216, 196, 48, 78, 103, 183, 225, 106, 227, 109, 97, 211, 69, 2, 105, 70, 87, 129, 26, 44, 142},
				StakingPart: []byte{195, 123, 27, 93, 192, 102, 159, 29, 60, 97, 166, 253, 219, 46, 143, 222, 150, 190, 135, 184, 129, 198, 11, 206, 142, 141, 84, 47},
				Network:     Address.TESTNET,
				AddressType: Address.KEY_SCRIPT,
				HeaderByte:  0b00100000,
				Hrp:         "addr_test",
			},
		},
		"Valid SCRIPT_SCRIPT testnet Address": {
			expected: "addr_test1xrphkx6acpnf78fuvxn0mkew3l0fd058hzquvz7w36x4gt7r0vd4msrxnuwnccdxlhdjar77j6lg0wypcc9uar5d2shs4p04xh",
			input: Address.Address{
				PaymentPart: []byte{195, 123, 27, 93, 192, 102, 159, 29, 60, 97, 166, 253, 219, 46, 143, 222, 150, 190, 135, 184, 129, 198, 11, 206, 142, 141, 84, 47},
				StakingPart: []byte{195, 123, 27, 93, 192, 102, 159, 29, 60, 97, 166, 253, 219, 46, 143, 222, 150, 190, 135, 184, 129, 198, 11, 206, 142, 141, 84, 47},
				Network:     Address.TESTNET,
				AddressType: Address.SCRIPT_SCRIPT,
				HeaderByte:  0b00110000,
				Hrp:         "addr_test",
			},
		},
		"Valid KEY_NONE testnet Address": {
			expected: "addr_test1vz2fxv2umyhttkxyxp8x0dlpdt3k6cwng5pxj3jhsydzerspjrlsz",
			input: Address.Address{
				PaymentPart: []byte{148, 147, 49, 92, 217, 46, 181, 216, 196, 48, 78, 103, 183, 225, 106, 227, 109, 97, 211, 69, 2, 105, 70, 87, 129, 26, 44, 142},
				StakingPart: make([]byte, 0),
				Network:     Address.TESTNET,
				AddressType: Address.KEY_NONE,
				HeaderByte:  0b01100000,
				Hrp:         "addr_test",
			},
		},
		"Valid SCRIPT_NONE testnet Address": {
			expected: "addr_test1wrphkx6acpnf78fuvxn0mkew3l0fd058hzquvz7w36x4gtcl6szpr",
			input: Address.Address{
				PaymentPart: []byte{195, 123, 27, 93, 192, 102, 159, 29, 60, 97, 166, 253, 219, 46, 143, 222, 150, 190, 135, 184, 129, 198, 11, 206, 142, 141, 84, 47},
				StakingPart: make([]byte, 0),
				Network:     Address.TESTNET,
				AddressType: Address.SCRIPT_NONE,
				HeaderByte:  0b01110000,
				Hrp:         "addr_test",
			},
		},
		"Valid NONE_KEY testnet Address": {
			expected: "stake_test1uqehkck0lajq8gr28t9uxnuvgcqrc6070x3k9r8048z8y5gssrtvn",
			input: Address.Address{
				PaymentPart: make([]byte, 0),
				StakingPart: []byte{51, 123, 98, 207, 255, 100, 3, 160, 106, 58, 203, 195, 79, 140, 70, 0, 60, 105, 254, 121, 163, 98, 140, 239, 169, 196, 114, 81},
				Network:     Address.TESTNET,
				AddressType: Address.NONE_KEY,
				HeaderByte:  0b11100000,
				Hrp:         "stake_test",
			},
		},
		"Valid NONE_SCRIPT testnet Address": {
			expected: "stake_test17rphkx6acpnf78fuvxn0mkew3l0fd058hzquvz7w36x4gtcljw6kf",
			input: Address.Address{
				PaymentPart: make([]byte, 0),
				StakingPart: []byte{195, 123, 27, 93, 192, 102, 159, 29, 60, 97, 166, 253, 219, 46, 143, 222, 150, 190, 135, 184, 129, 198, 11, 206, 142, 141, 84, 47},
				Network:     Address.TESTNET,
				AddressType: Address.NONE_SCRIPT,
				HeaderByte:  0b11110000,
				Hrp:         "stake_test",
			},
		},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			result := testCase.input.String()
			if result != testCase.expected {
				t.Errorf("\ntest: %v\nexpected: %v\nresult: %v", name, testCase.expected, result)
			}
		})
	}
}

func TestMarshalCbor(t *testing.T) {
	addr, _ := Address.DecodeAddress("addr1qx2fxv2umyhttkxyxp8x0dlpdt3k6cwng5pxj3jhsydzer3n0d3vllmyqwsx5wktcd8cc3sq835lu7drv2xwl2wywfgse35a3x")
	cbor, _ := addr.MarshalCBOR()
	encoded := hex.EncodeToString(cbor)
	if encoded != "5839019493315cd92eb5d8c4304e67b7e16ae36d61d34502694657811a2c8e337b62cfff6403a06a3acbc34f8c46003c69fe79a3628cefa9c47251" {
		t.Errorf("\nexpected: %v\nresult: %v", "5839019493315cd92eb5d8c4304e67b7e16ae36d61d34502694657811a2c8e337b62cfff6403a06a3acbc34f8c46003c69fe79a3628cefa9c47251", encoded)
	}
}

func TestUnmarshalCbor(t *testing.T) {
	cbor := "5839019493315cd92eb5d8c4304e67b7e16ae36d61d34502694657811a2c8e337b62cfff6403a06a3acbc34f8c46003c69fe79a3628cefa9c47251"
	decoded, _ := hex.DecodeString(cbor)
	addr := Address.Address{}
	addr.UnmarshalCBOR(decoded)
	if addr.String() != "addr1qx2fxv2umyhttkxyxp8x0dlpdt3k6cwng5pxj3jhsydzer3n0d3vllmyqwsx5wktcd8cc3sq835lu7drv2xwl2wywfgse35a3x" {
		t.Errorf("\nexpected: %v\nresult: %v", "addr1qx2fxv2umyhttkxyxp8x0dlpdt3k6cwng5pxj3jhsydzer3n0d3vllmyqwsx5wktcd8cc3sq835lu7drv2xwl2wywfgse35a3x", addr.String())
	}
}

func TestAddressFromBytes(t *testing.T) {
	addr, _ := Address.DecodeAddress("addr1qx2fxv2umyhttkxyxp8x0dlpdt3k6cwng5pxj3jhsydzer3n0d3vllmyqwsx5wktcd8cc3sq835lu7drv2xwl2wywfgse35a3x")
	newAddr := Address.WalletAddressFromBytes(addr.PaymentPart, addr.StakingPart, 0)
	if newAddr.String() != "addr1qx2fxv2umyhttkxyxp8x0dlpdt3k6cwng5pxj3jhsydzer3n0d3vllmyqwsx5wktcd8cc3sq835lu7drv2xwl2wywfgse35a3x" {
		t.Errorf("\nexpected: %v\nresult: %v", "addr1qx2fxv2umyhttkxyxp8x0dlpdt3k6cwng5pxj3jhsydzer3n0d3vllmyqwsx5wktcd8cc3sq835lu7drv2xwl2wywfgse35a3x", newAddr.String())
	}
	newAddr = Address.WalletAddressFromBytes(addr.PaymentPart, addr.StakingPart, 1)
	if newAddr.String() != "addr_test1qz2fxv2umyhttkxyxp8x0dlpdt3k6cwng5pxj3jhsydzer3n0d3vllmyqwsx5wktcd8cc3sq835lu7drv2xwl2wywfgs68faae" {
		t.Errorf("\nexpected: %v\nresult: %v", "addr_test1qz2fxv2umyhttkxyxp8x0dlpdt3k6cwng5pxj3jhsydzer3n0d3vllmyqwsx5wktcd8cc3sq835lu7drv2xwl2wywfgs68faae", newAddr.String())
	}
	newAddr = Address.WalletAddressFromBytes(addr.PaymentPart, nil, 0)
	if newAddr.String() != "addr1vx2fxv2umyhttkxyxp8x0dlpdt3k6cwng5pxj3jhsydzers66hrl8" {
		t.Errorf("\nexpected: %v\nresult: %v", "addr1vx2fxv2umyhttkxyxp8x0dlpdt3k6cwng5pxj3jhsydzers66hrl8", newAddr.String())
	}

	newAddr = Address.WalletAddressFromBytes(nil, addr.StakingPart, 0)
	if newAddr != nil {
		t.Errorf("\nexpected: %v\nresult: %v", nil, newAddr)
	}
}

func TestEquality(t *testing.T) {
	addr, _ := Address.DecodeAddress("addr1qx2fxv2umyhttkxyxp8x0dlpdt3k6cwng5pxj3jhsydzer3n0d3vllmyqwsx5wktcd8cc3sq835lu7drv2xwl2wywfgse35a3x")
	if !addr.Equal(&addr) {
		t.Errorf("\nexpected: %v\nresult: %v", true, addr.Equal(&addr))
	}
}

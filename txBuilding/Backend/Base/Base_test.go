package Base

import "testing"

const testAddress = "addr_test1vz5g0tr9z6tqflpsk4dguqce75zx0efmayflw0kl8xj" +
	"qu7c5n5xg7"

func TestParseAssetUnitRejectsInvalidInput(t *testing.T) {
	cases := []AddressAmount{
		{Unit: "abcd", Quantity: "1"},
		{
			Unit:     "00000000000000000000000000000000000000000000000000000000zz",
			Quantity: "1",
		},
	}

	for _, tc := range cases {
		t.Run(tc.Unit, func(t *testing.T) {
			output := Output{
				Address: testAddress,
				Amount: []AddressAmount{
					{Unit: "lovelace", Quantity: "1000000"},
					tc,
				},
			}
			if _, err := output.ToTransactionOutput(); err == nil {
				t.Fatal("expected invalid asset unit error")
			}
		})
	}
}

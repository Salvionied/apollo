package ogmios

import (
	"testing"

	"github.com/SundaeSwap-finance/ogmigo/v6/ouroboros/chainsync/num"
	"github.com/SundaeSwap-finance/ogmigo/v6/ouroboros/shared"
	"github.com/blinklabs-io/gouroboros/ledger/common"
)

func testAddress(t *testing.T) common.Address {
	t.Helper()
	var raw [57]byte
	raw[0] = 0x00
	raw[1] = 0xAA
	raw[29] = 0xBB
	addr, err := common.NewAddressFromBytes(raw[:])
	if err != nil {
		t.Fatal(err)
	}
	return addr
}

func TestSharedValueToUtxoRejectsNegativeLovelace(t *testing.T) {
	value := shared.Value{
		shared.AdaPolicy: {
			shared.AdaAsset: num.Int64(-1),
		},
	}
	if _, err := sharedValueToUtxo(common.Blake2b256{}, 0, value, testAddress(t)); err == nil {
		t.Fatal("expected negative lovelace error")
	}
}

func TestSharedValueToUtxoRejectsNegativeAssetQuantity(t *testing.T) {
	value := shared.Value{
		shared.AdaPolicy: {
			shared.AdaAsset: num.Int64(1000000),
		},
		"00000000000000000000000000000000000000000000000000000001": {
			"544f4b454e": num.Int64(-1),
		},
	}
	if _, err := sharedValueToUtxo(common.Blake2b256{}, 0, value, testAddress(t)); err == nil {
		t.Fatal("expected negative asset quantity error")
	}
}

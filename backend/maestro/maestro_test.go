package maestro

import (
	"math"
	"testing"

	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/maestro-org/go-sdk/models"
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

func TestMaestroUtxoToCommonRejectsInvalidAssetUnit(t *testing.T) {
	addr := testAddress(t)
	raw := models.Utxo{
		TxHash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Index:  0,
		Assets: []models.Asset{
			{Unit: "lovelace", Amount: 1000000},
			{Unit: "abcd", Amount: 1},
		},
	}
	if _, err := maestroUtxoToCommon(raw, addr); err == nil {
		t.Fatal("expected invalid asset unit error")
	}
}

func TestMaestroUtxoToCommonRejectsNegativeAssetQuantity(t *testing.T) {
	addr := testAddress(t)
	raw := models.Utxo{
		TxHash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Index:  0,
		Assets: []models.Asset{
			{Unit: "lovelace", Amount: 1000000},
			{Unit: "00000000000000000000000000000000000000000000000000000001", Amount: -1},
		},
	}
	if _, err := maestroUtxoToCommon(raw, addr); err == nil {
		t.Fatal("expected negative asset quantity error")
	}
}

func TestMaestroUtxoToCommonRejectsOutputIndexOverflow(t *testing.T) {
	addr := testAddress(t)
	raw := models.Utxo{
		TxHash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Index:  int64(math.MaxUint32) + 1,
		Assets: []models.Asset{{Unit: "lovelace", Amount: 1000000}},
	}
	if _, err := maestroUtxoToCommon(raw, addr); err == nil {
		t.Fatal("expected output index overflow error")
	}
}

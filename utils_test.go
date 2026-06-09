package apollo

import (
	"math/big"
	"testing"

	"github.com/blinklabs-io/gouroboros/ledger/babbage"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/mary"
	"github.com/blinklabs-io/gouroboros/ledger/shelley"
)

func makeTestUtxo(t *testing.T, txHash common.Blake2b256, index uint32, lovelace uint64) common.Utxo {
	t.Helper()
	addr := testAddress(t)
	input := shelley.ShelleyTransactionInput{
		TxId:        txHash,
		OutputIndex: index,
	}
	output := babbage.BabbageTransactionOutput{
		OutputAddress: addr,
		OutputAmount: mary.MaryTransactionOutputValue{
			Amount: lovelace,
		},
	}
	return common.Utxo{
		Id:     input,
		Output: &output,
	}
}

func TestSortUtxos(t *testing.T) {
	var hash1, hash2, hash3 common.Blake2b256
	hash1[0] = 1
	hash2[0] = 2
	hash3[0] = 3

	utxos := []common.Utxo{
		makeTestUtxo(t, hash1, 0, 1_000_000),
		makeTestUtxo(t, hash2, 0, 5_000_000),
		makeTestUtxo(t, hash3, 0, 3_000_000),
	}

	sorted := SortUtxos(utxos)
	if len(sorted) != 3 {
		t.Fatalf("expected 3 utxos, got %d", len(sorted))
	}

	// Should be sorted by descending amount (ADA-only first)
	amt0 := sorted[0].Output.Amount()
	amt1 := sorted[1].Output.Amount()
	if amt0.Cmp(amt1) < 0 {
		t.Error("expected descending order")
	}
}

func TestSortUtxosWithAssets(t *testing.T) {
	var hash1, hash2 common.Blake2b256
	hash1[0] = 1
	hash2[0] = 2

	addr := testAddress(t)

	// UTxO without assets
	utxo1 := makeTestUtxo(t, hash1, 0, 2_000_000)

	// UTxO with assets
	ma := testMultiAsset(1, "token", 100)
	output2 := babbage.BabbageTransactionOutput{
		OutputAddress: addr,
		OutputAmount: mary.MaryTransactionOutputValue{
			Amount: 3_000_000,
			Assets: ma,
		},
	}
	utxo2 := common.Utxo{
		Id: shelley.ShelleyTransactionInput{
			TxId:        hash2,
			OutputIndex: 0,
		},
		Output: &output2,
	}

	sorted := SortUtxos([]common.Utxo{utxo2, utxo1})
	// ADA-only UTxOs should come first
	if sorted[0].Output.Assets() != nil {
		t.Error("expected ADA-only UTxO first")
	}
	if sorted[1].Output.Assets() == nil {
		t.Error("expected UTxO with assets second")
	}
}

func TestSortInputs(t *testing.T) {
	var hash1, hash2 common.Blake2b256
	hash1[0] = 0xff
	hash2[0] = 0x01

	utxos := []common.Utxo{
		makeTestUtxo(t, hash1, 0, 1_000_000),
		makeTestUtxo(t, hash2, 0, 2_000_000),
	}

	sorted := SortInputs(utxos)
	if len(sorted) != 2 {
		t.Fatalf("expected 2, got %d", len(sorted))
	}
	// hash2 (0x01...) should come before hash1 (0xff...)
	firstAmt := sorted[0].Output.Amount()
	if firstAmt == nil || firstAmt.Cmp(big.NewInt(2_000_000)) != 0 {
		t.Error("expected hash2 utxo first (lower tx hash)")
	}
}

func TestSortInputsSameHash(t *testing.T) {
	var hash common.Blake2b256
	hash[0] = 0x01

	utxos := []common.Utxo{
		makeTestUtxo(t, hash, 5, 1_000_000),
		makeTestUtxo(t, hash, 1, 2_000_000),
	}

	sorted := SortInputs(utxos)
	// Index 1 should come before index 5
	if sorted[0].Id.Index() != 1 {
		t.Errorf("expected index 1 first, got %d", sorted[0].Id.Index())
	}
}

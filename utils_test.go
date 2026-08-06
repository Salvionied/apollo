package apollo

import (
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
	"sort"
	"strings"
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

func TestSortUtxosDoesNotPanicOnMalformedInput(t *testing.T) {
	valid := makeTestUtxo(t, common.Blake2b256{1}, 0, 1_000_000)
	var nilOutput *babbage.BabbageTransactionOutput
	malformed := common.Utxo{Id: valid.Id, Output: nilOutput}

	sorted := SortUtxos([]common.Utxo{malformed, valid})

	if len(sorted) != 2 || validateUtxo(sorted[0]) != nil {
		t.Fatal("expected valid UTxO to sort before malformed UTxO")
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

func TestSortInputsDoesNotPanicOnMalformedInput(t *testing.T) {
	valid := makeTestUtxo(t, common.Blake2b256{1}, 0, 1_000_000)
	var nilInput *shelley.ShelleyTransactionInput
	malformed := common.Utxo{Id: nilInput, Output: valid.Output}

	sorted := SortInputs([]common.Utxo{malformed, valid})

	if len(sorted) != 2 || validateUtxo(sorted[0]) != nil {
		t.Fatal("expected valid input to sort before malformed input")
	}
}

func TestUtxoRefPreservesUint32Index(t *testing.T) {
	utxo := makeTestUtxo(t, common.Blake2b256{1}, math.MaxUint32, 1_000_000)
	if got := utxoRef(utxo); !strings.HasSuffix(got, "#4294967295") {
		t.Fatalf("UTxO ref = %q, want uint32 index preserved", got)
	}
}

// BenchmarkSortUtxos and BenchmarkSortInputs cover the cost the sort keys
// were hoisted out of the comparators for: validating and hex-encoding once
// per element instead of once per comparison.
func BenchmarkSortUtxos(b *testing.B) {
	pool, _ := sweepSelectionPool(b, 10_000)
	for i := 0; i < b.N; i++ {
		SortUtxos(pool)
	}
}

func BenchmarkSortInputs(b *testing.B) {
	pool, _ := sweepSelectionPool(b, 10_000)
	for i := 0; i < b.N; i++ {
		SortInputs(pool)
	}
}

// referenceSortUtxos is SortUtxos as it was before the per-element sort keys
// were hoisted out of the comparator, kept to prove the hoisting changed no
// ordering. Do not optimize: it is the oracle, not production code.
func referenceSortUtxos(utxos []common.Utxo) []common.Utxo {
	res := make([]common.Utxo, len(utxos))
	copy(res, utxos)
	sort.SliceStable(res, func(i, j int) bool {
		iValid := validateUtxo(res[i]) == nil
		jValid := validateUtxo(res[j]) == nil
		if iValid != jValid {
			return iValid
		}
		if !iValid {
			return false
		}
		iHasAssets := res[i].Output.Assets() != nil
		jHasAssets := res[j].Output.Assets() != nil
		if !iHasAssets && !jHasAssets {
			iAmt := res[i].Output.Amount()
			jAmt := res[j].Output.Amount()
			if iAmt != nil && jAmt != nil {
				return iAmt.Cmp(jAmt) > 0
			}
			return false
		}
		if iHasAssets && jHasAssets {
			iAmt := res[i].Output.Amount()
			jAmt := res[j].Output.Amount()
			if iAmt != nil && jAmt != nil {
				return iAmt.Cmp(jAmt) > 0
			}
			return false
		}
		return jHasAssets
	})
	return res
}

// referenceSortInputs is the pre-hoist SortInputs; see referenceSortUtxos.
func referenceSortInputs(inputs []common.Utxo) []common.Utxo {
	sorted := make([]common.Utxo, len(inputs))
	copy(sorted, inputs)
	sort.SliceStable(sorted, func(i, j int) bool {
		iValid := validateUtxo(sorted[i]) == nil
		jValid := validateUtxo(sorted[j]) == nil
		if iValid != jValid {
			return iValid
		}
		if !iValid {
			return false
		}
		iId := hex.EncodeToString(sorted[i].Id.Id().Bytes())
		jId := hex.EncodeToString(sorted[j].Id.Id().Bytes())
		if iId != jId {
			return iId < jId
		}
		return sorted[i].Id.Index() < sorted[j].Id.Index()
	})
	return sorted
}

// sortIdentity identifies a UTxO by its output's address, which is safe to
// read for the malformed entries too and unique per element in the pool the
// test below builds.
func sortIdentity(utxo common.Utxo) string {
	return fmt.Sprintf("%p/%p", utxo.Id, utxo.Output)
}

// orderingPool builds a slice exercising every branch of both comparators:
// ADA-only and asset-carrying UTxOs, repeated amounts, repeated transaction
// ids with differing indexes, and malformed entries with a nil input or a nil
// output. Every element has a distinct identity so any reordering shows up.
func orderingPool(t *testing.T) []common.Utxo {
	t.Helper()
	amounts := []uint64{1_000_000, 2_000_000, 3_000_000}
	var pool []common.Utxo
	// The transaction ids straddle the hex digit/letter boundary, where a
	// lexicographic hex comparison and a raw byte comparison could disagree
	// if the encodings were not of equal width.
	for _, hash := range []byte{0x01, 0x0f, 0xa0, 0xff} {
		for index := uint32(0); index < 2; index++ {
			for _, amount := range amounts {
				pool = append(
					pool,
					makeSelectorUtxo(t, hash, index, amount, nil),
					makeSelectorUtxo(
						t, hash, index, amount,
						makeTestAssets(0xAA, "tokenA", 1),
					),
				)
			}
		}
	}
	var nilInput *shelley.ShelleyTransactionInput
	var nilOutput *babbage.BabbageTransactionOutput
	valid := makeSelectorUtxo(t, 0x09, 0, 4_000_000, nil)
	pool = append(pool,
		common.Utxo{Id: nilInput, Output: valid.Output},
		common.Utxo{Id: valid.Id, Output: nilOutput},
	)
	return pool
}

// TestSortOrderingMatchesPreHoistReference proves hoisting validation and hex
// encoding out of the sort comparators left both orderings byte-identical,
// including the stable order of tied elements, for every rotation of a pool
// full of ties.
func TestSortOrderingMatchesPreHoistReference(t *testing.T) {
	pool := orderingPool(t)
	sorts := []struct {
		name      string
		got       func([]common.Utxo) []common.Utxo
		reference func([]common.Utxo) []common.Utxo
	}{
		{"SortUtxos", SortUtxos, referenceSortUtxos},
		{"SortInputs", SortInputs, referenceSortInputs},
	}
	for _, s := range sorts {
		t.Run(s.name, func(t *testing.T) {
			for shift := range pool {
				input := make([]common.Utxo, 0, len(pool))
				input = append(input, pool[shift:]...)
				input = append(input, pool[:shift]...)

				got := s.got(input)
				want := s.reference(input)
				if len(got) != len(want) {
					t.Fatalf(
						"shift %d: got %d UTxOs, want %d",
						shift, len(got), len(want),
					)
				}
				for i := range want {
					if sortIdentity(got[i]) != sortIdentity(want[i]) {
						t.Fatalf(
							"shift %d: ordering differs at %d",
							shift, i,
						)
					}
				}
			}
		})
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

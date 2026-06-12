package apollo

import (
	"encoding/binary"
	"fmt"
	"math/big"
	"math/rand"
	"testing"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/babbage"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/mary"
	"github.com/blinklabs-io/gouroboros/ledger/shelley"
)

// benchAddress parses the shared test address once for pool generation.
func benchAddress(tb testing.TB) common.Address {
	tb.Helper()
	addr, err := common.NewAddress(validTestAddrBech32)
	if err != nil {
		tb.Fatal(err)
	}
	return addr
}

// benchUtxo builds a UTxO with a unique tx hash derived from seq.
func benchUtxo(addr common.Address, seq uint64, lovelace uint64, assets *common.MultiAsset[common.MultiAssetTypeOutput]) common.Utxo {
	var txHash common.Blake2b256
	binary.BigEndian.PutUint64(txHash[:8], seq)
	return common.Utxo{
		Id: shelley.ShelleyTransactionInput{
			TxId:        txHash,
			OutputIndex: 0,
		},
		Output: &babbage.BabbageTransactionOutput{
			OutputAddress: addr,
			OutputAmount: mary.MaryTransactionOutputValue{
				Amount: lovelace,
				Assets: assets,
			},
		},
	}
}

func benchPolicy(n uint64) common.Blake2b224 {
	var policy common.Blake2b224
	binary.BigEndian.PutUint64(policy[:8], n)
	return policy
}

func benchAsset(policyNum uint64, qty int64) *common.MultiAsset[common.MultiAssetTypeOutput] {
	return MultiAssetFromMap(map[common.Blake2b224]map[cbor.ByteString]*big.Int{
		benchPolicy(policyNum): {cbor.NewByteString([]byte(fmt.Sprintf("tok%02d", policyNum))): big.NewInt(qty)},
	})
}

type benchScenario struct {
	name   string
	pool   []common.Utxo
	target Value
}

// benchScenarios generates deterministic pools (fixed PRNG seed) for each
// benchmark scenario described in the design doc.
func benchScenarios(tb testing.TB) []benchScenario {
	tb.Helper()
	addr := benchAddress(tb)
	var seq uint64

	nextSeq := func() uint64 {
		seq++
		return seq
	}

	adaPool := func(rng *rand.Rand, n int) []common.Utxo {
		pool := make([]common.Utxo, 0, n)
		for i := 0; i < n; i++ {
			lovelace := 1_000_000 + rng.Uint64()%99_000_000 // 1-100 ADA
			pool = append(pool, benchUtxo(addr, nextSeq(), lovelace, nil))
		}
		return pool
	}

	// Multi-asset pool: 30% of UTxOs carry 1-3 assets from 50 policies.
	multiAssetPool := func(rng *rand.Rand, n int) []common.Utxo {
		pool := make([]common.Utxo, 0, n)
		for i := 0; i < n; i++ {
			lovelace := 2_000_000 + rng.Uint64()%18_000_000 // 2-20 ADA
			var assets *common.MultiAsset[common.MultiAssetTypeOutput]
			if rng.Intn(10) < 3 {
				numAssets := 1 + rng.Intn(3)
				for j := 0; j < numAssets; j++ {
					a := benchAsset(rng.Uint64()%50, 1+rng.Int63n(1000))
					if assets == nil {
						assets = a
					} else {
						assets.Add(a)
					}
				}
			}
			pool = append(pool, benchUtxo(addr, nextSeq(), lovelace, assets))
		}
		return pool
	}

	dustPool := func(rng *rand.Rand, n int) []common.Utxo {
		pool := make([]common.Utxo, 0, n)
		for i := 0; i < n; i++ {
			var lovelace uint64
			if rng.Intn(10) < 9 {
				lovelace = 1_000_000 + rng.Uint64()%1_000_000 // 1-2 ADA dust
			} else {
				lovelace = 50_000_000 + rng.Uint64()%50_000_000 // 50-100 ADA
			}
			pool = append(pool, benchUtxo(addr, nextSeq(), lovelace, nil))
		}
		return pool
	}

	maPool := multiAssetPool(rand.New(rand.NewSource(42)), 1000)
	// Target 10% of the pool's actual holdings of 5 policies, so the target
	// is always feasible.
	maTarget := NewSimpleValue(20_000_000)
	for p := uint64(0); p < 5; p++ {
		policy := benchPolicy(p)
		total := big.NewInt(0)
		for _, u := range maPool {
			if assets := u.Output.Assets(); assets != nil {
				if qty := assets.Asset(policy, []byte(fmt.Sprintf("tok%02d", p))); qty != nil {
					total.Add(total, qty)
				}
			}
		}
		if total.Sign() > 0 {
			qty := new(big.Int).Div(total, big.NewInt(10))
			if qty.Sign() == 0 {
				qty = big.NewInt(1)
			}
			ta := benchAsset(p, qty.Int64())
			if maTarget.Assets == nil {
				maTarget.Assets = ta
			} else {
				maTarget.Assets.Add(ta)
			}
		}
	}

	return []benchScenario{
		{name: "Ada100", pool: adaPool(rand.New(rand.NewSource(1)), 100), target: NewSimpleValue(50_000_000)},
		{name: "Ada10k", pool: adaPool(rand.New(rand.NewSource(2)), 10_000), target: NewSimpleValue(500_000_000)},
		{name: "MultiAsset1k", pool: maPool, target: maTarget},
		{name: "DustHeavy5k", pool: dustPool(rand.New(rand.NewSource(3)), 5_000), target: NewSimpleValue(200_000_000)},
	}
}

func BenchmarkCoinSelection(b *testing.B) {
	selectors := []CoinSelector{&LargestFirstSelector{}, &MACSSelector{}}
	for _, sc := range benchScenarios(b) {
		for _, sel := range selectors {
			b.Run(sc.name+"/"+sel.Name(), func(b *testing.B) {
				var totalInputs, totalExcess float64
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					selected, err := sel.Select(sc.pool, sc.target)
					if err != nil {
						b.Fatalf("Select failed: %v", err)
					}
					totalInputs += float64(len(selected))
					var coinSum uint64
					for _, u := range selected {
						coinSum += u.Output.Amount().Uint64()
					}
					totalExcess += float64(coinSum - sc.target.Coin)
				}
				b.ReportMetric(totalInputs/float64(b.N), "inputs/op")
				b.ReportMetric(totalExcess/float64(b.N)/1e6, "excessAda/op")
			})
		}
	}
}

// TestCoinSelectionComparison runs a multi-round wallet simulation mirroring
// the MACS paper's evaluation: each round pays a target from the pool,
// returns the change as a new UTxO, and adds small deposits. It logs pool
// health and selection quality metrics for both algorithms.
func TestCoinSelectionComparison(t *testing.T) {
	const rounds = 200
	addr := benchAddress(t)

	variants := []struct {
		label string
		sel   CoinSelector
	}{
		{"largest-first", &LargestFirstSelector{}},
		{"macs-pure", &MACSSelector{}},
		{"macs-sweep", NewMACSSelector()},
	}
	for _, v := range variants {
		sel := v.sel
		t.Run(v.label, func(t *testing.T) {
			rng := rand.New(rand.NewSource(7))
			var seq uint64
			nextSeq := func() uint64 {
				seq++
				return seq
			}

			// Starting pool: 20 mixed UTxOs plus two token holdings.
			pool := make([]common.Utxo, 0, 64)
			for i := 0; i < 20; i++ {
				lovelace := 5_000_000 + rng.Uint64()%45_000_000 // 5-50 ADA
				pool = append(pool, benchUtxo(addr, nextSeq(), lovelace, nil))
			}
			pool = append(pool,
				benchUtxo(addr, nextSeq(), 2_000_000, benchAsset(0, 10_000)),
				benchUtxo(addr, nextSeq(), 2_000_000, benchAsset(1, 10_000)),
			)

			var totalInputs, totalChange, maxPool float64
			var dustRounds, poolSizeSum int
			for r := uint64(0); r < rounds; r++ {
				target := NewSimpleValue(2_000_000 + rng.Uint64()%6_000_000) // 2-8 ADA
				if r%4 == 3 {
					// Every fourth round also pays out tokens if the pool has them.
					policy := benchPolicy(r % 2)
					name := []byte(fmt.Sprintf("tok%02d", r%2))
					total := big.NewInt(0)
					for _, u := range pool {
						if assets := u.Output.Assets(); assets != nil {
							if qty := assets.Asset(policy, name); qty != nil && qty.Sign() > 0 {
								total.Add(total, qty)
							}
						}
					}
					if total.Sign() > 0 {
						qty := new(big.Int).Div(total, big.NewInt(20))
						if qty.Sign() == 0 {
							qty = big.NewInt(1)
						}
						target.Assets = benchAsset(r%2, qty.Int64())
					}
				}

				selected, err := sel.Select(pool, target)
				if err != nil {
					t.Fatalf("round %d: Select failed: %v", r, err)
				}
				got := sumSelected(t, selected)
				if !got.GreaterOrEqual(target) {
					t.Fatalf("round %d: selection does not cover target", r)
				}

				// Remove selected UTxOs from the pool.
				usedRefs := make(map[string]bool, len(selected))
				for _, u := range selected {
					usedRefs[utxoRef(u)] = true
				}
				kept := pool[:0]
				for _, u := range pool {
					if !usedRefs[utxoRef(u)] {
						kept = append(kept, u)
					}
				}
				pool = kept

				// Return change as a single new UTxO.
				change, err := got.Sub(target)
				if err != nil {
					t.Fatalf("round %d: change underflow: %v", r, err)
				}
				if change.Coin > 0 || change.HasAssets() {
					changeAssets, err := normalizeChangeAssets(change.Assets)
					if err != nil {
						t.Fatalf("round %d: %v", r, err)
					}
					pool = append(pool, benchUtxo(addr, nextSeq(), change.Coin, changeAssets))
				}

				// Three deposits per round; one in five is sub-ADA dust.
				for d := 0; d < 3; d++ {
					var lovelace uint64
					if rng.Intn(5) == 0 {
						lovelace = 500_000 + rng.Uint64()%500_000 // 0.5-1 ADA
					} else {
						lovelace = 1_000_000 + rng.Uint64()%2_000_000 // 1-3 ADA
					}
					pool = append(pool, benchUtxo(addr, nextSeq(), lovelace, nil))
				}

				totalInputs += float64(len(selected))
				totalChange += float64(change.Coin) / 1e6
				poolSizeSum += len(pool)
				if float64(len(pool)) > maxPool {
					maxPool = float64(len(pool))
				}
				dust := 0
				for _, u := range pool {
					if u.Output.Amount().Uint64() < 1_000_000 {
						dust++
					}
				}
				if dust > 0 {
					dustRounds++
				}
			}

			dustFinal := 0
			for _, u := range pool {
				if u.Output.Amount().Uint64() < 1_000_000 {
					dustFinal++
				}
			}
			t.Logf("%s: avg inputs/tx=%.2f avg change=%.2f ADA avg pool=%d max pool=%.0f final pool=%d final dust=%d (dust present in %d/%d rounds)",
				v.label, totalInputs/rounds, totalChange/rounds, poolSizeSum/rounds, maxPool, len(pool), dustFinal, dustRounds, rounds)
		})
	}
}

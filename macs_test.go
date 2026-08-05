package apollo

import (
	"strconv"
	"testing"

	"github.com/blinklabs-io/gouroboros/ledger/common"
)

func TestMACSSelectorConformance(t *testing.T) {
	runSelectorConformance(t, func() CoinSelector { return &MACSSelector{} })
}

func TestMACSSelectorDefaultConformance(t *testing.T) {
	runSelectorConformance(t, func() CoinSelector { return NewMACSSelector() })
}

func TestMACSSelectorName(t *testing.T) {
	if name := (&MACSSelector{}).Name(); name != "macs" {
		t.Errorf("expected name macs, got %q", name)
	}
}

// TestMACSPrefersNearAverageUtxo pins the core MACS priority behavior:
// P(u,c) = v(u,c) / (|v(u,c) - avg| + 1) favors a UTxO close to the pool
// average over a large outlier, keeping the pool diverse and change small.
func TestMACSPrefersNearAverageUtxo(t *testing.T) {
	pool := []common.Utxo{
		makeSelectorUtxo(t, 0x01, 0, 10_000_000, nil),
		makeSelectorUtxo(t, 0x02, 0, 10_000_000, nil),
		makeSelectorUtxo(t, 0x03, 0, 10_000_000, nil),
		makeSelectorUtxo(t, 0x04, 0, 10_000_000, nil),
		makeSelectorUtxo(t, 0x05, 0, 12_000_000, nil),
		makeSelectorUtxo(t, 0x06, 0, 50_000_000, nil),
	}
	// Pool average is 17 ADA; the 12 ADA UTxO has the highest priority
	// (12/(5M+1) beats 50/(33M+1) and 10/(7M+1)).
	selected, err := (&MACSSelector{}).Select(
		t.Context(), pool, NewSimpleValue(5_000_000),
	)
	if err != nil {
		t.Fatalf("Select failed: %v", err)
	}
	if len(selected) != 1 {
		t.Fatalf("expected 1 input, got %d", len(selected))
	}
	if got := selected[0].Output.Amount().Uint64(); got != 12_000_000 {
		t.Errorf("expected the near-average 12 ADA UTxO, got %d lovelace", got)
	}
}

// TestMACSPrefersSingleSufficientUtxo pins the fix for the dust trap in the
// priority formula: avg(S,c) is a pool-wide mean, so a pool of many dust
// UTxOs pulls it down next to the dust and every dust UTxO then outscores the
// one UTxO that covers the whole target. A candidate that closes the deficit
// on its own must be preferred instead, at any dust count.
func TestMACSPrefersSingleSufficientUtxo(t *testing.T) {
	variants := []struct {
		label     string
		sel       *MACSSelector
		maxInputs int
	}{
		{label: "pure", sel: &MACSSelector{}, maxInputs: 1},
		// The default selector may also sweep dust, but nothing more.
		{
			label:     "default",
			sel:       NewMACSSelector(),
			maxInputs: 1 + NewMACSSelector().MaxDustInputs,
		},
	}
	for _, dust := range []int{10, 200, 1000} {
		for _, v := range variants {
			name := v.label + "/" + strconv.Itoa(dust)
			t.Run(name, func(t *testing.T) {
				pool := make([]common.Utxo, 0, dust+1)
				for i := 0; i < dust; i++ {
					pool = append(pool, makeSelectorUtxo(
						t, 0x10, uint32(i), 1_000_000, nil,
					))
				}
				sufficient := makeSelectorUtxo(
					t, 0x20, 0, 100_000_000, nil,
				)
				pool = append(pool, sufficient)

				target := NewSimpleValue(90_000_000)
				selected, err := v.sel.Select(t.Context(), pool, target)
				if err != nil {
					t.Fatalf("Select failed: %v", err)
				}
				if !sumSelected(t, selected).GreaterOrEqual(target) {
					t.Fatal("selection does not cover target")
				}
				if len(selected) > v.maxInputs {
					t.Errorf(
						"selected %d inputs for a target one UTxO covers, want at most %d",
						len(selected), v.maxInputs,
					)
				}
				found := false
				for _, u := range selected {
					if utxoRef(u) == utxoRef(sufficient) {
						found = true
					}
				}
				if !found {
					t.Error("the single sufficient UTxO was not selected")
				}
			})
		}
	}
}

// TestMACSStillPrefersMidRangeOverLargest guards the dust fix from
// degenerating MACS into largest-first: priority still decides between the
// UTxOs that cover the target alone, so the closer of the two to the pool
// average wins even though largest-first would take the biggest.
func TestMACSStillPrefersMidRangeOverLargest(t *testing.T) {
	pool := make([]common.Utxo, 0, 202)
	for i := 0; i < 200; i++ {
		pool = append(pool, makeSelectorUtxo(t, 0x10, uint32(i), 1_000_000, nil))
	}
	midRange := makeSelectorUtxo(t, 0x20, 0, 30_000_000, nil)
	largest := makeSelectorUtxo(t, 0x30, 0, 100_000_000, nil)
	pool = append(pool, midRange, largest)

	// The pool average is 1.63 ADA, so dust has the highest priority of all
	// but cannot cover 20 ADA alone; of the two that can, 30 ADA scores
	// 30/28.37 against 100 ADA's 100/98.37.
	selected, err := (&MACSSelector{}).Select(
		t.Context(), pool, NewSimpleValue(20_000_000),
	)
	if err != nil {
		t.Fatalf("Select failed: %v", err)
	}
	if len(selected) != 1 {
		t.Fatalf("expected 1 input, got %d", len(selected))
	}
	if utxoRef(selected[0]) != utxoRef(midRange) {
		t.Errorf(
			"expected the mid-range 30 ADA UTxO, got %s",
			utxoRef(selected[0]),
		)
	}
}

// TestMACSCoversDeficitAfterAssetPickTookTheLargest exercises the path where
// the class's largest holder has already gone to an earlier asset class, so
// no single candidate can close the coin deficit and the remainder has to be
// accumulated before a closer finishes the job.
func TestMACSCoversDeficitAfterAssetPickTookTheLargest(t *testing.T) {
	tokenHolder := makeSelectorUtxo(
		t, 0x01, 0, 100_000_000, makeTestAssets(0xAA, "tokenA", 50),
	)
	pool := []common.Utxo{
		tokenHolder,
		makeSelectorUtxo(t, 0x02, 0, 30_000_000, nil),
		makeSelectorUtxo(t, 0x03, 0, 30_000_000, nil),
		makeSelectorUtxo(t, 0x04, 0, 30_000_000, nil),
	}
	target := NewValue(160_000_000, makeTestAssets(0xAA, "tokenA", 50))

	selected, err := (&MACSSelector{}).Select(t.Context(), pool, target)
	if err != nil {
		t.Fatalf("Select failed: %v", err)
	}
	if !sumSelected(t, selected).GreaterOrEqual(target) {
		t.Fatal("selection does not cover target")
	}
	if len(selected) != 3 {
		t.Fatalf("expected 3 inputs, got %d", len(selected))
	}
	if utxoRef(selected[0]) != utxoRef(tokenHolder) {
		t.Errorf(
			"expected the token holder first, got %s",
			utxoRef(selected[0]),
		)
	}
}

// TestMACSMultiAssetSingleInput verifies MACS covers a joint coin+asset
// target with the asset-carrying UTxO alone instead of draining ADA-only
// UTxOs first like largest-first does.
func TestMACSMultiAssetSingleInput(t *testing.T) {
	assetUtxo := makeSelectorUtxo(t, 0x01, 0, 2_000_000, makeTestAssets(0xAA, "tokenA", 100))
	pool := []common.Utxo{
		assetUtxo,
		makeSelectorUtxo(t, 0x02, 0, 10_000_000, nil),
		makeSelectorUtxo(t, 0x03, 0, 2_000_000, makeTestAssets(0xAA, "tokenA", 10)),
	}
	target := NewValue(1_000_000, makeTestAssets(0xAA, "tokenA", 50))
	selected, err := (&MACSSelector{}).Select(t.Context(), pool, target)
	if err != nil {
		t.Fatalf("Select failed: %v", err)
	}
	if len(selected) != 1 {
		t.Fatalf("expected 1 input covering coin and asset, got %d", len(selected))
	}
	if utxoRef(selected[0]) != utxoRef(assetUtxo) {
		t.Errorf("expected the 100-tokenA UTxO, got %s", utxoRef(selected[0]))
	}
}

// TestMACSDustSweep verifies the bounded dust sweep: with sweeping enabled,
// a selection picks up extra sub-threshold ADA-only UTxOs (smallest first, up
// to the configured cap) so dust does not accumulate in the wallet's pool.
func TestMACSDustSweep(t *testing.T) {
	dust1 := makeSelectorUtxo(t, 0x01, 0, 400_000, nil)
	dust2 := makeSelectorUtxo(t, 0x02, 0, 600_000, nil)
	dust3 := makeSelectorUtxo(t, 0x03, 0, 800_000, nil)
	tokenDust := makeSelectorUtxo(t, 0x04, 0, 500_000, makeTestAssets(0xAA, "tokenA", 1))
	big1 := makeSelectorUtxo(t, 0x05, 0, 10_000_000, nil)
	big2 := makeSelectorUtxo(t, 0x06, 0, 11_000_000, nil)
	pool := []common.Utxo{dust1, dust2, dust3, tokenDust, big1, big2}

	sel := &MACSSelector{DustThreshold: 1_000_000, MaxDustInputs: 2}
	selected, err := sel.Select(
		t.Context(), pool, NewSimpleValue(5_000_000),
	)
	if err != nil {
		t.Fatalf("Select failed: %v", err)
	}

	got := make(map[string]bool, len(selected))
	for _, u := range selected {
		got[utxoRef(u)] = true
	}
	// One regular input covers the target; the two smallest ADA-only dust
	// UTxOs ride along. Token-carrying dust is never swept.
	if len(selected) != 3 {
		t.Fatalf("expected 3 inputs (1 cover + 2 dust), got %d", len(selected))
	}
	if !got[utxoRef(dust1)] || !got[utxoRef(dust2)] {
		t.Error("expected the two smallest dust UTxOs to be swept")
	}
	if got[utxoRef(tokenDust)] {
		t.Error("token-carrying dust must not be swept")
	}
}

// TestMACSDustSweepDisabledByDefault pins that the zero-value selector runs
// the pure algorithm with no sweeping.
func TestMACSDustSweepDisabledByDefault(t *testing.T) {
	pool := []common.Utxo{
		makeSelectorUtxo(t, 0x01, 0, 400_000, nil),
		makeSelectorUtxo(t, 0x02, 0, 10_000_000, nil),
	}
	selected, err := (&MACSSelector{}).Select(
		t.Context(), pool, NewSimpleValue(5_000_000),
	)
	if err != nil {
		t.Fatalf("Select failed: %v", err)
	}
	if len(selected) != 1 {
		t.Fatalf("expected 1 input with sweeping disabled, got %d", len(selected))
	}
}

// TestNewMACSSelectorDefaults pins the constructor's sweeping defaults.
func TestNewMACSSelectorDefaults(t *testing.T) {
	sel := NewMACSSelector()
	if sel.DustThreshold != 1_000_000 {
		t.Errorf("expected default DustThreshold 1_000_000, got %d", sel.DustThreshold)
	}
	if sel.MaxDustInputs != 2 {
		t.Errorf("expected default MaxDustInputs 2, got %d", sel.MaxDustInputs)
	}
}

// TestMACSPrunesRedundantInputs verifies the post-selection pruning pass:
// an input picked early for one asset that a later pick also covers must be
// dropped when the rest of the selection still covers the target.
func TestMACSPrunesRedundantInputs(t *testing.T) {
	u1 := makeSelectorUtxo(t, 0x01, 0, 1_000_000, makeTestAssets(0xAA, "tokenA", 50))
	u2Assets := makeTestAssets(0xAA, "tokenA", 50)
	u2Assets.Add(makeTestAssets(0xBB, "tokenB", 50))
	u2 := makeSelectorUtxo(t, 0x02, 0, 1_000_000, u2Assets)
	u3 := makeSelectorUtxo(t, 0x03, 0, 10_000_000, nil)
	pool := []common.Utxo{u1, u2, u3}

	targetAssets := makeTestAssets(0xAA, "tokenA", 50)
	targetAssets.Add(makeTestAssets(0xBB, "tokenB", 50))
	target := NewValue(4_000_000, targetAssets)

	selected, err := (&MACSSelector{}).Select(t.Context(), pool, target)
	if err != nil {
		t.Fatalf("Select failed: %v", err)
	}
	for _, u := range selected {
		if utxoRef(u) == utxoRef(u1) {
			t.Errorf("redundant UTxO %s was not pruned", utxoRef(u1))
		}
	}
	if len(selected) != 2 {
		t.Fatalf("expected 2 inputs after pruning, got %d", len(selected))
	}
	if !sumSelected(t, selected).GreaterOrEqual(target) {
		t.Error("pruned selection no longer covers target")
	}
}

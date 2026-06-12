package apollo

import (
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
	selected, err := (&MACSSelector{}).Select(pool, NewSimpleValue(5_000_000))
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
	selected, err := (&MACSSelector{}).Select(pool, target)
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
	selected, err := sel.Select(pool, NewSimpleValue(5_000_000))
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
	selected, err := (&MACSSelector{}).Select(pool, NewSimpleValue(5_000_000))
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

	selected, err := (&MACSSelector{}).Select(pool, target)
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

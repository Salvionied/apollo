package apollo

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/common"
)

// makeSelectorUtxo builds a UTxO for selector tests with a deterministic ref.
func makeSelectorUtxo(t *testing.T, txHashByte byte, index uint32, lovelace uint64, assets *common.MultiAsset[common.MultiAssetTypeOutput]) common.Utxo {
	t.Helper()
	var txHash common.Blake2b256
	txHash[0] = txHashByte
	return makeAssetTestUtxo(t, txHash, index, lovelace, assets)
}

// makeTestAssets builds a MultiAsset with a single policy and asset name.
func makeTestAssets(policyByte byte, name string, qty int64) *common.MultiAsset[common.MultiAssetTypeOutput] {
	var policy common.Blake2b224
	policy[0] = policyByte
	return MultiAssetFromMap(map[common.Blake2b224]map[cbor.ByteString]*big.Int{
		policy: {cbor.NewByteString([]byte(name)): big.NewInt(qty)},
	})
}

func sumSelected(t *testing.T, selected []common.Utxo) Value {
	t.Helper()
	total := Value{}
	for _, u := range selected {
		amt := u.Output.Amount()
		if amt == nil || !amt.IsUint64() {
			t.Fatalf("selected UTxO %s has invalid amount", utxoRef(u))
		}
		uv := NewValue(amt.Uint64(), CloneMultiAsset(u.Output.Assets()))
		var err error
		total, err = total.Add(uv)
		if err != nil {
			t.Fatalf("sum overflow: %v", err)
		}
	}
	return total
}

// runSelectorConformance runs the shared conformance suite that any
// CoinSelector implementation must pass.
func runSelectorConformance(t *testing.T, newSelector func() CoinSelector) {
	t.Helper()

	t.Run("CoversAdaTarget", func(t *testing.T) {
		pool := []common.Utxo{
			makeSelectorUtxo(t, 0x01, 0, 3_000_000, nil),
			makeSelectorUtxo(t, 0x02, 0, 10_000_000, nil),
			makeSelectorUtxo(t, 0x03, 0, 5_000_000, nil),
		}
		target := NewSimpleValue(12_000_000)
		selected, err := newSelector().Select(t.Context(), pool, target)
		if err != nil {
			t.Fatalf("Select failed: %v", err)
		}
		if !sumSelected(t, selected).GreaterOrEqual(target) {
			t.Errorf("selection does not cover target")
		}
	})

	t.Run("CoversMultiAssetTarget", func(t *testing.T) {
		pool := []common.Utxo{
			makeSelectorUtxo(t, 0x01, 0, 2_000_000, makeTestAssets(0xAA, "tokenA", 100)),
			makeSelectorUtxo(t, 0x02, 0, 10_000_000, nil),
			makeSelectorUtxo(t, 0x03, 0, 2_000_000, makeTestAssets(0xBB, "tokenB", 7)),
		}
		target := NewValue(5_000_000, makeTestAssets(0xAA, "tokenA", 50))
		selected, err := newSelector().Select(t.Context(), pool, target)
		if err != nil {
			t.Fatalf("Select failed: %v", err)
		}
		if !sumSelected(t, selected).GreaterOrEqual(target) {
			t.Errorf("selection does not cover multi-asset target")
		}
	})

	t.Run("NoDuplicateSelections", func(t *testing.T) {
		pool := []common.Utxo{
			makeSelectorUtxo(t, 0x01, 0, 2_000_000, makeTestAssets(0xAA, "tokenA", 100)),
			makeSelectorUtxo(t, 0x02, 0, 3_000_000, nil),
			makeSelectorUtxo(t, 0x03, 0, 4_000_000, nil),
		}
		target := NewValue(8_000_000, makeTestAssets(0xAA, "tokenA", 100))
		selected, err := newSelector().Select(t.Context(), pool, target)
		if err != nil {
			t.Fatalf("Select failed: %v", err)
		}
		seen := make(map[string]bool)
		for _, u := range selected {
			ref := utxoRef(u)
			if seen[ref] {
				t.Errorf("UTxO %s selected twice", ref)
			}
			seen[ref] = true
		}
	})

	t.Run("ErrorOnInsufficientCoin", func(t *testing.T) {
		pool := []common.Utxo{
			makeSelectorUtxo(t, 0x01, 0, 1_000_000, nil),
		}
		_, err := newSelector().Select(
			t.Context(), pool, NewSimpleValue(100_000_000),
		)
		if err == nil {
			t.Fatal("expected error for insufficient coin")
		}
		if !strings.Contains(err.Error(), "insufficient") {
			t.Errorf("expected insufficiency error, got: %v", err)
		}
	})

	t.Run("ErrorOnMissingAsset", func(t *testing.T) {
		pool := []common.Utxo{
			makeSelectorUtxo(t, 0x01, 0, 10_000_000, nil),
		}
		target := NewValue(1_000_000, makeTestAssets(0xAA, "tokenA", 1))
		_, err := newSelector().Select(t.Context(), pool, target)
		if err == nil {
			t.Fatal("expected error for missing asset")
		}
		if !strings.Contains(err.Error(), "insufficient") {
			t.Errorf("expected insufficiency error, got: %v", err)
		}
	})

	t.Run("EmptyTargetSelectsNothing", func(t *testing.T) {
		pool := []common.Utxo{
			makeSelectorUtxo(t, 0x01, 0, 1_000_000, nil),
		}
		selected, err := newSelector().Select(t.Context(), pool, Value{})
		if err != nil {
			t.Fatalf("Select failed: %v", err)
		}
		if len(selected) != 0 {
			t.Errorf("expected empty selection for empty target, got %d UTxOs", len(selected))
		}
	})

	t.Run("Deterministic", func(t *testing.T) {
		pool := []common.Utxo{
			makeSelectorUtxo(t, 0x01, 0, 2_000_000, makeTestAssets(0xAA, "tokenA", 30)),
			makeSelectorUtxo(t, 0x02, 0, 2_000_000, makeTestAssets(0xAA, "tokenA", 30)),
			makeSelectorUtxo(t, 0x03, 0, 5_000_000, nil),
			makeSelectorUtxo(t, 0x04, 0, 5_000_000, nil),
		}
		target := NewValue(6_000_000, makeTestAssets(0xAA, "tokenA", 40))
		first, err := newSelector().Select(t.Context(), pool, target)
		if err != nil {
			t.Fatalf("Select failed: %v", err)
		}
		second, err := newSelector().Select(t.Context(), pool, target)
		if err != nil {
			t.Fatalf("Select failed: %v", err)
		}
		if len(first) != len(second) {
			t.Fatalf("non-deterministic selection size: %d vs %d", len(first), len(second))
		}
		for i := range first {
			if utxoRef(first[i]) != utxoRef(second[i]) {
				t.Errorf("non-deterministic selection at %d: %s vs %s", i, utxoRef(first[i]), utxoRef(second[i]))
			}
		}
	})
}

func TestLargestFirstSelectorConformance(t *testing.T) {
	runSelectorConformance(t, func() CoinSelector { return &LargestFirstSelector{} })
}

func TestRandomImproveSelectorConformance(t *testing.T) {
	runSelectorConformance(t, func() CoinSelector { return NewRandomImproveSelector() })
}

func TestRandomImproveThenLargestFirstSelectorConformance(t *testing.T) {
	runSelectorConformance(t, func() CoinSelector { return NewRandomImproveThenLargestFirstSelector() })
}

func TestRandomImproveSelectorNames(t *testing.T) {
	if name := NewRandomImproveSelector().Name(); name != "random-improve" {
		t.Errorf("expected name random-improve, got %q", name)
	}
	if name := NewRandomImproveThenLargestFirstSelector().Name(); name != "random-improve-largest-first" {
		t.Errorf("expected name random-improve-largest-first, got %q", name)
	}
}

func TestLargestFirstSelectorName(t *testing.T) {
	if name := (&LargestFirstSelector{}).Name(); name != "largest-first" {
		t.Errorf("expected name largest-first, got %q", name)
	}
}

func TestCoinSelectorsRejectMalformedUTxOs(t *testing.T) {
	selectors := []CoinSelector{
		&LargestFirstSelector{},
		NewMACSSelector(),
		NewRandomImproveSelector(),
		NewRandomImproveThenLargestFirstSelector(),
	}
	for _, selector := range selectors {
		t.Run(selector.Name(), func(t *testing.T) {
			if _, err := selector.Select(
				t.Context(),
				[]common.Utxo{{}},
				NewSimpleValue(1),
			); err == nil {
				t.Fatal("expected malformed UTxO error")
			}
		})
	}
}

// TestLargestFirstSelectorOrder pins the exact legacy behavior: ADA-only
// UTxOs are consumed first in descending lovelace order, before any
// asset-carrying UTxOs.
func TestLargestFirstSelectorOrder(t *testing.T) {
	withAssets := makeSelectorUtxo(t, 0x0A, 0, 20_000_000, makeTestAssets(0xAA, "tokenA", 5))
	pool := []common.Utxo{
		makeSelectorUtxo(t, 0x01, 0, 3_000_000, nil),
		withAssets,
		makeSelectorUtxo(t, 0x02, 0, 10_000_000, nil),
		makeSelectorUtxo(t, 0x03, 0, 5_000_000, nil),
	}
	selected, err := (&LargestFirstSelector{}).Select(
		t.Context(), pool, NewSimpleValue(12_000_000),
	)
	if err != nil {
		t.Fatalf("Select failed: %v", err)
	}
	if len(selected) != 2 {
		t.Fatalf("expected 2 UTxOs selected, got %d", len(selected))
	}
	if got := selected[0].Output.Amount().Uint64(); got != 10_000_000 {
		t.Errorf("expected first selection of 10 ADA, got %d lovelace", got)
	}
	if got := selected[1].Output.Amount().Uint64(); got != 5_000_000 {
		t.Errorf("expected second selection of 5 ADA, got %d lovelace", got)
	}
}

// recordingSelector wraps a CoinSelector and records that it was invoked,
// proving the builder dispatches to the configured selector.
type recordingSelector struct {
	inner  CoinSelector
	called bool
}

func (r *recordingSelector) Name() string { return "recording" }

func (r *recordingSelector) Select(
	ctx context.Context,
	available []common.Utxo,
	target Value,
) ([]common.Utxo, error) {
	r.called = true
	return r.inner.Select(ctx, available, target)
}

func TestSetCoinSelectorUsedByComplete(t *testing.T) {
	cc := setupFixedContext()
	addr := testAddress(t)
	addTestUtxo(cc, addr, 10_000_000, 0x01, 0)

	rec := &recordingSelector{inner: &LargestFirstSelector{}}
	p, err := NewPayment(validTestAddrBech32, 2_000_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	a := newTestApollo(cc).
		SetWallet(NewExternalWallet(addr)).
		AddPayment(p).
		SetTtl(50000000).
		SetCoinSelector(rec)

	if _, err := a.Complete(); err != nil {
		t.Fatalf("Complete failed: %v", err)
	}
	if !rec.called {
		t.Error("custom coin selector was not invoked by Complete")
	}
}

// contextSelectors returns one instance of every in-tree selector, for the
// context tests below.
func contextSelectors() []CoinSelector {
	return []CoinSelector{
		&LargestFirstSelector{},
		&MACSSelector{},
		NewMACSSelector(),
		NewRandomImproveSelector(),
		NewRandomImproveThenLargestFirstSelector(),
	}
}

// sweepSelectionPool builds n equal 1 ADA UTxOs and a target needing 90% of
// them, so covering it takes thousands of picks: long enough that abandoning
// the search early is observable.
func sweepSelectionPool(tb testing.TB, n int) ([]common.Utxo, Value) {
	tb.Helper()
	addr := benchAddress(tb)
	pool := make([]common.Utxo, 0, n)
	var total uint64
	for i := 0; i < n; i++ {
		pool = append(pool, benchUtxo(addr, uint64(i)+1, 1_000_000, nil))
		total += 1_000_000
	}
	return pool, NewSimpleValue(total / 10 * 9)
}

// TestSelectorsAbortOnCanceledContext verifies a cancelled context abandons
// selection promptly, rather than after the whole pool has been searched.
func TestSelectorsAbortOnCanceledContext(t *testing.T) {
	pool, target := sweepSelectionPool(t, 10_000)
	for _, selector := range contextSelectors() {
		t.Run(selector.Name(), func(t *testing.T) {
			start := time.Now()
			if _, err := selector.Select(t.Context(), pool, target); err != nil {
				t.Fatalf("uncancelled Select failed: %v", err)
			}
			whole := time.Since(start)

			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			start = time.Now()
			selected, err := selector.Select(ctx, pool, target)
			aborted := time.Since(start)

			if !errors.Is(err, context.Canceled) {
				t.Fatalf("Select() error = %v, want context.Canceled", err)
			}
			if selected != nil {
				t.Errorf(
					"abandoned selection returned %d UTxOs, want none",
					len(selected),
				)
			}
			// The uncancelled search is only a few milliseconds on 386 CI,
			// so scheduler jitter can exceed whole/4. Keep the check, but
			// allow half the full search there.
			limit := whole / 4
			if runtime.GOARCH == "386" {
				limit = whole / 2
			}
			if aborted > limit {
				t.Errorf(
					"cancellation took %v, close to the full %v search",
					aborted, whole,
				)
			}
			t.Logf("full=%v cancelled=%v", whole, aborted)
		})
	}
}

// countdownContext reports itself done only after n calls to Err, so a test
// can prove a selector rechecks the context while searching instead of only
// on entry. Done is never closed: selectors poll Err.
type countdownContext struct {
	context.Context
	mu sync.Mutex
	n  int
}

func (c *countdownContext) Err() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.n > 0 {
		c.n--
		return nil
	}
	return context.Canceled
}

// TestSelectorsAbortMidSelection verifies the context is honoured inside the
// selection loops: a context that only reports itself done after the first few
// checks must still abort the search.
func TestSelectorsAbortMidSelection(t *testing.T) {
	pool, target := sweepSelectionPool(t, 4_000)
	for _, selector := range contextSelectors() {
		t.Run(selector.Name(), func(t *testing.T) {
			ctx := &countdownContext{Context: context.Background(), n: 3}
			selected, err := selector.Select(ctx, pool, target)
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("Select() error = %v, want context.Canceled", err)
			}
			if selected != nil {
				t.Errorf(
					"abandoned selection returned %d UTxOs, want none",
					len(selected),
				)
			}
		})
	}
}

// TestSelectorsTolerateNilContext pins that a third-party caller passing no
// context gets a selection rather than a panic.
func TestSelectorsTolerateNilContext(t *testing.T) {
	pool := []common.Utxo{
		makeSelectorUtxo(t, 0x01, 0, 10_000_000, nil),
	}
	// Deliberately nil: Select is exported and must not dereference it.
	var ctx context.Context
	for _, selector := range contextSelectors() {
		t.Run(selector.Name(), func(t *testing.T) {
			selected, err := selector.Select(
				ctx, pool, NewSimpleValue(1_000_000),
			)
			if err != nil {
				t.Fatalf("Select failed: %v", err)
			}
			if len(selected) != 1 {
				t.Fatalf("expected 1 input, got %d", len(selected))
			}
		})
	}
}

// TestCompletePropagatesContextCancellation verifies WithContext reaches coin
// selection through the builder.
func TestCompletePropagatesContextCancellation(t *testing.T) {
	cc := setupFixedContext()
	addr := testAddress(t)
	addTestUtxo(cc, addr, 10_000_000, 0x01, 0)

	p, err := NewPayment(validTestAddrBech32, 2_000_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	a := newTestApollo(cc).
		SetWallet(NewExternalWallet(addr)).
		AddPayment(p).
		SetTtl(50000000).
		WithContext(ctx)

	if _, err := a.Complete(); !errors.Is(err, context.Canceled) {
		t.Fatalf("Complete() error = %v, want context.Canceled", err)
	}
}

// TestSelectorsReportInsufficientUtxos pins that every in-tree selector
// reports an uncoverable target with ErrInsufficientUtxos, which is what
// RandomImproveThenLargestFirstSelector falls back on.
func TestSelectorsReportInsufficientUtxos(t *testing.T) {
	pool := []common.Utxo{
		makeSelectorUtxo(t, 0x01, 0, 1_000_000, nil),
	}
	targets := map[string]Value{
		"coin":  NewSimpleValue(100_000_000),
		"asset": NewValue(1_000_000, makeTestAssets(0xAA, "tokenA", 1)),
	}
	for _, selector := range contextSelectors() {
		for label, target := range targets {
			t.Run(selector.Name()+"/"+label, func(t *testing.T) {
				_, err := selector.Select(t.Context(), pool, target)
				if !errors.Is(err, ErrInsufficientUtxos) {
					t.Fatalf("Select() error = %v, want ErrInsufficientUtxos", err)
				}
			})
		}
	}
}

// improvedSelections runs Random-Improve over seeds 0-31 and returns the
// selections whose cover phase stopped at first, the UTxO that covers target
// alone, so anything after it was added by the improve phase.
func improvedSelections(t *testing.T, pool []common.Utxo, first common.Utxo, target Value) [][]common.Utxo {
	t.Helper()
	var out [][]common.Utxo
	for seed := int64(0); seed < 32; seed++ {
		selected, err := (&RandomImproveSelector{Seed: seed}).Select(
			t.Context(), pool, target,
		)
		if err != nil {
			t.Fatalf("Select(seed %d) failed: %v", seed, err)
		}
		if utxoRef(selected[0]) == utxoRef(first) {
			out = append(out, selected)
		}
	}
	if len(out) == 0 {
		t.Fatal("no seed covered with the expected UTxO first")
	}
	return out
}

func TestRandomImproveSelectorImprovesTowardDoubleTarget(t *testing.T) {
	first := makeSelectorUtxo(t, 0x01, 0, 10_000_000, nil)
	pool := []common.Utxo{first, makeSelectorUtxo(t, 0x02, 0, 5_000_000, nil)}
	for _, selected := range improvedSelections(t, pool, first, NewSimpleValue(10_000_000)) {
		if got := sumSelected(t, selected).Coin; got != 15_000_000 {
			t.Errorf("expected improved total of 15 ADA, got %d lovelace", got)
		}
	}
}

func TestRandomImproveSelectorDoesNotImprovePastDoubleTarget(t *testing.T) {
	pool := []common.Utxo{
		makeSelectorUtxo(t, 0x01, 0, 10_000_000, nil),
		makeSelectorUtxo(t, 0x02, 0, 11_000_000, nil),
	}
	selected, err := NewRandomImproveSelector().Select(
		t.Context(), pool, NewSimpleValue(10_000_000),
	)
	if err != nil {
		t.Fatalf("Select failed: %v", err)
	}
	if len(selected) != 1 {
		t.Fatalf("expected improvement to skip overshooting UTxO, got %d selections", len(selected))
	}
}

func TestRandomImproveSelectorImprovesTargetAssets(t *testing.T) {
	first := makeSelectorUtxo(t, 0x01, 0, 2_000_000, makeTestAssets(0xAA, "tokenA", 10))
	pool := []common.Utxo{
		first,
		makeSelectorUtxo(t, 0x02, 0, 1_000_000, makeTestAssets(0xAA, "tokenA", 5)),
	}
	target := NewValue(2_000_000, makeTestAssets(0xAA, "tokenA", 10))
	var policy common.Blake2b224
	policy[0] = 0xAA
	for _, selected := range improvedSelections(t, pool, first, target) {
		got := sumSelected(t, selected)
		if got.Coin != 3_000_000 {
			t.Fatalf("expected improved coin total of 3 ADA, got %d lovelace", got.Coin)
		}
		qty := got.Assets.Asset(policy, []byte("tokenA"))
		if qty == nil || qty.Cmp(big.NewInt(15)) != 0 {
			t.Fatalf("expected improved token total of 15, got %v", qty)
		}
	}
}

// TestRandomImproveSelectorSkipsUntargetedAssetsWhenImproving pins that the
// improve phase never adds a UTxO carrying an asset the target does not ask
// for. The ADA-only UTxO covers the target alone, so the token UTxO can only
// be selected by the cover phase, ahead of it.
func TestRandomImproveSelectorSkipsUntargetedAssetsWhenImproving(t *testing.T) {
	ada := makeSelectorUtxo(t, 0x01, 0, 10_000_000, nil)
	pool := []common.Utxo{
		ada,
		makeSelectorUtxo(t, 0x02, 0, 5_000_000, makeTestAssets(0xBB, "tokenB", 1)),
	}
	for _, selected := range improvedSelections(t, pool, ada, NewSimpleValue(10_000_000)) {
		if len(selected) != 1 {
			t.Fatal("improve phase added a UTxO with an untargeted asset")
		}
	}
}

// TestRandomImproveSelectorSeedChangesOrder proves Seed drives the ordering:
// with many interchangeable UTxOs, some other seed must pick a different set
// than seed 0.
func TestRandomImproveSelectorSeedChangesOrder(t *testing.T) {
	pool := make([]common.Utxo, 0, 16)
	for i := byte(1); i <= 16; i++ {
		pool = append(pool, makeSelectorUtxo(t, i, 0, uint64(i)*1_000_000, nil))
	}
	target := NewSimpleValue(20_000_000)
	refs := func(seed int64) string {
		selected, err := (&RandomImproveSelector{Seed: seed}).Select(
			t.Context(), pool, target,
		)
		if err != nil {
			t.Fatalf("Select(seed %d) failed: %v", seed, err)
		}
		var b strings.Builder
		for _, u := range selected {
			b.WriteString(utxoRef(u))
			b.WriteByte(',')
		}
		return b.String()
	}
	base := refs(0)
	if base != refs(0) {
		t.Fatal("same seed produced different selections")
	}
	for seed := int64(1); seed < 16; seed++ {
		if refs(seed) != base {
			return
		}
	}
	t.Fatal("seeds 1-15 all reproduced seed 0's selection; Seed is ignored")
}

func TestRandomImproveSelectorOverflowIsNotInsufficiency(t *testing.T) {
	maxUint64 := ^uint64(0)
	pool := []common.Utxo{
		makeSelectorUtxo(t, 0x01, 0, maxUint64-1, nil),
		makeSelectorUtxo(t, 0x02, 0, 2, nil),
	}
	_, err := NewRandomImproveSelector().Select(
		t.Context(), pool, NewSimpleValue(maxUint64),
	)
	if err == nil || !strings.Contains(err.Error(), "overflow") {
		t.Fatalf("expected overflow error, got: %v", err)
	}
	if errors.Is(err, ErrInsufficientUtxos) {
		t.Fatal("overflow must not be reported as insufficiency")
	}
}

type stubSelector struct {
	err error
}

func (s stubSelector) Name() string { return "stub" }

func (s stubSelector) Select(
	context.Context,
	[]common.Utxo,
	Value,
) ([]common.Utxo, error) {
	return nil, s.err
}

func TestRandomImproveThenLargestFirstSelectorFallsBack(t *testing.T) {
	pool := []common.Utxo{
		makeSelectorUtxo(t, 0x01, 0, 3_000_000, nil),
		makeSelectorUtxo(t, 0x02, 0, 10_000_000, nil),
		makeSelectorUtxo(t, 0x03, 0, 5_000_000, nil),
	}
	selector := &RandomImproveThenLargestFirstSelector{
		RandomImprove: stubSelector{
			err: fmt.Errorf("wrapped: %w", ErrInsufficientUtxos),
		},
	}
	selected, err := selector.Select(t.Context(), pool, NewSimpleValue(12_000_000))
	if err != nil {
		t.Fatalf("Select failed: %v", err)
	}
	if len(selected) != 2 {
		t.Fatalf("expected 2 UTxOs selected by fallback, got %d", len(selected))
	}
	if got := selected[0].Output.Amount().Uint64(); got != 10_000_000 {
		t.Errorf("expected fallback to select 10 ADA first, got %d lovelace", got)
	}
	if got := selected[1].Output.Amount().Uint64(); got != 5_000_000 {
		t.Errorf("expected fallback to select 5 ADA second, got %d lovelace", got)
	}
}

func TestRandomImproveThenLargestFirstSelectorReportsBothFailures(t *testing.T) {
	pool := []common.Utxo{
		makeSelectorUtxo(t, 0x01, 0, 1_000_000, nil),
	}
	_, err := NewRandomImproveThenLargestFirstSelector().Select(
		t.Context(), pool, NewSimpleValue(100_000_000),
	)
	if !errors.Is(err, ErrInsufficientUtxos) {
		t.Fatalf("Select() error = %v, want ErrInsufficientUtxos", err)
	}
	if !strings.Contains(err.Error(), "random-improve") ||
		!strings.Contains(err.Error(), "largest-first") {
		t.Fatalf("expected both selectors named in error, got: %v", err)
	}
}

// TestRandomImproveThenLargestFirstSelectorFallsBackOnlyOnInsufficiency pins
// that malformed input, overflow, a primary failure, and cancellation are
// reported rather than retried with the fallback selector.
func TestRandomImproveThenLargestFirstSelectorFallsBackOnlyOnInsufficiency(t *testing.T) {
	good := makeSelectorUtxo(t, 0x01, 0, 10_000_000, nil)
	badBase := makeSelectorUtxo(t, 0x02, 0, 1_000_000, nil)
	bad := common.Utxo{
		Id:     badBase.Id,
		Output: badAmountOutput{badBase.Output, new(big.Int).Lsh(big.NewInt(1), 64)},
	}
	maxUint64 := ^uint64(0)
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	cases := []struct {
		name    string
		ctx     context.Context
		primary CoinSelector
		pool    []common.Utxo
		target  Value
		want    string
	}{
		{
			name:    "primary error",
			ctx:     t.Context(),
			primary: stubSelector{err: errors.New("forced failure")},
			pool:    []common.Utxo{good},
			target:  NewSimpleValue(1_000_000),
			want:    "forced failure",
		},
		{
			name:   "invalid amount",
			ctx:    t.Context(),
			pool:   []common.Utxo{good, bad},
			target: NewSimpleValue(1_000_000),
			want:   "invalid",
		},
		{
			name: "overflow",
			ctx:  t.Context(),
			pool: []common.Utxo{
				makeSelectorUtxo(t, 0x01, 0, maxUint64-1, nil),
				makeSelectorUtxo(t, 0x02, 0, 2, nil),
			},
			target: NewSimpleValue(maxUint64),
			want:   "overflow",
		},
		{
			name:   "canceled",
			ctx:    canceled,
			pool:   []common.Utxo{good},
			target: NewSimpleValue(1_000_000),
			want:   "canceled",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fallback := &recordingSelector{inner: &LargestFirstSelector{}}
			selector := &RandomImproveThenLargestFirstSelector{
				RandomImprove: tc.primary,
				Fallback:      fallback,
			}
			_, err := selector.Select(tc.ctx, tc.pool, tc.target)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected %q error, got: %v", tc.want, err)
			}
			if fallback.called {
				t.Fatal("fallback ran for an error that is not insufficiency")
			}
		})
	}
}

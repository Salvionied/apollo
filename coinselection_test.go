package apollo

import (
	"context"
	"errors"
	"math/big"
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

func TestLargestFirstSelectorName(t *testing.T) {
	if name := (&LargestFirstSelector{}).Name(); name != "largest-first" {
		t.Errorf("expected name largest-first, got %q", name)
	}
}

func TestCoinSelectorsRejectMalformedUTxOs(t *testing.T) {
	selectors := []CoinSelector{
		&LargestFirstSelector{},
		NewMACSSelector(),
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
	a := New(cc).
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
			if aborted > whole/4 {
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
	a := New(cc).
		SetWallet(NewExternalWallet(addr)).
		AddPayment(p).
		SetTtl(50000000).
		WithContext(ctx)

	if _, err := a.Complete(); !errors.Is(err, context.Canceled) {
		t.Fatalf("Complete() error = %v, want context.Canceled", err)
	}
}

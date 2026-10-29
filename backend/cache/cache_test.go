package cache

import (
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/Salvionied/apollo/v2/backend"
	"github.com/Salvionied/apollo/v2/backend/fixed"
)

func TestCapabilitiesMatchWrappedContext(t *testing.T) {
	ctx := NewCachedChainContext(fixed.NewEmptyFixedChainContext(), 0)
	if !backend.Supports(ctx, backend.CapabilityProtocolParams|backend.CapabilityUtxoByRef) {
		t.Fatal("cache did not preserve supported capabilities")
	}
	if backend.Supports(ctx, backend.CapabilitySubmitTx) {
		t.Fatal("cache reported unsupported wrapped capability")
	}
}

// countingChainContext counts the MaxTxFee calls that reach the wrapped
// context.
type countingChainContext struct {
	*fixed.FixedChainContext
	maxTxFeeCalls int
	maxTxFeeErr   error
}

func (c *countingChainContext) MaxTxFee() (uint64, error) {
	c.maxTxFeeCalls++
	if c.maxTxFeeErr != nil {
		return 0, c.maxTxFeeErr
	}
	return c.FixedChainContext.MaxTxFee()
}

func TestMaxTxFeeCached(t *testing.T) {
	inner := &countingChainContext{
		FixedChainContext: fixed.NewEmptyFixedChainContext(),
	}
	want, err := inner.FixedChainContext.MaxTxFee()
	if err != nil {
		t.Fatal(err)
	}
	ctx := NewCachedChainContext(inner, time.Minute)

	for range 3 {
		got, err := ctx.MaxTxFee()
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("MaxTxFee() = %d, want %d", got, want)
		}
	}
	if inner.maxTxFeeCalls != 1 {
		t.Fatalf("inner MaxTxFee called %d times, want 1", inner.maxTxFeeCalls)
	}

	ctx.mu.Lock()
	ctx.maxTxFeeCacheAt = ctx.maxTxFeeCacheAt.Add(-time.Minute)
	ctx.mu.Unlock()
	if _, err := ctx.MaxTxFee(); err != nil {
		t.Fatal(err)
	}
	if inner.maxTxFeeCalls != 2 {
		t.Fatalf("inner MaxTxFee called %d times after expiry, want 2", inner.maxTxFeeCalls)
	}
}

func TestMaxTxFeeErrorNotCached(t *testing.T) {
	inner := &countingChainContext{
		FixedChainContext: fixed.NewEmptyFixedChainContext(),
		maxTxFeeErr:       errors.New("provider unavailable"),
	}
	ctx := NewCachedChainContext(inner, time.Minute)

	if _, err := ctx.MaxTxFee(); err == nil {
		t.Fatal("expected error")
	}
	inner.maxTxFeeErr = nil
	if _, err := ctx.MaxTxFee(); err != nil {
		t.Fatal(err)
	}
	if inner.maxTxFeeCalls != 2 {
		t.Fatalf("inner MaxTxFee called %d times, want 2", inner.maxTxFeeCalls)
	}
}

func TestMaxTxFeeZeroTTLNotCached(t *testing.T) {
	inner := &countingChainContext{
		FixedChainContext: fixed.NewEmptyFixedChainContext(),
	}
	ctx := NewCachedChainContext(inner, 0)
	for range 2 {
		if _, err := ctx.MaxTxFee(); err != nil {
			t.Fatal(err)
		}
	}
	if inner.maxTxFeeCalls != 2 {
		t.Fatalf("inner MaxTxFee called %d times, want 2", inner.maxTxFeeCalls)
	}
}

// TestProtocolParamsReturnsCopies checks that mutating a returned value,
// including its reference-script prices, cannot reach the cached copy.
func TestProtocolParamsReturnsCopies(t *testing.T) {
	pp, err := fixed.NewEmptyFixedChainContext().ProtocolParams()
	if err != nil {
		t.Fatal(err)
	}
	pp.CostModels = map[string][]int64{"PlutusV3": {100788, 420, 1}}
	pp.MinFeeRefScriptCostPerByteRational = big.NewRat(15, 1)
	pp.MinFeeReferenceScriptsMultiplierRational = big.NewRat(6, 5)
	ctx := NewCachedChainContext(
		fixed.NewFixedChainContext(pp, backend.GenesisParameters{}, 0),
		time.Minute,
	)

	for range 2 {
		first, err := ctx.ProtocolParams()
		if err != nil {
			t.Fatal(err)
		}
		first.CostModels["PlutusV3"][0] = -1
		first.MinFeeRefScriptCostPerByteRational.SetInt64(-1)
		first.MinFeeReferenceScriptsMultiplierRational.SetInt64(-1)
	}

	got, err := ctx.ProtocolParams()
	if err != nil {
		t.Fatal(err)
	}
	if got.CostModels["PlutusV3"][0] != 100788 {
		t.Fatal("mutating a returned cost model changed the cache")
	}
	if got.MinFeeRefScriptCostPerByteRational.Cmp(big.NewRat(15, 1)) != 0 {
		t.Fatalf("cached ref script price = %s, want 15", got.MinFeeRefScriptCostPerByteRational)
	}
	if got.MinFeeReferenceScriptsMultiplierRational.Cmp(big.NewRat(6, 5)) != 0 {
		t.Fatalf("cached ref script multiplier = %s, want 6/5", got.MinFeeReferenceScriptsMultiplierRational)
	}
}

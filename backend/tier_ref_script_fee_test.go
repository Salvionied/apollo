package backend

import (
	"math"
	"math/big"
	"testing"
)

// exactTierRefScriptFee is an independent exact-rational implementation of the
// ledger's tierRefScriptFee (multiplier as the exact rational mulNum/mulDen),
// used as the oracle for TierRefScriptFee.
func exactTierRefScriptFee(size int, base float64, incr int, mulNum, mulDen int64) int64 {
	if size <= 0 || base <= 0 {
		return 0
	}
	b := new(big.Rat).SetFloat64(base)
	m := big.NewRat(mulNum, mulDen)
	acc := new(big.Rat)
	price := new(big.Rat).Set(b)
	n := size
	for n >= incr {
		acc.Add(acc, new(big.Rat).Mul(new(big.Rat).SetInt64(int64(incr)), price))
		price.Mul(price, m)
		n -= incr
	}
	if n > 0 {
		acc.Add(acc, new(big.Rat).Mul(new(big.Rat).SetInt64(int64(n)), price))
	}
	return new(big.Int).Quo(acc.Num(), acc.Denom()).Int64()
}

func exactTierRefScriptFeeRational(size int, base *big.Rat, incr int, multiplier *big.Rat) int64 {
	if size <= 0 || base == nil || base.Sign() <= 0 {
		return 0
	}
	acc := new(big.Rat)
	price := new(big.Rat).Set(base)
	n := size
	for n >= incr {
		acc.Add(acc, new(big.Rat).Mul(new(big.Rat).SetInt64(int64(incr)), price))
		price.Mul(price, multiplier)
		n -= incr
	}
	if n > 0 {
		acc.Add(acc, new(big.Rat).Mul(new(big.Rat).SetInt64(int64(n)), price))
	}
	return new(big.Int).Quo(acc.Num(), acc.Denom()).Int64()
}

// TierRefScriptFee must equal the exact ledger value (floor of the exact
// rational accumulation) for all sizes, including multi-tier.
func TestTierRefScriptFeeIsExact(t *testing.T) {
	const base = 15.0
	const incr = DefaultRefScriptSizeIncrement // 25600
	const mul = DefaultRefScriptMultiplier     // 1.2
	for _, s := range []int{1, 100, 5000, 25599, 25600, 25601, 51200, 76800, 80000, 128000, 200000, 250880, 333333} {
		got := TierRefScriptFee(s, base, incr, mul)
		want := exactTierRefScriptFee(s, base, incr, 6, 5)
		if got != want {
			t.Errorf("TierRefScriptFee(%d) = %d, want exact %d", s, got, want)
		}
	}
}

// For a single tier (size < increment) with an integer base price, the fee is
// exactly size*base.
func TestTierRefScriptFeeSingleTierLinear(t *testing.T) {
	const base = 15.0
	for _, s := range []int{1, 1000, 5000, 25599} {
		if got, want := TierRefScriptFee(s, base, DefaultRefScriptSizeIncrement, DefaultRefScriptMultiplier), int64(s)*15; got != want {
			t.Errorf("single-tier TierRefScriptFee(%d) = %d, want %d", s, got, want)
		}
	}
}

// Dense sweep across the multi-tier regime (1..50 tiers): the rational
// implementation must equal the exact ledger oracle for every size, with no
// float drift able to push the floor below the ledger minimum.
func TestTierRefScriptFeeExactAcrossTiers(t *testing.T) {
	const base = 15.0
	const incr = DefaultRefScriptSizeIncrement
	const mul = DefaultRefScriptMultiplier
	for s := 1; s <= 50*incr; s += 97 {
		if got, want := TierRefScriptFee(s, base, incr, mul), exactTierRefScriptFee(s, base, incr, 6, 5); got != want {
			t.Fatalf("TierRefScriptFee(%d) = %d, want exact %d", s, got, want)
		}
	}
}

func TestTierRefScriptFeePreservesNonDefaultFractionalMultiplier(t *testing.T) {
	base := big.NewRat(121, 8)           // 15.125
	multiplier := big.NewRat(2469, 2000) // 1.2345
	const increment = 7

	for _, size := range []int{1, 7, 8, 53, 301, 1000} {
		want := exactTierRefScriptFeeRational(size, base, increment, multiplier)
		if got := TierRefScriptFeeRational(size, base, increment, multiplier); got != want {
			t.Errorf("TierRefScriptFeeRational(%d) = %d, want exact %d", size, got, want)
		}
		// The compatibility wrapper must not round 1.2345 to 1.235 before
		// converting it to a rational value.
		if got := TierRefScriptFee(size, 15.125, increment, 1.2345); got != want {
			t.Errorf("TierRefScriptFee(%d) = %d, want exact %d", size, got, want)
		}
	}
}

func TestProtocolParametersPreserveExactReferenceScriptRationals(t *testing.T) {
	pp := ProtocolParameters{
		MinFeeRefScriptCostPerByteRational:       big.NewRat(121, 8),
		MinFeeReferenceScriptsMultiplierRational: big.NewRat(2469, 2000),
		MinFeeReferenceScriptsRange:              7,
		MinFeeRefScriptCostPerByte:               15.125,
		MinFeeReferenceScriptsMultiplier:         1,
	}

	if got, want := pp.RefScriptFeePerByteRational(), big.NewRat(121, 8); got.Cmp(want) != 0 {
		t.Fatalf("RefScriptFeePerByteRational() = %s, want %s", got, want)
	}
	if got, want := pp.RefScriptMultiplierRational(), big.NewRat(2469, 2000); got.Cmp(want) != 0 {
		t.Fatalf("RefScriptMultiplierRational() = %s, want %s", got, want)
	}
	if got := pp.RefScriptSizeIncrement(); got != 7 {
		t.Fatalf("RefScriptSizeIncrement() = %d, want 7", got)
	}
}

func TestTierRefScriptFeeBoundsLargeTierCounts(t *testing.T) {
	const size = 1_000_000
	if got := TierRefScriptFeeRational(size, big.NewRat(1, 1), 1, big.NewRat(2, 1)); got != math.MaxInt64 {
		t.Fatalf("growing fee = %d, want MaxInt64 sentinel", got)
	}
	if got := TierRefScriptFeeRational(size, big.NewRat(1, 2), 1, big.NewRat(1, 1)); got != size/2 {
		t.Fatalf("constant price fee = %d, want %d", got, size/2)
	}
	if got := TierRefScriptFeeRational(size, big.NewRat(1, 1), 1, big.NewRat(1, 2)); got != 1 {
		t.Fatalf("decreasing fee = %d, want 1", got)
	}
	if _, err := TierRefScriptFeeRationalChecked(size, big.NewRat(1, 1), 1, big.NewRat(1_000_001, 1_000_000)); err == nil {
		t.Fatal("expected a work-limit error for a near-constant million-tier fee")
	}
}

func TestTierRefScriptFeeDecreasingPriceStaysExact(t *testing.T) {
	base := big.NewRat(1, 1)
	multiplier := big.NewRat(1, 2)
	for size := 1; size <= 100; size++ {
		want := exactTierRefScriptFeeRational(size, base, 1, multiplier)
		if got := TierRefScriptFeeRational(size, base, 1, multiplier); got != want {
			t.Fatalf("size %d: fee = %d, want %d", size, got, want)
		}
	}
}

func TestTierRefScriptFeeLargeFiniteDecreasingPrice(t *testing.T) {
	got, err := TierRefScriptFeeRationalChecked(600, big.NewRat(15, 1), 1, big.NewRat(99, 100))
	if err != nil || got != 1496 {
		t.Fatalf("600-tier decreasing fee = %d, %v; want 1496", got, err)
	}
	base := big.NewRat(15, 1)
	multiplier := big.NewRat(99, 100)
	want := exactTierRefScriptFeeRational(1201, base, 2, multiplier)
	got, err = TierRefScriptFeeRationalChecked(1201, base, 2, multiplier)
	if err != nil || got != want {
		t.Fatalf("partial final tier fee = %d, %v; want %d", got, err, want)
	}
}

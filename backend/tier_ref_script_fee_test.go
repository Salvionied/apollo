package backend

import (
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

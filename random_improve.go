package apollo

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"sort"

	"github.com/blinklabs-io/gouroboros/ledger/common"
)

// The cover and improve phases shuffle with different salts so that the
// improve order is not simply the cover order continued.
const (
	randomImproveCoverSalt   int64 = 0x4349503030303201
	randomImproveImproveSalt int64 = 0x52494d50524f5645
)

// RandomImproveSelector implements the Random-Improve algorithm from
// CIP-0002, adapted to Apollo's aggregate CoinSelector interface. It first
// covers the target by taking UTxOs in random order, drops any cover input a
// later pick made redundant, then adds ADA-only UTxOs that bring the lovelace
// total closer to an ideal of twice the target.
//
// It differs from CIP-0002 in three ways: it selects for the aggregate
// target rather than for each output; improvement may not take lovelace above
// twice the target, where CIP-0002 allows three times; and improvement only
// adds ADA-only UTxOs. Native-asset change needs its own min-UTxO lovelace, so
// an improvement that leaves surplus tokens can make a payment the cover alone
// pays for impossible to balance.
//
// The order is pseudo-random but deterministic: the same Seed, pool, and
// target always produce the same selection, as CoinSelector requires.
type RandomImproveSelector struct {
	// Seed selects the pseudo-random UTxO order. The zero value is a valid
	// seed.
	Seed int64
}

// NewRandomImproveSelector returns a RandomImproveSelector with seed 0.
func NewRandomImproveSelector() *RandomImproveSelector {
	return &RandomImproveSelector{}
}

// Name returns the algorithm's identifier.
func (s *RandomImproveSelector) Name() string { return "random-improve" }

// Select returns a subset of available whose summed value covers target. It
// returns an error wrapping ErrInsufficientUtxos when the pool cannot cover
// target.
func (s *RandomImproveSelector) Select(
	ctx context.Context,
	available []common.Utxo,
	target Value,
) ([]common.Utxo, error) {
	if target.Coin == 0 && !target.HasAssets() {
		return nil, nil
	}
	if err := selectionInterrupted(ctx); err != nil {
		return nil, err
	}
	if err := validateUtxos(available); err != nil {
		return nil, fmt.Errorf("invalid coin-selection input: %w", err)
	}
	targets := randomImproveTargets(target)
	cands, err := randomImproveCandidates(ctx, available, targets)
	if err != nil {
		return nil, err
	}

	// Only lovelace and the targeted assets decide coverage, so the running
	// total tracks just those instead of accumulating whole asset maps.
	acc := randomImproveTotal{amounts: make([]*big.Int, len(targets))}
	for k := range acc.amounts {
		acc.amounts[k] = new(big.Int)
	}
	selected := make([]bool, len(cands))
	var picks []int
	for i, idx := range randomImproveShuffle(len(cands), s.Seed^randomImproveCoverSalt) {
		if acc.covers(target.Coin, targets) {
			break
		}
		if i%selectionCancelStride == 0 {
			if err := selectionInterrupted(ctx); err != nil {
				return nil, err
			}
		}
		cand := &cands[idx]
		if cand.coin > math.MaxUint64-acc.coin {
			return nil, fmt.Errorf(
				"selection value overflow after adding UTxO %s",
				cand.ref,
			)
		}
		acc.add(cand)
		selected[idx] = true
		picks = append(picks, idx)
	}
	if !acc.covers(target.Coin, targets) {
		return nil, ErrInsufficientUtxos
	}

	// A random cover can take a small input before a large one that covers
	// on its own. Dropping it matters most for token UTxOs: surplus tokens
	// become change that needs min-UTxO lovelace the wallet may not have.
	kept := picks[:0]
	for i, idx := range picks {
		if i%selectionCancelStride == 0 {
			if err := selectionInterrupted(ctx); err != nil {
				return nil, err
			}
		}
		if acc.coversWithout(&cands[idx], target.Coin, targets) {
			acc.sub(&cands[idx])
			selected[idx] = false
			continue
		}
		kept = append(kept, idx)
	}
	picks = kept

	// A target above half the uint64 range has no representable ideal; the
	// cover is still valid, so improvement is skipped.
	if target.Coin <= math.MaxUint64/2 {
		ideal := target.Coin * 2
		for i, idx := range randomImproveShuffle(len(cands), s.Seed^randomImproveImproveSalt) {
			if i%selectionCancelStride == 0 {
				if err := selectionInterrupted(ctx); err != nil {
					return nil, err
				}
			}
			cand := &cands[idx]
			// Within the ideal, every added lovelace moves the total closer
			// to it, so the CIP-0002 distance test reduces to this bound.
			if selected[idx] || !cand.adaOnly || cand.coin == 0 ||
				cand.coin > ideal-acc.coin {
				continue
			}
			acc.coin += cand.coin
			selected[idx] = true
			picks = append(picks, idx)
		}
	}

	result := make([]common.Utxo, 0, len(picks))
	for _, idx := range picks {
		result = append(result, cands[idx].utxo)
	}
	return result, nil
}

// RandomImproveThenLargestFirstSelector follows the Cardano Wallet strategy
// described in CIP-0002: it tries Random-Improve and falls back to
// Largest-First only when Random-Improve reports ErrInsufficientUtxos. Any
// other error, including cancellation, is returned without a fallback.
type RandomImproveThenLargestFirstSelector struct {
	// RandomImprove is the primary selector. Nil means
	// NewRandomImproveSelector().
	RandomImprove CoinSelector
	// Fallback runs when RandomImprove returns an error matching
	// ErrInsufficientUtxos. Nil means LargestFirstSelector.
	Fallback CoinSelector
}

// NewRandomImproveThenLargestFirstSelector returns a selector that tries
// Random-Improve before Largest-First.
func NewRandomImproveThenLargestFirstSelector() *RandomImproveThenLargestFirstSelector {
	return &RandomImproveThenLargestFirstSelector{}
}

// Name returns the algorithm's identifier.
func (s *RandomImproveThenLargestFirstSelector) Name() string {
	return "random-improve-largest-first"
}

// Select returns a subset of available whose summed value covers target.
func (s *RandomImproveThenLargestFirstSelector) Select(
	ctx context.Context,
	available []common.Utxo,
	target Value,
) ([]common.Utxo, error) {
	primary := s.RandomImprove
	if primary == nil {
		primary = NewRandomImproveSelector()
	}
	selected, err := primary.Select(ctx, available, target)
	if err == nil {
		return selected, nil
	}
	if !errors.Is(err, ErrInsufficientUtxos) {
		return nil, fmt.Errorf("%s failed: %w", primary.Name(), err)
	}
	fallback := s.Fallback
	if fallback == nil {
		fallback = &LargestFirstSelector{}
	}
	selected, fallbackErr := fallback.Select(ctx, available, target)
	if fallbackErr != nil {
		return nil, fmt.Errorf(
			"%s failed: %w; %s failed: %w",
			primary.Name(), err, fallback.Name(), fallbackErr,
		)
	}
	return selected, nil
}

type randomImproveAsset struct {
	policy common.Blake2b224
	name   []byte
	qty    *big.Int
}

// randomImproveTargets lists the target's positive asset quantities in
// policy and name order.
func randomImproveTargets(target Value) []randomImproveAsset {
	if target.Assets == nil {
		return nil
	}
	var out []randomImproveAsset
	for _, policy := range target.Assets.Policies() {
		for _, name := range target.Assets.Assets(policy) {
			qty := target.Assets.Asset(policy, name)
			if qty == nil || qty.Sign() <= 0 {
				continue
			}
			out = append(out, randomImproveAsset{policy: policy, name: name, qty: qty})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if c := bytes.Compare(out[i].policy.Bytes(), out[j].policy.Bytes()); c != 0 {
			return c < 0
		}
		return bytes.Compare(out[i].name, out[j].name) < 0
	})
	return out
}

type randomImproveCandidate struct {
	utxo common.Utxo
	ref  string
	coin uint64
	// amounts holds the UTxO's quantity of each targeted asset, indexed
	// like the targets; nil when it holds none of them.
	amounts []*big.Int
	adaOnly bool
}

// randomImproveCandidates returns the pool sorted by ref, so the shuffle
// depends only on the seed and the pool's contents, not its order.
func randomImproveCandidates(
	ctx context.Context,
	available []common.Utxo,
	targets []randomImproveAsset,
) ([]randomImproveCandidate, error) {
	cands := make([]randomImproveCandidate, 0, len(available))
	for i := range available {
		if i%selectionCancelStride == 0 {
			if err := selectionInterrupted(ctx); err != nil {
				return nil, err
			}
		}
		amt := available[i].Output.Amount()
		if amt == nil || !amt.IsUint64() {
			return nil, fmt.Errorf(
				"UTxO %s has an invalid lovelace amount",
				utxoRef(available[i]),
			)
		}
		assets := available[i].Output.Assets()
		cand := randomImproveCandidate{
			utxo:    available[i],
			ref:     utxoRef(available[i]),
			coin:    amt.Uint64(),
			adaOnly: assets == nil || MultiAssetIsEmpty(assets),
		}
		if !cand.adaOnly {
			for k, t := range targets {
				qty := assets.Asset(t.policy, t.name)
				if qty == nil || qty.Sign() <= 0 {
					continue
				}
				if cand.amounts == nil {
					cand.amounts = make([]*big.Int, len(targets))
				}
				cand.amounts[k] = qty
			}
		}
		cands = append(cands, cand)
	}
	sort.Slice(cands, func(i, j int) bool {
		return cands[i].ref < cands[j].ref
	})
	return cands, nil
}

// randomImproveTotal is the selection's lovelace and targeted-asset total.
// Its big.Ints are private to one Select call, so they are updated in place.
type randomImproveTotal struct {
	coin    uint64
	amounts []*big.Int
}

// add assumes the caller has checked coin overflow.
func (t *randomImproveTotal) add(c *randomImproveCandidate) {
	t.coin += c.coin
	for k, qty := range c.amounts {
		if qty != nil {
			t.amounts[k].Add(t.amounts[k], qty)
		}
	}
}

func (t *randomImproveTotal) sub(c *randomImproveCandidate) {
	t.coin -= c.coin
	for k, qty := range c.amounts {
		if qty != nil {
			t.amounts[k].Sub(t.amounts[k], qty)
		}
	}
}

func (t *randomImproveTotal) covers(coin uint64, targets []randomImproveAsset) bool {
	if t.coin < coin {
		return false
	}
	for k, target := range targets {
		if t.amounts[k].Cmp(target.qty) < 0 {
			return false
		}
	}
	return true
}

// coversWithout reports whether the total still covers the target once c,
// which is part of it, is removed.
func (t *randomImproveTotal) coversWithout(
	c *randomImproveCandidate,
	coin uint64,
	targets []randomImproveAsset,
) bool {
	if t.coin-c.coin < coin {
		return false
	}
	var rest big.Int
	for k, qty := range c.amounts {
		if qty == nil {
			continue
		}
		if rest.Sub(t.amounts[k], qty).Cmp(targets[k].qty) < 0 {
			return false
		}
	}
	return true
}

// randomImproveShuffle returns a permutation of 0..n-1. A math/rand Source
// with a fixed seed yields the same sequence on every platform and Go
// release, which keeps the selection reproducible.
func randomImproveShuffle(n int, seed int64) []int {
	order := make([]int, n)
	for i := range order {
		order[i] = i
	}
	rng := rand.New(rand.NewSource(seed)) //nolint:gosec // deterministic ordering, not a secret
	rng.Shuffle(n, func(i, j int) {
		order[i], order[j] = order[j], order[i]
	})
	return order
}

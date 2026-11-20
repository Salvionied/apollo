package apollo

import (
	"bytes"
	"context"
	"errors"
	"fmt"
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
// covers the target by taking UTxOs in random order, then adds further UTxOs
// that move the selection closer to an ideal of twice the target.
//
// It differs from CIP-0002 in two ways: it selects for the aggregate target
// rather than for each output, and an improvement may not take any targeted
// asset class above twice its target, where CIP-0002 allows up to three
// times. During improvement it never adds a UTxO carrying an asset the target
// does not ask for.
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
	cands, err := randomImproveCandidates(ctx, available)
	if err != nil {
		return nil, err
	}

	coverOrder := randomImproveShuffle(cands, s.Seed^randomImproveCoverSalt)
	selected := make([]bool, len(cands))
	var result []common.Utxo
	total := Value{}
	for i, idx := range coverOrder {
		if total.GreaterOrEqual(target) {
			break
		}
		if i%selectionCancelStride == 0 {
			if err := selectionInterrupted(ctx); err != nil {
				return nil, err
			}
		}
		cand := cands[idx]
		total, err = total.Add(cand.value)
		if err != nil {
			return nil, fmt.Errorf(
				"selection value overflow after adding UTxO %s: %w",
				cand.ref, err,
			)
		}
		selected[idx] = true
		result = append(result, cand.utxo)
	}
	if !total.GreaterOrEqual(target) {
		return nil, ErrInsufficientUtxos
	}

	ideal, err := target.Add(target)
	if err != nil {
		// A target above half the uint64 range has no representable ideal.
		// The cover is still valid, so skip the optional improvement rather
		// than fail a satisfiable request.
		return result, nil
	}
	improveOrder := randomImproveShuffle(cands, s.Seed^randomImproveImproveSalt)
	for i, idx := range improveOrder {
		if i%selectionCancelStride == 0 {
			if err := selectionInterrupted(ctx); err != nil {
				return nil, err
			}
		}
		cand := cands[idx]
		if selected[idx] || randomImproveHasUntargetedAssets(cand.value, target) {
			continue
		}
		next, err := total.Add(cand.value)
		if err != nil {
			return nil, fmt.Errorf(
				"selection value overflow after adding UTxO %s: %w",
				cand.ref, err,
			)
		}
		if !randomImproveWithinIdeal(next, ideal) ||
			randomImproveDistance(next, ideal).Cmp(
				randomImproveDistance(total, ideal),
			) >= 0 {
			continue
		}
		total = next
		selected[idx] = true
		result = append(result, cand.utxo)
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

type randomImproveCandidate struct {
	utxo  common.Utxo
	ref   string
	value Value
}

// randomImproveCandidates returns the pool sorted by ref, so the shuffle
// depends only on the seed and the pool's contents, not its order.
func randomImproveCandidates(
	ctx context.Context,
	available []common.Utxo,
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
		cands = append(cands, randomImproveCandidate{
			utxo: available[i],
			ref:  utxoRef(available[i]),
			value: NewValue(
				amt.Uint64(),
				CloneMultiAsset(available[i].Output.Assets()),
			),
		})
	}
	sort.Slice(cands, func(i, j int) bool {
		return cands[i].ref < cands[j].ref
	})
	return cands, nil
}

// randomImproveShuffle returns a permutation of the candidate indexes. A
// math/rand Source with a fixed seed yields the same sequence on every
// platform and Go release, which keeps the selection reproducible.
func randomImproveShuffle(cands []randomImproveCandidate, seed int64) []int {
	order := make([]int, len(cands))
	for i := range order {
		order[i] = i
	}
	rng := rand.New(rand.NewSource(seed)) //nolint:gosec // deterministic ordering, not a secret
	rng.Shuffle(len(order), func(i, j int) {
		order[i], order[j] = order[j], order[i]
	})
	return order
}

func randomImproveHasUntargetedAssets(value Value, target Value) bool {
	if value.Assets == nil {
		return false
	}
	for _, policy := range value.Assets.Policies() {
		for _, name := range value.Assets.Assets(policy) {
			qty := value.Assets.Asset(policy, name)
			if qty == nil || qty.Sign() <= 0 {
				continue
			}
			if target.Assets == nil {
				return true
			}
			targetQty := target.Assets.Asset(policy, name)
			if targetQty == nil || targetQty.Sign() <= 0 {
				return true
			}
		}
	}
	return false
}

func randomImproveWithinIdeal(value Value, ideal Value) bool {
	if value.Coin > ideal.Coin {
		return false
	}
	if ideal.Assets == nil || value.Assets == nil {
		return true
	}
	for _, policy := range ideal.Assets.Policies() {
		for _, name := range ideal.Assets.Assets(policy) {
			idealQty := ideal.Assets.Asset(policy, name)
			if idealQty == nil || idealQty.Sign() <= 0 {
				continue
			}
			valueQty := value.Assets.Asset(policy, name)
			if valueQty != nil && valueQty.Cmp(idealQty) > 0 {
				return false
			}
		}
	}
	return true
}

// randomImproveDistance sums the absolute difference from ideal over lovelace
// and every targeted asset. Assets are visited in sorted order so the big.Int
// arithmetic is the same on every run.
func randomImproveDistance(value Value, ideal Value) *big.Int {
	dist := new(big.Int).SetUint64(absDiffUint64(value.Coin, ideal.Coin))
	if ideal.Assets == nil {
		return dist
	}
	policies := ideal.Assets.Policies()
	sort.Slice(policies, func(i, j int) bool {
		return bytes.Compare(policies[i].Bytes(), policies[j].Bytes()) < 0
	})
	for _, policy := range policies {
		names := ideal.Assets.Assets(policy)
		sort.Slice(names, func(i, j int) bool {
			return bytes.Compare(names[i], names[j]) < 0
		})
		for _, name := range names {
			idealQty := ideal.Assets.Asset(policy, name)
			if idealQty == nil || idealQty.Sign() <= 0 {
				continue
			}
			valueQty := big.NewInt(0)
			if value.Assets != nil {
				if qty := value.Assets.Asset(policy, name); qty != nil {
					valueQty = qty
				}
			}
			diff := new(big.Int).Sub(idealQty, valueQty)
			dist.Add(dist, diff.Abs(diff))
		}
	}
	return dist
}

func absDiffUint64(a, b uint64) uint64 {
	if a > b {
		return a - b
	}
	return b - a
}

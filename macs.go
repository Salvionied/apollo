package apollo

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"sort"

	"github.com/blinklabs-io/gouroboros/ledger/common"
)

// MACSSelector implements the Multi-Asset Coin Selection algorithm from
// "MACS: A Multi-Asset Coin Selection Algorithm for UTXO-based Blockchains"
// (Ramezan, Schneider, McCann — IEEE Blockchain 2023,
// DOI 10.1109/Blockchain60715.2023.00029). Each deficient asset class in the
// target (native assets first, lovelace last) is covered by repeatedly
// picking the unselected UTxO with the highest priority
//
//	P(u,c) = v(u,c) / (|v(u,c) - avg(S,c)| + 1)
//
// which favors valuable UTxOs near the pool-wide average value, keeping
// change small and the wallet's UTxO pool diverse. The paper's confirmation
// age and linkage factors are constant here because common.Utxo carries no
// such metadata. A final pass drops inputs made redundant by later picks,
// lowest priority first. Priorities are compared exactly via big.Int
// cross-multiplication, so selection is deterministic.
//
// The paper's dust-avoidance requirement is met with a bounded sweep: when a
// selection exists and sweeping is enabled, up to MaxDustInputs ADA-only
// UTxOs below DustThreshold lovelace are appended (smallest first), so dust
// is consolidated into change instead of accumulating. The zero value runs
// the pure algorithm with no sweeping; NewMACSSelector returns the
// recommended configuration.
type MACSSelector struct {
	// DustThreshold is the lovelace amount below which an ADA-only UTxO is
	// considered dust and eligible for sweeping. Zero disables sweeping.
	DustThreshold uint64
	// MaxDustInputs caps how many dust UTxOs one selection may sweep.
	MaxDustInputs int
}

// NewMACSSelector returns a MACS selector with dust sweeping enabled:
// UTxOs below 1 ADA (the order of Cardano's minimum UTxO value) are swept,
// at most two per selection.
func NewMACSSelector() *MACSSelector {
	return &MACSSelector{
		DustThreshold: 1_000_000,
		MaxDustInputs: 2,
	}
}

// Name returns the algorithm's identifier.
func (s *MACSSelector) Name() string { return "macs" }

// macsClass identifies one asset class of the target: a (policy, name) pair
// or the chain's native coin.
type macsClass struct {
	policy common.Blake2b224
	name   []byte
	isCoin bool
}

type macsCandidate struct {
	utxo common.Utxo
	ref  string
	coin uint64
}

// value returns the candidate's quantity of the given asset class. The
// returned big.Int must not be mutated: it may alias the UTxO's asset map.
func (c *macsCandidate) value(cls macsClass) *big.Int {
	if cls.isCoin {
		return new(big.Int).SetUint64(c.coin)
	}
	assets := c.utxo.Output.Assets()
	if assets == nil {
		return big.NewInt(0)
	}
	qty := assets.Asset(cls.policy, cls.name)
	if qty == nil || qty.Sign() <= 0 {
		return big.NewInt(0)
	}
	return qty
}

// macsPick records a selected candidate with the priority terms it was
// chosen under, for the pruning pass.
type macsPick struct {
	cand   *macsCandidate
	v, dev *big.Int
	pruned bool
}

// Select returns a subset of available whose summed value covers target.
func (s *MACSSelector) Select(available []common.Utxo, target Value) ([]common.Utxo, error) {
	remaining := target.Clone()
	if remaining.Coin == 0 && !remaining.HasAssets() {
		return nil, nil
	}

	// MACS reads every amount to compute pool averages, so the whole pool is
	// validated up front. Amounts come from a remote backend; reject anything
	// outside the uint64 lovelace range (big.Int.Uint64 is undefined out of range).
	cands := make([]*macsCandidate, 0, len(available))
	for i := range available {
		amt := available[i].Output.Amount()
		if amt == nil || !amt.IsUint64() {
			return nil, fmt.Errorf("UTxO %s has an invalid lovelace amount", utxoRef(available[i]))
		}
		cands = append(cands, &macsCandidate{
			utxo: available[i],
			ref:  utxoRef(available[i]),
			coin: amt.Uint64(),
		})
	}

	classes := macsTargetClasses(target)
	avgs := macsClassAverages(classes, cands)

	selected := make(map[string]bool)
	var picks []*macsPick
	for {
		clsIdx, deficient := macsFirstDeficit(remaining, classes)
		if !deficient {
			break
		}
		pick := macsBestCandidate(cands, selected, classes[clsIdx], avgs[clsIdx])
		if pick == nil {
			return nil, errors.New("insufficient UTxOs to cover required value")
		}
		selected[pick.cand.ref] = true
		picks = append(picks, pick)

		if remaining.Coin <= pick.cand.coin {
			remaining.Coin = 0
		} else {
			remaining.Coin -= pick.cand.coin
		}
		if remaining.Assets != nil && pick.cand.utxo.Output.Assets() != nil {
			subtractAssetsSaturating(remaining.Assets, pick.cand.utxo.Output.Assets())
		}
	}

	macsPruneRedundant(picks, classes, target)

	result := make([]common.Utxo, 0, len(picks))
	selectedAfterPrune := make(map[string]bool, len(picks))
	for _, p := range picks {
		if !p.pruned {
			result = append(result, p.cand.utxo)
			selectedAfterPrune[p.cand.ref] = true
		}
	}
	result = s.sweepDust(result, selectedAfterPrune, cands)
	return result, nil
}

// sweepDust appends up to MaxDustInputs unselected ADA-only UTxOs below
// DustThreshold, smallest first (ties by ref), consolidating dust into the
// transaction's change. Token-carrying UTxOs are never swept so change
// outputs do not accumulate assets the target did not ask for.
func (s *MACSSelector) sweepDust(result []common.Utxo, selected map[string]bool, cands []*macsCandidate) []common.Utxo {
	if s.DustThreshold == 0 || s.MaxDustInputs <= 0 || len(result) == 0 {
		return result
	}
	var dust []*macsCandidate
	for _, c := range cands {
		if selected[c.ref] || c.coin >= s.DustThreshold {
			continue
		}
		if c.utxo.Output.Assets() != nil {
			continue
		}
		dust = append(dust, c)
	}
	sort.Slice(dust, func(i, j int) bool {
		if dust[i].coin != dust[j].coin {
			return dust[i].coin < dust[j].coin
		}
		return dust[i].ref < dust[j].ref
	})
	for i := 0; i < len(dust) && i < s.MaxDustInputs; i++ {
		result = append(result, dust[i].utxo)
	}
	return result
}

// macsTargetClasses returns the target's asset classes in deterministic
// order: native assets sorted by policy then name, lovelace last (most UTxOs
// carry lovelace incidentally, so covering assets first minimizes inputs).
func macsTargetClasses(target Value) []macsClass {
	var classes []macsClass
	if target.Assets != nil {
		policies := target.Assets.Policies()
		sort.Slice(policies, func(i, j int) bool {
			return bytes.Compare(policies[i].Bytes(), policies[j].Bytes()) < 0
		})
		for _, policy := range policies {
			names := target.Assets.Assets(policy)
			sort.Slice(names, func(i, j int) bool {
				return bytes.Compare(names[i], names[j]) < 0
			})
			for _, name := range names {
				qty := target.Assets.Asset(policy, name)
				if qty == nil || qty.Sign() <= 0 {
					continue
				}
				classes = append(classes, macsClass{policy: policy, name: name})
			}
		}
	}
	return append(classes, macsClass{isCoin: true})
}

// macsClassAverages computes avg(S,c) for each class over the whole pool,
// counting UTxOs that lack the asset as zero, per the paper.
func macsClassAverages(classes []macsClass, cands []*macsCandidate) []*big.Int {
	avgs := make([]*big.Int, len(classes))
	n := big.NewInt(int64(len(cands)))
	for i, cls := range classes {
		total := big.NewInt(0)
		for _, c := range cands {
			total.Add(total, c.value(cls))
		}
		if len(cands) > 0 {
			total.Div(total, n)
		}
		avgs[i] = total
	}
	return avgs
}

// macsFirstDeficit returns the index of the first class the remaining target
// still needs, in class order.
func macsFirstDeficit(remaining Value, classes []macsClass) (int, bool) {
	for i, cls := range classes {
		if cls.isCoin {
			if remaining.Coin > 0 {
				return i, true
			}
			continue
		}
		if remaining.Assets == nil {
			continue
		}
		qty := remaining.Assets.Asset(cls.policy, cls.name)
		if qty != nil && qty.Sign() > 0 {
			return i, true
		}
	}
	return 0, false
}

// macsBestCandidate returns the unselected candidate holding the class with
// the highest priority, or nil if none holds it.
func macsBestCandidate(cands []*macsCandidate, selected map[string]bool, cls macsClass, avg *big.Int) *macsPick {
	var best *macsPick
	for _, c := range cands {
		if selected[c.ref] {
			continue
		}
		v := c.value(cls)
		if v.Sign() <= 0 {
			continue
		}
		dev := new(big.Int).Sub(v, avg)
		dev.Abs(dev)
		if best == nil || macsBetter(v, dev, c.ref, best.v, best.dev, best.cand.ref) {
			best = &macsPick{cand: c, v: v, dev: dev}
		}
	}
	return best
}

// macsBetter reports whether priority v1/(d1+1) beats v2/(d2+1), compared
// exactly by cross-multiplication. Ties prefer the larger value, then the
// smaller UTxO ref, so selection is deterministic.
func macsBetter(v1, d1 *big.Int, ref1 string, v2, d2 *big.Int, ref2 string) bool {
	one := big.NewInt(1)
	left := new(big.Int).Add(d2, one)
	left.Mul(left, v1)
	right := new(big.Int).Add(d1, one)
	right.Mul(right, v2)
	if cmp := left.Cmp(right); cmp != 0 {
		return cmp > 0
	}
	if cmp := v1.Cmp(v2); cmp != 0 {
		return cmp > 0
	}
	return ref1 < ref2
}

// macsPruneRedundant drops picks whose removal keeps the target covered,
// trying lowest-priority picks first. Greedy per-class coverage can select a
// UTxO for one asset that a later pick (chosen for a different asset) also
// carries, leaving the earlier pick redundant.
func macsPruneRedundant(picks []*macsPick, classes []macsClass, target Value) {
	if len(picks) < 2 {
		return
	}

	// Per-class totals over the current (unpruned) selection and the
	// required quantities.
	sums := make([]*big.Int, len(classes))
	need := make([]*big.Int, len(classes))
	for i, cls := range classes {
		sums[i] = big.NewInt(0)
		for _, p := range picks {
			sums[i].Add(sums[i], p.cand.value(cls))
		}
		if cls.isCoin {
			need[i] = new(big.Int).SetUint64(target.Coin)
		} else {
			need[i] = new(big.Int).Set(target.Assets.Asset(cls.policy, cls.name))
		}
	}

	order := make([]*macsPick, len(picks))
	copy(order, picks)
	sort.SliceStable(order, func(i, j int) bool {
		return macsBetter(order[j].v, order[j].dev, order[j].cand.ref, order[i].v, order[i].dev, order[i].cand.ref)
	})

	for _, p := range order {
		redundant := true
		for i, cls := range classes {
			rest := new(big.Int).Sub(sums[i], p.cand.value(cls))
			if rest.Cmp(need[i]) < 0 {
				redundant = false
				break
			}
		}
		if redundant {
			p.pruned = true
			for i, cls := range classes {
				sums[i].Sub(sums[i], p.cand.value(cls))
			}
		}
	}
}

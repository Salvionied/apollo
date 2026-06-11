package apollo

import (
	"errors"
	"fmt"

	"github.com/blinklabs-io/gouroboros/ledger/common"
)

// CoinSelector chooses UTxOs from an available pool to cover a target value.
// Implementations must be deterministic: the same pool and target must yield
// the same selection.
type CoinSelector interface {
	// Name returns the algorithm's identifier.
	Name() string
	// Select returns a subset of available whose summed value covers target.
	// The available pool has already been filtered of in-use UTxOs. It
	// returns an error if the pool cannot cover the target.
	Select(available []common.Utxo, target Value) ([]common.Utxo, error)
}

// defaultCoinSelector is used by Complete when no selector is configured.
// MACS is the default: benchmarks (see docs/design/2026-06-11-macs-coin-
// selection-design.md) show it selects far fewer inputs on multi-asset
// targets and produces much smaller change than largest-first, at comparable
// speed. Use SetCoinSelector(&LargestFirstSelector{}) for the legacy behavior.
var defaultCoinSelector CoinSelector = NewMACSSelector()

// LargestFirstSelector selects UTxOs greedily by descending lovelace amount,
// consuming ADA-only UTxOs before asset-carrying ones.
type LargestFirstSelector struct{}

// Name returns the algorithm's identifier.
func (s *LargestFirstSelector) Name() string { return "largest-first" }

// Select returns a subset of available whose summed value covers target.
func (s *LargestFirstSelector) Select(available []common.Utxo, target Value) ([]common.Utxo, error) {
	remaining := target.Clone()
	if remaining.Coin == 0 && !remaining.HasAssets() {
		return nil, nil
	}
	sorted := SortUtxos(available)
	var selected []common.Utxo
	for _, utxo := range sorted {
		amt := utxo.Output.Amount()
		// Amounts come from a remote backend; reject anything outside the
		// uint64 lovelace range (big.Int.Uint64 is undefined out of range).
		if amt == nil || !amt.IsUint64() {
			return nil, fmt.Errorf("UTxO %s has an invalid lovelace amount", utxoRef(utxo))
		}

		selected = append(selected, utxo)

		if remaining.Coin <= amt.Uint64() {
			remaining.Coin = 0
		} else {
			remaining.Coin -= amt.Uint64()
		}
		if remaining.Assets != nil && utxo.Output.Assets() != nil {
			subtractAssetsSaturating(remaining.Assets, utxo.Output.Assets())
		}

		if remaining.Coin == 0 && !remaining.HasAssets() {
			return selected, nil
		}
	}
	return nil, errors.New("insufficient UTxOs to cover required value")
}

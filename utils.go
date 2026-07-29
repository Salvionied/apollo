package apollo

import (
	"encoding/hex"
	"sort"

	"github.com/blinklabs-io/gouroboros/ledger/common"
)

// SortUtxos sorts a slice of UTxOs with ADA-only UTxOs first (by descending amount),
// then UTxOs with assets.
func SortUtxos(utxos []common.Utxo) []common.Utxo {
	res := make([]common.Utxo, len(utxos))
	copy(res, utxos)
	sort.SliceStable(res, func(i, j int) bool {
		iValid := validateUtxo(res[i]) == nil
		jValid := validateUtxo(res[j]) == nil
		if iValid != jValid {
			return iValid
		}
		if !iValid {
			return false
		}
		iHasAssets := res[i].Output.Assets() != nil
		jHasAssets := res[j].Output.Assets() != nil
		if !iHasAssets && !jHasAssets {
			iAmt := res[i].Output.Amount()
			jAmt := res[j].Output.Amount()
			if iAmt != nil && jAmt != nil {
				return iAmt.Cmp(jAmt) > 0
			}
			return false
		}
		if iHasAssets && jHasAssets {
			iAmt := res[i].Output.Amount()
			jAmt := res[j].Output.Amount()
			if iAmt != nil && jAmt != nil {
				return iAmt.Cmp(jAmt) > 0
			}
			return false
		}
		return jHasAssets
	})
	return res
}

// SortInputs sorts UTxOs by transaction ID and index for deterministic ordering.
func SortInputs(inputs []common.Utxo) []common.Utxo {
	sorted := make([]common.Utxo, len(inputs))
	copy(sorted, inputs)
	sort.SliceStable(sorted, func(i, j int) bool {
		iValid := validateUtxo(sorted[i]) == nil
		jValid := validateUtxo(sorted[j]) == nil
		if iValid != jValid {
			return iValid
		}
		if !iValid {
			return false
		}
		iId := hex.EncodeToString(sorted[i].Id.Id().Bytes())
		jId := hex.EncodeToString(sorted[j].Id.Id().Bytes())
		if iId != jId {
			return iId < jId
		}
		return sorted[i].Id.Index() < sorted[j].Id.Index()
	})
	return sorted
}

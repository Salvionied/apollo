package apollo

import (
	"encoding/hex"
	"sort"

	"github.com/SundaeSwap-finance/apollo/serialization/UTxO"
)

func SortUtxos(utxos []UTxO.UTxO) []UTxO.UTxO {
	res := make([]UTxO.UTxO, len(utxos))
	copy(res, utxos)
	// Sort UTXOs first by large ADA-only UTXOs, then by assets
	sort.Slice(res, func(i, j int) bool {
		if !res[i].Output.GetValue().HasAssets && !res[j].Output.GetValue().HasAssets {
			return res[i].Output.Lovelace() > res[j].Output.Lovelace()
		} else if res[i].Output.GetValue().HasAssets && res[j].Output.GetValue().HasAssets {
			return res[i].Output.GetAmount().Greater(res[j].Output.GetAmount())
		} else {
			return res[j].Output.GetAmount().HasAssets
		}
	})
	return res
}

type txIn struct {
	id string
	ix int
}

func SortInputs(inputs []UTxO.UTxO) []UTxO.UTxO {
	sortedInputs := make([]UTxO.UTxO, 0)
	sortedInputs = append(sortedInputs, inputs...)
	sort.Slice(sortedInputs, func(i, j int) bool {
		iTxId := hex.EncodeToString(sortedInputs[i].Input.TransactionId)
		jTxId := hex.EncodeToString(sortedInputs[j].Input.TransactionId)
		if iTxId != jTxId {
			return iTxId < jTxId
		} else {
			return sortedInputs[i].Input.Index < sortedInputs[j].Input.Index
		}
	})
	return sortedInputs
}

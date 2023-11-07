package apollo

import (
	"sort"

	"github.com/Salvionied/apollo/serialization/UTxO"
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

func SortInputs(inputs []UTxO.UTxO) []UTxO.UTxO {
	hashes := make([]string, 0)
	relationMap := map[string]UTxO.UTxO{}
	for _, utxo := range inputs {
		hashes = append(hashes, string(utxo.Input.String()))
		relationMap[string(utxo.Input.String())] = utxo
	}
	sort.Strings(hashes)
	sorted_inputs := make([]UTxO.UTxO, 0)
	for _, hash := range hashes {
		sorted_inputs = append(sorted_inputs, relationMap[hash])
	}
	return sorted_inputs
}

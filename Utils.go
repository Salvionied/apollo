package apollo

import (
	"sort"

	"github.com/Salvionied/apollo/serialization/UTxO"
)

func SortUtxos(utxos []UTxO.UTxO) []UTxO.UTxO {
	res := make([]UTxO.UTxO, len(utxos))
	copy(res, utxos)
	for i := 0; i < len(res); i++ {
		for j := i + 1; j < len(res); j++ {
			if res[i].Output.GetAmount().Greater(res[j].Output.GetAmount()) {
				res[i], res[j] = res[j], res[i]
			}
		}
	}
	return res
}

func SortInputs(inputs []UTxO.UTxO) []UTxO.UTxO {
	hashes := make([]string, 0)
	relationMap := map[string]UTxO.UTxO{}
	for _, utxo := range inputs {
		hashes = append(hashes, string(utxo.Input.TransactionId))
		relationMap[string(utxo.Input.TransactionId)] = utxo
	}
	sort.Strings(hashes)
	sorted_inputs := make([]UTxO.UTxO, 0)
	for _, hash := range hashes {
		sorted_inputs = append(sorted_inputs, relationMap[hash])
	}
	return sorted_inputs
}

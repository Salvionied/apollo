package apollo

import (
	"sort"

	"github.com/Salvionied/apollo/serialization/UTxO"
)

/**
	SortUtxos sorts a slice of UTxO objects in descending order
	based on their amounts.

	Params:
		utxos ([]UTxO.UTxO): A slice of UTxO objects to be sorted.

	Returns:
		[]UTxO.UTxO: A new slice of UTxO objects sorted by descending amounts.
*/	
func SortUtxos(utxos []UTxO.UTxO) []UTxO.UTxO {
	res := make([]UTxO.UTxO, len(utxos))
	copy(res, utxos)
	for i := 0; i < len(res); i++ {
		for j := i + 1; j < len(res); j++ {
			if res[i].Output.GetAmount().Less(res[j].Output.GetAmount()) {
				res[i], res[j] = res[j], res[i]
			}
		}
	}
	return res
}

/**
	SortInputs sorts a slice of UTxO objects based on their strings.

	Params:
		inputs ([]UTxO.UTxO): A slice of UTxO objects to be sorted.

	Returns:
		[]UTxO.UTxO: A new slice of UTxO objects sorted based on input strings.
*/
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

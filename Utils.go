package apollo

import (
	"encoding/hex"
	"sort"

	"github.com/Salvionied/apollo/v2/serialization/UTxO"
)

/*
*
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
	// Sort UTXOs first by large ADA-only UTXOs, then by assets
	sort.Slice(res, func(i, j int) bool {
		if !res[i].Output.GetValue().HasAssets &&
			!res[j].Output.GetValue().HasAssets {
			return res[i].Output.Lovelace() > res[j].Output.Lovelace()
		} else if res[i].Output.GetValue().HasAssets && res[j].Output.GetValue().HasAssets {
			return res[i].Output.GetAmount().Greater(res[j].Output.GetAmount())
		} else {
			return res[j].Output.GetAmount().HasAssets
		}
	})
	return res
}

/*
*

	SortInputs sorts a slice of UTxO objects based on their strings.

	Params:
		inputs ([]UTxO.UTxO): A slice of UTxO objects to be sorted.

	Returns:
		[]UTxO.UTxO: A new slice of UTxO objects sorted based on input strings.
*/
func SortInputs(inputs []UTxO.UTxO) []UTxO.UTxO {
	sortedInputs := make([]UTxO.UTxO, 0, len(inputs))
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

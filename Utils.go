package apollo

import (
	"encoding/hex"
	"sort"
	"strconv"

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
	txIns := make([]txIn, 0)
	relationMap := map[string]UTxO.UTxO{}
	for _, utxo := range inputs {
		newTxIn := txIn{
			id: hex.EncodeToString(utxo.Input.TransactionId),
			ix: utxo.Input.Index,
		}
		txIns = append(txIns, newTxIn)
		key := newTxIn.id + strconv.Itoa(newTxIn.ix)
		relationMap[key] = utxo
	}
	sort.Slice(txIns, func(i, j int) bool {
		if txIns[i].id < txIns[j].id {
			return true
		} else if txIns[i].id > txIns[j].id {
			return false
		} else {
			return txIns[i].ix < txIns[j].ix
		}
	})
	sorted_inputs := make([]UTxO.UTxO, 0)
	for _, txIn := range txIns {
		key := txIn.id + strconv.Itoa(txIn.ix)
		sorted_inputs = append(sorted_inputs, relationMap[key])
	}
	return sorted_inputs
}

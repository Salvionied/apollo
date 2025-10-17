package OgmiosChainContext

import (
	"bytes"
	"testing"

	"github.com/SundaeSwap-finance/ogmigo/v6/ouroboros/chainsync/num"
	"github.com/SundaeSwap-finance/ogmigo/v6/ouroboros/shared"
)

func TestRoundtripUtxo(t *testing.T) {
	ogmigoUtxo := shared.Utxo{
		Transaction: shared.UtxoTxID{
			ID: "1234",
		},
		Index:   uint32(1),
		Address: "addr_test1vqp4mmnx647vyutfwugav0yvxhl6pdkyg69x4xqzfl4vwwck92a9t",
		Value: map[string]map[string]num.Int{
			"ada": map[string]num.Int{
				"lovelace": num.Int64(2_000_000),
			},
			"99b071ce8580d6a3a11b4902145adb8bfd0d2a03935af8cf66403e15": map[string]num.Int{
				"524245525259": num.Int64(1_000_000_000),
			},
		},
		Datum:     "d8799fff",
		DatumHash: "",
		Script:    nil,
	}
	apolloUtxo := Utxo_OgmigoToApollo(ogmigoUtxo)
	roundtrip := Utxo_ApolloToOgmigo(apolloUtxo)
	if roundtrip.Transaction.ID != ogmigoUtxo.Transaction.ID {
		t.Fatalf("Transaction IDs don't match: %v,%v", roundtrip.Transaction.ID, ogmigoUtxo.Transaction.ID)
	}
	if roundtrip.Index != ogmigoUtxo.Index {
		t.Fatalf("Indices don't match: %v,%v", roundtrip.Index, ogmigoUtxo.Index)
	}
	if roundtrip.Address != ogmigoUtxo.Address {
		t.Fatalf("Addresses don't match: %v,%v", roundtrip.Address, ogmigoUtxo.Address)
	}
	if !shared.Equal(roundtrip.Value, ogmigoUtxo.Value) {
		t.Fatalf("Values don't match: %v,%v", roundtrip.Value, ogmigoUtxo.Value)
	}
	if roundtrip.Datum != ogmigoUtxo.Datum {
		t.Fatalf("Datums don't match: %v,%v", roundtrip.Datum, ogmigoUtxo.Datum)
	}
	if roundtrip.DatumHash != ogmigoUtxo.DatumHash {
		t.Fatalf("DatumHashes don't match: %v,%v", roundtrip.DatumHash, ogmigoUtxo.DatumHash)
	}
	if !bytes.Equal(roundtrip.Script, ogmigoUtxo.Script) {
		t.Fatalf("Scripts don't match: %v,%v", roundtrip.Script, ogmigoUtxo.Script)
	}
}

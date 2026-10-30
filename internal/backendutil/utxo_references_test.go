package backendutil

import (
	"testing"

	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/shelley"
)

func TestUtxoReferenceSetRejectsMissingAndDuplicateReferences(t *testing.T) {
	var references UtxoReferenceSet
	if err := references.Add(common.Utxo{}); err == nil {
		t.Fatal("expected a missing input reference to fail")
	}
	var txId common.Blake2b256
	txId[0] = 1
	utxo := common.Utxo{Id: shelley.ShelleyTransactionInput{TxId: txId, OutputIndex: 2}}
	if err := references.Add(utxo); err != nil {
		t.Fatal(err)
	}
	if err := references.Add(utxo); err == nil {
		t.Fatal("expected a duplicate input reference to fail")
	}
}

package Transaction

import (
	"fmt"

	"github.com/Salvionied/apollo/serialization"
	"github.com/Salvionied/apollo/serialization/Metadata"
	"github.com/Salvionied/apollo/serialization/TransactionBody"
	"github.com/Salvionied/apollo/serialization/TransactionWitnessSet"
	"github.com/Salvionied/cbor/v2"
)

type Transaction struct {
	_                     struct{} `cbor:",toarray"`
	TransactionBody       TransactionBody.TransactionBody
	TransactionWitnessSet TransactionWitnessSet.TransactionWitnessSet
	Valid                 bool
	AuxiliaryData         *Metadata.AuxiliaryData
}

func (tx *Transaction) Bytes() ([]byte, error) {
	cborred, err := cbor.Marshal(tx)
	if err != nil {
		return nil, fmt.Errorf("error marshaling transaction, %s", err)
	}
	return cborred, nil
}

func (tx *Transaction) Id() serialization.TransactionId {
	txId, _ := tx.TransactionBody.Id()
	return txId
}

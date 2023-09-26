package Transaction

import (
	"fmt"

	"github.com/Salvionied/cbor/v2"
	"github.com/SundaeSwap-finance/apollo/serialization"
	"github.com/SundaeSwap-finance/apollo/serialization/Metadata"
	"github.com/SundaeSwap-finance/apollo/serialization/TransactionBody"
	"github.com/SundaeSwap-finance/apollo/serialization/TransactionWitnessSet"
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

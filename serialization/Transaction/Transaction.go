package Transaction

import (
	"github.com/Salvionied/cbor/v2"
	"github.com/salvionied/apollo/serialization"
	"github.com/salvionied/apollo/serialization/Metadata"
	"github.com/salvionied/apollo/serialization/TransactionBody"
	"github.com/salvionied/apollo/serialization/TransactionWitnessSet"
)

type Transaction struct {
	_                     struct{} `cbor:",toarray"`
	TransactionBody       TransactionBody.TransactionBody
	TransactionWitnessSet TransactionWitnessSet.TransactionWitnessSet
	Valid                 bool
	AuxiliaryData         *Metadata.AuxiliaryData
}

func (tx *Transaction) Bytes() []byte {
	cborred, err := cbor.Marshal(tx)
	if err != nil {
		panic(err)
	}
	return cborred
}

func (tx *Transaction) Id() serialization.TransactionId {
	return tx.TransactionBody.Id()
}

package Transaction

import (
	"github.com/github.com/salvionied/apollo/serialization"
	"github.com/github.com/salvionied/apollo/serialization/Metadata"
	"github.com/github.com/salvionied/apollo/serialization/TransactionBody"
	"github.com/github.com/salvionied/apollo/serialization/TransactionWitnessSet"
)

type Transaction struct {
	_                     struct{} `cbor:",toarray"`
	TransactionBody       TransactionBody.TransactionBody
	TransactionWitnessSet TransactionWitnessSet.TransactionWitnessSet
	Valid                 bool
	AuxiliaryData         Metadata.AuxiliaryData
}

func (tx *Transaction) Id() serialization.TransactionId {
	return tx.TransactionBody.Id()
}

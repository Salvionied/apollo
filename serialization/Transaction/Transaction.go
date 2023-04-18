package Transaction

import (
	"Salvionied/apollo/serialization"
	"Salvionied/apollo/serialization/Metadata"
	"Salvionied/apollo/serialization/TransactionBody"
	"Salvionied/apollo/serialization/TransactionWitnessSet"
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

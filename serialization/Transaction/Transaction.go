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

func (tx *Transaction) Bytes() []byte {
	cborred, err := cbor.Marshal(tx)
	if err != nil {
		fmt.Println(err)
	}
	return cborred
}

func (tx *Transaction) Id() serialization.TransactionId {
	return tx.TransactionBody.Id()
}

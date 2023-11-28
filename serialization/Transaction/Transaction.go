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

func (tx *Transaction) Bytes() []byte {
	em, _ := cbor.CanonicalEncOptions().EncMode()
	cborred, err := em.Marshal(tx)
	if err != nil {
		fmt.Println(err)
	}
	return cborred
}

func (tx *Transaction) Id() serialization.TransactionId {
	return tx.TransactionBody.Id()
}

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


/**
	Bytes returns the CBOR-encoded byte representation
	of the Transaction.

	Returns:
		[]byte: The CBOR-encoded transaction bytes.
		error: An error if the Bytes fails.
*/
func (tx *Transaction) Bytes() ([]byte, error) {
	cborred, err := cbor.Marshal(tx)
	if err != nil {
		return nil, fmt.Errorf("error marshaling transaction, %s", err)
	}
	return cborred, nil
}

/**
	Id returns the unique identifier for the transaction.

	Returns:
		serialization.TransactionId: The transaction ID.
*/
func (tx *Transaction) Id() serialization.TransactionId {
	txId, _ := tx.TransactionBody.Id()
	return txId
}

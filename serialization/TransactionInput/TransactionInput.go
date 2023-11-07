package TransactionInput

import (
	"bytes"
	"encoding/hex"
	"strconv"
)

type TransactionInput struct {
	_             struct{} `cbor:",toarray"`
	TransactionId []byte
	Index         int
}

/**
	Clone returns a deep copy of the TransactionInput.

	Returns:
		TransactionInput: A deep copy of the TransactionInput.
*/
func (tx TransactionInput) Clone() TransactionInput {
	return TransactionInput{
		TransactionId: tx.TransactionId,
		Index:         tx.Index,
	}
}

/**
	EqualTo checks if the TransactionInput is equal to another TransactionInput.

	Params:
		other TransactionInput: The TransactionInput to compare.

	Returns:
		bool: True if the TransactionInput is equal to the other TransactionInput, false otherwise.
*/
func (tx TransactionInput) EqualTo(other TransactionInput) bool {
	return bytes.Equal(tx.TransactionId, other.TransactionId) && tx.Index == other.Index
}

/**
	LessThan checks if the TransacctionInput is less than
	another TransactionInput based on index.

	Params:
		other TransactionInput: The TransactionInput to compare.

	Returns:
		bool: True if the TransactionInput is less than the other TransactionInput, false otherwise.
*/
func (tx TransactionInput) LessThan(other TransactionInput) bool {
	return tx.Index < other.Index
}

/**
	String returns a string representationof the TransactionInput
	in the format "transaction_id.index".
*/
func (tx TransactionInput) String() string {
	return hex.EncodeToString(tx.TransactionId) + "." + strconv.Itoa(tx.Index)
}

// func (tx TransactionInput) Hash() string {
// 	final := append(tx.TransactionId[:], tx.Index)
// 	blake_2b, _ := blake2b.New(TRANSACTION_HASH_SIZE, final)
// 	KeyHash := blake_2b.Sum(make([]byte, 0))
// 	return string(KeyHash)
// }

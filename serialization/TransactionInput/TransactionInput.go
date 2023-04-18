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

func (tx TransactionInput) Clone() TransactionInput {
	return TransactionInput{
		TransactionId: tx.TransactionId,
		Index:         tx.Index,
	}
}

func (tx TransactionInput) EqualTo(other TransactionInput) bool {
	return bytes.Equal(tx.TransactionId, other.TransactionId) && tx.Index == other.Index
}

func (tx TransactionInput) LessThan(other TransactionInput) bool {
	return tx.Index < other.Index
}

func (tx TransactionInput) String() string {
	return hex.EncodeToString(tx.TransactionId) + "." + strconv.Itoa(tx.Index)
}

// func (tx TransactionInput) Hash() string {
// 	final := append(tx.TransactionId[:], tx.Index)
// 	blake_2b, _ := blake2b.New(TRANSACTION_HASH_SIZE, final)
// 	KeyHash := blake_2b.Sum(make([]byte, 0))
// 	return string(KeyHash)
// }

package TransactionInput

import (
	"bytes"
	"encoding/hex"
	"errors"
	"strconv"

	"github.com/blinklabs-io/gouroboros/cbor"
)

type TransactionInput struct {
	cbor.StructAsArray
	TransactionId []byte
	Index         int
}

/*
*

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

/*
*

	EqualTo checks if the TransactionInput is equal to another TransactionInput.

	Params:
		other TransactionInput: The TransactionInput to compare.

	Returns:


	bool: True if the TransactionInput is equal to the other TransactionInput, false otherwise.
*/
func (tx TransactionInput) EqualTo(other TransactionInput) bool {
	return bytes.Equal(tx.TransactionId, other.TransactionId) &&
		tx.Index == other.Index
}

/*
*

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

/*
*

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

func (ti *TransactionInput) UnmarshalCBOR(data []byte) error {
	var temp any
	_, err := cbor.Decode(data, &temp)
	if err != nil {
		return err
	}
	if m, ok := temp.(map[any]any); ok {
		// map case
		cleanM := make(map[any]any)
		for k, v := range m {
			if _, ok := k.([]any); !ok {
				cleanM[k] = v
			}
		}
		for k, v := range cleanM {
			var key = k
			if karr, ok := k.([]any); ok && len(karr) == 2 {
				if tag, ok := karr[0].(uint64); ok && tag == 0 {
					key = karr[1]
				}
			}
			switch key {
			case 0:
				ti.TransactionId = v.([]byte)
			case 1:
				ti.Index = int(v.(uint64))
			}
		}
	} else if arr, ok := temp.([]any); ok {
		// array case
		if len(arr) != 2 {
			return errors.New("expected array of 2 elements")
		}
		ti.TransactionId = arr[0].([]byte)
		ti.Index = int(arr[1].(uint64))
	} else {
		return errors.New("invalid ti CBOR")
	}
	return nil
}

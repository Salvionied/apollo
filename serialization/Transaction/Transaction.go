package Transaction

import (
	"errors"
	"fmt"

	"github.com/Salvionied/apollo/v2/serialization"
	"github.com/Salvionied/apollo/v2/serialization/Metadata"
	"github.com/Salvionied/apollo/v2/serialization/TransactionBody"
	"github.com/Salvionied/apollo/v2/serialization/TransactionWitnessSet"
	"github.com/blinklabs-io/gouroboros/cbor"
)

// cleanAndEncode cleans a map by removing keys that are slices (which can
// appear in certain CBOR encodings) and then encodes the result to CBOR bytes.
// For non-map values, it simply encodes them directly.
func cleanAndEncode(v any) ([]byte, error) {
	if vm, ok := v.(map[any]any); ok {
		cleanV := make(map[any]any)
		for vk, vv := range vm {
			if _, ok := vk.([]any); !ok {
				cleanV[vk] = vv
			}
		}
		return cbor.Encode(cleanV)
	}
	return cbor.Encode(v)
}

type Transaction struct {
	cbor.StructAsArray
	TransactionBody       TransactionBody.TransactionBody             `cbor:"0"`
	TransactionWitnessSet TransactionWitnessSet.TransactionWitnessSet `cbor:"1"`
	Valid                 bool                                        `cbor:"2"`
	AuxiliaryData         *Metadata.AuxiliaryData                     `cbor:"3,omitempty"`
}

// UnmarshalCBOR unmarshals a CBOR-encoded transaction into the Transaction struct.

/*
*

	Bytes returns the CBOR-encoded byte representation
	of the Transaction.

	Returns:
		[]byte: The CBOR-encoded transaction bytes.
		error: An error if the Bytes fails.
*/
func (tx *Transaction) Bytes() ([]byte, error) {
	cborred, err := cbor.Encode(tx)
	if err != nil {
		return nil, fmt.Errorf("error marshaling transaction, %w", err)
	}
	return cborred, nil
}

/*
*

	Id returns the unique identifier for the transaction.

	Returns:
		serialization.TransactionId: The transaction ID.
*/
func (tx *Transaction) Id() serialization.TransactionId {
	txId, _ := tx.TransactionBody.Id()
	return txId
}

/*
*

	UnmarshalCBOR unmarshals a CBOR-encoded transaction into the Transaction struct.

	Params:
		data ([]byte): The CBOR-encoded data.

	Returns:
		error: An error if unmarshaling fails.
*/
func (tx *Transaction) UnmarshalCBOR(data []byte) error {
	var decoded any
	_, err := cbor.Decode(data, &decoded)
	if err != nil {
		return err
	}

	// Handle tagged transaction
	if arr, ok := decoded.([]any); ok && len(arr) == 2 {
		if tag, ok := arr[0].(uint64); ok && tag == 258 {
			decoded = arr[1]
		}
	}

	if m, ok := decoded.(map[any]any); ok {
		// map case
		for k, v := range m {
			if _, ok := k.([]any); ok {
				continue
			}
			var key = k
			if karr, ok := k.([]any); ok && len(karr) == 2 {
				if tag, ok := karr[0].(uint64); ok && tag == 0 {
					key = karr[1]
				}
			}
			switch key {
			case 0:
				// body
				bodyBytes, err := cleanAndEncode(v)
				if err != nil {
					return err
				}
				if err := tx.TransactionBody.UnmarshalCBOR(bodyBytes); err != nil {
					return err
				}
			case 1:
				// witness set
				witnessBytes, err := cleanAndEncode(v)
				if err != nil {
					return err
				}
				if err := tx.TransactionWitnessSet.UnmarshalCBOR(witnessBytes); err != nil {
					return err
				}
			case 2:
				// valid
				valid, ok := v.(bool)
				if !ok {
					return errors.New("invalid valid field")
				}
				tx.Valid = valid
			case 3:
				// auxiliary data
				if v != nil {
					auxBytes, err := cleanAndEncode(v)
					if err != nil {
						return err
					}
					tx.AuxiliaryData = &Metadata.AuxiliaryData{}
					_, err = cbor.Decode(auxBytes, tx.AuxiliaryData)
					if err != nil {
						return err
					}
				}
			default:
				// ignore unknown fields
			}
		}
	} else if arr, ok := decoded.([]any); ok {
		// array case
		if len(arr) < 2 {
			return errors.New("invalid transaction")
		}
		// body
		bodyBytes, err := cleanAndEncode(arr[0])
		if err != nil {
			return err
		}
		if err := tx.TransactionBody.UnmarshalCBOR(bodyBytes); err != nil {
			return err
		}
		// witness set
		witnessBytes, err := cleanAndEncode(arr[1])
		if err != nil {
			return err
		}
		if err := tx.TransactionWitnessSet.UnmarshalCBOR(witnessBytes); err != nil {
			return err
		}
		if len(arr) > 2 {
			if arr[2] != nil {
				valid, ok := arr[2].(bool)
				if !ok {
					return errors.New("invalid valid field")
				}
				tx.Valid = valid
			}
		}
		if len(arr) > 3 {
			if arr[3] != nil {
				auxBytes, err := cleanAndEncode(arr[3])
				if err != nil {
					return err
				}
				tx.AuxiliaryData = &Metadata.AuxiliaryData{}
				_, err = cbor.Decode(auxBytes, tx.AuxiliaryData)
				if err != nil {
					return err
				}
			}
		}
	} else {
		return errors.New("invalid transaction CBOR")
	}
	return nil
}

package Transaction

import (
	"errors"
	"fmt"

	"github.com/Salvionied/apollo/serialization"
	"github.com/Salvionied/apollo/serialization/Metadata"
	"github.com/Salvionied/apollo/serialization/TransactionBody"
	"github.com/Salvionied/apollo/serialization/TransactionWitnessSet"
	"github.com/blinklabs-io/gouroboros/cbor"
)

type Transaction struct {
	TransactionBody       TransactionBody.TransactionBody
	TransactionWitnessSet TransactionWitnessSet.TransactionWitnessSet
	Valid                 bool
	AuxiliaryData         *Metadata.AuxiliaryData
}

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
	var decoded interface{}
	_, err := cbor.Decode(data, &decoded)
	if err != nil {
		return err
	}

	// Handle tagged transaction
	if arr, ok := decoded.([]interface{}); ok && len(arr) == 2 {
		if tag, ok := arr[0].(uint64); ok && tag == 258 {
			decoded = arr[1]
		}
	}

	if m, ok := decoded.(map[interface{}]interface{}); ok {
		// map case
		for k, v := range m {
			if _, ok := k.([]interface{}); ok {
				continue
			}
			var key = k
			if karr, ok := k.([]interface{}); ok && len(karr) == 2 {
				if tag, ok := karr[0].(uint64); ok && tag == 0 {
					key = karr[1]
				}
			}
			switch key {
			case 0:
				// body
				if vm, ok := v.(map[interface{}]interface{}); ok {
					cleanV := make(map[interface{}]interface{})
					for vk, vv := range vm {
						if _, ok := vk.([]interface{}); !ok {
							cleanV[vk] = vv
						}
					}
					bodyBytes, err := cbor.Encode(cleanV)
					if err != nil {
						return err
					}
					_, err = cbor.Decode(bodyBytes, &tx.TransactionBody)
					if err != nil {
						return err
					}
				} else {
					bodyBytes, err := cbor.Encode(v)
					if err != nil {
						return err
					}
					_, err = cbor.Decode(bodyBytes, &tx.TransactionBody)
					if err != nil {
						return err
					}
				}
			case 1:
				// witness set
				if vm, ok := v.(map[interface{}]interface{}); ok {
					cleanV := make(map[interface{}]interface{})
					for vk, vv := range vm {
						if _, ok := vk.([]interface{}); !ok {
							cleanV[vk] = vv
						}
					}
					witnessBytes, err := cbor.Encode(cleanV)
					if err != nil {
						return err
					}
					err = tx.TransactionWitnessSet.UnmarshalCBOR(witnessBytes)
					if err != nil {
						return err
					}
				} else {
					witnessBytes, err := cbor.Encode(v)
					if err != nil {
						return err
					}
					err = tx.TransactionWitnessSet.UnmarshalCBOR(witnessBytes)
					if err != nil {
						return err
					}
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
					if vm, ok := v.(map[interface{}]interface{}); ok {
						cleanV := make(map[interface{}]interface{})
						for vk, vv := range vm {
							if _, ok := vk.([]interface{}); !ok {
								cleanV[vk] = vv
							}
						}
						auxBytes, err := cbor.Encode(cleanV)
						if err != nil {
							return err
						}
						tx.AuxiliaryData = &Metadata.AuxiliaryData{}
						_, err = cbor.Decode(auxBytes, tx.AuxiliaryData)
						if err != nil {
							return err
						}
					} else {
						auxBytes, err := cbor.Encode(v)
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
			default:
				// ignore unknown fields
			}
		}
	} else if arr, ok := decoded.([]interface{}); ok {
		// array case
		if len(arr) < 2 {
			return errors.New("invalid transaction")
		}
		// body
		if vm, ok := arr[0].(map[interface{}]interface{}); ok {
			cleanV := make(map[interface{}]interface{})
			for vk, vv := range vm {
				if _, ok := vk.([]interface{}); !ok {
					cleanV[vk] = vv
				}
			}
			bodyBytes, err := cbor.Encode(cleanV)
			if err != nil {
				return err
			}
			_, err = cbor.Decode(bodyBytes, &tx.TransactionBody)
			if err != nil {
				return err
			}
		} else {
			bodyBytes, err := cbor.Encode(arr[0])
			if err != nil {
				return err
			}
			_, err = cbor.Decode(bodyBytes, &tx.TransactionBody)
			if err != nil {
				return err
			}
		}
		// witness set
		if vm, ok := arr[1].(map[interface{}]interface{}); ok {
			cleanV := make(map[interface{}]interface{})
			for vk, vv := range vm {
				if _, ok := vk.([]interface{}); !ok {
					cleanV[vk] = vv
				}
			}
			witnessBytes, err := cbor.Encode(cleanV)
			if err != nil {
				return err
			}
			err = tx.TransactionWitnessSet.UnmarshalCBOR(witnessBytes)
			if err != nil {
				return err
			}
		} else {
			witnessBytes, err := cbor.Encode(arr[1])
			if err != nil {
				return err
			}
			err = tx.TransactionWitnessSet.UnmarshalCBOR(witnessBytes)
			if err != nil {
				return err
			}
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
				if vm, ok := arr[3].(map[interface{}]interface{}); ok {
					cleanV := make(map[interface{}]interface{})
					for vk, vv := range vm {
						if _, ok := vk.([]interface{}); !ok {
							cleanV[vk] = vv
						}
					}
					auxBytes, err := cbor.Encode(cleanV)
					if err != nil {
						return err
					}
					tx.AuxiliaryData = &Metadata.AuxiliaryData{}
					_, err = cbor.Decode(auxBytes, tx.AuxiliaryData)
					if err != nil {
						return err
					}
				} else {
					auxBytes, err := cbor.Encode(arr[3])
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
		}
	} else {
		return errors.New("invalid transaction CBOR")
	}
	return nil
}

package Policy

import (
	"encoding/hex"
	"errors"
	"log"

	"github.com/Salvionied/cbor/v2"
)

type PolicyId struct {
	Value string
}

/**
	New creates a new PolicyId from a hexadecimal string

	Params:
		value string: The hexadecimal string representing the policy ID.
	
	Returns:
		*PolicyId: A pointer to the PolicyId.
		error: An error if the input string is not of the expected length.
*/
func New(value string) (*PolicyId, error) {
	if len(value) != 56 {
		return nil, errors.New("invalid length of a policy id")
	}
	return &PolicyId{
		Value: value,
	}, nil
}

/**
	FromBytes creates a new PolicyId from a byte slice.

	Params:
		value []byte: The byte slice representing the policy ID.
	
	Returns:
		*PolicyId: A pointer to the PolicyId.
		error: An error if the input byte slice is not of the expected length.
*/
func FromBytes(value []byte) (*PolicyId, error) {
	if len(value) != 28 {
		return nil, errors.New("invalid length of a policy id")
	}
	return &PolicyId{
		Value: hex.EncodeToString(value),
	}, nil
}

/**
	String returns the hexadecimal string representation
	of the PolicyId.
*/
func (policyId PolicyId) String() string {
	return policyId.Value
}

/**
	MarshalCBOR serializes the PolicyId to CBOR format.

	Returns:
		[]byte: The CBOR serialized data.
		error: An error if the serialization fails.
*/
func (policyId *PolicyId) MarshalCBOR() ([]byte, error) {
	res, err := hex.DecodeString(policyId.Value)
	if err != nil {
		return nil, err
	}
	if len(res) != 28 {
		return nil, errors.New("invalid length of a policy id")
	}

	if len(res) == 0 {
		return cbor.Marshal(make([]byte, 0))
	}
	return cbor.Marshal(res)

}

/**
	UnmarshalCBOR deserializes the PolicyId from CBOR format.

	Params:
		value []byte: The CBOR serialized data.

	Returns:
		error: An error if the deserialization fails.
*/
func (policyId *PolicyId) UnmarshalCBOR(value []byte) error {
	var res any
	err := cbor.Unmarshal(value, &res)
	if err != nil {
		log.Fatal(err)
		return err
	}
	switch res := res.(type) {
	case []byte:
		hexString := hex.EncodeToString(res)
		if len(hexString) != 56 {
			return errors.New("invalid length of a policy id")
		}
		policyId.Value = hexString
	case string:
		hexString := res
		if len(hexString) != 56 {
			return errors.New("invalid length of a policy id")
		}
		policyId.Value = hexString
	default:
		log.Fatal("Unknown type")
	}
	return nil
}

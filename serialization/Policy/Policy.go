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

func New(value string) (*PolicyId, error) {
	if len(value) != 56 {
		return nil, errors.New("invalid length of a policy id")
	}
	return &PolicyId{
		Value: value,
	}, nil
}

func FromBytes(value []byte) (*PolicyId, error) {
	if len(value) != 28 {
		return nil, errors.New("invalid length of a policy id")
	}
	return &PolicyId{
		Value: hex.EncodeToString(value),
	}, nil
}

func (policyId PolicyId) String() string {
	return policyId.Value
}

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
func (policyId *PolicyId) UnmarshalCBOR(value []byte) error {
	var res any
	err := cbor.Unmarshal(value, &res)
	if err != nil {
		log.Fatal(err, "HERE")
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

package VerificationKeyWitness

import (
	"errors"

	"github.com/Salvionied/apollo/serialization/Key"
	"github.com/blinklabs-io/gouroboros/cbor"
)

type VerificationKeyWitness struct {
	Vkey      Key.VerificationKey
	Signature []uint8
}

func (vkw *VerificationKeyWitness) UnmarshalCBOR(data []byte) error {
	var temp interface{}
	_, err := cbor.Decode(data, &temp)
	if err != nil {
		return err
	}
	if m, ok := temp.(map[interface{}]interface{}); ok {
		// map case
		cleanM := make(map[interface{}]interface{})
		for k, v := range m {
			if _, ok := k.([]interface{}); !ok {
				cleanM[k] = v
			}
		}
		data, err := cbor.Encode(cleanM)
		if err != nil {
			return err
		}
		_, err = cbor.Decode(data, vkw)
		if err != nil {
			return err
		}
	} else if arr, ok := temp.([]interface{}); ok {
		// array case
		if len(arr) != 2 {
			return errors.New("expected array of 2 elements")
		}
		if vkeyBytes, ok := arr[0].([]byte); ok {
			vkw.Vkey = Key.VerificationKey{Payload: vkeyBytes}
		}
		if sig, ok := arr[1].([]byte); ok {
			vkw.Signature = sig
		} else {
			// Handle case where sig is 0 or other
			vkw.Signature = []byte{}
		}
	} else {
		return errors.New("invalid vkw CBOR")
	}
	return nil
}

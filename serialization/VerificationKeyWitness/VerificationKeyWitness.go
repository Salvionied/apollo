package VerificationKeyWitness

import "github.com/salvionied/apollo/serialization/Key"

type VerificationKeyWitness struct {
	_         struct{} `cbor:",toarray"`
	Vkey      Key.VerificationKey
	Signature []uint8
}

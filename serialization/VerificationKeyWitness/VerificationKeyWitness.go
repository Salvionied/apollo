package VerificationKeyWitness

import "Salvionied/apollo/serialization/Key"

type VerificationKeyWitness struct {
	_         struct{} `cbor:",toarray"`
	Vkey      Key.VerificationKey
	Signature []uint8
}

package VerificationKeyWitness

import "github.com/SundaeSwap-finance/apollo/serialization/Key"

type VerificationKeyWitness struct {
	_         struct{} `cbor:",toarray"`
	Vkey      Key.VerificationKey
	Signature []uint8
}

package Certificate

import "github.com/Salvionied/apollo/serialization"

type StakeCredential struct {
	_          struct{} `cbor:"toarray"`
	_CODE      int      `cbor:",omitempty"`
	Credential serialization.ConstrainedBytes
}

// TODO
type Certificate struct {
	_               struct{} `cbor:"toarray"`
	_CODE           int
	StakeCredential *StakeCredential
}

type Certificates []*Certificate

func (sc *StakeCredential) Kind() int {
	return sc._CODE
}

func (c *Certificate) Kind() int {
	return c._CODE
}

func (sc *StakeCredential) KeyHash() serialization.PubKeyHash {
	res := serialization.PubKeyHash(sc.Credential.Payload)
	return res
}

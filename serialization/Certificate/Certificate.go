package Certificate

import (
	"github.com/Salvionied/apollo/serialization"
)

type StakeCredential struct {
	_          struct{} `cbor:",toarray"`
	Code       int      `cbor:",omitempty"`
	Credential serialization.ConstrainedBytes
}

// TODO
type Certificate struct {
	_               struct{} `cbor:",toarray"`
	Code            int
	StakeCredential *StakeCredential
}

func NewCertificateFromAddress(
	addr []byte,
	code int,
	credentialType int,
) Certificate {
	return Certificate{
		Code: code,
		StakeCredential: &StakeCredential{
			Code:       credentialType,
			Credential: serialization.ConstrainedBytes{Payload: addr},
		},
	}
}

func NewCertificates(certs []*Certificate) Certificates {
	return certs
}

type Certificates []*Certificate

func (sc *StakeCredential) Kind() int {
	return sc.Code
}

func (c *Certificate) Kind() int {
	return c.Code
}

func (sc *StakeCredential) KeyHash() serialization.PubKeyHash {
	res := serialization.PubKeyHash(sc.Credential.Payload)
	return res
}

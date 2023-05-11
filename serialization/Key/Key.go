package Key

import (
	"crypto/ed25519"
	"encoding/hex"
	"log"

	"github.com/salvionied/apollo/serialization"

	"github.com/Salvionied/cbor/v2"
	"golang.org/x/crypto/blake2b"
)

type SigningKey struct {
	Payload []byte
}

func Sign(message []byte, sk []byte) []byte {
	res := ed25519.Sign(sk, message)
	return res
}

func (sk SigningKey) Sign(data []byte) []byte {
	pk := sk.Payload
	signature := Sign(data, pk)
	return signature
}

type VerificationKey struct {
	Payload []byte
}

func (vk *VerificationKey) UnmarshalCBOR(data []byte) error {
	final_data := make([]byte, 0)
	err := cbor.Unmarshal(data, &final_data)
	if err != nil {
		log.Fatal(err)
	}
	vk.Payload = final_data
	return nil
}

func (vk *VerificationKey) MarshalCBOR() ([]byte, error) {
	return cbor.Marshal(vk.Payload)
}
func VerificationKeyFromCbor(cbor_string string) (*VerificationKey, error) {
	vkey := new(VerificationKey)
	value, err := hex.DecodeString(cbor_string)
	if err != nil {
		return nil, err
	}
	err = cbor.Unmarshal(value, vkey)
	if err != nil {
		return nil, err
	}
	return vkey, nil
}

func (vk VerificationKey) Hash() (serialization.PubKeyHash, error) {
	KeyHash, err := Blake224Hash(vk.Payload, 28)
	if err != nil {
		return serialization.PubKeyHash{}, err
	}
	r := serialization.PubKeyHash{}
	copy(r[:], KeyHash)
	return r, nil
}

type PaymentKeyPair struct {
	VerificationKey VerificationKey
	SigningKey      SigningKey
}

func PaymentKeyPairGenerate() PaymentKeyPair {
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		log.Fatal(err)
	}
	return PaymentKeyPair{VerificationKey{publicKey}, SigningKey{privateKey}}
}

type PaymentSigningKey SigningKey
type PaymentVerificationKey VerificationKey
type StakeSigningKey SigningKey
type StakeVerificationKey VerificationKey

func Blake224Hash(b []byte, len int) ([]byte, error) {
	hash, err := blake2b.New(len, nil)
	if err != nil {
		return nil, err
	}
	_, err = hash.Write(b)
	if err != nil {
		return nil, err
	}
	return hash.Sum(nil), err
}

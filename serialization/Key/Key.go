package Key

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"

	"github.com/Salvionied/apollo/crypto/bip32"
	"github.com/Salvionied/apollo/serialization"

	"github.com/Salvionied/cbor/v2"
	"golang.org/x/crypto/blake2b"
)

type SigningKey struct {
	Payload []byte
}

/*
*

	Sign function signs a message using the provided key and returns
	the signature.

	Params:
		message ([]byte): The message to sign.
		sk ([]byte): The signing key, which can be either an extended or an ed25519 private key.

	Returns:
		[]byte: The signature of the message.
		error: An error if the signing fails.
*/
func Sign(message []byte, sk []byte) ([]byte, error) {
	if len(sk) != ed25519.PrivateKeySize {
		sk, err := bip32.NewXPrv(sk)
		if err != nil {
			return nil, fmt.Errorf("error creating signing key from bytes, %s", err)
		}
		signature := sk.Sign(message)
		return signature, nil

	}
	res := ed25519.Sign(sk, message)
	return res, nil
}

/*
*

	Sign function signs a data byte slice using
	the signing key and the returns the signature.

	Params:
		data ([]byte): The data to sign.

	Returns:
		[]byte: The signature of the data.
		error: An error if the signing fails.
*/
func (sk SigningKey) Sign(data []byte) ([]byte, error) {
	pk := sk.Payload
	signature, err := Sign(data, pk)
	if err != nil {
		return nil, err
	}
	return signature, nil
}

type VerificationKey struct {
	Payload []byte
}

/*
*

	UnmarshalCBOR function unmarshals data into a VerificationKey instance.

	Params:
		data ([]byte): The CBOR data to unmarshal.

	Returns:
		error: An error if unmarshaling fails, nil otherwise.
*/
func (vk *VerificationKey) UnmarshalCBOR(data []byte) error {
	final_data := make([]byte, 0)
	err := cbor.Unmarshal(data, &final_data)
	if err != nil {
		return err
	}
	vk.Payload = final_data
	return nil
}

/*
*

	MarshalCBOR marshals the VerificationKey instance into CBOR data.

	Returns:
		([]byte, error): The CBOR-encoded data and error if marshaling fails, nil otherwise.
*/
func (vk *VerificationKey) MarshalCBOR() ([]byte, error) {
	return cbor.Marshal(vk.Payload)
}

/*
*

	VerificationKeyFromCbor creates a VerificationKey
	instance from a CBOR-encoded string.

	Params:
		cbor_string (string): The CBOR-encoded string.

	Returns:
		(*VerificationKey, error): A VerificationKey instance and an error
								   if decoding or unmarshaling fails, nil otherwise.
*/
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

/*
*

	Hash computes the has of the VerificationKey and returns it as
	public key.

	Returns:
		(serialization.PubKeyHash, error): The computed hash and an error
											if computation fails, nil otherwise.
*/
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

/*
*

	PaymentKeyPairGenerate generates a PaymentKey pair with a randomly
	generated key pair.

	Returns:
		PaymentKeyPair: A newly generated PaymentKeyPair.
		error: An error if the payment fails.
*/
func PaymentKeyPairGenerate() (*PaymentKeyPair, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, err
	}
	return &PaymentKeyPair{VerificationKey{publicKey}, SigningKey{privateKey}}, nil
}

type PaymentSigningKey SigningKey
type PaymentVerificationKey VerificationKey

/*
*

	Blake224Hash computes the Blake2b-224 hash of the provided byte slice.

	Params:
		b ([]byte): The input byte slice to hash.
		len (int): The length of the hash.

	Returns:
		([]byte, error): The computed hash and an error if computation fails,
						 nil otherwise.
*/
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

func SigningKeyFromHexString(hexString string) (*SigningKey, error) {
	skey := new(SigningKey)
	value, err := hex.DecodeString(hexString)
	if err != nil {
		return nil, err
	}
	skey.Payload = value
	return skey, nil
}

func VerificationKeyFromHexString(hexString string) (*VerificationKey, error) {
	vkey := new(VerificationKey)
	value, err := hex.DecodeString(hexString)
	if err != nil {
		return nil, err
	}
	vkey.Payload = value
	return vkey, nil
}

func (sk SigningKey) ToHexString() string {
	return hex.EncodeToString(sk.Payload)
}

func (vk VerificationKey) ToHexString() string {
	return hex.EncodeToString(vk.Payload)
}

func (sk *SigningKey) MarshalCBOR() ([]byte, error) {
	return cbor.Marshal(sk.Payload)
}

func (sk *SigningKey) UnmarshalCBOR(data []byte) error {
	final_data := make([]byte, 0)
	err := cbor.Unmarshal(data, &final_data)
	if err != nil {
		return err
	}
	sk.Payload = final_data
	return nil
}

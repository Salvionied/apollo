package Key_test

import (
	"crypto/ed25519"
	"encoding/hex"
	"testing"

	"github.com/Salvionied/apollo/v2/serialization/Key"
	"github.com/blinklabs-io/gouroboros/cbor"
)

var VkeyHex = "694c01268746fccf4a8b94213649a7041b7e20aa4e83b0df2397cadf7c85c5ac"

var SkeyHex = "48fe4baedf13260eb3e2138542bc843e4d272942d405edd3ce4e8eae3ef9eafe694c01268746fccf4a8b94213649a7041b7e20aa4e83b0df2397cadf7c85c5ac"
var Skey, _ = Key.SigningKeyFromHexString(SkeyHex)
var Vkey, _ = Key.VerificationKeyFromHexString(VkeyHex)

var VkeyCBOR = "5820694c01268746fccf4a8b94213649a7041b7e20aa4e83b0df2397cadf7c85c5ac"

var SkeyCBOR = "584048fe4baedf13260eb3e2138542bc843e4d272942d405edd3ce4e8eae3ef9eafe694c01268746fccf4a8b94213649a7041b7e20aa4e83b0df2397cadf7c85c5ac"

func TestGenerateKeyPair(t *testing.T) {
	pp, err := Key.PaymentKeyPairGenerate()
	if err != nil {
		t.Fatal(err)
	}
	if len(pp.VerificationKey.Payload) != 32 {
		t.Errorf("PaymentKeyPairGenerate() failed")
	}
	if len(pp.SigningKey.Payload) != 64 {
		t.Errorf("PaymentKeyPairGenerate() failed")
	}
	_, err = cbor.Encode(pp.VerificationKey.Payload)
	if err != nil {
		t.Errorf("PaymentKeyPairGenerate() failed")
	}
	_, err = cbor.Encode(pp.SigningKey.Payload)
	if err != nil {
		t.Errorf("PaymentKeyPairGenerate() failed")
	}
}

func TestSign(t *testing.T) {
	data := []byte("test")
	sig, err := Skey.Sign(data)
	if err != nil {
		t.Errorf("Sign() failed")
	}
	if len(sig) != 64 {
		t.Errorf("Sign() failed")
	}
	isValid := ed25519.Verify(Vkey.Payload, data, sig)
	if !isValid {
		t.Errorf("Sign() failed")
	}
}

func TestFromHexToHex(t *testing.T) {
	sk, err := Key.SigningKeyFromHexString(SkeyHex)
	if err != nil {
		t.Fatal(err)
	}
	vk, err := Key.VerificationKeyFromHexString(VkeyHex)
	if err != nil {
		t.Fatal(err)
	}
	if sk.ToHexString() != SkeyHex {
		t.Errorf("SigningKeyFromHexString() failed")
	}
	if vk.ToHexString() != VkeyHex {
		t.Errorf("VerificationKeyFromHexString() failed")
	}
}

func TestCborMarshalRound(t *testing.T) {
	sk, err := Key.SigningKeyFromHexString(SkeyHex)
	if err != nil {
		t.Errorf("SigningKeyFromHexString() failed")
	}
	if sk == nil {
		t.Fatal("SigningKeyFromHexString returned nil")
	}
	vk, err := Key.VerificationKeyFromHexString(VkeyHex)
	if err != nil {
		t.Errorf("VerificationKeyFromHexString() failed")
	}
	if vk == nil {
		t.Fatal("VerificationKeyFromHexString returned nil")
	}
	sk_cbor, err := sk.MarshalCBOR()
	if err != nil {
		t.Errorf("SigningKeyFromHexString() failed")
	}
	vk_cbor, err := vk.MarshalCBOR()
	if err != nil {
		t.Errorf("VerificationKeyFromHexString() failed")
	}
	if hex.EncodeToString(sk_cbor) != SkeyCBOR {
		t.Errorf("SigningKeyFromHexString() failed")
	}
	if hex.EncodeToString(vk_cbor) != VkeyCBOR {
		t.Errorf("VerificationKeyFromHexString() failed")
	}
	sk2 := new(Key.SigningKey)
	vk2 := new(Key.VerificationKey)
	err = sk2.UnmarshalCBOR(sk_cbor)
	if err != nil {
		t.Errorf("SigningKeyFromHexString() failed")
	}
	err = vk2.UnmarshalCBOR(vk_cbor)
	if err != nil {
		t.Errorf("VerificationKeyFromHexString() failed")
	}
	if sk2.ToHexString() != SkeyHex {
		t.Errorf("SigningKeyFromHexString() failed")
	}
	if vk2.ToHexString() != VkeyHex {
		t.Errorf("VerificationKeyFromHexString() failed")
	}
}

func TestVerificationKeyFromCbor(t *testing.T) {
	vk, err := Key.VerificationKeyFromHexString(VkeyHex)
	if err != nil {
		t.Fatal(err)
	}
	vk2, err := Key.VerificationKeyFromCbor(VkeyCBOR)
	if err != nil {
		t.Fatal(err)
	}
	if vk.ToHexString() != vk2.ToHexString() {
		t.Errorf("VerificationKeyFromHexString() failed")
	}

}

func TestHash(t *testing.T) {
	data := []byte("test")
	hash, err := Key.Blake224Hash(data, 28)
	if err != nil {
		t.Errorf("Blake224Hash() failed")
	}
	if len(hash) != 28 {
		t.Errorf("Blake224Hash() failed")
	}
	if hex.EncodeToString(
		hash[:],
	) != "0d06154ce8bf87d8823dc69fb1e9a9459755d9092e87108bd11fc8cc" {
		t.Errorf("Blake224Hash() failed, got %s", hex.EncodeToString(hash[:]))
	}
}

func TestVerificationKeyHash(t *testing.T) {
	hash, err := Vkey.Hash()
	if err != nil {
		t.Errorf("Hash() failed")
	}
	if len(hash) != 28 {
		t.Errorf("Hash() failed")
	}
	if hex.EncodeToString(
		hash[:],
	) != "b9df52987c59eec967744e77840acf844d5619daea5890f6a6539079" {
		t.Errorf("Hash() failed, got %s", hex.EncodeToString(hash[:]))
	}
}

func TestInvalidCborDataRaisesError(t *testing.T) {
	invalidCbor := "1fff"
	_, err := Key.VerificationKeyFromCbor(invalidCbor)
	if err == nil {
		t.Errorf("VerificationKeyFromCbor() failed")
	}
	skey := new(Key.SigningKey)
	err = skey.UnmarshalCBOR([]byte(invalidCbor))
	if err == nil {
		t.Errorf("UnmarshalCBOR() failed")
	}
	vkey := new(Key.VerificationKey)
	err = vkey.UnmarshalCBOR([]byte(invalidCbor))
	if err == nil {
		t.Errorf("UnmarshalCBOR() failed")
	}
}

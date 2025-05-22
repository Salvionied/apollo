package bip32

import (
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/binary"
	"encoding/hex"
	"slices"
)

// XPub is exntend public key for ed25519
type XPub struct {
	xpub []byte
}

// NewXPub create XPub by plain xpub bytes
func NewXPub(raw []byte) XPub {
	if len(raw) != XPubSize {
		panic("bip32-ed25519: NewXPub: size should be 64 bytes")
	}
	return XPub{xpub: slices.Clone(raw)}
}

// String implements Stringer interface and returns plain hex string
func (x XPub) String() string {
	return hex.EncodeToString(x.xpub)
}

// Bytes returns intenal bytes
func (x XPub) Bytes() []byte {
	return slices.Clone(x.xpub)
}

// PublicKey returns the current public key
func (x XPub) PublicKey() []byte {
	return slices.Clone(x.xpub[:32])
}

// ChainCode returns chain code bytes
func (x XPub) ChainCode() []byte {
	return slices.Clone(x.xpub[32:])
}

// Derive derives new XPub by a soft index
func (x XPub) Derive(index uint32) XPub {
	if index > HardIndex {
		panic("bip32-ed25519: xpub.Derive: expected a soft derivation")
	}

	var pubkey [32]byte
	copy(pubkey[:], x.xpub[:32])
	chaincode := slices.Clone(x.xpub[32:])

	zmac := hmac.New(sha512.New, chaincode)
	imac := hmac.New(sha512.New, chaincode)

	seri := make([]byte, 4)
	binary.LittleEndian.PutUint32(seri, index)

	_, _ = zmac.Write([]byte{2})
	_, _ = zmac.Write(pubkey[:])
	_, _ = zmac.Write(seri)

	_, _ = imac.Write([]byte{3})
	_, _ = imac.Write(pubkey[:])
	_, _ = imac.Write(seri)

	left, ok := pointPlus(&pubkey, pointOfTrunc28Mul8(zmac.Sum(nil)[:32]))
	if !ok {
		panic("bip32-ed25519: can't convert bytes to edwards25519.ExtendedGroupElement")
	}

	var out [64]byte
	copy(out[:32], left[:32])
	copy(out[32:], imac.Sum(nil)[32:])
	return XPub{xpub: out[:]}
}

// Verify verifies signature by message
func (x XPub) Verify(msg, sig []byte) bool {
	pk := x.xpub[:32]
	return ed25519.Verify(pk, msg, sig)
}

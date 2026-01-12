package bip32

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"slices"

	"github.com/Salvionied/apollo/v2/crypto/edwards25519"
)

const (
	XPrvSize  = 96
	XPubSize  = 64
	HardIndex = 0x80000000
)

// XPrv is exntend private key for ed25519
type XPrv struct {
	xprv []byte
}

// NewXPrv creates XPrv by plain xprv bytes
func NewXPrv(raw []byte) (XPrv, error) {
	if len(raw) != XPrvSize {
		return XPrv{}, errors.New(
			"bip32-ed25519: NewXPrv: size should be 96 bytes",
		)
	}

	if (raw[0] & 0b0000_0111) != 0b0000_0000 {
		return XPrv{}, errors.New(
			"bip32-ed25519: NewXPrv: the lowest 3 bits of the first byte of seed should be cleared",
		)
	}

	if (raw[31] & 0b1100_0000) != 0b0100_0000 {
		return XPrv{}, errors.New(
			"bip32-ed25519: NewXPrv: the highest bit of the last byte of seed should be cleared",
		)
	}

	return XPrv{xprv: slices.Clone(raw)}, nil
}

// NewRootXPrv creates XPrv by seed(bip39),the seed size should be 32 bytes at least
func NewRootXPrv(seed []byte) XPrv {
	// Let ˜k(seed) be 256-bit master secret
	// Then derive k = H512(˜k)and denote its left 32-byte by kL and right one by kR.
	secretKey := sha512.Sum512(seed[:32])

	// Modify kL:
	// the lowest 3 bits of the first byte are cleared
	secretKey[0] &= 0b1111_1000
	// the highest bit of the last byte is cleared
	// and third highest bit also should clear according bip32-ed25519 spec
	secretKey[31] &= 0b0101_1111
	// and the second highest bit of the last byte is set
	secretKey[31] |= 0b0100_0000

	xprv := make([]byte, XPrvSize)
	copy(xprv[:64], secretKey[:])

	// Derive c ← H256(0x01||˜k), where H256 is SHA-256, and call it the root chain code
	chaincode := sha256.Sum256(append([]byte{1}, seed...))
	copy(xprv[64:], chaincode[:])
	return XPrv{xprv}
}

// String implements Stringer interface and returns plain hex string
func (x XPrv) String() string {
	return hex.EncodeToString(x.xprv)
}

// Bytes returns intenal bytes
func (x XPrv) Bytes() []byte {
	return slices.Clone(x.xprv)
}

// ChainCode returns chain code bytes
func (x XPrv) ChainCode() []byte {
	return slices.Clone(x.xprv[64:])
}

// Derive derives new XPrv by an index
func (x XPrv) Derive(index uint32) XPrv {
	/*
		cP is the chain code.
		kP is (klP,krP) extended private key
		aP is the public key.
		ser32(i) serializes a uint32 i as a 4-byte little endian bytes

		If hardened child:
			let Z = HMAC-SHA512(Key = cP, Data = 0x00 || kP || ser32(i)).
			let I = HMAC-SHA512(Key = cP, Data = 0x01 || kP || ser32(i)).

		If normal child:
			let Z = HMAC-SHA512(Key = cP, Data = 0x02 || aP || ser32(i)).
			let I = HMAC-SHA512(Key = cP, Data = 0x03 || aP || ser32(i)).

		chain code
		The I is truncated to right 32 bytes.
	*/

	ekey := slices.Clone(x.xprv[:64])
	chaincode := slices.Clone(x.xprv[64:96])

	kl := slices.Clone(x.xprv[:32])
	kr := slices.Clone(x.xprv[32:64])

	zmac := hmac.New(sha512.New, chaincode)
	imac := hmac.New(sha512.New, chaincode)

	seri := make([]byte, 4)
	binary.LittleEndian.PutUint32(seri, index)

	if index >= HardIndex {
		_, _ = zmac.Write([]byte{0})
		_, _ = zmac.Write(ekey)
		_, _ = zmac.Write(seri)

		_, _ = imac.Write([]byte{1})
		_, _ = imac.Write(ekey)
		_, _ = imac.Write(seri)
	} else {
		pubkey := x.PublicKey()
		_, _ = zmac.Write([]byte{2})
		_, _ = zmac.Write(pubkey[:])
		_, _ = zmac.Write(seri)

		_, _ = imac.Write([]byte{3})
		_, _ = imac.Write(pubkey[:])
		_, _ = imac.Write(seri)
	}

	zout, iout := zmac.Sum(nil), imac.Sum(nil)
	zl, zr := zout[0:32], zout[32:64]

	result := make([]byte, 96)
	copy(result[0:32], add28Mul8(kl, zl)[:])   // kl
	copy(result[32:64], add256Bits(kr, zr)[:]) // kr
	copy(result[64:96], iout[32:])             // chain code

	return XPrv{result}
}

// DeriveHard derives new XPrv by a hardened index
func (x XPrv) DeriveHard(index uint32) XPrv {
	if index > HardIndex {
		panic("bip32-ed25519: xprv.DeriveHard: overflow")
	}
	return x.Derive(HardIndex + index)
}

// PublicKey returns the public key
func (x XPrv) PublicKey() []byte {
	var A edwards25519.ExtendedGroupElement

	var hBytes [32]byte
	copy(hBytes[:], x.xprv[:32]) // make sure prvkey is 32 bytes

	edwards25519.GeScalarMultBase(&A, &hBytes)
	var publicKeyBytes [32]byte
	A.ToBytes(&publicKeyBytes)

	return publicKeyBytes[:]
}

// Sign signs message
func (x XPrv) Sign(message []byte) []byte {
	var hsout [64]byte

	hasher := sha512.New()
	_, _ = hasher.Write(x.xprv[32:64])
	_, _ = hasher.Write(message)
	hasher.Sum(hsout[:0])

	var nonce [32]byte
	edwards25519.ScReduce(&nonce, &hsout)

	var r [32]byte
	var R edwards25519.ExtendedGroupElement
	edwards25519.GeScalarMultBase(&R, &nonce)
	R.ToBytes(&r)

	var sig [edwards25519.SignatureSize]byte
	copy(sig[:32], r[:])
	copy(sig[32:], x.PublicKey()[:])

	hasher.Reset()
	_, _ = hasher.Write(sig[:])
	_, _ = hasher.Write(message)
	hasher.Sum(hsout[:0])

	var a [32]byte
	edwards25519.ScReduce(&a, &hsout)

	var s, b [32]byte
	copy(b[:], x.xprv[:32])
	edwards25519.ScMulAdd(&s, &a, &b, &nonce)
	copy(sig[32:], s[:])

	return sig[:]
}

// Verify verifies signature by message
func (x XPrv) Verify(msg, sig []byte) bool {
	return edwards25519.Verify(x.PublicKey(), msg, sig)
}

// XPub returns extends public key for current XPrv
func (x XPrv) XPub() XPub {
	var xpub [64]byte
	copy(xpub[:32], x.PublicKey())
	copy(xpub[32:], x.xprv[64:96])
	return XPub{xpub: xpub[:]}
}

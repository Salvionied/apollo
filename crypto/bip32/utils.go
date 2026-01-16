package bip32

import "github.com/Salvionied/apollo/v2/crypto/edwards25519"

func add28Mul8(kl, zl []byte) *[32]byte {
	if kl == nil || zl == nil {
		return nil
	}
	var carry uint16 = 0
	var out [32]byte

	for i := range 28 {
		r := uint16(kl[i]) + uint16(zl[i])<<3 + carry
		out[i] = byte(r & 0xff)
		carry = r >> 8
	}

	for i := 28; i < 32; i++ {
		r := uint16(kl[i]) + carry
		out[i] = byte(r & 0xff)
		carry = r >> 8
	}

	return &out
}

func add256Bits(kr, zr []byte) *[32]byte {
	if kr == nil || zr == nil {
		return nil
	}
	var carry uint16 = 0
	var out [32]byte

	for i := range 32 {
		r := uint16(kr[i]) + uint16(zr[i]) + carry
		out[i] = byte(r)
		carry = r >> 8
	}

	return &out
}

func pointOfTrunc28Mul8(zl []byte) *[32]byte {
	copy := add28Mul8(make([]byte, 32), zl)
	var Ap edwards25519.ExtendedGroupElement
	edwards25519.GeScalarMultBase(&Ap, copy)

	var zl8b [32]byte
	Ap.ToBytes(&zl8b)
	return &zl8b
}

func pointPlus(pk, zl8 *[32]byte) (*[32]byte, bool) {
	var a edwards25519.ExtendedGroupElement
	if !a.FromBytes(pk) {
		return nil, false
	}

	var b edwards25519.ExtendedGroupElement
	if !b.FromBytes(zl8) {
		return nil, false
	}

	var c edwards25519.CachedGroupElement
	b.ToCached(&c)

	var r edwards25519.CompletedGroupElement
	edwards25519.GeAdd(&r, &a, &c)

	var p2 edwards25519.ProjectiveGroupElement
	r.ToProjective(&p2)

	var res [32]byte
	p2.ToBytes(&res)

	return &res, true
}

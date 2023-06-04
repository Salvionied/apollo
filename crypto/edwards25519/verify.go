// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package edwards25519

import (
	"bytes"
	"crypto/sha512"
)

const SignatureSize = 64

func Verify(pub, msg, sig []byte) bool {
	if len(sig) != SignatureSize || sig[63]&224 != 0 {
		return false
	}

	var A ExtendedGroupElement
	var publicKeyBytes [32]byte
	copy(publicKeyBytes[:], pub)
	if !A.FromBytes(&publicKeyBytes) {
		return false
	}
	FeNeg(&A.X, &A.X)
	FeNeg(&A.T, &A.T)

	h := sha512.New()
	_, _ = h.Write(sig[:32])
	_, _ = h.Write(pub[:])
	_, _ = h.Write(msg)
	var digest [64]byte
	h.Sum(digest[:0])

	var hReduced [32]byte
	ScReduce(&hReduced, &digest)

	var R ProjectiveGroupElement
	var s [32]byte
	copy(s[:], sig[32:])

	// https://tools.ietf.org/html/rfc8032#section-5.1.7 requires that s be in
	// the range [0, order) in order to prevent signature malleability.
	if !ScMinimal(&s) {
		return false
	}

	GeDoubleScalarMultVartime(&R, &hReduced, &A, &s)

	var checkR [32]byte
	R.ToBytes(&checkR)
	return bytes.Equal(sig[:32], checkR[:])
}

// Package backendutil holds parsing and validation helpers shared by the
// backend implementations in this module. These helpers exist to normalize
// provider API responses, so they are internal rather than part of Apollo's
// public API surface.
package backendutil

import (
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/common"
)

// ParseRational parses a provider-supplied rational number without passing it
// through floating point. Decimal and exponent forms accepted by JSON numbers
// are supported by math/big.
func ParseRational(s string) (*big.Rat, error) {
	value, ok := new(big.Rat).SetString(s)
	if !ok {
		return nil, fmt.Errorf("invalid rational number %q", s)
	}
	return value, nil
}

// BoundedInt converts an API-supplied int64 to int, rejecting negative values
// and values that would not fit in 32 bits.
func BoundedInt(v int64, name string) (int, error) {
	if v < 0 || v > math.MaxInt32 {
		return 0, fmt.Errorf("%s out of range: %d", name, v)
	}
	return int(v), nil
}

// BoundedIntFromUint64 converts an API-supplied uint64 to int, rejecting
// values that would not fit in 32 bits.
func BoundedIntFromUint64(v uint64, name string) (int, error) {
	if v > math.MaxInt32 {
		return 0, fmt.Errorf("%s out of range: %d", name, v)
	}
	return int(v), nil
}

// ParseAssetUnit splits an API asset unit into policy ID and asset name.
func ParseAssetUnit(unit string) (common.Blake2b224, cbor.ByteString, error) {
	if len(unit) < common.Blake2b224Size*2 {
		return common.Blake2b224{}, cbor.ByteString{}, fmt.Errorf("asset unit is too short: %q", unit)
	}
	policyHex := unit[:common.Blake2b224Size*2]
	nameHex := unit[common.Blake2b224Size*2:]

	policyBytes, err := hex.DecodeString(policyHex)
	if err != nil {
		return common.Blake2b224{}, cbor.ByteString{}, fmt.Errorf("invalid policy ID hex %q: %w", policyHex, err)
	}
	if len(policyBytes) != common.Blake2b224Size {
		return common.Blake2b224{}, cbor.ByteString{}, fmt.Errorf("invalid policy ID length: expected %d bytes, got %d", common.Blake2b224Size, len(policyBytes))
	}
	var policyId common.Blake2b224
	copy(policyId[:], policyBytes)

	nameBytes, err := hex.DecodeString(nameHex)
	if err != nil {
		return common.Blake2b224{}, cbor.ByteString{}, fmt.Errorf("invalid asset name hex %q: %w", nameHex, err)
	}
	if len(nameBytes) > 32 {
		return common.Blake2b224{}, cbor.ByteString{}, fmt.Errorf("invalid asset name length: expected at most 32 bytes, got %d", len(nameBytes))
	}
	return policyId, cbor.NewByteString(nameBytes), nil
}

// ParseRedeemerTag parses a redeemer purpose string to a RedeemerTag.
func ParseRedeemerTag(s string) (common.RedeemerTag, error) {
	switch strings.ToLower(s) {
	case "spend":
		return common.RedeemerTagSpend, nil
	case "mint":
		return common.RedeemerTagMint, nil
	case "cert", "publish":
		return common.RedeemerTagCert, nil
	case "reward", "withdraw":
		return common.RedeemerTagReward, nil
	default:
		return 0, fmt.Errorf("unsupported redeemer tag %q", s)
	}
}

// ParseFraction parses a fraction string (e.g. "1/2") to a float64.
func ParseFraction(s string) (float64, error) {
	parts := strings.Split(s, "/")
	if len(parts) == 2 {
		num, err := strconv.ParseFloat(parts[0], 64)
		if err != nil {
			return 0, fmt.Errorf("invalid numerator %q: %w", parts[0], err)
		}
		if math.IsNaN(num) || math.IsInf(num, 0) {
			return 0, fmt.Errorf("invalid numerator (NaN/Inf) in fraction %q", s)
		}
		den, err := strconv.ParseFloat(parts[1], 64)
		if err != nil {
			return 0, fmt.Errorf("invalid denominator %q: %w", parts[1], err)
		}
		if den == 0 || math.IsNaN(den) || math.IsInf(den, 0) {
			return 0, fmt.Errorf("invalid denominator in fraction %q", s)
		}
		return num / den, nil
	}
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid number %q: %w", s, err)
	}
	if math.IsNaN(val) || math.IsInf(val, 0) {
		return 0, fmt.Errorf("invalid number (NaN/Inf) %q", s)
	}
	return val, nil
}

// ScriptRefFromBytes builds a common.ScriptRef of the given script ref type
// from raw script bytes. Native scripts are decoded from their CBOR
// representation. When expectedHashHex is non-empty, the script hash is
// recomputed for the claimed language and compared against it, failing closed
// if a provider returns script bytes (or a language) that do not match the
// hash it claims for them. An empty expectedHashHex skips verification for
// providers that do not supply a script hash.
func ScriptRefFromBytes(scriptType uint, scriptBytes []byte, expectedHashHex string) (*common.ScriptRef, error) {
	var script common.Script
	switch scriptType {
	case common.ScriptRefTypeNativeScript:
		var native common.NativeScript
		if _, err := cbor.Decode(scriptBytes, &native); err != nil {
			return nil, fmt.Errorf("failed to decode native script: %w", err)
		}
		script = native
	case common.ScriptRefTypePlutusV1:
		script = common.PlutusV1Script(scriptBytes)
	case common.ScriptRefTypePlutusV2:
		script = common.PlutusV2Script(scriptBytes)
	case common.ScriptRefTypePlutusV3:
		script = common.PlutusV3Script(scriptBytes)
	case common.ScriptRefTypePlutusV4:
		script = common.PlutusV4Script(scriptBytes)
	default:
		return nil, fmt.Errorf("unsupported script ref type %d", scriptType)
	}
	if expectedHashHex != "" {
		expectedBytes, err := hex.DecodeString(expectedHashHex)
		if err != nil {
			return nil, fmt.Errorf("invalid script hash hex %q: %w", expectedHashHex, err)
		}
		if len(expectedBytes) != common.Blake2b224Size {
			return nil, fmt.Errorf("invalid script hash length: expected %d bytes, got %d", common.Blake2b224Size, len(expectedBytes))
		}
		var expected common.Blake2b224
		copy(expected[:], expectedBytes)
		if computed := script.Hash(); computed != expected {
			return nil, fmt.Errorf("reference script hash mismatch: computed %s, provider claimed %s",
				hex.EncodeToString(computed.Bytes()), expectedHashHex)
		}
	}
	return &common.ScriptRef{Type: scriptType, Script: script}, nil
}

package scriptutil

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/common"
)

// ErrNativeScriptNoType reports native-script JSON without a "type", which is
// also what a JSON null decodes to.
var ErrNativeScriptNoType = errors.New("native script JSON has no type")

// nativeScriptJSON is the cardano-cli simple-script JSON form.
type nativeScriptJSON struct {
	Type     string             `json:"type"`
	KeyHash  *string            `json:"keyHash"`
	Slot     *uint64            `json:"slot"`
	Required *uint64            `json:"required"`
	Scripts  []nativeScriptJSON `json:"scripts"`
}

// maxNativeScriptDepth bounds recursion over caller- or provider-supplied
// JSON: the top-level script may hold this many levels of nested scripts.
const maxNativeScriptDepth = 64

// NativeScriptCborFromJSON encodes the cardano-cli simple-script JSON form of
// a native script (sig, all, any, atLeast, after, before) as its ledger CBOR.
//
// Unknown fields, and fields that do not belong to a script's type, are
// rejected rather than dropped: a misspelled "scripts" or a stray "slot" on a
// sig script would otherwise silently change what the policy authorizes.
func NativeScriptCborFromJSON(data []byte) ([]byte, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var script nativeScriptJSON
	if err := dec.Decode(&script); err != nil {
		return nil, fmt.Errorf("invalid native script JSON: %w", err)
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return nil, errors.New("invalid native script JSON: trailing data")
	}
	if script.Type == "" {
		return nil, ErrNativeScriptNoType
	}
	encoded, err := script.toCBORValue(0)
	if err != nil {
		return nil, err
	}
	return cbor.Encode(encoded)
}

// toCBORValue returns the ledger CBOR structure of a native script:
// [0, keyhash] | [1, [scripts]] | [2, [scripts]] | [3, n, [scripts]] |
// [4, slot] (valid from slot) | [5, slot] (valid before slot).
func (s *nativeScriptJSON) toCBORValue(depth int) ([]any, error) {
	if depth > maxNativeScriptDepth {
		return nil, fmt.Errorf("nesting exceeds %d levels", maxNativeScriptDepth)
	}
	if err := s.checkFields(); err != nil {
		return nil, err
	}
	children := func() (any, error) {
		// A missing or null list decodes to nil, unlike an explicit empty
		// one; accepting it would make an always-true all or atLeast 0, or
		// an always-false any, out of incomplete input.
		if s.Scripts == nil {
			return nil, fmt.Errorf("%s script has no scripts", s.Type)
		}
		out := make([]any, 0, len(s.Scripts))
		for i := range s.Scripts {
			child, err := s.Scripts[i].toCBORValue(depth + 1)
			if err != nil {
				return nil, err
			}
			out = append(out, child)
		}
		// cardano-binary frames a list longer than 23 as indefinite-length,
		// and the script hash covers that framing, so a definite-length list
		// would give a different policy ID than the ledger.
		if len(out) > 23 {
			return cbor.IndefLengthList(out), nil
		}
		return out, nil
	}
	slot := func() (uint64, error) {
		if s.Slot == nil {
			return 0, fmt.Errorf("%s script has no slot", s.Type)
		}
		return *s.Slot, nil
	}
	switch s.Type {
	case "sig":
		if s.KeyHash == nil {
			return nil, errors.New("sig script has no key hash")
		}
		keyHash, err := hex.DecodeString(*s.KeyHash)
		if err != nil || len(keyHash) != common.Blake2b224Size {
			return nil, fmt.Errorf("invalid key hash %q", *s.KeyHash)
		}
		return []any{0, keyHash}, nil
	case "all", "any":
		scripts, err := children()
		if err != nil {
			return nil, err
		}
		tag := 1
		if s.Type == "any" {
			tag = 2
		}
		return []any{tag, scripts}, nil
	case "atLeast":
		if s.Required == nil {
			return nil, errors.New("atLeast script has no required count")
		}
		scripts, err := children()
		if err != nil {
			return nil, err
		}
		if *s.Required > uint64(len(s.Scripts)) {
			return nil, fmt.Errorf("atLeast requires %d of %d scripts", *s.Required, len(s.Scripts))
		}
		return []any{3, *s.Required, scripts}, nil
	case "after":
		v, err := slot()
		if err != nil {
			return nil, err
		}
		return []any{4, v}, nil
	case "before":
		v, err := slot()
		if err != nil {
			return nil, err
		}
		return []any{5, v}, nil
	default:
		return nil, fmt.Errorf("unsupported native script type %q", s.Type)
	}
}

// checkFields rejects a field set on a script type that does not use it.
// Null fields decode to nil and pass, since generic serializers write them.
func (s *nativeScriptJSON) checkFields() error {
	var key, slot, required, scripts bool
	switch s.Type {
	case "sig":
		key = true
	case "all", "any":
		scripts = true
	case "atLeast":
		required, scripts = true, true
	case "after", "before":
		slot = true
	default:
		return fmt.Errorf("unsupported native script type %q", s.Type)
	}
	for _, f := range []struct {
		name         string
		set, allowed bool
	}{
		{"keyHash", s.KeyHash != nil, key},
		{"slot", s.Slot != nil, slot},
		{"required", s.Required != nil, required},
		{"scripts", s.Scripts != nil, scripts},
	} {
		if f.set && !f.allowed {
			return fmt.Errorf("%s script has unexpected field %q", s.Type, f.name)
		}
	}
	return nil
}

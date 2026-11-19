package scriptutil

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/common"
)

// ErrNativeScriptNoType reports native-script JSON without a "type", which is
// also what a JSON null decodes to.
var ErrNativeScriptNoType = errors.New("native script JSON has no type")

// nativeScriptJSON is the cardano-cli simple-script JSON form.
type nativeScriptJSON struct {
	Type     string             `json:"type"`
	KeyHash  string             `json:"keyHash"`
	Slot     *uint64            `json:"slot"`
	Required *uint64            `json:"required"`
	Scripts  []nativeScriptJSON `json:"scripts"`
}

// maxNativeScriptDepth bounds recursion over caller- or provider-supplied
// JSON.
const maxNativeScriptDepth = 64

// NativeScriptCborFromJSON encodes the cardano-cli simple-script JSON form of
// a native script (sig, all, any, atLeast, after, before) as its ledger CBOR.
func NativeScriptCborFromJSON(data []byte) ([]byte, error) {
	var script nativeScriptJSON
	if err := json.Unmarshal(data, &script); err != nil {
		return nil, fmt.Errorf("invalid native script JSON: %w", err)
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
	children := func() ([]any, error) {
		out := make([]any, 0, len(s.Scripts))
		for i := range s.Scripts {
			child, err := s.Scripts[i].toCBORValue(depth + 1)
			if err != nil {
				return nil, err
			}
			out = append(out, child)
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
		keyHash, err := hex.DecodeString(s.KeyHash)
		if err != nil || len(keyHash) != common.Blake2b224Size {
			return nil, fmt.Errorf("invalid key hash %q", s.KeyHash)
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
		if *s.Required > uint64(len(s.Scripts)) {
			return nil, fmt.Errorf("atLeast requires %d of %d scripts", *s.Required, len(s.Scripts))
		}
		scripts, err := children()
		if err != nil {
			return nil, err
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

package apollo

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/common"
)

// The timelock reference script 7c169ca1... on preprod, in the JSON form
// Koios serves it; it nests every script form.
const nestedNativeScriptJSON = `{"type": "all", "scripts": [{"type": "any", "scripts": [{"type": "sig",
"keyHash": "da0eb5ed7611482ec5089b69d870e0c56c1c45180256112398e0835b"}]}, {"type": "atLeast",
"scripts": [{"slot": 1000, "type": "after"}, {"slot": 999999999999999, "type": "before"}], "required": 1}]}`

const nestedNativeScriptHash = "7c169ca1dfa6220de314daefc1c49b5111e3eb0eebe150fb2cfce774"

func TestNativeScriptFromJSONNestedScriptHash(t *testing.T) {
	script, err := NativeScriptFromJSON([]byte(nestedNativeScriptJSON))
	if err != nil {
		t.Fatal(err)
	}
	if got := script.Hash().String(); got != nestedNativeScriptHash {
		t.Fatalf("Hash() = %s, want %s", got, nestedNativeScriptHash)
	}
}

func TestNativeScriptFromJSONForms(t *testing.T) {
	const keyHex = "da0eb5ed7611482ec5089b69d870e0c56c1c45180256112398e0835b"
	key, err := hex.DecodeString(keyHex)
	if err != nil {
		t.Fatal(err)
	}
	sig := fmt.Sprintf(`{"type":"sig","keyHash":%q}`, keyHex)
	tests := []struct {
		name string
		json string
		want []any
	}{
		{"sig", sig, []any{0, key}},
		{"all", `{"type":"all","scripts":[` + sig + `]}`, []any{1, []any{[]any{0, key}}}},
		{"any", `{"type":"any","scripts":[` + sig + `]}`, []any{2, []any{[]any{0, key}}}},
		{"atLeast", `{"type":"atLeast","required":1,"scripts":[` + sig + `]}`, []any{3, 1, []any{[]any{0, key}}}},
		{"after", `{"type":"after","slot":42}`, []any{4, 42}},
		{"before", `{"type":"before","slot":42}`, []any{5, 42}},
		// Empty lists and a zero threshold are valid ledger scripts; parsing
		// must not add the restrictions NewNativeScriptNofK applies.
		{"empty all", `{"type":"all","scripts":[]}`, []any{1, []any{}}},
		{"empty any", `{"type":"any","scripts":[]}`, []any{2, []any{}}},
		{"atLeast zero of none", `{"type":"atLeast","required":0,"scripts":[]}`, []any{3, 0, []any{}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want, err := cbor.Encode(tt.want)
			if err != nil {
				t.Fatal(err)
			}
			script, err := NativeScriptFromJSON([]byte(tt.json))
			if err != nil {
				t.Fatal(err)
			}
			if got := script.Cbor(); !bytes.Equal(got, want) {
				t.Fatalf("Cbor() = %x, want %x", got, want)
			}
			var decoded common.NativeScript
			if _, err := cbor.Decode(want, &decoded); err != nil {
				t.Fatal(err)
			}
			if script.Hash() != decoded.Hash() {
				t.Fatalf("Hash() = %s, want %s", script.Hash(), decoded.Hash())
			}
		})
	}
}

func TestNativeScriptFromJSONRejectsBadInput(t *testing.T) {
	tests := map[string]string{
		"invalid json":      `{"type":`,
		"not an object":     `[1,2]`,
		"null":              `null`,
		"no type":           `{}`,
		"unknown type":      `{"type":"maybe"}`,
		"bad key hash":      `{"type":"sig","keyHash":"abcd"}`,
		"missing required":  `{"type":"atLeast","scripts":[]}`,
		"required too high": `{"type":"atLeast","required":2,"scripts":[{"type":"after","slot":1}]}`,
		"missing slot":      `{"type":"before"}`,
		"negative slot":     `{"type":"after","slot":-1}`,
		"bad nested script": `{"type":"all","scripts":[{"type":"sig","keyHash":"zz"}]}`,
		"too deep":          strings.Repeat(`{"type":"all","scripts":[`, 66) + strings.Repeat(`]}`, 66),
	}
	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := NativeScriptFromJSON([]byte(input)); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

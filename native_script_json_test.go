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

// The ledger frames a child list longer than 23 as an indefinite-length
// array, so the policy ID of such a script depends on that framing. Wanted
// hashes are from cardano-cli 11.0.0.0 `hash script`; key hashes are i+1 as
// 28-byte big-endian hex.
func TestNativeScriptFromJSONLongChildListsMatchCardanoCli(t *testing.T) {
	sigs := func(n, offset int) string {
		out := make([]string, n)
		for i := range out {
			out[i] = fmt.Sprintf(`{"type":"sig","keyHash":"%056x"}`, i+offset+1)
		}
		return strings.Join(out, ",")
	}
	tests := []struct {
		name, json, want string
	}{
		{"all of 23", `{"type":"all","scripts":[` + sigs(23, 0) + `]}`,
			"9da1f6c87b4d562498db5ae7088f84ef849626308ea60f6066fb8a8f"},
		{"all of 24", `{"type":"all","scripts":[` + sigs(24, 0) + `]}`,
			"29798b4edbc1d8cc390fcc023ec99cd3db58bde4e79ce97600c363a4"},
		{"nested any of 25 and atLeast of 24", `{"type":"all","scripts":[` +
			`{"type":"any","scripts":[` + sigs(25, 100) + `]},` +
			`{"type":"atLeast","required":2,"scripts":[` + sigs(24, 200) + `]},` +
			`{"type":"before","slot":5}]}`,
			"bced127df655fe60af33f3a03288d54ac44948f6d5cddd01b7d9b327"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			script, err := NativeScriptFromJSON([]byte(tt.json))
			if err != nil {
				t.Fatal(err)
			}
			if got := script.Hash().String(); got != tt.want {
				t.Fatalf("Hash() = %s, want %s (cbor %x)", got, tt.want, script.Cbor())
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

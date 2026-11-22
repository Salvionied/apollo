package apollo

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math"
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
		{"slot zero", `{"type":"after","slot":0}`, []any{4, 0}},
		{"max slot", fmt.Sprintf(`{"type":"before","slot":%d}`, uint64(math.MaxUint64)), []any{5, uint64(math.MaxUint64)}},
		{"atLeast zero of one", `{"type":"atLeast","required":0,"scripts":[` + sig + `]}`, []any{3, 0, []any{[]any{0, key}}}},
		// A generic serializer may write unused fields as null.
		{"null unused fields", fmt.Sprintf(`{"type":"sig","keyHash":%q,"slot":null,"required":null,"scripts":null}`, keyHex), []any{0, key}},
		{"65 nested scripts", strings.Repeat(`{"type":"all","scripts":[`, 64) + `{"type":"all","scripts":[]}` + strings.Repeat(`]}`, 64),
			nestedAll(65)},
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

// nestedAll returns n all scripts, each holding the next, the innermost empty.
func nestedAll(n int) []any {
	script := []any{1, []any{}}
	for range n - 1 {
		script = []any{1, []any{script}}
	}
	return script
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
			// AttachScript and Clone re-encode the script; the framing must survive.
			encoded, err := cbor.Encode(&script)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(encoded, script.Cbor()) {
				t.Fatalf("re-encoded %x, want %x", encoded, script.Cbor())
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
		"nested null":       `{"type":"all","scripts":[null]}`,
		"trailing data":     `{"type":"after","slot":1}{}`,
		"negative required": `{"type":"atLeast","required":-1,"scripts":[]}`,
		"sig no key hash":   `{"type":"sig"}`,
		// Missing or null scripts would otherwise be an always-true all or
		// atLeast 0, or an always-false any.
		"all no scripts":         `{"type":"all"}`,
		"all null scripts":       `{"type":"all","scripts":null}`,
		"any no scripts":         `{"type":"any"}`,
		"any null scripts":       `{"type":"any","scripts":null}`,
		"atLeast 0 no scripts":   `{"type":"atLeast","required":0}`,
		"atLeast 0 null scripts": `{"type":"atLeast","required":0,"scripts":null}`,
		"atLeast 1 no scripts":   `{"type":"atLeast","required":1}`,
		"unknown field":          `{"type":"all","scripts":[],"comment":"x"}`,
		"scripts typo":           `{"type":"all","script":[{"type":"after","slot":1}]}`,
		"nested unknown field":   `{"type":"all","scripts":[{"type":"after","slot":1,"slots":2}]}`,
		"keyHash on after":       `{"type":"after","slot":1,"keyHash":"da0eb5ed7611482ec5089b69d870e0c56c1c45180256112398e0835b"}`,
		"slot on sig":            `{"type":"sig","keyHash":"da0eb5ed7611482ec5089b69d870e0c56c1c45180256112398e0835b","slot":5}`,
		"required on all":        `{"type":"all","required":1,"scripts":[]}`,
		"scripts on sig":         `{"type":"sig","keyHash":"da0eb5ed7611482ec5089b69d870e0c56c1c45180256112398e0835b","scripts":[]}`,
		"scripts on before":      `{"type":"before","slot":1,"scripts":[]}`,
		"slot on atLeast":        `{"type":"atLeast","required":0,"slot":1,"scripts":[]}`,
	}
	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := NativeScriptFromJSON([]byte(input)); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

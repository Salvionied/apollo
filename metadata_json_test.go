package apollo

import (
	"math/big"
	"strings"
	"testing"
)

func TestShelleyMetadataFromJSONNoSchemaSuccess(t *testing.T) {
	jsonData := []byte(`{
		"0": 42,
		"1": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"2": "0xABcd",
		"3": [1, "0x0A", {"nested": "value"}],
		"4": {
			"key": "value",
			"7": 8,
			"0xAA": "0xBB"
		}
	}`)

	metadata, err := ShelleyMetadataFromJSON(jsonData)
	if err != nil {
		t.Fatalf("ShelleyMetadataFromJSON: %v", err)
	}

	requireBigIntString(t, metadata[0], "42")
	if got := metadata[1]; got != strings.Repeat("a", metadataStringMaxBytes) {
		t.Fatalf("metadata[1] = %v, want 64-byte string", got)
	}
	requireBytes(t, metadata[2], []byte{0xab, 0xcd})

	list, ok := metadata[3].([]any)
	if !ok {
		t.Fatalf("metadata[3] type = %T, want []any", metadata[3])
	}
	if len(list) != 3 {
		t.Fatalf("metadata[3] length = %d, want 3", len(list))
	}
	requireBigIntString(t, list[0], "1")
	requireBytes(t, list[1], []byte{0x0a})
	nestedMap, ok := list[2].(MetadataMap)
	if !ok {
		t.Fatalf("metadata[3][2] type = %T, want MetadataMap", list[2])
	}
	if got := findMetadataMapValue(t, nestedMap, "nested"); got != "value" {
		t.Fatalf("nested value = %v, want value", got)
	}

	metadataMap, ok := metadata[4].(MetadataMap)
	if !ok {
		t.Fatalf("metadata[4] type = %T, want MetadataMap", metadata[4])
	}
	if got := findMetadataMapValue(t, metadataMap, "key"); got != "value" {
		t.Fatalf("map string key value = %v, want value", got)
	}
	requireBigIntString(t, findMetadataMapValue(t, metadataMap, big.NewInt(7)), "8")
	requireBytes(t, findMetadataMapValue(t, metadataMap, []byte{0xaa}), []byte{0xbb})
}

func TestShelleyMetadataFromJSONNoSchemaMapOrderIsStable(t *testing.T) {
	metadata, err := ShelleyMetadataFromJSON([]byte(`{"0":{"b":1,"0x02":2,"10":3,"a":4}}`))
	if err != nil {
		t.Fatalf("ShelleyMetadataFromJSON: %v", err)
	}

	metadataMap, ok := metadata[0].(MetadataMap)
	if !ok {
		t.Fatalf("metadata[0] type = %T, want MetadataMap", metadata[0])
	}
	if len(metadataMap) != 4 {
		t.Fatalf("metadata[0] length = %d, want 4", len(metadataMap))
	}

	wantKeys := []any{[]byte{0x02}, big.NewInt(10), "a", "b"}
	for idx, wantKey := range wantKeys {
		if !metadataMapKeysEqual(metadataMap[idx].Key, wantKey) {
			t.Fatalf("entry %d key = %#v, want %#v", idx, metadataMap[idx].Key, wantKey)
		}
	}
}

func TestShelleyMetadataFromJSONNoSchemaErrors(t *testing.T) {
	oversize := strings.Repeat("a", metadataStringMaxBytes+1)
	tests := []struct {
		name    string
		json    string
		wantErr string
	}{
		{
			name:    "top level array",
			json:    `[1]`,
			wantErr: "metadata JSON top level must be an object",
		},
		{
			name:    "leading zero key",
			json:    `{"01":1}`,
			wantErr: `metadata top-level key "01" must be an unsigned integer without leading zeroes`,
		},
		{
			name:    "invalid key",
			json:    `{"not-int":1}`,
			wantErr: `metadata top-level key "not-int" must be an unsigned integer without leading zeroes`,
		},
		{
			name:    "null value",
			json:    `{"1":null}`,
			wantErr: "metadata[1]: null metadata values are not allowed",
		},
		{
			name:    "boolean value",
			json:    `{"1":true}`,
			wantErr: "metadata[1]: boolean metadata values are not allowed",
		},
		{
			name:    "integer out of range",
			json:    `{"1":18446744073709551616}`,
			wantErr: "metadata[1]: metadata integer 18446744073709551616 is outside the supported range",
		},
		{
			name:    "oversize string",
			json:    `{"1":"` + oversize + `"}`,
			wantErr: "metadata[1]: metadata text value exceeds 64 UTF-8 bytes",
		},
		{
			name:    "multiple JSON values",
			json:    `{"1":1} {"2":2}`,
			wantErr: "decode metadata JSON: multiple JSON values",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ShelleyMetadataFromJSON([]byte(tt.json))
			requireExactError(t, err, tt.wantErr)
		})
	}
}

func TestShelleyMetadataFromJSONIntegerLowerBound(t *testing.T) {
	// The Cardano transaction metadata integer range is [-(2^64-1), 2^64-1]
	// (cardano-api validateTxMetadata rejects n < negate(maxBound::Word64)).
	// -(2^64-1) is the smallest accepted value; -(2^64) must be rejected.
	t.Run("min accepted", func(t *testing.T) {
		md, err := ShelleyMetadataFromJSON([]byte(`{"1":-18446744073709551615}`))
		if err != nil {
			t.Fatalf("expected -18446744073709551615 to be accepted, got error: %v", err)
		}
		requireBigIntString(t, md[1], "-18446744073709551615")
	})
	t.Run("below min rejected", func(t *testing.T) {
		_, err := ShelleyMetadataFromJSON([]byte(`{"1":-18446744073709551616}`))
		requireExactError(t, err, "metadata[1]: metadata integer -18446744073709551616 is outside the supported range")
	})
}

func TestShelleyMetadataFromJSONWithDetailedSchemaSuccess(t *testing.T) {
	jsonData := []byte(`{
		"0": {"int": -1},
		"1": {"bytes": "ABcd"},
		"2": {"string": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		"3": {"list": [{"int": 1}, {"bytes": "0A"}]},
		"4": {"map": [
			{"k": {"string": "key"}, "v": {"int": 9}},
			{"k": {"bytes": "AA"}, "v": {"list": []}}
		]}
	}`)

	schema := MetadataJSONSchema(MetadataJSONDetailedSchema)
	metadata, err := ShelleyMetadataFromJSONWithSchema(jsonData, schema)
	if err != nil {
		t.Fatalf("ShelleyMetadataFromJSONWithSchema: %v", err)
	}

	requireBigIntString(t, metadata[0], "-1")
	requireBytes(t, metadata[1], []byte{0xab, 0xcd})
	if got := metadata[2]; got != strings.Repeat("a", metadataStringMaxBytes) {
		t.Fatalf("metadata[2] = %v, want 64-byte string", got)
	}

	list, ok := metadata[3].([]any)
	if !ok {
		t.Fatalf("metadata[3] type = %T, want []any", metadata[3])
	}
	requireBigIntString(t, list[0], "1")
	requireBytes(t, list[1], []byte{0x0a})

	metadataMap, ok := metadata[4].(MetadataMap)
	if !ok {
		t.Fatalf("metadata[4] type = %T, want MetadataMap", metadata[4])
	}
	if len(metadataMap) != 2 {
		t.Fatalf("metadata[4] length = %d, want 2", len(metadataMap))
	}
	requireMetadataMapEntry(t, metadataMap[0], "key", "9")
	requireBytes(t, metadataMap[1].Key, []byte{0xaa})
	if nested, ok := metadataMap[1].Value.([]any); !ok || len(nested) != 0 {
		t.Fatalf("metadata[4][1].Value = %#v, want empty []any", metadataMap[1].Value)
	}
}

func TestShelleyMetadataFromJSONWithDetailedSchemaErrors(t *testing.T) {
	oversize := strings.Repeat("a", metadataStringMaxBytes+1)
	tests := []struct {
		name    string
		json    string
		wantErr string
	}{
		{
			name:    "top level key invalid",
			json:    `{"01":{"int":1}}`,
			wantErr: `metadata top-level key "01" must be an unsigned integer without leading zeroes`,
		},
		{
			name:    "value not object",
			json:    `{"1":1}`,
			wantErr: "metadata[1]: detailed metadata value must be an object",
		},
		{
			name:    "wrong field count",
			json:    `{"1":{"int":1,"string":"x"}}`,
			wantErr: "metadata[1]: detailed metadata value must have exactly one field",
		},
		{
			name:    "unknown field",
			json:    `{"1":{"bad":1}}`,
			wantErr: `metadata[1]: unknown detailed metadata field "bad"`,
		},
		{
			name:    "int wrong type",
			json:    `{"1":{"int":"1"}}`,
			wantErr: "metadata[1].int: expected JSON number",
		},
		{
			name:    "int out of range",
			json:    `{"1":{"int":18446744073709551616}}`,
			wantErr: "metadata[1].int: metadata integer 18446744073709551616 is outside the supported range",
		},
		{
			name:    "bytes wrong type",
			json:    `{"1":{"bytes":1}}`,
			wantErr: "metadata[1].bytes: expected hex string",
		},
		{
			name:    "bytes malformed hex",
			json:    `{"1":{"bytes":"zz"}}`,
			wantErr: "metadata[1].bytes: expected hex-encoded bytes: encoding/hex: invalid byte: U+007A 'z'",
		},
		{
			name:    "string wrong type",
			json:    `{"1":{"string":1}}`,
			wantErr: "metadata[1].string: expected string",
		},
		{
			name:    "string oversize",
			json:    `{"1":{"string":"` + oversize + `"}}`,
			wantErr: "metadata[1].string: metadata text value exceeds 64 UTF-8 bytes",
		},
		{
			name:    "list wrong type",
			json:    `{"1":{"list":1}}`,
			wantErr: "metadata[1].list: expected array",
		},
		{
			name:    "map wrong type",
			json:    `{"1":{"map":1}}`,
			wantErr: "metadata[1].map: expected array",
		},
		{
			name:    "map entry wrong shape",
			json:    `{"1":{"map":[{"k":{"string":"x"}}]}}`,
			wantErr: "metadata[1].map[0]: map entry must be an object with k and v fields",
		},
		{
			name:    "multiple JSON values",
			json:    `{"1":{"int":1}} true`,
			wantErr: "decode metadata JSON: multiple JSON values",
		},
		{
			name:    "unsupported schema",
			json:    `{"1":{"int":1}}`,
			wantErr: "unsupported metadata JSON schema 99",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := MetadataJSONSchema(MetadataJSONDetailedSchema)
			if tt.name == "unsupported schema" {
				schema = MetadataJSONSchema(99)
			}
			_, err := ShelleyMetadataFromJSONWithSchema([]byte(tt.json), schema)
			requireExactError(t, err, tt.wantErr)
		})
	}
}

func requireExactError(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error %q", want)
	}
	if err.Error() != want {
		t.Fatalf("error = %q, want %q", err.Error(), want)
	}
}

func requireBigIntString(t *testing.T, value any, want string) {
	t.Helper()
	got, ok := value.(*big.Int)
	if !ok {
		t.Fatalf("value type = %T, want *big.Int", value)
	}
	if got.String() != want {
		t.Fatalf("integer = %s, want %s", got.String(), want)
	}
}

func requireBytes(t *testing.T, value any, want []byte) {
	t.Helper()
	got, ok := value.([]byte)
	if !ok {
		t.Fatalf("value type = %T, want []byte", value)
	}
	if string(got) != string(want) {
		t.Fatalf("bytes = %x, want %x", got, want)
	}
}

func requireMetadataMapEntry(t *testing.T, entry MetadataMapEntry, wantKey string, wantValue string) {
	t.Helper()
	if entry.Key != wantKey {
		t.Fatalf("entry key = %v, want %s", entry.Key, wantKey)
	}
	requireBigIntString(t, entry.Value, wantValue)
}

func findMetadataMapValue(t *testing.T, entries MetadataMap, wantKey any) any {
	t.Helper()
	for _, entry := range entries {
		if metadataMapKeysEqual(entry.Key, wantKey) {
			return entry.Value
		}
	}
	t.Fatalf("missing metadata map key %#v in %#v", wantKey, entries)
	return nil
}

func metadataMapKeysEqual(a any, b any) bool {
	switch want := b.(type) {
	case string:
		got, ok := a.(string)
		return ok && got == want
	case *big.Int:
		got, ok := a.(*big.Int)
		return ok && got.Cmp(want) == 0
	case []byte:
		got, ok := a.([]byte)
		return ok && string(got) == string(want)
	default:
		return false
	}
}

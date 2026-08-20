package plutusencoder

import (
	"encoding/hex"
	"math/big"
	"strings"
	"testing"

	"github.com/blinklabs-io/plutigo/data"
)

func TestMarshalMapDatum(t *testing.T) {
	d := MapDatum{
		Name:  "test",
		Value: 99,
	}
	pd, err := MarshalPlutus(&d)
	if err != nil {
		t.Fatal(err)
	}

	mapData, ok := pd.(*data.Map)
	if !ok {
		t.Fatalf("expected Map, got %T", pd)
	}
	if len(mapData.Pairs) != 2 {
		t.Fatalf("expected 2 pairs, got %d", len(mapData.Pairs))
	}

	// Verify map keys and values
	key0, ok := mapData.Pairs[0][0].(*data.ByteString)
	if !ok {
		t.Fatalf("expected ByteString key at index 0, got %T", mapData.Pairs[0][0])
	}
	if string(key0.Inner) != "name" {
		t.Errorf("expected key 'name', got '%s'", string(key0.Inner))
	}
	val0, ok := mapData.Pairs[0][1].(*data.ByteString)
	if !ok {
		t.Fatalf("expected ByteString value at index 0, got %T", mapData.Pairs[0][1])
	}
	if string(val0.Inner) != "test" {
		t.Errorf("expected value 'test', got '%s'", string(val0.Inner))
	}

	key1, ok := mapData.Pairs[1][0].(*data.ByteString)
	if !ok {
		t.Fatalf("expected ByteString key at index 1, got %T", mapData.Pairs[1][0])
	}
	if string(key1.Inner) != "value" {
		t.Errorf("expected key 'value', got '%s'", string(key1.Inner))
	}
	val1, ok := mapData.Pairs[1][1].(*data.Integer)
	if !ok {
		t.Fatalf("expected Integer value at index 1, got %T", mapData.Pairs[1][1])
	}
	if val1.Inner.Int64() != 99 {
		t.Errorf("expected value 99, got %d", val1.Inner.Int64())
	}
}

func TestUnmarshalMapDatum(t *testing.T) {
	original := MapDatum{Name: "test", Value: 99}
	pd, err := MarshalPlutus(&original)
	if err != nil {
		t.Fatal(err)
	}

	var decoded MapDatum
	err = UnmarshalPlutus(pd, &decoded)
	if err != nil {
		t.Fatal(err)
	}

	if decoded.Name != "test" {
		t.Errorf("expected 'test', got '%s'", decoded.Name)
	}
	if decoded.Value != 99 {
		t.Errorf("expected 99, got %d", decoded.Value)
	}
}

func TestUnmarshalMapConstrWrongFieldCount(t *testing.T) {
	// MapDatum expects a bare Map or a Constr wrapping exactly 1 Map.
	// Construct a Constr with 2 fields to trigger the new error path.
	pd := data.NewConstr(0,
		data.NewMap(nil),
		data.NewInteger(big.NewInt(1)),
	)
	var decoded MapDatum
	err := UnmarshalPlutus(pd, &decoded)
	if err == nil {
		t.Error("expected error for Constr with wrong field count wrapping Map")
	}
}

func TestUnmarshalMapMissingRequiredKey(t *testing.T) {
	// MapDatum requires both "name" and "value"; omit "value".
	pd := data.NewMap([][2]data.PlutusData{
		{data.NewByteString([]byte("name")), data.NewByteString([]byte("test"))},
	})
	var decoded MapDatum
	err := UnmarshalPlutus(pd, &decoded)
	if err == nil {
		t.Fatal("expected error for missing required map key")
	}
	if got := err.Error(); !strings.Contains(got, "value") || !strings.Contains(got, "missing required map key") {
		t.Errorf("expected descriptive missing-key error mentioning \"value\", got: %s", got)
	}
}

func TestUnmarshalMapOptionalFieldMissing(t *testing.T) {
	// "value" is tagged plutusOptional:"true", so omitting it must succeed
	// and leave the field at its zero value.
	pd := data.NewMap([][2]data.PlutusData{
		{data.NewByteString([]byte("name")), data.NewByteString([]byte("test"))},
	})
	var decoded optionalMapDatum
	err := UnmarshalPlutus(pd, &decoded)
	if err != nil {
		t.Fatalf("expected no error for missing optional key, got: %v", err)
	}
	if decoded.Name != "test" {
		t.Errorf("expected 'test', got '%s'", decoded.Name)
	}
	if decoded.Value != 0 {
		t.Errorf("expected zero value for missing optional field, got %d", decoded.Value)
	}
}

func TestUnmarshalMapNonByteStringKey(t *testing.T) {
	// An Integer key must be rejected, not silently dropped.
	pd := data.NewMap([][2]data.PlutusData{
		{data.NewByteString([]byte("name")), data.NewByteString([]byte("test"))},
		{data.NewInteger(big.NewInt(7)), data.NewInteger(big.NewInt(99))},
	})
	var decoded MapDatum
	err := UnmarshalPlutus(pd, &decoded)
	if err == nil {
		t.Fatal("expected error for non-ByteString map key")
	}
	if got := err.Error(); !strings.Contains(got, "expected ByteString key") {
		t.Errorf("expected descriptive non-ByteString-key error, got: %s", got)
	}
}

func TestUnmarshalMapDuplicateKey(t *testing.T) {
	// Duplicate keys allow shadowing and must be rejected.
	pd := data.NewMap([][2]data.PlutusData{
		{data.NewByteString([]byte("name")), data.NewByteString([]byte("real"))},
		{data.NewByteString([]byte("name")), data.NewByteString([]byte("shadow"))},
		{data.NewByteString([]byte("value")), data.NewInteger(big.NewInt(99))},
	})
	var decoded MapDatum
	err := UnmarshalPlutus(pd, &decoded)
	if err == nil {
		t.Fatal("expected error for duplicate map key")
	}
	if got := err.Error(); !strings.Contains(got, "duplicate key") {
		t.Errorf("expected descriptive duplicate-key error, got: %s", got)
	}
}

func TestMarshalMapDuplicatePlutusKey(t *testing.T) {
	type duplicateKeyDatum struct {
		_       struct{} `plutusType:"Map"`
		Name    string   `plutusType:"StringBytes" plutusKey:"name"`
		Alias   string   `plutusType:"StringBytes" plutusKey:"name"`
		Version int64    `plutusType:"Int"`
	}

	_, err := MarshalPlutus(&duplicateKeyDatum{Name: "real", Alias: "shadow", Version: 1})
	if err == nil {
		t.Fatal("expected error for duplicate plutusKey")
	}
	if got := err.Error(); !strings.Contains(got, "duplicate map key") || !strings.Contains(got, "name") {
		t.Fatalf("expected descriptive duplicate-key error, got: %s", got)
	}
}

func TestMarshalMapPlutusKeyCollidesWithDefaultName(t *testing.T) {
	type collidingKeyDatum struct {
		_     struct{} `plutusType:"Map"`
		Name  string   `plutusType:"StringBytes"`
		Alias string   `plutusType:"StringBytes" plutusKey:"Name"`
	}

	_, err := MarshalPlutus(&collidingKeyDatum{Name: "real", Alias: "shadow"})
	if err == nil {
		t.Fatal("expected error for plutusKey/default-name collision")
	}
	if got := err.Error(); !strings.Contains(got, "duplicate map key") || !strings.Contains(got, "Name") {
		t.Fatalf("expected descriptive duplicate-key error, got: %s", got)
	}
}

func TestMarshalSliceMapDuplicateElementKey(t *testing.T) {
	type mapEntry struct {
		Key   string `plutusType:"StringBytes"`
		Value int64  `plutusType:"Int"`
	}
	type duplicateSliceDatum struct {
		_       struct{}   `plutusType:"DefList" plutusConstr:"0"`
		Entries []mapEntry `plutusType:"Map"`
	}

	original := duplicateSliceDatum{Entries: []mapEntry{
		{Key: "same", Value: 1},
		{Key: "same", Value: 2},
	}}
	_, err := MarshalPlutus(&original)
	if err == nil {
		t.Fatal("expected error for duplicate slice-map element key")
	}
	if got := err.Error(); !strings.Contains(got, "duplicate map key") {
		t.Fatalf("expected descriptive duplicate-key error, got: %s", got)
	}
}

func TestRoundTripSliceMapUntaggedStringKey(t *testing.T) {
	type mapEntry struct {
		Key   string
		Value int64 `plutusType:"Int"`
	}
	type sliceMapDatum struct {
		_       struct{}   `plutusType:"DefList" plutusConstr:"0"`
		Entries []mapEntry `plutusType:"Map"`
	}

	original := sliceMapDatum{Entries: []mapEntry{
		{Key: "first", Value: 1},
		{Key: "second", Value: 2},
	}}
	pd, err := MarshalPlutus(&original)
	if err != nil {
		t.Fatal(err)
	}

	var decoded sliceMapDatum
	if err := UnmarshalPlutus(pd, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(decoded.Entries))
	}
	if decoded.Entries[0].Key != "first" || decoded.Entries[0].Value != 1 {
		t.Errorf("unexpected first entry: %+v", decoded.Entries[0])
	}
	if decoded.Entries[1].Key != "second" || decoded.Entries[1].Value != 2 {
		t.Errorf("unexpected second entry: %+v", decoded.Entries[1])
	}
}

func TestRoundTripSliceMapHexStringKey(t *testing.T) {
	type mapEntry struct {
		Key   string `plutusType:"HexString"`
		Value int64  `plutusType:"Int"`
	}
	type sliceMapDatum struct {
		_       struct{}   `plutusType:"DefList" plutusConstr:"0"`
		Entries []mapEntry `plutusType:"Map"`
	}

	original := sliceMapDatum{Entries: []mapEntry{{Key: "aabbccdd", Value: 7}}}
	pd, err := MarshalPlutus(&original)
	if err != nil {
		t.Fatal(err)
	}

	var decoded sliceMapDatum
	if err := UnmarshalPlutus(pd, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.Entries) != 1 || decoded.Entries[0].Key != "aabbccdd" || decoded.Entries[0].Value != 7 {
		t.Fatalf("unexpected decoded entries: %+v", decoded.Entries)
	}
}

func TestRoundTripSliceMapIntegerKey(t *testing.T) {
	type mapEntry struct {
		Key   int64 `plutusType:"Int"`
		Value int64 `plutusType:"Int"`
	}
	type sliceMapDatum struct {
		_       struct{}   `plutusType:"DefList" plutusConstr:"0"`
		Entries []mapEntry `plutusType:"Map"`
	}

	original := sliceMapDatum{Entries: []mapEntry{{Key: 7, Value: 42}}}
	pd, err := MarshalPlutus(&original)
	if err != nil {
		t.Fatal(err)
	}

	var decoded sliceMapDatum
	if err := UnmarshalPlutus(pd, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.Entries) != 1 || decoded.Entries[0].Key != 7 || decoded.Entries[0].Value != 42 {
		t.Fatalf("unexpected decoded entries: %+v", decoded.Entries)
	}
}

func TestRoundTripNativeStringIntMap(t *testing.T) {
	type nativeMapDatum struct {
		_      struct{}         `plutusType:"DefList" plutusConstr:"0"`
		Values map[string]int64 `plutusType:"Map"`
	}

	original := nativeMapDatum{Values: map[string]int64{
		"beta":  2,
		"alpha": 1,
		"gamma": 3,
	}}
	pd, err := MarshalPlutus(&original)
	if err != nil {
		t.Fatal(err)
	}

	encoded, err := data.Encode(pd)
	if err != nil {
		t.Fatal(err)
	}
	// Sorted by encoded key bytes: "beta"(len4) before "alpha"/"gamma"(len5).
	if got := hex.EncodeToString(encoded); got != "d87981a344626574610245616c706861014567616d6d6103" {
		t.Fatalf("expected deterministic native map CBOR, got %s", got)
	}

	var decoded nativeMapDatum
	if err := UnmarshalPlutus(pd, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.Values) != 3 || decoded.Values["alpha"] != 1 || decoded.Values["beta"] != 2 || decoded.Values["gamma"] != 3 {
		t.Fatalf("unexpected decoded map: %+v", decoded.Values)
	}
}

func TestNativeMapUnsupportedKey(t *testing.T) {
	type invalidMapDatum struct {
		_      struct{}           `plutusType:"DefList" plutusConstr:"0"`
		Values map[float64]string `plutusType:"Map"`
	}

	_, err := MarshalPlutus(&invalidMapDatum{Values: map[float64]string{1.5: "yes"}})
	if err == nil {
		t.Fatal("expected error for unsupported native map key")
	}
}

func TestNativeMapDuplicateEncodedKeys(t *testing.T) {
	type invalidMapDatum struct {
		_      struct{}                     `plutusType:"DefList" plutusConstr:"0"`
		Values map[duplicateNativeKey]int64 `plutusType:"Map"`
	}

	_, err := MarshalPlutus(&invalidMapDatum{Values: map[duplicateNativeKey]int64{
		"first":  1,
		"second": 2,
	}})
	if err == nil {
		t.Fatal("expected error for duplicate encoded native map key")
	}
}

type duplicateNativeKey string

func (key duplicateNativeKey) ToPlutusData() (data.PlutusData, error) {
	return data.NewByteString([]byte("duplicate")), nil
}

func (key duplicateNativeKey) FromPlutusData(pd data.PlutusData, res any) error {
	return nil
}

func TestUnmarshalMapInvalidOptionalTag(t *testing.T) {
	type badOptionalDatum struct {
		_     struct{} `plutusType:"Map"`
		Value int64    `plutusType:"Int" plutusKey:"value" plutusOptional:"yes"`
	}
	// Key absent, so the optional tag is consulted; "yes" is not a valid bool.
	pd := data.NewMap(nil)
	var decoded badOptionalDatum
	err := UnmarshalPlutus(pd, &decoded)
	if err == nil {
		t.Fatal("expected error for invalid plutusOptional tag value")
	}
	if got := err.Error(); !strings.Contains(got, "plutusOptional") {
		t.Errorf("expected descriptive invalid-tag error, got: %s", got)
	}
}

func TestUnmarshalMapExtraKeysIgnored(t *testing.T) {
	// Unknown ByteString keys remain allowed for forward compatibility.
	pd := data.NewMap([][2]data.PlutusData{
		{data.NewByteString([]byte("name")), data.NewByteString([]byte("test"))},
		{data.NewByteString([]byte("value")), data.NewInteger(big.NewInt(99))},
		{data.NewByteString([]byte("future")), data.NewInteger(big.NewInt(1))},
	})
	var decoded MapDatum
	err := UnmarshalPlutus(pd, &decoded)
	if err != nil {
		t.Fatalf("expected extra ByteString keys to be ignored, got: %v", err)
	}
	if decoded.Name != "test" || decoded.Value != 99 {
		t.Errorf("expected {test, 99}, got {%s, %d}", decoded.Name, decoded.Value)
	}
}

func TestUnmarshalMultiFieldMapValueRejectsConstr(t *testing.T) {
	type mapEntry struct {
		Key   string `plutusType:"StringBytes"`
		Left  int64  `plutusType:"Int"`
		Right int64  `plutusType:"Int"`
	}
	type sliceMapDatum struct {
		_       struct{}   `plutusType:"DefList" plutusConstr:"0"`
		Entries []mapEntry `plutusType:"Map"`
	}

	pd := data.NewConstr(0, data.NewMap([][2]data.PlutusData{
		{
			data.NewByteString([]byte("k")),
			data.NewConstr(0, data.NewInteger(big.NewInt(1)), data.NewInteger(big.NewInt(2))),
		},
	}))
	var decoded sliceMapDatum
	if err := UnmarshalPlutus(pd, &decoded); err == nil {
		t.Fatal("expected error when Constr is provided where List is required for multi-field map values")
	}
}

func TestRoundTripMultiFieldMapValueList(t *testing.T) {
	type mapEntry struct {
		Key   string `plutusType:"StringBytes"`
		Left  int64  `plutusType:"Int"`
		Right int64  `plutusType:"Int"`
	}
	type sliceMapDatum struct {
		_       struct{}   `plutusType:"DefList" plutusConstr:"0"`
		Entries []mapEntry `plutusType:"Map"`
	}

	original := sliceMapDatum{Entries: []mapEntry{{Key: "k", Left: 1, Right: 2}}}
	pd, err := MarshalPlutus(&original)
	if err != nil {
		t.Fatal(err)
	}
	var decoded sliceMapDatum
	if err := UnmarshalPlutus(pd, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.Entries) != 1 || decoded.Entries[0] != (mapEntry{Key: "k", Left: 1, Right: 2}) {
		t.Fatalf("unexpected decoded entries: %+v", decoded.Entries)
	}
}

func TestRoundTripOptionalMapDatum(t *testing.T) {
	// A well-formed map with all keys present still round-trips, including
	// structs that carry optional tags.
	original := optionalMapDatum{Name: "test", Value: 99}
	pd, err := MarshalPlutus(&original)
	if err != nil {
		t.Fatal(err)
	}

	var decoded optionalMapDatum
	err = UnmarshalPlutus(pd, &decoded)
	if err != nil {
		t.Fatal(err)
	}

	if decoded.Name != "test" {
		t.Errorf("expected 'test', got '%s'", decoded.Name)
	}
	if decoded.Value != 99 {
		t.Errorf("expected 99, got %d", decoded.Value)
	}
}

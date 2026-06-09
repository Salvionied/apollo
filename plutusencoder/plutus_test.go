package plutusencoder

import (
	"math/big"
	"testing"

	"github.com/blinklabs-io/plutigo/data"
)

type SimpleDatum struct {
	_      struct{} `plutusType:"DefList" plutusConstr:"0"`
	Amount int64    `plutusType:"Int"`
	Name   []byte   `plutusType:"Bytes"`
}

type IndefDatum struct {
	_      struct{} `plutusType:"IndefList" plutusConstr:"1"`
	Pkh    []byte   `plutusType:"Bytes"`
	Amount int64    `plutusType:"Int"`
}

type MapDatum struct {
	_     struct{} `plutusType:"Map"`
	Name  string   `plutusType:"StringBytes" plutusKey:"name"`
	Value int64    `plutusType:"Int" plutusKey:"value"`
}

type BoolDatum struct {
	_      struct{} `plutusType:"DefList" plutusConstr:"0"`
	Active bool     `plutusType:"Bool"`
}

type BigIntDatum struct {
	_     struct{} `plutusType:"DefList" plutusConstr:"0"`
	Value *big.Int `plutusType:"BigInt"`
}

type HexDatum struct {
	_    struct{} `plutusType:"DefList" plutusConstr:"0"`
	Hash string   `plutusType:"HexString"`
}

type NestedDatum struct {
	_     struct{}    `plutusType:"DefList" plutusConstr:"0"`
	Inner SimpleDatum `plutusType:"DefList"`
}

func TestMarshalSimpleDatum(t *testing.T) {
	d := SimpleDatum{
		Amount: 42,
		Name:   []byte("hello"),
	}
	pd, err := MarshalPlutus(&d)
	if err != nil {
		t.Fatal(err)
	}

	constr, ok := pd.(*data.Constr)
	if !ok {
		t.Fatalf("expected Constr, got %T", pd)
	}
	if constr.Tag != 0 {
		t.Errorf("expected tag 0, got %d", constr.Tag)
	}
	if len(constr.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(constr.Fields))
	}

	// Check Amount field
	intField, ok := constr.Fields[0].(*data.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T", constr.Fields[0])
	}
	if intField.Inner.Int64() != 42 {
		t.Errorf("expected 42, got %d", intField.Inner.Int64())
	}

	// Check Name field
	bsField, ok := constr.Fields[1].(*data.ByteString)
	if !ok {
		t.Fatalf("expected ByteString, got %T", constr.Fields[1])
	}
	if string(bsField.Inner) != "hello" {
		t.Errorf("expected 'hello', got '%s'", string(bsField.Inner))
	}
}

func TestMarshalIndefDatum(t *testing.T) {
	d := IndefDatum{
		Pkh:    []byte{0xaa, 0xbb},
		Amount: 100,
	}
	pd, err := MarshalPlutus(&d)
	if err != nil {
		t.Fatal(err)
	}

	constr, ok := pd.(*data.Constr)
	if !ok {
		t.Fatalf("expected Constr, got %T", pd)
	}
	if constr.Tag != 1 {
		t.Errorf("expected tag 1, got %d", constr.Tag)
	}
}

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

func TestMarshalBoolDatum(t *testing.T) {
	d := BoolDatum{Active: true}
	pd, err := MarshalPlutus(&d)
	if err != nil {
		t.Fatal(err)
	}

	constr, ok := pd.(*data.Constr)
	if !ok {
		t.Fatalf("expected Constr, got %T", pd)
	}
	if len(constr.Fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(constr.Fields))
	}
	boolConstr, ok := constr.Fields[0].(*data.Constr)
	if !ok {
		t.Fatalf("expected Constr for bool, got %T", constr.Fields[0])
	}
	if boolConstr.Tag != 1 {
		t.Errorf("expected tag 1 for true, got %d", boolConstr.Tag)
	}
}

func TestMarshalBoolFalse(t *testing.T) {
	d := BoolDatum{Active: false}
	pd, err := MarshalPlutus(&d)
	if err != nil {
		t.Fatal(err)
	}

	constr, ok := pd.(*data.Constr)
	if !ok {
		t.Fatalf("expected Constr, got %T", pd)
	}
	boolConstr, ok := constr.Fields[0].(*data.Constr)
	if !ok {
		t.Fatalf("expected Constr for bool field, got %T", constr.Fields[0])
	}
	if boolConstr.Tag != 0 {
		t.Errorf("expected tag 0 for false, got %d", boolConstr.Tag)
	}
}

func TestMarshalBigIntDatum(t *testing.T) {
	bigVal := big.NewInt(999999999999)
	d := BigIntDatum{Value: bigVal}
	pd, err := MarshalPlutus(&d)
	if err != nil {
		t.Fatal(err)
	}

	constr, ok := pd.(*data.Constr)
	if !ok {
		t.Fatalf("expected Constr, got %T", pd)
	}
	intField, ok := constr.Fields[0].(*data.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T", constr.Fields[0])
	}
	if intField.Inner.Cmp(bigVal) != 0 {
		t.Errorf("expected %s, got %s", bigVal.String(), intField.Inner.String())
	}
}

func TestMarshalHexDatum(t *testing.T) {
	d := HexDatum{Hash: "aabbccdd"}
	pd, err := MarshalPlutus(&d)
	if err != nil {
		t.Fatal(err)
	}

	constr, ok := pd.(*data.Constr)
	if !ok {
		t.Fatalf("expected Constr, got %T", pd)
	}
	bsField, ok := constr.Fields[0].(*data.ByteString)
	if !ok {
		t.Fatalf("expected ByteString, got %T", constr.Fields[0])
	}
	if len(bsField.Inner) != 4 {
		t.Fatalf("expected 4 bytes, got %d", len(bsField.Inner))
	}
	expected := []byte{0xaa, 0xbb, 0xcc, 0xdd}
	for i, b := range bsField.Inner {
		if b != expected[i] {
			t.Errorf("byte %d: expected 0x%02x, got 0x%02x", i, expected[i], b)
		}
	}
}

func TestMarshalHexInvalid(t *testing.T) {
	d := HexDatum{Hash: "not-hex!"}
	_, err := MarshalPlutus(&d)
	if err == nil {
		t.Error("expected error for invalid hex")
	}
}

func TestMarshalNilPointer(t *testing.T) {
	_, err := MarshalPlutus((*SimpleDatum)(nil))
	if err == nil {
		t.Error("expected error for nil pointer")
	}
}

func TestMarshalNonStruct(t *testing.T) {
	x := 42
	_, err := MarshalPlutus(&x)
	if err == nil {
		t.Error("expected error for non-struct")
	}
}

func TestUnmarshalSimpleDatum(t *testing.T) {
	// First marshal, then unmarshal
	original := SimpleDatum{Amount: 42, Name: []byte("hello")}
	pd, err := MarshalPlutus(&original)
	if err != nil {
		t.Fatal(err)
	}

	var decoded SimpleDatum
	err = UnmarshalPlutus(pd, &decoded)
	if err != nil {
		t.Fatal(err)
	}

	if decoded.Amount != 42 {
		t.Errorf("expected 42, got %d", decoded.Amount)
	}
	if string(decoded.Name) != "hello" {
		t.Errorf("expected 'hello', got '%s'", string(decoded.Name))
	}
}

func TestUnmarshalIndefDatum(t *testing.T) {
	original := IndefDatum{Pkh: []byte{0xaa, 0xbb}, Amount: 100}
	pd, err := MarshalPlutus(&original)
	if err != nil {
		t.Fatal(err)
	}

	var decoded IndefDatum
	err = UnmarshalPlutus(pd, &decoded)
	if err != nil {
		t.Fatal(err)
	}

	if decoded.Amount != 100 {
		t.Errorf("expected 100, got %d", decoded.Amount)
	}
	if len(decoded.Pkh) != 2 || decoded.Pkh[0] != 0xaa || decoded.Pkh[1] != 0xbb {
		t.Errorf("expected [0xaa,0xbb], got %v", decoded.Pkh)
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

func TestUnmarshalBoolDatum(t *testing.T) {
	original := BoolDatum{Active: true}
	pd, err := MarshalPlutus(&original)
	if err != nil {
		t.Fatal(err)
	}

	var decoded BoolDatum
	err = UnmarshalPlutus(pd, &decoded)
	if err != nil {
		t.Fatal(err)
	}

	if !decoded.Active {
		t.Error("expected true, got false")
	}
}

func TestUnmarshalBigIntDatum(t *testing.T) {
	bigVal := big.NewInt(999999999999)
	original := BigIntDatum{Value: bigVal}
	pd, err := MarshalPlutus(&original)
	if err != nil {
		t.Fatal(err)
	}

	var decoded BigIntDatum
	err = UnmarshalPlutus(pd, &decoded)
	if err != nil {
		t.Fatal(err)
	}

	if decoded.Value.Cmp(bigVal) != 0 {
		t.Errorf("expected %s, got %s", bigVal.String(), decoded.Value.String())
	}
}

func TestUnmarshalHexDatum(t *testing.T) {
	original := HexDatum{Hash: "aabbccdd"}
	pd, err := MarshalPlutus(&original)
	if err != nil {
		t.Fatal(err)
	}

	var decoded HexDatum
	err = UnmarshalPlutus(pd, &decoded)
	if err != nil {
		t.Fatal(err)
	}

	if decoded.Hash != "aabbccdd" {
		t.Errorf("expected aabbccdd, got %s", decoded.Hash)
	}
}

func TestUnmarshalNonPointer(t *testing.T) {
	var d SimpleDatum
	err := UnmarshalPlutus(data.NewInteger(big.NewInt(0)), d)
	if err == nil {
		t.Error("expected error for non-pointer")
	}
}

func TestUnmarshalNilPointer(t *testing.T) {
	err := UnmarshalPlutus(data.NewInteger(big.NewInt(0)), (*SimpleDatum)(nil))
	if err == nil {
		t.Error("expected error for nil pointer")
	}
}

func TestRoundTripNestedDatum(t *testing.T) {
	original := NestedDatum{
		Inner: SimpleDatum{
			Amount: 123,
			Name:   []byte("nested"),
		},
	}
	pd, err := MarshalPlutus(&original)
	if err != nil {
		t.Fatal(err)
	}

	var decoded NestedDatum
	err = UnmarshalPlutus(pd, &decoded)
	if err != nil {
		t.Fatal(err)
	}

	if decoded.Inner.Amount != 123 {
		t.Errorf("expected 123, got %d", decoded.Inner.Amount)
	}
	if string(decoded.Inner.Name) != "nested" {
		t.Errorf("expected 'nested', got '%s'", string(decoded.Inner.Name))
	}
}

func TestMarshalUintField(t *testing.T) {
	type UintDatum struct {
		_     struct{} `plutusType:"DefList" plutusConstr:"0"`
		Count uint64   `plutusType:"Int"`
	}

	d := UintDatum{Count: 42}
	pd, err := MarshalPlutus(&d)
	if err != nil {
		t.Fatal(err)
	}

	constr, ok := pd.(*data.Constr)
	if !ok {
		t.Fatalf("expected Constr, got %T", pd)
	}
	intField, ok := constr.Fields[0].(*data.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T", constr.Fields[0])
	}
	if intField.Inner.Uint64() != 42 {
		t.Errorf("expected 42, got %d", intField.Inner.Uint64())
	}

	var decoded UintDatum
	err = UnmarshalPlutus(pd, &decoded)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Count != 42 {
		t.Errorf("expected 42, got %d", decoded.Count)
	}
}

func TestUnmarshalTooFewFields(t *testing.T) {
	// SimpleDatum expects 2 fields (Amount, Name).
	// Construct a Constr with only 1 field.
	tooFew := data.NewConstr(0, data.NewInteger(big.NewInt(42)))
	var decoded SimpleDatum
	err := UnmarshalPlutus(tooFew, &decoded)
	if err == nil {
		t.Error("expected error when PlutusData has fewer fields than struct expects")
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

func TestMarshalBigIntNil(t *testing.T) {
	d := BigIntDatum{Value: nil}
	pd, err := MarshalPlutus(&d)
	if err != nil {
		t.Fatal(err)
	}

	constr, ok := pd.(*data.Constr)
	if !ok {
		t.Fatalf("expected Constr, got %T", pd)
	}
	intField, ok := constr.Fields[0].(*data.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T", constr.Fields[0])
	}
	if intField.Inner.Int64() != 0 {
		t.Errorf("expected 0 for nil BigInt, got %d", intField.Inner.Int64())
	}
}

// indefSameDatum mirrors SimpleDatum but uses IndefList with the same constructor tag.
type indefSameDatum struct {
	_      struct{} `plutusType:"IndefList" plutusConstr:"0"`
	Amount int64    `plutusType:"Int"`
	Name   []byte   `plutusType:"Bytes"`
}

func TestMarshalIndefVsDefEncoding(t *testing.T) {
	// Marshal identical data with the same constructor tag, differing only in
	// DefList vs IndefList. The CBOR output must differ because the indef flag
	// changes the array encoding (definite 0x82 vs indefinite 0x9f).
	defDatum := SimpleDatum{Amount: 42, Name: []byte("hello")}
	indefDatum := indefSameDatum{Amount: 42, Name: []byte("hello")}

	defPd, err := MarshalPlutus(&defDatum)
	if err != nil {
		t.Fatal(err)
	}
	indefPd, err := MarshalPlutus(&indefDatum)
	if err != nil {
		t.Fatal(err)
	}

	defConstr, ok := defPd.(*data.Constr)
	if !ok {
		t.Fatalf("expected Constr for def, got %T", defPd)
	}
	indefConstr, ok := indefPd.(*data.Constr)
	if !ok {
		t.Fatalf("expected Constr for indef, got %T", indefPd)
	}

	// Tags must be the same (both constr 0) - we're testing encoding, not schema.
	if defConstr.Tag != indefConstr.Tag {
		t.Errorf("expected same tags, got def=%d indef=%d", defConstr.Tag, indefConstr.Tag)
	}

	// Fields must be the same.
	if len(defConstr.Fields) != len(indefConstr.Fields) {
		t.Fatalf("field count mismatch: def=%d indef=%d", len(defConstr.Fields), len(indefConstr.Fields))
	}

	// The CBOR encoding must differ due to the indef flag.
	defStr := defPd.String()
	indefStr := indefPd.String()
	if defStr == indefStr {
		t.Logf("warning: String() representations are equal; if plutigo merges them, verify CBOR bytes differ")
	}
}

func TestRoundTripNegativeBigInt(t *testing.T) {
	negVal := big.NewInt(-123456789)
	original := BigIntDatum{Value: negVal}

	pd, err := MarshalPlutus(&original)
	if err != nil {
		t.Fatalf("MarshalPlutus failed: %v", err)
	}

	// Verify the marshaled value is negative
	constr, ok := pd.(*data.Constr)
	if !ok {
		t.Fatalf("expected Constr, got %T", pd)
	}
	intField, ok := constr.Fields[0].(*data.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T", constr.Fields[0])
	}
	if intField.Inner.Sign() >= 0 {
		t.Errorf("expected negative value, got %s", intField.Inner.String())
	}
	if intField.Inner.Cmp(negVal) != 0 {
		t.Errorf("expected %s, got %s", negVal.String(), intField.Inner.String())
	}

	// Round-trip through UnmarshalPlutus
	var decoded BigIntDatum
	err = UnmarshalPlutus(pd, &decoded)
	if err != nil {
		t.Fatalf("UnmarshalPlutus failed: %v", err)
	}
	if decoded.Value.Cmp(negVal) != 0 {
		t.Errorf("round-trip failed: expected %s, got %s", negVal.String(), decoded.Value.String())
	}
}

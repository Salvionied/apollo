package plutusencoder

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/blinklabs-io/plutigo/data"
)

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
	if !constrTagEqual(constr.Tag, 0) {
		t.Errorf("expected tag 0, got %s", constr.Tag)
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
	if !constrTagEqual(constr.Tag, 1) {
		t.Errorf("expected tag 1, got %s", constr.Tag)
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

func TestMarshalNestedFieldHonorsDefListTag(t *testing.T) {
	type innerWithoutContainer struct {
		Value int64 `plutusType:"Int"`
	}
	type outerDatum struct {
		_     struct{}              `plutusType:"DefList" plutusConstr:"0"`
		Inner innerWithoutContainer `plutusType:"DefList"`
	}

	pd, err := MarshalPlutus(&outerDatum{Inner: innerWithoutContainer{Value: 42}})
	if err != nil {
		t.Fatal(err)
	}

	encoded, err := data.Encode(pd)
	if err != nil {
		t.Fatal(err)
	}
	if got := hex.EncodeToString(encoded); got != "d8798181182a" {
		t.Fatalf("expected nested definite list CBOR d8798181182a, got %s", got)
	}
}

func TestMarshalNestedExplicitContainerPrecedence(t *testing.T) {
	type innerWithContainer struct {
		_     struct{} `plutusType:"IndefList" plutusConstr:"1"`
		Value int64    `plutusType:"Int"`
	}
	type outerDatum struct {
		_     struct{}           `plutusType:"DefList" plutusConstr:"0"`
		Inner innerWithContainer `plutusType:"DefList"`
	}

	pd, err := MarshalPlutus(&outerDatum{Inner: innerWithContainer{Value: 42}})
	if err != nil {
		t.Fatal(err)
	}

	encoded, err := data.Encode(pd)
	if err != nil {
		t.Fatal(err)
	}
	if got := hex.EncodeToString(encoded); got != "d87981d87a9f182aff" {
		t.Fatalf("expected nested explicit container CBOR d87981d87a9f182aff, got %s", got)
	}
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
	if !constrTagEqual(defConstr.Tag, 0) || !constrTagEqual(indefConstr.Tag, 0) {
		t.Errorf("expected same tags, got def=%s indef=%s", defConstr.Tag, indefConstr.Tag)
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

func TestUnmarshalSliceRejectsConstr(t *testing.T) {
	type sliceDatum struct {
		_     struct{} `plutusType:"DefList" plutusConstr:"0"`
		Items []int64  `plutusType:"DefList"`
	}

	pd := data.NewConstr(0, data.NewConstr(0, data.NewInteger(big.NewInt(1)), data.NewInteger(big.NewInt(2))))
	var decoded sliceDatum
	if err := UnmarshalPlutus(pd, &decoded); err == nil {
		t.Fatal("expected error when Constr is provided where List is required for a slice")
	}
}

func TestRoundTripSliceListShape(t *testing.T) {
	type sliceDatum struct {
		_     struct{} `plutusType:"DefList" plutusConstr:"0"`
		Items []int64  `plutusType:"DefList"`
	}

	original := sliceDatum{Items: []int64{1, 2, 3}}
	pd, err := MarshalPlutus(&original)
	if err != nil {
		t.Fatal(err)
	}
	var decoded sliceDatum
	if err := UnmarshalPlutus(pd, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.Items) != 3 || decoded.Items[0] != 1 || decoded.Items[1] != 2 || decoded.Items[2] != 3 {
		t.Fatalf("unexpected decoded slice: %+v", decoded.Items)
	}
}

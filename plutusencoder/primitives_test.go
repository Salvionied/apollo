package plutusencoder

import (
	"math/big"
	"testing"

	"github.com/blinklabs-io/plutigo/data"
)

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

package plutusencoder

import (
	"math/big"
	"testing"

	"github.com/blinklabs-io/plutigo/data"
)

type omitMapDatum struct {
	_       struct{} `plutusType:"Map"`
	Name    string   `plutusType:"StringBytes" plutusKey:"name"`
	Empty   string   `plutusType:"StringBytes, omitempty" plutusKey:"empty"`
	Amount  int64    `plutusType:"Int,omitempty" plutusKey:"amount"`
	Payload []byte   `plutusType:"Bytes,omitempty" plutusKey:"payload"`
	Enabled bool     `plutusType:"Bool,omitempty" plutusKey:"enabled"`
	Big     *big.Int `plutusType:"BigInt,omitempty" plutusKey:"big"`
}

func TestMapOmitEmptyValues(t *testing.T) {
	original := omitMapDatum{Name: "order-1"}

	pd, err := MarshalPlutus(&original)
	if err != nil {
		t.Fatal(err)
	}
	mapData, ok := pd.(*data.Map)
	if !ok {
		t.Fatalf("expected Map, got %T", pd)
	}
	if len(mapData.Pairs) != 1 {
		t.Fatalf("expected only the required name key, got %d pairs", len(mapData.Pairs))
	}
	key, ok := mapData.Pairs[0][0].(*data.ByteString)
	if !ok {
		t.Fatalf("expected ByteString key, got %T", mapData.Pairs[0][0])
	}
	if string(key.Inner) != "name" {
		t.Fatalf("expected only name key, got %q", string(key.Inner))
	}

	var decoded omitMapDatum
	if err := UnmarshalPlutus(pd, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Name != "order-1" || decoded.Empty != "" || decoded.Amount != 0 || len(decoded.Payload) != 0 || decoded.Enabled || decoded.Big != nil {
		t.Errorf("unexpected decoded datum: %+v", decoded)
	}
}

func TestMapOmitEmptyKeepsNonZeroValues(t *testing.T) {
	original := omitMapDatum{
		Name:    "order-1",
		Empty:   "non-empty",
		Amount:  42,
		Payload: []byte{0x01},
		Enabled: true,
		Big:     big.NewInt(9),
	}

	pd, err := MarshalPlutus(&original)
	if err != nil {
		t.Fatal(err)
	}
	mapData, ok := pd.(*data.Map)
	if !ok {
		t.Fatalf("expected Map, got %T", pd)
	}
	if len(mapData.Pairs) != 6 {
		t.Fatalf("expected all 6 keys, got %d", len(mapData.Pairs))
	}

	var decoded omitMapDatum
	if err := UnmarshalPlutus(pd, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Empty != "non-empty" || decoded.Amount != 42 || len(decoded.Payload) != 1 || !decoded.Enabled || decoded.Big.Cmp(big.NewInt(9)) != 0 {
		t.Errorf("unexpected decoded datum: %+v", decoded)
	}
}

type trailingOmitList struct {
	_      struct{} `plutusType:"DefList" plutusConstr:"0"`
	Name   string   `plutusType:"StringBytes"`
	Amount int64    `plutusType:"Int,omitempty"`
}

func TestListOmitEmptyDropsTrailingZero(t *testing.T) {
	pd, err := MarshalPlutus(&trailingOmitList{Name: "order"})
	if err != nil {
		t.Fatal(err)
	}
	constr, ok := pd.(*data.Constr)
	if !ok || len(constr.Fields) != 1 {
		t.Fatalf("expected one field, got %#v", pd)
	}

	var decoded trailingOmitList
	if err := UnmarshalPlutus(pd, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Name != "order" || decoded.Amount != 0 {
		t.Fatalf("unexpected decoded value: %+v", decoded)
	}
}

func TestListOmitEmptyKeepsTrailingValue(t *testing.T) {
	original := trailingOmitList{Name: "order", Amount: 7}
	pd, err := MarshalPlutus(&original)
	if err != nil {
		t.Fatal(err)
	}
	constr, ok := pd.(*data.Constr)
	if !ok || len(constr.Fields) != 2 {
		t.Fatalf("expected two fields, got %#v", pd)
	}
	var decoded trailingOmitList
	if err := UnmarshalPlutus(pd, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded != original {
		t.Fatalf("round trip = %+v, want %+v", decoded, original)
	}
}

func TestListOmitEmptyHoleReturnsError(t *testing.T) {
	type holed struct {
		_      struct{} `plutusType:"DefList" plutusConstr:"0"`
		Amount int64    `plutusType:"Int,omitempty"`
		Name   string   `plutusType:"StringBytes"`
	}
	_, err := MarshalPlutus(&holed{Name: "order"})
	if err == nil {
		t.Fatal("expected error when omitempty would leave a hole")
	}
}

func TestIndefListOmitEmptyDropsTrailingZero(t *testing.T) {
	type trailing struct {
		_      struct{} `plutusType:"IndefList" plutusConstr:"0"`
		Name   string   `plutusType:"StringBytes"`
		Amount int64    `plutusType:"Int,omitempty"`
	}
	pd, err := MarshalPlutus(&trailing{Name: "order"})
	if err != nil {
		t.Fatal(err)
	}
	constr, ok := pd.(*data.Constr)
	if !ok || len(constr.Fields) != 1 {
		t.Fatalf("expected one field, got %#v", pd)
	}
	var decoded trailing
	if err := UnmarshalPlutus(pd, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Name != "order" || decoded.Amount != 0 {
		t.Fatalf("unexpected decoded value: %+v", decoded)
	}
}

func TestListWithoutOmitEmptyKeepsZeroFields(t *testing.T) {
	type required struct {
		_      struct{} `plutusType:"DefList" plutusConstr:"0"`
		Name   string   `plutusType:"StringBytes"`
		Amount int64    `plutusType:"Int"`
	}
	pd, err := MarshalPlutus(&required{})
	if err != nil {
		t.Fatal(err)
	}
	constr, ok := pd.(*data.Constr)
	if !ok || len(constr.Fields) != 2 {
		t.Fatalf("zero fields must stay on the wire, got %#v", pd)
	}
	var decoded required
	if err := UnmarshalPlutus(pd, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Name != "" || decoded.Amount != 0 {
		t.Fatalf("unexpected decoded value: %+v", decoded)
	}
}

func TestOmitEmptyWithLeadingCommaReturnsError(t *testing.T) {
	type invalidMapDatum struct {
		_      struct{} `plutusType:"Map"`
		Amount int64    `plutusType:",omitempty" plutusKey:"amount"`
	}

	_, err := MarshalPlutus(&invalidMapDatum{Amount: 0})
	if err == nil {
		t.Fatal("expected error for missing field type before omitempty")
	}
}

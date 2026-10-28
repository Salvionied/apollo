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

func TestMaybeOmitEmptyKeepsJustZero(t *testing.T) {
	zero := int64(0)
	type last struct {
		_    struct{} `plutusType:"DefList" plutusConstr:"0"`
		Name string   `plutusType:"StringBytes"`
		N    *int64   `plutusType:"Maybe,omitempty"`
	}
	pd, err := MarshalPlutus(&last{Name: "order", N: &zero})
	if err != nil {
		t.Fatal(err)
	}
	if len(pd.(*data.Constr).Fields) != 2 {
		t.Fatalf("Just 0 was dropped: %#v", pd)
	}
	var decoded last
	if err := UnmarshalPlutus(pd, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.N == nil || *decoded.N != 0 {
		t.Fatalf("decoded %#v", decoded.N)
	}

	type before struct {
		_    struct{} `plutusType:"DefList" plutusConstr:"0"`
		N    *int64   `plutusType:"Maybe,omitempty"`
		Name string   `plutusType:"StringBytes"`
	}
	pd, err = MarshalPlutus(&before{Name: "order", N: &zero})
	if err != nil {
		t.Fatal(err)
	}
	if len(pd.(*data.Constr).Fields) != 2 {
		t.Fatalf("Just 0 before a required field was treated as a hole: %#v", pd)
	}
}

type alwaysZero int64

func (alwaysZero) IsZero() bool { return true }

func TestOmitEmptyHonorsPointerIsZero(t *testing.T) {
	kept := int64(0)
	type row struct {
		_    struct{}    `plutusType:"DefList" plutusConstr:"0"`
		Name string      `plutusType:"StringBytes"`
		N    *int64      `plutusType:"Maybe,omitempty"`
		Z    *alwaysZero `plutusType:"Maybe,omitempty"`
	}
	z := alwaysZero(3)
	pd, err := MarshalPlutus(&row{Name: "order", N: &kept, Z: &z})
	if err != nil {
		t.Fatal(err)
	}
	if len(pd.(*data.Constr).Fields) != 2 {
		t.Fatalf("expected name and Just 0, got %d fields", len(pd.(*data.Constr).Fields))
	}
}

func TestListOmitEmptyHoleAndShortRequired(t *testing.T) {
	type hole struct {
		_      struct{} `plutusType:"DefList" plutusConstr:"0"`
		Amount int64    `plutusType:"Int,omitempty"`
		Name   string   `plutusType:"StringBytes"`
	}
	if _, err := MarshalPlutus(&hole{Name: "order"}); err == nil {
		t.Fatal("expected a hole error")
	}

	type required struct {
		_    struct{} `plutusType:"DefList" plutusConstr:"0"`
		Name string   `plutusType:"StringBytes"`
		Note string   `plutusType:"StringBytes"`
	}
	short := data.NewConstr(0, data.NewByteString([]byte("order")))
	var decoded required
	if err := UnmarshalPlutus(short, &decoded); err == nil {
		t.Fatal("expected a short list to fail")
	}

	extra := data.NewConstr(0,
		data.NewByteString([]byte("order")),
		data.NewByteString([]byte("note")),
		data.NewInteger(big.NewInt(9)),
	)
	if err := UnmarshalPlutus(extra, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Name != "order" || decoded.Note != "note" {
		t.Fatalf("decoded %+v", decoded)
	}
}

func TestReusedListAndMapClearMissingFields(t *testing.T) {
	type row struct {
		_      struct{} `plutusType:"DefList" plutusConstr:"0"`
		Name   string   `plutusType:"StringBytes"`
		Amount int64    `plutusType:"Int,omitempty"`
		Flag   *int64   `plutusType:"Maybe,omitempty"`
	}
	flag := int64(4)
	pd, err := MarshalPlutus(&row{Name: "order"})
	if err != nil {
		t.Fatal(err)
	}
	decoded := row{Amount: 7, Flag: &flag}
	if err := UnmarshalPlutus(pd, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Name != "order" || decoded.Amount != 0 || decoded.Flag != nil {
		t.Fatalf("list reuse: %+v", decoded)
	}

	type mapped struct {
		_      struct{} `plutusType:"Map"`
		Name   string   `plutusType:"StringBytes" plutusKey:"name"`
		Amount int64    `plutusType:"Int,omitempty" plutusKey:"amount"`
		Flag   *int64   `plutusType:"Maybe,omitempty" plutusKey:"flag"`
	}
	pd, err = MarshalPlutus(&mapped{Name: "order"})
	if err != nil {
		t.Fatal(err)
	}
	mapData := pd.(*data.Map)
	if len(mapData.Pairs) != 1 {
		t.Fatalf("nil Maybe was encoded, pairs %d", len(mapData.Pairs))
	}
	mappedDecoded := mapped{Amount: 7, Flag: &flag}
	if err := UnmarshalPlutus(pd, &mappedDecoded); err != nil {
		t.Fatal(err)
	}
	if mappedDecoded.Name != "order" || mappedDecoded.Amount != 0 || mappedDecoded.Flag != nil {
		t.Fatalf("map reuse: %+v", mappedDecoded)
	}
}

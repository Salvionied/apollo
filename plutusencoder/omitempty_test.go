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

func TestOmitEmptyOnListFieldReturnsError(t *testing.T) {
	type invalidListDatum struct {
		_      struct{} `plutusType:"DefList" plutusConstr:"0"`
		Amount int64    `plutusType:"Int,omitempty"`
	}

	_, err := MarshalPlutus(&invalidListDatum{Amount: 0})
	if err == nil {
		t.Fatal("expected schema-safety error for omitempty on positional list field")
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

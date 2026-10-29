package plutusencoder

import (
	"math/big"
	"testing"

	"github.com/blinklabs-io/plutigo/data"
)

type offChainData struct {
	Value string
}

type ignoredSecret string

func (s ignoredSecret) ToPlutusData() (data.PlutusData, error) {
	return data.NewByteString([]byte(s)), nil
}

func (s ignoredSecret) FromPlutusData(data.PlutusData, any) error { return nil }

type ignoreListDatum struct {
	_        struct{}     `plutusType:"DefList" plutusConstr:"1"`
	Amount   int64        `plutusType:"Int"`
	OffChain offChainData `plutusType:"Ignore"`
	Name     []byte       `plutusType:"Bytes"`
}

type ignoreMapDatum struct {
	_        struct{}     `plutusType:"Map"`
	Name     string       `plutusType:"StringBytes" plutusKey:"name"`
	OffChain offChainData `plutusType:"Ignore"`
	Value    int64        `plutusType:"Int" plutusKey:"value"`
}

func TestIgnoreListField(t *testing.T) {
	original := ignoreListDatum{
		Amount:   42,
		OffChain: offChainData{Value: "not on chain"},
		Name:     []byte("order"),
	}

	pd, err := MarshalPlutus(&original)
	if err != nil {
		t.Fatal(err)
	}
	constr, ok := pd.(*data.Constr)
	if !ok {
		t.Fatalf("expected Constr, got %T", pd)
	}
	if !constrTagEqual(constr.Tag, 1) {
		t.Fatalf("expected tag 1, got %d", constr.Tag)
	}
	if len(constr.Fields) != 2 {
		t.Fatalf("ignored field consumed a wire slot: expected 2 fields, got %d", len(constr.Fields))
	}

	var decoded ignoreListDatum
	if err := UnmarshalPlutus(pd, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Amount != 42 {
		t.Errorf("expected amount 42, got %d", decoded.Amount)
	}
	if string(decoded.Name) != "order" {
		t.Errorf("expected name 'order', got %q", string(decoded.Name))
	}
	if decoded.OffChain.Value != "" {
		t.Errorf("expected ignored field to remain zero, got %q", decoded.OffChain.Value)
	}
}

func TestIgnoreMapField(t *testing.T) {
	original := ignoreMapDatum{
		Name:     "order-1",
		OffChain: offChainData{Value: "not on chain"},
		Value:    99,
	}

	pd, err := MarshalPlutus(&original)
	if err != nil {
		t.Fatal(err)
	}
	mapData, ok := pd.(*data.Map)
	if !ok {
		t.Fatalf("expected Map, got %T", pd)
	}
	if len(mapData.Pairs) != 2 {
		t.Fatalf("ignored field became a map key: expected 2 pairs, got %d", len(mapData.Pairs))
	}
	for i, pair := range mapData.Pairs {
		key, ok := pair[0].(*data.ByteString)
		if !ok {
			t.Fatalf("pair %d key: expected ByteString, got %T", i, pair[0])
		}
		if string(key.Inner) == "OffChain" {
			t.Fatalf("ignored field became map key: %q", string(key.Inner))
		}
	}

	var decoded ignoreMapDatum
	if err := UnmarshalPlutus(pd, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Name != "order-1" || decoded.Value != 99 {
		t.Errorf("expected order-1/99, got %s/%d", decoded.Name, decoded.Value)
	}
	if decoded.OffChain.Value != "" {
		t.Errorf("expected ignored field to remain zero, got %q", decoded.OffChain.Value)
	}
}

func TestIgnoreListFieldDoesNotConsumeTrailingWireValue(t *testing.T) {
	pd := data.NewConstr(1,
		data.NewInteger(big.NewInt(42)),
		data.NewByteString([]byte("order")),
	)
	var decoded ignoreListDatum
	if err := UnmarshalPlutus(pd, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Amount != 42 || string(decoded.Name) != "order" || decoded.OffChain.Value != "" {
		t.Errorf("unexpected decoded datum: %+v", decoded)
	}
}

func TestIgnoreFieldsInSliceMapEntries(t *testing.T) {
	type leadingSecretEntry struct {
		Secret ignoredSecret `plutusType:"Ignore"`
		Key    string        `plutusType:"StringBytes"`
		Value  int64         `plutusType:"Int"`
	}
	type trailingSecretEntry struct {
		Key    string        `plutusType:"StringBytes"`
		Secret ignoredSecret `plutusType:"Ignore"`
		Value  int64         `plutusType:"Int"`
	}
	type leadingDatum struct {
		_       struct{}             `plutusType:"DefList" plutusConstr:"0"`
		Entries []leadingSecretEntry `plutusType:"Map"`
	}
	type trailingDatum struct {
		_       struct{}              `plutusType:"DefList" plutusConstr:"0"`
		Entries []trailingSecretEntry `plutusType:"Map"`
	}
	for _, tc := range []struct {
		name  string
		value any
	}{
		{name: "before key", value: &leadingDatum{Entries: []leadingSecretEntry{{Secret: "private", Key: "name", Value: 42}}}},
		{name: "after key", value: &trailingDatum{Entries: []trailingSecretEntry{{Key: "name", Secret: "private", Value: 42}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pd, err := MarshalPlutus(tc.value)
			if err != nil {
				t.Fatal(err)
			}
			constr, ok := pd.(*data.Constr)
			if !ok || len(constr.Fields) != 1 {
				t.Fatalf("expected constructor with Map, got %T", pd)
			}
			mapData, ok := constr.Fields[0].(*data.Map)
			if !ok || len(mapData.Pairs) != 1 {
				t.Fatalf("expected one Map entry, got %T", constr.Fields[0])
			}
			key, ok := mapData.Pairs[0][0].(*data.ByteString)
			if !ok || string(key.Inner) != "name" {
				t.Fatalf("Map key = %v, want name", mapData.Pairs[0][0])
			}
			value, ok := mapData.Pairs[0][1].(*data.Integer)
			if !ok || value.Inner.Int64() != 42 {
				t.Fatalf("Map value = %v, want 42", mapData.Pairs[0][1])
			}
			switch tc.name {
			case "before key":
				var decoded leadingDatum
				if err := UnmarshalPlutus(pd, &decoded); err != nil || len(decoded.Entries) != 1 || decoded.Entries[0].Key != "name" || decoded.Entries[0].Value != 42 || decoded.Entries[0].Secret != "" {
					t.Fatalf("decoded = %+v, error = %v", decoded, err)
				}
			case "after key":
				var decoded trailingDatum
				if err := UnmarshalPlutus(pd, &decoded); err != nil || len(decoded.Entries) != 1 || decoded.Entries[0].Key != "name" || decoded.Entries[0].Value != 42 || decoded.Entries[0].Secret != "" {
					t.Fatalf("decoded = %+v, error = %v", decoded, err)
				}
			}
		})
	}
}

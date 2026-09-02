package plutusencoder

import (
	"math/big"
	"testing"

	"github.com/blinklabs-io/plutigo/data"
)

type offChainData struct {
	Value string
}

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
		t.Fatalf("expected tag 1, got %s", constr.Tag)
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

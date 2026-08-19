package plutusencoder

import (
	"math/big"
	"strings"
	"testing"

	"github.com/blinklabs-io/plutigo/data"
)

func TestMarshalUnknownContainerTag(t *testing.T) {
	type typoContainerDatum struct {
		_      struct{} `plutusType:"DefLists"`
		Amount int64    `plutusType:"Int"`
	}

	_, err := MarshalPlutus(&typoContainerDatum{Amount: 42})
	if err == nil {
		t.Fatal("expected error for unknown container tag")
	}
	if got := err.Error(); !strings.Contains(got, "plutusType") || !strings.Contains(got, "DefLists") {
		t.Fatalf("expected descriptive unknown-container error, got: %s", got)
	}
}

func TestUnmarshalUnknownContainerTag(t *testing.T) {
	type typoContainerDatum struct {
		_      struct{} `plutusType:"DefLists"`
		Amount int64    `plutusType:"Int"`
	}

	pd := data.NewList(data.NewInteger(new(big.Int).SetInt64(42)))
	var decoded typoContainerDatum
	err := UnmarshalPlutus(pd, &decoded)
	if err == nil {
		t.Fatal("expected error for unknown container tag")
	}
	if got := err.Error(); !strings.Contains(got, "plutusType") || !strings.Contains(got, "DefLists") {
		t.Fatalf("expected descriptive unknown-container error, got: %s", got)
	}
}

func TestMarshalUnknownFieldTagOption(t *testing.T) {
	type unknownOptionDatum struct {
		_      struct{} `plutusType:"DefList" plutusConstr:"0"`
		Amount int64    `plutusType:"Int, unknownOption"`
	}

	_, err := MarshalPlutus(&unknownOptionDatum{Amount: 42})
	if err == nil {
		t.Fatal("expected error for unknown field tag option")
	}
	if got := err.Error(); !strings.Contains(got, "plutusType") || !strings.Contains(got, "unknownOption") {
		t.Fatalf("expected descriptive unknown-option error, got: %s", got)
	}
}

func TestMarshalEmptyContainerTagStillDefaultsToIndefList(t *testing.T) {
	type untaggedContainerDatum struct {
		Amount int64 `plutusType:"Int"`
	}

	pd, err := MarshalPlutus(&untaggedContainerDatum{Amount: 42})
	if err != nil {
		t.Fatal(err)
	}
	list, ok := pd.(*data.List)
	if !ok {
		t.Fatalf("expected List, got %T", pd)
	}
	if len(list.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(list.Items))
	}
}

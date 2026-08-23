package plutusencoder_test

import (
	"encoding/hex"
	"fmt"

	"github.com/blinklabs-io/plutigo/data"

	"github.com/Salvionied/apollo/v2/plutusencoder"
)

// listDatum is a constructor-wrapped definite-length list. Fields are
// positional, so their declaration order is the encoded order.
type listDatum struct {
	_      struct{} `plutusType:"DefList" plutusConstr:"1"`
	Owner  []byte   `plutusType:"Bytes"`
	Amount int64    `plutusType:"Int"`
}

// mapDatum is keyed by field name. plutusKey renames a key, omitempty drops an
// empty value, and Ignore keeps a field out of the encoding entirely.
type mapDatum struct {
	_        struct{} `plutusType:"Map"`
	Owner    string   `plutusType:"StringBytes"           plutusKey:"owner"`
	Memo     string   `plutusType:"StringBytes,omitempty"`
	Internal string   `plutusType:"Ignore"`
}

func encodeHex(pd data.PlutusData) string {
	encoded, err := data.Encode(pd)
	if err != nil {
		panic(err)
	}
	return hex.EncodeToString(encoded)
}

// ExampleMarshalPlutus encodes a positional list datum wrapped in a
// constructor.
func ExampleMarshalPlutus() {
	pd, err := plutusencoder.MarshalPlutus(listDatum{
		Owner:  []byte{0xde, 0xad},
		Amount: 42,
	})
	if err != nil {
		panic(err)
	}

	fmt.Println(encodeHex(pd))
	// Output: d87a8242dead182a
}

// ExampleMarshalPlutus_map shows plutusKey, omitempty and Ignore. Memo is
// empty and Internal is ignored, so only the renamed owner key is encoded.
func ExampleMarshalPlutus_map() {
	pd, err := plutusencoder.MarshalPlutus(mapDatum{
		Owner:    "alice",
		Internal: "not encoded",
	})
	if err != nil {
		panic(err)
	}

	fmt.Println(encodeHex(pd))
	// Output: a1456f776e657245616c696365
}

// ExampleUnmarshalPlutus decodes back into the struct the value came from.
func ExampleUnmarshalPlutus() {
	pd, err := plutusencoder.MarshalPlutus(listDatum{
		Owner:  []byte{0xde, 0xad},
		Amount: 42,
	})
	if err != nil {
		panic(err)
	}

	var decoded listDatum
	if err := plutusencoder.UnmarshalPlutus(pd, &decoded); err != nil {
		panic(err)
	}

	fmt.Printf("%x %d\n", decoded.Owner, decoded.Amount)
	// Output: dead 42
}

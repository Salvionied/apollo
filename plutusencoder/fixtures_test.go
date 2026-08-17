package plutusencoder

import (
	"encoding/hex"
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

// indefSameDatum mirrors SimpleDatum but uses IndefList with the same constructor tag.
type indefSameDatum struct {
	_      struct{} `plutusType:"IndefList" plutusConstr:"0"`
	Amount int64    `plutusType:"Int"`
	Name   []byte   `plutusType:"Bytes"`
}

// optionalMapDatum mirrors MapDatum but marks Value as optional.
type optionalMapDatum struct {
	_     struct{} `plutusType:"Map"`
	Name  string   `plutusType:"StringBytes" plutusKey:"name"`
	Value int64    `plutusType:"Int" plutusKey:"value" plutusOptional:"true"`
}

func TestMarshalGoldenCbor(t *testing.T) {
	// Fixed wire bytes protect existing Apollo v2 encodings while this package's
	// implementation is split across smaller files.
	tests := []struct {
		name  string
		datum any
		want  string
	}{
		{
			name:  "simple definite constructor",
			datum: SimpleDatum{Amount: 42, Name: []byte("hello")},
			want:  "d87982182a4568656c6c6f",
		},
		{
			name:  "indefinite constructor",
			datum: IndefDatum{Pkh: []byte{0xaa, 0xbb}, Amount: 100},
			want:  "d87a9f42aabb1864ff",
		},
		{
			name:  "map datum",
			datum: MapDatum{Name: "test", Value: 99},
			want:  "a2446e616d6544746573744576616c75651863",
		},
		{
			name:  "bool true",
			datum: BoolDatum{Active: true},
			want:  "d87981d87a80",
		},
		{
			name:  "negative big integer",
			datum: BigIntDatum{Value: big.NewInt(-123456789)},
			want:  "d879813a075bcd14",
		},
		{
			name:  "hex bytes",
			datum: HexDatum{Hash: "aabbccdd"},
			want:  "d8798144aabbccdd",
		},
		{
			name:  "nested definite list",
			datum: NestedDatum{Inner: SimpleDatum{Amount: 123, Name: []byte("nested")}},
			want:  "d87981d87982187b466e6573746564",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pd, err := MarshalPlutus(tt.datum)
			if err != nil {
				t.Fatal(err)
			}

			encoded, err := data.Encode(pd)
			if err != nil {
				t.Fatal(err)
			}
			got := hex.EncodeToString(encoded)
			if got != tt.want {
				t.Fatalf("CBOR mismatch: expected %s, got %s", tt.want, got)
			}

			var roundTrip data.PlutusDataWrapper
			if err := roundTrip.UnmarshalCBOR(encoded); err != nil {
				t.Fatal(err)
			}
			if !roundTrip.Data.Equal(pd) {
				t.Fatalf("round trip changed data: expected %s, got %s", pd.String(), roundTrip.Data.String())
			}
		})
	}
}

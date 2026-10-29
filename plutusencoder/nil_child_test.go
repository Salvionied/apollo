package plutusencoder

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/plutigo/data"
)

func TestTypedNilChildrenAreRejected(t *testing.T) {
	hash := bytes.Repeat([]byte{0x11}, common.AddressHashSize)
	pay := data.NewConstr(0, data.NewByteString(hash))
	pointer := data.NewConstr(1, (*data.Integer)(nil), data.NewInteger(big.NewInt(0)), data.NewInteger(big.NewInt(0)))
	cases := []struct {
		name string
		dst  any
		pd   data.PlutusData
	}{
		{
			name: "integer",
			dst:  &SimpleDatum{},
			pd:   data.NewConstr(0, (*data.Integer)(nil), data.NewByteString([]byte("n"))),
		},
		{
			name: "bytes",
			dst:  &SimpleDatum{},
			pd:   data.NewConstr(0, data.NewInteger(big.NewInt(1)), (*data.ByteString)(nil)),
		},
		{
			name: "bool",
			dst:  &BoolDatum{},
			pd:   data.NewConstr(0, (*data.Constr)(nil)),
		},
		{
			name: "list element",
			dst: &struct {
				_     struct{} `plutusType:"DefList" plutusConstr:"0"`
				Items []int64  `plutusType:"DefList"`
			}{},
			pd: data.NewConstr(0, data.NewList((*data.Integer)(nil))),
		},
		{
			name: "map key",
			dst:  &MapDatum{},
			pd:   data.NewMap([][2]data.PlutusData{{(*data.ByteString)(nil), data.NewByteString([]byte("a"))}}),
		},
		{
			name: "address hash",
			dst:  &addressDatum{},
			pd:   data.NewConstr(0, data.NewConstr(0, data.NewConstr(0, (*data.ByteString)(nil)), data.NewConstr(1))),
		},
		{
			name: "pointer slot",
			dst:  &addressDatum{},
			pd:   data.NewConstr(0, data.NewConstr(0, pay, data.NewConstr(0, pointer))),
		},
	}
	for _, tc := range cases {
		err := UnmarshalPlutus(tc.pd, tc.dst)
		if err == nil {
			t.Errorf("%s: typed nil decoded", tc.name)
		}
	}
}

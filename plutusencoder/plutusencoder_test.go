package plutusencoder_test

import (
	"encoding/hex"
	"testing"

	"github.com/Salvionied/apollo/plutusencoder"
	"github.com/Salvionied/cbor/v2"
)

type BuyerDatum struct {
	_      struct{} `plutusType:"DefList" plutusConstr:"2"`
	Pkh    []byte   `plutusType:"Bytes"`
	Amount int64    `plutusType:"Int"`
	Skh    []byte   `plutusType:"Bytes"`
}
type Datum struct {
	_      struct{} `plutusType:"IndefList" plutusConstr:"1"`
	Pkh    []byte   `plutusType:"Bytes"`
	Amount int64    `plutusType:"Int"`
	Buyer  BuyerDatum
}

func TestPlutusMarshal(t *testing.T) {
	d := Datum{
		Pkh:    []byte{0x01, 0x02, 0x03, 0x04},
		Amount: 1000000,
		Buyer: BuyerDatum{
			Pkh:    []byte{0x01, 0x02, 0x03, 0x04},
			Amount: 1000000,
			Skh:    []byte{0x01, 0x02, 0x03, 0x04},
		},
	}
	marshaled, err := plutusencoder.MarshalPlutus(d)
	if err != nil {
		t.Error(err)
	}
	encoded, err := cbor.Marshal(marshaled)
	if hex.EncodeToString(encoded) != "d87a9f44010203041a000f4240d87b8344010203041a000f42404401020304ff" {
		t.Error("encoding error")
	}
}

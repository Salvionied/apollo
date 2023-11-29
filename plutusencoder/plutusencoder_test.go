package plutusencoder_test

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/Salvionied/apollo/plutusencoder"
	"github.com/Salvionied/apollo/serialization/PlutusData"
	"github.com/Salvionied/cbor/v2"
)

type BuyerDatum struct {
	_      struct{} `plutusType:"DefList" plutusConstr:"2"`
	Pkh    []byte   `plutusType:"Bytes"`
	Amount int64    `plutusType:"Int"`
	Skh    []byte   `plutusType:"Bytes"`
}
type Datum struct {
	_          struct{} `plutusType:"IndefList" plutusConstr:"1"`
	Pkh        []byte   `plutusType:"Bytes"`
	RandomText string   `plutusType:"StringBytes"`
	Amount     int64    `plutusType:"Int"`
	Buyer      BuyerDatum
}

func TestPlutusMarshal(t *testing.T) {
	d := Datum{
		Pkh:        []byte{0x01, 0x02, 0x03, 0x04},
		Amount:     1000000,
		RandomText: "Hello World",
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
	if hex.EncodeToString(encoded) != "d87a9f44010203044b48656c6c6f20576f726c641a000f4240d87b8344010203041a000f42404401020304ff" {
		t.Error("encoding error")
	}
}

func TestPlutusUnmarshal(t *testing.T) {
	p := "d87a9f44010203044b48656c6c6f20576f726c641a000f4240d87b8344010203041a000f42404401020304ff"
	decoded, err := hex.DecodeString(p)
	if err != nil {
		t.Error(err)
	}
	pd := PlutusData.PlutusData{}
	err = cbor.Unmarshal(decoded, &pd)
	if err != nil {
		t.Error(err)
	}
	d := new(Datum)
	err = plutusencoder.UnmarshalPlutus(&pd, d)
	if err != nil {
		t.Error(err)
	}
	if d.Amount != 1000000 {
		t.Error("amount not correct")
	}
	if d.RandomText != "Hello World" {
		t.Error("random text not correct")
	}
	if fmt.Sprintf("%x", d.Pkh) != "01020304" {
		t.Error("pkh not correct")
	}
	if fmt.Sprintf("%x", d.Buyer.Pkh) != "01020304" {
		t.Error("buyer pkh not correct")
	}
	if fmt.Sprintf("%x", d.Buyer.Skh) != "01020304" {
		t.Error("buyer skh not correct")
	}
	if d.Buyer.Amount != 1000000 {
		t.Error("buyer amount not correct")
	}

}

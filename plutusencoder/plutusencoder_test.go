package plutusencoder_test

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/Salvionied/apollo/plutusencoder"
	"github.com/Salvionied/apollo/serialization/Address"
	"github.com/Salvionied/apollo/serialization/PlutusData"
	"github.com/Salvionied/cbor/v2"
)

type TestAddress struct {
	_       struct{}        `plutusType:"DefList" plutusConstr:"2"`
	Address Address.Address `plutusType:"Address"`
}

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

type NestedList struct {
	_      struct{}     `plutusType:"IndefList" plutusConstr:"1"`
	Pkh    []byte       `plutusType:"Bytes"`
	Amount int64        `plutusType:"Int"`
	Buyers []BuyerDatum `plutusType:"IndefList"`
}

func GetDatum() PlutusData.PlutusData {
	x := PlutusData.PlutusData{
		PlutusDataType: PlutusData.PlutusArray,
		TagNr:          122,
		Value: PlutusData.PlutusIndefArray{
			PlutusData.PlutusData{
				PlutusDataType: PlutusData.PlutusBytes,
				Value:          []byte{0x01, 0x02, 0x03, 0x04},
			},
			PlutusData.PlutusData{
				PlutusDataType: PlutusData.PlutusBytes,
				Value:          []byte("Hello World"),
			},
			PlutusData.PlutusData{
				PlutusDataType: PlutusData.PlutusInt,
				Value:          uint64(1000000),
			},
			PlutusData.PlutusData{
				PlutusDataType: PlutusData.PlutusArray,
				TagNr:          2,
				Value: PlutusData.PlutusDefArray{
					PlutusData.PlutusData{
						PlutusDataType: PlutusData.PlutusBytes,
						Value:          []byte{0x01, 0x02, 0x03, 0x04},
					},
					PlutusData.PlutusData{
						PlutusDataType: PlutusData.PlutusInt,
						Value:          uint64(1000000),
					},
					PlutusData.PlutusData{
						PlutusDataType: PlutusData.PlutusBytes,
						Value:          []byte{0x01, 0x02, 0x03, 0x04},
					},
				},
			},
		},
	}
	return x
}

func TestNestedListMarshal(t *testing.T) {
	d := NestedList{
		Pkh:    []byte{0x01, 0x02, 0x03, 0x04},
		Amount: 1000000,
		Buyers: []BuyerDatum{
			BuyerDatum{
				Pkh:    []byte{0x01, 0x02, 0x03, 0x04},
				Amount: 1000000,
				Skh:    []byte{0x01, 0x02, 0x03, 0x04},
			},
			BuyerDatum{
				Pkh:    []byte{0x01, 0x02, 0x03, 0x04},
				Amount: 1000000,
				Skh:    []byte{0x01, 0x02, 0x03, 0x04},
			},
		},
	}
	marshaled, err := plutusencoder.MarshalPlutus(d)
	if err != nil {
		t.Error(err)
	}
	encoded, err := cbor.Marshal(marshaled)
	fmt.Println(hex.EncodeToString(encoded))
	if hex.EncodeToString(encoded) != "d87a9f44010203041a000f42409fd87b8344010203041a000f42404401020304d87b8344010203041a000f42404401020304ffff" {
		t.Error("encoding error")
	}
}

func TestNestedListUnmarshal(t *testing.T) {
	p := "d87a9f44010203041a000f42409fd87b8344010203041a000f42404401020304d87b8344010203041a000f42404401020304ffff"
	decoded, err := hex.DecodeString(p)
	if err != nil {
		t.Error(err)
	}
	pd := PlutusData.PlutusData{}
	err = cbor.Unmarshal(decoded, &pd)
	if err != nil {
		t.Error(err)
	}
	d := new(NestedList)
	err = plutusencoder.UnmarshalPlutus(&pd, d, 1)
	if err != nil {
		t.Error(err)
	}
	if d.Amount != 1000000 {
		t.Error("amount not correct")
	}
	if fmt.Sprintf("%x", d.Pkh) != "01020304" {
		t.Error("pkh not correct")
	}
	if fmt.Sprintf("%x", d.Buyers[0].Pkh) != "01020304" {
		t.Error("buyer pkh not correct")
	}
	if fmt.Sprintf("%x", d.Buyers[0].Skh) != "01020304" {
		t.Error("buyer skh not correct")
	}
	if d.Buyers[0].Amount != 1000000 {
		t.Error("buyer amount not correct")
	}
	if fmt.Sprintf("%x", d.Buyers[1].Pkh) != "01020304" {
		t.Error("buyer pkh not correct")
	}
	if fmt.Sprintf("%x", d.Buyers[1].Skh) != "01020304" {
		t.Error("buyer skh not correct")
	}
	if d.Buyers[1].Amount != 1000000 {
		t.Error("buyer amount not correct")
	}
	fmt.Println(d)
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
	err = plutusencoder.UnmarshalPlutus(&pd, d, 1)
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

func TestPDAddressesStruct(t *testing.T) {
	address := "addr1qxajla3qcrwckzkur8n0lt02rg2sepw3kgkstckmzrz4ccfm3j9pqrqkea3tns46e3qy2w42vl8dvvue8u45amzm3rjqvv2nxh"
	decoded_addr, _ := Address.DecodeAddress(address)

	d := TestAddress{
		Address: decoded_addr,
	}
	marshaled, err := plutusencoder.MarshalPlutus(d)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(marshaled)
	encoded, err := cbor.Marshal(marshaled)
	if hex.EncodeToString(encoded) != "d87b81d8799fd879581cbb2ff620c0dd8b0adc19e6ffadea1a150c85d1b22d05e2db10c55c61d8799fd8799fd879581c3b8c8a100c16cf62b9c2bacc40453aaa67ced633993f2b4eec5b88e4ffffff" {
		t.Error(hex.EncodeToString(encoded))
	}
}

func TestUnmarshalPDAddressesStruct(t *testing.T) {
	decoded, _ := hex.DecodeString("d87b81d8799fd879581cbb2ff620c0dd8b0adc19e6ffadea1a150c85d1b22d05e2db10c55c61d8799fd8799fd879581c3b8c8a100c16cf62b9c2bacc40453aaa67ced633993f2b4eec5b88e4ffffff")
	pd := PlutusData.PlutusData{}
	err := cbor.Unmarshal(decoded, &pd)
	if err != nil {
		t.Error(err)
	}
	d := new(TestAddress)
	err = plutusencoder.UnmarshalPlutus(&pd, d, 1)
	if err != nil {
		t.Error(err)
	}
	if d.Address.String() != "addr1qxajla3qcrwckzkur8n0lt02rg2sepw3kgkstckmzrz4ccfm3j9pqrqkea3tns46e3qy2w42vl8dvvue8u45amzm3rjqvv2nxh" {
		t.Error(d.Address.String())
	}
}

func TestPDAddress(t *testing.T) {
	addressKEY_KEY := "addr1qxajla3qcrwckzkur8n0lt02rg2sepw3kgkstckmzrz4ccfm3j9pqrqkea3tns46e3qy2w42vl8dvvue8u45amzm3rjqvv2nxh"
	decoded_addr, _ := Address.DecodeAddress(addressKEY_KEY)

	pd, err := plutusencoder.GetAddressPlutusData(decoded_addr)
	if err != nil {
		t.Error(err)
	}
	encoded, _ := cbor.Marshal(pd)
	if hex.EncodeToString(encoded) != "d8799fd879581cbb2ff620c0dd8b0adc19e6ffadea1a150c85d1b22d05e2db10c55c61d8799fd8799fd879581c3b8c8a100c16cf62b9c2bacc40453aaa67ced633993f2b4eec5b88e4ffffff" {
		t.Error(hex.EncodeToString(encoded))
	}
	addressSCRIPT_KEY := "addr1z99tz7hungv6furtdl3zn72sree86wtghlcr4jc637r2eadkp2avt5gp297dnxhxcmy6kkptepsr5pa409qa7gf8stzs0706a3"
	decoded_addr, _ = Address.DecodeAddress(addressSCRIPT_KEY)

	pd, err = plutusencoder.GetAddressPlutusData(decoded_addr)
	if err != nil {
		t.Error(err)
	}
	encoded, _ = cbor.Marshal(pd)
	if hex.EncodeToString(encoded) != "d8799fd87a581c4ab17afc9a19a4f06b6fe229f9501e727d3968bff03acb1a8f86acf5d8799fd8799fd879581cb60abac5d101517cd99ae6c6c9ab582bc8603a07b57941df212782c5ffffff" {
		t.Error(hex.EncodeToString(encoded))
	}
	addressSCRIPT_NONE := "addr1w9hvftxrlw74wzk6vf0jfyp8wl8vt4arf8aq70rm4paselc46ptfq"
	decoded_addr, _ = Address.DecodeAddress(addressSCRIPT_NONE)
	pd, err = plutusencoder.GetAddressPlutusData(decoded_addr)
	if err != nil {
		t.Error(err)
	}
	encoded, _ = cbor.Marshal(pd)
	if hex.EncodeToString(encoded) != "d8799fd87a581c6ec4acc3fbbd570ada625f24902777cec5d7a349fa0f3c7ba87b0cffd87a9fffff" {
		t.Error(hex.EncodeToString(encoded))
	}

}

func TestDecodeAddressStruct(t *testing.T) {
	decoded_addr, _ := hex.DecodeString("d8799fd87a581c6ec4acc3fbbd570ada625f24902777cec5d7a349fa0f3c7ba87b0cffd87a9fffff")
	var pd PlutusData.PlutusData
	err := cbor.Unmarshal(decoded_addr, &pd)
	if err != nil {
		t.Error(err)
	}
	address := plutusencoder.DecodePlutusAddress(pd, 0b0001)
	if address.String() != "addr1w9hvftxrlw74wzk6vf0jfyp8wl8vt4arf8aq70rm4paselc46ptfq" {
		t.Error(address)
	}
	decoded_addr, _ = hex.DecodeString("d8799fd87a581c4ab17afc9a19a4f06b6fe229f9501e727d3968bff03acb1a8f86acf5d8799fd8799fd879581cb60abac5d101517cd99ae6c6c9ab582bc8603a07b57941df212782c5ffffff")
	err = cbor.Unmarshal(decoded_addr, &pd)
	if err != nil {
		t.Error(err)
	}
	address = plutusencoder.DecodePlutusAddress(pd, 0b0001)
	if address.String() != "addr1z99tz7hungv6furtdl3zn72sree86wtghlcr4jc637r2eadkp2avt5gp297dnxhxcmy6kkptepsr5pa409qa7gf8stzs0706a3" {
		t.Error(address)
	}

	decoded_addr, _ = hex.DecodeString("d8799fd879581cbb2ff620c0dd8b0adc19e6ffadea1a150c85d1b22d05e2db10c55c61d8799fd8799fd879581c3b8c8a100c16cf62b9c2bacc40453aaa67ced633993f2b4eec5b88e4ffffff")
	err = cbor.Unmarshal(decoded_addr, &pd)
	if err != nil {
		t.Error(err)
	}
	address = plutusencoder.DecodePlutusAddress(pd, 0b0001)
	if address.String() != "addr1qxajla3qcrwckzkur8n0lt02rg2sepw3kgkstckmzrz4ccfm3j9pqrqkea3tns46e3qy2w42vl8dvvue8u45amzm3rjqvv2nxh" {
		t.Error(address)
	}

}

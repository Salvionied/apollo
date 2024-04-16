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
	if hex.EncodeToString(encoded) != "d87b81d8799fd8799f581cbb2ff620c0dd8b0adc19e6ffadea1a150c85d1b22d05e2db10c55c61ffd8799fd8799fd8799f581c3b8c8a100c16cf62b9c2bacc40453aaa67ced633993f2b4eec5b88e4ffffffff" {
		t.Error(hex.EncodeToString(encoded))
	}
}

func TestUnmarshalPDAddressesStruct(t *testing.T) {
	decoded, _ := hex.DecodeString("d87b81d8799fd8799f581cbb2ff620c0dd8b0adc19e6ffadea1a150c85d1b22d05e2db10c55c61ffd8799fd8799fd8799f581c3b8c8a100c16cf62b9c2bacc40453aaa67ced633993f2b4eec5b88e4ffffffff")
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
	if hex.EncodeToString(encoded) != "d8799fd8799f581cbb2ff620c0dd8b0adc19e6ffadea1a150c85d1b22d05e2db10c55c61ffd8799fd8799fd8799f581c3b8c8a100c16cf62b9c2bacc40453aaa67ced633993f2b4eec5b88e4ffffffff" {
		t.Error(hex.EncodeToString(encoded))
	}
	addressSCRIPT_KEY := "addr1z99tz7hungv6furtdl3zn72sree86wtghlcr4jc637r2eadkp2avt5gp297dnxhxcmy6kkptepsr5pa409qa7gf8stzs0706a3"
	decoded_addr, _ = Address.DecodeAddress(addressSCRIPT_KEY)

	pd, err = plutusencoder.GetAddressPlutusData(decoded_addr)
	if err != nil {
		t.Error(err)
	}
	encoded, _ = cbor.Marshal(pd)
	if hex.EncodeToString(encoded) != "d8799fd87a9f581c4ab17afc9a19a4f06b6fe229f9501e727d3968bff03acb1a8f86acf5ffd8799fd8799fd8799f581cb60abac5d101517cd99ae6c6c9ab582bc8603a07b57941df212782c5ffffffff" {
		t.Error(hex.EncodeToString(encoded))
	}
	addressSCRIPT_NONE := "addr1w9hvftxrlw74wzk6vf0jfyp8wl8vt4arf8aq70rm4paselc46ptfq"
	decoded_addr, _ = Address.DecodeAddress(addressSCRIPT_NONE)
	pd, err = plutusencoder.GetAddressPlutusData(decoded_addr)
	if err != nil {
		t.Error(err)
	}
	encoded, _ = cbor.Marshal(pd)
	if hex.EncodeToString(encoded) != "d8799fd87a9f581c6ec4acc3fbbd570ada625f24902777cec5d7a349fa0f3c7ba87b0cffffd87a9fffff" {
		t.Error(hex.EncodeToString(encoded))
	}

	addressKEY_NONE := "addr1v8qke3rhzmkk6ppn2t746t9ftux9h6aywke60k8zanc8lugs28jvm"
	decoded_addr, _ = Address.DecodeAddress(addressKEY_NONE)
	pd, err = plutusencoder.GetAddressPlutusData(decoded_addr)
	if err != nil {
		t.Error(err)
	}
	encoded, _ = cbor.Marshal(pd)
	if hex.EncodeToString(encoded) != "d8799fd8799f581cc16cc47716ed6d043352fd5d2ca95f0c5beba475b3a7d8e2ecf07ff1ffd87a9fffff" {
		t.Error(hex.EncodeToString(encoded))
	}

}

func TestDecodeAddressStruct(t *testing.T) {
	decoded_addr, _ := hex.DecodeString("d8799fd87a9f581c6ec4acc3fbbd570ada625f24902777cec5d7a349fa0f3c7ba87b0cffffd87a9fffff")
	var pd PlutusData.PlutusData
	err := cbor.Unmarshal(decoded_addr, &pd)
	if err != nil {
		t.Error(err)
	}
	address, _ := plutusencoder.DecodePlutusAddress(pd, 0b0001)
	if address.String() != "addr1w9hvftxrlw74wzk6vf0jfyp8wl8vt4arf8aq70rm4paselc46ptfq" {
		t.Error(address, "expected", "addr1w9hvftxrlw74wzk6vf0jfyp8wl8vt4arf8aq70rm4paselc46ptfq")
	}
	decoded_addr, _ = hex.DecodeString("d8799fd87a9f581c4ab17afc9a19a4f06b6fe229f9501e727d3968bff03acb1a8f86acf5ffd8799fd8799fd8799f581cb60abac5d101517cd99ae6c6c9ab582bc8603a07b57941df212782c5ffffffff")
	err = cbor.Unmarshal(decoded_addr, &pd)
	if err != nil {
		t.Error(err)
	}
	address, _ = plutusencoder.DecodePlutusAddress(pd, 0b0001)
	if address.String() != "addr1z99tz7hungv6furtdl3zn72sree86wtghlcr4jc637r2eadkp2avt5gp297dnxhxcmy6kkptepsr5pa409qa7gf8stzs0706a3" {
		t.Error(address, "expected", "addr1z99tz7hungv6furtdl3zn72sree86wtghlcr4jc637r2eadkp2avt5gp297dnxhxcmy6kkptepsr5pa409qa7gf8stzs0706a3")
	}

	decoded_addr, _ = hex.DecodeString("d8799fd8799f581cbb2ff620c0dd8b0adc19e6ffadea1a150c85d1b22d05e2db10c55c61ffd8799fd8799fd8799f581c3b8c8a100c16cf62b9c2bacc40453aaa67ced633993f2b4eec5b88e4ffffffff")
	err = cbor.Unmarshal(decoded_addr, &pd)
	if err != nil {
		t.Error(err)
	}
	address, _ = plutusencoder.DecodePlutusAddress(pd, 0b0001)
	if address.String() != "addr1qxajla3qcrwckzkur8n0lt02rg2sepw3kgkstckmzrz4ccfm3j9pqrqkea3tns46e3qy2w42vl8dvvue8u45amzm3rjqvv2nxh" {
		t.Error(address, "expected", "addr1qxajla3qcrwckzkur8n0lt02rg2sepw3kgkstckmzrz4ccfm3j9pqrqkea3tns46e3qy2w42vl8dvvue8u45amzm3rjqvv2nxh")
	}

}

type TagWithin7and1400 struct {
	_   struct{} `plutusType:"DefList" plutusConstr:"8"`
	Tag int64    `plutusType:"Int"`
}
type TagAbove1400 struct {
	_   struct{} `plutusType:"DefList" plutusConstr:"1450"`
	Tag int64    `plutusType:"Int"`
}

type InvalidTag struct {
	_   struct{} `plutusType:"DefList" plutusConstr:"test"`
	Tag int64    `plutusType:"Int"`
}

func TestPlutusMarshalCborTags(t *testing.T) {
	tw7 := TagWithin7and1400{
		Tag: 5,
	}
	marshaled, err := plutusencoder.MarshalPlutus(tw7)
	if err != nil {
		t.Error(err)
	}
	encoded, err := cbor.Marshal(marshaled)
	if err != nil {
		t.Error(err)
	}
	if hex.EncodeToString(encoded) != "d905018105" {
		t.Error("encoding error", hex.EncodeToString(encoded))
	}
	ta1400 := TagAbove1400{
		Tag: 1400,
	}
	_, err = plutusencoder.MarshalPlutus(ta1400)
	if err == nil {
		t.Error("should have thrown error")
	}

	invalid := InvalidTag{
		Tag: 5,
	}
	_, err = plutusencoder.MarshalPlutus(invalid)
	if err == nil {
		t.Error("should have thrown error")
	}

}

type InvalidStructTag struct {
	_ struct{} `plutusType:"test" plutusConstr:"2"`
}

func TestPlutusMarshalStructTags(t *testing.T) {
	invalid := InvalidStructTag{}
	_, err := plutusencoder.MarshalPlutus(invalid)
	if err == nil {
		t.Error("should have thrown error")
	}
}

type MapStruct struct {
	_     struct{} `plutusType:"Map" plutusConstr:"2"`
	Value int64    `plutusType:"Int"`
}

func TestPlutusMarshalMap(t *testing.T) {
	m := MapStruct{
		Value: 5,
	}
	marshaled, err := plutusencoder.MarshalPlutus(m)
	if err != nil {
		t.Error(err)
	}
	encoded, err := cbor.Marshal(marshaled)
	if err != nil {
		t.Error(err)
	}
	if hex.EncodeToString(encoded) != "d87ba14556616c756505" {
		t.Error("encoding error", hex.EncodeToString(encoded))
	}
}

type FieldConstrwithin7and1400 struct {
	_     struct{} `plutusType:"DefList" plutusConstr:"2"`
	Value int64    `plutusType:"Int" plutusConstr:"12"`
}

type FieldConstrAbove1400 struct {
	_     struct{} `plutusType:"DefList" plutusConstr:"2"`
	Value int64    `plutusType:"Int" plutusConstr:"1450"`
}

type FieldConstrBelow7 struct {
	_     struct{} `plutusType:"DefList" plutusConstr:"2"`
	Value int64    `plutusType:"Int" plutusConstr:"5"`
}

func TestFieldConstr(t *testing.T) {
	fc7 := FieldConstrwithin7and1400{
		Value: 5,
	}
	marshaled, err := plutusencoder.MarshalPlutus(fc7)
	if err != nil {
		t.Error(err)
	}
	encoded, err := cbor.Marshal(marshaled)
	if err != nil {
		t.Error(err)
	}
	if hex.EncodeToString(encoded) != "d87b81d9050505" {
		t.Error("encoding error", hex.EncodeToString(encoded))
	}
	fc1400 := FieldConstrAbove1400{
		Value: 1400,
	}
	_, err = plutusencoder.MarshalPlutus(fc1400)
	if err == nil {
		t.Error("should have thrown error")
	}

	fc5 := FieldConstrBelow7{
		Value: 5,
	}
	marshaled, err = plutusencoder.MarshalPlutus(fc5)
	if err != nil {
		t.Error(err)
	}
	encoded, err = cbor.Marshal(marshaled)
	if err != nil {
		t.Error(err)
	}
	if hex.EncodeToString(encoded) != "d87b81d87e05" {
		t.Error("encoding error", hex.EncodeToString(encoded))
	}
}

type MissingStructTag struct {
	Val int64 `plutusType:"Int"`
}

func TestMissingStructTag(t *testing.T) {
	m := MissingStructTag{
		Val: 5,
	}
	_, err := plutusencoder.MarshalPlutus(m)
	if err == nil {
		t.Error("should have thrown error")
	}
}

type SimpleIndefList struct {
	_     struct{} `plutusType:"IndefList" plutusConstr:"1"`
	Value int64    `plutusType:"Int"`
}

type SimpleDefList struct {
	_     struct{} `plutusType:"DefList" plutusConstr:"2"`
	Value int64    `plutusType:"Int"`
}

type SimpleMap struct {
	_     struct{} `plutusType:"Map" plutusConstr:"2"`
	Value int64    `plutusType:"Int"`
}

type AllTypesWithMap struct {
	_                 struct{}        `plutusType:"Map" plutusConstr:"2"`
	Int               int64           `plutusType:"Int"`
	StringBytes       string          `plutusType:"StringBytes"`
	HexString         string          `plutusType:"HexString"`
	Address           Address.Address `plutusType:"Address"`
	Bytes             []byte          `plutusType:"Bytes"`
	EmptyList         SimpleIndefList
	EmptyDefList      SimpleDefList
	EmptyMap          SimpleMap
	ArrayOfStructs    []SimpleIndefList `plutusType:"IndefList"`
	ArrayOfStructsDef []SimpleDefList   `plutusType:"DefList"`
}

func TestAllTypesWithMap(t *testing.T) {
	decodedAddr, _ := Address.DecodeAddress("addr1qxajla3qcrwckzkur8n0lt02rg2sepw3kgkstckmzrz4ccfm3j9pqrqkea3tns46e3qy2w42vl8dvvue8u45amzm3rjqvv2nxh")
	m := AllTypesWithMap{
		Int:               5,
		StringBytes:       "Hello World",
		HexString:         "",
		Address:           decodedAddr,
		Bytes:             []byte{0x01, 0x02, 0x03, 0x04},
		EmptyList:         SimpleIndefList{Value: 5},
		EmptyDefList:      SimpleDefList{Value: 5},
		EmptyMap:          SimpleMap{Value: 5},
		ArrayOfStructs:    []SimpleIndefList{},
		ArrayOfStructsDef: []SimpleDefList{{Value: 5}, {Value: 5}, {Value: 5}}}
	marshaled, err := plutusencoder.MarshalPlutus(m)
	if err != nil {
		t.Error(err)
	}
	if marshaled.PlutusDataType != PlutusData.PlutusMap {
		t.Error("wrong type")
	}
}

type AllTypesWithDefList struct {
	_                 struct{}        `plutusType:"DefList" plutusConstr:"2"`
	Int               int64           `plutusType:"Int"`
	StringBytes       string          `plutusType:"StringBytes"`
	HexString         string          `plutusType:"HexString"`
	Address           Address.Address `plutusType:"Address"`
	Bytes             []byte          `plutusType:"Bytes"`
	EmptyList         SimpleIndefList
	EmptyDefList      SimpleDefList
	EmptyMap          SimpleMap
	ArrayOfStructs    []SimpleIndefList `plutusType:"IndefList"`
	ArrayOfStructsDef []SimpleDefList   `plutusType:"DefList"`
}

func TestAllTypesWithDefList(t *testing.T) {
	decodedAddr, _ := Address.DecodeAddress("addr1qxajla3qcrwckzkur8n0lt02rg2sepw3kgkstckmzrz4ccfm3j9pqrqkea3tns46e3qy2w42vl8dvvue8u45amzm3rjqvv2nxh")
	m := AllTypesWithDefList{
		Int:               5,
		StringBytes:       "Hello World",
		HexString:         "fc11a9ef431f81b837736be5f53e4da29b9469c983d07f321262ce61",
		Address:           decodedAddr,
		Bytes:             []byte{0x01, 0x02, 0x03, 0x04},
		EmptyList:         SimpleIndefList{Value: 5},
		EmptyDefList:      SimpleDefList{Value: 5},
		EmptyMap:          SimpleMap{Value: 5},
		ArrayOfStructs:    []SimpleIndefList{},
		ArrayOfStructsDef: []SimpleDefList{{Value: 5}, {Value: 5}, {Value: 5}}}

	marshaled, err := plutusencoder.MarshalPlutus(m)
	if err != nil {
		t.Error(err)
	}
	encoded, err := cbor.Marshal(marshaled)
	if err != nil {
		t.Error(err)
	}
	if hex.EncodeToString(encoded) != "d87b8a054b48656c6c6f20576f726c64581cfc11a9ef431f81b837736be5f53e4da29b9469c983d07f321262ce61d8799fd8799f581cbb2ff620c0dd8b0adc19e6ffadea1a150c85d1b22d05e2db10c55c61ffd8799fd8799fd8799f581c3b8c8a100c16cf62b9c2bacc40453aaa67ced633993f2b4eec5b88e4ffffffff4401020304d87a9f05ffd87b8105d87ba14556616c7565059fff83d87b8105d87b8105d87b8105" {
		t.Error("encoding error", hex.EncodeToString(encoded))
	}

}

type AllTypesWithIndefList struct {
	_                 struct{}        `plutusType:"IndefList" plutusConstr:"1"`
	Int               int64           `plutusType:"Int"`
	StringBytes       string          `plutusType:"StringBytes"`
	HexString         string          `plutusType:"HexString"`
	Address           Address.Address `plutusType:"Address"`
	Bytes             []byte          `plutusType:"Bytes"`
	EmptyList         SimpleIndefList
	EmptyDefList      SimpleDefList
	EmptyMap          SimpleMap
	ArrayOfStructs    []SimpleIndefList `plutusType:"IndefList"`
	ArrayOfStructsDef []SimpleDefList   `plutusType:"DefList"`
}

func TestAllTypesWithIndefList(t *testing.T) {
	decodedAddr, _ := Address.DecodeAddress("addr1qxajla3qcrwckzkur8n0lt02rg2sepw3kgkstckmzrz4ccfm3j9pqrqkea3tns46e3qy2w42vl8dvvue8u45amzm3rjqvv2nxh")
	m := AllTypesWithIndefList{
		Int:               5,
		StringBytes:       "Hello World",
		HexString:         "fc11a9ef431f81b837736be5f53e4da29b9469c983d07f321262ce61",
		Address:           decodedAddr,
		Bytes:             []byte{0x01, 0x02, 0x03, 0x04},
		EmptyList:         SimpleIndefList{Value: 5},
		EmptyDefList:      SimpleDefList{Value: 5},
		EmptyMap:          SimpleMap{Value: 5},
		ArrayOfStructs:    []SimpleIndefList{},
		ArrayOfStructsDef: []SimpleDefList{{Value: 5}, {Value: 5}, {Value: 5}}}

	marshaled, err := plutusencoder.MarshalPlutus(m)
	if err != nil {
		t.Error(err)
	}
	encoded, err := cbor.Marshal(marshaled)
	if err != nil {
		t.Error(err)
	}
	if hex.EncodeToString(encoded) != "d87a9f054b48656c6c6f20576f726c64581cfc11a9ef431f81b837736be5f53e4da29b9469c983d07f321262ce61d8799fd8799f581cbb2ff620c0dd8b0adc19e6ffadea1a150c85d1b22d05e2db10c55c61ffd8799fd8799fd8799f581c3b8c8a100c16cf62b9c2bacc40453aaa67ced633993f2b4eec5b88e4ffffffff4401020304d87a9f05ffd87b8105d87ba14556616c7565059fff83d87b8105d87b8105d87b8105ff" {
		t.Error("encoding error", hex.EncodeToString(encoded))
	}

}

type InvalidInt struct {
	_     struct{} `plutusType:"DefList" plutusConstr:"2"`
	Value string   `plutusType:"Int"`
}

func TestInvalidInt(t *testing.T) {
	invalid := InvalidInt{
		Value: "test",
	}
	_, err := plutusencoder.MarshalPlutus(invalid)
	if err == nil {
		t.Error("should have thrown error")
	}
}

type InvalidStringBytes struct {
	_     struct{} `plutusType:"DefList" plutusConstr:"2"`
	Value int64    `plutusType:"StringBytes"`
}

func TestInvalidStringBytes(t *testing.T) {
	invalid := InvalidStringBytes{
		Value: 5,
	}
	_, err := plutusencoder.MarshalPlutus(invalid)
	if err == nil {
		t.Error("should have thrown error")
	}
}

type InvalidHexString struct {
	_     struct{} `plutusType:"DefList" plutusConstr:"2"`
	Value int64    `plutusType:"HexString"`
}

func TestInvalidHexString(t *testing.T) {
	invalid := InvalidHexString{
		Value: 5,
	}
	_, err := plutusencoder.MarshalPlutus(invalid)
	if err == nil {
		t.Error("should have thrown error")
	}
}

type InvalidBytes struct {
	_     struct{} `plutusType:"DefList" plutusConstr:"2"`
	Value int64    `plutusType:"Bytes"`
}

func TestInvalidBytes(t *testing.T) {
	invalid := InvalidBytes{
		Value: 5,
	}
	_, err := plutusencoder.MarshalPlutus(invalid)
	if err == nil {
		t.Error("should have thrown error")
	}
}

type NonHexString struct {
	_     struct{} `plutusType:"DefList" plutusConstr:"2"`
	Value string   `plutusType:"HexString"`
}

func TestNonHexString(t *testing.T) {
	invalid := NonHexString{
		Value: "test",
	}
	_, err := plutusencoder.MarshalPlutus(invalid)
	if err == nil {
		t.Error("should have thrown error")
	}
}

type InvalidFieldConstr struct {
	_     struct{} `plutusType:"DefList" plutusConstr:"2"`
	Value int64    `plutusType:"Int" plutusConstr:"test"`
}

func TestInvalidFieldConstr(t *testing.T) {
	invalid := InvalidFieldConstr{
		Value: 5,
	}
	_, err := plutusencoder.MarshalPlutus(invalid)
	if err == nil {
		t.Error("should have thrown error")
	}
}

func TestCborUnmarshal(t *testing.T) {
	invalidHex := "test"
	wrongStructHex := "d87b81d87e05"
	validHex := "d87b81d8799fd8799f581cbb2ff620c0dd8b0adc19e6ffadea1a150c85d1b22d05e2db10c55c61ffd8799fd8799fd8799f581c3b8c8a100c16cf62b9c2bacc40453aaa67ced633993f2b4eec5b88e4ffffffff"
	nonPlutusDataHex := "00120"
	str := TestAddress{}
	err := plutusencoder.CborUnmarshal(invalidHex, &str, 1)
	if err == nil {
		t.Error("should have thrown error")
	}
	err = plutusencoder.CborUnmarshal(wrongStructHex, &str, 1)
	if err == nil {
		t.Error("should have thrown error")
	}
	err = plutusencoder.CborUnmarshal(validHex, &str, 1)
	if err != nil {
		t.Error(err)
	}
	err = plutusencoder.CborUnmarshal(nonPlutusDataHex, &str, 1)
	if err == nil {
		t.Error("should have thrown error")
	}
}

func TestUnmarshalInDepth(t *testing.T) {
	allWithIndefCbor := "d87a9f054b48656c6c6f20576f726c64581cfc11a9ef431f81b837736be5f53e4da29b9469c983d07f321262ce61d8799fd8799f581cbb2ff620c0dd8b0adc19e6ffadea1a150c85d1b22d05e2db10c55c61ffd8799fd8799fd8799f581c3b8c8a100c16cf62b9c2bacc40453aaa67ced633993f2b4eec5b88e4ffffffff4401020304d87a9f05ffd87b8105d87ba14556616c7565059fff83d87b8105d87b8105d87b8105ff"
	str := AllTypesWithIndefList{}
	err := plutusencoder.CborUnmarshal(allWithIndefCbor, &str, 1)
	if err != nil {
		t.Error(err)
	}
	allWithDefCbor := "d87b8a054b48656c6c6f20576f726c64581cfc11a9ef431f81b837736be5f53e4da29b9469c983d07f321262ce61d8799fd8799f581cbb2ff620c0dd8b0adc19e6ffadea1a150c85d1b22d05e2db10c55c61ffd8799fd8799fd8799f581c3b8c8a100c16cf62b9c2bacc40453aaa67ced633993f2b4eec5b88e4ffffffff4401020304d87a9f05ffd87b8105d87ba14556616c7565059fff83d87b8105d87b8105d87b8105"
	str2 := AllTypesWithDefList{}
	err = plutusencoder.CborUnmarshal(allWithDefCbor, &str2, 1)
	if err != nil {
		t.Error(err)
	}
	pd1, err := plutusencoder.MarshalPlutus(str)
	if err != nil {
		t.Error(err)
	}
	pd2, err := plutusencoder.MarshalPlutus(str2)
	if err != nil {
		t.Error(err)
	}
	encoded1, err := cbor.Marshal(pd1)
	if err != nil {
		t.Error(err)
	}
	encoded2, err := cbor.Marshal(pd2)
	if err != nil {
		t.Error(err)
	}
	if hex.EncodeToString(encoded1) != allWithIndefCbor {
		t.Error("encoding error", hex.EncodeToString(encoded1))
	}
	if hex.EncodeToString(encoded2) != allWithDefCbor {
		t.Error("encoding error", hex.EncodeToString(encoded2))
	}

}

type NestedMap struct {
	_   struct{} `plutusType:"DefList" plutusConstr:"2"`
	Map SimpleMap
}

func TestMarshalMapNested(t *testing.T) {
	m := NestedMap{
		Map: SimpleMap{Value: 5},
	}
	marshaled, err := plutusencoder.MarshalPlutus(m)
	if err != nil {
		t.Error(err)
	}

	if marshaled.Value.(PlutusData.PlutusDefArray)[0].PlutusDataType != PlutusData.PlutusMap {
		t.Error("wrong type")
	}
	encoded, err := cbor.Marshal(marshaled)
	if err != nil {
		t.Error(err)
	}
	if hex.EncodeToString(encoded) != "d87b81d87ba14556616c756505" {
		t.Error("encoding error", hex.EncodeToString(encoded))
	}

	resultinStruct := NestedMap{}
	err = plutusencoder.CborUnmarshal("d87b81d87ba14556616c756505", &resultinStruct, 1)
	if err != nil {
		t.Error(err)
	}

}

type HexStringTest struct {
	_         struct{} `plutusType:"DefList" plutusConstr:"2"`
	HexString string   `plutusType:"HexString"`
}

func TestHexString(t *testing.T) {
	m := HexStringTest{
		HexString: "fc11a9ef431f81b837736be5f53e4da29b9469c983d07f321262ce61",
	}
	marshaled, err := plutusencoder.MarshalPlutus(m)
	if err != nil {
		t.Error(err)
	}

	if marshaled.Value.(PlutusData.PlutusDefArray)[0].PlutusDataType != PlutusData.PlutusBytes {
		t.Error("wrong type")
	}
	encoded, err := cbor.Marshal(marshaled)
	if err != nil {
		t.Error(err)
	}
	if hex.EncodeToString(encoded) != "d87b81581cfc11a9ef431f81b837736be5f53e4da29b9469c983d07f321262ce61" {
		t.Error("encoding error", hex.EncodeToString(encoded))
	}

	resultinStruct := HexStringTest{}
	err = plutusencoder.CborUnmarshal("d87b81581cfc11a9ef431f81b837736be5f53e4da29b9469c983d07f321262ce61", &resultinStruct, 1)
	if err != nil {
		t.Error(err)
	}

}

type InternalMap struct {
	_         struct{} `plutusType:"Map" plutusConstr:"2"`
	Value     int64    `plutusType:"Int"`
	Bytes     []byte   `plutusType:"Bytes"`
	HexString string   `plutusType:"HexString"`
}

type MapNested struct {
	_ struct{} `plutusType:"Map" plutusConstr:"2"`
	X InternalMap
}

func TestMapInternal(t *testing.T) {
	m := MapNested{
		X: InternalMap{
			Bytes:     []byte{0x01, 0x02, 0x03, 0x04},
			HexString: "fc11a9ef431f81b837736be5f53e4da29b9469c983d07f321262ce61",
		},
	}
	marshaled, err := plutusencoder.MarshalPlutus(m)
	if err != nil {
		t.Error(err)
	}
	encoded, err := cbor.Marshal(marshaled)
	if err != nil {
		t.Error(err)
	}
	if hex.EncodeToString(encoded) != "d87ba14158d87ba34556616c756500454279746573440102030449486578537472696e67581cfc11a9ef431f81b837736be5f53e4da29b9469c983d07f321262ce61" {
		t.Error("encoding error", hex.EncodeToString(encoded))
	}

	resultinStruct := MapNested{}
	err = plutusencoder.CborUnmarshal("d87ba14158d87ba34556616c756500454279746573440102030449486578537472696e67581cfc11a9ef431f81b837736be5f53e4da29b9469c983d07f321262ce61", &resultinStruct, 1)
	if err != nil {
		t.Error(err)
	}

}

type MapTest struct {
	_          struct{}          `plutusType:"Map" plutusConstr:"2"`
	Value      int64             `plutusType:"Int"`
	Bytes      []byte            `plutusType:"Bytes"`
	HexString  string            `plutusType:"HexString"`
	Address    Address.Address   `plutusType:"Address"`
	DefArray   []SimpleDefList   `plutusType:"DefList"`
	IndefArray []SimpleIndefList `plutusType:"IndefList"`
	Map        SimpleMap
}

func TestMap(t *testing.T) {
	addr, _ := Address.DecodeAddress("addr1qxajla3qcrwckzkur8n0lt02rg2sepw3kgkstckmzrz4ccfm3j9pqrqkea3tns46e3qy2w42vl8dvvue8u45amzm3rjqvv2nxh")
	m := MapTest{
		Value:      5,
		Bytes:      []byte{0x01, 0x02, 0x03, 0x04},
		HexString:  "fc11a9ef431f81b837736be5f53e4da29b9469c983d07f321262ce61",
		Address:    addr,
		DefArray:   []SimpleDefList{{Value: 5}, {Value: 5}, {Value: 5}},
		IndefArray: []SimpleIndefList{{Value: 5}, {Value: 5}, {Value: 5}},
		Map:        SimpleMap{Value: 5},
	}
	fmt.Println("MARSHAL")
	_, err := plutusencoder.MarshalPlutus(m)
	if err != nil {
		t.Error(err)
	}
	// encoded, err := cbor.Marshal(marshaled)
	// if err != nil {
	// 	t.Error(err)
	// }
	// if hex.EncodeToString(encoded) != "d87ba74a496e64656641727261799fd87a9f05ffd87a9f05ffd87a9f05ffff434d6170d87ba14556616c7565054556616c756505454279746573440102030449486578537472696e67581cfc11a9ef431f81b837736be5f53e4da29b9469c983d07f321262ce614741646472657373d8799fd8799f581cbb2ff620c0dd8b0adc19e6ffadea1a150c85d1b22d05e2db10c55c61ffd8799fd8799fd8799f581c3b8c8a100c16cf62b9c2bacc40453aaa67ced633993f2b4eec5b88e4ffffffff48446566417272617983d87b8105d87b8105d87b8105" {
	// 	t.Error("encoding error", hex.EncodeToString(encoded))
	// }
	//var err error
	resultinStruct := MapTest{}
	err = plutusencoder.CborUnmarshal("d87ba74a496e64656641727261799fd87a9f05ffd87a9f05ffd87a9f05ffff434d6170d87ba14556616c7565054556616c756505454279746573440102030449486578537472696e67581cfc11a9ef431f81b837736be5f53e4da29b9469c983d07f321262ce614741646472657373d8799fd8799f581cbb2ff620c0dd8b0adc19e6ffadea1a150c85d1b22d05e2db10c55c61ffd8799fd8799fd8799f581c3b8c8a100c16cf62b9c2bacc40453aaa67ced633993f2b4eec5b88e4ffffffff48446566417272617983d87b8105d87b8105d87b8105", &resultinStruct, 1)
	if err != nil {
		t.Error(err)
	}
	if resultinStruct.Address.String() != "addr1qxajla3qcrwckzkur8n0lt02rg2sepw3kgkstckmzrz4ccfm3j9pqrqkea3tns46e3qy2w42vl8dvvue8u45amzm3rjqvv2nxh" {
		t.Error("wrong address")
	}
	if resultinStruct.HexString != "fc11a9ef431f81b837736be5f53e4da29b9469c983d07f321262ce61" {
		t.Error("wrong hexstring")
	}
	if resultinStruct.Value != 5 {
		t.Error("wrong value")
	}
	if len(resultinStruct.Bytes) != 4 {
		t.Error("wrong bytes")
	}
	if len(resultinStruct.DefArray) != 3 {
		t.Error("wrong def array", len(resultinStruct.DefArray))
	}
	if len(resultinStruct.IndefArray) != 3 {
		t.Error("wrong indef array", len(resultinStruct.IndefArray))
	}
	if resultinStruct.Map.Value != 5 {
		t.Error("wrong map", resultinStruct.Map.Value)
	}

}

type InputMap struct {
	_      struct{} `plutusType:"IntMap"`
	TokenA plutusencoder.MultiAssetPlutus
	TokenB plutusencoder.MultiAssetPlutus
}

type Asset struct {
	_      struct{} `plutusType:"IndefList" plutusConstr:"0"`
	Policy []byte   `plutusType:"Bytes"`
	Name   []byte   `plutusType:"Bytes"`
}

type InputOutput struct {
	_      struct{} `plutusType:"IndefList"`
	Input  Asset
	Output Asset
}

type Date struct {
	_    struct{} `plutusType:"IndefList" plutusConstr:"7"`
	Date int64    `plutusType:"Int"`
}

type Price struct {
	_          struct{} `plutusType:"IndefList" plutusConstr:"0"`
	BasePrice  int64    `plutusType:"Int"`
	FinalPrice int64    `plutusType:"Int"`
}

type Pricing struct {
	_     struct{} `plutusType:"IndefList" plutusConstr:"2"`
	Price Price
}

type OrderMap struct {
	_         struct{} `plutusType:"Map"`
	EndDate   Date
	StartDate Date
	Price     Pricing
}

type EmptyStruct struct {
	_ struct{} `plutusType:"Map"`
}

type DatTest struct {
	_           struct{} `plutusType:"IndefList"`
	InputMap    InputMap
	InputOutput InputOutput
	OrderToken  Asset
	OrderMap    OrderMap
	EmptyMap    EmptyStruct
}

func TestComplextDatum(t *testing.T) {
	decodedCbor, _ := hex.DecodeString("9fa200a140a1401a0098968001a09fd8799f4040ffd8799f581ca0028f350aaabe0545fdcb56b039bfb08e4bb4d8c4d7c3c7d481c23545484f534b59ffffd8799f581c279afa4c71ea050339f0699de9095daa612485ee8489fc413e93ebda4953505f725650717978ffa347656e6444617465d905009f1b000003bbcc68f818ff457072696365d87b9fd8799f190cbb19c350ffff49737461727444617465d905009f1b0000018ce8cdd1b5ffa0ff")
	pd := PlutusData.PlutusData{}
	err := cbor.Unmarshal(decodedCbor, &pd)
	if err != nil {
		t.Error(err)
	}
	marshaled, _ := cbor.Marshal(pd)
	if hex.EncodeToString(marshaled) != hex.EncodeToString(decodedCbor) {
		t.Error("decoding error", hex.EncodeToString(marshaled))
	}
	dat := DatTest{}
	err = plutusencoder.UnmarshalPlutus(&pd, &dat, 1)
	if err != nil {
		fmt.Println("FROM PLUTUS")
		fmt.Println(err)
		fmt.Println("DATUM", dat)
		t.Error(err)
	}
	fmt.Println(dat.OrderMap.EndDate.Date)
	fmt.Println(dat.OrderMap.Price)
	if dat.OrderMap.EndDate.Date != 4105123199000 {
		t.Error("wrong date")
	}
	pdFromStruct, err := plutusencoder.MarshalPlutus(dat)
	if err != nil {
		fmt.Println("TO PLUTUS")
		t.Error(err)
	}
	fmt.Println("TRANSFORMED TO PLUTUS?")
	encoded, err := cbor.Marshal(pdFromStruct)
	if err != nil {
		fmt.Println("CBOR MARSHAL")
		t.Error(err)
	}
	fmt.Println("ENCODED", hex.EncodeToString(encoded))
	if len(encoded) != len(decodedCbor) {
		t.Error("encoding error", hex.EncodeToString(encoded))
	}
	// if hex.EncodeToString(encoded) != hex.EncodeToString(decodedCbor) {
	// 	t.Error("encoding error", hex.EncodeToString(encoded))
	// }

}

// type FeeDatumGens struct {
// 	InputToValueMap map[PlutusUtxo]PlutusValue
// 	EmptyValue      PlutusValue
// 	EmptyArray      []byte `plutusType:"Bytes", PlutusConstr:"1"`
// }

// func TestFeeDatumGens(t *testing.T) {

// }

// func TestDecodeFeeDatumGens(t *testing.T) {
// 	hexEncodedCBor := "d87983a1d87982d879815820b4bf6f7a29915cdf1aaac9d2112fc986bb3227d9cd04d7af418991cee23b07ed00a240a1401a000f4240581c95a427e384527065f2f8946f5e86320d0117839a5e98ea2c0b55fb00a14448554e54193a98a0d87a80"
// 	decoded_hex, _ := hex.DecodeString(hexEncodedCBor)
// 	pd := PlutusData.PlutusData{}
// 	err := cbor.Unmarshal(decoded_hex, &pd)
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	fmt.Println(pd.String())
// 	encoded, err := cbor.Marshal(pd)
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	fmt.Println(hex.EncodeToString(encoded))
// 	if hex.EncodeToString(encoded) != hexEncodedCBor {
// 		t.Error("encoding error", hex.EncodeToString(encoded))
// 	}

// 	// byteArr := []byte{216, 121, 130, 216, 121, 129, 88, 32, 180, 191, 111, 122, 41, 145, 92, 223, 26, 170, 201, 210, 17, 47, 201, 134, 187, 50, 39, 217, 205, 4, 215, 175, 65, 137, 145, 206, 226, 59, 7, 237, 0}
// 	// pd2 := PlutusData.PlutusData{}
// 	// err = cbor.Unmarshal(byteArr, &pd2)
// 	// if err != nil {
// 	// 	t.Error(err)
// 	// }
// 	// fmt.Println(pd2.String())
// 	//x := map[[]byte][]byte{}
// }

func TestAxoDatum(t *testing.T) {
	cborHex := "d8799fa200a140a1401a20c8558001a09fd8799f4040ffd8799f581c420000029ad9527271b1b1e3c27ee065c18df70a4a4cfc3093a41a444341584fffffd8799f581c93833468a70ab5eec63b00a1646297430a502d0da68d533b80f217894745466747613941ffa347656e6444617465d905009f1b000003bb2cc3d418ff457072696365d87b9fd8799f1b00044b339b77367b1b01bc16d674ec8000ffff49737461727444617465d905009f1b00000092f3973818ffa0ff"
	decodedHex, _ := hex.DecodeString(cborHex)
	pd := PlutusData.PlutusData{}
	err := cbor.Unmarshal(decodedHex, &pd)
	if err != nil {
		t.Error(err)
	}

}

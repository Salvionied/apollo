package AssetName_test

import (
	"encoding/hex"
	"testing"

	"github.com/Salvionied/apollo/serialization/AssetName"
	"github.com/fxamacker/cbor/v2"
)

func TestAssetNameCreators(t *testing.T) {
	assetName := AssetName.NewAssetNameFromString("test")
	if assetName.String() != "test" {
		t.Errorf("AssetName should be 'test'")
	}
	if assetName.HexString() != "74657374" {
		t.Errorf("AssetName should be '74657374'")
	}

	assetName = *AssetName.NewAssetNameFromHexString("74657374")
	if assetName.String() != "test" {
		t.Errorf("AssetName should be 'test'")
	}
	if assetName.HexString() != "74657374" {
		t.Errorf("AssetName should be '74657374'")
	}
}

func TestMarshal(t *testing.T) {
	assetName := AssetName.NewAssetNameFromString("test")
	val, err := assetName.MarshalCBOR()
	if err != nil {
		t.Errorf("AssetName should marshal")
	}
	if hex.EncodeToString(val) != "4474657374" {
		t.Errorf("AssetName should be '4474657374', %s", hex.EncodeToString(val))
	}

	an2 := AssetName.AssetName{}

	err = an2.UnmarshalCBOR(val)
	if err != nil {
		t.Errorf("AssetName should unmarshal")
	}

	if an2.String() != "test" {
		t.Errorf("AssetName should be 'test'")
	}
}

func TestEmptyNameMarshal(t *testing.T) {
	assetName := AssetName.NewAssetNameFromString("")
	val, err := assetName.MarshalCBOR()
	if err != nil {
		t.Errorf("AssetName should marshal")
	}
	if hex.EncodeToString(val) != "40" {
		t.Errorf("AssetName should be '40', %s", hex.EncodeToString(val))
	}

	an2 := AssetName.AssetName{}

	err = an2.UnmarshalCBOR(val)
	if err != nil {
		t.Errorf("AssetName should unmarshal")
	}

	if an2.String() != "" {
		t.Errorf("AssetName should be ''")
	}
}

func TestInvalidAssetNameCreation(t *testing.T) {
	assetName := AssetName.NewAssetNameFromHexString("fc11a9ef431f81b837736be5f53e4da29b9469c983d07f321262ce61fc11a9ef431f81b837736be5f53e4da29b9469c983d07f321262ce61")
	if assetName != nil {
		t.Errorf("AssetName should be nil")
	}
}

func TestInvalidLengthOnMarshal(t *testing.T) {
	assetName := AssetName.NewAssetNameFromString("fc11a9ef431f81b837736be5f53e4da29b9469c983d07f321262ce61fc11a9ef431f81b837736be5f53e4da29b9469c983d07f321262ce61")
	_, err := assetName.MarshalCBOR()
	if err == nil {
		t.Errorf("AssetName should not marshal")
	}
}

func TestInvalidUnMarshal(t *testing.T) {
	decoded, _ := hex.DecodeString("fc11a9ef431f81b837736be5f53e4da29b9469c983d07f321262ce61fc11a9ef431f81b837736be5f53e4da29b9469c983d07f321262ce61")
	marshaled, _ := cbor.Marshal(decoded)
	assetName := AssetName.AssetName{}
	err := assetName.UnmarshalCBOR(marshaled)
	if err == nil {
		t.Errorf("AssetName should not unmarshal")
	}

	invalidBytes := []byte{0x01, 0x02, 0x03}
	err = assetName.UnmarshalCBOR(invalidBytes)
	if err == nil {
		t.Errorf("AssetName should not unmarshal")
	}

}

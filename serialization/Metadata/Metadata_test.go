package Metadata_test

import (
	"encoding/hex"
	"testing"

	"github.com/Salvionied/apollo/serialization/Metadata"
	"github.com/Salvionied/apollo/serialization/NativeScript"
	"github.com/fxamacker/cbor/v2"
)

func TestAuxiliaryData(t *testing.T) {
	aux := Metadata.AuxiliaryData{}
	alonzoMeta := Metadata.ShelleyMaryMetadata{}

	meta := Metadata.Metadata{
		721: Metadata.TagMetadata{"{policy_id}": map[string]map[string]string{
			"{policy_name}": {
				"description": "<optional>",
				"name":        "<required>",
				"image":       "<required>",
				"type":        "<optional>",
			}},
		},
	}
	alonzoMeta.Metadata = meta
	aux.SetShelleyMetadata(alonzoMeta)
	marshaled, err := cbor.Marshal(aux)
	if err != nil {
		t.Errorf("Error while marshaling")
	}
	if hex.EncodeToString(marshaled) != `a11902d1a16b7b706f6c6963795f69647da16d7b706f6c6963795f6e616d657da4646e616d656a3c72657175697265643e64747970656a3c6f7074696f6e616c3e65696d6167656a3c72657175697265643e6b6465736372697074696f6e6a3c6f7074696f6e616c3e` {
		t.Errorf("InvalidReserialization got %s expected %s", hex.EncodeToString(marshaled), `a11902d1a16b7b706f6c6963795f69647da16d7b706f6c6963795f6e616d657da4646e616d656a3c72657175697265643e64747970656a3c6f7074696f6e616c3e65696d6167656a3c72657175697265643e6b6465736372697074696f6e6a3c6f7074696f6e616c3e`)
	}
	if len(aux.Hash()) != 32 {
		t.Errorf("Invalid Hashing Of AuxiliaryData")
	}
	if hex.EncodeToString(aux.Hash()) != "9ef720ec820d751e0b7d18534b37a19c2fea055ed49d496b5865d27e8ed34def" {
		t.Errorf("Invalid Hashing Of AuxiliaryData expected %s got %s", "9ef720ec820d751e0b7d18534b37a19c2fea055ed49d496b5865d27e8ed34def", hex.EncodeToString(aux.Hash()))
	}
}

func TestAuxiliaryDataExtended(t *testing.T) {
	Aux := Metadata.AuxiliaryData{}
	shelleyMeta := Metadata.ShelleyMaryMetadata{}
	meta := Metadata.Metadata{
		721: Metadata.TagMetadata{"{policy_id}": map[string]map[string]string{
			"{policy_name}": {
				"description": "<optional>",
				"name":        "<required>",
				"image":       "<required>",
				"type":        "<optional>",
			}},
		},
	}
	shelleyMeta.Metadata = meta

	Aux.SetShelleyMetadata(shelleyMeta)
	marshaled, err := cbor.Marshal(Aux)
	if err != nil {
		t.Errorf("Error while marshaling")
	}
	if hex.EncodeToString(marshaled) != `a11902d1a16b7b706f6c6963795f69647da16d7b706f6c6963795f6e616d657da4646e616d656a3c72657175697265643e64747970656a3c6f7074696f6e616c3e65696d6167656a3c72657175697265643e6b6465736372697074696f6e6a3c6f7074696f6e616c3e` {
		t.Errorf("InvalidReserialization got %s expected %s", hex.EncodeToString(marshaled), `a11902d1a16b7b706f6c6963795f69647da16d7b706f6c6963795f6e616d657da4646e616d656a3c72657175697265643e64747970656a3c6f7074696f6e616c3e65696d6167656a3c72657175697265643e6b6465736372697074696f6e6a3c6f7074696f6e616c3e`)
	}
	if len(Aux.Hash()) != 32 {
		t.Errorf("Invalid Hashing Of AuxiliaryData")
	}
	if hex.EncodeToString(Aux.Hash()) != "9ef720ec820d751e0b7d18534b37a19c2fea055ed49d496b5865d27e8ed34def" {
		t.Errorf("Invalid Hashing Of AuxiliaryData expected %s got %s", "9ef720ec820d751e0b7d18534b37a19c2fea055ed49d496b5865d27e8ed34def", hex.EncodeToString(Aux.Hash()))
	}
	AlonzoAux := Metadata.AuxiliaryData{}
	alonzoMeta := Metadata.AlonzoMetadata{}
	alonzoMeta.Metadata = meta
	alonzoMeta.NativeScripts = []NativeScript.NativeScript{{Tag: NativeScript.ScriptAll}}
	AlonzoAux.SetAlonzoMetadata(alonzoMeta)
	marshaled, err = cbor.Marshal(AlonzoAux)
	if err != nil {
		t.Errorf("Error while marshaling")
	}
	if hex.EncodeToString(marshaled) != `a200a11902d1a16b7b706f6c6963795f69647da16d7b706f6c6963795f6e616d657da4646e616d656a3c72657175697265643e64747970656a3c6f7074696f6e616c3e65696d6167656a3c72657175697265643e6b6465736372697074696f6e6a3c6f7074696f6e616c3e01818201f6` {
		t.Errorf("InvalidReserialization got %s expected %s", hex.EncodeToString(marshaled), `a200a11902d1a16b7b706f6c6963795f69647da16d7b706f6c6963795f6e616d657da4646e616d656a3c72657175697265643e64747970656a3c6f7074696f6e616c3e65696d6167656a3c72657175697265643e6b6465736372697074696f6e6a3c6f7074696f6e616c3e01818201f6`)
	}
	if len(AlonzoAux.Hash()) != 32 {
		t.Errorf("Invalid Hashing Of AuxiliaryData")
	}
	if hex.EncodeToString(Aux.Hash()) != "9ef720ec820d751e0b7d18534b37a19c2fea055ed49d496b5865d27e8ed34def" {
		t.Errorf("Invalid Hashing Of AuxiliaryData expected %s got %s", "9ef720ec820d751e0b7d18534b37a19c2fea055ed49d496b5865d27e8ed34def", hex.EncodeToString(Aux.Hash()))
	}
	BasicAux := Metadata.AuxiliaryData{}
	BasicAux.SetBasicMetadata(meta)
	marshaled, err = cbor.Marshal(BasicAux)
	if err != nil {
		t.Errorf("Error while marshaling")
	}
	if hex.EncodeToString(marshaled) != `a11902d1a16b7b706f6c6963795f69647da16d7b706f6c6963795f6e616d657da4646e616d656a3c72657175697265643e64747970656a3c6f7074696f6e616c3e65696d6167656a3c72657175697265643e6b6465736372697074696f6e6a3c6f7074696f6e616c3e` {
		t.Errorf("InvalidReserialization got %s expected %s", hex.EncodeToString(marshaled), `a11902d1a16b7b706f6c6963795f69647da16d7b706f6c6963795f6e616d657da4646e616d656a3c72657175697265643e64747970656a3c6f7074696f6e616c3e65696d6167656a3c72657175697265643e6b6465736372697074696f6e6a3c6f7074696f6e616c3e`)
	}
	if len(BasicAux.Hash()) != 32 {
		t.Errorf("Invalid Hashing Of AuxiliaryData")
	}
	if hex.EncodeToString(Aux.Hash()) != "9ef720ec820d751e0b7d18534b37a19c2fea055ed49d496b5865d27e8ed34def" {
		t.Errorf("Invalid Hashing Of AuxiliaryData expected %s got %s", "9ef720ec820d751e0b7d18534b37a19c2fea055ed49d496b5865d27e8ed34def", hex.EncodeToString(Aux.Hash()))
	}
}

func TestUnmarshal(t *testing.T) {
	cbor := "a11902d1a16b7b706f6c6963795f69647da16d7b706f6c6963795f6e616d657da4646e616d656a3c72657175697265643e64747970656a3c6f7074696f6e616c3e65696d6167656a3c72657175697265643e6b6465736372697074696f6e6a3c6f7074696f6e616c3e"
	Aux := Metadata.AuxiliaryData{}
	decoded, _ := hex.DecodeString(cbor)
	err := Aux.UnmarshalCBOR(decoded)
	if err != nil {
		t.Errorf("Error while unmarshaling")
	}
	invalidcbor := "1fff"
	decoded, _ = hex.DecodeString(invalidcbor)
	err = Aux.UnmarshalCBOR(decoded)
	if err == nil {
		t.Error("Should throw")
	}
}

func TestMarshalEmptyAux(t *testing.T) {
	Aux := Metadata.AuxiliaryData{}
	marshaled, err := cbor.Marshal(Aux)
	if err != nil {
		t.Errorf("Error while marshaling")
	}
	if hex.EncodeToString(marshaled) != `f6` {
		t.Errorf("InvalidReserialization got %s expected %s", hex.EncodeToString(marshaled), `f6`)
	}
}

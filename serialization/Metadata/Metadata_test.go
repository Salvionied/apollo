package Metadata_test

import (
	"bytes"
	"encoding/hex"
	"reflect"
	"strings"
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
	if hex.EncodeToString(
		marshaled,
	) != `a11902d1a16b7b706f6c6963795f69647da16d7b706f6c6963795f6e616d657da4646e616d656a3c72657175697265643e64747970656a3c6f7074696f6e616c3e65696d6167656a3c72657175697265643e6b6465736372697074696f6e6a3c6f7074696f6e616c3e` {
		t.Errorf(
			"InvalidReserialization got %s expected %s",
			hex.EncodeToString(marshaled),
			`a11902d1a16b7b706f6c6963795f69647da16d7b706f6c6963795f6e616d657da4646e616d656a3c72657175697265643e64747970656a3c6f7074696f6e616c3e65696d6167656a3c72657175697265643e6b6465736372697074696f6e6a3c6f7074696f6e616c3e`,
		)
	}
	if len(aux.Hash()) != 32 {
		t.Errorf("Invalid Hashing Of AuxiliaryData")
	}
	if hex.EncodeToString(
		aux.Hash(),
	) != "9ef720ec820d751e0b7d18534b37a19c2fea055ed49d496b5865d27e8ed34def" {
		t.Errorf(
			"Invalid Hashing Of AuxiliaryData expected %s got %s",
			"9ef720ec820d751e0b7d18534b37a19c2fea055ed49d496b5865d27e8ed34def",
			hex.EncodeToString(aux.Hash()),
		)
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
	if hex.EncodeToString(
		marshaled,
	) != `a11902d1a16b7b706f6c6963795f69647da16d7b706f6c6963795f6e616d657da4646e616d656a3c72657175697265643e64747970656a3c6f7074696f6e616c3e65696d6167656a3c72657175697265643e6b6465736372697074696f6e6a3c6f7074696f6e616c3e` {
		t.Errorf(
			"InvalidReserialization got %s expected %s",
			hex.EncodeToString(marshaled),
			`a11902d1a16b7b706f6c6963795f69647da16d7b706f6c6963795f6e616d657da4646e616d656a3c72657175697265643e64747970656a3c6f7074696f6e616c3e65696d6167656a3c72657175697265643e6b6465736372697074696f6e6a3c6f7074696f6e616c3e`,
		)
	}
	if len(Aux.Hash()) != 32 {
		t.Errorf("Invalid Hashing Of AuxiliaryData")
	}
	if hex.EncodeToString(
		Aux.Hash(),
	) != "9ef720ec820d751e0b7d18534b37a19c2fea055ed49d496b5865d27e8ed34def" {
		t.Errorf(
			"Invalid Hashing Of AuxiliaryData expected %s got %s",
			"9ef720ec820d751e0b7d18534b37a19c2fea055ed49d496b5865d27e8ed34def",
			hex.EncodeToString(Aux.Hash()),
		)
	}
	AlonzoAux := Metadata.AuxiliaryData{}
	alonzoMeta := Metadata.AlonzoMetadata{}
	alonzoMeta.Metadata = meta
	alonzoMeta.NativeScripts = []NativeScript.NativeScript{
		{Tag: NativeScript.ScriptAll},
	}
	AlonzoAux.SetAlonzoMetadata(alonzoMeta)
	marshaled, err = cbor.Marshal(AlonzoAux)
	if err != nil {
		t.Errorf("Error while marshaling")
	}
	if hex.EncodeToString(
		marshaled,
	) != `a200a11902d1a16b7b706f6c6963795f69647da16d7b706f6c6963795f6e616d657da4646e616d656a3c72657175697265643e64747970656a3c6f7074696f6e616c3e65696d6167656a3c72657175697265643e6b6465736372697074696f6e6a3c6f7074696f6e616c3e01818201f6` {
		t.Errorf(
			"InvalidReserialization got %s expected %s",
			hex.EncodeToString(marshaled),
			`a200a11902d1a16b7b706f6c6963795f69647da16d7b706f6c6963795f6e616d657da4646e616d656a3c72657175697265643e64747970656a3c6f7074696f6e616c3e65696d6167656a3c72657175697265643e6b6465736372697074696f6e6a3c6f7074696f6e616c3e01818201f6`,
		)
	}
	if len(AlonzoAux.Hash()) != 32 {
		t.Errorf("Invalid Hashing Of AuxiliaryData")
	}
	if hex.EncodeToString(
		Aux.Hash(),
	) != "9ef720ec820d751e0b7d18534b37a19c2fea055ed49d496b5865d27e8ed34def" {
		t.Errorf(
			"Invalid Hashing Of AuxiliaryData expected %s got %s",
			"9ef720ec820d751e0b7d18534b37a19c2fea055ed49d496b5865d27e8ed34def",
			hex.EncodeToString(Aux.Hash()),
		)
	}
	BasicAux := Metadata.AuxiliaryData{}
	BasicAux.SetBasicMetadata(meta)
	marshaled, err = cbor.Marshal(BasicAux)
	if err != nil {
		t.Errorf("Error while marshaling")
	}
	if hex.EncodeToString(
		marshaled,
	) != `a11902d1a16b7b706f6c6963795f69647da16d7b706f6c6963795f6e616d657da4646e616d656a3c72657175697265643e64747970656a3c6f7074696f6e616c3e65696d6167656a3c72657175697265643e6b6465736372697074696f6e6a3c6f7074696f6e616c3e` {
		t.Errorf(
			"InvalidReserialization got %s expected %s",
			hex.EncodeToString(marshaled),
			`a11902d1a16b7b706f6c6963795f69647da16d7b706f6c6963795f6e616d657da4646e616d656a3c72657175697265643e64747970656a3c6f7074696f6e616c3e65696d6167656a3c72657175697265643e6b6465736372697074696f6e6a3c6f7074696f6e616c3e`,
		)
	}
	if len(BasicAux.Hash()) != 32 {
		t.Errorf("Invalid Hashing Of AuxiliaryData")
	}
	if hex.EncodeToString(
		Aux.Hash(),
	) != "9ef720ec820d751e0b7d18534b37a19c2fea055ed49d496b5865d27e8ed34def" {
		t.Errorf(
			"Invalid Hashing Of AuxiliaryData expected %s got %s",
			"9ef720ec820d751e0b7d18534b37a19c2fea055ed49d496b5865d27e8ed34def",
			hex.EncodeToString(Aux.Hash()),
		)
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
		t.Errorf(
			"InvalidReserialization got %s expected %s",
			hex.EncodeToString(marshaled),
			`f6`,
		)
	}
}

func TestShelleyMaryMetadataFromJSONNoSchema(t *testing.T) {
	jsonData := []byte(`{
		"721": {
			"policy": {
				"Token": {
					"name": "Apollo",
					"serial": 1,
					"raw": "0xdeadbeef"
				}
			}
		}
	}`)
	shelleyMeta, err := Metadata.ShelleyMaryMetadataFromJSON(jsonData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	top, ok := shelleyMeta.Metadata[721].(Metadata.MetadatumMap)
	if !ok {
		t.Fatalf("metadata label 721 has type %T", shelleyMeta.Metadata[721])
	}
	policy := metadataMapValue(t, top, "policy").(Metadata.MetadatumMap)
	token := metadataMapValue(t, policy, "Token").(Metadata.MetadatumMap)
	if got := metadataMapValue(t, token, "name"); got != "Apollo" {
		t.Fatalf("name = %v", got)
	}
	if got := metadataMapValue(t, token, "serial"); got != int64(1) {
		t.Fatalf("serial = %v", got)
	}
	raw, ok := metadataMapValue(t, token, "raw").([]byte)
	if !ok {
		t.Fatalf("raw has unexpected type")
	}
	if !bytes.Equal(raw, []byte{0xde, 0xad, 0xbe, 0xef}) {
		t.Fatalf("raw = %x", raw)
	}
	aux := Metadata.AuxiliaryData{}
	aux.SetShelleyMetadata(shelleyMeta)
	if _, err := cbor.Marshal(aux); err != nil {
		t.Fatalf("metadata did not marshal: %v", err)
	}
}

func TestShelleyMaryMetadataFromJSONNoSchemaMapKeys(t *testing.T) {
	shelleyMeta, err := Metadata.ShelleyMaryMetadataFromJSON(
		[]byte(`{"1":{"42":"number key","01":"text key","0xdead":"bytes key"}}`),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	top := shelleyMeta.Metadata[1].(Metadata.MetadatumMap)
	if got := metadataMapValue(t, top, int64(42)); got != "number key" {
		t.Fatalf("numeric key value = %v", got)
	}
	if got := metadataMapValue(t, top, "01"); got != "text key" {
		t.Fatalf("leading-zero key value = %v", got)
	}
	if got := metadataMapValue(t, top, []byte{0xde, 0xad}); got != "bytes key" {
		t.Fatalf("bytes key value = %v", got)
	}
}

func TestShelleyMaryMetadataFromDetailedJSON(t *testing.T) {
	jsonData := []byte(`{
		"1": {
			"map": [
				{"k": {"string": "name"}, "v": {"string": "Apollo"}},
				{"k": {"bytes": "deadbeef"}, "v": {"list": [{"int": 1}, {"int": -1}]}}
			]
		}
	}`)
	shelleyMeta, err := Metadata.ShelleyMaryMetadataFromJSONWithSchema(
		jsonData,
		Metadata.JSONDetailedSchema,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	top := shelleyMeta.Metadata[1].(Metadata.MetadatumMap)
	if got := metadataMapValue(t, top, "name"); got != "Apollo" {
		t.Fatalf("name = %v", got)
	}
	list, ok := metadataMapValue(t, top, []byte{0xde, 0xad, 0xbe, 0xef}).([]any)
	if !ok {
		t.Fatalf("bytes-key value has unexpected type")
	}
	if !reflect.DeepEqual(list, []any{int64(1), int64(-1)}) {
		t.Fatalf("list = %#v", list)
	}
	aux := Metadata.AuxiliaryData{}
	aux.SetShelleyMetadata(shelleyMeta)
	if _, err := cbor.Marshal(aux); err != nil {
		t.Fatalf("metadata did not marshal: %v", err)
	}
}

func TestShelleyMaryMetadataFromJSONRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name string
		json string
	}{
		{name: "top level array", json: `[]`},
		{name: "bad label", json: `{"01":"x"}`},
		{name: "bool", json: `{"1":true}`},
		{name: "float", json: `{"1":1.2}`},
		{name: "long text", json: `{"1":"` + strings.Repeat("a", 65) + `"}`},
		{name: "long bytes", json: `{"1":"0x` + strings.Repeat("aa", 65) + `"}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Metadata.ShelleyMaryMetadataFromJSON([]byte(test.json))
			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func metadataMapValue(
	t *testing.T,
	metadataMap Metadata.MetadatumMap,
	key any,
) any {
	t.Helper()
	for _, entry := range metadataMap {
		if reflect.DeepEqual(entry.Key, key) {
			return entry.Value
		}
	}
	t.Fatalf("metadata key %v not found", key)
	return nil
}

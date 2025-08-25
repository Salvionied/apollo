package NativeScript_test

import (
	"encoding/hex"
	"log"
	"reflect"
	"testing"

	"github.com/Salvionied/apollo/serialization/NativeScript"
	"github.com/fxamacker/cbor/v2"
)

func TestNativeScriptsSerializationAndHash(t *testing.T) {
	cborHex := "8202838201818200581cbdb17f2e0cc15ba1fc39b149d46a80211ada8c6a839c2e006ed8ef398201818200581c0966fdfb5f72bfc22f4e6b5195b7efb80e023f5204bbf37067049b4a8201818200581c6c70c3fc4f73e5bbb54cc87bdf943fad2213da692b260e12749bcc10"
	nativeScript := new(NativeScript.NativeScript)
	decoded, err := hex.DecodeString(cborHex)
	if err != nil {
		log.Fatal(err)
	}
	err = cbor.Unmarshal(decoded, &nativeScript)
	if err != nil {
		log.Fatal(err)
	}
	result, _ := cbor.Marshal(nativeScript)
	if hex.EncodeToString(result) != cborHex {
		t.Errorf("InvalidReserialization")
	}
	hash, _ := nativeScript.Hash()
	if hex.EncodeToString(hash.Bytes()) != "1d8b26107c604d36e24963be3ba26f264245cae0e10c7fa15846efd2" {
		t.Errorf("Invalid Hashing Of NativeScript")
	}
}

func TestSerializeAndDeserializeAnyType(t *testing.T) {
	pkns := NativeScript.NewScriptPubKey([]byte("test"))
	any := NativeScript.NewScriptAny([]NativeScript.NativeScript{pkns})
	all := NativeScript.NewScriptAll([]NativeScript.NativeScript{any})
	nok := NativeScript.NewScriptNofK([]NativeScript.NativeScript{all}, 1)
	ib := NativeScript.NewInvalidBefore(1)
	ih := NativeScript.NewInvalidHereafter(1)
	pknsBytes, _ := pkns.MarshalCBOR()
	anyBytes, _ := any.MarshalCBOR()
	allBytes, _ := all.MarshalCBOR()
	nokBytes, _ := nok.MarshalCBOR()
	ibBytes, _ := ib.MarshalCBOR()
	ihBytes, _ := ih.MarshalCBOR()
	if hex.EncodeToString(pknsBytes) != "82004474657374" {
		t.Errorf("Invalid serialization of ScriptPubKey %s", hex.EncodeToString(pknsBytes))
	}
	if hex.EncodeToString(anyBytes) != "82028182004474657374" {
		t.Errorf("Invalid serialization of ScriptAny %s", hex.EncodeToString(anyBytes))
	}
	if hex.EncodeToString(allBytes) != "82018182028182004474657374" {
		t.Errorf("Invalid serialization of ScriptAll %s", hex.EncodeToString(allBytes))
	}
	if hex.EncodeToString(nokBytes) != "8303018182018182028182004474657374" {
		t.Errorf("Invalid serialization of ScriptNofK %s", hex.EncodeToString(nokBytes))
	}
	if hex.EncodeToString(ibBytes) != "820401" {
		t.Errorf("Invalid serialization of InvalidBefore %s", hex.EncodeToString(ibBytes))
	}
	if hex.EncodeToString(ihBytes) != "820501" {
		t.Errorf("Invalid serialization of InvalidHereafter %s", hex.EncodeToString(ihBytes))
	}
	pknsDeserialized := NativeScript.NativeScript{}
	anyDeserialized := NativeScript.NativeScript{}
	allDeserialized := NativeScript.NativeScript{}
	nokDeserialized := NativeScript.NativeScript{}
	ibDeserialized := NativeScript.NativeScript{}
	ihDeserialized := NativeScript.NativeScript{}
	err := cbor.Unmarshal(pknsBytes, &pknsDeserialized)
	if err != nil {
		log.Fatal(err)
	}
	err = cbor.Unmarshal(anyBytes, &anyDeserialized)
	if err != nil {
		log.Fatal(err)
	}
	err = cbor.Unmarshal(allBytes, &allDeserialized)
	if err != nil {
		log.Fatal(err)
	}
	err = cbor.Unmarshal(nokBytes, &nokDeserialized)
	if err != nil {
		log.Fatal(err)
	}
	err = cbor.Unmarshal(ibBytes, &ibDeserialized)
	if err != nil {
		log.Fatal(err)
	}
	err = cbor.Unmarshal(ihBytes, &ihDeserialized)
	if err != nil {
		log.Fatal(err)
	}
	if reflect.DeepEqual(pkns, pknsDeserialized) != true {
		t.Errorf("Invalid deserialization of ScriptPubKey")
	}
	if reflect.DeepEqual(any, anyDeserialized) != true {
		t.Errorf("Invalid deserialization of ScriptAny")
	}
	if reflect.DeepEqual(all, allDeserialized) != true {
		t.Errorf("Invalid deserialization of ScriptAll")
	}
	if reflect.DeepEqual(nok, nokDeserialized) != true {
		t.Errorf("Invalid deserialization of ScriptNofK")
	}
	if reflect.DeepEqual(ib, ibDeserialized) != true {
		t.Errorf("Invalid deserialization of InvalidBefore")
	}
	if reflect.DeepEqual(ih, ihDeserialized) != true {
		t.Errorf("Invalid deserialization of InvalidHereafter")
	}

}

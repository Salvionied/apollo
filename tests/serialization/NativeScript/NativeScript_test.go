package serialization_test

import (
	"encoding/hex"
	"fmt"
	"log"
	"testing"

	"github.com/Salvionied/cbor/v2"
	"github.com/SundaeSwap-finance/apollo/serialization/NativeScript"
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
	hash := nativeScript.Hash()
	if fmt.Sprint(hex.EncodeToString(hash.Bytes())) != "1d8b26107c604d36e24963be3ba26f264245cae0e10c7fa15846efd2" {
		t.Errorf("Invalid Hashing Of NativeScript")
	}
}

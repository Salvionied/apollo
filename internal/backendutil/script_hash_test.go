package backendutil

import (
	"bytes"
	"testing"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/common"
)

func TestScriptRefFromHashPreservesMatchingScripts(t *testing.T) {
	nativeBytes := []byte{0x82, 0x04, 0x00}
	var native common.NativeScript
	if _, err := cbor.DecodeExact(nativeBytes, &native); err != nil {
		t.Fatal(err)
	}
	scriptBytes := []byte{0x49, 0x48, 0x01, 0x00}
	for _, tc := range []struct {
		name  string
		hash  common.Blake2b224
		bytes []byte
		typ   uint
	}{
		{"native", native.Hash(), nativeBytes, common.ScriptRefTypeNativeScript},
		{"plutus v1", common.PlutusV1Script(scriptBytes).Hash(), scriptBytes, common.ScriptRefTypePlutusV1},
		{"plutus v2", common.PlutusV2Script(scriptBytes).Hash(), scriptBytes, common.ScriptRefTypePlutusV2},
		{"plutus v3", common.PlutusV3Script(scriptBytes).Hash(), scriptBytes, common.ScriptRefTypePlutusV3},
		{"plutus v4", common.PlutusV4Script(scriptBytes).Hash(), scriptBytes, common.ScriptRefTypePlutusV4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ref, err := ScriptRefFromHash(tc.hash, tc.bytes)
			if err != nil {
				t.Fatal(err)
			}
			if ref.Type != tc.typ || !bytes.Equal(ref.Script.RawScriptBytes(), tc.bytes) {
				t.Fatalf("script ref = %+v, want type %d and bytes %x", ref, tc.typ, tc.bytes)
			}
		})
	}
}

func TestScriptRefFromHashRejectsOtherAndTrailingBytes(t *testing.T) {
	nativeBytes := []byte{0x82, 0x04, 0x00}
	var native common.NativeScript
	if _, err := cbor.DecodeExact(nativeBytes, &native); err != nil {
		t.Fatal(err)
	}
	for _, bytes := range [][]byte{
		{0x82, 0x04, 0x01},
		{0x82, 0x04, 0x00, 0x00},
	} {
		if _, err := ScriptRefFromHash(native.Hash(), bytes); err == nil {
			t.Fatalf("accepted bytes %x under native script hash", bytes)
		}
	}
}

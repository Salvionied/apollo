package blockfrost

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/common"
)

func TestScriptCborRejectsBytesForAnotherHash(t *testing.T) {
	want := common.PlutusV2Script([]byte{0x01, 0x02}).Hash()
	other := []byte{0x03, 0x04}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"cbor": hex.EncodeToString(other)})
	}))
	t.Cleanup(server.Close)
	if _, err := NewBlockFrostChainContext(server.URL, 0, "").ScriptCbor(want); err == nil {
		t.Fatal("expected script hash mismatch")
	}
}

func TestScriptCborAcceptsMatchingNativeAndPlutus(t *testing.T) {
	nativeBytes := []byte{0x82, 0x04, 0x00}
	var native common.NativeScript
	if _, err := cbor.DecodeExact(nativeBytes, &native); err != nil {
		t.Fatal(err)
	}
	plutusBytes := []byte{0x49, 0x48, 0x01, 0x00}
	for _, tc := range []struct {
		name  string
		hash  common.Blake2b224
		bytes []byte
	}{
		{"native", native.Hash(), nativeBytes},
		{"plutus", common.PlutusV2Script(plutusBytes).Hash(), plutusBytes},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_ = json.NewEncoder(w).Encode(map[string]string{"cbor": hex.EncodeToString(tc.bytes)})
			}))
			t.Cleanup(server.Close)
			got, err := NewBlockFrostChainContext(server.URL, 0, "").ScriptCbor(tc.hash)
			if err != nil || !bytes.Equal(got, tc.bytes) {
				t.Fatalf("ScriptCbor() = %x, %v, want %x, nil", got, err, tc.bytes)
			}
		})
	}
}

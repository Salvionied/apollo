package apollo

import (
	"bytes"
	"encoding/hex"
	"slices"
	"testing"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/common"
)

// The Conway native-script CDDL is:
//
//	native_script =
//	  [ script_pubkey      = (0, addr_keyhash)
//	  // script_all         = (1, [* native_script])
//	  // script_any         = (2, [* native_script])
//	  // script_n_of_k      = (3, int64, [* native_script])
//	  // invalid_before     = (4, slot_no)
//	  // invalid_hereafter  = (5, slot_no)
//	  ]
//
// Every expectation below is built from that grammar with the CBOR header bytes
// spelled out, never from Apollo's own output: a constructor emitting the wrong
// leading type tag is a silent authorization change (All becoming Any drops a
// multisig to a single signer) and must not be able to define its own oracle.

// nativeScriptKeyHash returns a 28-byte addr_keyhash of a repeated byte.
func nativeScriptKeyHash(fill byte) common.Blake2b224 {
	var hash common.Blake2b224
	copy(hash[:], bytes.Repeat([]byte{fill}, len(hash)))
	return hash
}

// cddlPubkey encodes (0, addr_keyhash): array(2), uint(0), bstr(28).
func cddlPubkey(keyHash common.Blake2b224) []byte {
	return slices.Concat([]byte{0x82, 0x00, 0x58, 0x1c}, keyHash.Bytes())
}

// cddlScriptList encodes (tag, [* native_script]) over two sub-scripts, for
// tag 1 (all) and 2 (any): array(2), uint(tag), array(2), scripts...
func cddlScriptList(tag byte, first, second []byte) []byte {
	return slices.Concat([]byte{0x82, tag, 0x82}, first, second)
}

// cddlNofK encodes (3, int64, [* native_script]) over two sub-scripts:
// array(3), uint(3), uint(n), array(2), scripts...
func cddlNofK(n byte, first, second []byte) []byte {
	return slices.Concat([]byte{0x83, 0x03, n, 0x82}, first, second)
}

// nativeScriptCbor encodes a script the way Apollo's callers serialize it.
func nativeScriptCbor(t *testing.T, ns common.NativeScript) []byte {
	t.Helper()
	scriptCbor, err := cbor.Encode(&ns)
	if err != nil {
		t.Fatal(err)
	}
	return scriptCbor
}

// nativeScriptTypeTag reads the leading integer of a native script. Native
// scripts are [type, ...] arrays, so this is the authorization semantics of the
// script, decoded without reference to any Apollo type.
func nativeScriptTypeTag(t *testing.T, scriptCbor []byte) uint64 {
	t.Helper()
	var elements []cbor.RawMessage
	if _, err := cbor.Decode(scriptCbor, &elements); err != nil {
		t.Fatalf("native script is not a CBOR array: %v", err)
	}
	if len(elements) < 2 {
		t.Fatalf("native script has %d elements, want at least 2",
			len(elements))
	}
	var tag uint64
	if _, err := cbor.Decode(elements[0], &tag); err != nil {
		t.Fatalf("native script type tag is not an integer: %v", err)
	}
	return tag
}

// cddlScriptHash derives the script hash of the expected bytes: native script
// hashes are blake2b-224 over a 0x00 language tag followed by the script CBOR.
func cddlScriptHash(scriptCbor []byte) common.ScriptHash {
	return common.ScriptHash(
		common.Blake2b224Hash(slices.Concat([]byte{0x00}, scriptCbor)),
	)
}

func TestNativeScriptConstructorsMatchCddl(t *testing.T) {
	keyHashA := nativeScriptKeyHash(0xaa)
	keyHashB := nativeScriptKeyHash(0xbb)
	pubkeyA := cddlPubkey(keyHashA)
	pubkeyB := cddlPubkey(keyHashB)

	scriptA, err := NewNativeScriptPubkey(keyHashA)
	if err != nil {
		t.Fatal(err)
	}
	scriptB, err := NewNativeScriptPubkey(keyHashB)
	if err != nil {
		t.Fatal(err)
	}
	members := []common.NativeScript{scriptA, scriptB}

	tests := []struct {
		name    string
		build   func() (common.NativeScript, error)
		wantTag uint64
		want    []byte
	}{
		{
			name:    "pubkey",
			build:   func() (common.NativeScript, error) { return scriptA, nil },
			wantTag: 0,
			want:    pubkeyA,
		},
		{
			name: "all",
			build: func() (common.NativeScript, error) {
				return NewNativeScriptAll(members)
			},
			wantTag: 1,
			want:    cddlScriptList(0x01, pubkeyA, pubkeyB),
		},
		{
			name: "any",
			build: func() (common.NativeScript, error) {
				return NewNativeScriptAny(members)
			},
			wantTag: 2,
			want:    cddlScriptList(0x02, pubkeyA, pubkeyB),
		},
		{
			name: "nofk",
			build: func() (common.NativeScript, error) {
				return NewNativeScriptNofK(2, members)
			},
			wantTag: 3,
			want:    cddlNofK(0x02, pubkeyA, pubkeyB),
		},
		{
			name: "invalid_before",
			build: func() (common.NativeScript, error) {
				return NewNativeScriptInvalidBefore(1000)
			},
			wantTag: 4,
			// array(2), uint(4), uint16(1000)
			want: []byte{0x82, 0x04, 0x19, 0x03, 0xe8},
		},
		{
			name: "invalid_hereafter",
			build: func() (common.NativeScript, error) {
				return NewNativeScriptInvalidHereafter(2000)
			},
			wantTag: 5,
			// array(2), uint(5), uint16(2000)
			want: []byte{0x82, 0x05, 0x19, 0x07, 0xd0},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ns, err := test.build()
			if err != nil {
				t.Fatal(err)
			}
			got := nativeScriptCbor(t, ns)
			if !bytes.Equal(got, test.want) {
				t.Errorf("CBOR mismatch:\n got %s\nwant %s",
					hex.EncodeToString(got), hex.EncodeToString(test.want))
			}
			if tag := nativeScriptTypeTag(t, got); tag != test.wantTag {
				t.Errorf("type tag %d, want %d", tag, test.wantTag)
			}
			if hash := ns.Hash(); hash != cddlScriptHash(test.want) {
				t.Errorf("script hash %s, want %s",
					hash, cddlScriptHash(test.want))
			}
		})
	}
}

// TestNativeScriptAllIsNotAny pins the pair of constructors that are byte-wise
// identical apart from the type tag. Emitting Any where All was asked for turns
// an n-of-n multisig into a 1-of-n, so the two must never collide.
func TestNativeScriptAllIsNotAny(t *testing.T) {
	scriptA, err := NewNativeScriptPubkey(nativeScriptKeyHash(0x01))
	if err != nil {
		t.Fatal(err)
	}
	scriptB, err := NewNativeScriptPubkey(nativeScriptKeyHash(0x02))
	if err != nil {
		t.Fatal(err)
	}
	members := []common.NativeScript{scriptA, scriptB}

	all, err := NewNativeScriptAll(members)
	if err != nil {
		t.Fatal(err)
	}
	anyOf, err := NewNativeScriptAny(members)
	if err != nil {
		t.Fatal(err)
	}
	allCbor := nativeScriptCbor(t, all)
	anyCbor := nativeScriptCbor(t, anyOf)

	if bytes.Equal(allCbor, anyCbor) {
		t.Fatalf("all and any encode identically: %s",
			hex.EncodeToString(allCbor))
	}
	if all.Hash() == anyOf.Hash() {
		t.Fatalf("all and any share script hash %s", all.Hash())
	}
	if tag := nativeScriptTypeTag(t, allCbor); tag != 1 {
		t.Errorf("all type tag %d, want 1", tag)
	}
	if tag := nativeScriptTypeTag(t, anyCbor); tag != 2 {
		t.Errorf("any type tag %d, want 2", tag)
	}
}

// TestNativeScriptTimelocksAreNotInterchangeable pins the other confusable
// pair: swapping invalid_before and invalid_hereafter inverts a timelock, so
// the same slot must yield different scripts under the two constructors.
func TestNativeScriptTimelocksAreNotInterchangeable(t *testing.T) {
	const slot = 42

	before, err := NewNativeScriptInvalidBefore(slot)
	if err != nil {
		t.Fatal(err)
	}
	hereafter, err := NewNativeScriptInvalidHereafter(slot)
	if err != nil {
		t.Fatal(err)
	}
	beforeCbor := nativeScriptCbor(t, before)
	hereafterCbor := nativeScriptCbor(t, hereafter)

	// array(2), uint(4|5), uint(42)
	if want := []byte{0x82, 0x04, 0x18, slot}; !bytes.Equal(
		beforeCbor, want,
	) {
		t.Errorf("invalid_before CBOR %s, want %s",
			hex.EncodeToString(beforeCbor), hex.EncodeToString(want))
	}
	if want := []byte{0x82, 0x05, 0x18, slot}; !bytes.Equal(
		hereafterCbor, want,
	) {
		t.Errorf("invalid_hereafter CBOR %s, want %s",
			hex.EncodeToString(hereafterCbor), hex.EncodeToString(want))
	}
	if before.Hash() == hereafter.Hash() {
		t.Fatalf("timelocks share script hash %s", before.Hash())
	}
}

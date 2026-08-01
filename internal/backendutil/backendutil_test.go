package backendutil

import (
	"encoding/hex"
	"testing"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/common"
)

func TestParseFractionValid(t *testing.T) {
	val, err := ParseFraction("1/2")
	if err != nil {
		t.Fatal(err)
	}
	if val != 0.5 {
		t.Errorf("expected 0.5, got %f", val)
	}
}

func TestParseRedeemerTagConwayPurposes(t *testing.T) {
	tests := map[string]common.RedeemerTag{
		"vote": common.RedeemerTagVoting, "voting": common.RedeemerTagVoting,
		"propose": common.RedeemerTagProposing, "proposing": common.RedeemerTagProposing,
		"guard": common.RedeemerTagGuarding, "guarding": common.RedeemerTagGuarding,
	}
	for input, want := range tests {
		t.Run(input, func(t *testing.T) {
			got, err := ParseRedeemerTag(input)
			if err != nil || got != want {
				t.Fatalf("ParseRedeemerTag(%q) = %v, %v; want %v, nil", input, got, err, want)
			}
		})
	}
}

func TestParseFractionPlainNumber(t *testing.T) {
	val, err := ParseFraction("0.0577")
	if err != nil {
		t.Fatal(err)
	}
	if val < 0.0576 || val > 0.0578 {
		t.Errorf("expected ~0.0577, got %f", val)
	}
}

func TestParseFractionInvalidNumerator(t *testing.T) {
	_, err := ParseFraction("abc/100")
	if err == nil {
		t.Error("expected error for invalid numerator")
	}
}

func TestParseFractionInvalidDenominator(t *testing.T) {
	_, err := ParseFraction("1/xyz")
	if err == nil {
		t.Error("expected error for invalid denominator")
	}
}

func TestParseFractionDivisionByZero(t *testing.T) {
	_, err := ParseFraction("1/0")
	if err == nil {
		t.Error("expected error for division by zero")
	}
}

func TestParseFractionInvalidString(t *testing.T) {
	_, err := ParseFraction("not-a-number")
	if err == nil {
		t.Error("expected error for invalid string")
	}
}

func TestParseAssetUnit(t *testing.T) {
	policyHex := "00000000000000000000000000000000000000000000000000000001"
	assetNameHex := hex.EncodeToString([]byte("TOKEN"))
	policyID, assetName, err := ParseAssetUnit(policyHex + assetNameHex)
	if err != nil {
		t.Fatal(err)
	}
	if got := hex.EncodeToString(policyID.Bytes()); got != policyHex {
		t.Fatalf("policy ID = %s, want %s", got, policyHex)
	}
	if want := cbor.NewByteString([]byte("TOKEN")); assetName != want {
		t.Fatalf("asset name = %s, want %s", assetName.String(), want.String())
	}
}

func TestParseAssetUnitRejectsInvalidInput(t *testing.T) {
	tests := []string{
		"abcd",
		"0000000000000000000000000000000000000000000000000000000z",
		"00000000000000000000000000000000000000000000000000000001zz",
		"00000000000000000000000000000000000000000000000000000001" +
			"000000000000000000000000000000000000000000000000000000000000000000",
	}
	for _, unit := range tests {
		t.Run(unit, func(t *testing.T) {
			if _, _, err := ParseAssetUnit(unit); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestParseAssetUnitAllowsEmptyAssetName(t *testing.T) {
	policyHex := "00000000000000000000000000000000000000000000000000000001"
	policyID, assetName, err := ParseAssetUnit(policyHex)
	if err != nil {
		t.Fatal(err)
	}
	var expected common.Blake2b224
	expected[27] = 1
	if policyID != expected {
		t.Fatalf("policy ID = %x, want %x", policyID.Bytes(), expected.Bytes())
	}
	if want := cbor.NewByteString(nil); assetName != want {
		t.Fatalf("asset name = %s, want empty", assetName.String())
	}
}

func TestBoundedInt(t *testing.T) {
	if v, err := BoundedInt(12345, "x"); err != nil || v != 12345 {
		t.Errorf("BoundedInt(12345) = %d, %v; want 12345, nil", v, err)
	}
	if _, err := BoundedInt(-1, "x"); err == nil {
		t.Error("BoundedInt(-1) should error")
	}
	if _, err := BoundedInt(int64(1)<<32, "x"); err == nil {
		t.Error("BoundedInt(2^32) should error")
	}
}

func TestBoundedIntFromUint64(t *testing.T) {
	if v, err := BoundedIntFromUint64(12345, "x"); err != nil || v != 12345 {
		t.Errorf("BoundedIntFromUint64(12345) = %d, %v; want 12345, nil", v, err)
	}
	if _, err := BoundedIntFromUint64(uint64(1)<<32, "x"); err == nil {
		t.Error("BoundedIntFromUint64(2^32) should error")
	}
}

func TestScriptRefFromBytesVerifiesHash(t *testing.T) {
	script := common.PlutusV2Script([]byte{0x01, 0x02, 0x03})
	correctHash := hex.EncodeToString(script.Hash().Bytes())

	ref, err := ScriptRefFromBytes(common.ScriptRefTypePlutusV2, script, correctHash)
	if err != nil {
		t.Fatal(err)
	}
	if ref.Type != common.ScriptRefTypePlutusV2 {
		t.Fatalf("script ref type = %d, want %d", ref.Type, common.ScriptRefTypePlutusV2)
	}
	if _, ok := ref.Script.(common.PlutusV2Script); !ok {
		t.Fatalf("expected PlutusV2 script, got %T", ref.Script)
	}
}

func TestScriptRefFromBytesPlutusV4(t *testing.T) {
	script := common.PlutusV4Script([]byte{0x01, 0x02, 0x03})
	correctHash := hex.EncodeToString(script.Hash().Bytes())

	ref, err := ScriptRefFromBytes(common.ScriptRefTypePlutusV4, script, correctHash)
	if err != nil {
		t.Fatal(err)
	}
	if ref.Type != common.ScriptRefTypePlutusV4 {
		t.Fatalf("script ref type = %d, want %d", ref.Type, common.ScriptRefTypePlutusV4)
	}
	if _, ok := ref.Script.(common.PlutusV4Script); !ok {
		t.Fatalf("expected PlutusV4 script, got %T", ref.Script)
	}
}

func TestScriptRefFromBytesRejectsHashMismatch(t *testing.T) {
	script := common.PlutusV2Script([]byte{0x01, 0x02, 0x03})
	// The same bytes hashed as PlutusV1 produce a different script hash.
	wrongHash := hex.EncodeToString(common.PlutusV1Script(script).Hash().Bytes())
	if _, err := ScriptRefFromBytes(common.ScriptRefTypePlutusV2, script, wrongHash); err == nil {
		t.Fatal("expected script hash mismatch error")
	}
}

func TestScriptRefFromBytesRejectsClaimedLanguageMismatch(t *testing.T) {
	// Provider claims PlutusV1 for bytes whose hash was computed as PlutusV2.
	scriptBytes := []byte{0x01, 0x02, 0x03}
	v2Hash := hex.EncodeToString(common.PlutusV2Script(scriptBytes).Hash().Bytes())
	if _, err := ScriptRefFromBytes(common.ScriptRefTypePlutusV1, scriptBytes, v2Hash); err == nil {
		t.Fatal("expected script hash mismatch error for wrong language claim")
	}
}

func TestScriptRefFromBytesSkipsVerificationWithoutHash(t *testing.T) {
	ref, err := ScriptRefFromBytes(common.ScriptRefTypePlutusV3, []byte{0x0a}, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := ref.Script.(common.PlutusV3Script); !ok {
		t.Fatalf("expected PlutusV3 script, got %T", ref.Script)
	}
}

func TestScriptRefFromBytesRejectsInvalidHashHex(t *testing.T) {
	if _, err := ScriptRefFromBytes(common.ScriptRefTypePlutusV2, []byte{0x01}, "zz"); err == nil {
		t.Fatal("expected invalid hash hex error")
	}
	if _, err := ScriptRefFromBytes(common.ScriptRefTypePlutusV2, []byte{0x01}, "abcd"); err == nil {
		t.Fatal("expected invalid hash length error")
	}
}

func TestScriptRefFromBytesRejectsUnsupportedType(t *testing.T) {
	if _, err := ScriptRefFromBytes(99, []byte{0x01}, ""); err == nil {
		t.Fatal("expected unsupported script ref type error")
	}
}

func TestScriptRefFromBytesNativeScript(t *testing.T) {
	// Native script: ScriptPubkey = [0, key_hash]
	keyHash := make([]byte, 28)
	keyHash[0] = 0xAA
	scriptCbor, err := cbor.Encode([]any{0, keyHash})
	if err != nil {
		t.Fatal(err)
	}
	ref, err := ScriptRefFromBytes(common.ScriptRefTypeNativeScript, scriptCbor, "")
	if err != nil {
		t.Fatal(err)
	}
	native, ok := ref.Script.(common.NativeScript)
	if !ok {
		t.Fatalf("expected native script, got %T", ref.Script)
	}

	// Round-trip the computed hash through verification.
	correctHash := hex.EncodeToString(native.Hash().Bytes())
	if _, err := ScriptRefFromBytes(common.ScriptRefTypeNativeScript, scriptCbor, correctHash); err != nil {
		t.Fatalf("expected native script hash to verify: %v", err)
	}
	wrongHash := hex.EncodeToString(make([]byte, common.Blake2b224Size))
	if _, err := ScriptRefFromBytes(common.ScriptRefTypeNativeScript, scriptCbor, wrongHash); err == nil {
		t.Fatal("expected native script hash mismatch error")
	}
}

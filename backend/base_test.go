package backend

import (
	"encoding/hex"
	"testing"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/common"
)

func TestCoinsPerUtxoByteValueDefault(t *testing.T) {
	pp := ProtocolParameters{}
	val := pp.CoinsPerUtxoByteValue()
	if val != 4310 {
		t.Errorf("expected default 4310, got %d", val)
	}
}

func TestCoinsPerUtxoByteValueFromString(t *testing.T) {
	pp := ProtocolParameters{CoinsPerUtxoByte: "4310"}
	val := pp.CoinsPerUtxoByteValue()
	if val != 4310 {
		t.Errorf("expected 4310, got %d", val)
	}
}

func TestCoinsPerUtxoByteValueCustom(t *testing.T) {
	pp := ProtocolParameters{CoinsPerUtxoByte: "8620"}
	val := pp.CoinsPerUtxoByteValue()
	if val != 8620 {
		t.Errorf("expected 8620, got %d", val)
	}
}

func TestCoinsPerUtxoByteValueInvalid(t *testing.T) {
	pp := ProtocolParameters{CoinsPerUtxoByte: "not-a-number"}
	val := pp.CoinsPerUtxoByteValue()
	if val != 4310 {
		t.Errorf("expected fallback 4310, got %d", val)
	}
}

func TestProtocolParametersStruct(t *testing.T) {
	pp := ProtocolParameters{
		MinFeeConstant:    155381,
		MinFeeCoefficient: 44,
		MaxTxSize:         16384,
		CoinsPerUtxoByte:  "4310",
	}
	if pp.MinFeeConstant != 155381 {
		t.Errorf("expected 155381, got %d", pp.MinFeeConstant)
	}
	if pp.MinFeeCoefficient != 44 {
		t.Errorf("expected 44, got %d", pp.MinFeeCoefficient)
	}
	if pp.MaxTxSize != 16384 {
		t.Errorf("expected 16384, got %d", pp.MaxTxSize)
	}
}

func TestParseFractionValid(t *testing.T) {
	val, err := ParseFraction("1/2")
	if err != nil {
		t.Fatal(err)
	}
	if val != 0.5 {
		t.Errorf("expected 0.5, got %f", val)
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

func TestGenesisParametersStruct(t *testing.T) {
	gp := GenesisParameters{
		NetworkMagic: 764824073,
		EpochLength:  432000,
	}
	if gp.NetworkMagic != 764824073 {
		t.Errorf("expected 764824073, got %d", gp.NetworkMagic)
	}
	if gp.EpochLength != 432000 {
		t.Errorf("expected 432000, got %d", gp.EpochLength)
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

func TestCoinsPerUtxoByteValueRejectsOutOfRange(t *testing.T) {
	pp := ProtocolParameters{CoinsPerUtxoByte: "-4310"}
	if v := pp.CoinsPerUtxoByteValue(); v != 4310 {
		t.Errorf("negative value should fall back to 4310, got %d", v)
	}
	pp = ProtocolParameters{CoinsPerUtxoByte: "9223372036854775807"}
	if v := pp.CoinsPerUtxoByteValue(); v != 4310 {
		t.Errorf("oversized value should fall back to 4310, got %d", v)
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

func TestComputeMaxTxFeeOverflow(t *testing.T) {
	pp := ProtocolParameters{
		MaxTxSize:         1 << 30,
		MinFeeCoefficient: int64(1) << 40,
		MinFeeConstant:    155381,
	}
	if _, err := ComputeMaxTxFee(pp); err == nil {
		t.Error("expected overflow error, got nil")
	}
	pp = ProtocolParameters{MaxTxSize: 16384, MinFeeCoefficient: 44, MinFeeConstant: 155381}
	fee, err := ComputeMaxTxFee(pp)
	if err != nil || fee != 16384*44+155381 {
		t.Errorf("ComputeMaxTxFee = %d, %v; want %d, nil", fee, err, 16384*44+155381)
	}
}

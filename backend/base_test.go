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

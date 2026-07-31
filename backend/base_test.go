package backend

import "testing"

type nilChainContext struct {
	ChainContext
}

func TestCapabilitiesOfNilContext(t *testing.T) {
	if got := CapabilitiesOf(nil); got != 0 {
		t.Fatalf("CapabilitiesOf(nil) = %d, want 0", got)
	}

	var ctx *nilChainContext
	if got := CapabilitiesOf(ctx); got != 0 {
		t.Fatalf("CapabilitiesOf(typed nil) = %d, want 0", got)
	}
	if Supports(ctx, CapabilityProtocolParams) {
		t.Fatal("Supports reports a capability for a typed nil context")
	}
}

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

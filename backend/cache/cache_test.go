package cache

import (
	"math/big"
	"testing"
	"time"

	"github.com/Salvionied/apollo/v2/backend"
	"github.com/Salvionied/apollo/v2/backend/fixed"
)

func TestCapabilitiesMatchWrappedContext(t *testing.T) {
	ctx := NewCachedChainContext(fixed.NewEmptyFixedChainContext(), 0)
	if !backend.Supports(ctx, backend.CapabilityProtocolParams|backend.CapabilityUtxoByRef) {
		t.Fatal("cache did not preserve supported capabilities")
	}
	if backend.Supports(ctx, backend.CapabilitySubmitTx) {
		t.Fatal("cache reported unsupported wrapped capability")
	}
}

func TestProtocolParamsPricesDoNotShareCacheState(t *testing.T) {
	pp := backend.ProtocolParameters{
		MinFeeRefScriptCostPerByteRational:       big.NewRat(121, 8),
		MinFeeReferenceScriptsMultiplierRational: big.NewRat(2469, 2000),
	}
	ctx := NewCachedChainContext(fixed.NewFixedChainContext(pp, backend.GenesisParameters{}, 0), time.Hour)
	for i := 0; i < 2; i++ {
		got, err := ctx.ProtocolParams()
		if err != nil {
			t.Fatal(err)
		}
		if got.MinFeeRefScriptCostPerByteRational.Cmp(big.NewRat(121, 8)) != 0 ||
			got.MinFeeReferenceScriptsMultiplierRational.Cmp(big.NewRat(2469, 2000)) != 0 {
			t.Fatalf("cached prices changed after return %d", i)
		}
		got.MinFeeRefScriptCostPerByteRational.SetInt64(1000)
		got.MinFeeReferenceScriptsMultiplierRational.SetInt64(1000)
	}
	got, err := ctx.ProtocolParams()
	if err != nil {
		t.Fatal(err)
	}
	if got.MinFeeRefScriptCostPerByteRational.Cmp(big.NewRat(121, 8)) != 0 ||
		got.MinFeeReferenceScriptsMultiplierRational.Cmp(big.NewRat(2469, 2000)) != 0 {
		t.Fatal("cached prices changed after caller mutation")
	}
}

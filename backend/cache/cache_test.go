package cache

import (
	"testing"

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

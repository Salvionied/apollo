package fixed

import (
	"errors"
	"testing"

	"github.com/blinklabs-io/gouroboros/ledger/common"

	"github.com/Salvionied/apollo/v2/backend"
)

func TestFixedCapabilitiesAndUnsupportedOperations(t *testing.T) {
	ctx := NewEmptyFixedChainContext()
	if !backend.Supports(ctx, backend.CapabilityProtocolParams|backend.CapabilityUtxoByRef) {
		t.Fatal("expected fixed context capabilities to be reported")
	}
	if backend.Supports(ctx, backend.CapabilityCurrentEpoch|backend.CapabilityTip|backend.CapabilitySubmitTx|backend.CapabilityEvaluateTx) {
		t.Fatal("fixed context must not report chain-node capabilities")
	}

	for _, call := range []struct {
		name string
		call func() (uint64, error)
	}{
		{"current epoch", ctx.CurrentEpoch},
		{"tip", ctx.Tip},
	} {
		t.Run(call.name, func(t *testing.T) {
			value, err := call.call()
			if err != nil {
				t.Fatalf("expected legacy zero-value result, got %v", err)
			}
			if value != 0 {
				t.Fatalf("expected legacy zero-value result, got %d", value)
			}
		})
	}

	tests := []struct {
		name       string
		capability backend.Capability
		call       func() error
	}{
		{"submit", backend.CapabilitySubmitTx, func() error { _, err := ctx.SubmitTx(nil); return err }},
		{"evaluate", backend.CapabilityEvaluateTx, func() error { _, err := ctx.EvaluateTx(nil, nil); return err }},
		{"script", backend.CapabilityScriptCbor, func() error { _, err := ctx.ScriptCbor(common.Blake2b224{}); return err }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.call()
			if !errors.Is(err, backend.ErrUnsupported) {
				t.Fatalf("expected ErrUnsupported, got %v", err)
			}
			var unsupported *backend.UnsupportedError
			if !errors.As(err, &unsupported) || unsupported.Capability != test.capability {
				t.Fatalf("unexpected unsupported error: %#v", err)
			}
		})
	}
}

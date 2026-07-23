package backend

import (
	"context"
	"errors"
	"testing"

	"github.com/blinklabs-io/gouroboros/ledger/common"
)

type trackingLegacyContext struct {
	legacyChainContext
	protocolCalls int
}

func (c *trackingLegacyContext) ProtocolParams() (ProtocolParameters, error) {
	c.protocolCalls++
	return ProtocolParameters{MaxTxSize: 1}, nil
}

type trackingContextChain struct {
	trackingLegacyContext
	received context.Context
}

func (c *trackingContextChain) ProtocolParamsContext(ctx context.Context) (ProtocolParameters, error) {
	c.received = ctx
	return ProtocolParameters{MaxTxSize: 2}, ctx.Err()
}

func (c *trackingContextChain) GenesisParamsContext(context.Context) (GenesisParameters, error) {
	return GenesisParameters{}, nil
}

func (c *trackingContextChain) CurrentEpochContext(context.Context) (uint64, error) {
	return 0, nil
}

func (c *trackingContextChain) MaxTxFeeContext(context.Context) (uint64, error) {
	return 0, nil
}

func (c *trackingContextChain) TipContext(context.Context) (uint64, error) {
	return 0, nil
}

func (c *trackingContextChain) UtxosContext(context.Context, common.Address) ([]common.Utxo, error) {
	return nil, nil
}

func (c *trackingContextChain) SubmitTxContext(context.Context, []byte) (common.Blake2b256, error) {
	return common.Blake2b256{}, nil
}

func (c *trackingContextChain) EvaluateTxContext(
	context.Context,
	[]byte,
	[]common.Utxo,
) (map[common.RedeemerKey]common.ExUnits, error) {
	return nil, nil
}

func (c *trackingContextChain) UtxoByRefContext(
	context.Context,
	common.Blake2b256,
	uint32,
) (*common.Utxo, error) {
	return nil, nil
}

func (c *trackingContextChain) ScriptCborContext(
	context.Context,
	common.Blake2b224,
) ([]byte, error) {
	return nil, nil
}

func TestProtocolParamsContextDispatchesCancellation(t *testing.T) {
	chainContext := &trackingContextChain{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := ProtocolParamsContext(ctx, chainContext)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("ProtocolParamsContext() error = %v, want context.Canceled", err)
	}
	if chainContext.received != ctx {
		t.Fatal("context-aware implementation did not receive the caller context")
	}
	if chainContext.protocolCalls != 0 {
		t.Fatalf("legacy ProtocolParams called %d times, want 0", chainContext.protocolCalls)
	}
}

func TestProtocolParamsContextFallsBackToLegacyContext(t *testing.T) {
	chainContext := &trackingLegacyContext{}

	params, err := ProtocolParamsContext(context.Background(), chainContext)
	if err != nil {
		t.Fatalf("ProtocolParamsContext() error = %v", err)
	}
	if params.MaxTxSize != 1 {
		t.Fatalf("ProtocolParamsContext() MaxTxSize = %d, want 1", params.MaxTxSize)
	}
	if chainContext.protocolCalls != 1 {
		t.Fatalf("legacy ProtocolParams called %d times, want 1", chainContext.protocolCalls)
	}
}

func TestProtocolParamsContextNormalizesNilContext(t *testing.T) {
	chainContext := &trackingContextChain{}

	if _, err := ProtocolParamsContext(nil, chainContext); err != nil {
		t.Fatalf("ProtocolParamsContext(nil) error = %v", err)
	}
	if chainContext.received == nil {
		t.Fatal("context-aware implementation received a nil context")
	}
}

func TestContextHelpersRejectNilChainContext(t *testing.T) {
	if _, err := SubmitTxContext(context.Background(), nil, nil); err == nil {
		t.Fatal("SubmitTxContext(nil chain context) returned nil error")
	}

	var typedNil *trackingContextChain
	if _, err := UtxoByRefContext(context.Background(), typedNil, common.Blake2b256{}, 0); err == nil {
		t.Fatal("UtxoByRefContext(typed nil chain context) returned nil error")
	}
}

func TestBindContextDispatchesThroughHistoricInterface(t *testing.T) {
	chainContext := &trackingContextChain{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := BindContext(ctx, chainContext).ProtocolParams()
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("bound ProtocolParams() error = %v, want context.Canceled", err)
	}
}

var _ ContextChainContext = (*trackingContextChain)(nil)

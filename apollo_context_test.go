package apollo

import (
	"context"
	"encoding/hex"
	"errors"
	"strings"
	"testing"

	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/conway"

	"github.com/Salvionied/apollo/v2/backend"
)

type contextTrackingChain struct {
	backend.ChainContext
	received context.Context
}

func (c *contextTrackingChain) ProtocolParamsContext(context.Context) (backend.ProtocolParameters, error) {
	return backend.ProtocolParameters{}, nil
}

func (c *contextTrackingChain) GenesisParamsContext(context.Context) (backend.GenesisParameters, error) {
	return backend.GenesisParameters{}, nil
}

func (c *contextTrackingChain) CurrentEpochContext(context.Context) (uint64, error) {
	return 0, nil
}

func (c *contextTrackingChain) MaxTxFeeContext(context.Context) (uint64, error) {
	return 0, nil
}

func (c *contextTrackingChain) TipContext(context.Context) (uint64, error) {
	return 0, nil
}

func (c *contextTrackingChain) UtxosContext(context.Context, common.Address) ([]common.Utxo, error) {
	return nil, nil
}

func (c *contextTrackingChain) SubmitTxContext(context.Context, []byte) (common.Blake2b256, error) {
	return common.Blake2b256{}, nil
}

func (c *contextTrackingChain) EvaluateTxContext(
	context.Context,
	[]byte,
	[]common.Utxo,
) (map[common.RedeemerKey]common.ExUnits, error) {
	return nil, nil
}

func (c *contextTrackingChain) UtxoByRefContext(
	ctx context.Context,
	_ common.Blake2b256,
	_ uint32,
) (*common.Utxo, error) {
	c.received = ctx
	return nil, ctx.Err()
}

func (c *contextTrackingChain) ScriptCborContext(context.Context, common.Blake2b224) ([]byte, error) {
	return nil, nil
}

func TestApolloWithContextPropagatesCancellation(t *testing.T) {
	chainContext := &contextTrackingChain{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := New(chainContext).
		WithContext(ctx).
		UtxoFromRef(hex.EncodeToString(make([]byte, common.Blake2b256Size)), 0)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("UtxoFromRef() error = %v, want context.Canceled", err)
	}
	if chainContext.received != ctx {
		t.Fatal("context-aware backend did not receive the Apollo context")
	}
}

func TestApolloWithContextNormalizesNil(t *testing.T) {
	chainContext := &contextTrackingChain{}

	if _, err := New(chainContext).
		WithContext(nil).
		UtxoFromRef(hex.EncodeToString(make([]byte, common.Blake2b256Size)), 0); err != nil {
		t.Fatalf("UtxoFromRef() error = %v", err)
	}
	if chainContext.received == nil {
		t.Fatal("context-aware backend received a nil context")
	}
}

func TestApolloLookupRejectsNilChainContext(t *testing.T) {
	txHash := hex.EncodeToString(make([]byte, common.Blake2b256Size))
	if _, err := New(nil).UtxoFromRef(txHash, 0); err == nil {
		t.Fatal("UtxoFromRef() with nil chain context returned nil error")
	}

	var typedNil *contextTrackingChain
	if _, err := New(typedNil).UtxoFromRef(txHash, 0); err == nil {
		t.Fatal("UtxoFromRef() with typed nil chain context returned nil error")
	}
}

func TestApolloSubmitRejectsNilChainContext(t *testing.T) {
	builder := New(nil)
	builder.tx = &conway.ConwayTransaction{}

	if _, err := builder.Submit(); err == nil || !strings.Contains(err.Error(), "chain context is nil") {
		t.Fatalf("Submit() error = %v, want nil chain context error", err)
	}
}

var _ backend.ContextChainContext = (*contextTrackingChain)(nil)

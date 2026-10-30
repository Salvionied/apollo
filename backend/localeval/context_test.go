package localeval

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/blinklabs-io/gouroboros/ledger/common"

	"github.com/Salvionied/apollo/v2/backend"
	"github.com/Salvionied/apollo/v2/backend/fixed"
)

type ctxKey struct{}

// contextAwareFixed makes a fixed context implement ContextChainContext and
// records the contexts and UTxO lookups the wrapper passes to it.
type contextAwareFixed struct {
	*fixed.FixedChainContext
	lookups  int
	contexts []context.Context
}

var _ backend.ContextChainContext = (*contextAwareFixed)(nil)

func (c *contextAwareFixed) seen(ctx context.Context) { c.contexts = append(c.contexts, ctx) }

func (c *contextAwareFixed) ProtocolParamsContext(ctx context.Context) (backend.ProtocolParameters, error) {
	c.seen(ctx)
	return c.ProtocolParams()
}

func (c *contextAwareFixed) GenesisParamsContext(ctx context.Context) (backend.GenesisParameters, error) {
	c.seen(ctx)
	return c.GenesisParams()
}

func (c *contextAwareFixed) CurrentEpochContext(ctx context.Context) (uint64, error) {
	c.seen(ctx)
	return c.CurrentEpoch()
}

func (c *contextAwareFixed) MaxTxFeeContext(ctx context.Context) (uint64, error) {
	c.seen(ctx)
	return c.MaxTxFee()
}

func (c *contextAwareFixed) TipContext(ctx context.Context) (uint64, error) {
	c.seen(ctx)
	return c.Tip()
}

func (c *contextAwareFixed) UtxosContext(ctx context.Context, address common.Address) ([]common.Utxo, error) {
	c.seen(ctx)
	return c.Utxos(address)
}

func (c *contextAwareFixed) SubmitTxContext(ctx context.Context, txCbor []byte) (common.Blake2b256, error) {
	c.seen(ctx)
	return c.SubmitTx(txCbor)
}

func (c *contextAwareFixed) EvaluateTxContext(ctx context.Context, txCbor []byte, utxos []common.Utxo) (map[common.RedeemerKey]common.ExUnits, error) {
	c.seen(ctx)
	return c.EvaluateTx(txCbor, utxos)
}

func (c *contextAwareFixed) UtxoByRefContext(ctx context.Context, txHash common.Blake2b256, index uint32) (*common.Utxo, error) {
	c.seen(ctx)
	c.lookups++
	return c.UtxoByRef(txHash, index)
}

func (c *contextAwareFixed) ScriptCborContext(ctx context.Context, scriptHash common.Blake2b224) ([]byte, error) {
	c.seen(ctx)
	return c.ScriptCbor(scriptHash)
}

// capabilityStub reports a fixed context with a reduced capability set.
type capabilityStub struct {
	*fixed.FixedChainContext
	caps backend.CapabilitySet
}

func (c capabilityStub) Capabilities() backend.CapabilitySet { return c.caps }

func syntheticFixed(t *testing.T, gp backend.GenesisParameters) (syntheticTx, *fixed.FixedChainContext) {
	t.Helper()
	tx := buildSyntheticTx(t, compileScript(t, syntheticV3Source))
	cc := fixed.NewFixedChainContext(tx.params.ProtocolParams, gp, 1)
	return tx, cc
}

func checkSyntheticResult(t *testing.T, tx syntheticTx, got map[common.RedeemerKey]common.ExUnits, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
	if got[common.RedeemerKey{Tag: common.RedeemerTagSpend}] != syntheticReferenceV3 ||
		got[common.RedeemerKey{Tag: common.RedeemerTagMint, Index: tx.mintIndex[0]}] != syntheticReferenceV1 ||
		len(got) != 4 {
		t.Fatalf("unexpected result %v", got)
	}
}

func TestLocalEvalCapabilities(t *testing.T) {
	inner := fixed.NewEmptyFixedChainContext()
	caps := NewLocalEvalChainContext(inner).Capabilities()
	for _, capability := range []backend.Capability{
		backend.CapabilityEvaluateTx,
		backend.CapabilityEvaluateTxAdditionalUtxos,
		backend.CapabilityProtocolParams,
		backend.CapabilityUtxoByRef,
	} {
		if !caps.Has(capability) {
			t.Errorf("missing %s", capability)
		}
	}
	if caps.Has(backend.CapabilitySubmitTx) {
		t.Error("gained transaction submission")
	}

	noParams := capabilityStub{inner, backend.CapabilitySet(backend.CapabilityGenesisParams | backend.CapabilityUtxoByRef)}
	if NewLocalEvalChainContext(noParams).Capabilities().Has(backend.CapabilityEvaluateTx) {
		t.Error("reported evaluation without protocol parameters")
	}
	noGenesis := capabilityStub{inner, backend.CapabilitySet(backend.CapabilityProtocolParams)}
	if NewLocalEvalChainContext(noGenesis).Capabilities().Has(backend.CapabilityEvaluateTx) {
		t.Error("reported evaluation without genesis parameters or a slot config")
	}
	if !NewLocalEvalChainContext(noGenesis, WithSlotConfig(PreviewSlotConfig)).Capabilities().Has(backend.CapabilityEvaluateTx) {
		t.Error("did not report evaluation with an explicit slot config")
	}
}

func TestLocalEvalResolvesFromWrappedContext(t *testing.T) {
	tx, inner := syntheticFixed(t, backend.GenesisParameters{NetworkMagic: MainnetNetworkMagic})
	inner.AddUtxoByRef(tx.utxos[0])
	cc := NewLocalEvalChainContext(inner)
	got, err := cc.EvaluateTx(tx.txCbor, nil)
	checkSyntheticResult(t, tx, got, err)
}

func TestLocalEvalPrefersAdditionalUtxos(t *testing.T) {
	tx, base := syntheticFixed(t, backend.GenesisParameters{NetworkMagic: MainnetNetworkMagic})
	inner := &contextAwareFixed{FixedChainContext: base}
	cc := NewLocalEvalChainContext(inner)
	got, err := cc.EvaluateTx(tx.txCbor, tx.utxos)
	checkSyntheticResult(t, tx, got, err)
	if inner.lookups != 0 {
		t.Errorf("queried the wrapped context %d times for supplied UTxOs", inner.lookups)
	}
}

func TestLocalEvalMissingUtxo(t *testing.T) {
	tx, inner := syntheticFixed(t, backend.GenesisParameters{NetworkMagic: MainnetNetworkMagic})
	if _, err := NewLocalEvalChainContext(inner).EvaluateTx(tx.txCbor, nil); err == nil {
		t.Fatal("evaluation succeeded without the spent UTxO")
	}
}

func TestLocalEvalPassesContext(t *testing.T) {
	tx, base := syntheticFixed(t, backend.GenesisParameters{NetworkMagic: MainnetNetworkMagic})
	base.AddUtxoByRef(tx.utxos[0])
	inner := &contextAwareFixed{FixedChainContext: base}
	cc := NewLocalEvalChainContext(inner)

	ctx := context.WithValue(context.Background(), ctxKey{}, "caller")
	got, err := cc.EvaluateTxContext(ctx, tx.txCbor, nil)
	checkSyntheticResult(t, tx, got, err)
	// Protocol parameters, genesis parameters and the spent UTxO.
	if len(inner.contexts) != 3 || inner.lookups != 1 {
		t.Fatalf("wrapped context saw %d calls and %d lookups, want 3 and 1", len(inner.contexts), inner.lookups)
	}
	for _, seen := range inner.contexts {
		if seen.Value(ctxKey{}) != "caller" {
			t.Fatal("wrapped context did not receive the caller's context")
		}
	}

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	inner.contexts = nil
	if _, err := cc.EvaluateTxContext(cancelled, tx.txCbor, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if len(inner.contexts) != 0 {
		t.Error("queried the wrapped context after cancellation")
	}
}

func TestLocalEvalSlotConfigFromGenesis(t *testing.T) {
	// A network outside the known set maps slot 0 to the system start, so a
	// system start 4492800 seconds before mainnet's Shelley start reproduces
	// mainnet's mapping for Shelley-era slots.
	tx, inner := syntheticFixed(t, backend.GenesisParameters{
		NetworkMagic: 42,
		SystemStart:  1596059091 - 4492800,
		SlotLength:   1,
	})
	got, err := NewLocalEvalChainContext(inner).EvaluateTx(tx.txCbor, tx.utxos)
	checkSyntheticResult(t, tx, got, err)

	// Any other start makes the V3 script see a different time and fail.
	_, inner = syntheticFixed(t, backend.GenesisParameters{NetworkMagic: 42, SystemStart: 1596059091, SlotLength: 1})
	if _, err := NewLocalEvalChainContext(inner).EvaluateTx(tx.txCbor, tx.utxos); err == nil {
		t.Fatal("evaluation succeeded with the wrong system start")
	}

	for name, gp := range map[string]backend.GenesisParameters{
		"no system start": {NetworkMagic: 42, SlotLength: 1},
		"no slot length":  {NetworkMagic: 42, SystemStart: 1596059091},
	} {
		_, inner := syntheticFixed(t, gp)
		if _, err := NewLocalEvalChainContext(inner).EvaluateTx(tx.txCbor, tx.utxos); err == nil {
			t.Errorf("%s: evaluation succeeded", name)
		}
	}
}

func TestLocalEvalWithSlotConfigOverridesGenesis(t *testing.T) {
	tx, inner := syntheticFixed(t, backend.GenesisParameters{NetworkMagic: MainnetNetworkMagic})
	shifted := MainnetSlotConfig
	shifted.ZeroTime = shifted.ZeroTime.Add(time.Second)
	if _, err := NewLocalEvalChainContext(inner, WithSlotConfig(shifted)).EvaluateTx(tx.txCbor, tx.utxos); err == nil {
		t.Fatal("evaluation ignored the explicit slot config")
	}
}

func TestLocalEvalProtocolParamsError(t *testing.T) {
	tx, _ := syntheticFixed(t, backend.GenesisParameters{NetworkMagic: MainnetNetworkMagic})
	pp := tx.params.ProtocolParams
	pp.ProtocolMajorVersion = 0
	inner := fixed.NewFixedChainContext(pp, backend.GenesisParameters{NetworkMagic: MainnetNetworkMagic}, 1)
	if _, err := NewLocalEvalChainContext(inner).EvaluateTx(tx.txCbor, tx.utxos); err == nil {
		t.Fatal("evaluation succeeded without a protocol version")
	}
}

func TestLocalEvalDelegates(t *testing.T) {
	tx, inner := syntheticFixed(t, backend.GenesisParameters{NetworkMagic: MainnetNetworkMagic})
	inner.AddUtxoByRef(tx.utxos[0])
	cc := NewLocalEvalChainContext(inner)
	if cc.Unwrap() != backend.ChainContext(inner) {
		t.Error("Unwrap did not return the wrapped context")
	}
	if cc.NetworkId() != 1 {
		t.Errorf("NetworkId = %d", cc.NetworkId())
	}
	pp, err := cc.ProtocolParams()
	if err != nil || pp.ProtocolMajorVersion != 11 {
		t.Errorf("ProtocolParams = %+v, %v", pp, err)
	}
	gp, err := cc.GenesisParams()
	if err != nil || gp.NetworkMagic != MainnetNetworkMagic {
		t.Errorf("GenesisParams = %+v, %v", gp, err)
	}
	if _, err := cc.MaxTxFee(); err != nil {
		t.Errorf("MaxTxFee: %v", err)
	}
	utxo, err := cc.UtxoByRef(tx.utxos[0].Id.Id(), tx.utxos[0].Id.Index())
	if err != nil || utxo == nil {
		t.Errorf("UtxoByRef = %v, %v", utxo, err)
	}
	if _, err := cc.SubmitTx(tx.txCbor); !errors.Is(err, backend.ErrUnsupported) {
		t.Errorf("SubmitTx err = %v, want ErrUnsupported", err)
	}
	if _, err := cc.ScriptCbor(common.Blake2b224{}); !errors.Is(err, backend.ErrUnsupported) {
		t.Errorf("ScriptCbor err = %v, want ErrUnsupported", err)
	}
	if _, err := cc.Utxos(common.Address{}); err != nil {
		t.Errorf("Utxos: %v", err)
	}
	if _, err := cc.Tip(); err != nil {
		t.Errorf("Tip: %v", err)
	}
	if _, err := cc.CurrentEpoch(); err != nil {
		t.Errorf("CurrentEpoch: %v", err)
	}
}

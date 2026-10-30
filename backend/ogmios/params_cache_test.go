package ogmios

import (
	"encoding/json"
	"errors"
	"math/big"
	"testing"
	"time"
)

func newStubbedContext(
	t *testing.T,
	client *stubOgmiosClient,
) *OgmiosChainContext {
	t.Helper()
	ctx, err := NewOgmiosChainContextFromClients(client, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	return ctx
}

// TestProtocolParamsCached covers the transaction builder's call pattern: it
// asks for protocol parameters and the max fee many times while building one
// transaction, which must not become one Ogmios query per call.
func TestProtocolParamsCached(t *testing.T) {
	client := &stubOgmiosClient{
		protocolParams: json.RawMessage(protocolParamsBody(adaLovelace)),
	}
	ctx := newStubbedContext(t, client)

	for range 5 {
		pp, err := ctx.ProtocolParams()
		if err != nil {
			t.Fatal(err)
		}
		assertOgmiosV6ProtocolParams(t, pp)
		if _, err := ctx.MaxTxFee(); err != nil {
			t.Fatal(err)
		}
	}
	if client.protocolParamsCalls != 1 {
		t.Fatalf(
			"queried protocol parameters %d times, want 1",
			client.protocolParamsCalls,
		)
	}
}

func TestProtocolParamsCacheReturnsCopies(t *testing.T) {
	client := &stubOgmiosClient{
		protocolParams: json.RawMessage(protocolParamsBody(adaLovelace)),
	}
	ctx := newStubbedContext(t, client)

	first, err := ctx.ProtocolParams()
	if err != nil {
		t.Fatal(err)
	}
	first.CostModels["PlutusV3"][0] = -1
	first.CostModels["PlutusV9"] = []int64{1}
	first.MinFeeRefScriptCostPerByteRational.SetInt64(-1)
	first.MinFeeReferenceScriptsMultiplierRational.SetInt64(-1)

	second, err := ctx.ProtocolParams()
	if err != nil {
		t.Fatal(err)
	}
	if got := second.CostModels["PlutusV3"][0]; got != 100788 {
		t.Fatalf("cached cost model entry = %d, want 100788", got)
	}
	if _, ok := second.CostModels["PlutusV9"]; ok {
		t.Fatal("a cost model added to a returned value reached the cache")
	}
	if got := second.MinFeeRefScriptCostPerByteRational; got.Cmp(
		big.NewRat(15, 1),
	) != 0 {
		t.Fatalf("cached ref script base price = %s, want 15", got)
	}
	if got := second.MinFeeReferenceScriptsMultiplierRational; got.Cmp(
		big.NewRat(6, 5),
	) != 0 {
		t.Fatalf("cached ref script multiplier = %s, want 6/5", got)
	}
	if client.protocolParamsCalls != 1 {
		t.Fatalf(
			"queried protocol parameters %d times, want 1",
			client.protocolParamsCalls,
		)
	}
}

func TestProtocolParamsCacheExpires(t *testing.T) {
	client := &stubOgmiosClient{
		protocolParams: json.RawMessage(protocolParamsBody(adaLovelace)),
	}
	ctx := newStubbedContext(t, client)

	if _, err := ctx.ProtocolParams(); err != nil {
		t.Fatal(err)
	}
	ctx.mu.Lock()
	ctx.paramsCacheAt = ctx.paramsCacheAt.Add(-cacheExpiry)
	ctx.mu.Unlock()
	if _, err := ctx.ProtocolParams(); err != nil {
		t.Fatal(err)
	}
	if client.protocolParamsCalls != 2 {
		t.Fatalf(
			"queried protocol parameters %d times, want 2",
			client.protocolParamsCalls,
		)
	}
}

// TestProtocolParamsFailuresNotCached covers both a failed query and a
// response that does not decode: neither may be remembered, so the next call
// asks Ogmios again and succeeds once Ogmios answers properly.
func TestProtocolParamsFailuresNotCached(t *testing.T) {
	tests := map[string]func(*stubOgmiosClient){
		"query error": func(c *stubOgmiosClient) {
			c.protocolParamsErr = errors.New("connection refused")
		},
		"malformed response": func(c *stubOgmiosClient) {
			c.protocolParams = json.RawMessage(`{"minFeeCoefficient": "x"}`)
		},
		"missing required fields": func(c *stubOgmiosClient) {
			c.protocolParams = json.RawMessage(`{}`)
		},
	}
	for name, breakClient := range tests {
		t.Run(name, func(t *testing.T) {
			client := &stubOgmiosClient{}
			breakClient(client)
			ctx := newStubbedContext(t, client)

			if _, err := ctx.ProtocolParams(); err == nil {
				t.Fatal("expected error")
			}
			if _, err := ctx.MaxTxFee(); err == nil {
				t.Fatal("expected MaxTxFee error")
			}

			client.protocolParamsErr = nil
			client.protocolParams = json.RawMessage(
				protocolParamsBody(adaLovelace),
			)
			pp, err := ctx.ProtocolParams()
			if err != nil {
				t.Fatal(err)
			}
			assertOgmiosV6ProtocolParams(t, pp)
			if client.protocolParamsCalls != 3 {
				t.Fatalf(
					"queried protocol parameters %d times, want 3",
					client.protocolParamsCalls,
				)
			}
		})
	}
}

func TestGenesisParamsCached(t *testing.T) {
	client := &stubOgmiosClient{
		genesis: json.RawMessage(ogmiosV6ShelleyGenesisBody),
	}
	ctx := newStubbedContext(t, client)

	for range 3 {
		gp, err := ctx.GenesisParams()
		if err != nil {
			t.Fatal(err)
		}
		if got, want := gp.NetworkMagic, 764824073; got != want {
			t.Fatalf("NetworkMagic = %d, want %d", got, want)
		}
	}
	if len(client.eras) != 1 {
		t.Fatalf("queried genesis %d times, want 1", len(client.eras))
	}

	ctx.mu.Lock()
	ctx.genesisCacheAt = time.Now().Add(-cacheExpiry)
	ctx.mu.Unlock()
	if _, err := ctx.GenesisParams(); err != nil {
		t.Fatal(err)
	}
	if len(client.eras) != 2 {
		t.Fatalf("queried genesis %d times after expiry, want 2", len(client.eras))
	}
}

func TestGenesisParamsFailuresNotCached(t *testing.T) {
	client := &stubOgmiosClient{genesis: json.RawMessage(`{"startTime": 1}`)}
	ctx := newStubbedContext(t, client)

	if _, err := ctx.GenesisParams(); err == nil {
		t.Fatal("expected error")
	}
	client.genesis = json.RawMessage(ogmiosV6ShelleyGenesisBody)
	if _, err := ctx.GenesisParams(); err != nil {
		t.Fatal(err)
	}
	if len(client.eras) != 2 {
		t.Fatalf("queried genesis %d times, want 2", len(client.eras))
	}
}

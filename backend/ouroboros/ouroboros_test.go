package ouroboros

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/blinklabs-io/gouroboros/ledger/common"

	"github.com/Salvionied/apollo/v2/backend"
)

func TestNewOuroborosChainContextValidatesConfig(t *testing.T) {
	valid := Config{Network: "unix", Address: "/run/cardano/node.socket", NetworkMagic: 764824073}
	if _, err := NewOuroborosChainContext(valid); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}
	tests := map[string]struct {
		mutate  func(*Config)
		wantErr string
	}{
		"missing network": {func(c *Config) { c.Network = "" }, "must be \"unix\" or \"tcp\""},
		"unknown network": {func(c *Config) { c.Network = "udp" }, "must be \"unix\" or \"tcp\""},
		"missing address": {func(c *Config) { c.Address = "  " }, "address is required"},
		"missing magic":   {func(c *Config) { c.NetworkMagic = 0 }, "network magic is required"},
		"negative dial":   {func(c *Config) { c.DialTimeout = -time.Second }, "dial timeout"},
		"negative query":  {func(c *Config) { c.QueryTimeout = -time.Second }, "query timeout"},
		"negative submit": {func(c *Config) { c.SubmitTimeout = -time.Second }, "submit timeout"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			cfg := valid
			test.mutate(&cfg)
			ctx, err := NewOuroborosChainContext(cfg)
			if err == nil || ctx != nil {
				t.Fatalf("got (%v, %v), want a rejected configuration", ctx, err)
			}
			if !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("error = %q, want substring %q", err, test.wantErr)
			}
		})
	}
}

func TestDialsConfiguredSocket(t *testing.T) {
	ctx, err := NewOuroborosChainContext(Config{
		Network:      "unix",
		Address:      t.TempDir() + "/missing.socket",
		NetworkMagic: 2,
		NetworkId:    0,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = ctx.Tip()
	if err == nil || !strings.Contains(err.Error(), "missing.socket") {
		t.Fatalf("got %v, want a dial error naming the socket", err)
	}
	if ctx.NetworkId() != 0 {
		t.Fatalf("NetworkId() = %d, want 0", ctx.NetworkId())
	}
}

func TestCapabilitiesExcludeWhatTheNodeCannotServe(t *testing.T) {
	ctx := newContext(nil, Config{NetworkMagic: 1})
	caps := ctx.Capabilities()
	for _, missing := range []backend.Capability{
		backend.CapabilityScriptCbor,
		backend.CapabilityEvaluateTx,
		backend.CapabilityEvaluateTxAdditionalUtxos,
	} {
		if caps.Has(missing) {
			t.Errorf("reports %v", missing)
		}
	}
	for _, present := range []backend.Capability{
		backend.CapabilityProtocolParams,
		backend.CapabilityGenesisParams,
		backend.CapabilityCurrentEpoch,
		backend.CapabilityTip,
		backend.CapabilityUtxos,
		backend.CapabilitySubmitTx,
		backend.CapabilityUtxoByRef,
	} {
		if !caps.Has(present) {
			t.Errorf("does not report %v", present)
		}
	}

	if _, err := ctx.EvaluateTx(nil, nil); !errors.Is(err, backend.ErrUnsupported) {
		t.Errorf("EvaluateTx error = %v, want ErrUnsupported", err)
	}
	if _, err := ctx.ScriptCbor(common.Blake2b224{}); !errors.Is(err, backend.ErrUnsupported) {
		t.Errorf("ScriptCbor error = %v, want ErrUnsupported", err)
	}
}

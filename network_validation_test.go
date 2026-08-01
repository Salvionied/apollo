package apollo

import (
	"strings"
	"testing"

	"github.com/blinklabs-io/gouroboros/ledger/common"

	"github.com/Salvionied/apollo/v2/backend/fixed"
)

// networkTestContext returns a fixed chain context on the given network, with
// the same protocol and genesis parameters the empty fixed context uses.
func networkTestContext(t *testing.T, network uint8) *fixed.FixedChainContext {
	t.Helper()
	base := fixed.NewEmptyFixedChainContext()
	pp, err := base.ProtocolParams()
	if err != nil {
		t.Fatal(err)
	}
	gp, err := base.GenesisParams()
	if err != nil {
		t.Fatal(err)
	}
	return fixed.NewFixedChainContext(pp, gp, network)
}

// networkTestBuilder returns a builder funded on the wallet's own network,
// ready for Complete().
func networkTestBuilder(
	t *testing.T,
	contextNetwork, walletNetwork uint8,
) *Apollo {
	t.Helper()
	cc := networkTestContext(t, contextNetwork)
	walletAddr := syntheticAddress(
		t,
		common.AddressTypeKeyKey,
		walletNetwork,
	)
	addTestUtxo(cc, walletAddr, 10_000_000, 0x01, 0)
	return New(cc).
		SetWallet(NewExternalWallet(walletAddr)).
		SetTtl(50000000)
}

func requireNetworkMismatch(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected a network mismatch error, got none")
	}
	if !strings.Contains(err.Error(), "network mismatch") {
		t.Fatalf("expected a network mismatch error, got: %v", err)
	}
}

// TestCompleteRejectsOutputAddressOnOtherNetwork covers a preprod payee
// pasted into a mainnet builder, and the reverse.
func TestCompleteRejectsOutputAddressOnOtherNetwork(t *testing.T) {
	for _, network := range []uint8{
		common.AddressNetworkTestnet,
		common.AddressNetworkMainnet,
	} {
		foreign := common.AddressNetworkMainnet
		if network == common.AddressNetworkMainnet {
			foreign = common.AddressNetworkTestnet
		}
		payee := syntheticAddress(t, common.AddressTypeKeyKey, uint8(foreign))
		a := networkTestBuilder(t, network, network).
			PayToAddress(payee, 2_000_000)
		_, err := a.Complete()
		requireNetworkMismatch(t, err)
	}
}

// TestCompleteRejectsChangeAddressOnOtherNetwork covers change silently
// leaving for an address on the wrong network.
func TestCompleteRejectsChangeAddressOnOtherNetwork(t *testing.T) {
	change := syntheticAddress(
		t,
		common.AddressTypeKeyKey,
		common.AddressNetworkTestnet,
	)
	a := networkTestBuilder(
		t,
		common.AddressNetworkMainnet,
		common.AddressNetworkMainnet,
	)
	payee := syntheticAddress(
		t,
		common.AddressTypeKeyKey,
		common.AddressNetworkMainnet,
	)
	_, err := a.
		PayToAddress(payee, 2_000_000).
		SetChangeAddress(change).
		Complete()
	requireNetworkMismatch(t, err)
}

// TestCompleteRejectsWalletOnOtherNetwork covers the bursa default: a wallet
// derived for mainnet pointed at a preprod context for testing.
func TestCompleteRejectsWalletOnOtherNetwork(t *testing.T) {
	a := networkTestBuilder(
		t,
		common.AddressNetworkTestnet,
		common.AddressNetworkMainnet,
	)
	payee := syntheticAddress(
		t,
		common.AddressTypeKeyKey,
		common.AddressNetworkMainnet,
	)
	_, err := a.PayToAddress(payee, 2_000_000).Complete()
	requireNetworkMismatch(t, err)
}

// TestCompleteRejectsInputAddressOnOtherNetwork covers coin selection sourced
// from an address on the wrong network.
func TestCompleteRejectsInputAddressOnOtherNetwork(t *testing.T) {
	foreign := syntheticAddress(
		t,
		common.AddressTypeKeyKey,
		common.AddressNetworkMainnet,
	)
	payee := syntheticAddress(
		t,
		common.AddressTypeKeyKey,
		common.AddressNetworkTestnet,
	)
	a := networkTestBuilder(
		t,
		common.AddressNetworkTestnet,
		common.AddressNetworkTestnet,
	)
	_, err := a.
		AddInputAddress(foreign).
		PayToAddress(payee, 2_000_000).
		Complete()
	requireNetworkMismatch(t, err)
}

// TestCompleteRejectsInputUtxoOnOtherNetwork covers a pinned input whose
// output address is on the wrong network.
func TestCompleteRejectsInputUtxoOnOtherNetwork(t *testing.T) {
	foreignCtx := networkTestContext(t, common.AddressNetworkMainnet)
	foreignAddr := syntheticAddress(
		t,
		common.AddressTypeKeyKey,
		common.AddressNetworkMainnet,
	)
	addTestUtxo(foreignCtx, foreignAddr, 10_000_000, 0x09, 0)
	foreignUtxos, err := foreignCtx.Utxos(foreignAddr)
	if err != nil {
		t.Fatal(err)
	}
	if len(foreignUtxos) != 1 {
		t.Fatalf("expected 1 UTxO, got %d", len(foreignUtxos))
	}
	payee := syntheticAddress(
		t,
		common.AddressTypeKeyKey,
		common.AddressNetworkTestnet,
	)
	a := networkTestBuilder(
		t,
		common.AddressNetworkTestnet,
		common.AddressNetworkTestnet,
	)
	_, err = a.
		AddInput(foreignUtxos[0]).
		PayToAddress(payee, 2_000_000).
		Complete()
	requireNetworkMismatch(t, err)
}

// TestCompleteAcceptsMatchingNetworks proves the check never blocks a caller
// whose wallet, payee, change and context all agree - including the mainnet
// case, where the wallet network is bursa's default.
func TestCompleteAcceptsMatchingNetworks(t *testing.T) {
	for _, network := range []uint8{
		common.AddressNetworkTestnet,
		common.AddressNetworkMainnet,
	} {
		a := networkTestBuilder(t, network, network)
		payee := syntheticAddress(t, common.AddressTypeKeyKey, network)
		change := syntheticAddress(t, common.AddressTypeKeyNone, network)
		a, err := a.
			PayToAddress(payee, 2_000_000).
			SetChangeAddress(change).
			Complete()
		if err != nil {
			t.Fatalf("network %d: unexpected error: %v", network, err)
		}
		tx := a.GetTx()
		if tx == nil {
			t.Fatalf("network %d: expected a transaction", network)
		}
		if tx.Body.TxNetworkId == nil {
			t.Fatalf("network %d: expected a network id in the body", network)
		}
		if *tx.Body.TxNetworkId != network {
			t.Fatalf("network %d: body network id = %d",
				network, *tx.Body.TxNetworkId)
		}
		for i, out := range tx.Body.TxOutputs {
			addr := out.Address()
			if addr.NetworkId() != uint(network) {
				t.Fatalf("network %d: output %d is on network %d",
					network, i, addr.NetworkId())
			}
		}
	}
}

// TestCompleteRejectsContextWithUnknownNetworkId covers a backend reporting a
// network id Cardano does not have.
func TestCompleteRejectsContextWithUnknownNetworkId(t *testing.T) {
	a := networkTestBuilder(t, 7, common.AddressNetworkTestnet)
	payee := syntheticAddress(
		t,
		common.AddressTypeKeyKey,
		common.AddressNetworkTestnet,
	)
	_, err := a.PayToAddress(payee, 2_000_000).Complete()
	if err == nil {
		t.Fatal("expected an error for an unknown context network id")
	}
	if !strings.Contains(err.Error(), "invalid network") {
		t.Fatalf("expected an invalid network error, got: %v", err)
	}
}

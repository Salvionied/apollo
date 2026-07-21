package backend

import (
	"errors"
	"testing"

	"github.com/blinklabs-io/gouroboros/ledger/common"
)

func TestCapabilitySetHas(t *testing.T) {
	capabilities := CapabilitySet(CapabilityProtocolParams | CapabilitySubmitTx)
	if !capabilities.Has(CapabilityProtocolParams | CapabilitySubmitTx) {
		t.Fatal("expected both reported capabilities to be present")
	}
	if capabilities.Has(CapabilityScriptCbor) {
		t.Fatal("unexpected script CBOR capability")
	}
}

func TestCapabilitiesOfLegacyChainContextDefaultsToHistoricContract(t *testing.T) {
	if got := CapabilitiesOf(legacyChainContext{}); got != CapabilitySet(AllCapabilities) {
		t.Fatalf("CapabilitiesOf() = %b, want %b", got, AllCapabilities)
	}
}

func TestUnsupportedErrorIdentity(t *testing.T) {
	err := NewUnsupportedError("test backend", CapabilityScriptCbor)
	if !errors.Is(err, ErrUnsupported) {
		t.Fatalf("errors.Is(%v, ErrUnsupported) = false", err)
	}
	var unsupported *UnsupportedError
	if !errors.As(err, &unsupported) {
		t.Fatalf("errors.As(%v, *UnsupportedError) = false", err)
	}
	if unsupported.Backend != "test backend" || unsupported.Capability != CapabilityScriptCbor {
		t.Fatalf("unexpected unsupported error: %#v", unsupported)
	}
}

// legacyChainContext intentionally does not implement CapabilityReporter.
// It models ChainContext implementations compiled before capabilities existed.
type legacyChainContext struct{}

func (legacyChainContext) ProtocolParams() (ProtocolParameters, error) {
	return ProtocolParameters{}, nil
}
func (legacyChainContext) GenesisParams() (GenesisParameters, error)   { return GenesisParameters{}, nil }
func (legacyChainContext) NetworkId() uint8                            { return 0 }
func (legacyChainContext) CurrentEpoch() (uint64, error)               { return 0, nil }
func (legacyChainContext) MaxTxFee() (uint64, error)                   { return 0, nil }
func (legacyChainContext) Tip() (uint64, error)                        { return 0, nil }
func (legacyChainContext) Utxos(common.Address) ([]common.Utxo, error) { return nil, nil }
func (legacyChainContext) SubmitTx([]byte) (common.Blake2b256, error) {
	return common.Blake2b256{}, nil
}
func (legacyChainContext) EvaluateTx([]byte, []common.Utxo) (map[common.RedeemerKey]common.ExUnits, error) {
	return nil, nil
}
func (legacyChainContext) UtxoByRef(common.Blake2b256, uint32) (*common.Utxo, error) {
	return nil, nil
}
func (legacyChainContext) ScriptCbor(common.Blake2b224) ([]byte, error) { return nil, nil }

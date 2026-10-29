package backend

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/blinklabs-io/gouroboros/ledger/common"
)

var submittedTx = []byte{0x84, 0xa0, 0xa0, 0xf5, 0xf6}

type submittedHashLegacyContext struct {
	legacyChainContext
	hash common.Blake2b256
}

func (c submittedHashLegacyContext) SubmitTx([]byte) (common.Blake2b256, error) {
	return c.hash, nil
}

type submittedHashContext struct {
	trackingContextChain
	hash common.Blake2b256
}

func (c submittedHashContext) SubmitTxContext(context.Context, []byte) (common.Blake2b256, error) {
	return c.hash, nil
}

func TestSubmitTxContextRejectsUnrelatedID(t *testing.T) {
	for _, chain := range []ChainContext{
		submittedHashLegacyContext{},
		&submittedHashContext{},
	} {
		if _, err := SubmitTxContext(t.Context(), chain, submittedTx); err == nil || !strings.Contains(err.Error(), "transaction ID") {
			t.Fatalf("SubmitTxContext(%T) error = %v, want transaction ID mismatch", chain, err)
		}
	}
}

func TestSubmitTxContextAcceptsMatchingID(t *testing.T) {
	want := common.Blake2b256Hash([]byte{0xa0})
	for _, chain := range []ChainContext{
		submittedHashLegacyContext{hash: want},
		&submittedHashContext{hash: want},
	} {
		got, err := SubmitTxContext(t.Context(), chain, submittedTx)
		if err != nil || got != want {
			t.Fatalf("SubmitTxContext(%T) = %x, %v, want %x, nil", chain, got, err, want)
		}
	}
}

type mutatingSubmitLegacyContext struct {
	legacyChainContext
}

func (mutatingSubmitLegacyContext) SubmitTx(txCbor []byte) (common.Blake2b256, error) {
	txCbor[3] = 1
	return common.Blake2b256Hash([]byte{0xa1, 0x02, 0x01}), nil
}

type mutatingSubmitContext struct {
	trackingContextChain
}

func (*mutatingSubmitContext) SubmitTxContext(_ context.Context, txCbor []byte) (common.Blake2b256, error) {
	txCbor[3] = 1
	return common.Blake2b256Hash([]byte{0xa1, 0x02, 0x01}), nil
}

func TestSubmitTxContextChecksOriginalBody(t *testing.T) {
	for _, chain := range []ChainContext{
		mutatingSubmitLegacyContext{},
		&mutatingSubmitContext{},
	} {
		txCbor := []byte{0x84, 0xa1, 0x02, 0x00, 0xa0, 0xf5, 0xf6}
		original := bytes.Clone(txCbor)
		if _, err := SubmitTxContext(t.Context(), chain, txCbor); err == nil || !strings.Contains(err.Error(), "transaction ID") {
			t.Fatalf("SubmitTxContext(%T) error = %v, want transaction ID mismatch", chain, err)
		}
		if !bytes.Equal(txCbor, original) {
			t.Fatalf("SubmitTxContext(%T) changed caller bytes: %x", chain, txCbor)
		}
	}
}

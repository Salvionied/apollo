package apollo

import (
	"testing"

	"github.com/Salvionied/apollo/constants"
)

func TestSetWalletFromKeypairRejectsInvalidInput(t *testing.T) {
	builder := New(nil)

	builder.SetWalletFromKeypair("not hex", "00", constants.TESTNET)

	if builder.err == nil {
		t.Fatal("expected keypair validation error")
	}
	if builder.wallet != nil {
		t.Fatal("wallet should not be set for invalid key material")
	}
}

package apollo

import (
	"strings"
	"testing"

	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/conway"
)

// Compile-time assertions on the accessor shapes.
//
// GetTx deliberately returns the concrete Conway transaction: Apollo builds
// Conway, and callers need fields no era-neutral interface exposes. ToTxOut is
// deliberately era-neutral because PaymentI is implemented by third parties, so
// changing its signature for a later era would break every implementation.
var (
	_ func(*Apollo) *conway.ConwayTransaction          = (*Apollo).GetTx
	_ func(*Payment) (common.TransactionOutput, error) = (*Payment).ToTxOut
)

// foreignOutput is a minimal common.TransactionOutput that is not a Babbage
// output.
type foreignOutput struct {
	common.TransactionOutput
}

// TestForeignOutputRejectedWithClearError covers the cost of making
// PaymentI.ToTxOut era-neutral: a third-party payment may now return an output
// Apollo cannot place in a Conway body, and that must be a clear error rather
// than a panic or a silently wrong transaction.
func TestForeignOutputRejectedWithClearError(t *testing.T) {
	_, err := babbageOutputOf(&foreignOutput{})
	if err == nil {
		t.Fatal("expected a non-Babbage output to be rejected")
	}
	if !strings.Contains(err.Error(), "Babbage") {
		t.Errorf("error does not explain the format requirement: %v", err)
	}

	if _, err := babbageOutputOf(nil); err == nil {
		t.Error("expected a nil output to be rejected")
	}
}

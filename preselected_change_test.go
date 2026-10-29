package apollo

import (
	"testing"

	"github.com/blinklabs-io/gouroboros/ledger/common"
)

func completePinned(t *testing.T, pinned common.Utxo, pool []common.Utxo, pay int64) *Apollo {
	t.Helper()
	addr := testAddress(t)
	a, err := newTestApollo(setupFixedContext()).
		SetWallet(NewExternalWallet(addr)).
		AddInput(pinned).
		AddLoadedUTxOs(pool...).
		PayToAddress(addr, pay).
		SetTtl(50_000_000).
		Complete()
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func assertConserved(t *testing.T, a *Apollo, utxos ...common.Utxo) {
	t.Helper()
	body := a.GetTx().Body
	byRef := map[string]common.Utxo{}
	for _, u := range utxos {
		byRef[utxoRef(u)] = u
	}
	var in, out uint64
	for _, i := range body.Inputs() {
		in += byRef[i.String()].Output.Amount().Uint64()
	}
	for _, o := range body.Outputs() {
		out += o.Amount().Uint64()
	}
	if in != out+body.TxFee {
		t.Errorf("inputs %d != outputs %d + fee %d", in, out, body.TxFee)
	}
}

// Pinned inputs that cover the payment but leave change below min-UTxO must
// not burn that change as fee while the pool can fund a change output.
func TestPinnedInputKeepsChangeWhenPoolCanFundIt(t *testing.T) {
	addr := testAddress(t)
	pinned := benchUtxo(addr, 1, 3_000_000, nil)
	funding := benchUtxo(addr, 2, 30_000_000, nil)
	pool := []common.Utxo{funding}
	a := completePinned(t, pinned, pool, 2_000_000)
	body := a.GetTx().Body
	if len(body.Inputs()) != 2 || len(body.Outputs()) != 2 {
		t.Fatalf("got %d inputs and %d outputs, want 2 and 2 (fee %d)", len(body.Inputs()), len(body.Outputs()), body.TxFee)
	}
	if body.TxFee > 300_000 {
		t.Errorf("fee %d still absorbs change", body.TxFee)
	}
	assertConserved(t, a, pinned, funding)
}

// With nothing else to spend, the pinned input still pays and the remainder
// becomes fee, as before.
func TestPinnedInputFoldsChangeWhenPoolIsEmpty(t *testing.T) {
	addr := testAddress(t)
	pinned := benchUtxo(addr, 1, 3_000_000, nil)
	a := completePinned(t, pinned, nil, 2_000_000)
	body := a.GetTx().Body
	if len(body.Inputs()) != 1 || len(body.Outputs()) != 1 || body.TxFee != 1_000_000 {
		t.Fatalf("got %d inputs, %d outputs, fee %d", len(body.Inputs()), len(body.Outputs()), body.TxFee)
	}
}

// A pinned input carrying tokens whose leftover ADA cannot fund the change
// that must return them draws the shortfall from the pool.
func TestPinnedTokenInputFundsItsChange(t *testing.T) {
	addr := testAddress(t)
	pinned := benchUtxo(addr, 1, 2_500_000, benchAsset(7, 100))
	funding := benchUtxo(addr, 2, 30_000_000, nil)
	pool := []common.Utxo{funding}
	a := completePinned(t, pinned, pool, 2_000_000)
	body := a.GetTx().Body
	if len(body.Inputs()) != 2 {
		t.Fatalf("got %d inputs, want 2", len(body.Inputs()))
	}
	assertConserved(t, a, pinned, funding)
}

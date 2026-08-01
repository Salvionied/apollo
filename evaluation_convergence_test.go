package apollo

import (
	"math/big"
	"testing"
)

// buildTransfer completes a plain lovelace payment from a wallet holding
// exactly balance, with no scripts involved.
func buildTransfer(
	t *testing.T,
	balance uint64,
	pay int64,
) (*Apollo, error) {
	t.Helper()
	cc := setupFixedContext()
	addr := testAddress(t)
	addTestUtxo(cc, addr, balance, 0x01, 0)

	p, err := NewPayment(validTestAddrBech32, pay, nil)
	if err != nil {
		t.Fatal(err)
	}
	return New(cc).
		SetWallet(NewExternalWallet(addr)).
		AddPayment(p).
		SetTtl(50_000_000).
		Complete()
}

// minFeeForTx returns the protocol minimum size-based fee for the serialized
// transaction, computed independently of the builder.
func minFeeForTx(t *testing.T, a *Apollo) uint64 {
	t.Helper()
	txCbor, err := a.GetTxCbor()
	if err != nil {
		t.Fatal(err)
	}
	pp, err := a.Context.ProtocolParams()
	if err != nil {
		t.Fatal(err)
	}
	if pp.MinFeeCoefficient <= 0 || pp.MinFeeConstant <= 0 {
		t.Fatalf(
			"protocol params not usable: minFeeA=%d minFeeB=%d",
			pp.MinFeeCoefficient, pp.MinFeeConstant,
		)
	}
	//nolint:gosec // protocol params validated positive above
	return uint64(len(txCbor))*uint64(pp.MinFeeCoefficient) +
		uint64(pp.MinFeeConstant)
}

// TestDustChangeIsAbsorbedIntoFee is the regression guard for the fee
// convergence loop. When change falls below the change output's min-UTxO,
// buildBalancedOutputs drops the change output and adds the remainder to the
// fee. The loop previously fed that dust-inclusive fee back into estimateFee,
// which cannot predict the surcharge, so the comparison never matched and
// every such transaction failed with "did not converge".
func TestDustChangeIsAbsorbedIntoFee(t *testing.T) {
	const pay = 2_000_000

	// Change of 900_000 is below the ~978_370 min-UTxO for an ADA-only change
	// output, so it must be absorbed rather than emitted.
	a, err := buildTransfer(t, pay+900_000, pay)
	if err != nil {
		t.Fatalf("dust-change transfer failed to build: %v", err)
	}

	tx := a.GetTx()
	if got := len(tx.Body.Outputs()); got != 1 {
		t.Errorf("built %d outputs, want 1 (change must be absorbed)", got)
	}
	if got, want := tx.Body.TxFee, uint64(900_000); got != want {
		t.Errorf("fee = %d, want %d (payment remainder absorbed)", got, want)
	}
	assertValueConserved(t, a, pay+900_000)
}

// TestDustChangeBandBuilds sweeps the whole range in which change is dust and
// asserts every balance produces a valid, fully funded transaction.
func TestDustChangeBandBuilds(t *testing.T) {
	const pay = 2_000_000

	// The lower bound is imposed by coin selection, which currently reserves
	// the whole of MaxTxFee (876_277 on these parameters) rather than an
	// estimate, so balances below payment+MaxTxFee never reach the fee loop.
	// That over-reserve is a separate defect; this test covers the range the
	// convergence loop is actually reached for, spanning both the dust-absorbed
	// and change-emitted outcomes.
	for balance := uint64(pay + 880_000); balance <= pay+1_200_000; balance += 5_000 {
		a, err := buildTransfer(t, balance, pay)
		if err != nil {
			t.Errorf("balance %d: %v", balance, err)
			continue
		}
		tx := a.GetTx()
		outputs := len(tx.Body.Outputs())
		if outputs != 1 && outputs != 2 {
			t.Errorf("balance %d: built %d outputs, want 1 or 2", balance, outputs)
		}
		// Whether change was absorbed or emitted, the fee must still cover the
		// transaction's own size. Absorbing dust must never underpay.
		if minFee := minFeeForTx(t, a); tx.Body.TxFee < minFee {
			t.Errorf(
				"balance %d: fee %d is below the minimum %d for the built tx",
				balance, tx.Body.TxFee, minFee,
			)
		}
		assertValueConserved(t, a, balance)
	}
}

// TestChangeAboveMinUtxoIsStillEmitted confirms the fix did not turn ordinary
// change into absorbed dust.
func TestChangeAboveMinUtxoIsStillEmitted(t *testing.T) {
	const pay = 2_000_000
	const balance = 10_000_000

	a, err := buildTransfer(t, balance, pay)
	if err != nil {
		t.Fatal(err)
	}
	tx := a.GetTx()
	if got := len(tx.Body.Outputs()); got != 2 {
		t.Fatalf("built %d outputs, want 2 (payment plus change)", got)
	}
	if minFee := minFeeForTx(t, a); tx.Body.TxFee < minFee {
		t.Errorf("fee %d is below the minimum %d", tx.Body.TxFee, minFee)
	}
	// With ample change the fee should be the size-based fee, not inflated.
	if tx.Body.TxFee > 1_000_000 {
		t.Errorf("fee %d is implausibly large for a simple transfer", tx.Body.TxFee)
	}
	assertValueConserved(t, a, balance)
}

// TestForceFeeStillBypassesEstimation guards the other branch of the loop:
// with a forced fee, no re-estimation happens and the forced value is charged.
func TestForceFeeStillBypassesEstimation(t *testing.T) {
	cc := setupFixedContext()
	addr := testAddress(t)
	addTestUtxo(cc, addr, 10_000_000, 0x01, 0)

	p, err := NewPayment(validTestAddrBech32, 2_000_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	a, err := New(cc).
		SetWallet(NewExternalWallet(addr)).
		AddPayment(p).
		SetTtl(50_000_000).
		ForceFee(300_000).
		Complete()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := a.GetTx().Body.TxFee, uint64(300_000); got != want {
		t.Errorf("forced fee = %d, want %d", got, want)
	}
}

// assertValueConserved checks the Cardano balance equation against the single
// input the helpers create: sum(outputs) + fee must equal the input value.
func assertValueConserved(t *testing.T, a *Apollo, inputValue uint64) {
	t.Helper()
	tx := a.GetTx()
	out := new(big.Int)
	for _, o := range tx.Body.Outputs() {
		out.Add(out, o.Amount())
	}
	got := new(big.Int).Add(out, new(big.Int).SetUint64(tx.Body.TxFee))
	want := new(big.Int).SetUint64(inputValue)
	if got.Cmp(want) != 0 {
		t.Errorf(
			"value not conserved: outputs %s + fee %d = %s, inputs %d",
			out, tx.Body.TxFee, got, inputValue,
		)
	}
}

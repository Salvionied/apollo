package apollo

import (
	"strings"
	"testing"
)

// TestWalletCanSpendCloseToItsBalance is the regression guard for the coin
// selection fee reserve. Selection previously reserved the whole of MaxTxFee --
// the fee for a maximum-size transaction, 876277 lovelace on these parameters --
// so a wallet holding less than payment+876277 was refused with "insufficient
// UTxOs to cover required value" even though it could comfortably afford the
// transaction. The reserve is now derived from the shape being built, so the
// requirement is payment plus the actual fee.
func TestWalletCanSpendCloseToItsBalance(t *testing.T) {
	const pay = 2_000_000

	// A wallet holding the payment plus a little over the real fee must be able
	// to send. The old reserve needed pay+876277.
	const balance = pay + 200_000

	a, err := buildTransfer(t, balance, pay)
	if err != nil {
		t.Fatalf("balance %d could not send %d: %v", balance, pay, err)
	}
	tx := a.GetTx()
	if got := len(tx.Body.Inputs()); got != 1 {
		t.Errorf("spent %d inputs, want 1", got)
	}
	// The fee must still cover the transaction's own size.
	if minFee := minFeeForTx(t, a); tx.Body.TxFee < minFee {
		t.Errorf("fee %d is below the minimum %d", tx.Body.TxFee, minFee)
	}
	assertValueConserved(t, a, balance)
}

// TestSpendableThresholdIsPaymentPlusFee walks up from an unaffordable balance
// and pins where spending becomes possible: it must be governed by the real fee,
// not by MaxTxFee.
func TestSpendableThresholdIsPaymentPlusFee(t *testing.T) {
	const pay = 2_000_000

	var firstOK uint64
	for balance := uint64(pay); balance <= pay+900_000; balance += 5_000 {
		if _, err := buildTransfer(t, balance, pay); err == nil {
			firstOK = balance
			break
		}
	}
	if firstOK == 0 {
		t.Fatal("no balance up to payment+900000 could send the payment")
	}

	overhead := firstOK - pay
	t.Logf("first spendable balance %d (overhead %d lovelace)", firstOK, overhead)

	// The overhead must be in the region of a real transaction fee, not the
	// maximum-size fee the old reserve assumed.
	const maxTxFeeOnFixedParams = 876_277
	if overhead >= maxTxFeeOnFixedParams {
		t.Errorf(
			"overhead is %d lovelace, still at or above the MaxTxFee reserve %d",
			overhead, maxTxFeeOnFixedParams,
		)
	}
	if overhead > 400_000 {
		t.Errorf("overhead %d lovelace is larger than a plausible fee", overhead)
	}
}

// TestGenuinelyInsufficientBalanceStillFails confirms the tighter reserve did
// not turn a real shortfall into a silently wrong transaction.
func TestGenuinelyInsufficientBalanceStillFails(t *testing.T) {
	const pay = 2_000_000

	// Below payment plus any possible fee, this cannot be funded.
	if _, err := buildTransfer(t, pay+1_000, pay); err == nil {
		t.Fatal("expected an error when the balance cannot cover payment plus fee")
	} else if !strings.Contains(err.Error(), "coin selection failed") {
		t.Errorf("unexpected error for an unfundable payment: %v", err)
	}
}

// TestReserveGrowsForManyInputs exercises the path where the first estimated
// reserve is too small because selection has to gather many UTxOs: the reserve
// must grow and re-select rather than proceed underfunded.
func TestReserveGrowsForManyInputs(t *testing.T) {
	cc := setupFixedContext()
	addr := testAddress(t)
	// Many small UTxOs, so covering the payment needs a lot of them and the
	// transaction is far larger than the pre-selection estimate assumed.
	const utxoCount = 60
	const perUtxo = 2_000_000
	for i := range utxoCount {
		addTestUtxo(cc, addr, perUtxo, byte(i+1), 0)
	}

	p, err := NewPayment(validTestAddrBech32, 100_000_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	a, err := New(cc).
		SetWallet(NewExternalWallet(addr)).
		AddPayment(p).
		SetTtl(50_000_000).
		Complete()
	if err != nil {
		t.Fatalf("multi-input transfer failed to build: %v", err)
	}

	tx := a.GetTx()
	inputs := len(tx.Body.Inputs())
	if inputs < 2 {
		t.Fatalf("spent %d inputs, expected many", inputs)
	}
	minFee := minFeeForTx(t, a)
	t.Logf("inputs=%d fee=%d minFee=%d", inputs, tx.Body.TxFee, minFee)
	if tx.Body.TxFee < minFee {
		t.Errorf(
			"fee %d is below the minimum %d for a %d-input transaction",
			tx.Body.TxFee, minFee, inputs,
		)
	}
}

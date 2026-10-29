package apollo

import (
	"strings"
	"testing"

	"github.com/Salvionied/apollo/v2/backend"
	"github.com/Salvionied/apollo/v2/backend/fixed"
)

func newTestApollo(context backend.ChainContext) *Apollo {
	return New(context).SetMaxFee(2_000_000).SetMaxPaymentIncrease(2_000_000)
}

func spendingLimitContext(t *testing.T, change func(*backend.ProtocolParameters)) *fixed.FixedChainContext {
	t.Helper()
	params, err := setupFixedContext().ProtocolParams()
	if err != nil {
		t.Fatal(err)
	}
	change(&params)
	return fixed.NewFixedChainContext(params, backend.GenesisParameters{NetworkMagic: 1}, 0)
}

func TestCompleteRequiresApprovedFeeLimit(t *testing.T) {
	_, err := New(setupFixedContext()).Complete()
	if err == nil || !strings.Contains(err.Error(), "SetMaxFee") {
		t.Fatalf("missing fee limit: %v", err)
	}
}

func TestPaymentIncreaseRequiresApprovalWithoutMutatingOriginal(t *testing.T) {
	cc := spendingLimitContext(t, func(pp *backend.ProtocolParameters) {
		pp.CoinsPerUtxoByte = "100000"
	})
	addr := testAddress(t)
	addTestUtxo(cc, addr, 50_000_000, 0x71, 0)
	payment, err := NewPayment(validTestAddrBech32, 500_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = New(cc).SetWallet(NewExternalWallet(addr)).AddPayment(payment).
		SetMaxFee(2_000_000).Complete()
	if err == nil || !strings.Contains(err.Error(), "approved minimum-UTxO increase") {
		t.Fatalf("inflated payment was not rejected: %v", err)
	}
	if payment.Lovelace != 500_000 {
		t.Fatalf("failed build changed original payment to %d", payment.Lovelace)
	}
	if !strings.Contains(err.Error(), "requested 500000") {
		t.Fatalf("error omitted requested amount: %v", err)
	}
}

func TestApprovedPaymentIncreaseBuilds(t *testing.T) {
	cc := setupFixedContext()
	addr := testAddress(t)
	addTestUtxo(cc, addr, 10_000_000, 0x72, 0)
	payment, err := NewPayment(validTestAddrBech32, 500_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	a, err := New(cc).SetWallet(NewExternalWallet(addr)).AddPayment(payment).
		SetMaxFee(1_000_000).SetMaxPaymentIncrease(1_000_000).Complete()
	if err != nil {
		t.Fatal(err)
	}
	if got := a.GetTx().Body.TxOutputs[0].OutputAmount.Amount; got <= 500_000 || got > 1_500_000 {
		t.Fatalf("approved payment output = %d", got)
	}
	if payment.Lovelace != 500_000 {
		t.Fatalf("builder changed original payment to %d", payment.Lovelace)
	}
}

func TestPaymentIncreaseLimitAppliesAcrossOutputs(t *testing.T) {
	cc := setupFixedContext()
	addr := testAddress(t)
	addTestUtxo(cc, addr, 10_000_000, 0x75, 0)
	first, err := NewPayment(validTestAddrBech32, 500_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewPayment(validTestAddrBech32, 500_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = New(cc).SetWallet(NewExternalWallet(addr)).AddPayment(first).AddPayment(second).
		SetMaxFee(1_000_000).SetMaxPaymentIncrease(600_000).Complete()
	if err == nil || !strings.Contains(err.Error(), "payment 1 requires") {
		t.Fatalf("cumulative payment increase was not rejected: %v", err)
	}
	if first.Lovelace != 500_000 || second.Lovelace != 500_000 {
		t.Fatalf("failed build changed payments: %d, %d", first.Lovelace, second.Lovelace)
	}
}

func TestCloneRetainsApprovedSpendingLimits(t *testing.T) {
	original := New(setupFixedContext()).SetMaxFee(300_000).SetMaxPaymentIncrease(400_000)
	clone := original.Clone()
	if !clone.maxFeeSet || clone.maxFee != 300_000 || clone.maxPaymentIncrease != 400_000 {
		t.Fatalf("clone lost spending limits: fee %d (set %t), payment increase %d", clone.maxFee, clone.maxFeeSet, clone.maxPaymentIncrease)
	}
}

func TestInflatedFeeParametersExceedApprovedMaximum(t *testing.T) {
	cc := spendingLimitContext(t, func(pp *backend.ProtocolParameters) {
		pp.MinFeeConstant = 5_000_000
	})
	addr := testAddress(t)
	addTestUtxo(cc, addr, 50_000_000, 0x73, 0)
	payment, err := NewPayment(validTestAddrBech32, 2_000_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = New(cc).SetWallet(NewExternalWallet(addr)).AddPayment(payment).
		SetMaxFee(1_000_000).Complete()
	if err == nil || !strings.Contains(err.Error(), "exceeds approved maximum") {
		t.Fatalf("inflated fee was not rejected: %v", err)
	}
}

func TestInflatedChangeMinimumCannotBecomeFee(t *testing.T) {
	cc := spendingLimitContext(t, func(pp *backend.ProtocolParameters) {
		pp.CoinsPerUtxoByte = "100000"
	})
	addr := testAddress(t)
	addTestUtxo(cc, addr, 10_000_000, 0x74, 0)
	payment, err := NewPayment(validTestAddrBech32, 2_000_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = New(cc).SetWallet(NewExternalWallet(addr)).AddPayment(payment).
		SetMaxFee(1_000_000).Complete()
	if err == nil || !strings.Contains(err.Error(), "exceeds approved maximum") {
		t.Fatalf("inflated change minimum was not rejected: %v", err)
	}
}

func TestApprovedFeeAndDustRemainWithinLimit(t *testing.T) {
	a, err := buildTransfer(t, 2_900_000, 2_000_000)
	if err != nil {
		t.Fatal(err)
	}
	if fee := a.GetTx().Body.TxFee; fee != 900_000 || fee > 2_000_000 {
		t.Fatalf("approved fee including dust = %d", fee)
	}
}

func TestSignRechecksFeeLimitAfterBodyMutation(t *testing.T) {
	a, err := buildTransfer(t, 10_000_000, 2_000_000)
	if err != nil {
		t.Fatal(err)
	}
	a.GetTx().Body.TxFee = 2_000_001
	_, err = a.SignWithSkey(nil)
	if err == nil || !strings.Contains(err.Error(), "exceeds approved maximum") {
		t.Fatalf("modified fee reached signing: %v", err)
	}
}

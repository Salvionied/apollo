package apollo

import (
	"math/big"
	"testing"

	"github.com/Salvionied/apollo/v2/backend/fixed"
	"github.com/blinklabs-io/gouroboros/ledger/common"
)

func poolParams(b byte) common.PoolRegistrationCertificate {
	var operator common.Blake2b224
	operator[0] = b
	p := common.PoolRegistrationCertificate{
		Operator: operator,
		Pledge:   1_000_000,
		Cost:     340_000_000,
		Margin:   common.GenesisRat{Rat: big.NewRat(1, 100)},
	}
	var reward common.Blake2b224
	reward[0] = b
	if err := p.SetRewardAccountCredential(common.Credential{Credential: reward}, common.AddressNetworkTestnet); err != nil {
		panic(err)
	}
	return p
}

func TestUpdatePoolChargesNoDeposit(t *testing.T) {
	a := New(setupFixedContext()).UpdatePool(poolParams(1))
	if a.err != nil {
		t.Fatal(a.err)
	}
	if len(a.certificates) != 1 || a.certificates[0].Type != uint(common.CertificateTypePoolRegistration) {
		t.Fatalf("expected one pool registration certificate, got %+v", a.certificates)
	}
	got, err := a.certificateDepositAdjustment(2_000_000, 500_000_000)
	if err != nil {
		t.Fatal(err)
	}
	if got != 0 {
		t.Fatalf("pool update deposit = %d, want 0", got)
	}
}

func TestRegisterPoolSameOperatorChargedOnce(t *testing.T) {
	a := New(setupFixedContext()).RegisterPool(poolParams(1)).RegisterPool(poolParams(1))
	got, err := a.certificateDepositAdjustment(2_000_000, 500_000_000)
	if err != nil {
		t.Fatal(err)
	}
	if got != 500_000_000 {
		t.Fatalf("deposit = %d, want one pool deposit", got)
	}
}

func TestUpdatePoolBesideNewPool(t *testing.T) {
	a := New(setupFixedContext()).UpdatePool(poolParams(1)).RegisterPool(poolParams(2))
	got, err := a.certificateDepositAdjustment(2_000_000, 500_000_000)
	if err != nil {
		t.Fatal(err)
	}
	if got != 500_000_000 {
		t.Fatalf("deposit = %d, want one pool deposit for the new pool", got)
	}
}

func TestUpdatePoolSurvivesClone(t *testing.T) {
	a := New(setupFixedContext()).UpdatePool(poolParams(1)).Clone()
	got, err := a.certificateDepositAdjustment(2_000_000, 500_000_000)
	if err != nil {
		t.Fatal(err)
	}
	if got != 0 {
		t.Fatalf("cloned pool update deposit = %d, want 0", got)
	}
}

func TestUpdatePoolRejectsBadMargin(t *testing.T) {
	p := poolParams(1)
	p.Margin = common.GenesisRat{}
	if a := New(setupFixedContext()).UpdatePool(p); a.err == nil {
		t.Fatal("nil margin accepted")
	}
}

func TestCompleteWithPoolUpdateConservesValue(t *testing.T) {
	cc := fixed.NewEmptyFixedChainContext()
	addr := testAddress(t)
	// far less than a pool deposit: an update must not need one
	const input = uint64(20_000_000)
	addTestUtxo(cc, addr, input, 0x01, 0)

	a, err := New(cc).
		SetWallet(NewExternalWallet(addr)).
		SetTtl(50000000).
		UpdatePool(poolParams(1)).
		Complete()
	if err != nil {
		t.Fatal(err)
	}

	tx := a.GetTx()
	if len(tx.Body.TxCertificates) != 1 {
		t.Fatalf("expected 1 certificate, got %d", len(tx.Body.TxCertificates))
	}
	produced := tx.Body.Fee().Uint64()
	for _, o := range tx.Body.Outputs() {
		produced += o.Amount().Uint64()
	}
	if produced != input {
		t.Fatalf("outputs + fee = %d, want consumed %d", produced, input)
	}
}

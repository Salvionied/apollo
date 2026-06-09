package apollo

import (
	"testing"

	"github.com/blinklabs-io/gouroboros/ledger/common"

	"github.com/Salvionied/apollo/v2/backend/fixed"
)

func TestRegisterStake(t *testing.T) {
	cc := setupFixedContext()
	addr := testAddress(t)
	w := NewExternalWallet(addr)
	a := New(cc).SetWallet(w)

	cred := common.Credential{CredType: 0, Credential: addr.StakeKeyHash()}
	a, err := a.RegisterStake(&cred)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.certificates) != 1 {
		t.Fatalf("expected 1 certificate, got %d", len(a.certificates))
	}
	if a.certificates[0].Type != uint(common.CertificateTypeStakeRegistration) {
		t.Errorf("expected type %d, got %d", common.CertificateTypeStakeRegistration, a.certificates[0].Type)
	}
}

func TestRegisterStakeFromWallet(t *testing.T) {
	cc := setupFixedContext()
	addr := testAddress(t)
	w := NewExternalWallet(addr)
	a := New(cc).SetWallet(w)

	a, err := a.RegisterStake(nil) // should use wallet
	if err != nil {
		t.Fatal(err)
	}
	if len(a.certificates) != 1 {
		t.Fatalf("expected 1 certificate, got %d", len(a.certificates))
	}
}

func TestRegisterStakeNoWalletNoCred(t *testing.T) {
	cc := setupFixedContext()
	a := New(cc)
	_, err := a.RegisterStake(nil)
	if err == nil {
		t.Error("expected error when no wallet and no credential")
	}
}

func TestRegisterStakeFromBech32(t *testing.T) {
	cc := setupFixedContext()
	a := New(cc)
	// Pass bech32 string directly - resolveCredential handles it
	a, err := a.RegisterStake(validTestAddrBech32)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.certificates) != 1 {
		t.Fatalf("expected 1 certificate, got %d", len(a.certificates))
	}
}

func TestDeregisterStake(t *testing.T) {
	cc := setupFixedContext()
	addr := testAddress(t)
	w := NewExternalWallet(addr)
	a := New(cc).SetWallet(w)

	cred := common.Credential{CredType: 0, Credential: addr.StakeKeyHash()}
	a, err := a.DeregisterStake(&cred)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.certificates) != 1 {
		t.Fatalf("expected 1 certificate, got %d", len(a.certificates))
	}
	if a.certificates[0].Type != uint(common.CertificateTypeStakeDeregistration) {
		t.Errorf("expected type %d, got %d", common.CertificateTypeStakeDeregistration, a.certificates[0].Type)
	}
}

func TestDeregisterStakeFromBech32(t *testing.T) {
	cc := setupFixedContext()
	a := New(cc)
	a, err := a.DeregisterStake(validTestAddrBech32)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.certificates) != 1 {
		t.Fatalf("expected 1 certificate, got %d", len(a.certificates))
	}
}

func TestDelegateStake(t *testing.T) {
	cc := setupFixedContext()
	a := New(cc)

	addr := testAddress(t)
	cred := common.Credential{CredType: 0, Credential: addr.StakeKeyHash()}
	var poolHash common.Blake2b224
	poolHash[0] = 0xaa

	a, err := a.DelegateStake(cred, poolHash)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.certificates) != 1 {
		t.Fatalf("expected 1 certificate, got %d", len(a.certificates))
	}
	if a.certificates[0].Type != uint(common.CertificateTypeStakeDelegation) {
		t.Errorf("expected type %d, got %d", common.CertificateTypeStakeDelegation, a.certificates[0].Type)
	}
}

func TestDelegateStakeFromBech32(t *testing.T) {
	cc := setupFixedContext()
	a := New(cc)
	var poolHash common.Blake2b224
	poolHash[0] = 0xbb

	// Pass bech32 string directly
	a, err := a.DelegateStake(validTestAddrBech32, poolHash)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.certificates) != 1 {
		t.Fatalf("expected 1 certificate, got %d", len(a.certificates))
	}
}

func TestRegisterAndDelegateStake(t *testing.T) {
	cc := setupFixedContext()
	a := New(cc)

	tAddr := testAddress(t)
	cred := common.Credential{CredType: 0, Credential: tAddr.StakeKeyHash()}
	var poolHash common.Blake2b224
	poolHash[0] = 0xcc

	a, err := a.RegisterAndDelegateStake(cred, poolHash, StakeDeposit)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.certificates) != 1 {
		t.Fatalf("expected 1 certificate, got %d", len(a.certificates))
	}
	if a.certificates[0].Type != uint(common.CertificateTypeStakeRegistrationDelegation) {
		t.Errorf("expected type %d, got %d", common.CertificateTypeStakeRegistrationDelegation, a.certificates[0].Type)
	}
}

func TestRegisterAndDelegateStakeFromBech32(t *testing.T) {
	cc := setupFixedContext()
	a := New(cc)
	var poolHash common.Blake2b224
	poolHash[0] = 0xdd

	a, err := a.RegisterAndDelegateStake(validTestAddrBech32, poolHash, StakeDeposit)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.certificates) != 1 {
		t.Fatalf("expected 1 certificate, got %d", len(a.certificates))
	}
}

func TestDelegateVote(t *testing.T) {
	cc := setupFixedContext()
	a := New(cc)

	tAddr := testAddress(t)
	cred := common.Credential{CredType: 0, Credential: tAddr.StakeKeyHash()}
	drep := common.Drep{Type: common.DrepTypeAbstain}

	a, err := a.DelegateVote(cred, drep)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.certificates) != 1 {
		t.Fatalf("expected 1 certificate, got %d", len(a.certificates))
	}
	if a.certificates[0].Type != uint(common.CertificateTypeVoteDelegation) {
		t.Errorf("expected type %d, got %d", common.CertificateTypeVoteDelegation, a.certificates[0].Type)
	}
}

func TestDelegateVoteFromBech32(t *testing.T) {
	cc := setupFixedContext()
	a := New(cc)
	drep := common.Drep{Type: common.DrepTypeNoConfidence}

	a, err := a.DelegateVote(validTestAddrBech32, drep)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.certificates) != 1 {
		t.Fatalf("expected 1 certificate, got %d", len(a.certificates))
	}
}

func TestDelegateStakeAndVote(t *testing.T) {
	cc := setupFixedContext()
	a := New(cc)

	tAddr := testAddress(t)
	cred := common.Credential{CredType: 0, Credential: tAddr.StakeKeyHash()}
	var poolHash common.Blake2b224
	poolHash[0] = 0xaa
	drep := common.Drep{Type: common.DrepTypeAbstain}

	a, err := a.DelegateStakeAndVote(cred, poolHash, drep)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.certificates) != 1 {
		t.Fatalf("expected 1 certificate, got %d", len(a.certificates))
	}
	if a.certificates[0].Type != uint(common.CertificateTypeStakeVoteDelegation) {
		t.Errorf("expected type %d, got %d", common.CertificateTypeStakeVoteDelegation, a.certificates[0].Type)
	}
}

func TestDelegateStakeAndVoteFromBech32(t *testing.T) {
	cc := setupFixedContext()
	a := New(cc)
	var poolHash common.Blake2b224
	poolHash[0] = 0xee
	drep := common.Drep{Type: common.DrepTypeAbstain}

	a, err := a.DelegateStakeAndVote(validTestAddrBech32, poolHash, drep)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.certificates) != 1 {
		t.Fatalf("expected 1 certificate, got %d", len(a.certificates))
	}
}

func TestRegisterAndDelegateVote(t *testing.T) {
	cc := setupFixedContext()
	a := New(cc)

	tAddr := testAddress(t)
	cred := common.Credential{CredType: 0, Credential: tAddr.StakeKeyHash()}
	drep := common.Drep{Type: common.DrepTypeAbstain}

	a, err := a.RegisterAndDelegateVote(cred, drep, StakeDeposit)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.certificates) != 1 {
		t.Fatalf("expected 1 certificate, got %d", len(a.certificates))
	}
	if a.certificates[0].Type != uint(common.CertificateTypeVoteRegistrationDelegation) {
		t.Errorf("expected type %d, got %d", common.CertificateTypeVoteRegistrationDelegation, a.certificates[0].Type)
	}
}

func TestRegisterAndDelegateVoteFromBech32(t *testing.T) {
	cc := setupFixedContext()
	a := New(cc)
	drep := common.Drep{Type: common.DrepTypeAbstain}

	a, err := a.RegisterAndDelegateVote(validTestAddrBech32, drep, StakeDeposit)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.certificates) != 1 {
		t.Fatalf("expected 1 certificate, got %d", len(a.certificates))
	}
}

func TestRegisterAndDelegateStakeAndVote(t *testing.T) {
	cc := setupFixedContext()
	a := New(cc)

	tAddr := testAddress(t)
	cred := common.Credential{CredType: 0, Credential: tAddr.StakeKeyHash()}
	var poolHash common.Blake2b224
	poolHash[0] = 0xaa
	drep := common.Drep{Type: common.DrepTypeAbstain}

	a, err := a.RegisterAndDelegateStakeAndVote(cred, poolHash, drep, StakeDeposit)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.certificates) != 1 {
		t.Fatalf("expected 1 certificate, got %d", len(a.certificates))
	}
	if a.certificates[0].Type != uint(common.CertificateTypeStakeVoteRegistrationDelegation) {
		t.Errorf("expected type %d, got %d", common.CertificateTypeStakeVoteRegistrationDelegation, a.certificates[0].Type)
	}
}

func TestRegisterAndDelegateStakeAndVoteFromBech32(t *testing.T) {
	cc := setupFixedContext()
	a := New(cc)
	var poolHash common.Blake2b224
	poolHash[0] = 0xaa
	drep := common.Drep{Type: common.DrepTypeAbstain}

	a, err := a.RegisterAndDelegateStakeAndVote(validTestAddrBech32, poolHash, drep, StakeDeposit)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.certificates) != 1 {
		t.Fatalf("expected 1 certificate, got %d", len(a.certificates))
	}
}

func TestRegisterPool(t *testing.T) {
	cc := setupFixedContext()
	a := New(cc)

	var operator common.Blake2b224
	operator[0] = 0x01
	params := common.PoolRegistrationCertificate{
		Operator: operator,
		Pledge:   1000000,
		Cost:     340000000,
	}

	a.RegisterPool(params)
	if len(a.certificates) != 1 {
		t.Fatalf("expected 1 certificate, got %d", len(a.certificates))
	}
	if a.certificates[0].Type != uint(common.CertificateTypePoolRegistration) {
		t.Errorf("expected type %d, got %d", common.CertificateTypePoolRegistration, a.certificates[0].Type)
	}
}

func TestDeregisterPool(t *testing.T) {
	cc := setupFixedContext()
	a := New(cc)

	var poolHash common.Blake2b224
	poolHash[0] = 0x01

	a.DeregisterPool(poolHash, 100)
	if len(a.certificates) != 1 {
		t.Fatalf("expected 1 certificate, got %d", len(a.certificates))
	}
	if a.certificates[0].Type != uint(common.CertificateTypePoolRetirement) {
		t.Errorf("expected type %d, got %d", common.CertificateTypePoolRetirement, a.certificates[0].Type)
	}
}

func TestSetCertificates(t *testing.T) {
	cc := setupFixedContext()
	a := New(cc)

	certs := []common.CertificateWrapper{
		{Type: uint(common.CertificateTypeStakeRegistration)},
		{Type: uint(common.CertificateTypeStakeDelegation)},
	}
	a.SetCertificates(certs)
	if len(a.certificates) != 2 {
		t.Errorf("expected 2 certificates, got %d", len(a.certificates))
	}
}

func TestCertificateDepositAdjustment(t *testing.T) {
	cc := setupFixedContext()
	a := New(cc)

	// Registration adds deposit
	tAddr := testAddress(t)
	cred := common.Credential{CredType: 0, Credential: tAddr.StakeKeyHash()}
	a, err := a.RegisterStake(&cred)
	if err != nil {
		t.Fatal(err)
	}
	adj := a.certificateDepositAdjustment(StakeDeposit)
	if adj != StakeDeposit {
		t.Errorf("expected deposit of %d, got %d", StakeDeposit, adj)
	}

	// Deregistration subtracts deposit
	a, err = a.DeregisterStake(&cred)
	if err != nil {
		t.Fatal(err)
	}
	adj = a.certificateDepositAdjustment(StakeDeposit)
	if adj != 0 {
		t.Errorf("expected net 0 (reg+dereg), got %d", adj)
	}
}

func TestCertificateDepositAdjustmentDeregOnly(t *testing.T) {
	cc := setupFixedContext()
	a := New(cc)

	tAddr := testAddress(t)
	cred := common.Credential{CredType: 0, Credential: tAddr.StakeKeyHash()}
	a, err := a.DeregisterStake(&cred)
	if err != nil {
		t.Fatal(err)
	}
	adj := a.certificateDepositAdjustment(StakeDeposit)
	if adj != -StakeDeposit {
		t.Errorf("expected deposit refund of %d, got %d", -StakeDeposit, adj)
	}
}

func TestGetStakeCredentialFromAddress(t *testing.T) {
	addr := testAddress(t)
	cred, err := GetStakeCredentialFromAddress(addr)
	if err != nil {
		t.Fatal(err)
	}
	if cred.CredType != 0 {
		t.Errorf("expected CredType 0, got %d", cred.CredType)
	}
	if cred.Credential == (common.Blake2b224{}) {
		t.Error("expected non-zero credential hash")
	}
}

func TestGetStakeCredentialFromWallet(t *testing.T) {
	cc := setupFixedContext()
	addr := testAddress(t)
	w := NewExternalWallet(addr)
	a := New(cc).SetWallet(w)

	cred, err := a.GetStakeCredentialFromWallet()
	if err != nil {
		t.Fatal(err)
	}
	if cred.Credential == (common.Blake2b224{}) {
		t.Error("expected non-zero credential hash")
	}
}

func TestGetStakeCredentialFromWalletNoWallet(t *testing.T) {
	cc := setupFixedContext()
	a := New(cc)
	_, err := a.GetStakeCredentialFromWallet()
	if err == nil {
		t.Error("expected error when no wallet")
	}
}

func TestCompleteWithStakeRegistration(t *testing.T) {
	cc := fixed.NewEmptyFixedChainContext()
	addr := testAddress(t)
	// Need enough to cover deposit + payment + fee
	addTestUtxo(cc, addr, 20_000_000, 0x01, 0)

	w := NewExternalWallet(addr)
	a := New(cc).SetWallet(w).SetTtl(50000000)

	p, err := NewPayment(validTestAddrBech32, 2_000_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	a.AddPayment(p)
	cred := common.Credential{CredType: 0, Credential: addr.StakeKeyHash()}
	a, err = a.RegisterStake(&cred)
	if err != nil {
		t.Fatal(err)
	}

	a, err = a.Complete()
	if err != nil {
		t.Fatal(err)
	}

	tx := a.GetTx()
	if tx == nil {
		t.Fatal("expected non-nil transaction")
	}
	if len(tx.Body.TxCertificates) != 1 {
		t.Errorf("expected 1 certificate in tx body, got %d", len(tx.Body.TxCertificates))
	}
}

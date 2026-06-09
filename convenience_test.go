package apollo

import (
	"bytes"
	"testing"

	"github.com/blinklabs-io/gouroboros/ledger/common"
)

// --- Bech32 Convenience Method Tests ---

func TestAddInputAddressFromBech32(t *testing.T) {
	cc := setupFixedContext()
	a := New(cc)

	a, err := a.AddInputAddressFromBech32(validTestAddrBech32)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.inputAddresses) != 1 {
		t.Fatalf("expected 1 input address, got %d", len(a.inputAddresses))
	}
}

func TestAddInputAddressFromBech32Invalid(t *testing.T) {
	cc := setupFixedContext()
	a := New(cc)

	_, err := a.AddInputAddressFromBech32("not-a-valid-address")
	if err == nil {
		t.Error("expected error for invalid bech32")
	}
}

func TestPayToAddressBech32(t *testing.T) {
	cc := setupFixedContext()
	a := New(cc)

	a, err := a.PayToAddressBech32(validTestAddrBech32, 2_000_000)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.payments) != 1 {
		t.Fatalf("expected 1 payment, got %d", len(a.payments))
	}
}

func TestPayToAddressBech32WithUnits(t *testing.T) {
	cc := setupFixedContext()
	a := New(cc)
	unit := NewUnit("a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4", "746f6b656e", 50)

	a, err := a.PayToAddressBech32(validTestAddrBech32, 3_000_000, unit)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.payments) != 1 {
		t.Fatalf("expected 1 payment, got %d", len(a.payments))
	}
}

func TestPayToAddressBech32Invalid(t *testing.T) {
	cc := setupFixedContext()
	a := New(cc)

	_, err := a.PayToAddressBech32("invalid", 1_000_000)
	if err == nil {
		t.Error("expected error for invalid bech32")
	}
}

func TestSetChangeAddressBech32(t *testing.T) {
	cc := setupFixedContext()
	a := New(cc)

	a, err := a.SetChangeAddressBech32(validTestAddrBech32)
	if err != nil {
		t.Fatal(err)
	}
	if a.changeAddress == nil {
		t.Fatal("expected change address to be set")
	}
}

func TestSetChangeAddressBech32Invalid(t *testing.T) {
	cc := setupFixedContext()
	a := New(cc)

	_, err := a.SetChangeAddressBech32("bad-address")
	if err == nil {
		t.Error("expected error for invalid bech32")
	}
}

// --- Datum Convenience Method Tests ---

func TestAttachDatum(t *testing.T) {
	cc := setupFixedContext()
	a := New(cc)
	datum := common.Datum{}

	a.AttachDatum(&datum)
	if len(a.datums) != 1 {
		t.Errorf("expected 1 datum, got %d", len(a.datums))
	}
}

func TestAttachDatumNil(t *testing.T) {
	cc := setupFixedContext()
	a := New(cc)

	a.AttachDatum(nil)
	if len(a.datums) != 0 {
		t.Errorf("expected 0 datums for nil, got %d", len(a.datums))
	}
}

func TestAttachDatumMatchesAddDatum(t *testing.T) {
	cc := setupFixedContext()
	datum := common.Datum{}

	a1 := New(cc)
	a1.AddDatum(&datum)

	a2 := New(cc)
	a2.AttachDatum(&datum)

	if len(a1.datums) != len(a2.datums) {
		t.Errorf("AddDatum and AttachDatum produced different results: %d vs %d", len(a1.datums), len(a2.datums))
	}
}

func TestPayToContractAsHash(t *testing.T) {
	cc := setupFixedContext()
	a := New(cc)
	addr := testAddress(t)
	datumHash := make([]byte, 32)
	datumHash[0] = 0xAA

	a.PayToContractAsHash(addr, datumHash, 5_000_000)
	if len(a.payments) != 1 {
		t.Fatalf("expected 1 payment, got %d", len(a.payments))
	}
	p, ok := a.payments[0].(*Payment)
	if !ok {
		t.Fatal("expected *Payment type")
	}
	if !bytes.Equal(p.DatumHash, datumHash) {
		t.Error("datum hash mismatch")
	}
	// PayToContractAsHash should NOT add datum to witness set
	if len(a.datums) != 0 {
		t.Errorf("expected 0 datums in witness set, got %d", len(a.datums))
	}
}

func TestPayToContractAsHashWithUnits(t *testing.T) {
	cc := setupFixedContext()
	a := New(cc)
	addr := testAddress(t)
	datumHash := make([]byte, 32)
	datumHash[0] = 0xBB
	unit := NewUnit("a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4", "746f6b656e", 10)

	a.PayToContractAsHash(addr, datumHash, 3_000_000, unit)
	if len(a.payments) != 1 {
		t.Fatalf("expected 1 payment, got %d", len(a.payments))
	}
	p, ok := a.payments[0].(*Payment)
	if !ok {
		t.Fatal("expected *Payment type")
	}
	if len(p.Units) != 1 {
		t.Errorf("expected 1 unit, got %d", len(p.Units))
	}
	if p.Lovelace != 3_000_000 {
		t.Errorf("expected 3000000 lovelace, got %d", p.Lovelace)
	}
}

// --- Version-Specific Reference Script Tests ---

func TestPayToAddressWithV1ReferenceScript(t *testing.T) {
	cc := setupFixedContext()
	a := New(cc)
	addr := testAddress(t)
	script := common.PlutusV1Script([]byte{0x01, 0x02, 0x03})

	a, err := a.PayToAddressWithV1ReferenceScript(addr, 5_000_000, script)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.payments) != 1 {
		t.Fatalf("expected 1 payment, got %d", len(a.payments))
	}
	p, ok := a.payments[0].(*Payment)
	if !ok {
		t.Fatal("expected *Payment type")
	}
	if p.ScriptRef == nil {
		t.Error("expected script ref to be set")
	}
	if p.ScriptRef.Type != 1 {
		t.Errorf("expected script ref type 1 (V1), got %d", p.ScriptRef.Type)
	}
}

func TestPayToAddressWithV2ReferenceScript(t *testing.T) {
	cc := setupFixedContext()
	a := New(cc)
	addr := testAddress(t)
	script := common.PlutusV2Script([]byte{0x01, 0x02, 0x03})

	a, err := a.PayToAddressWithV2ReferenceScript(addr, 5_000_000, script)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.payments) != 1 {
		t.Fatalf("expected 1 payment, got %d", len(a.payments))
	}
	p, ok := a.payments[0].(*Payment)
	if !ok {
		t.Fatal("expected *Payment type")
	}
	if p.ScriptRef == nil {
		t.Error("expected script ref to be set")
	}
	if p.ScriptRef.Type != 2 {
		t.Errorf("expected script ref type 2 (V2), got %d", p.ScriptRef.Type)
	}
}

func TestPayToAddressWithV3ReferenceScript(t *testing.T) {
	cc := setupFixedContext()
	a := New(cc)
	addr := testAddress(t)
	script := common.PlutusV3Script([]byte{0x01, 0x02, 0x03})

	a, err := a.PayToAddressWithV3ReferenceScript(addr, 5_000_000, script)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.payments) != 1 {
		t.Fatalf("expected 1 payment, got %d", len(a.payments))
	}
	p, ok := a.payments[0].(*Payment)
	if !ok {
		t.Fatal("expected *Payment type")
	}
	if p.ScriptRef == nil {
		t.Error("expected script ref to be set")
	}
	if p.ScriptRef.Type != 3 {
		t.Errorf("expected script ref type 3 (V3), got %d", p.ScriptRef.Type)
	}
}

func TestPayToContractWithReferenceScript(t *testing.T) {
	cc := setupFixedContext()
	a := New(cc)
	addr := testAddress(t)
	datum := common.Datum{}
	script := common.PlutusV2Script([]byte{0x04, 0x05})

	a, err := a.PayToContractWithReferenceScript(addr, &datum, 10_000_000, script)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.payments) != 1 {
		t.Fatalf("expected 1 payment, got %d", len(a.payments))
	}
	p, ok := a.payments[0].(*Payment)
	if !ok {
		t.Fatal("expected *Payment type")
	}
	if p.ScriptRef == nil {
		t.Error("expected script ref to be set")
	}
	if p.Datum == nil {
		t.Error("expected datum to be set")
	}
	if !p.IsInline {
		t.Error("expected inline datum")
	}
}

func TestPayToContractWithReferenceScriptNilDatum(t *testing.T) {
	cc := setupFixedContext()
	a := New(cc)
	addr := testAddress(t)
	script := common.PlutusV2Script([]byte{0x04, 0x05})

	a, err := a.PayToContractWithReferenceScript(addr, nil, 10_000_000, script)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.payments) != 1 {
		t.Fatalf("expected 1 payment, got %d", len(a.payments))
	}
	p, ok := a.payments[0].(*Payment)
	if !ok {
		t.Fatal("expected *Payment type")
	}
	if p.Datum != nil {
		t.Error("expected nil datum")
	}
	if p.ScriptRef == nil {
		t.Error("expected script ref to be set")
	}
}

func TestPayToContractWithV1ReferenceScript(t *testing.T) {
	cc := setupFixedContext()
	a := New(cc)
	addr := testAddress(t)
	datum := common.Datum{}
	script := common.PlutusV1Script([]byte{0x01})

	a, err := a.PayToContractWithV1ReferenceScript(addr, &datum, 5_000_000, script)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.payments) != 1 {
		t.Fatalf("expected 1 payment, got %d", len(a.payments))
	}
	p := a.payments[0].(*Payment)
	if p.ScriptRef.Type != 1 {
		t.Errorf("expected script ref type 1 (V1), got %d", p.ScriptRef.Type)
	}
}

func TestPayToContractWithV2ReferenceScript(t *testing.T) {
	cc := setupFixedContext()
	a := New(cc)
	addr := testAddress(t)
	datum := common.Datum{}
	script := common.PlutusV2Script([]byte{0x02})

	a, err := a.PayToContractWithV2ReferenceScript(addr, &datum, 5_000_000, script)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.payments) != 1 {
		t.Fatalf("expected 1 payment, got %d", len(a.payments))
	}
	p := a.payments[0].(*Payment)
	if p.ScriptRef.Type != 2 {
		t.Errorf("expected script ref type 2 (V2), got %d", p.ScriptRef.Type)
	}
}

func TestPayToContractWithV3ReferenceScript(t *testing.T) {
	cc := setupFixedContext()
	a := New(cc)
	addr := testAddress(t)
	datum := common.Datum{}
	script := common.PlutusV3Script([]byte{0x03})

	a, err := a.PayToContractWithV3ReferenceScript(addr, &datum, 5_000_000, script)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.payments) != 1 {
		t.Fatalf("expected 1 payment, got %d", len(a.payments))
	}
	p := a.payments[0].(*Payment)
	if p.ScriptRef.Type != 3 {
		t.Errorf("expected script ref type 3 (V3), got %d", p.ScriptRef.Type)
	}
}

// --- Staking FromAddress Convenience Tests ---
// Note: The existing staking_test.go tests the unified methods with bech32 strings
// passed directly. These tests exercise the explicit typed convenience wrappers.

func TestConvenienceRegisterStakeFromAddress(t *testing.T) {
	cc := setupFixedContext()
	a := New(cc)
	addr := testAddress(t)

	a, err := a.RegisterStakeFromAddress(addr)
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

func TestConvenienceRegisterStakeFromBech32(t *testing.T) {
	cc := setupFixedContext()
	a := New(cc)

	a, err := a.RegisterStakeFromBech32(validTestAddrBech32)
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

func TestConvenienceRegisterStakeFromBech32Invalid(t *testing.T) {
	cc := setupFixedContext()
	a := New(cc)

	_, err := a.RegisterStakeFromBech32("not-valid")
	if err == nil {
		t.Error("expected error for invalid bech32")
	}
}

func TestConvenienceDeregisterStakeFromAddress(t *testing.T) {
	cc := setupFixedContext()
	a := New(cc)
	addr := testAddress(t)

	a, err := a.DeregisterStakeFromAddress(addr)
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

func TestConvenienceDeregisterStakeFromBech32(t *testing.T) {
	cc := setupFixedContext()
	a := New(cc)

	a, err := a.DeregisterStakeFromBech32(validTestAddrBech32)
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

func TestConvenienceDelegateStakeFromAddress(t *testing.T) {
	cc := setupFixedContext()
	a := New(cc)
	addr := testAddress(t)
	var poolHash common.Blake2b224
	poolHash[0] = 0xAA

	a, err := a.DelegateStakeFromAddress(addr, poolHash)
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

func TestConvenienceDelegateStakeFromBech32(t *testing.T) {
	cc := setupFixedContext()
	a := New(cc)
	var poolHash common.Blake2b224
	poolHash[0] = 0xBB

	a, err := a.DelegateStakeFromBech32(validTestAddrBech32, poolHash)
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

func TestConvenienceDelegateVoteFromAddress(t *testing.T) {
	cc := setupFixedContext()
	a := New(cc)
	addr := testAddress(t)
	drep := common.Drep{Type: common.DrepTypeAbstain}

	a, err := a.DelegateVoteFromAddress(addr, drep)
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

func TestConvenienceDelegateVoteFromBech32(t *testing.T) {
	cc := setupFixedContext()
	a := New(cc)
	drep := common.Drep{Type: common.DrepTypeNoConfidence}

	a, err := a.DelegateVoteFromBech32(validTestAddrBech32, drep)
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

func TestConvenienceDelegateStakeAndVoteFromAddress(t *testing.T) {
	cc := setupFixedContext()
	a := New(cc)
	addr := testAddress(t)
	var poolHash common.Blake2b224
	poolHash[0] = 0xCC
	drep := common.Drep{Type: common.DrepTypeAbstain}

	a, err := a.DelegateStakeAndVoteFromAddress(addr, poolHash, drep)
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

func TestConvenienceDelegateStakeAndVoteFromBech32(t *testing.T) {
	cc := setupFixedContext()
	a := New(cc)
	var poolHash common.Blake2b224
	poolHash[0] = 0xDD
	drep := common.Drep{Type: common.DrepTypeAbstain}

	a, err := a.DelegateStakeAndVoteFromBech32(validTestAddrBech32, poolHash, drep)
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

func TestConvenienceRegisterAndDelegateStakeFromAddress(t *testing.T) {
	cc := setupFixedContext()
	a := New(cc)
	addr := testAddress(t)
	var poolHash common.Blake2b224
	poolHash[0] = 0xEE

	a, err := a.RegisterAndDelegateStakeFromAddress(addr, poolHash, StakeDeposit)
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

func TestConvenienceRegisterAndDelegateStakeFromBech32(t *testing.T) {
	cc := setupFixedContext()
	a := New(cc)
	var poolHash common.Blake2b224
	poolHash[0] = 0xFF

	a, err := a.RegisterAndDelegateStakeFromBech32(validTestAddrBech32, poolHash, StakeDeposit)
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

func TestConvenienceRegisterAndDelegateVoteFromAddress(t *testing.T) {
	cc := setupFixedContext()
	a := New(cc)
	addr := testAddress(t)
	drep := common.Drep{Type: common.DrepTypeAbstain}

	a, err := a.RegisterAndDelegateVoteFromAddress(addr, drep, StakeDeposit)
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

func TestConvenienceRegisterAndDelegateVoteFromBech32(t *testing.T) {
	cc := setupFixedContext()
	a := New(cc)
	drep := common.Drep{Type: common.DrepTypeAbstain}

	a, err := a.RegisterAndDelegateVoteFromBech32(validTestAddrBech32, drep, StakeDeposit)
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

func TestConvenienceRegisterAndDelegateStakeAndVoteFromAddress(t *testing.T) {
	cc := setupFixedContext()
	a := New(cc)
	addr := testAddress(t)
	var poolHash common.Blake2b224
	poolHash[0] = 0xAA
	drep := common.Drep{Type: common.DrepTypeAbstain}

	a, err := a.RegisterAndDelegateStakeAndVoteFromAddress(addr, poolHash, drep, StakeDeposit)
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

func TestConvenienceRegisterAndDelegateStakeAndVoteFromBech32(t *testing.T) {
	cc := setupFixedContext()
	a := New(cc)
	var poolHash common.Blake2b224
	poolHash[0] = 0xBB
	drep := common.Drep{Type: common.DrepTypeAbstain}

	a, err := a.RegisterAndDelegateStakeAndVoteFromBech32(validTestAddrBech32, poolHash, drep, StakeDeposit)
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

// --- Equivalence Tests: FromAddress/FromBech32 produce same result as unified ---

func TestFromAddressMatchesUnified(t *testing.T) {
	addr := testAddress(t)
	var poolHash common.Blake2b224
	poolHash[0] = 0x11
	drep := common.Drep{Type: common.DrepTypeAbstain}

	tests := []struct {
		name     string
		fromAddr func(*Apollo) (*Apollo, error)
		unified  func(*Apollo) (*Apollo, error)
	}{
		{
			"RegisterStake",
			func(a *Apollo) (*Apollo, error) { return a.RegisterStakeFromAddress(addr) },
			func(a *Apollo) (*Apollo, error) { return a.RegisterStake(addr) },
		},
		{
			"DeregisterStake",
			func(a *Apollo) (*Apollo, error) { return a.DeregisterStakeFromAddress(addr) },
			func(a *Apollo) (*Apollo, error) { return a.DeregisterStake(addr) },
		},
		{
			"DelegateStake",
			func(a *Apollo) (*Apollo, error) { return a.DelegateStakeFromAddress(addr, poolHash) },
			func(a *Apollo) (*Apollo, error) { return a.DelegateStake(addr, poolHash) },
		},
		{
			"DelegateVote",
			func(a *Apollo) (*Apollo, error) { return a.DelegateVoteFromAddress(addr, drep) },
			func(a *Apollo) (*Apollo, error) { return a.DelegateVote(addr, drep) },
		},
		{
			"DelegateStakeAndVote",
			func(a *Apollo) (*Apollo, error) { return a.DelegateStakeAndVoteFromAddress(addr, poolHash, drep) },
			func(a *Apollo) (*Apollo, error) { return a.DelegateStakeAndVote(addr, poolHash, drep) },
		},
		{
			"RegisterAndDelegateStake",
			func(a *Apollo) (*Apollo, error) { return a.RegisterAndDelegateStakeFromAddress(addr, poolHash, StakeDeposit) },
			func(a *Apollo) (*Apollo, error) { return a.RegisterAndDelegateStake(addr, poolHash, StakeDeposit) },
		},
		{
			"RegisterAndDelegateVote",
			func(a *Apollo) (*Apollo, error) { return a.RegisterAndDelegateVoteFromAddress(addr, drep, StakeDeposit) },
			func(a *Apollo) (*Apollo, error) { return a.RegisterAndDelegateVote(addr, drep, StakeDeposit) },
		},
		{
			"RegisterAndDelegateStakeAndVote",
			func(a *Apollo) (*Apollo, error) {
				return a.RegisterAndDelegateStakeAndVoteFromAddress(addr, poolHash, drep, StakeDeposit)
			},
			func(a *Apollo) (*Apollo, error) {
				return a.RegisterAndDelegateStakeAndVote(addr, poolHash, drep, StakeDeposit)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cc := setupFixedContext()

			a1 := New(cc)
			a1, err := tt.fromAddr(a1)
			if err != nil {
				t.Fatalf("FromAddress: %v", err)
			}

			a2 := New(cc)
			a2, err = tt.unified(a2)
			if err != nil {
				t.Fatalf("unified: %v", err)
			}

			if len(a1.certificates) != len(a2.certificates) {
				t.Errorf("certificate count mismatch: FromAddress=%d, unified=%d",
					len(a1.certificates), len(a2.certificates))
				return
			}
			if len(a1.certificates) > 0 && a1.certificates[0].Type != a2.certificates[0].Type {
				t.Errorf("certificate type mismatch: FromAddress=%d, unified=%d",
					a1.certificates[0].Type, a2.certificates[0].Type)
			}
		})
	}
}

func TestFromBech32MatchesUnified(t *testing.T) {
	var poolHash common.Blake2b224
	poolHash[0] = 0x22
	drep := common.Drep{Type: common.DrepTypeAbstain}

	tests := []struct {
		name       string
		fromBech32 func(*Apollo) (*Apollo, error)
		unified    func(*Apollo) (*Apollo, error)
	}{
		{
			"RegisterStake",
			func(a *Apollo) (*Apollo, error) { return a.RegisterStakeFromBech32(validTestAddrBech32) },
			func(a *Apollo) (*Apollo, error) { return a.RegisterStake(validTestAddrBech32) },
		},
		{
			"DeregisterStake",
			func(a *Apollo) (*Apollo, error) { return a.DeregisterStakeFromBech32(validTestAddrBech32) },
			func(a *Apollo) (*Apollo, error) { return a.DeregisterStake(validTestAddrBech32) },
		},
		{
			"DelegateStake",
			func(a *Apollo) (*Apollo, error) { return a.DelegateStakeFromBech32(validTestAddrBech32, poolHash) },
			func(a *Apollo) (*Apollo, error) { return a.DelegateStake(validTestAddrBech32, poolHash) },
		},
		{
			"DelegateVote",
			func(a *Apollo) (*Apollo, error) { return a.DelegateVoteFromBech32(validTestAddrBech32, drep) },
			func(a *Apollo) (*Apollo, error) { return a.DelegateVote(validTestAddrBech32, drep) },
		},
		{
			"DelegateStakeAndVote",
			func(a *Apollo) (*Apollo, error) {
				return a.DelegateStakeAndVoteFromBech32(validTestAddrBech32, poolHash, drep)
			},
			func(a *Apollo) (*Apollo, error) {
				return a.DelegateStakeAndVote(validTestAddrBech32, poolHash, drep)
			},
		},
		{
			"RegisterAndDelegateStake",
			func(a *Apollo) (*Apollo, error) {
				return a.RegisterAndDelegateStakeFromBech32(validTestAddrBech32, poolHash, StakeDeposit)
			},
			func(a *Apollo) (*Apollo, error) {
				return a.RegisterAndDelegateStake(validTestAddrBech32, poolHash, StakeDeposit)
			},
		},
		{
			"RegisterAndDelegateVote",
			func(a *Apollo) (*Apollo, error) {
				return a.RegisterAndDelegateVoteFromBech32(validTestAddrBech32, drep, StakeDeposit)
			},
			func(a *Apollo) (*Apollo, error) {
				return a.RegisterAndDelegateVote(validTestAddrBech32, drep, StakeDeposit)
			},
		},
		{
			"RegisterAndDelegateStakeAndVote",
			func(a *Apollo) (*Apollo, error) {
				return a.RegisterAndDelegateStakeAndVoteFromBech32(validTestAddrBech32, poolHash, drep, StakeDeposit)
			},
			func(a *Apollo) (*Apollo, error) {
				return a.RegisterAndDelegateStakeAndVote(validTestAddrBech32, poolHash, drep, StakeDeposit)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cc := setupFixedContext()

			a1 := New(cc)
			a1, err := tt.fromBech32(a1)
			if err != nil {
				t.Fatalf("FromBech32: %v", err)
			}

			a2 := New(cc)
			a2, err = tt.unified(a2)
			if err != nil {
				t.Fatalf("unified: %v", err)
			}

			if len(a1.certificates) != len(a2.certificates) {
				t.Errorf("certificate count mismatch: FromBech32=%d, unified=%d",
					len(a1.certificates), len(a2.certificates))
				return
			}
			if len(a1.certificates) > 0 && a1.certificates[0].Type != a2.certificates[0].Type {
				t.Errorf("certificate type mismatch: FromBech32=%d, unified=%d",
					a1.certificates[0].Type, a2.certificates[0].Type)
			}
		})
	}
}

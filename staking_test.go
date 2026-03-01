package apollo_test

import (
	"bytes"
	"testing"

	"github.com/Salvionied/apollo"
	"github.com/Salvionied/apollo/serialization"
	"github.com/Salvionied/apollo/serialization/Address"
	"github.com/Salvionied/apollo/serialization/Certificate"
	"github.com/Salvionied/apollo/serialization/PlutusData"
	"github.com/Salvionied/apollo/serialization/Redeemer"
	testutils "github.com/Salvionied/apollo/testUtils"
	"github.com/Salvionied/apollo/txBuilding/Backend/FixedChainContext"
)

// addrWithStake is a testnet base address (payment + staking part).
const addrWithStake = "addr_test1qz2fxv2umyhttkxyxp8x0dlpdt3k6cwng5pxj3jhsydzer3jcu5d8ps7zex2k2xt3uqxgjqnnj83ws8lhrn648jjxtwq2ytjqp"

// addrNoStake is an enterprise address without staking part.
const addrNoStake = "addr_test1vr2p8st5t5cxqglyjky7vk98k7jtfhdpvhl4e97cezuhn0cqcexl7"

func makeStakeCredential(
	t *testing.T,
) (*Certificate.Credential, Address.Address) {
	t.Helper()
	decoded, err := Address.DecodeAddress(addrWithStake)
	if err != nil {
		t.Fatalf("decode address: %v", err)
	}
	cred, err := apollo.GetStakeCredentialFromAddress(decoded)
	if err != nil {
		t.Fatalf("get stake credential: %v", err)
	}
	return cred, decoded
}

func newTestApollo(t *testing.T) *apollo.Apollo {
	t.Helper()
	cc := FixedChainContext.InitFixedChainContext()
	a := apollo.New(&cc)
	utxos := testutils.InitUtxos()
	decoded, _ := Address.DecodeAddress(
		testutils.TESTADDRESS,
	)
	a = a.
		AddLoadedUTxOs(utxos...).
		SetChangeAddress(decoded).
		SetTtl(300)
	return a
}

// --- GetStakeCredentialFromAddress tests ---

func TestGetStakeCredentialFromAddress(t *testing.T) {
	decoded, err := Address.DecodeAddress(addrWithStake)
	if err != nil {
		t.Fatalf("decode address: %v", err)
	}
	cred, err := apollo.GetStakeCredentialFromAddress(decoded)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cred.Code != 0 {
		t.Fatalf("expected code 0, got %d", cred.Code)
	}
	if len(cred.Hash.Payload) != 28 {
		t.Fatalf(
			"expected 28-byte hash, got %d",
			len(cred.Hash.Payload),
		)
	}
	if !bytes.Equal(cred.Hash.Payload, decoded.StakingPart) {
		t.Fatal("credential hash does not match staking part")
	}
}

func TestGetStakeCredentialFromAddress_NoStake(t *testing.T) {
	decoded, err := Address.DecodeAddress(addrNoStake)
	if err != nil {
		t.Fatalf("decode address: %v", err)
	}
	_, err = apollo.GetStakeCredentialFromAddress(decoded)
	if err == nil {
		t.Fatal("expected error for address without staking part")
	}
}

// --- SetCertificates tests ---

func TestSetCertificates(t *testing.T) {
	a := newTestApollo(t)
	cred, _ := makeStakeCredential(t)

	certs := Certificate.NewCertificates(
		Certificate.StakeRegistration{Stake: *cred},
	)
	a = a.SetCertificates(&certs)

	built, _, err := a.Complete()
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	tx := built.GetTx()
	if tx.TransactionBody.Certificates == nil {
		t.Fatal("certificates not set in tx body")
	}
	if len(*tx.TransactionBody.Certificates) != 1 {
		t.Fatalf(
			"expected 1 certificate, got %d",
			len(*tx.TransactionBody.Certificates),
		)
	}
	cert := (*tx.TransactionBody.Certificates)[0]
	if cert.Kind() != 0 {
		t.Fatalf("expected kind 0 (StakeRegistration), got %d", cert.Kind())
	}
}

func TestSetCertificatesMultiple(t *testing.T) {
	a := newTestApollo(t)
	cred, _ := makeStakeCredential(t)
	pool := serialization.PubKeyHash{}
	pool[0] = 0xAB

	certs := Certificate.NewCertificates(
		Certificate.StakeRegistration{Stake: *cred},
		Certificate.StakeDelegation{
			Stake:       *cred,
			PoolKeyHash: pool,
		},
	)
	a = a.SetCertificates(&certs)

	built, _, err := a.Complete()
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	tx := built.GetTx()
	if len(*tx.TransactionBody.Certificates) != 2 {
		t.Fatalf(
			"expected 2 certificates, got %d",
			len(*tx.TransactionBody.Certificates),
		)
	}
	if (*tx.TransactionBody.Certificates)[0].Kind() != 0 {
		t.Fatal("first cert should be StakeRegistration (kind 0)")
	}
	if (*tx.TransactionBody.Certificates)[1].Kind() != 2 {
		t.Fatal("second cert should be StakeDelegation (kind 2)")
	}
}

// --- RegisterStake tests ---

func TestRegisterStake(t *testing.T) {
	a := newTestApollo(t)
	cred, _ := makeStakeCredential(t)

	a, err := a.RegisterStake(cred)
	if err != nil {
		t.Fatalf("RegisterStake: %v", err)
	}

	built, _, err := a.Complete()
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	tx := built.GetTx()
	if tx.TransactionBody.Certificates == nil {
		t.Fatal("certificates nil after RegisterStake")
	}
	certs := *tx.TransactionBody.Certificates
	if len(certs) != 1 {
		t.Fatalf("expected 1 cert, got %d", len(certs))
	}
	if certs[0].Kind() != 0 {
		t.Fatalf(
			"expected StakeRegistration (kind 0), got %d",
			certs[0].Kind(),
		)
	}
	sc := certs[0].StakeCredential()
	if sc == nil {
		t.Fatal("stake credential is nil")
	}
	if !bytes.Equal(sc.Hash.Payload, cred.Hash.Payload) {
		t.Fatal("stake credential hash mismatch")
	}
}

func TestRegisterStakeFromAddress(t *testing.T) {
	a := newTestApollo(t)
	decoded, err := Address.DecodeAddress(addrWithStake)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	a, err = a.RegisterStakeFromAddress(decoded)
	if err != nil {
		t.Fatalf("RegisterStakeFromAddress: %v", err)
	}

	built, _, err := a.Complete()
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	certs := *built.GetTx().TransactionBody.Certificates
	if len(certs) != 1 || certs[0].Kind() != 0 {
		t.Fatal("expected single StakeRegistration cert")
	}
}

func TestRegisterStakeFromBech32(t *testing.T) {
	a := newTestApollo(t)

	a, err := a.RegisterStakeFromBech32(addrWithStake)
	if err != nil {
		t.Fatalf("RegisterStakeFromBech32: %v", err)
	}

	built, _, err := a.Complete()
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	certs := *built.GetTx().TransactionBody.Certificates
	if len(certs) != 1 || certs[0].Kind() != 0 {
		t.Fatal("expected single StakeRegistration cert")
	}
}

func TestRegisterStakeFromAddress_NoStake(t *testing.T) {
	a := newTestApollo(t)
	decoded, _ := Address.DecodeAddress(addrNoStake)

	_, err := a.RegisterStakeFromAddress(decoded)
	if err == nil {
		t.Fatal("expected error for address without staking part")
	}
}

// --- DeregisterStake tests ---

func TestDeregisterStake(t *testing.T) {
	a := newTestApollo(t)
	cred, _ := makeStakeCredential(t)

	a, err := a.DeregisterStake(cred)
	if err != nil {
		t.Fatalf("DeregisterStake: %v", err)
	}

	built, _, err := a.Complete()
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	certs := *built.GetTx().TransactionBody.Certificates
	if len(certs) != 1 {
		t.Fatalf("expected 1 cert, got %d", len(certs))
	}
	if certs[0].Kind() != 1 {
		t.Fatalf(
			"expected StakeDeregistration (kind 1), got %d",
			certs[0].Kind(),
		)
	}
	sc := certs[0].StakeCredential()
	if sc == nil {
		t.Fatal("stake credential is nil")
	}
	if !bytes.Equal(sc.Hash.Payload, cred.Hash.Payload) {
		t.Fatal("stake credential hash mismatch")
	}
}

func TestDeregisterStakeFromAddress(t *testing.T) {
	a := newTestApollo(t)
	decoded, _ := Address.DecodeAddress(addrWithStake)

	a, err := a.DeregisterStakeFromAddress(decoded)
	if err != nil {
		t.Fatalf("DeregisterStakeFromAddress: %v", err)
	}

	built, _, err := a.Complete()
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	certs := *built.GetTx().TransactionBody.Certificates
	if len(certs) != 1 || certs[0].Kind() != 1 {
		t.Fatal("expected single StakeDeregistration cert")
	}
}

func TestDeregisterStakeFromBech32(t *testing.T) {
	a := newTestApollo(t)

	a, err := a.DeregisterStakeFromBech32(addrWithStake)
	if err != nil {
		t.Fatalf("DeregisterStakeFromBech32: %v", err)
	}

	built, _, err := a.Complete()
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	certs := *built.GetTx().TransactionBody.Certificates
	if len(certs) != 1 || certs[0].Kind() != 1 {
		t.Fatal("expected single StakeDeregistration cert")
	}
}

func TestDeregisterStakeFromAddress_NoStake(t *testing.T) {
	a := newTestApollo(t)
	decoded, _ := Address.DecodeAddress(addrNoStake)

	_, err := a.DeregisterStakeFromAddress(decoded)
	if err == nil {
		t.Fatal("expected error for address without staking part")
	}
}

// --- DelegateStake tests ---

func TestDelegateStake(t *testing.T) {
	a := newTestApollo(t)
	cred, _ := makeStakeCredential(t)
	pool := serialization.PubKeyHash{}
	pool[0] = 0xDE
	pool[1] = 0xAD

	a, err := a.DelegateStake(cred, pool)
	if err != nil {
		t.Fatalf("DelegateStake: %v", err)
	}

	built, _, err := a.Complete()
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	certs := *built.GetTx().TransactionBody.Certificates
	if len(certs) != 1 {
		t.Fatalf("expected 1 cert, got %d", len(certs))
	}
	if certs[0].Kind() != 2 {
		t.Fatalf(
			"expected StakeDelegation (kind 2), got %d",
			certs[0].Kind(),
		)
	}
	sc := certs[0].StakeCredential()
	if sc == nil {
		t.Fatal("stake credential is nil")
	}
	if !bytes.Equal(sc.Hash.Payload, cred.Hash.Payload) {
		t.Fatal("stake credential hash mismatch")
	}
}

func TestDelegateStakeFromAddress(t *testing.T) {
	a := newTestApollo(t)
	decoded, _ := Address.DecodeAddress(addrWithStake)
	pool := serialization.PubKeyHash{}
	pool[0] = 0xBE

	a, err := a.DelegateStakeFromAddress(decoded, pool)
	if err != nil {
		t.Fatalf("DelegateStakeFromAddress: %v", err)
	}

	built, _, err := a.Complete()
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	certs := *built.GetTx().TransactionBody.Certificates
	if len(certs) != 1 || certs[0].Kind() != 2 {
		t.Fatal("expected single StakeDelegation cert")
	}
}

func TestDelegateStakeFromBech32(t *testing.T) {
	a := newTestApollo(t)
	pool := serialization.PubKeyHash{}
	pool[0] = 0xCF

	a, err := a.DelegateStakeFromBech32(addrWithStake, pool)
	if err != nil {
		t.Fatalf("DelegateStakeFromBech32: %v", err)
	}

	built, _, err := a.Complete()
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	certs := *built.GetTx().TransactionBody.Certificates
	if len(certs) != 1 || certs[0].Kind() != 2 {
		t.Fatal("expected single StakeDelegation cert")
	}
}

func TestDelegateStakeFromAddress_NoStake(t *testing.T) {
	a := newTestApollo(t)
	decoded, _ := Address.DecodeAddress(addrNoStake)
	pool := serialization.PubKeyHash{}

	_, err := a.DelegateStakeFromAddress(decoded, pool)
	if err == nil {
		t.Fatal("expected error for address without staking part")
	}
}

// --- RegisterAndDelegateStake tests ---

func TestRegisterAndDelegateStake(t *testing.T) {
	a := newTestApollo(t)
	cred, _ := makeStakeCredential(t)
	pool := serialization.PubKeyHash{}
	pool[0] = 0x42

	a, err := a.RegisterAndDelegateStake(
		cred,
		pool,
		2_000_000,
	)
	if err != nil {
		t.Fatalf("RegisterAndDelegateStake: %v", err)
	}

	built, _, err := a.Complete()
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	certs := *built.GetTx().TransactionBody.Certificates
	if len(certs) != 1 {
		t.Fatalf("expected 1 cert, got %d", len(certs))
	}
	if certs[0].Kind() != 11 {
		t.Fatalf(
			"expected StakeRegDelegCert (kind 11), got %d",
			certs[0].Kind(),
		)
	}
}

func TestRegisterAndDelegateStakeFromBech32(t *testing.T) {
	a := newTestApollo(t)
	pool := serialization.PubKeyHash{}
	pool[0] = 0x43

	a, err := a.RegisterAndDelegateStakeFromBech32(
		addrWithStake,
		pool,
		2_000_000,
	)
	if err != nil {
		t.Fatalf("RegisterAndDelegateStakeFromBech32: %v", err)
	}

	built, _, err := a.Complete()
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	certs := *built.GetTx().TransactionBody.Certificates
	if len(certs) != 1 || certs[0].Kind() != 11 {
		t.Fatal("expected single StakeRegDelegCert (kind 11)")
	}
}

// --- AddWithdrawal tests ---

func TestAddWithdrawal(t *testing.T) {
	a := newTestApollo(t)
	decoded, _ := Address.DecodeAddress(addrWithStake)

	a = a.AddWithdrawal(
		decoded,
		5_000_000,
		PlutusData.PlutusData{},
	)

	built, _, err := a.Complete()
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	tx := built.GetTx()
	if tx.TransactionBody.Withdrawals == nil {
		t.Fatal("withdrawals nil after AddWithdrawal")
	}
	if tx.TransactionBody.Withdrawals.Size() != 1 {
		t.Fatalf(
			"expected 1 withdrawal, got %d",
			tx.TransactionBody.Withdrawals.Size(),
		)
	}
}

func TestAddWithdrawalMultiple(t *testing.T) {
	a := newTestApollo(t)

	// Use two different addresses with staking parts.
	decoded1, _ := Address.DecodeAddress(addrWithStake)
	// Build a second distinct address with different staking part.
	addr2 := Address.Address{
		PaymentPart: decoded1.PaymentPart,
		StakingPart: make([]byte, 28),
		Network:     decoded1.Network,
		AddressType: decoded1.AddressType,
		HeaderByte:  decoded1.HeaderByte,
		Hrp:         decoded1.Hrp,
	}
	addr2.StakingPart[0] = 0xFF

	a = a.AddWithdrawal(
		decoded1,
		1_000_000,
		PlutusData.PlutusData{},
	)
	a = a.AddWithdrawal(
		addr2,
		2_000_000,
		PlutusData.PlutusData{},
	)

	built, _, err := a.Complete()
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if built.GetTx().TransactionBody.Withdrawals.Size() != 2 {
		t.Fatalf(
			"expected 2 withdrawals, got %d",
			built.GetTx().TransactionBody.Withdrawals.Size(),
		)
	}
}

func TestAddWithdrawalWithRedeemer(t *testing.T) {
	a := newTestApollo(t)
	decoded, _ := Address.DecodeAddress(addrWithStake)

	rd := PlutusData.PlutusData{
		TagNr:          121,
		PlutusDataType: PlutusData.PlutusArray,
		Value:          PlutusData.PlutusIndefArray{},
	}

	a = a.AddWithdrawal(decoded, 3_000_000, rd)

	built, _, err := a.Complete()
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	tx := built.GetTx()
	if tx.TransactionBody.Withdrawals == nil {
		t.Fatal("withdrawals nil")
	}
	if tx.TransactionBody.Withdrawals.Size() != 1 {
		t.Fatalf(
			"expected 1 withdrawal, got %d",
			tx.TransactionBody.Withdrawals.Size(),
		)
	}
}

// --- Certificate chaining tests ---

func TestCertificateChaining(t *testing.T) {
	a := newTestApollo(t)
	cred, _ := makeStakeCredential(t)
	pool := serialization.PubKeyHash{}
	pool[0] = 0x99

	// Chain: register then delegate
	a, err := a.RegisterStake(cred)
	if err != nil {
		t.Fatalf("RegisterStake: %v", err)
	}
	a, err = a.DelegateStake(cred, pool)
	if err != nil {
		t.Fatalf("DelegateStake: %v", err)
	}

	built, _, err := a.Complete()
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	certs := *built.GetTx().TransactionBody.Certificates
	if len(certs) != 2 {
		t.Fatalf("expected 2 certs, got %d", len(certs))
	}
	if certs[0].Kind() != 0 {
		t.Fatal("first cert should be StakeRegistration")
	}
	if certs[1].Kind() != 2 {
		t.Fatal("second cert should be StakeDelegation")
	}
}

func TestCertificateWithWithdrawal(t *testing.T) {
	a := newTestApollo(t)
	cred, decoded := makeStakeCredential(t)

	a, err := a.DeregisterStake(cred)
	if err != nil {
		t.Fatalf("DeregisterStake: %v", err)
	}
	a = a.AddWithdrawal(
		decoded,
		1_000_000,
		PlutusData.PlutusData{},
	)

	built, _, err := a.Complete()
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	tx := built.GetTx()
	if tx.TransactionBody.Certificates == nil {
		t.Fatal("certificates nil")
	}
	if len(*tx.TransactionBody.Certificates) != 1 {
		t.Fatal("expected 1 certificate")
	}
	if tx.TransactionBody.Withdrawals == nil {
		t.Fatal("withdrawals nil")
	}
	if tx.TransactionBody.Withdrawals.Size() != 1 {
		t.Fatal("expected 1 withdrawal")
	}
}

// --- Withdrawal package tests ---

func TestWithdrawalNew(t *testing.T) {
	a := newTestApollo(t)
	decoded, _ := Address.DecodeAddress(addrWithStake)

	a = a.AddWithdrawal(
		decoded,
		0,
		PlutusData.PlutusData{},
	)

	built, _, err := a.Complete()
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if built.GetTx().TransactionBody.Withdrawals == nil {
		t.Fatal("withdrawals should not be nil")
	}
}

// --- SetCertificates overwrites previous ---

func TestSetCertificatesOverwrites(t *testing.T) {
	a := newTestApollo(t)
	cred, _ := makeStakeCredential(t)

	certs1 := Certificate.NewCertificates(
		Certificate.StakeRegistration{Stake: *cred},
	)
	a = a.SetCertificates(&certs1)

	certs2 := Certificate.NewCertificates(
		Certificate.StakeDeregistration{Stake: *cred},
	)
	a = a.SetCertificates(&certs2)

	built, _, err := a.Complete()
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	certs := *built.GetTx().TransactionBody.Certificates
	if len(certs) != 1 {
		t.Fatalf("expected 1 cert, got %d", len(certs))
	}
	if certs[0].Kind() != 1 {
		t.Fatalf(
			"expected StakeDeregistration (kind 1), got %d",
			certs[0].Kind(),
		)
	}
}

// --- Redeemer tag verification ---

func TestWithdrawalRedeemerTag(t *testing.T) {
	if Redeemer.REWARD != 3 {
		t.Fatalf(
			"expected REWARD tag = 3, got %d",
			Redeemer.REWARD,
		)
	}
	if Redeemer.CERT != 2 {
		t.Fatalf(
			"expected CERT tag = 2, got %d",
			Redeemer.CERT,
		)
	}
}

// --- Transaction serialization with certificates ---

func TestTxWithCertificatesSerializes(t *testing.T) {
	a := newTestApollo(t)
	cred, _ := makeStakeCredential(t)
	pool := serialization.PubKeyHash{}
	pool[0] = 0x11

	a, err := a.RegisterStake(cred)
	if err != nil {
		t.Fatalf("RegisterStake: %v", err)
	}

	built, _, err := a.Complete()
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}

	txBytes, err := built.GetTx().Bytes()
	if err != nil {
		t.Fatalf("tx Bytes: %v", err)
	}
	if len(txBytes) == 0 {
		t.Fatal("serialized tx is empty")
	}
}

func TestTxWithWithdrawalSerializes(t *testing.T) {
	a := newTestApollo(t)
	decoded, _ := Address.DecodeAddress(addrWithStake)

	a = a.AddWithdrawal(
		decoded,
		5_000_000,
		PlutusData.PlutusData{},
	)

	built, _, err := a.Complete()
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}

	txBytes, err := built.GetTx().Bytes()
	if err != nil {
		t.Fatalf("tx Bytes: %v", err)
	}
	if len(txBytes) == 0 {
		t.Fatal("serialized tx is empty")
	}
}

// --- Combined registration + delegation + withdrawal ---

func TestFullStakingFlow(t *testing.T) {
	a := newTestApollo(t)
	cred, decoded := makeStakeCredential(t)
	pool := serialization.PubKeyHash{}
	pool[0] = 0x77

	// Register + delegate + withdraw in one transaction
	a, err := a.RegisterStake(cred)
	if err != nil {
		t.Fatalf("RegisterStake: %v", err)
	}
	a, err = a.DelegateStake(cred, pool)
	if err != nil {
		t.Fatalf("DelegateStake: %v", err)
	}
	a = a.AddWithdrawal(
		decoded,
		0,
		PlutusData.PlutusData{},
	)

	built, _, err := a.Complete()
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}

	tx := built.GetTx()
	certs := *tx.TransactionBody.Certificates
	if len(certs) != 2 {
		t.Fatalf("expected 2 certs, got %d", len(certs))
	}
	if tx.TransactionBody.Withdrawals == nil {
		t.Fatal("withdrawals nil")
	}
	if tx.TransactionBody.Withdrawals.Size() != 1 {
		t.Fatal("expected 1 withdrawal")
	}

	txBytes, err := tx.Bytes()
	if err != nil {
		t.Fatalf("tx Bytes: %v", err)
	}
	if len(txBytes) == 0 {
		t.Fatal("serialized tx is empty")
	}
}

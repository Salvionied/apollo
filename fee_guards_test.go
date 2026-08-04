package apollo

import (
	"math/big"
	"strings"
	"testing"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/babbage"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/mary"
	"github.com/blinklabs-io/gouroboros/ledger/shelley"
)

// TestOutputValueSizeIsEnforced is the regression guard for max_val_size.
//
// The ledger bounds the value portion of an output, not the whole output, so a
// wallet holding many native assets could build a change output the node
// refuses with OutputTooBigUTxO while the transaction stayed within
// max_tx_size. Apollo parsed max_val_size in every backend and enforced it
// nowhere.
func TestOutputValueSizeIsEnforced(t *testing.T) {
	cc := setupFixedContext()
	addr := testAddress(t)

	// A UTxO carrying enough distinct assets that the resulting change value
	// cannot fit inside max_val_size (5000 bytes on the fixed context's
	// parameters).
	utxo := manyAssetUtxo(t, addr, 400)
	cc.AddUtxo(addr, utxo)

	p, err := NewPayment(validTestAddrBech32, 2_000_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = New(cc).
		SetWallet(NewExternalWallet(addr)).
		AddInput(utxo).
		AddPayment(p).
		SetTtl(50_000_000).
		Complete()
	if err == nil {
		t.Fatal("expected an oversized output value to be rejected")
	}
	if !strings.Contains(err.Error(), "max_val_size") {
		t.Errorf("error does not mention max_val_size: %v", err)
	}
}

// TestOrdinaryOutputValuePasses confirms the new check does not reject normal
// transactions.
func TestOrdinaryOutputValuePasses(t *testing.T) {
	if _, err := buildTransfer(t, 10_000_000, 2_000_000); err != nil {
		t.Fatalf("a plain transfer was rejected: %v", err)
	}
}

// TestWitnessCountCoversDistinctInputCredentials is the regression guard for the
// fee undercount. Estimating one witness plus the registered required signers
// ignores inputs spanning several payment credentials, and each missing witness
// is around 102 bytes against a fee with no slack.
func TestWitnessCountCoversDistinctInputCredentials(t *testing.T) {
	cc := setupFixedContext()
	first := testAddress(t)
	second := testAddressVariant(t, 0xcc)

	addTestUtxo(cc, first, 5_000_000, 0x01, 0)
	addTestUtxo(cc, second, 5_000_000, 0x02, 0)

	firstUtxos, err := cc.Utxos(first)
	if err != nil {
		t.Fatal(err)
	}
	secondUtxos, err := cc.Utxos(second)
	if err != nil {
		t.Fatal(err)
	}

	a := New(cc).SetWallet(NewExternalWallet(first)).SetTtl(50_000_000)
	for _, u := range append(firstUtxos, secondUtxos...) {
		a = a.AddInput(u)
	}
	p, err := NewPayment(validTestAddrBech32, 2_000_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	a, err = a.AddPayment(p).Complete()
	if err != nil {
		t.Fatal(err)
	}

	// Two distinct payment credentials are spent, so two signatures are needed.
	inputs := append(append([]common.Utxo{}, firstUtxos...), secondUtxos...)
	if got := a.estimatedWitnessCount(inputs); got < 2 {
		t.Errorf(
			"estimatedWitnessCount = %d for inputs under 2 distinct payment "+
				"credentials, want at least 2",
			got,
		)
	}

	// And the fee must cover a transaction carrying that many witnesses.
	assertFeeCoversWitnesses(t, a, 2)
}

// TestWitnessCountCoversWithdrawalStakeKey covers the other undercounted case: a
// withdrawal is authorised by the stake credential of its reward address, which
// needs its own signature.
func TestWitnessCountCoversWithdrawalStakeKey(t *testing.T) {
	cc := setupFixedContext()
	addr := testAddress(t)
	addTestUtxo(cc, addr, 10_000_000, 0x01, 0)

	a, err := New(cc).
		SetWallet(NewExternalWallet(addr)).
		SetTtl(50_000_000).
		AddWithdrawal(stakeAddressFor(t, addr), 1_000_000, nil, nil).
		Complete()
	if err != nil {
		t.Fatal(err)
	}
	// The payment key and the stake key are distinct credentials.
	if got := a.estimatedWitnessCount(nil); got < 2 {
		t.Errorf(
			"estimatedWitnessCount = %d with a withdrawal, want at least 2",
			got,
		)
	}
	assertFeeCoversWitnesses(t, a, 2)
}

// TestWitnessCountCoversCertificateStakeKey guards stake certificates, whose
// key credentials also require a vkey witness even without a withdrawal.
func TestWitnessCountCoversCertificateStakeKey(t *testing.T) {
	cc := setupFixedContext()
	addr := testAddress(t)
	addTestUtxo(cc, addr, 10_000_000, 0x01, 0)
	stakeHash := common.Blake2b224{0x42}
	credential := common.Credential{
		CredType:   common.CredentialTypeAddrKeyHash,
		Credential: stakeHash,
	}
	a := New(cc).
		SetWallet(NewExternalWallet(addr)).
		SetCertificates([]common.CertificateWrapper{{
			Type: uint(common.CertificateTypeStakeDelegation),
			Certificate: &common.StakeDelegationCertificate{
				CertType:        uint(common.CertificateTypeStakeDelegation),
				StakeCredential: &credential,
			},
		}})
	if got := a.estimatedWitnessCount(nil); got < 2 {
		t.Errorf(
			"estimatedWitnessCount = %d with a certificate stake key, want at least 2",
			got,
		)
	}
}

// TestWitnessCountNeverBelowPrevious pins the safety property: the new estimate
// can raise a fee but never lower one.
func TestWitnessCountNeverBelowPrevious(t *testing.T) {
	cc := setupFixedContext()
	addr := testAddress(t)
	addTestUtxo(cc, addr, 10_000_000, 0x01, 0)

	a := New(cc).SetWallet(NewExternalWallet(addr))
	for _, hash := range []common.Blake2b224{{0x01}, {0x02}, {0x03}} {
		a = a.AddRequiredSigner(hash)
	}
	if got, want := a.estimatedWitnessCount(nil), 1+3; got < want {
		t.Errorf("estimatedWitnessCount = %d, want at least %d", got, want)
	}
}

// assertFeeCoversWitnesses checks the committed fee covers the transaction's own
// size once it carries witnessCount signatures.
func assertFeeCoversWitnesses(t *testing.T, a *Apollo, witnessCount int) {
	t.Helper()
	txCbor, err := a.GetTxCbor()
	if err != nil {
		t.Fatal(err)
	}
	pp, err := a.Context.ProtocolParams()
	if err != nil {
		t.Fatal(err)
	}
	// A vkey witness is a 32-byte key plus a 64-byte signature, so roughly 102
	// bytes once CBOR framing is included.
	const witnessBytes = 102
	//nolint:gosec // protocol params and sizes are small positive values
	size := int64(len(txCbor)) + int64(witnessCount*witnessBytes)
	minFee := size*pp.MinFeeCoefficient + pp.MinFeeConstant
	fee := new(big.Int).SetUint64(a.GetTx().Body.TxFee)
	if fee.Cmp(big.NewInt(minFee)) < 0 {
		t.Errorf(
			"fee %s does not cover %d bytes with %d witnesses (need %d)",
			fee, size, witnessCount, minFee,
		)
	}
}

// TestResolveCredentialRejectsCorruptedAddress guards the fifth bech32 entry
// point. resolveCredential backs the string form of RegisterStake,
// DelegateStake and DelegateVote, and called common.NewAddress directly, which
// falls back to base58 on a checksum failure.
func TestResolveCredentialRejectsCorruptedAddress(t *testing.T) {
	// A mainnet address specifically. Shelley mainnet addresses are composed
	// entirely of base58-legal characters, so common.NewAddress re-decodes a
	// corrupted one through its base58 fallback instead of reporting the
	// checksum failure. Testnet addresses are incidentally immune because "_"
	// is not in the base58 alphabet, so a testnet address cannot exercise this.
	payment := make([]byte, common.AddressHashSize)
	stake := make([]byte, common.AddressHashSize)
	for i := range payment {
		payment[i] = 0xaa
		stake[i] = 0xbb
	}
	addr, err := common.NewAddressFromParts(
		common.AddressTypeKeyKey, 1, payment, stake,
	)
	if err != nil {
		t.Fatal(err)
	}
	valid := addr.String()
	if !strings.HasPrefix(valid, "addr1") {
		t.Fatalf("expected a mainnet address, got %s", valid)
	}

	a := New(setupFixedContext())
	if _, err := a.resolveCredential(valid); err != nil {
		t.Fatalf("a valid address was rejected: %v", err)
	}

	var accepted []string
	for i := 10; i < len(valid); i++ {
		for _, c := range "qpzry9x8gf2tvdw0s3jn54khce6mua7l" {
			if rune(valid[i]) == c {
				continue
			}
			corrupted := valid[:i] + string(c) + valid[i+1:]
			if _, err := a.resolveCredential(corrupted); err == nil {
				accepted = append(accepted, corrupted)
			}
		}
	}
	if len(accepted) > 0 {
		t.Errorf(
			"resolveCredential accepted %d corrupted addresses, first: %s",
			len(accepted), accepted[0],
		)
	}
}

// testAddressVariant builds an address distinct from testAddress by changing the
// payment credential, so a test can spend inputs under two credentials.
func testAddressVariant(t *testing.T, fill byte) common.Address {
	t.Helper()
	base := testAddress(t)
	payment := make([]byte, common.AddressHashSize)
	for i := range payment {
		payment[i] = fill
	}
	addr, err := common.NewAddressFromParts(
		base.Type(),
		//nolint:gosec // network id is 0 or 1
		uint8(base.NetworkId()),
		payment,
		base.StakeKeyHash().Bytes(),
	)
	if err != nil {
		t.Fatalf("build variant address: %v", err)
	}
	return addr
}

// manyAssetUtxo builds a UTxO holding count distinct native assets, each under
// its own policy, so its value serializes past max_val_size.
func manyAssetUtxo(t *testing.T, addr common.Address, count int) common.Utxo {
	t.Helper()
	policies := make(
		map[common.Blake2b224]map[cbor.ByteString]common.MultiAssetTypeOutput,
		count,
	)
	for i := range count {
		var policy common.Blake2b224
		policy[0] = byte(i % 256)
		policy[1] = byte(i / 256)
		policies[policy] = map[cbor.ByteString]common.MultiAssetTypeOutput{
			cbor.NewByteString([]byte{byte(i % 256)}): big.NewInt(1),
		}
	}
	assets := common.NewMultiAsset[common.MultiAssetTypeOutput](policies)
	var txHash common.Blake2b256
	txHash[0] = 0xfe
	output := babbage.BabbageTransactionOutput{
		OutputAddress: addr,
		OutputAmount: mary.MaryTransactionOutputValue{
			// Enough ADA that the change output clears its raised min-UTxO, so
			// the size check is what rejects it rather than insufficient funds.
			Amount: 500_000_000,
			Assets: &assets,
		},
	}
	return common.Utxo{
		Id: shelley.ShelleyTransactionInput{
			TxId:        txHash,
			OutputIndex: 0,
		},
		Output: &output,
	}
}

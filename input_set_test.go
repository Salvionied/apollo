package apollo

import (
	"testing"

	"github.com/blinklabs-io/gouroboros/ledger/common"
)

// stakeAddressFor builds a reward address controlled by the same stake
// credential as addr, suitable for AddWithdrawal.
func stakeAddressFor(t *testing.T, addr common.Address) common.Address {
	t.Helper()
	stake, err := common.NewAddressFromParts(
		common.AddressTypeNoneKey,
		uint8(addr.NetworkId()), //nolint:gosec // network id is 0 or 1
		nil,
		addr.StakeKeyHash().Bytes(),
	)
	if err != nil {
		t.Fatalf("build reward address: %v", err)
	}
	return stake
}

// TestWithdrawalTransactionSpendsAtLeastOneInput is the regression guard for a
// transaction built with an empty input set.
//
// Withdrawals are implicit inputs in Cardano's balance equation, so a reward
// claim large enough to cover the whole selection target made coin selection
// return no UTxOs at all. The resulting body had inputs = [], which the ledger
// rejects with InputSetEmptyUTxO. It also carried no payment-key witness and
// nothing establishing the transaction's uniqueness.
func TestWithdrawalTransactionSpendsAtLeastOneInput(t *testing.T) {
	for _, reward := range []uint64{
		500_000,     // below the selection reserve: selection ran before
		5_000_000,   // above it: selection short-circuited
		50_000_000,  // far above it
		500_000_000, // far above the wallet balance itself
	} {
		cc := setupFixedContext()
		addr := testAddress(t)
		addTestUtxo(cc, addr, 10_000_000, 0x01, 0)

		a, err := New(cc).
			SetWallet(NewExternalWallet(addr)).
			SetTtl(50_000_000).
			AddWithdrawal(stakeAddressFor(t, addr), reward, nil, nil).
			Complete()
		if err != nil {
			t.Errorf("reward %d: Complete failed: %v", reward, err)
			continue
		}

		inputs := a.GetTx().Body.Inputs()
		if len(inputs) == 0 {
			t.Errorf(
				"reward %d: built a transaction with an empty input set "+
					"(InputSetEmptyUTxO)",
				reward,
			)
			continue
		}
		t.Logf(
			"reward %d -> inputs=%d outputs=%d fee=%d",
			reward, len(inputs), len(a.GetTx().Body.Outputs()),
			a.GetTx().Body.TxFee,
		)
	}
}

// TestWithdrawalOnlyTransactionConservesValue checks that forcing an input into
// a withdrawal-funded transaction still balances: the spent UTxO plus the
// reward must equal the outputs plus the fee.
func TestWithdrawalOnlyTransactionConservesValue(t *testing.T) {
	const balance = 10_000_000
	const reward = 50_000_000

	cc := setupFixedContext()
	addr := testAddress(t)
	addTestUtxo(cc, addr, balance, 0x01, 0)

	a, err := New(cc).
		SetWallet(NewExternalWallet(addr)).
		SetTtl(50_000_000).
		AddWithdrawal(stakeAddressFor(t, addr), reward, nil, nil).
		Complete()
	if err != nil {
		t.Fatal(err)
	}

	tx := a.GetTx()
	inputs := tx.Body.Inputs()
	if len(inputs) != 1 {
		t.Fatalf("spent %d inputs, want 1", len(inputs))
	}

	var out uint64
	for _, o := range tx.Body.Outputs() {
		out += o.Amount().Uint64()
	}
	// One spent UTxO of `balance`, plus the withdrawal, funds outputs and fee.
	if got, want := out+tx.Body.TxFee, uint64(balance+reward); got != want {
		t.Errorf(
			"value not conserved: outputs %d + fee %d = %d, want %d",
			out, tx.Body.TxFee, got, want,
		)
	}
	if got := len(tx.Body.Withdrawals()); got != 1 {
		t.Errorf("body has %d withdrawals, want 1", got)
	}
}

// TestPreselectedInputSatisfiesTheInputSet confirms the guarantee is satisfied
// by a caller-pinned input without selection contributing another one.
func TestPreselectedInputSatisfiesTheInputSet(t *testing.T) {
	cc := setupFixedContext()
	addr := testAddress(t)
	addTestUtxo(cc, addr, 10_000_000, 0x01, 0)
	addTestUtxo(cc, addr, 10_000_000, 0x02, 0)

	utxos, err := cc.Utxos(addr)
	if err != nil {
		t.Fatal(err)
	}
	if len(utxos) < 1 {
		t.Fatal("fixture produced no UTxOs")
	}

	a, err := New(cc).
		SetWallet(NewExternalWallet(addr)).
		SetTtl(50_000_000).
		AddInput(utxos[0]).
		AddWithdrawal(stakeAddressFor(t, addr), 50_000_000, nil, nil).
		Complete()
	if err != nil {
		t.Fatal(err)
	}
	if got := len(a.GetTx().Body.Inputs()); got != 1 {
		t.Errorf("spent %d inputs, want exactly the 1 preselected", got)
	}
}

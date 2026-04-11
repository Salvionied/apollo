package apollo_test

import (
	"bytes"
	"math"
	"testing"

	"github.com/Salvionied/apollo"
	"github.com/Salvionied/apollo/serialization"
	"github.com/Salvionied/apollo/serialization/Address"
	"github.com/Salvionied/apollo/serialization/Certificate"
	"github.com/Salvionied/apollo/serialization/Governance"
	testutils "github.com/Salvionied/apollo/testUtils"
	"github.com/Salvionied/apollo/txBuilding/Backend/FixedChainContext"
)

func newGovernanceTestApollo(t *testing.T) *apollo.Apollo {
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

func TestSetCurrentTreasuryValue(t *testing.T) {
	a := newGovernanceTestApollo(t)
	a = a.SetCurrentTreasuryValue(1_000_000_000)

	built, _, err := a.Complete()
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	tx := built.GetTx()
	if tx.TransactionBody.CurrentTreasuryValue !=
		1_000_000_000 {
		t.Fatalf(
			"expected treasury value 1000000000, got %d",
			tx.TransactionBody.CurrentTreasuryValue,
		)
	}
}

func TestAddTreasuryDonation(t *testing.T) {
	a := newGovernanceTestApollo(t)
	a = a.
		AddTreasuryDonation(5_000_000).
		AddTreasuryDonation(2_500_000)

	built, _, err := a.Complete()
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	tx := built.GetTx()
	if tx.TransactionBody.Donation != 7_500_000 {
		t.Fatalf(
			"expected donation 7500000, got %d",
			tx.TransactionBody.Donation,
		)
	}
}

func TestTreasuryFieldsNotSetByDefault(t *testing.T) {
	a := newGovernanceTestApollo(t)

	built, _, err := a.Complete()
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	tx := built.GetTx()
	if tx.TransactionBody.CurrentTreasuryValue != 0 {
		t.Fatal(
			"CurrentTreasuryValue should be 0 by default",
		)
	}
	if tx.TransactionBody.Donation != 0 {
		t.Fatal("Donation should be 0 by default")
	}
}

func TestSetCurrentTreasuryValueNegative(t *testing.T) {
	a := newGovernanceTestApollo(t)
	a = a.SetCurrentTreasuryValue(-1)
	_, _, err := a.Complete()
	if err == nil {
		t.Fatal("expected error for negative treasury value")
	}
}

func TestAddTreasuryDonationNegative(t *testing.T) {
	a := newGovernanceTestApollo(t)
	a = a.AddTreasuryDonation(-1)
	_, _, err := a.Complete()
	if err == nil {
		t.Fatal("expected error for negative donation amount")
	}
}

func TestAddTreasuryDonationOverflow(t *testing.T) {
	a := newGovernanceTestApollo(t)
	a = a.AddTreasuryDonation(math.MaxInt64)
	a = a.AddTreasuryDonation(1)
	_, _, err := a.Complete()
	if err == nil {
		t.Fatal("expected error for donation amount overflow")
	}
}

func TestAddVote(t *testing.T) {
	a := newGovernanceTestApollo(t)
	voter := Governance.Voter{
		Role: Governance.DRepKeyHash,
		Hash: serialization.ConstrainedBytes{
			Payload: make([]byte, 28),
		},
	}
	actionId := Governance.GovActionId{
		TransactionHash: make([]byte, 32),
		GovActionIndex:  0,
	}
	procedure := Governance.VotingProcedure{
		Vote:   Governance.VoteYes,
		Anchor: nil,
	}
	a = a.AddVote(voter, actionId, procedure)

	built, _, err := a.Complete()
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	tx := built.GetTx()
	if tx.TransactionBody.VotingProcedures == nil {
		t.Fatal("VotingProcedures should not be nil")
	}
	vp := *tx.TransactionBody.VotingProcedures
	if len(vp) != 1 {
		t.Fatalf("expected 1 voter, got %d", len(vp))
	}
	if len(vp[0].Votes) != 1 {
		t.Fatalf(
			"expected 1 vote, got %d",
			len(vp[0].Votes),
		)
	}
	if vp[0].Votes[0].Procedure.Vote !=
		Governance.VoteYes {
		t.Fatal("expected VoteYes")
	}
}

func TestAddMultipleVotes(t *testing.T) {
	a := newGovernanceTestApollo(t)
	voter := Governance.Voter{
		Role: Governance.DRepKeyHash,
		Hash: serialization.ConstrainedBytes{
			Payload: make([]byte, 28),
		},
	}
	action1 := Governance.GovActionId{
		TransactionHash: make([]byte, 32),
		GovActionIndex:  0,
	}
	action2 := Governance.GovActionId{
		TransactionHash: make([]byte, 32),
		GovActionIndex:  1,
	}
	a = a.
		AddVote(
			voter,
			action1,
			Governance.VotingProcedure{
				Vote: Governance.VoteYes,
			},
		).
		AddVote(
			voter,
			action2,
			Governance.VotingProcedure{
				Vote: Governance.VoteNo,
			},
		)

	built, _, err := a.Complete()
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	tx := built.GetTx()
	vp := *tx.TransactionBody.VotingProcedures
	if len(vp) != 1 {
		t.Fatalf(
			"expected 1 voter (grouped), got %d",
			len(vp),
		)
	}
	if len(vp[0].Votes) != 2 {
		t.Fatalf(
			"expected 2 votes, got %d",
			len(vp[0].Votes),
		)
	}
}

func TestAddProposal(t *testing.T) {
	a := newGovernanceTestApollo(t)
	proposal := Governance.ProposalProcedure{
		Deposit:       2_000_000,
		RewardAccount: make([]byte, 29),
		Action:        Governance.InfoAction{},
		Anchor: Certificate.Anchor{
			Url:      "https://example.com/proposal",
			DataHash: make([]byte, 32),
		},
	}
	a = a.AddProposal(proposal)

	built, _, err := a.Complete()
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	tx := built.GetTx()
	if tx.TransactionBody.ProposalProcedures == nil {
		t.Fatal(
			"ProposalProcedures should not be nil",
		)
	}
	pp := *tx.TransactionBody.ProposalProcedures
	if len(pp) != 1 {
		t.Fatalf(
			"expected 1 proposal, got %d",
			len(pp),
		)
	}
	if pp[0].Deposit != 2_000_000 {
		t.Fatalf(
			"expected deposit 2000000, got %d",
			pp[0].Deposit,
		)
	}
}

func TestAddMultipleProposals(t *testing.T) {
	a := newGovernanceTestApollo(t)
	a = a.
		AddProposal(Governance.ProposalProcedure{
			Deposit:       2_000_000,
			RewardAccount: make([]byte, 29),
			Action:        Governance.InfoAction{},
			Anchor: Certificate.Anchor{
				Url:      "https://example.com/proposal-1",
				DataHash: make([]byte, 32),
			},
		}).
		AddProposal(Governance.ProposalProcedure{
			Deposit:       3_000_000,
			RewardAccount: append(make([]byte, 28), 0x01),
			Action: Governance.NoConfidence{
				PrevActionId: nil,
			},
			Anchor: Certificate.Anchor{
				Url:      "https://example.com/proposal-2",
				DataHash: bytes.Repeat([]byte{0x02}, 32),
			},
		})

	built, _, err := a.Complete()
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	tx := built.GetTx()
	if tx.TransactionBody.ProposalProcedures == nil {
		t.Fatal("ProposalProcedures should not be nil")
	}
	pp := *tx.TransactionBody.ProposalProcedures
	if len(pp) != 2 {
		t.Fatalf("expected 2 proposals, got %d", len(pp))
	}
	if pp[0].Deposit != 2_000_000 {
		t.Fatalf(
			"expected first deposit 2000000, got %d",
			pp[0].Deposit,
		)
	}
	if pp[1].Deposit != 3_000_000 {
		t.Fatalf(
			"expected second deposit 3000000, got %d",
			pp[1].Deposit,
		)
	}
}

func TestVotingAndProposalFieldsNilByDefault(
	t *testing.T,
) {
	a := newGovernanceTestApollo(t)

	built, _, err := a.Complete()
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	tx := built.GetTx()
	if tx.TransactionBody.VotingProcedures != nil {
		t.Fatal(
			"VotingProcedures should be nil by default",
		)
	}
	if tx.TransactionBody.ProposalProcedures != nil {
		t.Fatal(
			"ProposalProcedures should be nil by default",
		)
	}
}

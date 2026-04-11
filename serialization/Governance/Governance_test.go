package Governance

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"testing"

	"github.com/Salvionied/apollo/serialization"
	"github.com/Salvionied/apollo/serialization/Certificate"
	"github.com/fxamacker/cbor/v2"
)

// ----------------------------------------------------------------
// helpers
// ----------------------------------------------------------------

// cborRoundTrip verifies that data is valid CBOR by trying a
// standard decode. For data with complex map keys (e.g.,
// arrays), the standard decode may fail -- that is acceptable
// since we verify semantic round-trip in individual tests.
func cborRoundTrip(t *testing.T, data []byte) {
	t.Helper()
	var v1 any
	if err := cbor.Unmarshal(data, &v1); err != nil {
		// Complex map keys; skip generic round-trip check.
		return
	}
	bz2, err := cbor.Marshal(v1)
	if err != nil {
		t.Fatalf("re-marshal: %v", err)
	}
	var v2 any
	if err := cbor.Unmarshal(bz2, &v2); err != nil {
		t.Fatalf("unmarshal re-encoded: %v", err)
	}
	if !reflect.DeepEqual(v1, v2) {
		t.Fatalf(
			"round-trip mismatch:\n  v1=%#v\n  v2=%#v",
			v1,
			v2,
		)
	}
}

func testHash(b ...byte) serialization.ConstrainedBytes {
	return serialization.ConstrainedBytes{Payload: b}
}

func sampleAnchor() Certificate.Anchor {
	return Certificate.Anchor{
		Url:      "https://example.com/meta",
		DataHash: []byte{0xAA, 0xBB, 0xCC},
	}
}

func sampleGovActionId() GovActionId {
	return GovActionId{
		TransactionHash: bytes.Repeat([]byte{0x01}, 32),
		GovActionIndex:  7,
	}
}

// ----------------------------------------------------------------
// TestVoterRoundTrip
// ----------------------------------------------------------------

func TestVoterRoundTrip(t *testing.T) {
	roles := []VoterRole{
		ConstitutionalCommitteeKeyHash,
		ConstitutionalCommitteeScript,
		DRepKeyHash,
		DRepScript,
		StakePoolOperator,
	}
	for _, role := range roles {
		v := Voter{
			Role: role,
			Hash: testHash(0xDE, 0xAD),
		}
		bz, err := cbor.Marshal(v)
		if err != nil {
			t.Fatalf("marshal voter role %d: %v", role, err)
		}
		cborRoundTrip(t, bz)

		var got Voter
		if err := cbor.Unmarshal(bz, &got); err != nil {
			t.Fatalf(
				"unmarshal voter role %d: %v",
				role,
				err,
			)
		}
		if got.Role != v.Role {
			t.Fatalf(
				"role mismatch: %d vs %d",
				got.Role,
				v.Role,
			)
		}
		if !bytes.Equal(
			got.Hash.Payload,
			v.Hash.Payload,
		) {
			t.Fatalf("hash mismatch")
		}
	}
}

// ----------------------------------------------------------------
// TestGovActionIdRoundTrip
// ----------------------------------------------------------------

func TestGovActionIdRoundTrip(t *testing.T) {
	aid := sampleGovActionId()
	bz, err := cbor.Marshal(aid)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	cborRoundTrip(t, bz)

	var got GovActionId
	if err := cbor.Unmarshal(bz, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !bytes.Equal(
		got.TransactionHash,
		aid.TransactionHash,
	) {
		t.Fatalf("tx hash mismatch")
	}
	if got.GovActionIndex != aid.GovActionIndex {
		t.Fatalf("index mismatch")
	}
}

// ----------------------------------------------------------------
// TestVotingProcedureRoundTrip
// ----------------------------------------------------------------

func TestVotingProcedureRoundTrip(t *testing.T) {
	// Without anchor
	proc := VotingProcedure{Vote: VoteYes, Anchor: nil}
	bz, err := cbor.Marshal(proc)
	if err != nil {
		t.Fatalf("marshal no anchor: %v", err)
	}
	cborRoundTrip(t, bz)

	var got VotingProcedure
	if err := cbor.Unmarshal(bz, &got); err != nil {
		t.Fatalf("unmarshal no anchor: %v", err)
	}
	if got.Vote != VoteYes || got.Anchor != nil {
		t.Fatalf(
			"unexpected: vote=%d anchor=%v",
			got.Vote,
			got.Anchor,
		)
	}

	// With anchor
	anch := sampleAnchor()
	proc2 := VotingProcedure{Vote: VoteAbstain, Anchor: &anch}
	bz2, err := cbor.Marshal(proc2)
	if err != nil {
		t.Fatalf("marshal with anchor: %v", err)
	}
	cborRoundTrip(t, bz2)

	var got2 VotingProcedure
	if err := cbor.Unmarshal(bz2, &got2); err != nil {
		t.Fatalf("unmarshal with anchor: %v", err)
	}
	if got2.Vote != VoteAbstain {
		t.Fatalf("vote mismatch")
	}
	if got2.Anchor == nil {
		t.Fatalf("expected anchor")
	}
	if got2.Anchor.Url != anch.Url {
		t.Fatalf("anchor url mismatch")
	}
}

// ----------------------------------------------------------------
// TestVotingProceduresRoundTrip
// ----------------------------------------------------------------

func TestVotingProceduresRoundTrip(t *testing.T) {
	voter1 := Voter{
		Role: DRepKeyHash,
		Hash: testHash(0x01, 0x02),
	}
	voter2 := Voter{
		Role: StakePoolOperator,
		Hash: testHash(0x03, 0x04),
	}
	aid1 := GovActionId{
		TransactionHash: bytes.Repeat([]byte{0xAA}, 32),
		GovActionIndex:  0,
	}
	aid2 := GovActionId{
		TransactionHash: bytes.Repeat([]byte{0xBB}, 32),
		GovActionIndex:  1,
	}
	anch := sampleAnchor()

	vp := VotingProcedures{
		{
			Voter: voter1,
			Votes: []ActionVote{
				{
					ActionId: aid1,
					Procedure: VotingProcedure{
						Vote:   VoteYes,
						Anchor: nil,
					},
				},
				{
					ActionId: aid2,
					Procedure: VotingProcedure{
						Vote:   VoteNo,
						Anchor: &anch,
					},
				},
			},
		},
		{
			Voter: voter2,
			Votes: []ActionVote{
				{
					ActionId: aid1,
					Procedure: VotingProcedure{
						Vote:   VoteAbstain,
						Anchor: nil,
					},
				},
			},
		},
	}

	bz, err := vp.MarshalCBOR()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got VotingProcedures
	if err := got.UnmarshalCBOR(bz); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Re-marshal and compare bytes
	bz2, err := got.MarshalCBOR()
	if err != nil {
		t.Fatalf("re-marshal: %v", err)
	}
	if !bytes.Equal(bz, bz2) {
		t.Fatalf(
			"round-trip bytes mismatch:\n  orig=%x\n  rt  =%x",
			bz,
			bz2,
		)
	}
}

// ----------------------------------------------------------------
// TestVotingProceduresAdd
// ----------------------------------------------------------------

func TestVotingProceduresAdd(t *testing.T) {
	var vp VotingProcedures

	voter := Voter{
		Role: DRepKeyHash,
		Hash: testHash(0x01),
	}
	aid1 := sampleGovActionId()
	aid2 := GovActionId{
		TransactionHash: bytes.Repeat([]byte{0x02}, 32),
		GovActionIndex:  3,
	}

	vp.Add(
		voter,
		aid1,
		VotingProcedure{Vote: VoteYes, Anchor: nil},
	)
	if len(vp) != 1 {
		t.Fatalf("expected 1 voter entry, got %d", len(vp))
	}
	if len(vp[0].Votes) != 1 {
		t.Fatalf(
			"expected 1 vote, got %d",
			len(vp[0].Votes),
		)
	}

	// Add another vote for the same voter
	vp.Add(
		voter,
		aid2,
		VotingProcedure{Vote: VoteNo, Anchor: nil},
	)
	if len(vp) != 1 {
		t.Fatalf(
			"expected 1 voter entry after second add, got %d",
			len(vp),
		)
	}
	if len(vp[0].Votes) != 2 {
		t.Fatalf(
			"expected 2 votes, got %d",
			len(vp[0].Votes),
		)
	}

	// Add a different voter
	voter2 := Voter{
		Role: StakePoolOperator,
		Hash: testHash(0x99),
	}
	vp.Add(
		voter2,
		aid1,
		VotingProcedure{Vote: VoteAbstain, Anchor: nil},
	)
	if len(vp) != 2 {
		t.Fatalf("expected 2 voter entries, got %d", len(vp))
	}
}

func TestVotingProceduresAddReplacesDuplicateAction(
	t *testing.T,
) {
	var vp VotingProcedures

	voter := Voter{
		Role: DRepKeyHash,
		Hash: testHash(0x01),
	}
	action := sampleGovActionId()

	vp.Add(
		voter,
		action,
		VotingProcedure{Vote: VoteYes},
	)
	vp.Add(
		voter,
		action,
		VotingProcedure{Vote: VoteNo},
	)

	if len(vp) != 1 {
		t.Fatalf("expected 1 voter entry, got %d", len(vp))
	}
	if len(vp[0].Votes) != 1 {
		t.Fatalf(
			"expected 1 vote after replacement, got %d",
			len(vp[0].Votes),
		)
	}
	if vp[0].Votes[0].Procedure.Vote != VoteNo {
		t.Fatalf(
			"expected replacement vote %d, got %d",
			VoteNo,
			vp[0].Votes[0].Procedure.Vote,
		)
	}
}

func TestVotingProceduresMarshalCBORCanonicalOrder(
	t *testing.T,
) {
	voter1 := Voter{
		Role: DRepKeyHash,
		Hash: testHash(0x01, 0x02),
	}
	voter2 := Voter{
		Role: StakePoolOperator,
		Hash: testHash(0x03, 0x04),
	}
	action1 := sampleGovActionId()
	action2 := GovActionId{
		TransactionHash: bytes.Repeat([]byte{0x02}, 32),
		GovActionIndex:  1,
	}

	ordered := VotingProcedures{
		{
			Voter: voter1,
			Votes: []ActionVote{
				{
					ActionId:  action1,
					Procedure: VotingProcedure{Vote: VoteYes},
				},
				{
					ActionId:  action2,
					Procedure: VotingProcedure{Vote: VoteNo},
				},
			},
		},
		{
			Voter: voter2,
			Votes: []ActionVote{
				{
					ActionId:  action1,
					Procedure: VotingProcedure{Vote: VoteAbstain},
				},
			},
		},
	}
	reversed := VotingProcedures{
		{
			Voter: voter2,
			Votes: []ActionVote{
				{
					ActionId:  action1,
					Procedure: VotingProcedure{Vote: VoteAbstain},
				},
			},
		},
		{
			Voter: voter1,
			Votes: []ActionVote{
				{
					ActionId:  action2,
					Procedure: VotingProcedure{Vote: VoteNo},
				},
				{
					ActionId:  action1,
					Procedure: VotingProcedure{Vote: VoteYes},
				},
			},
		},
	}

	orderedBz, err := ordered.MarshalCBOR()
	if err != nil {
		t.Fatalf("marshal ordered: %v", err)
	}
	reversedBz, err := reversed.MarshalCBOR()
	if err != nil {
		t.Fatalf("marshal reversed: %v", err)
	}
	if !bytes.Equal(orderedBz, reversedBz) {
		t.Fatalf("expected canonical map ordering")
	}
}

// ----------------------------------------------------------------
// TestInfoActionRoundTrip
// ----------------------------------------------------------------

func TestInfoActionRoundTrip(t *testing.T) {
	action := InfoAction{}
	bz, err := action.MarshalCBOR()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	cborRoundTrip(t, bz)

	got, err := UnmarshalGovAction(bz)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.GovActionType() != 6 {
		t.Fatalf("type mismatch: %d", got.GovActionType())
	}
	if _, ok := got.(InfoAction); !ok {
		t.Fatalf("expected InfoAction, got %T", got)
	}
}

// ----------------------------------------------------------------
// TestNoConfidenceRoundTrip
// ----------------------------------------------------------------

func TestNoConfidenceRoundTrip(t *testing.T) {
	// Without prev
	nc := NoConfidence{PrevActionId: nil}
	bz, err := nc.MarshalCBOR()
	if err != nil {
		t.Fatalf("marshal nil prev: %v", err)
	}
	cborRoundTrip(t, bz)
	got, err := UnmarshalGovAction(bz)
	if err != nil {
		t.Fatalf("unmarshal nil prev: %v", err)
	}
	ncGot := got.(NoConfidence)
	if ncGot.PrevActionId != nil {
		t.Fatalf("expected nil prev")
	}

	// With prev
	aid := sampleGovActionId()
	nc2 := NoConfidence{PrevActionId: &aid}
	bz2, err := nc2.MarshalCBOR()
	if err != nil {
		t.Fatalf("marshal with prev: %v", err)
	}
	cborRoundTrip(t, bz2)
	got2, err := UnmarshalGovAction(bz2)
	if err != nil {
		t.Fatalf("unmarshal with prev: %v", err)
	}
	ncGot2 := got2.(NoConfidence)
	if ncGot2.PrevActionId == nil {
		t.Fatalf("expected non-nil prev")
	}
	if ncGot2.PrevActionId.GovActionIndex != 7 {
		t.Fatalf("prev action index mismatch")
	}
}

// ----------------------------------------------------------------
// TestHardForkInitiationRoundTrip
// ----------------------------------------------------------------

func TestHardForkInitiationRoundTrip(t *testing.T) {
	aid := sampleGovActionId()
	hf := HardForkInitiation{
		PrevActionId: &aid,
		ProtocolVersion: ProtocolVersion{
			Major: 10,
			Minor: 0,
		},
	}
	bz, err := hf.MarshalCBOR()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	cborRoundTrip(t, bz)

	got, err := UnmarshalGovAction(bz)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	hfGot := got.(HardForkInitiation)
	if hfGot.ProtocolVersion.Major != 10 {
		t.Fatalf("major mismatch")
	}
	if hfGot.ProtocolVersion.Minor != 0 {
		t.Fatalf("minor mismatch")
	}
}

// ----------------------------------------------------------------
// TestTreasuryWithdrawalsRoundTrip
// ----------------------------------------------------------------

func TestTreasuryWithdrawalsRoundTrip(t *testing.T) {
	tw := TreasuryWithdrawals{
		Withdrawals: []Withdrawal{
			{
				RewardAccount: []byte{0xE0, 0x01, 0x02},
				Coin:          5000000,
			},
		},
		PrevActionId: nil,
	}
	bz, err := tw.MarshalCBOR()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	cborRoundTrip(t, bz)

	got, err := UnmarshalGovAction(bz)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	twGot := got.(TreasuryWithdrawals)
	if len(twGot.Withdrawals) != 1 {
		t.Fatalf(
			"expected 1 withdrawal, got %d",
			len(twGot.Withdrawals),
		)
	}
	if twGot.Withdrawals[0].Coin != 5000000 {
		t.Fatalf("coin mismatch")
	}
}

// ----------------------------------------------------------------
// TestParameterChangeRoundTrip
// ----------------------------------------------------------------

func TestParameterChangeRoundTrip(t *testing.T) {
	aid := sampleGovActionId()
	pc := ParameterChange{
		PrevActionId: &aid,
		ParamUpdate: map[int]any{
			0:  500000,
			1:  1000000,
			18: []any{uint64(1), uint64(2)},
		},
	}
	bz, err := pc.MarshalCBOR()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	cborRoundTrip(t, bz)

	got, err := UnmarshalGovAction(bz)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	pcGot := got.(ParameterChange)
	if pcGot.PrevActionId == nil {
		t.Fatalf("expected prev action id")
	}
	if len(pcGot.ParamUpdate) == 0 {
		t.Fatalf("expected params")
	}
}

// ----------------------------------------------------------------
// TestUpdateCommitteeRoundTrip
// ----------------------------------------------------------------

func TestUpdateCommitteeRoundTrip(t *testing.T) {
	aid := sampleGovActionId()
	removed := []Certificate.Credential{
		{
			Code: 0,
			Hash: testHash(0xAA, 0xBB),
		},
	}
	added := []AddedCommitteeMember{
		{
			Credential: Certificate.Credential{
				Code: 1,
				Hash: testHash(0xCC, 0xDD),
			},
			Epoch: 300,
		},
	}
	uc := UpdateCommittee{
		PrevActionId: &aid,
		Removed:      removed,
		Added:        added,
		Quorum: Certificate.UnitInterval{
			Num: 2,
			Den: 3,
		},
	}
	bz, err := uc.MarshalCBOR()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	cborRoundTrip(t, bz)

	got, err := UnmarshalGovAction(bz)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	ucGot := got.(UpdateCommittee)
	if len(ucGot.Removed) != 1 {
		t.Fatalf(
			"expected 1 removed, got %d",
			len(ucGot.Removed),
		)
	}
	if len(ucGot.Added) != 1 {
		t.Fatalf(
			"expected 1 added, got %d",
			len(ucGot.Added),
		)
	}
	if ucGot.Quorum.Num != 2 || ucGot.Quorum.Den != 3 {
		t.Fatalf("quorum mismatch")
	}
}

// ----------------------------------------------------------------
// TestNewConstitutionRoundTrip
// ----------------------------------------------------------------

func TestNewConstitutionRoundTrip(t *testing.T) {
	// With script hash
	nc := NewConstitution{
		PrevActionId: nil,
		Anchor:       sampleAnchor(),
		ScriptHash:   bytes.Repeat([]byte{0xFF}, 28),
	}
	bz, err := nc.MarshalCBOR()
	if err != nil {
		t.Fatalf("marshal with script: %v", err)
	}
	cborRoundTrip(t, bz)
	got, err := UnmarshalGovAction(bz)
	if err != nil {
		t.Fatalf("unmarshal with script: %v", err)
	}
	ncGot := got.(NewConstitution)
	if ncGot.ScriptHash == nil {
		t.Fatalf("expected script hash")
	}

	// Without script hash (null)
	nc2 := NewConstitution{
		PrevActionId: nil,
		Anchor:       sampleAnchor(),
		ScriptHash:   nil,
	}
	bz2, err := nc2.MarshalCBOR()
	if err != nil {
		t.Fatalf("marshal null script: %v", err)
	}
	cborRoundTrip(t, bz2)
	got2, err := UnmarshalGovAction(bz2)
	if err != nil {
		t.Fatalf("unmarshal null script: %v", err)
	}
	ncGot2 := got2.(NewConstitution)
	if ncGot2.ScriptHash != nil {
		t.Fatalf("expected nil script hash")
	}
}

// ----------------------------------------------------------------
// TestProposalProcedureRoundTrip
// ----------------------------------------------------------------

func TestProposalProcedureRoundTrip(t *testing.T) {
	actions := []GovAction{
		InfoAction{},
		NoConfidence{PrevActionId: nil},
		HardForkInitiation{
			PrevActionId: nil,
			ProtocolVersion: ProtocolVersion{
				Major: 9,
				Minor: 1,
			},
		},
		TreasuryWithdrawals{
			Withdrawals: []Withdrawal{
				{
					RewardAccount: []byte{0xE0, 0x01},
					Coin:          100,
				},
			},
			PrevActionId: nil,
		},
		ParameterChange{
			PrevActionId: nil,
			ParamUpdate:  map[int]any{0: 42},
		},
		NewConstitution{
			PrevActionId: nil,
			Anchor:       sampleAnchor(),
			ScriptHash:   nil,
		},
	}

	for _, action := range actions {
		pp := ProposalProcedure{
			Deposit:       2000000,
			RewardAccount: []byte{0xE1, 0x02, 0x03},
			Action:        action,
			Anchor:        sampleAnchor(),
		}
		bz, err := pp.MarshalCBOR()
		if err != nil {
			t.Fatalf(
				"marshal type %d: %v",
				action.GovActionType(),
				err,
			)
		}
		cborRoundTrip(t, bz)

		got, err := UnmarshalProposalProcedure(bz)
		if err != nil {
			t.Fatalf(
				"unmarshal type %d: %v",
				action.GovActionType(),
				err,
			)
		}
		if got.Action.GovActionType() != action.GovActionType() {
			t.Fatalf(
				"action type mismatch: %d vs %d",
				got.Action.GovActionType(),
				action.GovActionType(),
			)
		}
		if got.Deposit != 2000000 {
			t.Fatalf("deposit mismatch")
		}

		// Verify the re-marshal produces identical CBOR
		bz2, err := got.MarshalCBOR()
		if err != nil {
			t.Fatalf(
				"re-marshal type %d: %v",
				action.GovActionType(),
				err,
			)
		}
		if !bytes.Equal(bz, bz2) {
			t.Fatalf(
				"round-trip bytes mismatch type %d",
				action.GovActionType(),
			)
		}
	}
}

// ----------------------------------------------------------------
// TestProposalProceduresRoundTrip
// ----------------------------------------------------------------

func TestProposalProceduresRoundTrip(t *testing.T) {
	ps := ProposalProcedures{
		{
			Deposit:       1000000,
			RewardAccount: []byte{0xE0, 0x01},
			Action:        InfoAction{},
			Anchor:        sampleAnchor(),
		},
		{
			Deposit:       2000000,
			RewardAccount: []byte{0xE0, 0x02},
			Action: NoConfidence{
				PrevActionId: nil,
			},
			Anchor: sampleAnchor(),
		},
	}

	bz, err := ps.MarshalCBOR()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	cborRoundTrip(t, bz)

	var got ProposalProcedures
	if err := got.UnmarshalCBOR(bz); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 proposals, got %d", len(got))
	}

	bz2, err := got.MarshalCBOR()
	if err != nil {
		t.Fatalf("re-marshal: %v", err)
	}
	if !bytes.Equal(bz, bz2) {
		t.Fatalf(
			"round-trip bytes mismatch:\n  orig=%x\n  rt  =%x",
			bz,
			bz2,
		)
	}
}

// ----------------------------------------------------------------
// TestUnmarshalGovAction
// ----------------------------------------------------------------

func TestUnmarshalGovAction(t *testing.T) {
	aid := sampleGovActionId()
	cases := []struct {
		name   string
		action GovAction
		kind   int
	}{
		{"InfoAction", InfoAction{}, 6},
		{
			"NoConfidence_nil",
			NoConfidence{PrevActionId: nil},
			3,
		},
		{
			"NoConfidence_prev",
			NoConfidence{PrevActionId: &aid},
			3,
		},
		{
			"HardForkInitiation",
			HardForkInitiation{
				PrevActionId: nil,
				ProtocolVersion: ProtocolVersion{
					Major: 9,
					Minor: 0,
				},
			},
			1,
		},
		{
			"TreasuryWithdrawals",
			TreasuryWithdrawals{
				Withdrawals: []Withdrawal{
					{
						RewardAccount: []byte{0xE0},
						Coin:          100,
					},
				},
				PrevActionId: nil,
			},
			2,
		},
		{
			"ParameterChange",
			ParameterChange{
				PrevActionId: nil,
				ParamUpdate:  map[int]any{0: 1},
			},
			0,
		},
		{
			"UpdateCommittee",
			UpdateCommittee{
				PrevActionId: nil,
				Removed: []Certificate.Credential{
					{Code: 0, Hash: testHash(0x01)},
				},
				Added: []AddedCommitteeMember{
					{
						Credential: Certificate.Credential{
							Code: 0,
							Hash: testHash(0x02),
						},
						Epoch: 100,
					},
				},
				Quorum: Certificate.UnitInterval{
					Num: 1,
					Den: 2,
				},
			},
			4,
		},
		{
			"NewConstitution",
			NewConstitution{
				PrevActionId: nil,
				Anchor:       sampleAnchor(),
				ScriptHash:   nil,
			},
			5,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bz, err := tc.action.MarshalCBOR()
			if err != nil {
				t.Fatalf("marshal %s: %v", tc.name, err)
			}
			got, err := UnmarshalGovAction(bz)
			if err != nil {
				t.Fatalf("unmarshal %s: %v", tc.name, err)
			}
			if got.GovActionType() != tc.kind {
				t.Fatalf(
					"%s: kind %d, want %d",
					tc.name,
					got.GovActionType(),
					tc.kind,
				)
			}

			// Verify round-trip stability via byte comparison
			bz2, err := got.MarshalCBOR()
			if err != nil {
				t.Fatalf(
					"re-marshal %s: %v",
					tc.name,
					err,
				)
			}
			if !bytes.Equal(bz, bz2) {
				t.Fatalf(
					"%s round-trip bytes mismatch",
					tc.name,
				)
			}
		})
	}
}

// ----------------------------------------------------------------
// TestUnmarshalGovAction_Errors
// ----------------------------------------------------------------

func TestUnmarshalGovAction_Errors(t *testing.T) {
	// Empty array
	bz, _ := cbor.Marshal([]any{})
	if _, err := UnmarshalGovAction(bz); err == nil {
		t.Fatalf("expected error for empty array")
	}

	// Unknown type
	bz2, _ := cbor.Marshal([]any{uint64(99)})
	if _, err := UnmarshalGovAction(bz2); err == nil {
		t.Fatalf("expected error for unknown type")
	}

	// Bad kind type
	bz3, _ := cbor.Marshal([]any{"bad"})
	if _, err := UnmarshalGovAction(bz3); err == nil {
		t.Fatalf("expected error for bad kind type")
	}

	// Wrong length for NoConfidence
	bz4, _ := cbor.Marshal(
		[]any{uint64(3), nil, "extra"},
	)
	if _, err := UnmarshalGovAction(bz4); err == nil {
		t.Fatalf("expected error for wrong NoConfidence len")
	}

	// Wrong length for InfoAction
	bz5, _ := cbor.Marshal(
		[]any{uint64(6), "extra"},
	)
	if _, err := UnmarshalGovAction(bz5); err == nil {
		t.Fatalf("expected error for wrong InfoAction len")
	}
}

// ----------------------------------------------------------------
// TestVotingProcedures_Empty
// ----------------------------------------------------------------

func TestVotingProcedures_Empty(t *testing.T) {
	vp := VotingProcedures{}
	bz, err := vp.MarshalCBOR()
	if err != nil {
		t.Fatalf("marshal empty: %v", err)
	}

	var got VotingProcedures
	if err := got.UnmarshalCBOR(bz); err != nil {
		t.Fatalf("unmarshal empty: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty, got %d", len(got))
	}
}

// ----------------------------------------------------------------
// TestProposalProcedures_Empty
// ----------------------------------------------------------------

func TestProposalProcedures_Empty(t *testing.T) {
	ps := ProposalProcedures{}
	bz, err := ps.MarshalCBOR()
	if err != nil {
		t.Fatalf("marshal empty: %v", err)
	}
	var got ProposalProcedures
	if err := got.UnmarshalCBOR(bz); err != nil {
		t.Fatalf("unmarshal empty: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty, got %d", len(got))
	}
}

// ----------------------------------------------------------------
// TestUpdateCommitteeRoundTrip_EmptyMembers
// ----------------------------------------------------------------

func TestUpdateCommitteeRoundTrip_EmptyMembers(t *testing.T) {
	uc := UpdateCommittee{
		PrevActionId: nil,
		Removed:      []Certificate.Credential{},
		Added:        []AddedCommitteeMember{},
		Quorum: Certificate.UnitInterval{
			Num: 1,
			Den: 1,
		},
	}
	bz, err := uc.MarshalCBOR()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got, err := UnmarshalGovAction(bz)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	ucGot := got.(UpdateCommittee)
	if len(ucGot.Removed) != 0 {
		t.Fatalf("expected 0 removed")
	}
	if len(ucGot.Added) != 0 {
		t.Fatalf("expected 0 added")
	}
}

// ----------------------------------------------------------------
// TestProposalProcedureWithUpdateCommittee
// ----------------------------------------------------------------

func TestProposalProcedureWithUpdateCommittee(
	t *testing.T,
) {
	uc := UpdateCommittee{
		PrevActionId: nil,
		Removed: []Certificate.Credential{
			{Code: 0, Hash: testHash(0x01, 0x02)},
		},
		Added: []AddedCommitteeMember{
			{
				Credential: Certificate.Credential{
					Code: 1,
					Hash: testHash(0x03, 0x04),
				},
				Epoch: 500,
			},
		},
		Quorum: Certificate.UnitInterval{
			Num: 2,
			Den: 3,
		},
	}
	pp := ProposalProcedure{
		Deposit:       5000000,
		RewardAccount: []byte{0xE1, 0x01},
		Action:        uc,
		Anchor:        sampleAnchor(),
	}
	bz, err := pp.MarshalCBOR()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got, err := UnmarshalProposalProcedure(bz)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Action.GovActionType() != 4 {
		t.Fatalf("expected type 4")
	}
}

func TestProposalProcedureMarshalCBORNilAction(
	t *testing.T,
) {
	pp := ProposalProcedure{
		Deposit:       1,
		RewardAccount: []byte{0xE1},
		Action:        nil,
		Anchor:        sampleAnchor(),
	}
	if _, err := pp.MarshalCBOR(); err == nil {
		t.Fatal("expected error for nil action")
	}
}

func TestDecodeMapPairsSupportsExtendedAndIndefiniteLengths(
	t *testing.T,
) {
	t.Run("extended_length", func(t *testing.T) {
		data := []byte{
			0xba, 0x00, 0x00, 0x00, 0x01,
			0x01, 0x02,
		}
		pairs, err := decodeMapPairs(data)
		if err != nil {
			t.Fatalf("decode map: %v", err)
		}
		if len(pairs) != 1 {
			t.Fatalf("expected 1 pair, got %d", len(pairs))
		}
		if !bytes.Equal(pairs[0].key, []byte{0x01}) {
			t.Fatalf("unexpected key: %x", pairs[0].key)
		}
		if !bytes.Equal(pairs[0].value, []byte{0x02}) {
			t.Fatalf("unexpected value: %x", pairs[0].value)
		}
	})

	t.Run("indefinite_length", func(t *testing.T) {
		var buf bytes.Buffer
		encMode, err := cbor.EncOptions{
			IndefLength: cbor.IndefLengthAllowed,
		}.EncMode()
		if err != nil {
			t.Fatalf("enc mode: %v", err)
		}
		enc := encMode.NewEncoder(&buf)
		if err := enc.StartIndefiniteMap(); err != nil {
			t.Fatalf("start map: %v", err)
		}
		if err := enc.Encode(uint64(1)); err != nil {
			t.Fatalf("encode key: %v", err)
		}
		if err := enc.Encode(uint64(2)); err != nil {
			t.Fatalf("encode value: %v", err)
		}
		if err := enc.EndIndefinite(); err != nil {
			t.Fatalf("end map: %v", err)
		}

		pairs, err := decodeMapPairs(buf.Bytes())
		if err != nil {
			t.Fatalf("decode indefinite map: %v", err)
		}
		if len(pairs) != 1 {
			t.Fatalf("expected 1 pair, got %d", len(pairs))
		}
	})
}

func TestDecodeArrayItemsSupportsExtendedAndIndefiniteLengths(
	t *testing.T,
) {
	t.Run("extended_length", func(t *testing.T) {
		data := make([]byte, 0, 11)
		data = append(data, 0x9b)
		var length [8]byte
		binary.BigEndian.PutUint64(length[:], 2)
		data = append(data, length[:]...)
		data = append(data, 0x01, 0x02)

		items, err := decodeArrayItems(data)
		if err != nil {
			t.Fatalf("decode array: %v", err)
		}
		if len(items) != 2 {
			t.Fatalf("expected 2 items, got %d", len(items))
		}
	})

	t.Run("indefinite_length", func(t *testing.T) {
		var buf bytes.Buffer
		encMode, err := cbor.EncOptions{
			IndefLength: cbor.IndefLengthAllowed,
		}.EncMode()
		if err != nil {
			t.Fatalf("enc mode: %v", err)
		}
		enc := encMode.NewEncoder(&buf)
		if err := enc.StartIndefiniteArray(); err != nil {
			t.Fatalf("start array: %v", err)
		}
		if err := enc.Encode(uint64(1)); err != nil {
			t.Fatalf("encode first item: %v", err)
		}
		if err := enc.Encode(uint64(2)); err != nil {
			t.Fatalf("encode second item: %v", err)
		}
		if err := enc.EndIndefinite(); err != nil {
			t.Fatalf("end array: %v", err)
		}

		items, err := decodeArrayItems(buf.Bytes())
		if err != nil {
			t.Fatalf("decode indefinite array: %v", err)
		}
		if len(items) != 2 {
			t.Fatalf("expected 2 items, got %d", len(items))
		}
	})
}

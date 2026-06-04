package apollo

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/conway"
)

func newGovernanceTestApollo(t *testing.T) *Apollo {
	t.Helper()
	cc := setupFixedContext()
	addr := testAddress(t)
	addTestUtxo(cc, addr, 100_000_000, 0x90, 0)
	return New(cc).SetWallet(NewExternalWallet(addr)).SetTtl(50000000)
}

func TestSetCurrentTreasuryValue(t *testing.T) {
	a := newGovernanceTestApollo(t).SetCurrentTreasuryValue(1_000_000_000)

	a, err := a.Complete()
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if got := a.GetTx().Body.TxCurrentTreasuryValue; got != 1_000_000_000 {
		t.Fatalf("expected treasury value 1000000000, got %d", got)
	}
}

func TestAddTreasuryDonation(t *testing.T) {
	a := newGovernanceTestApollo(t).
		AddTreasuryDonation(5_000_000).
		AddTreasuryDonation(2_500_000)

	a, err := a.Complete()
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if got := a.GetTx().Body.TxDonation; got != 7_500_000 {
		t.Fatalf("expected donation 7500000, got %d", got)
	}
}

func TestTreasuryFieldsNotSetByDefault(t *testing.T) {
	a := newGovernanceTestApollo(t)

	a, err := a.Complete()
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if a.GetTx().Body.TxCurrentTreasuryValue != 0 {
		t.Fatal("TxCurrentTreasuryValue should be 0 by default")
	}
	if a.GetTx().Body.TxDonation != 0 {
		t.Fatal("TxDonation should be 0 by default")
	}
}

func TestSetCurrentTreasuryValueNegative(t *testing.T) {
	a := newGovernanceTestApollo(t).SetCurrentTreasuryValue(-1)
	_, err := a.Complete()
	if err == nil {
		t.Fatal("expected error for negative treasury value")
	}
}

func TestAddTreasuryDonationNegative(t *testing.T) {
	a := newGovernanceTestApollo(t).AddTreasuryDonation(-1)
	_, err := a.Complete()
	if err == nil {
		t.Fatal("expected error for negative donation amount")
	}
}

func TestAddTreasuryDonationOverflow(t *testing.T) {
	a := newGovernanceTestApollo(t).
		AddTreasuryDonation(math.MaxInt64).
		AddTreasuryDonation(1)
	_, err := a.Complete()
	if err == nil {
		t.Fatal("expected error for donation amount overflow")
	}
}

func TestAddVote(t *testing.T) {
	a := newGovernanceTestApollo(t)
	voter := testVoter(0x01)
	actionId := testGovActionId(0)
	a.AddVote(voter, actionId, common.VotingProcedure{Vote: common.GovVoteYes})

	a, err := a.Complete()
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	vp := a.GetTx().Body.TxVotingProcedures
	if len(vp) != 1 {
		t.Fatalf("expected 1 voter, got %d", len(vp))
	}
	for _, votes := range vp {
		if len(votes) != 1 {
			t.Fatalf("expected 1 vote, got %d", len(votes))
		}
		for _, procedure := range votes {
			if procedure.Vote != common.GovVoteYes {
				t.Fatalf("expected yes vote, got %d", procedure.Vote)
			}
		}
	}
}

func TestAddVoteReplacesExistingVoterAction(t *testing.T) {
	a := newGovernanceTestApollo(t)
	voter := testVoter(0x01)
	actionId := testGovActionId(0)
	a.AddVote(voter, actionId, common.VotingProcedure{Vote: common.GovVoteYes})
	a.AddVote(voter, actionId, common.VotingProcedure{Vote: common.GovVoteNo})

	if len(a.votingProcedures) != 1 {
		t.Fatalf("expected 1 voter, got %d", len(a.votingProcedures))
	}
	for _, votes := range a.votingProcedures {
		if len(votes) != 1 {
			t.Fatalf("expected 1 action, got %d", len(votes))
		}
		for _, procedure := range votes {
			if procedure.Vote != common.GovVoteNo {
				t.Fatalf("expected replacement no vote, got %d", procedure.Vote)
			}
		}
	}
}

func TestAddProposal(t *testing.T) {
	a := newGovernanceTestApollo(t)
	proposal := testInfoProposal(t, 2_000_000, "https://example.com/proposal")
	a.AddProposal(proposal)

	a, err := a.Complete()
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	pp := a.GetTx().Body.TxProposalProcedures
	if len(pp) != 1 {
		t.Fatalf("expected 1 proposal, got %d", len(pp))
	}
	if pp[0].PPDeposit != 2_000_000 {
		t.Fatalf("expected deposit 2000000, got %d", pp[0].PPDeposit)
	}
}

func TestAddMultipleProposals(t *testing.T) {
	a := newGovernanceTestApollo(t)
	a.AddProposal(testInfoProposal(t, 2_000_000, "https://example.com/proposal-1"))
	a.AddProposal(testInfoProposal(t, 3_000_000, "https://example.com/proposal-2"))

	a, err := a.Complete()
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	pp := a.GetTx().Body.TxProposalProcedures
	if len(pp) != 2 {
		t.Fatalf("expected 2 proposals, got %d", len(pp))
	}
	if pp[0].PPDeposit != 2_000_000 || pp[1].PPDeposit != 3_000_000 {
		t.Fatalf("unexpected proposal deposits: %d, %d", pp[0].PPDeposit, pp[1].PPDeposit)
	}
}

func TestVotingAndProposalFieldsNilByDefault(t *testing.T) {
	a := newGovernanceTestApollo(t)

	a, err := a.Complete()
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if a.GetTx().Body.TxVotingProcedures != nil {
		t.Fatal("TxVotingProcedures should be nil by default")
	}
	if a.GetTx().Body.TxProposalProcedures != nil {
		t.Fatal("TxProposalProcedures should be nil by default")
	}
}

func TestRegisterDRep(t *testing.T) {
	a := newGovernanceTestApollo(t)
	cred := testCredential(0xdd)
	a.RegisterDRep(cred, 2_000_000, testGovAnchor("https://example.com/drep.json"))

	cert := requireCertificate[*common.RegistrationDrepCertificate](t, a, common.CertificateTypeRegistrationDrep)
	if cert.DrepCredential.Credential != cred.Credential {
		t.Fatal("drep credential hash mismatch")
	}
	if cert.Anchor == nil {
		t.Fatal("expected anchor")
	}
}

func TestRegisterDRepNoAnchor(t *testing.T) {
	a := newGovernanceTestApollo(t)
	cred := testCredential(0xee)
	a.RegisterDRep(cred, 2_000_000, nil)

	cert := requireCertificate[*common.RegistrationDrepCertificate](t, a, common.CertificateTypeRegistrationDrep)
	if cert.DrepCredential.Credential != cred.Credential {
		t.Fatal("drep credential hash mismatch")
	}
	if cert.Anchor != nil {
		t.Fatal("expected nil anchor")
	}
}

func TestRegisterDRepNegativeCoinSetsError(t *testing.T) {
	a := newGovernanceTestApollo(t)
	cred := testCredential(0xdd)

	ret := a.RegisterDRep(cred, -1, nil)

	if ret != a {
		t.Fatal("RegisterDRep should return the same Apollo builder")
	}
	if a.err == nil {
		t.Fatal("expected builder error")
	}
	if got := a.err.Error(); got != "RegisterDRep: coin must be non-negative" {
		t.Fatalf("error = %q", got)
	}
	if len(a.certificates) != 0 {
		t.Fatalf("expected no certificates, got %d", len(a.certificates))
	}
}

func TestRetireDRep(t *testing.T) {
	a := newGovernanceTestApollo(t)
	cred := testCredential(0xaa)
	a.RetireDRep(cred, 2_000_000)

	cert := requireCertificate[*common.DeregistrationDrepCertificate](t, a, common.CertificateTypeDeregistrationDrep)
	if cert.DrepCredential.Credential != cred.Credential {
		t.Fatal("drep credential hash mismatch")
	}
}

func TestRetireDRepNegativeCoinSetsError(t *testing.T) {
	a := newGovernanceTestApollo(t)
	cred := testCredential(0xaa)

	ret := a.RetireDRep(cred, -1)

	if ret != a {
		t.Fatal("RetireDRep should return the same Apollo builder")
	}
	if a.err == nil {
		t.Fatal("expected builder error")
	}
	if got := a.err.Error(); got != "RetireDRep: coin must be non-negative" {
		t.Fatalf("error = %q", got)
	}
	if len(a.certificates) != 0 {
		t.Fatalf("expected no certificates, got %d", len(a.certificates))
	}
}

func TestUpdateDRep(t *testing.T) {
	a := newGovernanceTestApollo(t)
	cred := testCredential(0xbb)
	a.UpdateDRep(cred, testGovAnchor("https://example.com/update.json"))

	cert := requireCertificate[*common.UpdateDrepCertificate](t, a, common.CertificateTypeUpdateDrep)
	if cert.DrepCredential.Credential != cred.Credential {
		t.Fatal("drep credential hash mismatch")
	}
	if cert.Anchor == nil {
		t.Fatal("expected anchor")
	}
}

func TestUpdateDRepNoAnchor(t *testing.T) {
	a := newGovernanceTestApollo(t)
	cred := testCredential(0xbc)
	a.UpdateDRep(cred, nil)

	cert := requireCertificate[*common.UpdateDrepCertificate](t, a, common.CertificateTypeUpdateDrep)
	if cert.DrepCredential.Credential != cred.Credential {
		t.Fatal("drep credential hash mismatch")
	}
	if cert.Anchor != nil {
		t.Fatal("expected nil anchor")
	}
}

func TestAuthorizeCommitteeHotKey(t *testing.T) {
	a := newGovernanceTestApollo(t)
	cold := testCredential(0x11)
	hot := testCredential(0x22)
	a.AuthorizeCommitteeHotKey(cold, hot)

	cert := requireCertificate[*common.AuthCommitteeHotCertificate](t, a, common.CertificateTypeAuthCommitteeHot)
	if cert.ColdCredential.Credential != cold.Credential {
		t.Fatal("cold credential hash mismatch")
	}
	if cert.HotCredential.Credential != hot.Credential {
		t.Fatal("hot credential hash mismatch")
	}
}

func TestResignCommitteeColdKey(t *testing.T) {
	a := newGovernanceTestApollo(t)
	cold := testCredential(0x33)
	a.ResignCommitteeColdKey(cold, testGovAnchor("https://example.com/resign.json"))

	cert := requireCertificate[*common.ResignCommitteeColdCertificate](t, a, common.CertificateTypeResignCommitteeCold)
	if cert.ColdCredential.Credential != cold.Credential {
		t.Fatal("cold credential hash mismatch")
	}
	if cert.Anchor == nil {
		t.Fatal("expected anchor")
	}
}

func TestResignCommitteeColdKeyNoAnchor(t *testing.T) {
	a := newGovernanceTestApollo(t)
	cold := testCredential(0x44)
	a.ResignCommitteeColdKey(cold, nil)

	cert := requireCertificate[*common.ResignCommitteeColdCertificate](t, a, common.CertificateTypeResignCommitteeCold)
	if cert.ColdCredential.Credential != cold.Credential {
		t.Fatal("cold credential hash mismatch")
	}
	if cert.Anchor != nil {
		t.Fatal("expected nil anchor")
	}
}

func TestSetShelleyMetadataFromJSONInvalidDoesNotMutate(t *testing.T) {
	a := New(nil)
	originalMetadata := map[uint64]any{1: "keep"}
	a.SetShelleyMetadata(originalMetadata)
	originalAuxData := auxDataSnapshot(t, a.auxiliaryData)

	ret, err := a.SetShelleyMetadataFromJSON([]byte("{"))

	if ret != a {
		t.Fatal("SetShelleyMetadataFromJSON should return the same Apollo builder")
	}
	if err == nil {
		t.Fatal("expected invalid JSON error")
	}
	if got := err.Error(); got != "decode metadata JSON: unexpected EOF" {
		t.Fatalf("error = %q", got)
	}
	if got := auxDataSnapshot(t, a.auxiliaryData); got != originalAuxData {
		t.Fatalf("auxiliaryData should not be mutated on error, got %s want %s", got, originalAuxData)
	}
}

func TestSetShelleyMetadataFromJSONWithInvalidSchemaDoesNotMutate(t *testing.T) {
	a := New(nil)
	originalMetadata := map[uint64]any{1: "keep"}
	a.SetShelleyMetadata(originalMetadata)
	originalAuxData := auxDataSnapshot(t, a.auxiliaryData)

	ret, err := a.SetShelleyMetadataFromJSONWithSchema([]byte(`{"1":1}`), MetadataJSONSchema(99))

	if ret != a {
		t.Fatal("SetShelleyMetadataFromJSONWithSchema should return the same Apollo builder")
	}
	if err == nil {
		t.Fatal("expected unsupported schema error")
	}
	if got := err.Error(); got != "unsupported metadata JSON schema 99" {
		t.Fatalf("error = %q", got)
	}
	if got := auxDataSnapshot(t, a.auxiliaryData); got != originalAuxData {
		t.Fatalf("auxiliaryData should not be mutated on error, got %s want %s", got, originalAuxData)
	}
}

func auxDataSnapshot(t *testing.T, data *auxData) string {
	t.Helper()
	if data == nil {
		return "null"
	}
	snapshot, err := json.Marshal(data.metadata)
	if err != nil {
		t.Fatalf("marshal auxiliaryData metadata: %v", err)
	}
	return string(snapshot)
}

func testCredential(first byte) common.Credential {
	var hash common.Blake2b224
	hash[0] = first
	return common.Credential{CredType: common.CredentialTypeAddrKeyHash, Credential: hash}
}

func testVoter(first byte) common.Voter {
	var hash [28]byte
	hash[0] = first
	return common.Voter{Type: common.VoterTypeDRepKeyHash, Hash: hash}
}

func testGovActionId(index uint32) common.GovActionId {
	var txId [32]byte
	txId[0] = 0xab
	return common.GovActionId{TransactionId: txId, GovActionIdx: index}
}

func testGovAnchor(url string) *common.GovAnchor {
	var hash [32]byte
	hash[0] = 0xcd
	return &common.GovAnchor{Url: url, DataHash: hash}
}

func testInfoProposal(t *testing.T, deposit uint64, url string) conway.ConwayProposalProcedure {
	t.Helper()
	addr := testAddress(t)
	return conway.ConwayProposalProcedure{
		PPDeposit:       deposit,
		PPRewardAccount: addr,
		PPGovAction: conway.ConwayGovAction{
			Type:   uint(common.GovActionTypeInfo),
			Action: &common.InfoGovAction{Type: uint(common.GovActionTypeInfo)},
		},
		PPAnchor: *testGovAnchor(url),
	}
}

func requireCertificate[T any](t *testing.T, a *Apollo, certType common.CertificateType) T {
	t.Helper()
	if len(a.certificates) != 1 {
		t.Fatalf("expected 1 certificate, got %d", len(a.certificates))
	}
	if a.certificates[0].Type != uint(certType) {
		t.Fatalf("expected certificate type %d, got %d", certType, a.certificates[0].Type)
	}
	cert, ok := a.certificates[0].Certificate.(T)
	if !ok {
		t.Fatalf("unexpected certificate concrete type %T", a.certificates[0].Certificate)
	}
	return cert
}

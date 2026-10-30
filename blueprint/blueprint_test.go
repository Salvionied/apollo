package blueprint

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/Salvionied/apollo/v2"
	"github.com/Salvionied/apollo/v2/backend"
	"github.com/Salvionied/apollo/v2/backend/fixed"
	"github.com/blinklabs-io/gouroboros/ledger/babbage"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/mary"
	"github.com/blinklabs-io/gouroboros/ledger/shelley"
	"github.com/blinklabs-io/plutigo/data"
)

func loadFixture(t *testing.T, name string) *Blueprint {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	bp, err := Parse(raw)
	if err != nil {
		t.Fatalf("%s: Parse: %v", name, err)
	}
	return bp
}

func assertWireRoundTrip(t *testing.T, pd data.PlutusData) {
	t.Helper()
	encoded, err := data.Encode(pd)
	if err != nil {
		t.Fatalf("data.Encode: %v", err)
	}
	decoded, err := data.Decode(encoded)
	if err != nil {
		t.Fatalf("data.Decode: %v", err)
	}
	again, err := data.Encode(decoded)
	if err != nil {
		t.Fatalf("data.Encode decoded: %v", err)
	}
	if !bytes.Equal(encoded, again) {
		t.Fatal("PlutusData wire round-trip changed bytes")
	}
}

type escrowNote struct {
	Tag    []byte
	Weight int64
}

type optionNote struct {
	Some *escrowNote
	None *struct{}
}

type escrowDatum struct {
	Owner  []byte
	Amount int64
	Memo   optionNote
	Steps  []escrowNote
}

func TestSpendRecordFixture(t *testing.T) {
	bp := loadFixture(t, "spend_record.json")
	owner := bytes.Repeat([]byte{0x11}, 28)

	for _, width := range []int64{1, 255, 65535, 1_000_000_000} {
		original := escrowDatum{
			Owner:  owner,
			Amount: width,
			Memo:   optionNote{Some: &escrowNote{Tag: []byte("memo"), Weight: width}},
			Steps:  []escrowNote{{Tag: []byte("a"), Weight: 1}, {Tag: []byte("bb"), Weight: 2}},
		}
		pd, err := bp.EncodeDatum("escrow.spend", original)
		if err != nil {
			t.Fatalf("width %d: %v", width, err)
		}
		assertWireRoundTrip(t, pd)

		var decoded escrowDatum
		if err := bp.DecodeDatum("escrow.spend", pd, &decoded); err != nil {
			t.Fatalf("width %d decode: %v", width, err)
		}
		reencoded, err := bp.EncodeDatum("escrow.spend", decoded)
		if err != nil {
			t.Fatal(err)
		}
		b1, err := data.Encode(pd)
		if err != nil {
			t.Fatal(err)
		}
		b2, err := data.Encode(reencoded)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(b1, b2) {
			t.Fatalf("width %d: re-encoded datum differs", width)
		}
	}

	// Below minimum amount and short owner must fail.
	bad := escrowDatum{
		Owner:  []byte{0x01},
		Amount: 1,
		Memo:   optionNote{None: &struct{}{}},
		Steps:  []escrowNote{{Tag: []byte("a"), Weight: 1}},
	}
	if _, err := bp.EncodeDatum("escrow.spend", bad); err == nil {
		t.Fatal("expected short owner to fail minLength")
	}
}

type mintAction struct {
	Bootstrap *struct{}
	MintCount *int64
	BurnBatch *[][]byte
}

func TestMintPolicyFixture(t *testing.T) {
	bp := loadFixture(t, "mint_policy.json")
	qty := int64(25)
	cases := []mintAction{
		{Bootstrap: &struct{}{}},
		{MintCount: &qty},
		{BurnBatch: &[][]byte{[]byte("one"), []byte("two")}},
	}
	for i, c := range cases {
		pd, err := bp.EncodeRedeemer("policy.mint", "mint", c)
		if err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
		assertWireRoundTrip(t, pd)
		var out mintAction
		if err := bp.DecodeRedeemer("policy.mint", "mint", pd, &out); err != nil {
			t.Fatalf("case %d decode: %v", i, err)
		}
	}

	dupNames := mintAction{BurnBatch: &[][]byte{[]byte("same"), []byte("same")}}
	if _, err := bp.EncodeRedeemer("policy.mint", "mint", dupNames); err == nil {
		t.Fatal("expected duplicate BurnBatch names to fail uniqueItems")
	}
	overMax := int64(5000)
	if _, err := bp.EncodeRedeemer("policy.mint", "mint", mintAction{MintCount: &overMax}); err == nil {
		t.Fatal("expected quantity above maximum to fail")
	}
}

type nestedAccount struct {
	CredentialHash []byte
	Epoch          int64
}

type nestedState struct {
	Account nestedAccount
}

type spendTarget struct {
	Target []byte
}

func TestMultiPurposeFixture(t *testing.T) {
	bp := loadFixture(t, "multi_purpose.json")
	hash := bytes.Repeat([]byte{0x22}, 28)

	datum := nestedState{Account: nestedAccount{CredentialHash: hash, Epoch: 510}}
	pd, err := bp.EncodeDatum("pool.main", datum)
	if err != nil {
		t.Fatal(err)
	}
	var outDatum nestedState
	if err := bp.DecodeDatum("pool.main", pd, &outDatum); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(outDatum.Account.CredentialHash, hash) || outDatum.Account.Epoch != 510 {
		t.Fatalf("unexpected decoded state: %+v", outDatum)
	}

	spendPd, err := bp.EncodeRedeemer("pool.main", "spend", spendTarget{Target: hash})
	if err != nil {
		t.Fatal(err)
	}
	var outSpend spendTarget
	if err := bp.DecodeRedeemer("pool.main", "spend", spendPd, &outSpend); err != nil {
		t.Fatal(err)
	}

	withdrawPd, err := bp.EncodeRedeemer("pool.main", "withdraw", int64(99))
	if err != nil {
		t.Fatal(err)
	}
	var outWithdraw int64
	if err := bp.DecodeRedeemer("pool.main", "withdraw", withdrawPd, &outWithdraw); err != nil {
		t.Fatal(err)
	}
	if outWithdraw != 99 {
		t.Fatalf("withdraw redeemer = %d, want 99", outWithdraw)
	}

	if _, err := bp.EncodeRedeemer("pool.main", "mint", int64(1)); err == nil {
		t.Fatal("expected unsupported purpose mint to fail")
	}
}

type pairTuple struct {
	Key []byte
	Val int64
}

type vaultDatum struct {
	Balances  map[string]int64
	PairTuple pairTuple
}

func TestAssetMapAndTupleStayDistinct(t *testing.T) {
	bp := loadFixture(t, "asset_map_tuple.json")
	original := vaultDatum{
		Balances:  map[string]int64{"aa": 10, "bb": 20},
		PairTuple: pairTuple{Key: []byte{0x01, 0x02}, Val: 7},
	}
	pd, err := bp.EncodeDatum("vault.spend", original)
	if err != nil {
		t.Fatal(err)
	}
	constr := pd.(*data.Constr)
	if _, ok := constr.Fields[0].(*data.Map); !ok {
		t.Fatalf("field 0 must be a Plutus Map, got %T", constr.Fields[0])
	}
	if _, ok := constr.Fields[1].(*data.List); !ok {
		t.Fatalf("field 1 must be a Plutus List tuple, got %T", constr.Fields[1])
	}

	var decoded vaultDatum
	if err := bp.DecodeDatum("vault.spend", pd, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Balances["aa"] != 10 || decoded.Balances["bb"] != 20 || decoded.PairTuple.Val != 7 {
		t.Fatalf("unexpected decoded vault: %+v", decoded)
	}

	swapped := data.NewConstr(0, constr.Fields[1], constr.Fields[0])
	if err := bp.DecodeDatum("vault.spend", swapped, &decoded); err == nil {
		t.Fatal("expected swapped map and tuple encodings to fail")
	}
}

type wideDatum struct {
	Huge       *big.Int
	Buckets    []map[string]int64
	Extra      data.PlutusData
	UnitMarker struct{}
}

func TestWideDatumFixture(t *testing.T) {
	bp := loadFixture(t, "wide_datum.json")
	bigVal := new(big.Int).Lsh(big.NewInt(1), 80)
	original := wideDatum{
		Huge:    bigVal,
		Buckets: []map[string]int64{{}, {"cafe": 3}},
		Extra:   data.NewConstr(7, data.NewByteString([]byte("raw"))),
	}
	pd, err := bp.EncodeDatum("wide.spend", original)
	if err != nil {
		t.Fatal(err)
	}
	assertWireRoundTrip(t, pd)
	if pd.(*data.Constr).Tag.Uint64() != 4 {
		t.Fatalf("expected constructor index 4, got %s", pd.(*data.Constr).Tag)
	}

	var decoded wideDatum
	if err := bp.DecodeDatum("wide.spend", pd, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Huge.Cmp(bigVal) != 0 || len(decoded.Buckets) != 2 || len(decoded.Buckets[0]) != 0 {
		t.Fatalf("unexpected decoded wide datum: %+v", decoded)
	}
}

type govCredential struct {
	KeyCredential    *[]byte
	ScriptCredential *[]byte
}

type govActionId struct {
	TxId         []byte
	GovActionIdx int64
}

type optionActionId struct {
	Some *govActionId
	None *struct{}
}

type govAnchor struct {
	Url      []byte
	DataHash []byte
}

type optionAnchor struct {
	Some *govAnchor
	None *struct{}
}

type govQuorum struct {
	Numerator   int64
	Denominator int64
}

type govParamChange struct {
	Previous  optionActionId
	MaxTxSize int64
}

type govHardFork struct {
	Major int64
	Minor int64
}

type govUpdateCommittee struct {
	Removed []govCredential
	Added   map[string]int64
	Quorum  govQuorum
}

type govAction struct {
	ParameterChange     *govParamChange
	HardForkInitiation  *govHardFork
	TreasuryWithdrawals *map[string]int64
	NoConfidence        *optionActionId
	UpdateCommittee     *govUpdateCommittee
	NewConstitution     *[]byte
	InfoAction          *struct{}
}

type govProposalDatum struct {
	PreviousAction optionActionId
	Action         govAction
	Anchor         govAnchor
}

type govVoter struct {
	CommitteeVoter *govCredential
	DRepVoter      *govCredential
	StakePoolVoter *[]byte
}

type govVoteDecision struct {
	VoteNo      *struct{}
	VoteYes     *struct{}
	VoteAbstain *struct{}
}

type govBallot struct {
	Voter    govVoter
	ActionId govActionId
	Decision govVoteDecision
	Anchor   optionAnchor
}

type govRedeemer struct {
	CastSingle *govBallot
	CastBatch  *[]govBallot
}

func TestGovernanceFixtureMatchesGouroborosAndBuildsVoteTx(t *testing.T) {
	bp := loadFixture(t, "governance.json")
	txId := bytes.Repeat([]byte{0x44}, 32)
	credHash := bytes.Repeat([]byte{0x55}, 28)
	anchorHash := bytes.Repeat([]byte{0x66}, 32)

	actionDef, err := bp.Definition("gov/ActionId")
	if err != nil {
		t.Fatal(err)
	}
	actionPd, err := bp.Encode(actionDef, govActionId{TxId: txId, GovActionIdx: 2})
	if err != nil {
		t.Fatal(err)
	}
	ledgerAction := common.GovActionId{TransactionId: [32]byte(txId), GovActionIdx: 2}
	b1, err := data.Encode(actionPd)
	if err != nil {
		t.Fatal(err)
	}
	b2, err := data.Encode(ledgerAction.ToPlutusData())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b1, b2) {
		t.Fatal("blueprint ActionId does not match common.GovActionId.ToPlutusData")
	}

	voterDef, err := bp.Definition("gov/Voter")
	if err != nil {
		t.Fatal(err)
	}
	voterPd, err := bp.Encode(voterDef, govVoter{DRepVoter: &govCredential{KeyCredential: &credHash}})
	if err != nil {
		t.Fatal(err)
	}
	ledgerVoter := common.Voter{Type: common.VoterTypeDRepKeyHash, Hash: [28]byte(credHash)}
	b1, err = data.Encode(voterPd)
	if err != nil {
		t.Fatal(err)
	}
	b2, err = data.Encode(ledgerVoter.ToPlutusData())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b1, b2) {
		t.Fatal("blueprint Voter does not match common.Voter.ToPlutusData")
	}

	decisionDef, err := bp.Definition("gov/VoteDecision")
	if err != nil {
		t.Fatal(err)
	}
	decisionPd, err := bp.Encode(decisionDef, govVoteDecision{VoteYes: &struct{}{}})
	if err != nil {
		t.Fatal(err)
	}
	b1, err = data.Encode(decisionPd)
	if err != nil {
		t.Fatal(err)
	}
	b2, err = data.Encode(common.Vote(common.GovVoteYes).ToPlutusData())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b1, b2) {
		t.Fatal("blueprint VoteDecision does not match common.Vote.ToPlutusData")
	}

	// Round-trip all seven governance action variants.
	withdrawals := map[string]int64{hex.EncodeToString(credHash): 5_000_000}
	prevSome := optionActionId{Some: &govActionId{TxId: txId, GovActionIdx: 0}}
	actions := []govAction{
		{ParameterChange: &govParamChange{Previous: prevSome, MaxTxSize: 16384}},
		{HardForkInitiation: &govHardFork{Major: 10, Minor: 0}},
		{TreasuryWithdrawals: &withdrawals},
		{NoConfidence: &prevSome},
		{UpdateCommittee: &govUpdateCommittee{
			Removed: []govCredential{{KeyCredential: &credHash}},
			Added:   map[string]int64{hex.EncodeToString(credHash): 600},
			Quorum:  govQuorum{Numerator: 2, Denominator: 3},
		}},
		{NewConstitution: &credHash},
		{InfoAction: &struct{}{}},
	}
	for idx, act := range actions {
		datum := govProposalDatum{
			PreviousAction: optionActionId{None: &struct{}{}},
			Action:         act,
			Anchor:         govAnchor{Url: []byte("https://example.org/gov.json"), DataHash: anchorHash},
		}
		pd, err := bp.EncodeDatum("governance.guard", datum)
		if err != nil {
			t.Fatalf("action %d encode: %v", idx, err)
		}
		if pd.(*data.Constr).Fields[1].(*data.Constr).Tag.Cmp(big.NewInt(int64(idx))) != 0 {
			t.Fatalf("action %d encoded with wrong constructor tag", idx)
		}
		var out govProposalDatum
		if err := bp.DecodeDatum("governance.guard", pd, &out); err != nil {
			t.Fatalf("action %d decode: %v", idx, err)
		}
	}

	ballot := govBallot{
		Voter:    govVoter{DRepVoter: &govCredential{KeyCredential: &credHash}},
		ActionId: govActionId{TxId: txId, GovActionIdx: 2},
		Decision: govVoteDecision{VoteYes: &struct{}{}},
		Anchor:   optionAnchor{None: &struct{}{}},
	}
	redeemerPd, err := bp.EncodeRedeemer("governance.guard", "spend", govRedeemer{CastBatch: &[]govBallot{ballot}})
	if err != nil {
		t.Fatal(err)
	}
	var outRedeemer govRedeemer
	if err := bp.DecodeRedeemer("governance.guard", "spend", redeemerPd, &outRedeemer); err != nil {
		t.Fatal(err)
	}
	if outRedeemer.CastBatch == nil || len(*outRedeemer.CastBatch) != 1 {
		t.Fatalf("unexpected decoded governance redeemer: %+v", outRedeemer)
	}

	// Build a transaction carrying the same vote through AddVote.
	cc, walletAddr := newBlueprintTxContext(t)
	txBuilder, err := apollo.New(cc).SetMaxFee(2_000_000).
		SetWallet(apollo.NewExternalWallet(walletAddr)).
		SetTtl(50000000).
		AddVote(ledgerVoter, ledgerAction, common.VotingProcedure{Vote: common.GovVoteYes}).
		Complete()
	if err != nil {
		t.Fatalf("Complete vote tx: %v", err)
	}
	proc, ok := txBuilder.GetTx().Body.TxVotingProcedures.Lookup(ledgerVoter, ledgerAction)
	if !ok || proc.Vote != common.GovVoteYes {
		t.Fatalf("transaction vote does not match encoded ballot: %+v", proc)
	}
}

func TestInvalidBlueprintCases(t *testing.T) {
	if _, err := Parse([]byte("{not-json")); err == nil {
		t.Fatal("expected truncated JSON to fail")
	}

	raw, err := os.ReadFile(filepath.Join("testdata", "invalid_cases.json"))
	if err != nil {
		t.Fatal(err)
	}
	var cases map[string]json.RawMessage
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatal(err)
	}
	for name, doc := range cases {
		if _, err := Parse(doc); err == nil {
			t.Fatalf("case %s: expected Parse to fail", name)
		}
	}

	bp := loadFixture(t, "spend_record.json")
	note := escrowNote{Tag: []byte("x"), Weight: 1}
	bothSet := escrowDatum{
		Owner:  bytes.Repeat([]byte{0x11}, 28),
		Amount: 1,
		Memo:   optionNote{Some: &note, None: &struct{}{}},
		Steps:  []escrowNote{note},
	}
	if _, err := bp.EncodeDatum("escrow.spend", bothSet); err == nil {
		t.Fatal("expected anyOf with both variants set to fail")
	}
	neitherSet := escrowDatum{
		Owner:  bytes.Repeat([]byte{0x11}, 28),
		Amount: 1,
		Steps:  []escrowNote{note},
	}
	if _, err := bp.EncodeDatum("escrow.spend", neitherSet); err == nil {
		t.Fatal("expected anyOf with neither variant set to fail")
	}
	emptySteps := escrowDatum{
		Owner:  bytes.Repeat([]byte{0x11}, 28),
		Amount: 1,
		Memo:   optionNote{None: &struct{}{}},
	}
	if _, err := bp.EncodeDatum("escrow.spend", emptySteps); err == nil {
		t.Fatal("expected empty steps list to fail minItems")
	}

	if _, err := bp.Script("does.not.exist"); err == nil {
		t.Fatal("expected missing validator script to fail")
	}
	script, err := bp.Script("escrow.spend")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := script.ApplyParams(int64(1)); err == nil {
		t.Fatal("expected parameter count mismatch to fail")
	}
	if _, err := script.V1(); err == nil {
		t.Fatal("expected V1() on a v3 validator to fail")
	}
	if _, err := script.SpendAddress(9, nil); err == nil {
		t.Fatal("expected invalid network id to fail")
	}
}

type lockDatum struct {
	Beneficiary []byte
	Nonce       int64
}

type lockRedeemer struct {
	Step int64
}

type paramConfigBody struct {
	Owner     []byte
	Threshold *big.Int
	Tags      []int64
}

type paramConfig struct {
	Config   *paramConfigBody
	Disabled *struct{}
}

func TestBlueprintBuildsSpendMintWithdrawAndParameterizedScripts(t *testing.T) {
	bp := loadFixture(t, "tx_actions.json")
	beneficiary := bytes.Repeat([]byte{0x77}, 28)

	// 1. Spend: encode datum and redeemer, derive script address, pay and collect.
	spendScript, err := bp.Script("lock.spend")
	if err != nil {
		t.Fatal(err)
	}
	spendAddr, err := spendScript.SpendAddress(common.AddressNetworkTestnet, nil)
	if err != nil {
		t.Fatal(err)
	}
	datumPd, err := bp.EncodeDatum("lock.spend", lockDatum{Beneficiary: beneficiary, Nonce: 1})
	if err != nil {
		t.Fatal(err)
	}
	redeemerPd, err := bp.EncodeRedeemer("lock.spend", "spend", lockRedeemer{Step: 2})
	if err != nil {
		t.Fatal(err)
	}
	v3Spend, err := spendScript.V3()
	if err != nil {
		t.Fatal(err)
	}

	cc, walletAddr := newBlueprintTxContext(t)
	scriptUtxo := makeScriptUtxo(t, spendAddr, datumPd, 5_000_000)
	cc.AddUtxo(spendAddr, scriptUtxo)
	builtSpend, err := apollo.New(cc).SetMaxFee(2_000_000).
		SetWallet(apollo.NewExternalWallet(walletAddr)).
		DisableExecutionUnitsEstimation().
		AttachScript(v3Spend).
		CollectFrom(scriptUtxo, common.Datum{Data: redeemerPd}, common.ExUnits{Memory: 1000, Steps: 1000}).
		PayToContract(spendAddr, &common.Datum{Data: datumPd}, 2_000_000).
		Complete()
	if err != nil {
		t.Fatalf("spend tx Complete: %v", err)
	}
	if len(builtSpend.GetTx().WitnessSet.WsPlutusV3Scripts.Items()) != 1 {
		t.Fatal("expected attached V3 spend script in witness set")
	}

	// 2. Mint: apply seed parameter, derive unit/policy id, and build a mint transaction.
	mintBase, err := bp.Script("token.mint")
	if err != nil {
		t.Fatal(err)
	}
	mintApplied, err := mintBase.ApplyParams(int64(7))
	if err != nil {
		t.Fatal(err)
	}
	unit, err := mintApplied.Script.Unit([]byte("nft"), 1)
	if err != nil {
		t.Fatal(err)
	}
	mintRedeemerPd, err := bp.EncodeRedeemer("token.mint", "mint", int64(1))
	if err != nil {
		t.Fatal(err)
	}
	v3Mint, err := mintApplied.Script.V3()
	if err != nil {
		t.Fatal(err)
	}
	ccMint, walletAddr := newBlueprintTxContext(t)
	builtMint, err := apollo.New(ccMint).SetMaxFee(2_000_000).
		SetWallet(apollo.NewExternalWallet(walletAddr)).
		DisableExecutionUnitsEstimation().
		AttachScript(v3Mint).
		Mint(unit, &common.Datum{Data: mintRedeemerPd}, &common.ExUnits{Memory: 1000, Steps: 1000}).
		Complete()
	if err != nil {
		t.Fatalf("mint tx Complete: %v", err)
	}
	if len(builtMint.GetTx().Body.TxMint.Policies()) != 1 {
		t.Fatal("expected 1 mint policy in transaction body")
	}

	// 3. Withdraw: derive reward address and attach withdrawal redeemer.
	withdrawScript, err := bp.Script("reward.withdraw")
	if err != nil {
		t.Fatal(err)
	}
	rewardAddr, err := withdrawScript.RewardAddress(common.AddressNetworkTestnet)
	if err != nil {
		t.Fatal(err)
	}
	withdrawRedeemerPd, err := bp.EncodeRedeemer("reward.withdraw", "withdraw", []byte{0xca, 0xfe})
	if err != nil {
		t.Fatal(err)
	}
	v3Withdraw, err := withdrawScript.V3()
	if err != nil {
		t.Fatal(err)
	}
	ccWithdraw, walletAddr := newBlueprintTxContext(t)
	builtWithdraw, err := apollo.New(ccWithdraw).SetMaxFee(2_000_000).
		SetWallet(apollo.NewExternalWallet(walletAddr)).
		DisableExecutionUnitsEstimation().
		AttachScript(v3Withdraw).
		AddWithdrawal(rewardAddr, 0, &common.Datum{Data: withdrawRedeemerPd}, &common.ExUnits{Memory: 1000, Steps: 1000}).
		Complete()
	if err != nil {
		t.Fatalf("withdraw tx Complete: %v", err)
	}
	if len(builtWithdraw.GetTx().Body.TxWithdrawals) != 1 {
		t.Fatal("expected 1 withdrawal in transaction body")
	}

	// 4. Parameterized Aiken vector: blueprint ApplyParams must match apollo.ApplyParamsToScript.
	paramScript, err := bp.Script("params.else")
	if err != nil {
		t.Fatal(err)
	}
	big70 := new(big.Int).Lsh(big.NewInt(1), 70)
	extra := data.NewMap([][2]data.PlutusData{
		{data.NewInteger(big.NewInt(1)), data.NewList()},
		{data.NewByteString(nil), data.NewConstr(7)},
		{
			data.NewConstr(200, data.NewByteString(make([]byte, 70))),
			data.NewInteger(new(big.Int).Neg(big70)),
		},
	})
	cfg := paramConfig{
		Config: &paramConfigBody{
			Owner:     []byte{0xca, 0xfe},
			Threshold: big70,
			Tags:      []int64{1, 2, 3},
		},
	}
	applied, err := paramScript.ApplyParams([]byte{0xde, 0xad, 0xbe, 0xef}, int64(-1_000_000), cfg, extra)
	if err != nil {
		t.Fatal(err)
	}
	// Stepwise ApplyParams must produce the exact same bytes and hash.
	step1, err := paramScript.ApplyParams([]byte{0xde, 0xad, 0xbe, 0xef}, int64(-1_000_000))
	if err != nil {
		t.Fatal(err)
	}
	if len(step1.Script.RemainingParameters()) != 2 {
		t.Fatalf("remaining params after step1 = %d, want 2", len(step1.Script.RemainingParameters()))
	}
	step2, err := step1.Script.ApplyParams(cfg, extra)
	if err != nil {
		t.Fatal(err)
	}
	if step2.HashHex != applied.HashHex || !bytes.Equal(step2.Script.Bytes(), applied.Script.Bytes()) {
		t.Fatal("stepwise ApplyParams does not match single-call ApplyParams")
	}
	direct, err := apollo.ApplyParamsToScript(
		common.PlutusV3Script(paramScript.Bytes()),
		data.NewByteString([]byte{0xde, 0xad, 0xbe, 0xef}),
		data.NewInteger(big.NewInt(-1_000_000)),
		data.NewConstr(0,
			data.NewByteString([]byte{0xca, 0xfe}),
			data.NewInteger(big70),
			data.NewList(data.NewInteger(big.NewInt(1)), data.NewInteger(big.NewInt(2)), data.NewInteger(big.NewInt(3))),
		),
		extra,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(applied.Script.Bytes(), []byte(direct)) {
		t.Fatal("blueprint ApplyParams bytes differ from apollo.ApplyParamsToScript")
	}
}

func TestBlueprintDijkstraV4GuardAndDeveloperHelpers(t *testing.T) {
	bp, err := ParseFile(filepath.Join("testdata", "spend_record.json"))
	if err != nil {
		t.Fatal(err)
	}
	names := bp.ValidatorNames()
	if len(names) != 1 || names[0] != "escrow.spend" {
		t.Fatalf("ValidatorNames = %v", names)
	}

	v4Raw := []byte(`{
		"preamble": {"title": "dijkstra/guard", "version": "1.0.0", "plutusVersion": "v4"},
		"validators": [{
			"title": "guard.observe",
			"datum": {"purpose": "guard", "schema": {"dataType": "value"}},
			"redeemer": {
				"oneOf": [
					{"purpose": "guard", "schema": {"dataType": "integer"}},
					{"purpose": "vote", "schema": {"dataType": "bytes"}},
					{"purpose": "propose", "schema": {"dataType": "integer"}}
				]
			},
			"parameters": [{"title": "nonce", "purpose": "guard", "schema": {"dataType": "integer"}}],
			"compiledCode": "46010100200101"
		}]
	}`)
	v4Bp, err := Parse(v4Raw)
	if err != nil {
		t.Fatal(err)
	}
	baseScript, err := v4Bp.Script("guard.observe")
	if err != nil {
		t.Fatal(err)
	}
	expectedHash, err := baseScript.HashHex()
	if err != nil {
		t.Fatal(err)
	}
	v4Bp.Validators[0].Hash = expectedHash
	if err := baseScript.VerifyHash(); err != nil {
		t.Fatalf("VerifyHash: %v", err)
	}
	// A mismatched hash in plutus.json must make Script() fail automatically.
	v4Bp.Validators[0].Hash = "00000000000000000000000000000000000000000000000000000000"
	if _, err := v4Bp.Script("guard.observe"); err == nil {
		t.Fatal("expected unparameterized validator with wrong hash to fail")
	}
	v4Bp.Validators[0].Hash = expectedHash

	appliedV4, err := baseScript.ApplyParamsData(data.NewInteger(big.NewInt(42)))
	if err != nil {
		t.Fatal(err)
	}
	if appliedV4.HashHex == expectedHash {
		t.Fatal("expected parameter application to change script hash")
	}
	if err := appliedV4.Script.VerifyHash(); err == nil {
		t.Fatal("expected VerifyHash to fail after ApplyParamsData")
	}
	if appliedV4.Script.CborHex() != "4c010100320014c102182a0001" {
		t.Fatalf("unexpected V4 applied CborHex: %s", appliedV4.Script.CborHex())
	}
	if _, err := appliedV4.Script.V4(); err != nil {
		t.Fatal(err)
	}
	if _, err := appliedV4.Script.SpendAddressBech32(common.AddressNetworkTestnet, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := appliedV4.Script.RewardAddressBech32(common.AddressNetworkTestnet); err != nil {
		t.Fatal(err)
	}

	// Encode and decode Plutus V4 data.Value (CBOR tag 1401) through datum helpers.
	v4Val := &data.Value{
		Inner: data.NewMap([][2]data.PlutusData{
			{
				data.NewByteString(bytes.Repeat([]byte{0xaa}, 28)),
				data.NewMap([][2]data.PlutusData{{data.NewByteString([]byte("T")), data.NewInteger(big.NewInt(5))}}),
			},
		}).(*data.Map),
	}
	datumVal, err := v4Bp.EncodeDatumValue("guard.observe", v4Val)
	if err != nil {
		t.Fatal(err)
	}
	var decodedV4 data.Value
	if err := v4Bp.DecodeDatumValue("guard.observe", datumVal, &decodedV4); err != nil {
		t.Fatal(err)
	}
	if !decodedV4.Equal(v4Val) {
		t.Fatal("decoded V4 data.Value does not equal input")
	}

	// Build a Dijkstra transaction guarded by the V4 validator via PrepareGuard.
	bundle, err := appliedV4.Script.PrepareGuard(int64(9))
	if err != nil {
		t.Fatal(err)
	}
	if bundle.WitnessScript != nil {
		t.Fatal("Plutus V4 GuardBundle must leave WitnessScript nil")
	}
	var decodedGuardRedeemer int64
	if err := v4Bp.DecodeRedeemerValue("guard.observe", "guard", bundle.Redeemer, &decodedGuardRedeemer); err != nil {
		t.Fatal(err)
	}
	if decodedGuardRedeemer != 9 {
		t.Fatalf("decoded guard redeemer = %d, want 9", decodedGuardRedeemer)
	}

	cc, walletAddr := newBlueprintTxContext(t)
	pp, err := cc.ProtocolParams()
	if err != nil {
		t.Fatal(err)
	}
	pp.ProtocolMajorVersion = 12
	pp.MinFeeRefScriptCostPerByte = 15
	pp.MinFeeReferenceScriptsRange = 1000
	pp.MinFeeReferenceScriptsMultiplierRational = big.NewRat(2, 1)
	costModel := make([]int64, 300)
	for i := range costModel {
		costModel[i] = 10
	}
	pp.CostModels["PlutusV4"] = costModel
	cc = fixed.NewFixedChainContext(pp, backend.GenesisParameters{NetworkMagic: 1}, 0)
	var feeTx, collTx, refTx common.Blake2b256
	feeTx[0] = 0x10
	collTx[0] = 0x20
	refTx[0] = 0x40
	feeOut := babbage.BabbageTransactionOutput{
		OutputAddress: walletAddr,
		OutputAmount:  mary.MaryTransactionOutputValue{Amount: 50_000_000},
	}
	collOut := babbage.BabbageTransactionOutput{
		OutputAddress: walletAddr,
		OutputAmount:  mary.MaryTransactionOutputValue{Amount: 10_000_000},
	}
	cc.AddUtxo(walletAddr, common.Utxo{
		Id:     shelley.ShelleyTransactionInput{TxId: feeTx, OutputIndex: 0},
		Output: &feeOut,
	})
	cc.AddUtxo(walletAddr, common.Utxo{
		Id:     shelley.ShelleyTransactionInput{TxId: collTx, OutputIndex: 0},
		Output: &collOut,
	})
	cc.AddUtxoByRef(common.Utxo{
		Id: shelley.ShelleyTransactionInput{TxId: refTx, OutputIndex: 0},
		Output: &babbage.BabbageTransactionOutput{
			OutputAddress:  walletAddr,
			OutputAmount:   mary.MaryTransactionOutputValue{Amount: 5_000_000},
			TxOutScriptRef: bundle.ScriptRef,
		},
	})

	exUnits := common.ExUnits{Memory: 1000, Steps: 1000}
	dijkstraBuilder, err := apollo.New(cc).SetMaxFee(2_000_000).
		SetWallet(apollo.NewExternalWallet(walletAddr)).
		SetTargetEra(apollo.EraDijkstra).
		DisableExecutionUnitsEstimation().
		AddGuard(bundle.Credential, bundle.Redeemer, &exUnits).
		AddRequiredTopLevelGuard(bundle.Credential, datumVal).
		AddReferenceInput(hex.EncodeToString(refTx[:]), 0)
	if err != nil {
		t.Fatal(err)
	}
	dijkstraBuilder, err = dijkstraBuilder.Complete()
	if err != nil {
		t.Fatalf("Dijkstra guard Complete: %v", err)
	}
	if dijkstraBuilder.GetDijkstraTx() == nil {
		t.Fatal("expected completed Dijkstra transaction")
	}

	// Test DecodeOutputDatum, DatumHash, EncodeDefinition, EncodeParameter, ApplyValidatorParams, and WriteFile.
	spendUt, err := bp.EncodeDatum("escrow.spend", escrowDatum{
		Owner:  bytes.Repeat([]byte{0x11}, 28),
		Amount: 5,
		Memo:   optionNote{None: &struct{}{}},
		Steps:  []escrowNote{{Tag: []byte("s"), Weight: 1}},
	})
	if err != nil {
		t.Fatal(err)
	}
	utxo := makeScriptUtxo(t, walletAddr, spendUt, 2_000_000)
	var fromOutput escrowDatum
	if err := bp.DecodeOutputDatum("escrow.spend", utxo.Output, &fromOutput); err != nil {
		t.Fatal(err)
	}
	if fromOutput.Amount != 5 {
		t.Fatalf("DecodeOutputDatum amount = %d, want 5", fromOutput.Amount)
	}
	dh, err := bp.DatumHashHex("escrow.spend", fromOutput)
	if err != nil {
		t.Fatal(err)
	}
	if len(dh) != 64 {
		t.Fatalf("unexpected DatumHashHex: %s", dh)
	}
	notePd, err := bp.EncodeDefinition("escrow/Note", escrowNote{Tag: []byte("n"), Weight: 3})
	if err != nil {
		t.Fatal(err)
	}
	var noteOut escrowNote
	if err := bp.DecodeDefinition("escrow/Note", notePd, &noteOut); err != nil {
		t.Fatal(err)
	}
	paramPd, err := v4Bp.EncodeParameter("guard.observe", 0, int64(42))
	if err != nil {
		t.Fatal(err)
	}
	var paramOut int64
	if err := v4Bp.DecodeParameter("guard.observe", 0, paramPd, &paramOut); err != nil {
		t.Fatal(err)
	}
	appliedBp, err := v4Bp.ApplyValidatorParams("guard.observe", int64(42))
	if err != nil {
		t.Fatal(err)
	}
	outPath := filepath.Join(t.TempDir(), "applied.plutus.json")
	if err := appliedBp.WriteFile(outPath); err != nil {
		t.Fatal(err)
	}
	reparsed, err := ParseFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	reScript, err := reparsed.Script("guard.observe")
	if err != nil {
		t.Fatal(err)
	}
	if len(reScript.RemainingParameters()) != 0 || reScript.CborHex() != appliedV4.Script.CborHex() {
		t.Fatal("WriteFile/ParseFile after ApplyValidatorParams did not preserve applied script")
	}
}

func newBlueprintTxContext(t *testing.T) (*fixed.FixedChainContext, common.Address) {
	t.Helper()
	pp := backend.ProtocolParameters{
		MinFeeConstant:      155381,
		MinFeeCoefficient:   44,
		MaxTxSize:           16384,
		CoinsPerUtxoByte:    "4310",
		CollateralPercent:   150,
		MaxCollateralInputs: 3,
		MaxValSize:          "5000",
		PriceMem:            0.0577,
		PriceStep:           0.0000721,
		MaxTxExMem:          "14000000",
		MaxTxExSteps:        "10000000000",
		KeyDeposits:         "2000000",
		PoolDeposits:        "500000000",
		CostModels: map[string][]int64{
			"PlutusV3": {100, 100, 100},
		},
	}
	cc := fixed.NewFixedChainContext(pp, backend.GenesisParameters{NetworkMagic: 1}, 0)
	addr, err := common.NewAddressFromParts(
		common.AddressTypeKeyKey,
		common.AddressNetworkTestnet,
		bytes.Repeat([]byte{0x01}, 28),
		bytes.Repeat([]byte{0x02}, 28),
	)
	if err != nil {
		t.Fatal(err)
	}
	var feeTx, collTx common.Blake2b256
	feeTx[0] = 0x10
	collTx[0] = 0x20
	feeOut := babbage.BabbageTransactionOutput{
		OutputAddress: addr,
		OutputAmount:  mary.MaryTransactionOutputValue{Amount: 50_000_000},
	}
	collOut := babbage.BabbageTransactionOutput{
		OutputAddress: addr,
		OutputAmount:  mary.MaryTransactionOutputValue{Amount: 10_000_000},
	}
	cc.AddUtxo(addr, common.Utxo{
		Id:     shelley.ShelleyTransactionInput{TxId: feeTx, OutputIndex: 0},
		Output: &feeOut,
	})
	cc.AddUtxo(addr, common.Utxo{
		Id:     shelley.ShelleyTransactionInput{TxId: collTx, OutputIndex: 0},
		Output: &collOut,
	})
	return cc, addr
}

func makeScriptUtxo(t *testing.T, scriptAddr common.Address, datum data.PlutusData, lovelace uint64) common.Utxo {
	t.Helper()
	var txHash common.Blake2b256
	txHash[0] = 0x30
	opt, err := apollo.NewDatumOptionInline(&common.Datum{Data: datum})
	if err != nil {
		t.Fatal(err)
	}
	out := babbage.BabbageTransactionOutput{
		OutputAddress: scriptAddr,
		OutputAmount:  mary.MaryTransactionOutputValue{Amount: lovelace},
		DatumOption:   opt,
	}
	return common.Utxo{
		Id:     shelley.ShelleyTransactionInput{TxId: txHash, OutputIndex: 0},
		Output: &out,
	}
}

package apollo

import (
	"math/big"
	"strings"
	"testing"

	"github.com/Salvionied/apollo/v2/backend/fixed"
	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/conway"
	"github.com/blinklabs-io/gouroboros/ledger/dijkstra"
	"github.com/blinklabs-io/plutigo/data"
)

var guardTestScript = common.PlutusV3Script{0x01, 0x02, 0x03}

func guardScriptCredential() common.Credential {
	return common.Credential{
		CredType:   common.CredentialTypeScriptHash,
		Credential: common.CredentialHash(guardTestScript.Hash()),
	}
}

func guardKeyCredential(b byte) common.Credential {
	var hash common.Blake2b224
	hash[0] = b
	return common.Credential{CredType: common.CredentialTypeAddrKeyHash, Credential: hash}
}

func guardRedeemer() *common.Datum {
	return &common.Datum{Data: data.NewInteger(big.NewInt(42))}
}

// guardTestBuilder returns a Dijkstra builder whose chain context evaluates
// redeemers with result, or reports evaluation as unsupported when result is nil.
func guardTestBuilder(t *testing.T, result map[common.RedeemerKey]common.ExUnits) (*Apollo, *capturingEvalContext) {
	t.Helper()
	base := dijkstraTestContext(t, dijkstra.MinProtocolVersionDijkstra)
	cc := &capturingEvalContext{FixedChainContext: base, result: result}
	addr := testAddress(t)
	addTestUtxo(base, addr, 20_000_000, 0x01, 0)
	addTestUtxo(base, addr, 20_000_000, 0x02, 0)
	payment, err := NewPayment(validTestAddrBech32, 2_000_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	return New(cc).
		SetWallet(NewExternalWallet(addr)).
		SetTargetEra(EraDijkstra).
		AddPayment(payment), cc
}

// decodeCompletedDijkstraTx round-trips the builder's transaction through the
// gouroboros Dijkstra decoder, which enforces the guard wire rules.
func decodeCompletedDijkstraTx(t *testing.T, a *Apollo) *dijkstra.DijkstraTransaction {
	t.Helper()
	txCbor, err := a.GetTxCbor()
	if err != nil {
		t.Fatal(err)
	}
	tx, err := dijkstra.NewDijkstraTransactionFromCbor(txCbor)
	if err != nil {
		t.Fatalf("decode Dijkstra transaction: %v", err)
	}
	return tx
}

func TestAddGuardPlutusScriptGuard(t *testing.T) {
	signer := guardKeyCredential(0x11)
	a, _ := guardTestBuilder(t, nil)
	exUnits := common.ExUnits{Memory: 1000, Steps: 2000}
	a, err := a.
		DisableExecutionUnitsEstimation().
		AttachScript(guardTestScript).
		AddRequiredSigner(signer.Credential).
		AddGuard(guardScriptCredential(), guardRedeemer(), &exUnits).
		Complete()
	if err != nil {
		t.Fatal(err)
	}
	tx := decodeCompletedDijkstraTx(t, a)
	guards := tx.GuardingCredentials()
	if len(guards) != 2 {
		t.Fatalf("guards = %d, want 2", len(guards))
	}
	if credentialKey(guards[0]) != credentialKey(signer) || credentialKey(guards[1]) != credentialKey(guardScriptCredential()) {
		t.Fatalf("guard order = %s, %s; want required signer then script guard", credentialKey(guards[0]), credentialKey(guards[1]))
	}
	redeemer, ok := tx.WitnessSet.WsRedeemers.Redeemers[common.RedeemerKey{Tag: common.RedeemerTagGuarding, Index: 1}]
	if !ok {
		t.Fatalf("missing guarding redeemer at index 1: %v", tx.WitnessSet.WsRedeemers.Redeemers)
	}
	if redeemer.ExUnits != exUnits {
		t.Fatalf("guarding redeemer ExUnits = %+v, want %+v", redeemer.ExUnits, exUnits)
	}
	if tx.Body.TxScriptDataHash == nil {
		t.Fatal("script data hash must be set for a guarding redeemer")
	}
	if len(tx.Body.TxCollateral.Items()) == 0 {
		t.Fatal("collateral must be selected for a Plutus guard")
	}
}

func TestAddGuardEvaluatesGuardingRedeemer(t *testing.T) {
	key := common.RedeemerKey{Tag: common.RedeemerTagGuarding, Index: 0}
	a, cc := guardTestBuilder(t, map[common.RedeemerKey]common.ExUnits{key: {Memory: 5000, Steps: 7000}})
	a, err := a.
		AttachScript(guardTestScript).
		AddGuard(guardScriptCredential(), guardRedeemer(), nil).
		Complete()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := dijkstra.NewDijkstraTransactionFromCbor(cc.lastTxCbor); err != nil {
		t.Fatalf("evaluation request is not a Dijkstra transaction: %v", err)
	}
	redeemer := decodeCompletedDijkstraTx(t, a).WitnessSet.WsRedeemers.Redeemers[key]
	if redeemer.ExUnits.Memory < 5000 || redeemer.ExUnits.Steps < 7000 {
		t.Fatalf("guarding redeemer ExUnits = %+v, want the evaluated budget", redeemer.ExUnits)
	}
}

func TestAddGuardRejectsMissingEvaluationResult(t *testing.T) {
	// UTxO RPC has no guarding purpose, so its evaluator cannot return one.
	other := common.RedeemerKey{Tag: common.RedeemerTagSpend, Index: 9}
	a, _ := guardTestBuilder(t, map[common.RedeemerKey]common.ExUnits{other: {Memory: 1, Steps: 1}})
	_, err := a.
		AttachScript(guardTestScript).
		AddGuard(guardScriptCredential(), guardRedeemer(), nil).
		Complete()
	if err == nil {
		t.Fatal("Complete succeeded without a guarding redeemer budget")
	}

	a, _ = guardTestBuilder(t, map[common.RedeemerKey]common.ExUnits{
		{Tag: common.RedeemerTagGuarding, Index: 3}: {Memory: 1, Steps: 1},
	})
	_, err = a.
		AttachScript(guardTestScript).
		AddGuard(guardScriptCredential(), guardRedeemer(), nil).
		Complete()
	if err == nil || !strings.Contains(err.Error(), "guarding redeemer index 3 out of range") {
		t.Fatalf("Complete error = %v, want out-of-range guarding index", err)
	}
}

func TestAddGuardKeyHashGuard(t *testing.T) {
	guard := guardKeyCredential(0x22)
	a, _ := guardTestBuilder(t, nil)
	withoutGuard, err := a.Clone().Complete()
	if err != nil {
		t.Fatal(err)
	}
	a, err = a.AddGuard(guard, nil, nil).Complete()
	if err != nil {
		t.Fatal(err)
	}
	tx := decodeCompletedDijkstraTx(t, a)
	if guards := tx.GuardingCredentials(); len(guards) != 1 || credentialKey(guards[0]) != credentialKey(guard) {
		t.Fatalf("guards = %v, want the key-hash guard", guards)
	}
	if len(tx.WitnessSet.WsRedeemers.Redeemers) != 0 {
		t.Fatal("a key-hash guard must not add a redeemer")
	}
	if a.GetDijkstraTx().Body.TxFee <= withoutGuard.GetDijkstraTx().Body.TxFee {
		t.Fatal("fee must account for the key-hash guard's signature")
	}
}

func TestDijkstraGuardsDeduplicateRequiredSigners(t *testing.T) {
	signer := guardKeyCredential(0x33)
	a, _ := guardTestBuilder(t, nil)
	a, err := a.
		AddRequiredSigner(signer.Credential).
		AddRequiredSigner(signer.Credential).
		Complete()
	if err != nil {
		t.Fatal(err)
	}
	if got := decodeCompletedDijkstraTx(t, a).Body.TxGuards.KeyHashes; len(got) != 1 {
		t.Fatalf("key-hash guards = %v, want one deduplicated signer", got)
	}
}

func TestAddRequiredTopLevelGuard(t *testing.T) {
	a, _ := guardTestBuilder(t, nil)
	guard := guardKeyCredential(0x44)
	a, err := a.
		AddGuard(guard, nil, nil).
		AddRequiredTopLevelGuard(guard, nil).
		Complete()
	if err != nil {
		t.Fatal(err)
	}
	required := decodeCompletedDijkstraTx(t, a).Body.TxRequiredTopLevelGuards
	if len(required) != 1 {
		t.Fatalf("required top-level guards = %d, want 1", len(required))
	}
	for cred, datum := range required {
		if credentialKey(*cred) != credentialKey(guard) || datum != nil {
			t.Fatalf("required top-level guard = %s with datum %v", credentialKey(*cred), datum)
		}
	}
}

func TestAddRequiredTopLevelGuardAcceptsRequiredSigner(t *testing.T) {
	signer := guardKeyCredential(0x45)
	a, _ := guardTestBuilder(t, nil)
	a, err := a.
		AddRequiredSigner(signer.Credential).
		AddRequiredTopLevelGuard(signer, nil).
		Complete()
	if err != nil {
		t.Fatal(err)
	}
	if got := decodeCompletedDijkstraTx(t, a).Body.TxRequiredTopLevelGuards; len(got) != 1 {
		t.Fatalf("required top-level guards = %d, want 1", len(got))
	}
}

func TestAddRequiredTopLevelGuardRequiresGuard(t *testing.T) {
	a, _ := guardTestBuilder(t, nil)
	_, err := a.AddRequiredTopLevelGuard(guardKeyCredential(0x55), nil).Complete()
	if err == nil || !strings.Contains(err.Error(), "not among the transaction guards") {
		t.Fatalf("Complete error = %v, want missing guard error", err)
	}
}

func TestGuardBuilderRejectsInvalidInput(t *testing.T) {
	exUnits := &common.ExUnits{Memory: 1, Steps: 1}
	tests := []struct {
		name  string
		build func(*Apollo) *Apollo
		want  string
	}{
		{"conway guard", func(a *Apollo) *Apollo {
			return a.SetTargetEra(EraConway).AddGuard(guardKeyCredential(1), nil, nil)
		}, "require the Dijkstra target era"},
		{"conway required guard", func(a *Apollo) *Apollo {
			return a.SetTargetEra(EraConway).AddRequiredTopLevelGuard(guardKeyCredential(1), nil)
		}, "require the Dijkstra target era"},
		{"key guard redeemer", func(a *Apollo) *Apollo {
			return a.AddGuard(guardKeyCredential(1), guardRedeemer(), nil)
		}, "must not have a redeemer"},
		{"exunits without redeemer", func(a *Apollo) *Apollo {
			return a.AddGuard(guardScriptCredential(), nil, exUnits)
		}, "require a redeemer"},
		{"unknown credential type", func(a *Apollo) *Apollo {
			return a.AddGuard(common.Credential{CredType: 7}, nil, nil)
		}, "unsupported guard credential type"},
		{"conflicting redeemer", func(a *Apollo) *Apollo {
			other := &common.Datum{Data: data.NewByteString([]byte{1})}
			return a.AddGuard(guardScriptCredential(), guardRedeemer(), nil).
				AddGuard(guardScriptCredential(), other, nil)
		}, "conflicting redeemer"},
		{"redeemer removed", func(a *Apollo) *Apollo {
			return a.AddGuard(guardScriptCredential(), guardRedeemer(), nil).
				AddGuard(guardScriptCredential(), nil, nil)
		}, "conflicting redeemer"},
		{"key required guard datum", func(a *Apollo) *Apollo {
			return a.AddRequiredTopLevelGuard(guardKeyCredential(1), guardRedeemer())
		}, "must not have a datum"},
		{"required guard empty datum", func(a *Apollo) *Apollo {
			return a.AddRequiredTopLevelGuard(guardScriptCredential(), &common.Datum{})
		}, "missing Plutus data"},
		{"duplicate required guard", func(a *Apollo) *Apollo {
			return a.AddRequiredTopLevelGuard(guardKeyCredential(1), nil).
				AddRequiredTopLevelGuard(guardKeyCredential(1), nil)
		}, "duplicate required top-level guard"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := New(dijkstraTestContext(t, dijkstra.MinProtocolVersionDijkstra))
			if tt.name != "conway guard" && tt.name != "conway required guard" {
				a.SetTargetEra(EraDijkstra)
			}
			_, err := tt.build(a).Complete()
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestAddGuardRepeatedIdenticalGuardIsIdempotent(t *testing.T) {
	a, _ := guardTestBuilder(t, nil)
	exUnits := common.ExUnits{Memory: 1000, Steps: 2000}
	a, err := a.
		DisableExecutionUnitsEstimation().
		AttachScript(guardTestScript).
		AddGuard(guardScriptCredential(), guardRedeemer(), &exUnits).
		AddGuard(guardScriptCredential(), guardRedeemer(), &exUnits).
		Complete()
	if err != nil {
		t.Fatal(err)
	}
	if guards := decodeCompletedDijkstraTx(t, a).GuardingCredentials(); len(guards) != 1 {
		t.Fatalf("guards = %d, want 1", len(guards))
	}
}

func TestCloneCopiesGuards(t *testing.T) {
	a, _ := guardTestBuilder(t, nil)
	datum := guardRedeemer()
	a.AddGuard(guardScriptCredential(), guardRedeemer(), nil).
		AddRequiredTopLevelGuard(guardScriptCredential(), datum)
	clone := a.Clone()
	a.AddGuard(guardKeyCredential(0x66), nil, nil)
	if len(clone.guards) != 1 || len(clone.guardRedeemers) != 1 || len(clone.requiredTopLevelGuards) != 1 {
		t.Fatalf("clone guards = %d, redeemers = %d, required = %d; want 1 each",
			len(clone.guards), len(clone.guardRedeemers), len(clone.requiredTopLevelGuards))
	}
	if clone.requiredTopLevelGuards[0].Datum == datum {
		t.Fatal("clone must not share the required guard datum")
	}
	encoded, err := cbor.Encode(clone.requiredTopLevelGuards[0].Datum)
	if err != nil || len(encoded) == 0 {
		t.Fatalf("cloned datum does not encode: %v", err)
	}
}

// guardLedgerState resolves UTxOs from the fixed chain context for ledger
// rules. Methods outside the overridden set panic through the nil embedded
// interface, so a rule that needs more state fails the test loudly.
type guardLedgerState struct {
	common.LedgerState
	cc *capturingEvalContext
}

func (s guardLedgerState) UtxoById(input common.TransactionInput) (common.Utxo, error) {
	utxo, err := s.cc.UtxoByRef(input.Id(), input.Index())
	if err != nil {
		return common.Utxo{}, err
	}
	return *utxo, nil
}

func (s guardLedgerState) NetworkId() uint { return uint(s.cc.NetworkId()) }

func (s guardLedgerState) CostModels() map[common.PlutusLanguage]common.CostModel {
	return nil
}

func TestAddGuardSatisfiesDijkstraWitnessRules(t *testing.T) {
	costModel := make([]int64, 297)
	for i := range costModel {
		costModel[i] = int64(i + 1)
	}
	base := dijkstraTestContext(t, dijkstra.MinProtocolVersionDijkstra)
	pp, err := base.ProtocolParams()
	if err != nil {
		t.Fatal(err)
	}
	pp.CostModels = map[string][]int64{"PlutusV3": costModel}
	gp, err := base.GenesisParams()
	if err != nil {
		t.Fatal(err)
	}
	cc := &capturingEvalContext{FixedChainContext: fixed.NewFixedChainContext(pp, gp, base.NetworkId())}
	addr := testAddress(t)
	addTestUtxo(cc.FixedChainContext, addr, 20_000_000, 0x01, 0)
	addTestUtxo(cc.FixedChainContext, addr, 20_000_000, 0x02, 0)
	payment, err := NewPayment(validTestAddrBech32, 2_000_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	exUnits := common.ExUnits{Memory: 1000, Steps: 2000}
	a, err := New(cc).
		SetWallet(NewExternalWallet(addr)).
		SetTargetEra(EraDijkstra).
		AddPayment(payment).
		DisableExecutionUnitsEstimation().
		AttachScript(guardTestScript).
		AddRequiredSigner(guardKeyCredential(0x11).Credential).
		AddGuard(guardScriptCredential(), guardRedeemer(), &exUnits).
		Complete()
	if err != nil {
		t.Fatal(err)
	}
	tx := decodeCompletedDijkstraTx(t, a)
	ls := guardLedgerState{cc: cc}
	ledgerPparams := &dijkstra.DijkstraProtocolParameters{
		ConwayProtocolParameters: conway.ConwayProtocolParameters{
			CostModels: map[uint][]int64{2: costModel},
		},
	}
	rules := map[string]func(common.Transaction, uint64, common.LedgerState, common.ProtocolParameters) error{
		"redeemer and script witnesses": dijkstra.UtxoValidateRedeemerAndScriptWitnesses,
		"extraneous redeemers":          dijkstra.UtxoValidateExtraneousRedeemers,
		"script data hash":              dijkstra.UtxoValidateScriptDataHash,
	}
	for name, rule := range rules {
		if err := rule(tx, 0, ls, ledgerPparams); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}

package apollo

import (
	"encoding/hex"
	"errors"
	"math/big"
	"testing"

	"github.com/Salvionied/apollo/v2/backend/fixed"
	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/babbage"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/conway"
	"github.com/blinklabs-io/gouroboros/ledger/dijkstra"
	"github.com/blinklabs-io/gouroboros/ledger/mary"
	"github.com/blinklabs-io/gouroboros/ledger/shelley"
)

var plutusV4TestScript = common.PlutusV4Script{0x04, 0x05, 0x06, 0x07}

// TestDijkstraPlutusV4SpendAndGuardSatisfyLedgerRules builds a Dijkstra
// transaction that spends a V4-locked UTxO and is guarded by the same V4
// script, supplied through a reference input, then checks it against the
// gouroboros Dijkstra witness, cost-model, script-integrity, and fee rules.
func TestDijkstraPlutusV4SpendAndGuardSatisfyLedgerRules(t *testing.T) {
	costModel := make([]int64, 300)
	for i := range costModel {
		costModel[i] = int64(i + 7)
	}
	base := dijkstraTestContext(t, dijkstra.MinProtocolVersionDijkstra)
	pp, err := base.ProtocolParams()
	if err != nil {
		t.Fatal(err)
	}
	pp.CostModels = map[string][]int64{"PlutusV4": costModel}
	withDijkstraRefScriptTiers(&pp)
	gp, err := base.GenesisParams()
	if err != nil {
		t.Fatal(err)
	}
	cc := &capturingEvalContext{FixedChainContext: fixed.NewFixedChainContext(pp, gp, base.NetworkId())}
	addr := testAddress(t)
	addTestUtxo(cc.FixedChainContext, addr, 20_000_000, 0x01, 0)
	addTestUtxo(cc.FixedChainContext, addr, 20_000_000, 0x02, 0)

	scriptHash := plutusV4TestScript.Hash()
	scriptAddr, err := common.NewAddressFromParts(
		common.AddressTypeScriptNone,
		common.AddressNetworkTestnet,
		scriptHash.Bytes(),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	var refTxHash common.Blake2b256
	refTxHash[0] = 0xf4
	cc.AddUtxoByRef(common.Utxo{
		Id: shelley.ShelleyTransactionInput{TxId: refTxHash, OutputIndex: 0},
		Output: &babbage.BabbageTransactionOutput{
			OutputAddress: addr,
			OutputAmount:  mary.MaryTransactionOutputValue{Amount: 5_000_000},
			TxOutScriptRef: &common.ScriptRef{
				Type:   common.ScriptRefTypePlutusV4,
				Script: plutusV4TestScript,
			},
		},
	})
	datumOption, err := NewDatumOptionInline(guardRedeemer())
	if err != nil {
		t.Fatal(err)
	}
	var lockedTxHash common.Blake2b256
	lockedTxHash[0] = 0xf5
	locked := common.Utxo{
		Id: shelley.ShelleyTransactionInput{TxId: lockedTxHash, OutputIndex: 0},
		Output: &babbage.BabbageTransactionOutput{
			OutputAddress: scriptAddr,
			OutputAmount:  mary.MaryTransactionOutputValue{Amount: 5_000_000},
			DatumOption:   datumOption,
		},
	}
	cc.AddUtxoByRef(locked)

	payment, err := NewPayment(validTestAddrBech32, 2_000_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	exUnits := common.ExUnits{Memory: 10_000, Steps: 20_000}
	scriptCred := common.Credential{
		CredType:   common.CredentialTypeScriptHash,
		Credential: common.CredentialHash(scriptHash),
	}
	a, err := newTestApollo(cc).
		SetWallet(NewExternalWallet(addr)).
		SetTargetEra(EraDijkstra).
		DisableExecutionUnitsEstimation().
		AddPayment(payment).
		CollectFrom(locked, *guardRedeemer(), exUnits).
		AddGuard(scriptCred, guardRedeemer(), &exUnits).
		AddReferenceInput(hex.EncodeToString(refTxHash.Bytes()), 0)
	if err != nil {
		t.Fatal(err)
	}
	a, err = a.Complete()
	if err != nil {
		t.Fatal(err)
	}
	tx := decodeCompletedDijkstraTx(t, a)
	if len(tx.WitnessSet.PlutusV4Scripts()) != 0 {
		t.Fatal("a V4 script must not be placed in the witness set")
	}
	if _, ok := tx.WitnessSet.WsRedeemers.Redeemers[common.RedeemerKey{Tag: common.RedeemerTagGuarding, Index: 0}]; !ok {
		t.Fatal("missing V4 guarding redeemer")
	}

	ls := guardLedgerState{cc: cc}
	ledgerPparams := &dijkstra.DijkstraProtocolParameters{
		ConwayProtocolParameters: conway.ConwayProtocolParameters{
			MinFeeA:                    44,
			MinFeeB:                    155381,
			MinFeeRefScriptCostPerByte: &cbor.Rat{Rat: big.NewRat(15, 1)},
			ExecutionCosts: common.ExUnitPrice{
				MemPrice:  &cbor.Rat{Rat: big.NewRat(577, 10_000)},
				StepPrice: &cbor.Rat{Rat: big.NewRat(721, 10_000_000)},
			},
			CostModels: map[uint][]int64{3: costModel},
		},
		RefScriptCostStride:     1000,
		RefScriptCostMultiplier: &cbor.Rat{Rat: big.NewRat(2, 1)},
	}
	rules := map[string]func(common.Transaction, uint64, common.LedgerState, common.ProtocolParameters) error{
		"redeemer and script witnesses": dijkstra.UtxoValidateRedeemerAndScriptWitnesses,
		"extraneous redeemers":          dijkstra.UtxoValidateExtraneousRedeemers,
		"cost models present":           dijkstra.UtxoValidateCostModelsPresent,
		"script data hash":              dijkstra.UtxoValidateScriptDataHash,
		"fee too small":                 dijkstra.UtxoValidateFeeTooSmallUtxo,
	}
	for name, rule := range rules {
		if err := rule(tx, 0, ls, ledgerPparams); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestAttachScriptV4InDijkstraRequiresReferenceInput(t *testing.T) {
	a := newTestApollo(dijkstraTestContext(t, dijkstra.MinProtocolVersionDijkstra)).
		SetTargetEra(EraDijkstra).
		AttachScript(plutusV4TestScript)
	if _, err := a.Complete(); !errors.Is(err, ErrPlutusV4WitnessScript) {
		t.Fatalf("Complete error = %v, want %v", err, ErrPlutusV4WitnessScript)
	}
}

package apollo

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"math/big"
	"strings"
	"testing"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/babbage"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/dijkstra"
	"github.com/blinklabs-io/gouroboros/ledger/mary"
	"github.com/blinklabs-io/gouroboros/ledger/shelley"
	plutigoData "github.com/blinklabs-io/plutigo/data"

	"github.com/Salvionied/apollo/v2/backend/fixed"
)

func explain(t *testing.T, a *Apollo) *BuildReport {
	t.Helper()
	report, err := a.Explain()
	if err != nil {
		t.Fatal(err)
	}
	return report
}

// assertReportMatchesTx checks the report against the transaction Complete
// built: hash, inputs, outputs, fee, and redeemer indexes.
func assertReportMatchesTx(t *testing.T, a *Apollo, report *BuildReport) {
	t.Helper()
	tx := a.GetTx()
	bodyBytes, err := cbor.Encode(&tx.Body)
	if err != nil {
		t.Fatal(err)
	}
	if want := hex.EncodeToString(common.Blake2b256Hash(bodyBytes).Bytes()); report.TxHash != want {
		t.Errorf("TxHash = %s, want %s", report.TxHash, want)
	}
	txBytes, err := cbor.Encode(tx)
	if err != nil {
		t.Fatal(err)
	}
	if report.TxSize != len(txBytes) {
		t.Errorf("TxSize = %d, want %d", report.TxSize, len(txBytes))
	}
	inputs := tx.Body.Inputs()
	if len(report.Inputs.Final) != len(inputs) {
		t.Fatalf("%d final inputs, transaction has %d", len(report.Inputs.Final), len(inputs))
	}
	for i, input := range inputs {
		if report.Inputs.Final[i].Ref != input.String() {
			t.Errorf("final input %d = %s, transaction has %s", i, report.Inputs.Final[i].Ref, input.String())
		}
	}
	if len(report.Outputs.Final) != len(tx.Body.TxOutputs) {
		t.Fatalf("%d final outputs, transaction has %d", len(report.Outputs.Final), len(tx.Body.TxOutputs))
	}
	for i, output := range tx.Body.TxOutputs {
		got := report.Outputs.Final[i]
		if got.Address != output.OutputAddress.String() || got.Value.Coin != output.OutputAmount.Amount {
			t.Errorf("final output %d = %s %d, transaction has %s %d",
				i, got.Address, got.Value.Coin, output.OutputAddress.String(), output.OutputAmount.Amount)
		}
	}
	if report.Fees.Final != int64(tx.Body.TxFee) || report.Balance.Fee != report.Fees.Final { //nolint:gosec // test fees are small
		t.Errorf("fee = %d / %d, transaction has %d", report.Fees.Final, report.Balance.Fee, tx.Body.TxFee)
	}
	if report.Fees.Final != report.Fees.Requested+report.Fees.AbsorbedChange {
		t.Errorf("final fee %d != requested %d + absorbed %d",
			report.Fees.Final, report.Fees.Requested, report.Fees.AbsorbedChange)
	}
	if !report.Fees.Forced {
		parts := report.Fees.SizeFee + report.Fees.Padding + report.Fees.ExecutionUnitFee + report.Fees.ReferenceScriptFee
		if parts != report.Fees.Requested {
			t.Errorf("fee parts sum to %d, requested fee is %d", parts, report.Fees.Requested)
		}
	}
	var witnessRedeemers int
	for key, value := range tx.WitnessSet.WsRedeemers.Iter() {
		witnessRedeemers++
		found := false
		for _, r := range report.Redeemers {
			if r.Tag == key.Tag && r.Index == key.Index && r.ExUnits == value.ExUnits {
				found = true
			}
		}
		if !found {
			t.Errorf("redeemer %v/%d is not in the report", key.Tag, key.Index)
		}
	}
	if len(report.Redeemers) != witnessRedeemers {
		t.Errorf("%d reported redeemers, transaction has %d", len(report.Redeemers), witnessRedeemers)
	}
}

func TestExplainRequiresBuiltTransaction(t *testing.T) {
	cc := setupFixedContext()
	if _, err := newTestApollo(cc).Explain(); err == nil {
		t.Fatal("expected Explain to fail before Complete")
	}

	addr := testAddress(t)
	addTestUtxo(cc, addr, 10_000_000, 0x01, 0)
	p, err := NewPayment(validTestAddrBech32, 2_000_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	failed, err := newTestApollo(cc).SetWallet(NewExternalWallet(addr)).AddPayment(p).
		SetMaxFee(1).SetTtl(50000000).Complete()
	if err == nil {
		t.Fatal("expected Complete to fail under a 1-lovelace fee limit")
	}
	if _, err := failed.Explain(); err == nil {
		t.Fatal("expected Explain to fail after a failed Complete")
	}

	a, err := newTestApollo(cc).SetWallet(NewExternalWallet(addr)).AddPayment(p).SetTtl(50000000).Complete()
	if err != nil {
		t.Fatal(err)
	}
	txCbor, err := a.GetTxCbor()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.LoadTxCbor(hex.EncodeToString(txCbor)); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Explain(); err == nil {
		t.Fatal("expected Explain to fail once a transaction is loaded from CBOR")
	}
}

func TestBuildReportSimpleTransfer(t *testing.T) {
	cc := setupFixedContext()
	addr := testAddress(t)
	addTestUtxo(cc, addr, 10_000_000, 0x01, 0)

	p, err := NewPayment(validTestAddrBech32, 2_000_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	a, err := newTestApollo(cc).
		SetWallet(NewExternalWallet(addr)).
		AddPayment(p).
		SetFeePadding(5_000).
		SetTtl(50000000).
		Complete()
	if err != nil {
		t.Fatal(err)
	}

	report := explain(t, a)
	assertReportMatchesTx(t, a, report)
	if report.Era != EraConway {
		t.Errorf("Era = %v, want Conway", report.Era)
	}
	if report.CoinSelector != defaultCoinSelector.Name() {
		t.Errorf("CoinSelector = %q, want %q", report.CoinSelector, defaultCoinSelector.Name())
	}
	if report.Fees.Padding != 5_000 || report.Fees.SizeFee == 0 || report.Fees.Forced {
		t.Errorf("fees = %+v, want padding 5000 and a size fee", report.Fees)
	}
	if report.Inputs.AvailableUtxoCount != 1 || len(report.Inputs.Preselected) != 0 || len(report.Inputs.Selected) != 1 {
		t.Errorf("inputs = %+v, want one available and one selected", report.Inputs)
	}
	if report.Balance.TotalInput.Coin != 10_000_000 {
		t.Errorf("TotalInput = %d, want 10000000", report.Balance.TotalInput.Coin)
	}
	if report.Outputs.ChangeIndex != 1 || !report.Outputs.Final[1].IsChange || report.Outputs.Final[0].IsChange {
		t.Fatalf("ChangeIndex = %d, want the change output at 1", report.Outputs.ChangeIndex)
	}
	change := report.Outputs.Final[1].Value.Coin
	if report.Outputs.Change.Coin != change || report.Balance.Change.Coin != change {
		t.Errorf("change = %d / %d, want %d", report.Outputs.Change.Coin, report.Balance.Change.Coin, change)
	}
	if report.Balance.TotalOutput.Coin != 2_000_000+change {
		t.Errorf("TotalOutput = %d, want %d", report.Balance.TotalOutput.Coin, 2_000_000+change)
	}
	if report.Balance.TotalInput.Coin != report.Balance.TotalOutput.Coin+uint64(report.Fees.Final) { //nolint:gosec // non-negative fee
		t.Errorf("inputs %d != outputs %d + fee %d", report.Balance.TotalInput.Coin, report.Balance.TotalOutput.Coin, report.Fees.Final)
	}
}

func TestBuildReportIsACopy(t *testing.T) {
	cc := setupFixedContext()
	addr := testAddress(t)
	addTestUtxo(cc, addr, 10_000_000, 0x01, 0)
	p, err := NewPayment(validTestAddrBech32, 2_000_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	a, err := newTestApollo(cc).SetWallet(NewExternalWallet(addr)).AddPayment(p).SetTtl(50000000).Complete()
	if err != nil {
		t.Fatal(err)
	}

	report := explain(t, a)
	want := report.Outputs.Change.Coin
	report.Inputs.Final[0].Value.Coin = 1
	report.Outputs.Final[0].Value.Coin = 1
	report.Outputs.Requested[0].Coin = 1
	report.Outputs.Change.Coin = 1
	report.Inputs.InputAddresses = append(report.Inputs.InputAddresses[:0], "x")
	again := explain(t, a)
	if again.Inputs.Final[0].Value.Coin == 1 || again.Outputs.Final[0].Value.Coin == 1 ||
		again.Outputs.Requested[0].Coin == 1 || again.Outputs.Change.Coin != want {
		t.Fatal("changing a returned report changed the builder's report")
	}

	clone := a.Clone()
	clone.buildReport.Outputs.Change.Coin = 1
	if explain(t, a).Outputs.Change.Coin != want {
		t.Fatal("Clone shares build report state with the original")
	}
}

func TestBuildReportPreselectedAndSelectedInputs(t *testing.T) {
	cc := setupFixedContext()
	addr := testAddress(t)
	var preselectedHash, selectedHash common.Blake2b256
	preselectedHash[0] = 0x01
	selectedHash[0] = 0x02
	preselected := makeTestUtxo(t, preselectedHash, 0, 3_000_000)
	selected := makeTestUtxo(t, selectedHash, 0, 10_000_000)

	p, err := NewPayment(validTestAddrBech32, 8_000_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	a, err := newTestApollo(cc).
		SetWallet(NewExternalWallet(addr)).
		AddInput(preselected).
		AddLoadedUTxOs(selected).
		AddPayment(p).
		SetTtl(50000000).
		Complete()
	if err != nil {
		t.Fatal(err)
	}

	report := explain(t, a)
	assertReportMatchesTx(t, a, report)
	if len(report.Inputs.Preselected) != 1 || report.Inputs.Preselected[0].Ref != utxoRef(preselected) {
		t.Errorf("Preselected = %+v, want %s", report.Inputs.Preselected, utxoRef(preselected))
	}
	if len(report.Inputs.Selected) != 1 || report.Inputs.Selected[0].Ref != utxoRef(selected) {
		t.Errorf("Selected = %+v, want %s", report.Inputs.Selected, utxoRef(selected))
	}
	if report.Inputs.SelectionTarget.Coin <= 8_000_000 {
		t.Errorf("SelectionTarget = %d, want the payment plus a fee reserve", report.Inputs.SelectionTarget.Coin)
	}
	if report.Fees.SelectionReserve <= 0 || report.Fees.SelectionReserve > report.Fees.Preliminary {
		t.Errorf("SelectionReserve = %d, want within (0, %d]", report.Fees.SelectionReserve, report.Fees.Preliminary)
	}
}

func TestBuildReportMinUtxoIncrease(t *testing.T) {
	cc := setupFixedContext()
	addr := testAddress(t)
	addTestUtxo(cc, addr, 10_000_000, 0x01, 0)

	p, err := NewPayment(validTestAddrBech32, 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	a, err := newTestApollo(cc).
		SetWallet(NewExternalWallet(addr)).
		AddPayment(p).
		SetTtl(50000000).
		Complete()
	if err != nil {
		t.Fatal(err)
	}

	report := explain(t, a)
	assertReportMatchesTx(t, a, report)
	if len(report.Outputs.Requested) != 1 || report.Outputs.Requested[0].Coin != 1 {
		t.Fatalf("Requested = %+v, want one payment of 1 lovelace", report.Outputs.Requested)
	}
	base := report.Outputs.Base[0]
	if base.Value.Coin < uint64(base.MinLovelace) || base.MinUtxoIncrease != base.Value.Coin-1 { //nolint:gosec // positive min lovelace
		t.Errorf("base output = %+v, want raised to its min UTxO with the increase recorded", base)
	}
	if report.Outputs.Final[0].MinUtxoIncrease != base.MinUtxoIncrease {
		t.Errorf("final MinUtxoIncrease = %d, want %d", report.Outputs.Final[0].MinUtxoIncrease, base.MinUtxoIncrease)
	}
	if change := report.Outputs.Final[report.Outputs.ChangeIndex]; change.MinUtxoIncrease != 0 {
		t.Errorf("change MinUtxoIncrease = %d, want 0", change.MinUtxoIncrease)
	}
}

// ADA-only change below the minimum UTxO is added to the fee and no change
// output is written; the report must say so rather than show a change
// output or a fee it did not request.
func TestBuildReportAbsorbedChange(t *testing.T) {
	cc := setupFixedContext()
	addr := testAddress(t)
	addTestUtxo(cc, addr, 2_250_000, 0x01, 0)

	p, err := NewPayment(validTestAddrBech32, 2_000_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	a, err := newTestApollo(cc).SetWallet(NewExternalWallet(addr)).AddPayment(p).SetTtl(50000000).Complete()
	if err != nil {
		t.Fatal(err)
	}

	report := explain(t, a)
	assertReportMatchesTx(t, a, report)
	if report.Outputs.ChangeIndex != -1 || len(report.Outputs.Final) != 1 {
		t.Fatalf("ChangeIndex = %d with %d outputs, want no change output", report.Outputs.ChangeIndex, len(report.Outputs.Final))
	}
	if report.Fees.AbsorbedChange <= 0 {
		t.Fatalf("AbsorbedChange = %d, want the dust change", report.Fees.AbsorbedChange)
	}
	if report.Fees.Final != 250_000 {
		t.Errorf("final fee = %d, want the whole 250000 surplus", report.Fees.Final)
	}
}

func TestBuildReportScriptCollateralAndRedeemer(t *testing.T) {
	cc := setupFixedContext()
	addr := testAddress(t)
	addTestUtxo(cc, addr, 30_000_000, 0x01, 0)
	addTestUtxo(cc, addr, 20_000_000, 0x02, 0)

	policy := "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4"
	datum := common.Datum{Data: plutigoData.NewInteger(big.NewInt(1))}
	exUnits := common.ExUnits{Memory: 1_000, Steps: 2_000}
	p, err := NewPayment(validTestAddrBech32, 2_000_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	a, err := newTestApollo(cc).
		SetWallet(NewExternalWallet(addr)).
		AttachScript(common.PlutusV2Script([]byte{0x01, 0x02})).
		DisableExecutionUnitsEstimation().
		Mint(NewUnit(policy, "746f6b656e", 1), &datum, &exUnits).
		AddPayment(p).
		SetTtl(50000000).
		Complete()
	if err != nil {
		t.Fatal(err)
	}

	report := explain(t, a)
	assertReportMatchesTx(t, a, report)
	if report.Scripts.PlutusV2 != 1 || len(report.Scripts.ScriptHashes) != 1 {
		t.Errorf("Scripts = %+v, want one PlutusV2 script", report.Scripts)
	}
	tx := a.GetTx()
	if len(report.Collateral.Inputs) != 1 || !report.Collateral.AutoSelected {
		t.Fatalf("Collateral = %+v, want one auto-selected input", report.Collateral)
	}
	// The reserved collateral is not available to coin selection.
	if report.Inputs.AvailableUtxoCount != 1 {
		t.Errorf("AvailableUtxoCount = %d, want 1 once collateral is reserved", report.Inputs.AvailableUtxoCount)
	}
	if want, _ := collateralRequirement(report.Fees.Final, 150); report.Collateral.Required != want {
		t.Errorf("collateral Required = %d, want %d", report.Collateral.Required, want)
	}
	if report.Collateral.Total != int64(tx.Body.TxTotalCollateral) || report.Collateral.Total < report.Collateral.Required { //nolint:gosec // small test value
		t.Errorf("collateral Total = %d, transaction has %d", report.Collateral.Total, tx.Body.TxTotalCollateral)
	}
	if report.Collateral.Return == nil || report.Collateral.Return.Value.Coin != tx.Body.TxCollateralReturn.OutputAmount.Amount {
		t.Errorf("collateral Return = %+v, want the transaction's return", report.Collateral.Return)
	}
	if len(report.Redeemers) != 1 {
		t.Fatalf("%d redeemers, want 1", len(report.Redeemers))
	}
	if r := report.Redeemers[0]; r.Tag != common.RedeemerTagMint || r.Index != 0 || r.Source != policy || r.ExUnits != exUnits {
		t.Errorf("redeemer = %+v, want mint 0 for %s", r, policy)
	}
	if report.Fees.ExecutionUnits != exUnits || report.Fees.ExecutionUnitFee == 0 {
		t.Errorf("execution units = %+v fee %d", report.Fees.ExecutionUnits, report.Fees.ExecutionUnitFee)
	}
	if !report.Balance.Mint.HasAssets() || report.Balance.Burn.HasAssets() {
		t.Errorf("Mint = %+v Burn = %+v, want a mint and no burn", report.Balance.Mint, report.Balance.Burn)
	}

	// The returned report's assets and collateral return are copies too.
	report.Balance.Mint.Assets.Add(report.Balance.Mint.Assets)
	report.Collateral.Return.Value.Coin = 1
	again := explain(t, a)
	if again.Balance.Mint.Assets.Asset(mintPolicyID(t, policy), []byte("token")).Cmp(big.NewInt(1)) != 0 {
		t.Error("changing a returned report's assets changed the builder's report")
	}
	if again.Collateral.Return.Value.Coin == 1 {
		t.Error("changing a returned report's collateral return changed the builder's report")
	}
}

func mintPolicyID(t *testing.T, policy string) common.Blake2b224 {
	t.Helper()
	raw, err := hex.DecodeString(policy)
	if err != nil {
		t.Fatal(err)
	}
	return common.NewBlake2b224(raw)
}

func TestBuildReportReferenceScriptFee(t *testing.T) {
	cc := setupFixedContext()
	addr := testAddress(t)
	addTestUtxo(cc, addr, 5_000_000, 0x01, 0)
	addTestUtxo(cc, addr, 5_000_000, 0x02, 0)

	var refTxHash common.Blake2b256
	refTxHash[0] = 0xee
	refTxHash[1] = 0xff
	hashHex := hex.EncodeToString(refTxHash.Bytes())
	cc.AddUtxoByRef(common.Utxo{
		Id: shelley.ShelleyTransactionInput{TxId: refTxHash, OutputIndex: 0},
		Output: &babbage.BabbageTransactionOutput{
			OutputAddress: addr,
			OutputAmount:  mary.MaryTransactionOutputValue{Amount: 2_000_000},
			TxOutScriptRef: &common.ScriptRef{
				Type:   common.ScriptRefTypePlutusV2,
				Script: common.PlutusV2Script(bytes.Repeat([]byte{0x42}, 60_000)),
			},
		},
	})

	p, err := NewPayment(validTestAddrBech32, 4_000_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	a, err := newTestApollo(cc).SetWallet(NewExternalWallet(addr)).AddPayment(p).AddReferenceInput(hashHex, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.Complete(); err != nil {
		t.Fatal(err)
	}

	report := explain(t, a)
	assertReportMatchesTx(t, a, report)
	if report.Fees.ReferenceScriptBytes != 60_000 {
		t.Errorf("ReferenceScriptBytes = %d, want 60000", report.Fees.ReferenceScriptBytes)
	}
	if report.Fees.ReferenceScriptFee == 0 || report.Fees.ReferenceScriptReserve != report.Fees.ReferenceScriptFee {
		t.Errorf("reference-script fee = %d, reserve = %d, want equal and non-zero",
			report.Fees.ReferenceScriptFee, report.Fees.ReferenceScriptReserve)
	}
	if len(report.Scripts.ReferenceInputRefs) != 1 || report.Scripts.ReferenceInputRefs[0] != hashHex+"#0" {
		t.Errorf("ReferenceInputRefs = %v", report.Scripts.ReferenceInputRefs)
	}
}

// assembledFailsContext fails reference-input lookups once the builder has
// assembled its transaction, as when a reference UTxO is spent after Complete
// priced it.
type assembledFailsContext struct {
	*fixed.FixedChainContext
	builder *Apollo
}

func (c *assembledFailsContext) UtxoByRef(txHash common.Blake2b256, index uint32) (*common.Utxo, error) {
	if c.builder.tx != nil {
		return nil, errors.New("reference input spent")
	}
	return c.FixedChainContext.UtxoByRef(txHash, index)
}

func TestBuildReportDoesNotQueryReferenceInputsAgain(t *testing.T) {
	cc := setupFixedContext()
	addr := testAddress(t)
	addTestUtxo(cc, addr, 10_000_000, 0x01, 0)
	var refTxHash common.Blake2b256
	refTxHash[0] = 0xee
	cc.AddUtxoByRef(common.Utxo{
		Id: shelley.ShelleyTransactionInput{TxId: refTxHash, OutputIndex: 0},
		Output: &babbage.BabbageTransactionOutput{
			OutputAddress: addr,
			OutputAmount:  mary.MaryTransactionOutputValue{Amount: 2_000_000},
			TxOutScriptRef: &common.ScriptRef{
				Type:   common.ScriptRefTypePlutusV2,
				Script: common.PlutusV2Script(bytes.Repeat([]byte{0x42}, 1_000)),
			},
		},
	})
	ctx := &assembledFailsContext{FixedChainContext: cc}
	p, err := NewPayment(validTestAddrBech32, 2_000_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	a, err := newTestApollo(ctx).SetWallet(NewExternalWallet(addr)).AddPayment(p).SetTtl(50000000).
		AddReferenceInput(hex.EncodeToString(refTxHash.Bytes()), 0)
	if err != nil {
		t.Fatal(err)
	}
	ctx.builder = a
	if _, err := a.Complete(); err != nil {
		t.Fatal(err)
	}
	if got := explain(t, a).Fees.ReferenceScriptBytes; got != 1_000 {
		t.Errorf("ReferenceScriptBytes = %d, want 1000", got)
	}
}

func TestBuildReportWithdrawal(t *testing.T) {
	cc := setupFixedContext()
	addr := testAddress(t)
	addTestUtxo(cc, addr, 10_000_000, 0x01, 0)

	a, err := newTestApollo(cc).
		SetWallet(NewExternalWallet(addr)).
		SetTtl(50_000_000).
		AddWithdrawal(stakeAddressFor(t, addr), 1_000_000, nil, nil).
		Complete()
	if err != nil {
		t.Fatal(err)
	}
	report := explain(t, a)
	assertReportMatchesTx(t, a, report)
	if report.Balance.Withdrawals.Coin != 1_000_000 || report.Balance.TotalInput.Coin != 11_000_000 {
		t.Errorf("Withdrawals = %d, TotalInput = %d, want 1000000 and 11000000",
			report.Balance.Withdrawals.Coin, report.Balance.TotalInput.Coin)
	}
}

func TestBuildReportDijkstra(t *testing.T) {
	// A direct deposit is a Dijkstra-only body field, so the Dijkstra body
	// encodes differently from the Conway body it is converted from.
	a, err := dijkstraRefScriptBuilder(t, EraDijkstra, withDijkstraRefScriptTiers).
		AddDirectDeposit(stakeAddressFor(t, testAddress(t)), 1_000_000).
		Complete()
	if err != nil {
		t.Fatal(err)
	}
	report := explain(t, a)
	if report.Balance.DirectDeposit != 1_000_000 {
		t.Errorf("DirectDeposit = %d, want 1000000", report.Balance.DirectDeposit)
	}
	tx := a.GetDijkstraTx()
	bodyBytes, err := encodeDijkstraBody(&tx.Body)
	if err != nil {
		t.Fatal(err)
	}
	if want := hex.EncodeToString(common.Blake2b256Hash(bodyBytes).Bytes()); report.TxHash != want {
		t.Errorf("TxHash = %s, want the Dijkstra body hash %s", report.TxHash, want)
	}
	if report.Era != EraDijkstra {
		t.Errorf("Era = %v, want Dijkstra", report.Era)
	}
	if report.Fees.Final != int64(tx.Body.TxFee) { //nolint:gosec // small test fee
		t.Errorf("fee = %d, transaction has %d", report.Fees.Final, tx.Body.TxFee)
	}
	if report.Fees.ReferenceScriptBytes != dijkstraRefScriptTestSize || report.Fees.ReferenceScriptFee == 0 {
		t.Errorf("reference scripts = %d bytes, fee %d", report.Fees.ReferenceScriptBytes, report.Fees.ReferenceScriptFee)
	}
}

// Pinned inputs that cover the payment but leave change too small to emit
// make selection raise its target to fund a change output; the report must
// show the raised target that drew in the extra input.
func TestBuildReportSelectionTargetIncludesChangeFunding(t *testing.T) {
	addr := testAddress(t)
	pinned := benchUtxo(addr, 1, 2_300_000, nil)
	funding := benchUtxo(addr, 2, 30_000_000, nil)
	a := completePinned(t, pinned, []common.Utxo{funding}, 2_000_000)

	report := explain(t, a)
	assertReportMatchesTx(t, a, report)
	if len(report.Inputs.Selected) != 1 || report.Inputs.Selected[0].Ref != utxoRef(funding) {
		t.Fatalf("Selected = %+v, want the funding UTxO", report.Inputs.Selected)
	}
	if report.Inputs.SelectionTarget.Coin <= 2_300_000 {
		t.Errorf("SelectionTarget = %d, which the pinned 2300000 already covers; want the raised target",
			report.Inputs.SelectionTarget.Coin)
	}
}

func TestBuildReportSpendAndRewardRedeemers(t *testing.T) {
	cc := setupFixedContext()
	addr := testAddress(t)
	addTestUtxo(cc, addr, 30_000_000, 0x41, 0)
	addTestUtxo(cc, addr, 20_000_000, 0x43, 0)
	var scriptHash common.Blake2b256
	scriptHash[0] = 0x42
	scriptUtxo := makeTestUtxo(t, scriptHash, 0, 10_000_000)
	reward := stakeAddressFor(t, addr)

	redeemer := common.Datum{Data: plutigoData.NewInteger(big.NewInt(1))}
	spendUnits := common.ExUnits{Memory: 1_000, Steps: 2_000}
	rewardUnits := common.ExUnits{Memory: 3_000, Steps: 4_000}
	p, err := NewPayment(validTestAddrBech32, 2_000_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	a, err := newTestApollo(cc).
		SetWallet(NewExternalWallet(addr)).
		AttachScript(common.PlutusV2Script([]byte{0x01, 0x02})).
		DisableExecutionUnitsEstimation().
		CollectFrom(scriptUtxo, redeemer, spendUnits).
		AddWithdrawal(reward, 1_000_000, &redeemer, &rewardUnits).
		AddPayment(p).
		SetTtl(50_000_000).
		Complete()
	if err != nil {
		t.Fatal(err)
	}

	report := explain(t, a)
	assertReportMatchesTx(t, a, report)
	want := map[common.RedeemerTag]RedeemerReport{
		common.RedeemerTagSpend:  {Source: utxoRef(scriptUtxo), ExUnits: spendUnits},
		common.RedeemerTagReward: {Index: 0, Source: reward.String(), ExUnits: rewardUnits},
	}
	if len(report.Redeemers) != len(want) {
		t.Fatalf("Redeemers = %+v, want spend and reward", report.Redeemers)
	}
	for _, r := range report.Redeemers {
		w, ok := want[r.Tag]
		if !ok || r.Source != w.Source || r.ExUnits != w.ExUnits {
			t.Errorf("redeemer = %+v, want %+v", r, w)
		}
		if r.Tag == common.RedeemerTagSpend && report.Inputs.Final[r.Index].Ref != utxoRef(scriptUtxo) {
			t.Errorf("spend redeemer index %d points at %s", r.Index, report.Inputs.Final[r.Index].Ref)
		}
	}
	total := common.ExUnits{Memory: 4_000, Steps: 6_000}
	if report.Fees.ExecutionUnits != total {
		t.Errorf("ExecutionUnits = %+v, want %+v", report.Fees.ExecutionUnits, total)
	}
}

func TestBuildReportGuardRedeemerAndPlutusV4(t *testing.T) {
	base := dijkstraTestContext(t, dijkstra.MinProtocolVersionDijkstra)
	pp, err := base.ProtocolParams()
	if err != nil {
		t.Fatal(err)
	}
	gp, err := base.GenesisParams()
	if err != nil {
		t.Fatal(err)
	}
	pp.CostModels["PlutusV4"] = pp.CostModels["PlutusV3"]
	cc := fixed.NewFixedChainContext(pp, gp, base.NetworkId())
	addr := testAddress(t)
	addTestUtxo(cc, addr, 20_000_000, 0x01, 0)
	addTestUtxo(cc, addr, 20_000_000, 0x02, 0)
	payment, err := NewPayment(validTestAddrBech32, 2_000_000, nil)
	if err != nil {
		t.Fatal(err)
	}

	signer := guardKeyCredential(0x11)
	exUnits := common.ExUnits{Memory: 1000, Steps: 2000}
	a, err := newTestApollo(cc).
		SetWallet(NewExternalWallet(addr)).
		SetTargetEra(EraDijkstra).
		AddPayment(payment).
		DisableExecutionUnitsEstimation().
		AttachScript(guardTestScript).
		AttachScript(common.PlutusV4Script([]byte{0x04, 0x05})).
		AddRequiredSigner(signer.Credential).
		AddGuard(guardScriptCredential(), guardRedeemer(), &exUnits).
		Complete()
	if err != nil {
		t.Fatal(err)
	}

	report := explain(t, a)
	if report.Scripts.PlutusV3 != 1 || report.Scripts.PlutusV4 != 1 {
		t.Errorf("Scripts = %+v, want one PlutusV3 and one PlutusV4 script", report.Scripts)
	}
	if len(report.Redeemers) != 1 {
		t.Fatalf("Redeemers = %+v, want one guarding redeemer", report.Redeemers)
	}
	r := report.Redeemers[0]
	if r.Tag != common.RedeemerTagGuarding || r.Index != 1 || r.ExUnits != exUnits ||
		r.Source != credentialKey(guardScriptCredential()) {
		t.Errorf("redeemer = %+v, want guarding 1 for %s", r, credentialKey(guardScriptCredential()))
	}
	built := a.dijkstraTx.Body.TxGuards.Credentials
	if len(report.Scripts.Guards) != len(built) || len(built) != 2 {
		t.Fatalf("Guards = %v, transaction has %d guards, want 2", report.Scripts.Guards, len(built))
	}
	for i, credential := range built {
		if report.Scripts.Guards[i] != credentialKey(credential) {
			t.Errorf("guard %d = %s, transaction has %s", i, report.Scripts.Guards[i], credentialKey(credential))
		}
	}
	if report.Scripts.Guards[1] != r.Source {
		t.Errorf("guard 1 = %s, want the guarding redeemer's source %s", report.Scripts.Guards[1], r.Source)
	}
}

func TestBuildReportDijkstraProposalDeposit(t *testing.T) {
	cc := dijkstraTestContext(t, dijkstra.MinProtocolVersionDijkstra)
	addr := testAddress(t)
	addTestUtxo(cc, addr, 10_000_000, 0x01, 0)
	account, err := withdrawalRewardAddress(addr)
	if err != nil {
		t.Fatal(err)
	}
	proposal := dijkstra.DijkstraProposalProcedure{
		PPDeposit:       2_000_000,
		PPRewardAccount: account,
		PPGovAction: dijkstra.DijkstraGovAction{
			Type:   uint(common.GovActionTypeInfo),
			Action: &common.InfoGovAction{Type: uint(common.GovActionTypeInfo)},
		},
		PPAnchor: common.GovAnchor{Url: "https://example.com/info"},
	}
	a, err := newTestApollo(cc).
		SetWallet(NewExternalWallet(addr)).
		SetTargetEra(EraDijkstra).
		AddDijkstraProposal(proposal).
		AddTreasuryDonation(500_000).
		Complete()
	if err != nil {
		t.Fatal(err)
	}
	report := explain(t, a)
	if report.Balance.ProposalDeposit != 2_000_000 || report.Balance.TreasuryDonation != 500_000 ||
		report.Balance.GovernanceRequired.Coin != 2_500_000 {
		t.Errorf("ProposalDeposit = %d, TreasuryDonation = %d, GovernanceRequired = %d, want 2000000, 500000, 2500000",
			report.Balance.ProposalDeposit, report.Balance.TreasuryDonation, report.Balance.GovernanceRequired.Coin)
	}
}

func TestExplainReportsBuildReportFailure(t *testing.T) {
	var nilBuilder *Apollo
	if _, err := nilBuilder.Explain(); err == nil {
		t.Fatal("expected Explain on a nil builder to fail")
	}

	cc := setupFixedContext()
	addr := testAddress(t)
	addTestUtxo(cc, addr, 10_000_000, 0x01, 0)
	p, err := NewPayment(validTestAddrBech32, 2_000_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	a, err := newTestApollo(cc).SetWallet(NewExternalWallet(addr)).AddPayment(p).SetTtl(50000000).Complete()
	if err != nil {
		t.Fatal(err)
	}
	cause := errors.New("report failed")
	a.buildReportErr = cause
	if _, err := a.Explain(); !errors.Is(err, cause) {
		t.Fatalf("Explain error = %v, want it to wrap %v", err, cause)
	}
}

// nilNameSelector is a typed-nil coin selector whose methods dereference
// their receiver.
type nilNameSelector struct{ name string }

func (s *nilNameSelector) Name() string { return s.name }

func (s *nilNameSelector) Select(context.Context, []common.Utxo, Value) ([]common.Utxo, error) {
	return nil, errors.New(s.name)
}

func TestBuildReportTypedNilCoinSelector(t *testing.T) {
	// The pinned input covers the payment and its change, so selection never
	// calls the selector, and only the report would.
	addr := testAddress(t)
	a, err := newTestApollo(setupFixedContext()).
		SetWallet(NewExternalWallet(addr)).
		SetCoinSelector((*nilNameSelector)(nil)).
		AddInput(benchUtxo(addr, 1, 10_000_000, nil)).
		PayToAddress(addr, 2_000_000).
		SetTtl(50_000_000).
		Complete()
	if err != nil {
		t.Fatal(err)
	}
	if got := explain(t, a).CoinSelector; got != defaultCoinSelector.Name() {
		t.Errorf("CoinSelector = %q, want the default %q", got, defaultCoinSelector.Name())
	}
}

func TestBuildReportForcedAndSetFee(t *testing.T) {
	for _, tc := range []struct {
		name string
		set  func(*Apollo) *Apollo
	}{
		{"ForceFee", func(a *Apollo) *Apollo { return a.ForceFee(300_000) }},
		{"SetFee", func(a *Apollo) *Apollo { return a.SetFee(300_000) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cc := setupFixedContext()
			addr := testAddress(t)
			addTestUtxo(cc, addr, 10_000_000, 0x01, 0)
			p, err := NewPayment(validTestAddrBech32, 2_000_000, nil)
			if err != nil {
				t.Fatal(err)
			}
			a, err := tc.set(newTestApollo(cc).SetWallet(NewExternalWallet(addr)).AddPayment(p).
				SetFeePadding(1_000).SetTtl(50000000)).Complete()
			if err != nil {
				t.Fatal(err)
			}
			report := explain(t, a)
			assertReportMatchesTx(t, a, report)
			if !report.Fees.Forced || report.Fees.SizeFee != 0 || report.Fees.Requested != 301_000 {
				t.Errorf("Fees = %+v, want forced, no size fee, requested 301000", report.Fees)
			}
		})
	}
}

func TestBuildReportBurn(t *testing.T) {
	cc := setupFixedContext()
	addr := testAddress(t)
	policy := strings.Repeat("aa", 28)
	policyID := mintPolicyID(t, policy)
	assets := common.NewMultiAsset[common.MultiAssetTypeOutput](
		map[common.Blake2b224]map[cbor.ByteString]common.MultiAssetTypeOutput{
			policyID: {cbor.NewByteString([]byte("tok")): big.NewInt(5)},
		})
	var txHash common.Blake2b256
	txHash[0] = 0x01
	cc.AddUtxo(addr, makeAssetTestUtxo(t, txHash, 0, 10_000_000, &assets))
	p, err := NewPayment(validTestAddrBech32, 2_000_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	a, err := newTestApollo(cc).SetWallet(NewExternalWallet(addr)).AddPayment(p).SetTtl(50000000).
		Mint(NewUnit(policy, "746f6b", -2), nil, nil).
		Complete()
	if err != nil {
		t.Fatal(err)
	}
	report := explain(t, a)
	assertReportMatchesTx(t, a, report)
	if got := report.Balance.Burn.Assets.Asset(policyID, []byte("tok")); got == nil || got.Cmp(big.NewInt(2)) != 0 {
		t.Errorf("Burn = %v, want 2", got)
	}
	if got := report.Balance.Mint.Assets.Asset(policyID, []byte("tok")); got == nil || got.Cmp(big.NewInt(-2)) != 0 {
		t.Errorf("Mint = %v, want -2", got)
	}
	if got := report.Inputs.SelectionTarget.Assets.Asset(policyID, []byte("tok")); got == nil || got.Cmp(big.NewInt(2)) != 0 {
		t.Errorf("SelectionTarget assets = %v, want the 2 burned tokens", got)
	}
}

func TestBuildReportCertificateDepositAndRefund(t *testing.T) {
	cc := setupFixedContext()
	addr := testAddress(t)
	addTestUtxo(cc, addr, 10_000_000, 0x01, 0)
	cred := common.Credential{CredType: common.CredentialTypeAddrKeyHash, Credential: addr.StakeKeyHash()}

	register, err := newTestApollo(cc).SetWallet(NewExternalWallet(addr)).SetTtl(50000000).RegisterStake(&cred)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := register.Complete(); err != nil {
		t.Fatal(err)
	}
	report := explain(t, register)
	assertReportMatchesTx(t, register, report)
	if report.Balance.CertificateDeposit != 2_000_000 || report.Balance.CertificateRefund.Coin != 0 {
		t.Errorf("CertificateDeposit = %d, CertificateRefund = %d, want 2000000 and 0",
			report.Balance.CertificateDeposit, report.Balance.CertificateRefund.Coin)
	}

	deregister, err := newTestApollo(cc).SetWallet(NewExternalWallet(addr)).SetTtl(50000000).DeregisterStake(&cred)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := deregister.Complete(); err != nil {
		t.Fatal(err)
	}
	report = explain(t, deregister)
	assertReportMatchesTx(t, deregister, report)
	if report.Balance.CertificateDeposit != 0 || report.Balance.CertificateRefund.Coin != 2_000_000 {
		t.Errorf("CertificateDeposit = %d, CertificateRefund = %d, want 0 and 2000000",
			report.Balance.CertificateDeposit, report.Balance.CertificateRefund.Coin)
	}
}

func TestBuildReportCollateralOverlap(t *testing.T) {
	cc := setupFixedContext()
	addr := testAddress(t)
	addTestUtxo(cc, addr, 20_000_000, 0x01, 0)
	datum := common.Datum{Data: plutigoData.NewInteger(big.NewInt(1))}
	p, err := NewPayment(validTestAddrBech32, 2_000_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	a, err := newTestApollo(cc).
		SetWallet(NewExternalWallet(addr)).
		AttachScript(common.PlutusV2Script([]byte{0x01, 0x02})).
		DisableExecutionUnitsEstimation().
		Mint(NewUnit("a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4", "746f6b656e", 1),
			&datum, &common.ExUnits{Memory: 1, Steps: 1}).
		AddPayment(p).
		SetTtl(50000000).
		Complete()
	if err != nil {
		t.Fatal(err)
	}
	report := explain(t, a)
	assertReportMatchesTx(t, a, report)
	if len(report.Collateral.Inputs) != 1 || report.Collateral.OverlapRef != report.Collateral.Inputs[0].Ref ||
		report.Collateral.OverlapRef != report.Inputs.Final[0].Ref {
		t.Errorf("OverlapRef = %q, collateral %v, inputs %v, want the single UTxO in both",
			report.Collateral.OverlapRef, report.Collateral.Inputs, report.Inputs.Final)
	}
}

package apollo

import (
	"bytes"
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/babbage"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/mary"
	"github.com/blinklabs-io/gouroboros/ledger/shelley"
	plutigoData "github.com/blinklabs-io/plutigo/data"
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

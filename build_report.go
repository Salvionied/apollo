package apollo

import (
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strconv"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/babbage"
	"github.com/blinklabs-io/gouroboros/ledger/common"

	"github.com/Salvionied/apollo/v2/backend"
)

// BuildReport describes the transaction Complete built: how its inputs were
// chosen, how its outputs and change were formed, and what its fee,
// collateral, and redeemers consist of. Obtain it with Explain.
type BuildReport struct {
	// TxHash is the hex transaction ID.
	TxHash string
	// TxSize is the encoded size of the transaction as built, before
	// signatures are added.
	TxSize       int
	Era          TargetEra
	CoinSelector string
	Inputs       InputReport
	Outputs      OutputReport
	Fees         FeeReport
	Balance      BalanceReport
	Collateral   CollateralReport
	// Redeemers are in ledger order, indexed as in the transaction.
	Redeemers []RedeemerReport
	Scripts   ScriptReport
}

// InputReport describes how the transaction inputs were chosen.
type InputReport struct {
	InputAddresses []string
	// AvailableUtxoCount is the number of UTxOs coin selection could draw
	// on: loaded UTxOs not already spent as inputs or reserved as collateral.
	AvailableUtxoCount int
	// Preselected are the inputs added explicitly, and Selected the ones coin
	// selection added. Final is every input in transaction order.
	Preselected []UtxoReport
	Selected    []UtxoReport
	Final       []UtxoReport
	// SelectionTarget is the value coin selection last had to cover,
	// including the fee reserve, any tokens being burned, and any raise to
	// fund a change output that pinned inputs alone would leave too small.
	SelectionTarget Value
}

// UtxoReport describes one UTxO.
type UtxoReport struct {
	// Ref is "<transaction hash>#<index>".
	Ref     string
	Address string
	Value   Value
}

// OutputReport describes the payment outputs and change.
type OutputReport struct {
	// Requested holds each payment's value before the minimum-UTxO
	// adjustment, in payment order.
	Requested []Value
	// Base holds the payment outputs after the minimum-UTxO adjustment.
	Base []TxOutputReport
	// Final holds the outputs in the transaction, including change.
	Final []TxOutputReport
	// ChangeIndex is the index of the change output in Final, or -1 when the
	// transaction has no change output.
	ChangeIndex int
	Change      Value
}

// TxOutputReport describes one transaction output.
type TxOutputReport struct {
	Index    int
	Address  string
	Value    Value
	IsChange bool
	// MinLovelace is the output's minimum-UTxO requirement.
	MinLovelace int64
	// MinUtxoIncrease is the lovelace the builder added to a payment so that
	// it meets MinLovelace. It is zero for change.
	MinUtxoIncrease uint64
	HasInlineDatum  bool
	HasDatumHash    bool
	HasScriptRef    bool
}

// FeeReport describes the transaction fee.
type FeeReport struct {
	// Final is the fee in the transaction.
	Final int64
	// Requested is the fee the builder settled on, including Padding. Final
	// exceeds it by AbsorbedChange.
	Requested int64
	// AbsorbedChange is ADA-only change below the minimum UTxO, added to the
	// fee instead of being returned in a change output.
	AbsorbedChange int64
	// Forced reports that the fee was set by the caller rather than
	// estimated.
	Forced  bool
	Padding int64
	// SizeFee is the estimated part of Requested that is neither padding,
	// execution-unit fee, nor reference-script fee: the fee for the
	// transaction's size. It is zero when the fee was forced.
	SizeFee              int64
	ReferenceScriptBytes int
	ReferenceScriptFee   int64
	ExecutionUnits       common.ExUnits
	ExecutionUnitFee     int64
	// Preliminary is the fee ceiling coin selection could fall back to:
	// the maximum transaction fee plus ReferenceScriptReserve.
	Preliminary int64
	// ReferenceScriptReserve is the reference-script fee known before
	// coin selection, from reference and preselected inputs.
	ReferenceScriptReserve int64
	// SelectionReserve is the fee reserve coin selection last used.
	SelectionReserve int64
}

// BalanceReport describes the components of the balance equation.
type BalanceReport struct {
	// TotalInput is the value of the inputs plus mint, withdrawals, and
	// certificate refunds.
	TotalInput Value
	// TotalRequired is the value of the payment outputs plus certificate and
	// direct deposits.
	TotalRequired Value
	// TotalOutput is the value of every output, including change.
	TotalOutput Value
	Change      Value
	Fee         int64
	Withdrawals Value
	// Mint is the net mint, with burns as negative quantities; Burn holds the
	// burned quantities as positive amounts.
	Mint               Value
	Burn               Value
	CertificateDeposit int64
	CertificateRefund  Value
	// GovernanceRequired is the value proposal deposits and the treasury
	// donation take.
	GovernanceRequired Value
	// ProposalDeposit is the deposit of every Conway and Dijkstra proposal.
	ProposalDeposit  uint64
	TreasuryDonation uint64
	DirectDeposit    uint64
}

// CollateralReport describes the collateral inputs and return.
type CollateralReport struct {
	Inputs []UtxoReport
	// Required is the collateral the fee requires. It is zero when the
	// transaction has no collateral.
	Required int64
	Total    int64
	Return   *TxOutputReport
	// AutoSelected reports that the builder chose the collateral.
	AutoSelected bool
	// OverlapRef is an auto-selected collateral UTxO also spent as an input,
	// or empty.
	OverlapRef string
}

// RedeemerReport describes one redeemer.
type RedeemerReport struct {
	Tag   common.RedeemerTag
	Index uint32
	// Source is what the redeemer's index points at: the spent UTxO ref, the
	// minting policy ID, the reward address, or the guard credential.
	Source  string
	ExUnits common.ExUnits
}

// ScriptReport describes the scripts, datums, and signers.
type ScriptReport struct {
	PlutusV1           int
	PlutusV2           int
	PlutusV3           int
	PlutusV4           int
	Native             int
	Datums             int
	ReferenceInputRefs []string
	ScriptHashes       []string
	RequiredSignerKeys []string
}

// buildReportInputs holds the values Complete used, so the report describes
// the transaction it built rather than a recomputation of it.
type buildReportInputs struct {
	pp                  backend.ProtocolParameters
	requested           []Value
	baseOutputs         []babbage.BabbageTransactionOutput
	outputs             []babbage.BabbageTransactionOutput
	selection           coinSelection
	inputs              []common.Utxo
	totalInput          Value
	totalRequired       Value
	governanceRequired  Value
	refundValue         Value
	stakeDeposit        int64
	poolDeposit         int64
	prelimFee           int64
	refScriptFeeReserve int64
	selectionReserve    int64
	requestedFee        int64
	fee                 int64
}

// Explain returns the report for the transaction Complete built. The report
// is a copy; changing it does not affect the builder. A transaction loaded
// with LoadTxCbor has no report, since the decisions that built it are not
// recorded in its CBOR.
func (a *Apollo) Explain() (*BuildReport, error) {
	if a.buildReportErr != nil {
		return nil, fmt.Errorf("build report unavailable: %w", a.buildReportErr)
	}
	if a.buildReport == nil {
		if a.tx == nil && a.dijkstraTx == nil {
			return nil, errors.New("transaction not built - call Complete() first")
		}
		return nil, errors.New("no build report: the transaction was loaded from CBOR or Complete failed")
	}
	return a.buildReport.clone(), nil
}

func (a *Apollo) newBuildReport(in buildReportInputs) (*BuildReport, error) {
	var bodyBytes, txBytes []byte
	var err error
	if a.dijkstraTx != nil {
		if bodyBytes, err = encodeDijkstraBody(&a.dijkstraTx.Body); err == nil {
			txBytes, err = cbor.Encode(a.dijkstraTx)
		}
	} else {
		if bodyBytes, err = cbor.Encode(&a.tx.Body); err == nil {
			txBytes, err = cbor.Encode(a.tx)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("encode transaction: %w", err)
	}
	coinsPerUtxoByte := in.pp.CoinsPerUtxoByteValue()

	base, err := outputReports(in.baseOutputs, -1, in.requested, coinsPerUtxoByte)
	if err != nil {
		return nil, err
	}
	changeIndex := -1
	if len(in.outputs) > len(in.baseOutputs) {
		changeIndex = len(in.baseOutputs)
	}
	final, err := outputReports(in.outputs, changeIndex, in.requested, coinsPerUtxoByte)
	if err != nil {
		return nil, err
	}
	var change Value
	if changeIndex >= 0 {
		change = final[changeIndex].Value.Clone()
	}
	totalOutput, err := a.totalOutputValue(in.outputs)
	if err != nil {
		return nil, err
	}

	redeemers := a.redeemerReports(in.inputs)
	var exUnits common.ExUnits
	for _, r := range redeemers {
		exUnits.Memory += r.ExUnits.Memory
		exUnits.Steps += r.ExUnits.Steps
	}
	exUnitFee, err := executionUnitFee(in.pp, exUnits.Memory, exUnits.Steps)
	if err != nil {
		return nil, err
	}
	refScriptBytes, err := a.totalReferenceScriptSize(in.inputs)
	if err != nil {
		return nil, err
	}
	refScriptFee, err := a.eraReferenceScriptFee(refScriptBytes, in.pp)
	if err != nil {
		return nil, err
	}
	forced := a.forceFee || a.Fee > 0
	var sizeFee int64
	if !forced {
		sizeFee = max(in.requestedFee-a.FeePadding-exUnitFee-refScriptFee, 0)
	}

	mint, err := a.mintValue()
	if err != nil {
		return nil, err
	}
	burn, err := a.burnRequirementValue()
	if err != nil {
		return nil, err
	}
	depositAdjustment, err := a.certificateDepositAdjustment(in.stakeDeposit, in.poolDeposit)
	if err != nil {
		return nil, err
	}
	directDeposit, err := dijkstraDirectDepositTotal(a.directDeposits)
	if err != nil {
		return nil, err
	}
	var collateralRequired int64
	if len(a.collaterals) > 0 {
		if collateralRequired, err = collateralRequirement(in.fee, in.pp.CollateralPercent); err != nil {
			return nil, err
		}
	}
	var collateralReturn *TxOutputReport
	if a.collateralReturn != nil {
		r, err := outputReport(0, *a.collateralReturn, false, 0, coinsPerUtxoByte)
		if err != nil {
			return nil, err
		}
		collateralReturn = &r
	}

	return &BuildReport{
		TxHash:       hex.EncodeToString(common.Blake2b256Hash(bodyBytes).Bytes()),
		TxSize:       len(txBytes),
		Era:          a.targetEra,
		CoinSelector: a.coinSelectorName(),
		Inputs: InputReport{
			InputAddresses:     addressStrings(a.inputAddresses),
			AvailableUtxoCount: in.selection.available,
			Preselected:        utxoReports(a.preselectedUtxos),
			Selected:           utxoReports(in.selection.utxos),
			Final:              utxoReports(in.inputs),
			SelectionTarget:    in.selection.target.Clone(),
		},
		Outputs: OutputReport{
			Requested:   cloneValues(in.requested),
			Base:        base,
			Final:       final,
			ChangeIndex: changeIndex,
			Change:      change,
		},
		Fees: FeeReport{
			Final:                  in.fee,
			Requested:              in.requestedFee,
			AbsorbedChange:         in.fee - in.requestedFee,
			Forced:                 forced,
			Padding:                a.FeePadding,
			SizeFee:                sizeFee,
			ReferenceScriptBytes:   refScriptBytes,
			ReferenceScriptFee:     refScriptFee,
			ExecutionUnits:         exUnits,
			ExecutionUnitFee:       exUnitFee,
			Preliminary:            in.prelimFee,
			ReferenceScriptReserve: in.refScriptFeeReserve,
			SelectionReserve:       in.selectionReserve,
		},
		Balance: BalanceReport{
			TotalInput:         in.totalInput.Clone(),
			TotalRequired:      in.totalRequired.Clone(),
			TotalOutput:        totalOutput,
			Change:             change.Clone(),
			Fee:                in.fee,
			Withdrawals:        a.totalWithdrawalValue(),
			Mint:               mint,
			Burn:               burn,
			CertificateDeposit: max(depositAdjustment, 0),
			CertificateRefund:  in.refundValue.Clone(),
			GovernanceRequired: in.governanceRequired.Clone(),
			ProposalDeposit:    in.governanceRequired.Coin - uint64(a.treasuryDonation), //nolint:gosec // AddTreasuryDonation keeps it non-negative
			TreasuryDonation:   uint64(a.treasuryDonation),                              //nolint:gosec // AddTreasuryDonation keeps it non-negative
			DirectDeposit:      directDeposit,
		},
		Collateral: CollateralReport{
			Inputs:       utxoReports(a.collaterals),
			Required:     collateralRequired,
			Total:        a.totalCollateral,
			Return:       collateralReturn,
			AutoSelected: a.collateralAutoSelected,
			OverlapRef:   a.collateralOverlapRef,
		},
		Redeemers: redeemers,
		Scripts: ScriptReport{
			PlutusV1:           len(a.v1scripts),
			PlutusV2:           len(a.v2scripts),
			PlutusV3:           len(a.v3scripts),
			PlutusV4:           len(a.v4scripts),
			Native:             len(a.nativescripts),
			Datums:             len(a.datums),
			ReferenceInputRefs: referenceInputRefs(a),
			ScriptHashes:       append([]string(nil), a.scriptHashes...),
			RequiredSignerKeys: requiredSignerKeys(a.requiredSigners),
		},
	}, nil
}

func (a *Apollo) coinSelectorName() string {
	if a.coinSelector != nil {
		return a.coinSelector.Name()
	}
	return defaultCoinSelector.Name()
}

// redeemerReports lists the redeemers buildWitnessSet writes, with what each
// index refers to.
func (a *Apollo) redeemerReports(inputs []common.Utxo) []RedeemerReport {
	redeemerMap := a.buildRedeemerMap(inputs)
	reports := make([]RedeemerReport, 0, len(redeemerMap))
	for key, value := range redeemerMap {
		reports = append(reports, RedeemerReport{
			Tag:     key.Tag,
			Index:   key.Index,
			Source:  a.redeemerSource(key, inputs),
			ExUnits: value.ExUnits,
		})
	}
	sort.Slice(reports, func(i, j int) bool {
		if reports[i].Tag != reports[j].Tag {
			return reports[i].Tag < reports[j].Tag
		}
		return reports[i].Index < reports[j].Index
	})
	return reports
}

func (a *Apollo) redeemerSource(key common.RedeemerKey, inputs []common.Utxo) string {
	switch key.Tag {
	case common.RedeemerTagSpend:
		if uint64(key.Index) < uint64(len(inputs)) {
			return utxoRef(inputs[key.Index])
		}
	case common.RedeemerTagMint:
		if policies := a.sortedMintPolicyIds(); uint64(key.Index) < uint64(len(policies)) {
			return policies[key.Index]
		}
	case common.RedeemerTagReward:
		if keys := a.sortedWithdrawalKeys(); uint64(key.Index) < uint64(len(keys)) {
			return a.withdrawals[keys[key.Index]].Address.String()
		}
	case common.RedeemerTagGuarding:
		if guard, ok := a.guardRedeemerKeyAt(key.Index); ok {
			return guard
		}
	}
	return ""
}

// outputReports describes outputs. requested[i] is the value payment i asked
// for, from which the minimum-UTxO increase of output i is measured; change
// follows the payments, so it has no requested value.
func outputReports(
	outputs []babbage.BabbageTransactionOutput,
	changeIndex int,
	requested []Value,
	coinsPerUtxoByte int64,
) ([]TxOutputReport, error) {
	reports := make([]TxOutputReport, 0, len(outputs))
	for i := range outputs {
		var increase uint64
		if i < len(requested) && outputs[i].OutputAmount.Amount > requested[i].Coin {
			increase = outputs[i].OutputAmount.Amount - requested[i].Coin
		}
		report, err := outputReport(i, outputs[i], i == changeIndex, increase, coinsPerUtxoByte)
		if err != nil {
			return nil, err
		}
		reports = append(reports, report)
	}
	return reports, nil
}

func outputReport(
	index int,
	output babbage.BabbageTransactionOutput,
	isChange bool,
	increase uint64,
	coinsPerUtxoByte int64,
) (TxOutputReport, error) {
	minLovelace, err := MinLovelacePostAlonzo(&output, coinsPerUtxoByte)
	if err != nil {
		return TxOutputReport{}, fmt.Errorf("min UTxO of output %d: %w", index, err)
	}
	return TxOutputReport{
		Index:           index,
		Address:         output.OutputAddress.String(),
		Value:           ValueFromMaryValue(output.OutputAmount),
		IsChange:        isChange,
		MinLovelace:     minLovelace,
		MinUtxoIncrease: increase,
		HasInlineDatum:  output.Datum() != nil,
		HasDatumHash:    output.DatumHash() != nil,
		HasScriptRef:    output.ScriptRef() != nil,
	}, nil
}

func utxoReports(utxos []common.Utxo) []UtxoReport {
	reports := make([]UtxoReport, 0, len(utxos))
	for _, utxo := range utxos {
		report := UtxoReport{Ref: utxoRef(utxo)}
		if utxo.Output != nil {
			report.Address = utxo.Output.Address().String()
			if amount := utxo.Output.Amount(); amount != nil && amount.IsUint64() {
				report.Value.Coin = amount.Uint64()
			}
			report.Value.Assets = CloneMultiAsset(utxo.Output.Assets())
		}
		reports = append(reports, report)
	}
	return reports
}

func addressStrings(addrs []common.Address) []string {
	result := make([]string, len(addrs))
	for i, addr := range addrs {
		result[i] = addr.String()
	}
	return result
}

func referenceInputRefs(a *Apollo) []string {
	refs := make([]string, len(a.referenceInputs))
	for i, input := range a.referenceInputs {
		refs[i] = hex.EncodeToString(input.TxId.Bytes()) + "#" + strconv.FormatUint(uint64(input.OutputIndex), 10)
	}
	return refs
}

func requiredSignerKeys(signers []common.Blake2b224) []string {
	keys := make([]string, len(signers))
	for i, signer := range signers {
		keys[i] = hex.EncodeToString(signer.Bytes())
	}
	return keys
}

func cloneValues(values []Value) []Value {
	if values == nil {
		return nil
	}
	cp := make([]Value, len(values))
	for i, v := range values {
		cp[i] = v.Clone()
	}
	return cp
}

func (r *BuildReport) clone() *BuildReport {
	if r == nil {
		return nil
	}
	cp := *r
	cp.Inputs.InputAddresses = append([]string(nil), r.Inputs.InputAddresses...)
	cp.Inputs.Preselected = cloneUtxoReports(r.Inputs.Preselected)
	cp.Inputs.Selected = cloneUtxoReports(r.Inputs.Selected)
	cp.Inputs.Final = cloneUtxoReports(r.Inputs.Final)
	cp.Inputs.SelectionTarget = r.Inputs.SelectionTarget.Clone()
	cp.Outputs.Requested = cloneValues(r.Outputs.Requested)
	cp.Outputs.Base = cloneTxOutputReports(r.Outputs.Base)
	cp.Outputs.Final = cloneTxOutputReports(r.Outputs.Final)
	cp.Outputs.Change = r.Outputs.Change.Clone()
	cp.Balance.TotalInput = r.Balance.TotalInput.Clone()
	cp.Balance.TotalRequired = r.Balance.TotalRequired.Clone()
	cp.Balance.TotalOutput = r.Balance.TotalOutput.Clone()
	cp.Balance.Change = r.Balance.Change.Clone()
	cp.Balance.Withdrawals = r.Balance.Withdrawals.Clone()
	cp.Balance.Mint = r.Balance.Mint.Clone()
	cp.Balance.Burn = r.Balance.Burn.Clone()
	cp.Balance.CertificateRefund = r.Balance.CertificateRefund.Clone()
	cp.Balance.GovernanceRequired = r.Balance.GovernanceRequired.Clone()
	cp.Collateral.Inputs = cloneUtxoReports(r.Collateral.Inputs)
	if r.Collateral.Return != nil {
		ret := *r.Collateral.Return
		ret.Value = r.Collateral.Return.Value.Clone()
		cp.Collateral.Return = &ret
	}
	cp.Redeemers = append([]RedeemerReport(nil), r.Redeemers...)
	cp.Scripts.ReferenceInputRefs = append([]string(nil), r.Scripts.ReferenceInputRefs...)
	cp.Scripts.ScriptHashes = append([]string(nil), r.Scripts.ScriptHashes...)
	cp.Scripts.RequiredSignerKeys = append([]string(nil), r.Scripts.RequiredSignerKeys...)
	return &cp
}

func cloneUtxoReports(reports []UtxoReport) []UtxoReport {
	if reports == nil {
		return nil
	}
	cp := make([]UtxoReport, len(reports))
	for i, report := range reports {
		cp[i] = report
		cp[i].Value = report.Value.Clone()
	}
	return cp
}

func cloneTxOutputReports(reports []TxOutputReport) []TxOutputReport {
	if reports == nil {
		return nil
	}
	cp := make([]TxOutputReport, len(reports))
	for i, report := range reports {
		cp[i] = report
		cp[i].Value = report.Value.Clone()
	}
	return cp
}

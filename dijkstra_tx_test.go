package apollo

import (
	"bytes"
	"encoding/hex"
	"math/big"
	"strings"
	"testing"

	"github.com/Salvionied/apollo/v2/backend"
	"github.com/Salvionied/apollo/v2/backend/fixed"
	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/babbage"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/conway"
	"github.com/blinklabs-io/gouroboros/ledger/dijkstra"
	"github.com/blinklabs-io/gouroboros/ledger/mary"
	"github.com/blinklabs-io/gouroboros/ledger/shelley"
)

func dijkstraTestContext(t *testing.T, protocolMajor int) *fixed.FixedChainContext {
	t.Helper()
	base := setupFixedContext()
	pp, err := base.ProtocolParams()
	if err != nil {
		t.Fatal(err)
	}
	costModel := make([]int64, 297)
	for i := range costModel {
		costModel[i] = int64(i + 1)
	}
	pp.CostModels = map[string][]int64{"PlutusV3": costModel}
	pp.ProtocolMajorVersion = protocolMajor
	gp, err := base.GenesisParams()
	if err != nil {
		t.Fatal(err)
	}
	return fixed.NewFixedChainContext(pp, gp, base.NetworkId())
}

func TestDijkstraPlutusV4ReferenceOutput(t *testing.T) {
	cc := dijkstraTestContext(t, dijkstra.MinProtocolVersionDijkstra)
	addr := testAddress(t)
	addTestUtxo(cc, addr, 10_000_000, 0x01, 0)
	payment, err := NewPayment(validTestAddrBech32, 2_000_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	builder := New(cc).SetWallet(NewExternalWallet(addr)).SetTargetEra(EraDijkstra).AddPayment(payment)
	if _, err := builder.PayToAddressWithReferenceScript(addr, 1_000_000, common.PlutusV4Script{0x01}); err != nil {
		t.Fatalf("add V4 reference script output: %v", err)
	}
	if _, err := builder.Complete(); err != nil {
		t.Fatalf("complete Dijkstra transaction with V4 reference script: %v", err)
	}
	tx := builder.GetDijkstraTx()
	if tx == nil || len(tx.WitnessSet.WsPlutusV4Scripts.Items()) != 0 {
		t.Fatal("Dijkstra V4 reference script was unexpectedly placed in witness set")
	}
	found := false
	for _, output := range tx.Body.TxOutputs {
		if output.Output != nil && output.Output.ScriptRef() != nil {
			typeID, typeErr := scriptRefType(output.Output.ScriptRef())
			if typeErr == nil && typeID == common.ScriptRefTypePlutusV4 {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("Dijkstra transaction did not retain the Plutus V4 reference script")
	}
}

func TestCompleteDijkstraTransaction(t *testing.T) {
	cc := dijkstraTestContext(t, dijkstra.MinProtocolVersionDijkstra)
	addr := testAddress(t)
	addTestUtxo(cc, addr, 10_000_000, 0x01, 0)
	account, err := withdrawalRewardAddress(addr)
	if err != nil {
		t.Fatal(err)
	}
	accountBytes, err := account.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	exactBalance := uint64(3_000_000)
	payment, err := NewPayment(validTestAddrBech32, 2_000_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	builder, err := New(cc).
		SetWallet(NewExternalWallet(addr)).
		SetTargetEra(EraDijkstra).
		AddDirectDeposit(account, 1_000_000).
		SetAccountBalanceInterval(account, dijkstra.DijkstraAccountBalanceInterval{Exact: &exactBalance}).
		AddPayment(payment).
		Complete()
	if err != nil {
		t.Fatal(err)
	}
	if builder.GetTx() != nil {
		t.Fatal("GetTx returned a Conway transaction for a Dijkstra build")
	}
	tx := builder.GetDijkstraTx()
	if tx == nil {
		t.Fatal("GetDijkstraTx returned nil")
	}
	key := cbor.NewByteString(accountBytes)
	if got := tx.Body.TxDirectDeposits[key]; got != 1_000_000 {
		t.Fatalf("direct deposit = %d, want 1000000", got)
	}
	if got := tx.Body.TxBalanceIntervals[key]; got == nil || got.Exact == nil || *got.Exact != exactBalance {
		t.Fatalf("account balance interval = %#v, want exact %d", got, exactBalance)
	}
	txCbor, err := builder.GetTxCbor()
	if err != nil {
		t.Fatal(err)
	}
	var envelope []cbor.RawMessage
	if _, err := cbor.Decode(txCbor, &envelope); err != nil {
		t.Fatalf("decode Dijkstra envelope: %v", err)
	}
	if len(envelope) != 3 {
		t.Fatalf("Dijkstra envelope components = %d, want 3", len(envelope))
	}
	if _, err := dijkstra.NewDijkstraTransactionFromCbor(txCbor); err != nil {
		t.Fatalf("decode Dijkstra transaction: %v", err)
	}

	loaded, err := New(cc).LoadTxCbor(hex.EncodeToString(txCbor))
	if err != nil {
		t.Fatalf("load Dijkstra transaction: %v", err)
	}
	if loaded.GetDijkstraTx() == nil || loaded.GetTx() != nil {
		t.Fatal("LoadTxCbor did not preserve the Dijkstra transaction type")
	}
}

func TestCompleteDijkstraRequiresProtocolVersion12(t *testing.T) {
	cc := dijkstraTestContext(t, dijkstra.MinProtocolVersionDijkstra-1)
	addr := testAddress(t)
	addTestUtxo(cc, addr, 10_000_000, 0x01, 0)
	payment, err := NewPayment(validTestAddrBech32, 2_000_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = New(cc).
		SetWallet(NewExternalWallet(addr)).
		SetTargetEra(EraDijkstra).
		AddPayment(payment).
		Complete()
	if err == nil {
		t.Fatal("Complete succeeded on protocol version below Dijkstra")
	}
}

const dijkstraRefScriptTestSize = 3000

// dijkstraRefScriptBuilder returns a Dijkstra builder spending a wallet UTxO
// and referencing a UTxO that carries a dijkstraRefScriptTestSize-byte script.
func dijkstraRefScriptBuilder(t *testing.T, era TargetEra, mutate func(*backend.ProtocolParameters)) *Apollo {
	t.Helper()
	base := setupFixedContext()
	pp, err := base.ProtocolParams()
	if err != nil {
		t.Fatal(err)
	}
	pp.ProtocolMajorVersion = dijkstra.MinProtocolVersionDijkstra
	if mutate != nil {
		mutate(&pp)
	}
	gp, err := base.GenesisParams()
	if err != nil {
		t.Fatal(err)
	}
	cc := fixed.NewFixedChainContext(pp, gp, base.NetworkId())
	addr := testAddress(t)
	addTestUtxo(cc, addr, 10_000_000, 0x01, 0)

	var refTxHash common.Blake2b256
	refTxHash[0] = 0xd1
	cc.AddUtxoByRef(common.Utxo{
		Id: shelley.ShelleyTransactionInput{TxId: refTxHash, OutputIndex: 0},
		Output: &babbage.BabbageTransactionOutput{
			OutputAddress: addr,
			OutputAmount:  mary.MaryTransactionOutputValue{Amount: 5_000_000},
			TxOutScriptRef: &common.ScriptRef{
				Type:   common.ScriptRefTypePlutusV2,
				Script: common.PlutusV2Script(bytes.Repeat([]byte{0x42}, dijkstraRefScriptTestSize)),
			},
		},
	})

	payment, err := NewPayment(validTestAddrBech32, 2_000_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	a, err := New(cc).
		SetWallet(NewExternalWallet(addr)).
		SetTargetEra(era).
		AddPayment(payment).
		AddReferenceInput(hex.EncodeToString(refTxHash.Bytes()), 0)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func withDijkstraRefScriptTiers(pp *backend.ProtocolParameters) {
	pp.MinFeeReferenceScriptsRange = 1000
	pp.MinFeeReferenceScriptsMultiplierRational = big.NewRat(2, 1)
}

func TestCompleteDijkstraRefScriptFeeUsesProtocolTierParams(t *testing.T) {
	a, err := dijkstraRefScriptBuilder(t, EraDijkstra, withDijkstraRefScriptTiers).Complete()
	if err != nil {
		t.Fatal(err)
	}
	tx := a.GetDijkstraTx()
	if tx == nil {
		t.Fatal("GetDijkstraTx returned nil")
	}
	ledgerPparams := &dijkstra.DijkstraProtocolParameters{
		ConwayProtocolParameters: conway.ConwayProtocolParameters{
			MinFeeA:                    44,
			MinFeeB:                    155381,
			MinFeeRefScriptCostPerByte: &cbor.Rat{Rat: big.NewRat(15, 1)},
		},
		RefScriptCostStride:     1000,
		RefScriptCostMultiplier: &cbor.Rat{Rat: big.NewRat(2, 1)},
	}
	minFee, err := dijkstra.MinFeeTxWithRefScriptSize(tx, ledgerPparams, dijkstraRefScriptTestSize)
	if err != nil {
		t.Fatal(err)
	}
	if tx.Body.TxFee < minFee {
		t.Fatalf("fee %d is below the Dijkstra ledger minimum %d", tx.Body.TxFee, minFee)
	}
	// With Conway's fixed 25600-byte stride the whole script would be priced at
	// the base rate; the protocol-supplied stride and multiplier must apply.
	ledgerPparams.RefScriptCostStride = 25600
	ledgerPparams.RefScriptCostMultiplier = &cbor.Rat{Rat: big.NewRat(6, 5)}
	conwayTierFee, err := dijkstra.MinFeeTxWithRefScriptSize(tx, ledgerPparams, dijkstraRefScriptTestSize)
	if err != nil {
		t.Fatal(err)
	}
	if minFee <= conwayTierFee {
		t.Fatalf("protocol tier minimum %d should exceed the Conway-tier minimum %d", minFee, conwayTierFee)
	}
}

func TestCompleteDijkstraRefScriptRequiresTierParams(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*backend.ProtocolParameters)
	}{
		{"neither", nil},
		{"stride only", func(pp *backend.ProtocolParameters) { pp.MinFeeReferenceScriptsRange = 1000 }},
		{"multiplier only", func(pp *backend.ProtocolParameters) {
			pp.MinFeeReferenceScriptsMultiplierRational = big.NewRat(2, 1)
		}},
		{"zero multiplier", func(pp *backend.ProtocolParameters) {
			pp.MinFeeReferenceScriptsRange = 1000
			pp.MinFeeReferenceScriptsMultiplierRational = new(big.Rat)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := dijkstraRefScriptBuilder(t, EraDijkstra, tt.mutate).Complete()
			if err == nil || !strings.Contains(err.Error(), "cost stride and multiplier") {
				t.Fatalf("Complete error = %v, want missing tier parameter error", err)
			}
		})
	}
}

func TestCompleteConwayRefScriptKeepsDefaultTierParams(t *testing.T) {
	if _, err := dijkstraRefScriptBuilder(t, EraConway, nil).Complete(); err != nil {
		t.Fatalf("Conway build must keep defaulting the tier parameters: %v", err)
	}
}

func TestCompleteDijkstraRefScriptSizeLimit(t *testing.T) {
	tests := []struct {
		name    string
		limit   int
		wantErr bool
	}{
		{"below size", dijkstraRefScriptTestSize - 1, true},
		{"at size", dijkstraRefScriptTestSize, false},
		{"unreported", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := dijkstraRefScriptBuilder(t, EraDijkstra, func(pp *backend.ProtocolParameters) {
				withDijkstraRefScriptTiers(pp)
				pp.MaximumReferenceScriptsSize = tt.limit
			}).Complete()
			if tt.wantErr {
				if err == nil || !strings.Contains(err.Error(), "exceeding the protocol maximum") {
					t.Fatalf("Complete error = %v, want size limit error", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Complete failed: %v", err)
			}
		})
	}
}

// TestCompleteDijkstraPartialWithdrawal pins CIP-159 partial withdrawals and
// CIP-181: Apollo neither requires the full reward balance nor a DRep
// delegation for the withdrawing account.
func TestCompleteDijkstraPartialWithdrawal(t *testing.T) {
	cc := dijkstraTestContext(t, dijkstra.MinProtocolVersionDijkstra)
	addr := testAddress(t)
	addTestUtxo(cc, addr, 10_000_000, 0x01, 0)
	account, err := withdrawalRewardAddress(addr)
	if err != nil {
		t.Fatal(err)
	}
	payment, err := NewPayment(validTestAddrBech32, 2_000_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	a, err := New(cc).
		SetWallet(NewExternalWallet(addr)).
		SetTargetEra(EraDijkstra).
		AddPayment(payment).
		AddWithdrawal(account, 1_234_567, nil, nil).
		Complete()
	if err != nil {
		t.Fatal(err)
	}
	txCbor, err := a.GetTxCbor()
	if err != nil {
		t.Fatal(err)
	}
	tx, err := dijkstra.NewDijkstraTransactionFromCbor(txCbor)
	if err != nil {
		t.Fatal(err)
	}
	if len(tx.Body.TxWithdrawals) != 1 {
		t.Fatalf("withdrawals = %d, want 1", len(tx.Body.TxWithdrawals))
	}
	for withdrawalAddr, amount := range tx.Body.TxWithdrawals {
		if withdrawalAddr.String() != account.String() || amount != 1_234_567 {
			t.Fatalf("withdrawal = %s %d, want %s 1234567", withdrawalAddr, amount, account)
		}
	}
	if tx.Body.TxCertificates != nil {
		t.Fatal("a withdrawal must not add a DRep delegation certificate")
	}
}

func TestAddDijkstraProposalPledgeLeverage(t *testing.T) {
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
			Type: uint(common.GovActionTypeParameterChange),
			Action: &dijkstra.DijkstraParameterChangeGovAction{
				Type: uint(common.GovActionTypeParameterChange),
				ParamUpdate: dijkstra.DijkstraProtocolParameterUpdate{
					MaxPledgeLeverage: &cbor.Rat{Rat: big.NewRat(25, 2)},
				},
			},
		},
		PPAnchor: common.GovAnchor{Url: "https://example.com/leverage"},
	}
	a, err := New(cc).
		SetWallet(NewExternalWallet(addr)).
		SetTargetEra(EraDijkstra).
		AddDijkstraProposal(proposal).
		Complete()
	if err != nil {
		t.Fatal(err)
	}
	txCbor, err := a.GetTxCbor()
	if err != nil {
		t.Fatal(err)
	}
	tx, err := dijkstra.NewDijkstraTransactionFromCbor(txCbor)
	if err != nil {
		t.Fatal(err)
	}
	if len(tx.Body.TxProposalProcedures) != 1 {
		t.Fatalf("proposals = %d, want 1", len(tx.Body.TxProposalProcedures))
	}
	action, ok := tx.Body.TxProposalProcedures[0].PPGovAction.Action.(*dijkstra.DijkstraParameterChangeGovAction)
	if !ok {
		t.Fatalf("proposal action = %T, want a Dijkstra parameter change", tx.Body.TxProposalProcedures[0].PPGovAction.Action)
	}
	if leverage := action.ParamUpdate.MaxPledgeLeverage; leverage == nil || leverage.Cmp(big.NewRat(25, 2)) != 0 {
		t.Fatalf("max pledge leverage = %v, want 25/2", leverage)
	}
}

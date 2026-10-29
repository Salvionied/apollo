package localeval

import (
	"strings"
	"testing"

	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/shelley"

	"github.com/Salvionied/apollo/v2/backend"
)

func TestLedgerParamsRejects(t *testing.T) {
	valid := backend.ProtocolParameters{
		ProtocolMajorVersion: 10,
		MaxTxExMem:           "14000000",
		MaxTxExSteps:         "10000000000",
		CostModels:           map[string][]int64{"PlutusV3": {1, 2, 3}},
	}
	if _, _, err := ledgerParams(valid); err != nil {
		t.Fatalf("valid parameters rejected: %v", err)
	}
	tests := map[string]func(*backend.ProtocolParameters){
		"no major version":   func(pp *backend.ProtocolParameters) { pp.ProtocolMajorVersion = 0 },
		"empty memory limit": func(pp *backend.ProtocolParameters) { pp.MaxTxExMem = "" },
		"non-numeric steps":  func(pp *backend.ProtocolParameters) { pp.MaxTxExSteps = "lots" },
		"zero steps":         func(pp *backend.ProtocolParameters) { pp.MaxTxExSteps = "0" },
		"negative memory":    func(pp *backend.ProtocolParameters) { pp.MaxTxExMem = "-1" },
		"unknown language": func(pp *backend.ProtocolParameters) {
			pp.CostModels = map[string][]int64{"PlutusV9": {1}}
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			pp := valid
			pp.CostModels = map[string][]int64{"PlutusV3": {1, 2, 3}}
			mutate(&pp)
			if _, _, err := ledgerParams(pp); err == nil {
				t.Fatal("parameters accepted")
			}
		})
	}
}

func TestLedgerParamsCopiesCostModels(t *testing.T) {
	costs := []int64{1, 2, 3}
	pp := backend.ProtocolParameters{
		ProtocolMajorVersion: 10,
		MaxTxExMem:           "14000000",
		MaxTxExSteps:         "10000000000",
		CostModels: map[string][]int64{
			"PlutusV1": costs, "PlutusV2": costs, "PlutusV3": costs, "PlutusV4": costs,
		},
	}
	ledger, budget, err := ledgerParams(pp)
	if err != nil {
		t.Fatal(err)
	}
	if budget != (common.ExUnits{Memory: 14000000, Steps: 10000000000}) {
		t.Errorf("budget = %+v", budget)
	}
	if ledger.ProtocolVersion.Major != 10 {
		t.Errorf("major = %d", ledger.ProtocolVersion.Major)
	}
	for language := range uint(4) {
		if len(ledger.CostModels[language]) != 3 {
			t.Errorf("language %d cost model = %v", language, ledger.CostModels[language])
		}
	}
	ledger.CostModels[2][0] = 99
	if costs[0] != 1 {
		t.Error("converted cost models alias the caller's")
	}
}

func TestEvaluateRejectsMalformedInput(t *testing.T) {
	tx := buildSyntheticTx(t, compileScript(t, syntheticV3Source))

	for name, txCbor := range map[string][]byte{
		"empty":          nil,
		"not an array":   {0x01},
		"two components": mustEncode(t, []any{map[uint]any{}, map[uint]any{}}),
		"five components": mustEncode(t, []any{
			map[uint]any{}, map[uint]any{}, true, nil, nil,
		}),
		"truncated": tx.txCbor[:len(tx.txCbor)-1],
	} {
		if _, err := Evaluate(txCbor, tx.utxos, tx.params); err == nil {
			t.Errorf("%s: evaluation succeeded", name)
		}
	}

	var nilOutput *shelley.ShelleyTransactionOutput
	badUtxos := []common.Utxo{{Id: tx.utxos[0].Id, Output: nilOutput}}
	_, err := Evaluate(tx.txCbor, badUtxos, tx.params)
	if err == nil || !strings.Contains(err.Error(), "missing transaction output") {
		t.Errorf("typed-nil output: err = %v", err)
	}
}

func TestEvaluateWithoutRedeemers(t *testing.T) {
	tx := buildSyntheticTx(t, compileScript(t, syntheticV3Source))
	body := map[uint]any{
		0: []any{[]any{tx.utxos[0].Id.Id().Bytes(), 0}},
		1: []any{},
		2: uint64(200_000),
	}
	got, err := Evaluate(mustEncode(t, []any{body, map[uint]any{}, true, nil}), tx.utxos, tx.params)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("got %v, want no results", got)
	}
}

package localeval

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/alonzo"
	"github.com/blinklabs-io/gouroboros/ledger/babbage"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/conway"
	"github.com/blinklabs-io/gouroboros/ledger/shelley"

	"github.com/Salvionied/apollo/v2/backend"
)

// The fixtures in testdata are mainnet transaction
// deab9ef3ce7415e492dd6f1eb5ff2920b34bccc0dcee8f4bbc76e97736de8a56, a
// Plutus V3 one-shot mint in slot 196791729 (epoch 653, protocol 11), with
// the outputs it spends and references and the epoch's V3 cost model.
const mainnetMintTxHash = "deab9ef3ce7415e492dd6f1eb5ff2920b34bccc0dcee8f4bbc76e97736de8a56"

func readHexFixture(t *testing.T, name string) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := hex.DecodeString(strings.TrimSpace(string(raw)))
	if err != nil {
		t.Fatalf("decode %s: %v", name, err)
	}
	return decoded
}

func decodeFixture(t *testing.T, name string, v any) {
	t.Helper()
	if _, err := cbor.Decode(readHexFixture(t, name), v); err != nil {
		t.Fatalf("decode %s: %v", name, err)
	}
}

type mainnetMint struct {
	txCbor []byte
	tx     *conway.ConwayTransaction
	utxos  []common.Utxo
	params Params
}

func loadMainnetMint(t *testing.T) mainnetMint {
	t.Helper()
	txCbor := readHexFixture(t, "mainnet_tx_oneshot_mint.hex")
	tx, err := conway.NewConwayTransactionFromCbor(txCbor)
	if err != nil {
		t.Fatal(err)
	}
	if got := tx.Hash().String(); got != mainnetMintTxHash {
		t.Fatalf("fixture hash = %s, want %s", got, mainnetMintTxHash)
	}

	var inputs []shelley.ShelleyTransactionInput
	var outputs []alonzo.AlonzoTransactionOutput
	var refInputs []shelley.ShelleyTransactionInput
	var refOutputs []babbage.BabbageTransactionOutput
	decodeFixture(t, "oneshot_mint_inputs.hex", &inputs)
	decodeFixture(t, "oneshot_mint_input_outputs.hex", &outputs)
	decodeFixture(t, "oneshot_mint_ref_input.hex", &refInputs)
	decodeFixture(t, "oneshot_mint_ref_output.hex", &refOutputs)
	if len(inputs) != len(outputs) || len(refInputs) != len(refOutputs) {
		t.Fatal("fixture inputs and outputs do not pair up")
	}
	utxos := make([]common.Utxo, 0, len(inputs)+len(refInputs))
	for i := range inputs {
		utxos = append(utxos, common.Utxo{Id: inputs[i], Output: &outputs[i]})
	}
	for i := range refInputs {
		utxos = append(utxos, common.Utxo{Id: refInputs[i], Output: &refOutputs[i]})
	}

	raw, err := os.ReadFile(filepath.Join("testdata", "plutus_v3_cost_model_epoch653.json"))
	if err != nil {
		t.Fatal(err)
	}
	var costModel []int64
	if err := json.Unmarshal(raw, &costModel); err != nil {
		t.Fatal(err)
	}

	return mainnetMint{
		txCbor: txCbor,
		tx:     tx,
		utxos:  utxos,
		params: Params{
			ProtocolParams: backend.ProtocolParameters{
				ProtocolMajorVersion: 11,
				MaxTxExMem:           "16500000",
				MaxTxExSteps:         "10000000000",
				CostModels:           map[string][]int64{"PlutusV3": costModel},
			},
			SlotConfig: MainnetSlotConfig,
		},
	}
}

// mainnetMintReference is what cardano-cli 10.14.0.0 reported for the mint
// redeemer with "transaction calculate-plutus-script-cost offline". The
// transaction itself declares a rounded-up budget of 3,000,000 memory and
// 1,000,000,000 steps, so it cannot serve as the reference.
var mainnetMintReference = common.ExUnits{Memory: 375245, Steps: 128734952}

// TestEvaluateMatchesMainnetTransaction checks the evaluator against the
// Haskell ledger on a transaction the chain accepted, including its
// reference input's inline datum and validity interval.
func TestEvaluateMatchesMainnetTransaction(t *testing.T) {
	fixture := loadMainnetMint(t)

	got, err := Evaluate(fixture.txCbor, fixture.utxos, fixture.params)
	if err != nil {
		t.Fatal(err)
	}

	key := common.RedeemerKey{Tag: common.RedeemerTagMint, Index: 0}
	if len(got) != 1 {
		t.Fatalf("got %d results, want 1: %v", len(got), got)
	}
	if got[key] != mainnetMintReference {
		t.Errorf("redeemer %v: got %+v, want %+v", key, got[key], mainnetMintReference)
	}
	// The declared budget must cover what the script used, as it did on chain.
	declared := fixture.tx.Witnesses().Redeemers().Value(0, common.RedeemerTagMint)
	if got[key].Memory > declared.ExUnits.Memory || got[key].Steps > declared.ExUnits.Steps {
		t.Errorf("evaluated %+v exceeds the on-chain budget %+v", got[key], declared.ExUnits)
	}
}

func TestEvaluateMainnetTransactionNeedsEveryInput(t *testing.T) {
	fixture := loadMainnetMint(t)
	for i := range fixture.utxos {
		missing := append(append([]common.Utxo(nil), fixture.utxos[:i]...), fixture.utxos[i+1:]...)
		if _, err := Evaluate(fixture.txCbor, missing, fixture.params); err == nil {
			t.Errorf("evaluating without %s succeeded", fixture.utxos[i].Id.String())
		}
	}
}

func TestEvaluateMainnetTransactionBudget(t *testing.T) {
	fixture := loadMainnetMint(t)
	used, err := Evaluate(fixture.txCbor, fixture.utxos, fixture.params)
	if err != nil {
		t.Fatal(err)
	}
	var steps int64
	for _, units := range used {
		steps = max(steps, units.Steps)
	}

	// A script is held to the protocol's per-transaction limit, so a limit
	// one step below what it needs makes it fail.
	tight := fixture.params
	tight.ProtocolParams.MaxTxExSteps = strconv.FormatInt(steps-1, 10)
	if _, err := Evaluate(fixture.txCbor, fixture.utxos, tight); err == nil {
		t.Fatal("evaluation succeeded with a step limit below the script's use")
	}
	exact := fixture.params
	exact.ProtocolParams.MaxTxExSteps = strconv.FormatInt(steps, 10)
	if _, err := Evaluate(fixture.txCbor, fixture.utxos, exact); err != nil {
		t.Fatalf("evaluation failed with a step limit equal to the script's use: %v", err)
	}
}

func TestEvaluateMainnetTransactionWrongCostModel(t *testing.T) {
	fixture := loadMainnetMint(t)
	params := fixture.params
	params.ProtocolParams.CostModels = nil
	if _, err := Evaluate(fixture.txCbor, fixture.utxos, params); err == nil {
		t.Fatal("evaluation without a V3 cost model succeeded")
	}
}

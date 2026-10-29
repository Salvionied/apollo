package localeval

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/babbage"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/conway"
	"github.com/blinklabs-io/gouroboros/ledger/shelley"
	"github.com/blinklabs-io/plutigo/data"
	"github.com/blinklabs-io/plutigo/syn"

	"github.com/Salvionied/apollo/v2/backend"
)

// The synthetic transaction runs one script per Plutus language. The V3
// script succeeds only when the transaction's validity interval starts at
// the POSIX time the mainnet slot mapping gives syntheticValidFrom, so it
// also checks the slot conversion end to end.
const (
	syntheticValidFrom uint64 = 150_000_000
	syntheticTTL       uint64 = syntheticValidFrom + 1000
	// syntheticValidFromMillis is (Shelley start + slots since) in ms.
	syntheticValidFromMillis int64 = (1596059091 + int64(syntheticValidFrom) - 4492800) * 1000
)

// field returns UPLC selecting field n of the constructor x.
func field(n int, x string) string {
	list := "[(force (force (builtin sndPair))) [(builtin unConstrData) " + x + "]]"
	for range n {
		list = "[(force (builtin tailList)) " + list + "]"
	}
	return "[(force (builtin headList)) " + list + "]"
}

// check returns UPLC that is unit when cond holds and fails otherwise.
func check(cond string) string {
	return "(force [(force (builtin ifThenElse)) " + cond + " (delay (con unit ())) (delay (error))])"
}

// isConstr0 holds when x is a constructor with tag 0, as every script
// context is.
func isConstr0(x string) string {
	return "[(builtin equalsInteger) [(force (force (builtin fstPair))) [(builtin unConstrData) " + x + "]] (con integer 0)]"
}

var (
	// V1 and V2 minting policies take a redeemer and a context.
	syntheticV1Source = "(program 1.0.0 (lam r (lam ctx " + check(isConstr0("ctx")) + ")))"
	syntheticV2Source = "(program 1.0.0 (lam r (lam ctx " + check(isConstr0("[(lam x x) ctx]")) + ")))"
	// The V3 script reads txInfo.validRange.from and requires it to be
	// Finite syntheticValidFromMillis.
	syntheticV3Source = fmt.Sprintf(
		"(program 1.1.0 (lam ctx %s))",
		check(fmt.Sprintf(
			"[(builtin equalsInteger) [(builtin unIData) %s] (con integer %d)]",
			field(0, field(0, field(0, field(7, field(0, "ctx"))))),
			syntheticValidFromMillis,
		)),
	)
	syntheticFailingSource = "(program 1.1.0 (lam ctx (error)))"
)

// compileScript returns the script bytes a witness set carries: the flat
// encoding wrapped in a CBOR byte string.
func compileScript(t *testing.T, source string) []byte {
	t.Helper()
	program, err := syn.Parse(source)
	if err != nil {
		t.Fatalf("parse %s: %v", source, err)
	}
	debruijn, err := syn.NameToDeBruijn(program)
	if err != nil {
		t.Fatal(err)
	}
	flat, err := syn.Encode(debruijn)
	if err != nil {
		t.Fatal(err)
	}
	wrapped, err := cbor.Encode(flat)
	if err != nil {
		t.Fatal(err)
	}
	return wrapped
}

type syntheticTx struct {
	txCbor []byte
	utxos  []common.Utxo
	params Params
	// mintIndex maps a language number to its policy's mint redeemer index.
	mintIndex map[uint]uint32
}

// buildSyntheticTx returns a transaction that spends one input locked by the
// V3 script and mints under a V1, a V2 and a V3 policy. v3 replaces the V3
// script, so a failing script can take its place.
func buildSyntheticTx(t *testing.T, v3 []byte) syntheticTx {
	t.Helper()
	scripts := map[uint][]byte{
		0: compileScript(t, syntheticV1Source),
		1: compileScript(t, syntheticV2Source),
		2: v3,
	}
	hashes := map[uint]common.Blake2b224{
		0: common.PlutusV1Script(scripts[0]).Hash(),
		1: common.PlutusV2Script(scripts[1]).Hash(),
		2: common.PlutusV3Script(scripts[2]).Hash(),
	}

	languages := []uint{0, 1, 2}
	sort.Slice(languages, func(i, j int) bool {
		return bytes.Compare(hashes[languages[i]].Bytes(), hashes[languages[j]].Bytes()) < 0
	})
	mintIndex := make(map[uint]uint32, len(languages))
	mint := make(map[cbor.ByteString]map[cbor.ByteString]int64, len(languages))
	for i, language := range languages {
		mintIndex[language] = uint32(i) //nolint:gosec // three policies
		mint[cbor.NewByteString(hashes[language].Bytes())] = map[cbor.ByteString]int64{
			cbor.NewByteString(nil): 1,
		}
	}

	// An enterprise mainnet address paying to the V3 script.
	scriptAddr := append([]byte{0x71}, hashes[2].Bytes()...)
	var spentId common.Blake2b256
	spentId[0] = 0x5a
	spent, err := shelley.NewShelleyTransactionInput(spentId.String(), 0)
	if err != nil {
		t.Fatal(err)
	}
	spentOutput := babbage.BabbageTransactionOutput{}
	if _, err := cbor.Decode(mustEncode(t, map[uint]any{0: scriptAddr, 1: uint64(10_000_000)}), &spentOutput); err != nil {
		t.Fatal(err)
	}

	// Map-form redeemers, which both Conway and Dijkstra accept, each with
	// the integer 0 as its data and no declared units.
	redeemers := conway.ConwayRedeemers{Redeemers: map[common.RedeemerKey]common.RedeemerValue{}}
	for _, key := range []common.RedeemerKey{
		{Tag: common.RedeemerTagSpend, Index: 0},
		{Tag: common.RedeemerTagMint, Index: mintIndex[0]},
		{Tag: common.RedeemerTagMint, Index: mintIndex[1]},
		{Tag: common.RedeemerTagMint, Index: mintIndex[2]},
	} {
		redeemers.Redeemers[key] = common.RedeemerValue{
			Data: common.Datum{Data: data.NewInteger(big.NewInt(0))},
		}
	}
	body := map[uint]any{
		0:  []any{[]any{spentId.Bytes(), 0}},
		1:  []any{map[uint]any{0: scriptAddr, 1: uint64(9_000_000)}},
		2:  uint64(1_000_000),
		3:  syntheticTTL,
		8:  syntheticValidFrom,
		9:  mint,
		13: []any{[]any{spentId.Bytes(), 0}},
	}
	witnesses := map[uint]any{
		3: []any{[]byte(scripts[0])},
		5: cbor.RawMessage(mustEncode(t, &redeemers)),
		6: []any{[]byte(scripts[1])},
		7: []any{[]byte(scripts[2])},
	}
	txCbor := mustEncode(t, []any{body, witnesses, true, nil})

	return syntheticTx{
		txCbor: txCbor,
		utxos:  []common.Utxo{{Id: spent, Output: &spentOutput}},
		params: Params{
			ProtocolParams: syntheticProtocolParams(t),
			SlotConfig:     MainnetSlotConfig,
		},
		mintIndex: mintIndex,
	}
}

func mustEncode(t *testing.T, v any) []byte {
	t.Helper()
	encoded, err := cbor.Encode(v)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

// syntheticProtocolParams uses cardano-cli's default V1 and V2 cost models
// and the mainnet epoch 653 V3 cost model.
func syntheticProtocolParams(t *testing.T) backend.ProtocolParameters {
	t.Helper()
	costModels := map[string][]int64{}
	for name, file := range map[string]string{
		"PlutusV1": "plutus_v1_cost_model_default.json",
		"PlutusV2": "plutus_v2_cost_model_default.json",
		"PlutusV3": "plutus_v3_cost_model_epoch653.json",
	} {
		raw, err := os.ReadFile(filepath.Join("testdata", file))
		if err != nil {
			t.Fatal(err)
		}
		var costs []int64
		if err := json.Unmarshal(raw, &costs); err != nil {
			t.Fatal(err)
		}
		costModels[name] = costs
	}
	return backend.ProtocolParameters{
		ProtocolMajorVersion: 11,
		MaxTxExMem:           "16500000",
		MaxTxExSteps:         "10000000000",
		CostModels:           costModels,
	}
}

// The units cardano-cli 10.14.0.0 reported for the synthetic transaction
// with "transaction calculate-plutus-script-cost offline". The spend and the
// V3 mint run the same script on the same context, so they cost the same.
// TestWriteCardanoCliInputs writes the files to reproduce them.
var (
	syntheticReferenceV1 = common.ExUnits{Memory: 2766, Steps: 1000721}
	syntheticReferenceV2 = common.ExUnits{Memory: 3066, Steps: 1069721}
	syntheticReferenceV3 = common.ExUnits{Memory: 9338, Steps: 3329517}
)

func TestEvaluateSyntheticMatchesReference(t *testing.T) {
	tx := buildSyntheticTx(t, compileScript(t, syntheticV3Source))
	got, err := Evaluate(tx.txCbor, tx.utxos, tx.params)
	if err != nil {
		t.Fatal(err)
	}
	want := map[common.RedeemerKey]common.ExUnits{
		{Tag: common.RedeemerTagSpend, Index: 0}:              syntheticReferenceV3,
		{Tag: common.RedeemerTagMint, Index: tx.mintIndex[0]}: syntheticReferenceV1,
		{Tag: common.RedeemerTagMint, Index: tx.mintIndex[1]}: syntheticReferenceV2,
		{Tag: common.RedeemerTagMint, Index: tx.mintIndex[2]}: syntheticReferenceV3,
	}
	if len(got) != len(want) {
		t.Fatalf("got %d results, want %d: %v", len(got), len(want), got)
	}
	for key, units := range want {
		if got[key] != units {
			t.Errorf("redeemer %v: got %+v, want %+v", key, got[key], units)
		}
	}
}

func TestEvaluateSyntheticSlotMapping(t *testing.T) {
	// The V3 script fails unless validRange.from is the POSIX time of
	// syntheticValidFrom on mainnet, so a mapping that is off by one slot
	// makes evaluation fail.
	tx := buildSyntheticTx(t, compileScript(t, syntheticV3Source))
	shifted := tx.params
	shifted.SlotConfig.ZeroSlot++
	if _, err := Evaluate(tx.txCbor, tx.utxos, shifted); err == nil {
		t.Fatal("evaluation with a shifted slot mapping succeeded")
	}
	// A validity interval that starts before the mapping covers is an error.
	early := tx.params
	early.SlotConfig.ZeroSlot = syntheticValidFrom + 1
	if _, err := Evaluate(tx.txCbor, tx.utxos, early); err == nil {
		t.Fatal("evaluation with a slot before the mapping's zero slot succeeded")
	}
}

func TestEvaluateFailingScript(t *testing.T) {
	tx := buildSyntheticTx(t, compileScript(t, syntheticFailingSource))
	got, err := Evaluate(tx.txCbor, tx.utxos, tx.params)
	if err == nil {
		t.Fatalf("evaluation of a failing script succeeded: %v", got)
	}
	var failed common.PlutusScriptFailedError
	if !errors.As(err, &failed) {
		t.Fatalf("error %v (%T) does not wrap PlutusScriptFailedError", err, err)
	}
}

func TestEvaluateMissingCostModel(t *testing.T) {
	tx := buildSyntheticTx(t, compileScript(t, syntheticV3Source))
	params := tx.params
	params.ProtocolParams.CostModels = map[string][]int64{
		"PlutusV1": params.ProtocolParams.CostModels["PlutusV1"],
		"PlutusV3": params.ProtocolParams.CostModels["PlutusV3"],
	}
	if _, err := Evaluate(tx.txCbor, tx.utxos, params); err == nil {
		t.Fatal("evaluation of a V2 script without a V2 cost model succeeded")
	}
}

// TestEvaluateDijkstraTopLevel runs the synthetic transaction in Dijkstra's
// envelope. Dijkstra gives V1-V3 scripts the same contexts Conway does, so
// the units must match the Conway reference.
func TestEvaluateDijkstraTopLevel(t *testing.T) {
	tx := buildSyntheticTx(t, compileScript(t, syntheticV3Source))
	var parts []cbor.RawMessage
	if _, err := cbor.Decode(tx.txCbor, &parts); err != nil {
		t.Fatal(err)
	}
	dijkstraTx := mustEncode(t, []any{parts[0], parts[1], nil})
	params := tx.params
	params.ProtocolParams.ProtocolMajorVersion = 12

	got, err := Evaluate(dijkstraTx, tx.utxos, params)
	if err != nil {
		t.Fatal(err)
	}
	want := map[common.RedeemerKey]common.ExUnits{
		{Tag: common.RedeemerTagSpend, Index: 0}:              syntheticReferenceV3,
		{Tag: common.RedeemerTagMint, Index: tx.mintIndex[0]}: syntheticReferenceV1,
		{Tag: common.RedeemerTagMint, Index: tx.mintIndex[1]}: syntheticReferenceV2,
		{Tag: common.RedeemerTagMint, Index: tx.mintIndex[2]}: syntheticReferenceV3,
	}
	if len(got) != len(want) {
		t.Fatalf("got %d results, want %d: %v", len(got), len(want), got)
	}
	for key, units := range want {
		if got[key] != units {
			t.Errorf("redeemer %v: got %+v, want %+v", key, got[key], units)
		}
	}
}

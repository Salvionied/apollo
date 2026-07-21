package utxorpc

import (
	"errors"
	"math/big"
	"reflect"
	"strings"
	"testing"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/conway"
	"github.com/blinklabs-io/plutigo/data"
	cardano "github.com/utxorpc/go-codegen/utxorpc/v1alpha/cardano"
	submit "github.com/utxorpc/go-codegen/utxorpc/v1alpha/submit"
)

func evalTxResponse(errors []*cardano.EvalError, redeemers []*cardano.Redeemer) *submit.EvalTxResponse {
	return &submit.EvalTxResponse{
		Report: &submit.AnyChainEval{
			Chain: &submit.AnyChainEval_Cardano{
				Cardano: &cardano.TxEval{
					Errors:    errors,
					Redeemers: redeemers,
				},
			},
		},
	}
}

func TestCostModelsFromRpcIncludesPlutusV4(t *testing.T) {
	models := costModelsFromRpc(&cardano.CostModels{
		PlutusV4: &cardano.CostModel{Values: []int64{1, 2, 3}},
	})
	got := models["PlutusV4"]
	if !reflect.DeepEqual(got, []int64{1, 2, 3}) {
		t.Fatalf("PlutusV4 cost model = %v, want [1 2 3]", got)
	}
}

func TestEvalTxResponseRejectsSingleEvaluationError(t *testing.T) {
	msg := evalTxResponse([]*cardano.EvalError{
		{Msg: "PreservationOfValue"},
	}, nil)

	_, err := evalTxResponseToExUnits(msg)
	var evaluationErr *EvaluationError
	if !errors.As(err, &evaluationErr) {
		t.Fatalf("expected EvaluationError, got %v", err)
	}
	if !reflect.DeepEqual(evaluationErr.Messages, []string{"PreservationOfValue"}) {
		t.Fatalf("unexpected messages: %#v", evaluationErr.Messages)
	}
}

func TestEvalTxResponsePreservesMultipleEvaluationErrors(t *testing.T) {
	msg := evalTxResponse([]*cardano.EvalError{
		{Msg: "  PreservationOfValue  "},
		{Msg: "ReqSignerMissing"},
		{Msg: " ScriptIntegrityHash "},
		{Msg: "Plutus phase-2 error"},
	}, nil)

	_, err := evalTxResponseToExUnits(msg)
	var evaluationErr *EvaluationError
	if !errors.As(err, &evaluationErr) {
		t.Fatalf("expected EvaluationError, got %v", err)
	}
	want := []string{
		"PreservationOfValue",
		"ReqSignerMissing",
		"ScriptIntegrityHash",
		"Plutus phase-2 error",
	}
	if !reflect.DeepEqual(evaluationErr.Messages, want) {
		t.Fatalf("unexpected messages: got %#v, want %#v", evaluationErr.Messages, want)
	}
}

func TestEvalTxResponseRejectsBlankEvaluationErrors(t *testing.T) {
	msg := evalTxResponse([]*cardano.EvalError{
		nil,
		{Msg: " \t\n "},
	}, nil)

	_, err := evalTxResponseToExUnits(msg)
	var evaluationErr *EvaluationError
	if !errors.As(err, &evaluationErr) {
		t.Fatalf("expected EvaluationError, got %v", err)
	}
	if !reflect.DeepEqual(evaluationErr.Messages, []string{"script evaluation failed without an error message"}) {
		t.Fatalf("unexpected messages: %#v", evaluationErr.Messages)
	}
}

func TestEvalTxResponseRejectsErrorsWithRedeemers(t *testing.T) {
	msg := evalTxResponse(
		[]*cardano.EvalError{{Msg: "Plutus phase-2 error"}},
		[]*cardano.Redeemer{{
			Purpose: cardano.RedeemerPurpose_REDEEMER_PURPOSE_SPEND,
			Index:   0,
			ExUnits: &cardano.ExUnits{Memory: 1700, Steps: 476468},
		}},
	)

	result, err := evalTxResponseToExUnits(msg)
	if result != nil {
		t.Fatalf("expected no partial result, got %#v", result)
	}
	var evaluationErr *EvaluationError
	if !errors.As(err, &evaluationErr) {
		t.Fatalf("expected EvaluationError, got %v", err)
	}
}

func TestEvalTxResponseToExpectedExUnitsPreservesEvaluationError(t *testing.T) {
	msg := evalTxResponse([]*cardano.EvalError{
		{Msg: "ReqSignerMissing"},
	}, []*cardano.Redeemer{{
		Purpose: cardano.RedeemerPurpose_REDEEMER_PURPOSE_SPEND,
		Index:   0,
		ExUnits: &cardano.ExUnits{Memory: 1700, Steps: 476468},
	}})

	expected := map[common.RedeemerKey]struct{}{
		{Tag: common.RedeemerTagSpend, Index: 0}: {},
	}
	result, err := evalTxResponseToExpectedExUnits(msg, expected)
	if result != nil {
		t.Fatalf("expected no partial result, got %#v", result)
	}
	var evaluationErr *EvaluationError
	if !errors.As(err, &evaluationErr) {
		t.Fatalf("expected EvaluationError, got %v", err)
	}
	if !reflect.DeepEqual(evaluationErr.Messages, []string{"ReqSignerMissing"}) {
		t.Fatalf("unexpected messages: %#v", evaluationErr.Messages)
	}
}

func TestEvalTxResponseRejectsDuplicateRedeemerKey(t *testing.T) {
	msg := evalTxResponse(nil, []*cardano.Redeemer{
		{
			Purpose: cardano.RedeemerPurpose_REDEEMER_PURPOSE_SPEND,
			Index:   0,
			ExUnits: &cardano.ExUnits{Memory: 1700, Steps: 476468},
		},
		{
			Purpose: cardano.RedeemerPurpose_REDEEMER_PURPOSE_SPEND,
			Index:   0,
			ExUnits: &cardano.ExUnits{Memory: 250, Steps: 1000},
		},
	})

	if _, err := evalTxResponseToExUnits(msg); err == nil {
		t.Fatal("expected error for duplicate redeemer key")
	}
}

func TestEvalTxResponseToExUnitsRejectsNilResponse(t *testing.T) {
	if _, err := evalTxResponseToExUnits(nil); err == nil {
		t.Fatal("expected error for nil response")
	}
}

func TestEvalTxResponseToExUnitsRejectsMissingReport(t *testing.T) {
	if _, err := evalTxResponseToExUnits(&submit.EvalTxResponse{}); err == nil {
		t.Fatal("expected error for missing report")
	}
}

func TestEvalTxResponseToExUnitsRejectsMissingCardanoReport(t *testing.T) {
	msg := &submit.EvalTxResponse{Report: &submit.AnyChainEval{}}
	if _, err := evalTxResponseToExUnits(msg); err == nil {
		t.Fatal("expected error for missing cardano report")
	}
}

func TestEvalTxResponseToExUnitsRejectsZeroResults(t *testing.T) {
	msg := &submit.EvalTxResponse{
		Report: &submit.AnyChainEval{
			Chain: &submit.AnyChainEval_Cardano{
				Cardano: &cardano.TxEval{},
			},
		},
	}
	if _, err := evalTxResponseToExUnits(msg); err == nil {
		t.Fatal("expected error for zero evaluation results")
	}
}

func TestEvalTxResponseToExUnitsRejectsMissingExUnits(t *testing.T) {
	msg := &submit.EvalTxResponse{
		Report: &submit.AnyChainEval{
			Chain: &submit.AnyChainEval_Cardano{
				Cardano: &cardano.TxEval{
					Redeemers: []*cardano.Redeemer{
						{
							Purpose: cardano.RedeemerPurpose_REDEEMER_PURPOSE_SPEND,
							Index:   0,
						},
					},
				},
			},
		},
	}
	if _, err := evalTxResponseToExUnits(msg); err == nil {
		t.Fatal("expected error for redeemer without ExUnits")
	}
}

func TestEvalTxResponseToExUnitsConvertsRedeemers(t *testing.T) {
	msg := &submit.EvalTxResponse{
		Report: &submit.AnyChainEval{
			Chain: &submit.AnyChainEval_Cardano{
				Cardano: &cardano.TxEval{
					Redeemers: []*cardano.Redeemer{
						{
							Purpose: cardano.RedeemerPurpose_REDEEMER_PURPOSE_SPEND,
							Index:   0,
							ExUnits: &cardano.ExUnits{Memory: 1700, Steps: 476468},
						},
						{
							Purpose: cardano.RedeemerPurpose_REDEEMER_PURPOSE_MINT,
							Index:   1,
							ExUnits: &cardano.ExUnits{Memory: 250, Steps: 1000},
						},
					},
				},
			},
		},
	}
	result, err := evalTxResponseToExpectedExUnits(msg, map[common.RedeemerKey]struct{}{
		{Tag: common.RedeemerTagSpend, Index: 0}: {},
		{Tag: common.RedeemerTagMint, Index: 1}:  {},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result))
	}
	spendKey := common.RedeemerKey{Tag: common.RedeemerTagSpend, Index: 0}
	if eu := result[spendKey]; eu.Memory != 1700 || eu.Steps != 476468 {
		t.Fatalf("unexpected spend budget %+v", eu)
	}
	mintKey := common.RedeemerKey{Tag: common.RedeemerTagMint, Index: 1}
	if eu := result[mintKey]; eu.Memory != 250 || eu.Steps != 1000 {
		t.Fatalf("unexpected mint budget %+v", eu)
	}
}

func TestEvalTxResponseToExUnitsAcceptsZeroBasedSpend(t *testing.T) {
	msg := &submit.EvalTxResponse{
		Report: &submit.AnyChainEval{
			Chain: &submit.AnyChainEval_Cardano{
				Cardano: &cardano.TxEval{
					Redeemers: []*cardano.Redeemer{
						{
							Purpose: cardano.RedeemerPurpose_REDEEMER_PURPOSE_UNSPECIFIED,
							Index:   0,
							ExUnits: &cardano.ExUnits{Memory: 1, Steps: 2},
						},
					},
				},
			},
		},
	}

	result, err := evalTxResponseToExpectedExUnits(msg, map[common.RedeemerKey]struct{}{
		{Tag: common.RedeemerTagSpend, Index: 0}: {},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := result[common.RedeemerKey{Tag: common.RedeemerTagSpend, Index: 0}]; !ok {
		t.Fatalf("expected zero-based purpose 0 to map to spend, got %#v", result)
	}
}

func TestEvalTxResponseToExUnitsAcceptsZeroBasedMint(t *testing.T) {
	msg := &submit.EvalTxResponse{
		Report: &submit.AnyChainEval{
			Chain: &submit.AnyChainEval_Cardano{
				Cardano: &cardano.TxEval{
					Redeemers: []*cardano.Redeemer{
						{
							Purpose: cardano.RedeemerPurpose_REDEEMER_PURPOSE_SPEND,
							Index:   1,
							ExUnits: &cardano.ExUnits{Memory: 1, Steps: 2},
						},
					},
				},
			},
		},
	}

	result, err := evalTxResponseToExpectedExUnits(msg, map[common.RedeemerKey]struct{}{
		{Tag: common.RedeemerTagMint, Index: 1}: {},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := result[common.RedeemerKey{Tag: common.RedeemerTagMint, Index: 1}]; !ok {
		t.Fatalf("expected zero-based purpose 1 to map to mint, got %#v", result)
	}
}

func TestEvalTxResponseToExpectedExUnitsStandardRedeemerPurposes(t *testing.T) {
	tests := []struct {
		name    string
		purpose cardano.RedeemerPurpose
		tag     common.RedeemerTag
	}{
		{"spend", cardano.RedeemerPurpose_REDEEMER_PURPOSE_SPEND, common.RedeemerTagSpend},
		{"mint", cardano.RedeemerPurpose_REDEEMER_PURPOSE_MINT, common.RedeemerTagMint},
		{"cert", cardano.RedeemerPurpose_REDEEMER_PURPOSE_CERT, common.RedeemerTagCert},
		{"reward", cardano.RedeemerPurpose_REDEEMER_PURPOSE_REWARD, common.RedeemerTagReward},
		{"vote", cardano.RedeemerPurpose_REDEEMER_PURPOSE_VOTE, common.RedeemerTagVoting},
		{"propose", cardano.RedeemerPurpose_REDEEMER_PURPOSE_PROPOSE, common.RedeemerTagProposing},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			expected := redeemerKeys(common.RedeemerKey{Tag: tc.tag, Index: 2})
			result, err := evalTxResponseToExpectedExUnits(evalResponse(redeemer(tc.purpose, 2)), expected)
			if err != nil {
				t.Fatal(err)
			}
			if _, ok := result[common.RedeemerKey{Tag: tc.tag, Index: 2}]; !ok {
				t.Fatalf("expected %v key, got %#v", tc.tag, result)
			}
		})
	}
}

func TestEvalTxResponseToExpectedExUnitsMixedPurposeEncodings(t *testing.T) {
	tests := []struct {
		name      string
		redeemers []*cardano.Redeemer
		expected  map[common.RedeemerKey]struct{}
	}{
		{
			name: "standard",
			redeemers: []*cardano.Redeemer{
				redeemer(cardano.RedeemerPurpose_REDEEMER_PURPOSE_SPEND, 0),
				redeemer(cardano.RedeemerPurpose_REDEEMER_PURPOSE_MINT, 1),
			},
			expected: redeemerKeys(
				common.RedeemerKey{Tag: common.RedeemerTagSpend, Index: 0},
				common.RedeemerKey{Tag: common.RedeemerTagMint, Index: 1},
			),
		},
		{
			name: "zero-based",
			redeemers: []*cardano.Redeemer{
				redeemer(cardano.RedeemerPurpose_REDEEMER_PURPOSE_UNSPECIFIED, 0),
				redeemer(cardano.RedeemerPurpose_REDEEMER_PURPOSE_SPEND, 1),
			},
			expected: redeemerKeys(
				common.RedeemerKey{Tag: common.RedeemerTagSpend, Index: 0},
				common.RedeemerKey{Tag: common.RedeemerTagMint, Index: 1},
			),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := evalTxResponseToExpectedExUnits(evalResponse(tc.redeemers...), tc.expected)
			if err != nil {
				t.Fatal(err)
			}
			if !sameRedeemerKeySet(result, tc.expected) {
				t.Fatalf("expected keys %s, got %s", formatRedeemerKeys(tc.expected), formatRedeemerKeys(redeemerKeySet(result)))
			}
		})
	}
}

func TestEvalTxResponseToExpectedExUnitsUsesFallbackForMintReportedAsSpend(t *testing.T) {
	expected := redeemerKeys(common.RedeemerKey{Tag: common.RedeemerTagMint, Index: 7})
	result, err := evalTxResponseToExpectedExUnits(
		evalResponse(redeemer(cardano.RedeemerPurpose_REDEEMER_PURPOSE_SPEND, 7)),
		expected,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := result[common.RedeemerKey{Tag: common.RedeemerTagMint, Index: 7}]; !ok {
		t.Fatalf("expected mint key, got %#v", result)
	}
}

func TestEvalTxResponseToExpectedExUnitsRejectsMismatchedRedeemerKeys(t *testing.T) {
	tests := []struct {
		name      string
		redeemers []*cardano.Redeemer
		expected  map[common.RedeemerKey]struct{}
	}{
		{
			name:      "partial",
			redeemers: []*cardano.Redeemer{redeemer(cardano.RedeemerPurpose_REDEEMER_PURPOSE_SPEND, 0)},
			expected: redeemerKeys(
				common.RedeemerKey{Tag: common.RedeemerTagSpend, Index: 0},
				common.RedeemerKey{Tag: common.RedeemerTagMint, Index: 1},
			),
		},
		{
			name: "extra",
			redeemers: []*cardano.Redeemer{
				redeemer(cardano.RedeemerPurpose_REDEEMER_PURPOSE_SPEND, 0),
				redeemer(cardano.RedeemerPurpose_REDEEMER_PURPOSE_MINT, 1),
			},
			expected: redeemerKeys(common.RedeemerKey{Tag: common.RedeemerTagSpend, Index: 0}),
		},
		{
			name: "duplicate",
			redeemers: []*cardano.Redeemer{
				redeemer(cardano.RedeemerPurpose_REDEEMER_PURPOSE_SPEND, 0),
				redeemer(cardano.RedeemerPurpose_REDEEMER_PURPOSE_SPEND, 0),
			},
			expected: redeemerKeys(common.RedeemerKey{Tag: common.RedeemerTagSpend, Index: 0}),
		},
		{
			name:      "unknown purpose",
			redeemers: []*cardano.Redeemer{redeemer(cardano.RedeemerPurpose(99), 0)},
			expected:  redeemerKeys(common.RedeemerKey{Tag: common.RedeemerTagSpend, Index: 0}),
		},
		{
			name:      "transaction has no redeemers",
			redeemers: []*cardano.Redeemer{redeemer(cardano.RedeemerPurpose_REDEEMER_PURPOSE_SPEND, 0)},
			expected:  redeemerKeys(),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := evalTxResponseToExpectedExUnits(evalResponse(tc.redeemers...), tc.expected); err == nil {
				t.Fatal("expected redeemer key mismatch error")
			}
		})
	}
}

func TestEvalTxResponseToExpectedExUnitsFormatsKeysDeterministically(t *testing.T) {
	expected := redeemerKeys(
		common.RedeemerKey{Tag: common.RedeemerTagMint, Index: 2},
		common.RedeemerKey{Tag: common.RedeemerTagSpend, Index: 1},
	)
	_, err := evalTxResponseToExpectedExUnits(
		evalResponse(redeemer(cardano.RedeemerPurpose_REDEEMER_PURPOSE_REWARD, 9)),
		expected,
	)
	if err == nil {
		t.Fatal("expected redeemer key mismatch error")
	}
	if got, want := err.Error(), "expected keys [0:1, 1:2]"; !strings.Contains(got, want) {
		t.Fatalf("expected deterministic key order %q, got %q", want, got)
	}
}

func TestExpectedRedeemerKeys(t *testing.T) {
	expected := redeemerKeys(
		common.RedeemerKey{Tag: common.RedeemerTagSpend, Index: 1},
		common.RedeemerKey{Tag: common.RedeemerTagVoting, Index: 2},
	)
	tx := conway.ConwayTransaction{
		WitnessSet: conway.ConwayTransactionWitnessSet{
			WsRedeemers: conway.ConwayRedeemers{
				Redeemers: map[common.RedeemerKey]common.RedeemerValue{
					{Tag: common.RedeemerTagSpend, Index: 1}:  testRedeemerValue(),
					{Tag: common.RedeemerTagVoting, Index: 2}: testRedeemerValue(),
				},
			},
		},
		TxIsValid: true,
	}
	txCbor, err := cbor.Encode(tx)
	if err != nil {
		t.Fatal(err)
	}
	actual, err := expectedRedeemerKeys(txCbor)
	if err != nil {
		t.Fatal(err)
	}
	if !sameRedeemerKeySet(actual, expected) {
		t.Fatalf("expected keys %s, got %s", formatRedeemerKeys(expected), formatRedeemerKeys(actual))
	}
}

func TestExpectedRedeemerKeysRejectsMalformedCbor(t *testing.T) {
	if _, err := expectedRedeemerKeys([]byte{0xff}); err == nil {
		t.Fatal("expected malformed transaction CBOR error")
	}
}

func TestEvaluateTxRejectsMalformedTransactionBeforeRequest(t *testing.T) {
	ctx := &UtxoRpcChainContext{}
	if _, err := ctx.EvaluateTx([]byte{0xff}, nil); err == nil {
		t.Fatal("expected malformed transaction CBOR error")
	}
}

func TestEvalTxResponseToExpectedExUnitsRejectsInvalidExUnits(t *testing.T) {
	tests := []struct {
		name     string
		redeemer *cardano.Redeemer
	}{
		{
			name: "missing",
			redeemer: &cardano.Redeemer{
				Purpose: cardano.RedeemerPurpose_REDEEMER_PURPOSE_SPEND,
				Index:   0,
			},
		},
		{
			name: "overflow",
			redeemer: &cardano.Redeemer{
				Purpose: cardano.RedeemerPurpose_REDEEMER_PURPOSE_SPEND,
				Index:   0,
				ExUnits: &cardano.ExUnits{Memory: ^uint64(0), Steps: ^uint64(0)},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			expected := redeemerKeys(common.RedeemerKey{Tag: common.RedeemerTagSpend, Index: 0})
			if _, err := evalTxResponseToExpectedExUnits(evalResponse(tc.redeemer), expected); err == nil {
				t.Fatal("expected invalid ExUnits error")
			}
		})
	}
}

func redeemerKeys(keys ...common.RedeemerKey) map[common.RedeemerKey]struct{} {
	result := make(map[common.RedeemerKey]struct{}, len(keys))
	for _, key := range keys {
		result[key] = struct{}{}
	}
	return result
}

func redeemer(purpose cardano.RedeemerPurpose, index uint32) *cardano.Redeemer {
	return &cardano.Redeemer{
		Purpose: purpose,
		Index:   index,
		ExUnits: &cardano.ExUnits{Memory: 1, Steps: 2},
	}
}

func evalResponse(redeemers ...*cardano.Redeemer) *submit.EvalTxResponse {
	return &submit.EvalTxResponse{
		Report: &submit.AnyChainEval{
			Chain: &submit.AnyChainEval_Cardano{
				Cardano: &cardano.TxEval{Redeemers: redeemers},
			},
		},
	}
}

func testRedeemerValue() common.RedeemerValue {
	return common.RedeemerValue{
		Data:    common.Datum{Data: data.NewInteger(big.NewInt(0))},
		ExUnits: common.ExUnits{Memory: 1, Steps: 1},
	}
}

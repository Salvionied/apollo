package utxorpc

import (
	"errors"
	"reflect"
	"testing"

	"github.com/blinklabs-io/gouroboros/ledger/common"
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
	result, err := evalTxResponseToExUnits(msg)
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

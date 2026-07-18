package utxorpc

import (
	"testing"

	"github.com/blinklabs-io/gouroboros/ledger/common"
	cardano "github.com/utxorpc/go-codegen/utxorpc/v1alpha/cardano"
	submit "github.com/utxorpc/go-codegen/utxorpc/v1alpha/submit"
)

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

	result, err := evalTxResponseToExUnits(msg)
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

	result, err := evalTxResponseToExUnits(msg)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := result[common.RedeemerKey{Tag: common.RedeemerTagMint, Index: 1}]; !ok {
		t.Fatalf("expected zero-based purpose 1 to map to mint, got %#v", result)
	}
}

package apollo

import (
	"math/big"
	"strings"
	"testing"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/conway"
	plutigoData "github.com/blinklabs-io/plutigo/data"

	"github.com/Salvionied/apollo/v2/backend"
	"github.com/Salvionied/apollo/v2/backend/fixed"
)

type capturingEvalContext struct {
	*fixed.FixedChainContext
	lastTxCbor []byte
	result     map[common.RedeemerKey]common.ExUnits
}

func (c *capturingEvalContext) EvaluateTx(txCbor []byte, _ []common.Utxo) (map[common.RedeemerKey]common.ExUnits, error) {
	c.lastTxCbor = append([]byte(nil), txCbor...)
	return c.result, nil
}

func TestEstimateExecutionUnitsUsesLedgerCorrectScriptDataHash(t *testing.T) {
	pp := backend.ProtocolParameters{
		MinFeeConstant:      155381,
		MinFeeCoefficient:   44,
		MaxTxSize:           16384,
		CoinsPerUtxoByte:    "4310",
		CollateralPercent:   150,
		MaxCollateralInputs: 3,
		MaxValSize:          "5000",
		PriceMem:            0.0577,
		PriceStep:           0.0000721,
		MaxTxExMem:          "14000000",
		MaxTxExSteps:        "10000000000",
		KeyDeposits:         "2000000",
		PoolDeposits:        "500000000",
		CostModels: map[string][]int64{
			"PlutusV2": {4, 5, 6},
			"PlutusV3": {7, 8, 9},
		},
	}
	gp := backend.GenesisParameters{NetworkMagic: 1}
	base := fixed.NewFixedChainContext(pp, gp, 0)
	cc := &capturingEvalContext{
		FixedChainContext: base,
		result: map[common.RedeemerKey]common.ExUnits{
			{Tag: common.RedeemerTagMint, Index: 0}: {Memory: 1000, Steps: 2000},
		},
	}

	addr := testAddress(t)
	addTestUtxo(base, addr, 20_000_000, 0x11, 0)
	addTestUtxo(base, addr, 10_000_000, 0x12, 0)

	inlineDatum := common.Datum{Data: plutigoData.NewInteger(big.NewInt(42))}
	script := common.PlutusV2Script([]byte{0x01, 0x02})
	policyHex := strings.Repeat("ab", 28)
	unit := NewUnit(policyHex, "746f6b656e", 1)
	redeemer := common.Datum{Data: plutigoData.NewInteger(big.NewInt(1))}

	a := New(cc).
		SetWallet(NewExternalWallet(addr)).
		AddInputAddress(addr).
		AttachScript(script).
		Mint(unit, &redeemer, nil).
		PayToContract(addr, &inlineDatum, 2_000_000).
		SetTtl(50_000_000)

	if _, err := a.Complete(); err != nil {
		t.Fatalf("Complete failed: %v", err)
	}
	if len(cc.lastTxCbor) == 0 {
		t.Fatal("EvaluateTx was not called")
	}
	if len(a.datums) != 0 {
		t.Fatalf("inline output datum must not become a witness datum, got %d", len(a.datums))
	}

	var prelim conway.ConwayTransaction
	if _, err := cbor.Decode(cc.lastTxCbor, &prelim); err != nil {
		t.Fatalf("decode preliminary EvalTx: %v", err)
	}
	if prelim.Body.TxScriptDataHash == nil {
		t.Fatal("preliminary body missing script data hash")
	}
	if len(prelim.WitnessSet.WsPlutusData.Items()) != 0 {
		t.Fatalf("expected no witness datums, got %d", len(prelim.WitnessSet.WsPlutusData.Items()))
	}

	// Independent ledger preimage: redeemers || (omit empty datums) || lang views.
	redeemerMap := prelim.WitnessSet.WsRedeemers.Redeemers
	if len(redeemerMap) == 0 {
		t.Fatal("expected mint redeemer in preliminary witness set")
	}
	expected, err := ComputeScriptDataHash(redeemerMap, nil, map[string][]int64{
		"PlutusV2": {4, 5, 6},
	})
	if err != nil {
		t.Fatal(err)
	}
	if expected == nil {
		t.Fatal("expected independent script data hash")
	}
	if *prelim.Body.TxScriptDataHash != *expected {
		t.Fatalf("preliminary TxScriptDataHash mismatch:\n got %x\nwant %x",
			prelim.Body.TxScriptDataHash.Bytes(), expected.Bytes())
	}

	// Legacy empty-datum-array preimage must not match.
	redeemerBytes, err := cbor.Encode(redeemerMap)
	if err != nil {
		t.Fatal(err)
	}
	emptyDatums, err := cbor.Encode([]common.Datum{})
	if err != nil {
		t.Fatal(err)
	}
	langViews, err := common.EncodeLangViews(
		map[uint]struct{}{1: {}},
		map[uint][]int64{1: {4, 5, 6}},
	)
	if err != nil {
		t.Fatal(err)
	}
	legacy := common.Blake2b256Hash(append(append(append([]byte{}, redeemerBytes...), emptyDatums...), langViews...))
	if *prelim.Body.TxScriptDataHash == legacy {
		t.Fatal("preliminary hash unexpectedly matches empty-datums legacy preimage")
	}
}

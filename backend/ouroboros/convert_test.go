package ouroboros

import (
	"encoding/hex"
	"math"
	"math/big"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/babbage"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/conway"
	"github.com/blinklabs-io/gouroboros/ledger/dijkstra"
	"github.com/blinklabs-io/gouroboros/ledger/mary"
	"github.com/blinklabs-io/gouroboros/protocol/localstatequery"

	"github.com/Salvionied/apollo/v2/backend"
)

const testAddress = "addr_test1vz2fxv2umyhttkxyxp8x0dlpdt3k6cwng5pxj3jhsydzerspjrlsz"

func rat(num, den int64) *cbor.Rat {
	return &cbor.Rat{Rat: big.NewRat(num, den)}
}

// testConwayParams resembles mainnet's Conway parameters.
func testConwayParams() conway.ConwayProtocolParameters {
	return conway.ConwayProtocolParameters{
		MinFeeA:            44,
		MinFeeB:            155381,
		MaxBlockBodySize:   90112,
		MaxTxSize:          16384,
		MaxBlockHeaderSize: 1100,
		KeyDeposit:         2000000,
		PoolDeposit:        500000000,
		MaxEpoch:           18,
		NOpt:               500,
		A0:                 rat(3, 10),
		Rho:                rat(3, 1000),
		Tau:                rat(1, 5),
		ProtocolVersion:    common.ProtocolParametersProtocolVersion{Major: 10, Minor: 0},
		MinPoolCost:        170000000,
		AdaPerUtxoByte:     4310,
		CostModels: map[uint][]int64{
			0: {100788, 420},
			2: {100788, 420, 1},
		},
		ExecutionCosts: common.ExUnitPrice{
			MemPrice:  rat(577, 10000),
			StepPrice: rat(721, 10000000),
		},
		MaxTxExUnits:               common.ExUnits{Memory: 14000000, Steps: 10000000000},
		MaxBlockExUnits:            common.ExUnits{Memory: 62000000, Steps: 20000000000},
		MaxValueSize:               5000,
		CollateralPercentage:       150,
		MaxCollateralInputs:        3,
		MinFeeRefScriptCostPerByte: rat(15, 1),
	}
}

func TestProtocolParamsFromConway(t *testing.T) {
	p := testConwayParams()
	pp, err := protocolParamsFromLedger(&p)
	if err != nil {
		t.Fatal(err)
	}
	want := backend.ProtocolParameters{
		MinFeeCoefficient:           44,
		MinFeeConstant:              155381,
		MaxBlockSize:                90112,
		MaxTxSize:                   16384,
		MaxBlockHeaderSize:          1100,
		KeyDeposits:                 "2000000",
		PoolDeposits:                "500000000",
		PoolInfluence:               0.3,
		MonetaryExpansion:           0.003,
		TreasuryExpansion:           0.2,
		ProtocolMajorVersion:        10,
		MinPoolCost:                 "170000000",
		PriceMem:                    0.0577,
		PriceStep:                   0.0000721,
		MaxTxExMem:                  "14000000",
		MaxTxExSteps:                "10000000000",
		MaxBlockExMem:               "62000000",
		MaxBlockExSteps:             "20000000000",
		MaxValSize:                  "5000",
		CollateralPercent:           150,
		MaxCollateralInputs:         3,
		CoinsPerUtxoByte:            "4310",
		MaximumReferenceScriptsSize: 200 * 1024,
		MinFeeRefScriptCostPerByte:  15,
	}
	if got := pp.MinFeeRefScriptCostPerByteRational; got == nil || got.Cmp(big.NewRat(15, 1)) != 0 {
		t.Fatalf("reference script price = %v, want 15", got)
	}
	if got := pp.CostModels; len(got) != 2 || len(got["PlutusV1"]) != 2 || len(got["PlutusV3"]) != 3 {
		t.Fatalf("CostModels = %v, want PlutusV1 and PlutusV3", got)
	}
	pp.MinFeeRefScriptCostPerByteRational, pp.CostModels = nil, nil
	if !reflect.DeepEqual(pp, want) {
		t.Fatalf("protocol parameters =\n%+v\nwant\n%+v", pp, want)
	}

	// The conversion copies: changing the node's values afterwards must
	// not reach the converted parameters.
	pp, err = protocolParamsFromLedger(&p)
	if err != nil {
		t.Fatal(err)
	}
	p.CostModels[0][0] = -1
	p.MinFeeRefScriptCostPerByte.SetInt64(-1)
	if pp.CostModels["PlutusV1"][0] != 100788 || pp.MinFeeRefScriptCostPerByteRational.Sign() < 0 {
		t.Fatal("converted parameters share memory with the node's")
	}
	if pp.HasRefScriptTierParams() {
		t.Fatal("Conway parameters must leave the reference-script tier to the ledger defaults")
	}
}

func TestProtocolParamsFromDijkstra(t *testing.T) {
	p := dijkstra.DijkstraProtocolParameters{
		ConwayProtocolParameters: testConwayParams(),
		MaxRefScriptSizePerTx:    300 * 1024,
		RefScriptCostStride:      30000,
		RefScriptCostMultiplier:  rat(3, 2),
	}
	pp, err := protocolParamsFromLedger(&p)
	if err != nil {
		t.Fatal(err)
	}
	if pp.MaximumReferenceScriptsSize != 300*1024 {
		t.Errorf("MaximumReferenceScriptsSize = %d, want %d", pp.MaximumReferenceScriptsSize, 300*1024)
	}
	if pp.MinFeeReferenceScriptsRange != 30000 {
		t.Errorf("MinFeeReferenceScriptsRange = %d, want 30000", pp.MinFeeReferenceScriptsRange)
	}
	if got := pp.MinFeeReferenceScriptsMultiplierRational; got == nil || got.Cmp(big.NewRat(3, 2)) != 0 {
		t.Errorf("reference script multiplier = %v, want 3/2", got)
	}
	if !pp.HasRefScriptTierParams() {
		t.Error("Dijkstra parameters must supply the reference-script tier")
	}
}

func TestProtocolParamsRejectsUnusableValues(t *testing.T) {
	tests := map[string]func(*conway.ConwayProtocolParameters){
		"missing memory price": func(p *conway.ConwayProtocolParameters) { p.ExecutionCosts.MemPrice = nil },
		"missing step price":   func(p *conway.ConwayProtocolParameters) { p.ExecutionCosts.StepPrice = &cbor.Rat{} },
		"negative step price":  func(p *conway.ConwayProtocolParameters) { p.ExecutionCosts.StepPrice = rat(-1, 1) },
		"negative ref price":   func(p *conway.ConwayProtocolParameters) { p.MinFeeRefScriptCostPerByte = rat(-1, 1) },
		"negative ex units":    func(p *conway.ConwayProtocolParameters) { p.MaxTxExUnits.Memory = -1 },
		"oversized tx size":    func(p *conway.ConwayProtocolParameters) { p.MaxTxSize = math.MaxInt32 + 1 },
	}
	for name, corrupt := range tests {
		t.Run(name, func(t *testing.T) {
			p := testConwayParams()
			corrupt(&p)
			if _, err := protocolParamsFromLedger(&p); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
	t.Run("non-positive dijkstra multiplier", func(t *testing.T) {
		p := dijkstra.DijkstraProtocolParameters{
			ConwayProtocolParameters: testConwayParams(),
			RefScriptCostMultiplier:  rat(0, 1),
		}
		if _, err := protocolParamsFromLedger(&p); err == nil {
			t.Fatal("expected an error")
		}
	})
}

func TestProtocolParamsRejectsPreConwayEra(t *testing.T) {
	_, err := protocolParamsFromLedger(&babbage.BabbageProtocolParameters{})
	if err == nil || !strings.Contains(err.Error(), "Conway") {
		t.Fatalf("got %v, want an error naming the Conway requirement", err)
	}
	if _, err := protocolParamsFromLedger(nil); err == nil {
		t.Fatal("expected an error for no parameters")
	}
	if _, err := protocolParamsFromLedger((*conway.ConwayProtocolParameters)(nil)); err == nil {
		t.Fatal("expected an error for typed-nil parameters")
	}
}

func TestCostModelsName(t *testing.T) {
	got := costModelsFromLedger(map[uint][]int64{0: {1}, 1: {2}, 2: {3}, 3: {4}})
	for _, name := range []string{"PlutusV1", "PlutusV2", "PlutusV3", "PlutusV4"} {
		if _, ok := got[name]; !ok {
			t.Errorf("missing %s in %v", name, got)
		}
	}
}

// mainnetGenesis is mainnet's Shelley genesis as the node encodes it: system
// start 2017-09-23T21:44:51Z, a one-second slot in microseconds.
func mainnetGenesis() *localstatequery.GenesisConfigResult {
	return &localstatequery.GenesisConfigResult{
		Start: localstatequery.SystemStartResult{
			Year:        *big.NewInt(2017),
			Day:         266,
			Picoseconds: *big.NewInt((21*3600 + 44*60 + 51) * 1_000_000_000_000),
		},
		NetworkMagic:      764824073,
		NetworkId:         1,
		ActiveSlotsCoeff:  []any{uint64(1), uint64(20)},
		SecurityParam:     2160,
		EpochLength:       432000,
		SlotsPerKESPeriod: 129600,
		MaxKESEvolutions:  62,
		SlotLength:        1_000_000,
		UpdateQuorum:      5,
		MaxLovelaceSupply: 45000000000000000,
	}
}

func TestGenesisParamsFromLedger(t *testing.T) {
	gp, err := genesisParamsFromLedger(mainnetGenesis())
	if err != nil {
		t.Fatal(err)
	}
	want := backend.GenesisParameters{
		ActiveSlotsCoefficient: 0.05,
		UpdateQuorum:           5,
		MaxLovelaceSupply:      "45000000000000000",
		NetworkMagic:           764824073,
		EpochLength:            432000,
		SystemStart:            time.Date(2017, 9, 23, 21, 44, 51, 0, time.UTC).Unix(),
		SlotsPerKesPeriod:      129600,
		SlotLength:             1,
		MaxKesEvolutions:       62,
		SecurityParam:          2160,
	}
	if gp != want {
		t.Fatalf("genesis =\n%+v\nwant\n%+v", gp, want)
	}

	// The coefficient's parts may arrive as any CBOR integer type.
	g := mainnetGenesis()
	g.ActiveSlotsCoeff = []any{big.NewInt(1), int64(20)}
	if gp, err := genesisParamsFromLedger(g); err != nil || gp.ActiveSlotsCoefficient != 0.05 {
		t.Fatalf("got (%v, %v), want coefficient 0.05", gp.ActiveSlotsCoefficient, err)
	}
}

func TestGenesisParamsRejectsMalformed(t *testing.T) {
	tests := map[string]func(*localstatequery.GenesisConfigResult){
		"coefficient above one":    func(g *localstatequery.GenesisConfigResult) { g.ActiveSlotsCoeff = []any{uint64(21), uint64(20)} },
		"zero coefficient":         func(g *localstatequery.GenesisConfigResult) { g.ActiveSlotsCoeff = []any{uint64(0), uint64(20)} },
		"zero denominator":         func(g *localstatequery.GenesisConfigResult) { g.ActiveSlotsCoeff = []any{uint64(1), uint64(0)} },
		"coefficient not a pair":   func(g *localstatequery.GenesisConfigResult) { g.ActiveSlotsCoeff = []any{uint64(1)} },
		"coefficient not integers": func(g *localstatequery.GenesisConfigResult) { g.ActiveSlotsCoeff = []any{"1", "20"} },
		"zero slot length":         func(g *localstatequery.GenesisConfigResult) { g.SlotLength = 0 },
		"sub-second slot length":   func(g *localstatequery.GenesisConfigResult) { g.SlotLength = 200_000 },
		"fractional slot seconds":  func(g *localstatequery.GenesisConfigResult) { g.SlotLength = 1_500_000 },
		"zero epoch length":        func(g *localstatequery.GenesisConfigResult) { g.EpochLength = 0 },
		"negative max supply":      func(g *localstatequery.GenesisConfigResult) { g.MaxLovelaceSupply = -1 },
		"day zero":                 func(g *localstatequery.GenesisConfigResult) { g.Start.Day = 0 },
		"day past year end":        func(g *localstatequery.GenesisConfigResult) { g.Start.Day = 366 },
		"year before unix epoch":   func(g *localstatequery.GenesisConfigResult) { g.Start.Year = *big.NewInt(1969) },
		"time of day past midnight": func(g *localstatequery.GenesisConfigResult) {
			g.Start.Picoseconds = *big.NewInt(86_400 * 1_000_000_000_000)
		},
		"negative time of day": func(g *localstatequery.GenesisConfigResult) { g.Start.Picoseconds = *big.NewInt(-1) },
	}
	for name, corrupt := range tests {
		t.Run(name, func(t *testing.T) {
			g := mainnetGenesis()
			corrupt(g)
			if _, err := genesisParamsFromLedger(g); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
	if _, err := genesisParamsFromLedger(nil); err == nil {
		t.Fatal("expected an error for no genesis")
	}
}

func TestGenesisLeapDay(t *testing.T) {
	g := mainnetGenesis()
	g.Start = localstatequery.SystemStartResult{Year: *big.NewInt(2024), Day: 366}
	gp, err := genesisParamsFromLedger(g)
	if err != nil {
		t.Fatal(err)
	}
	if want := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC).Unix(); gp.SystemStart != want {
		t.Fatalf("SystemStart = %d, want %d", gp.SystemStart, want)
	}
}

func TestUtxosAreSortedAndBounded(t *testing.T) {
	addr, err := common.NewAddress(testAddress)
	if err != nil {
		t.Fatal(err)
	}
	out := babbage.BabbageTransactionOutput{OutputAddress: addr, OutputAmount: mary.MaryTransactionOutputValue{Amount: 5}}
	var a, b common.Blake2b256
	a[0], b[0] = 1, 2
	utxos, err := utxosFromLedger(map[localstatequery.UtxoId]babbage.BabbageTransactionOutput{
		{Hash: b, Idx: 0}: out,
		{Hash: a, Idx: 3}: out,
		{Hash: a, Idx: 1}: out,
	})
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, 0, len(utxos))
	for _, u := range utxos {
		got = append(got, u.Id.String())
	}
	want := []string{a.String() + "#1", a.String() + "#3", b.String() + "#0"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("order = %v, want %v", got, want)
	}

	if _, err := utxosFromLedger(map[localstatequery.UtxoId]babbage.BabbageTransactionOutput{{Idx: -1}: out}); err == nil {
		t.Fatal("expected an error for a negative output index")
	}
}

func TestTxHash(t *testing.T) {
	hash, err := txHash(testTx())
	if err != nil {
		t.Fatal(err)
	}
	if want := common.Blake2b256Hash([]byte{0xa0}); hash != want {
		t.Fatalf("txHash = %s, want %s", hash, want)
	}
	for _, bad := range []string{"", "a0", "81a0", "ff"} {
		raw, err := hex.DecodeString(bad)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := txHash(raw); err == nil {
			t.Errorf("txHash(%s) succeeded, want an error", bad)
		}
	}
}

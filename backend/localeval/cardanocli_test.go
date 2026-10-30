package localeval

import (
	"encoding/hex"
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/plutigo/data"
)

// TestWriteCardanoCliInputs writes the files cardano-cli needs to evaluate
// the test transactions with the Haskell ledger, which is how the reference
// units in these tests were obtained:
//
//	LOCALEVAL_CARDANO_CLI_DIR=/some/dir go test ./backend/localeval -run TestWriteCardanoCliInputs
//	cd /some/dir/<case> && cardano-cli conway transaction \
//	  calculate-plutus-script-cost offline --start-time-posix 1506203091 \
//	  --era-history-file era-history.json --utxo-file utxo.json \
//	  --protocol-params-file pparams.json --tx-file tx.json
func TestWriteCardanoCliInputs(t *testing.T) {
	dir := os.Getenv("LOCALEVAL_CARDANO_CLI_DIR")
	if dir == "" {
		t.Skip("set LOCALEVAL_CARDANO_CLI_DIR to write cardano-cli inputs")
	}
	mainnet := loadMainnetMint(t)
	synthetic := buildSyntheticTx(t, compileScript(t, syntheticV3Source))
	failing := buildSyntheticTx(t, compileScript(t, syntheticFailingSource))
	cases := map[string]struct {
		txCbor []byte
		utxos  []common.Utxo
		params Params
	}{
		"mainnet":   {mainnet.txCbor, mainnet.utxos, mainnet.params},
		"synthetic": {synthetic.txCbor, synthetic.utxos, synthetic.params},
		"failing":   {failing.txCbor, failing.utxos, failing.params},
	}
	for name, c := range cases {
		out := filepath.Join(dir, name)
		if err := os.MkdirAll(out, 0o750); err != nil { //nolint:gosec // a directory the developer chose
			t.Fatal(err)
		}
		write := func(file string, v any) {
			encoded, err := json.MarshalIndent(v, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(out, file), encoded, 0o600); err != nil { //nolint:gosec // a directory the developer chose
				t.Fatal(err)
			}
		}
		write("tx.json", textEnvelope("Tx ConwayEra", c.txCbor))
		write("utxo.json", cliUtxo(t, c.utxos))
		write("pparams.json", cliProtocolParams(c.params))
		write("era-history.json", textEnvelope("EraHistory", mainnetEraHistory(t)))
	}
}

func textEnvelope(kind string, cborBytes []byte) map[string]string {
	return map[string]string{"type": kind, "description": "", "cborHex": hex.EncodeToString(cborBytes)}
}

// mainnetEraHistory encodes mainnet's hard-fork history the way a node
// answers an era-history query: Byron's 20-second slots, then one-second
// slots from each later era's first epoch.
func mainnetEraHistory(t *testing.T) []byte {
	t.Helper()
	type list = []any
	picos := func(seconds int64) *big.Int {
		return new(big.Int).Mul(big.NewInt(seconds), big.NewInt(1_000_000_000_000))
	}
	bound := func(seconds int64, slot, epoch uint64) list { return list{picos(seconds), slot, epoch} }
	safeZone := func(slots uint64) list { return list{uint8(0), slots, list{uint8(0)}} }
	starts := []struct {
		seconds     int64
		slot, epoch uint64
	}{
		{89856000, 4492800, 208},    // Shelley
		{101952000, 16588800, 236},  // Allegra
		{108432000, 23068800, 251},  // Mary
		{125280000, 39916800, 290},  // Alonzo
		{157680000, 72316800, 365},  // Babbage
		{219024000, 133660800, 507}, // Conway
	}
	eras := make(list, 0, 1+len(starts))
	eras = append(eras, list{bound(0, 0, 0), bound(89856000, 4492800, 208), list{21600, 20000, safeZone(4320), 4320}})
	for i, start := range starts {
		var end any // the last era is unbounded
		if i+1 < len(starts) {
			next := starts[i+1]
			end = bound(next.seconds, next.slot, next.epoch)
		}
		eras = append(eras, list{bound(start.seconds, start.slot, start.epoch), end, list{432000, 1000, safeZone(129600), 129600}})
	}
	return mustEncode(t, eras)
}

func cliUtxo(t *testing.T, utxos []common.Utxo) map[string]any {
	t.Helper()
	result := make(map[string]any, len(utxos))
	for _, utxo := range utxos {
		output := utxo.Output
		value := map[string]any{"lovelace": output.Amount()}
		if assets := output.Assets(); assets != nil {
			for _, policy := range assets.Policies() {
				names := map[string]any{}
				for _, name := range assets.Assets(policy) {
					names[hex.EncodeToString(name)] = assets.Asset(policy, name)
				}
				value[hex.EncodeToString(policy.Bytes())] = names
			}
		}
		entry := map[string]any{
			"address":         output.Address().String(),
			"value":           value,
			"datum":           nil,
			"datumhash":       nil,
			"referenceScript": nil,
		}
		if datum := output.Datum(); datum != nil {
			entry["inlineDatum"] = scriptDataJSON(t, datum.Data)
			entry["inlineDatumRaw"] = hex.EncodeToString(datum.Cbor())
			entry["inlineDatumhash"] = output.DatumHash().String()
		}
		result[utxo.Id.String()] = entry
	}
	return result
}

// scriptDataJSON renders Plutus data in cardano-cli's detailed schema.
func scriptDataJSON(t *testing.T, d data.PlutusData) any {
	t.Helper()
	switch v := d.(type) {
	case *data.Constr:
		fields := make([]any, 0, len(v.Fields))
		for _, f := range v.Fields {
			fields = append(fields, scriptDataJSON(t, f))
		}
		return map[string]any{"constructor": v.Tag, "fields": fields}
	case *data.Map:
		pairs := make([]any, 0, len(v.Pairs))
		for _, pair := range v.Pairs {
			pairs = append(pairs, map[string]any{"k": scriptDataJSON(t, pair[0]), "v": scriptDataJSON(t, pair[1])})
		}
		return map[string]any{"map": pairs}
	case *data.List:
		items := make([]any, 0, len(v.Items))
		for _, item := range v.Items {
			items = append(items, scriptDataJSON(t, item))
		}
		return map[string]any{"list": items}
	case *data.Integer:
		return map[string]any{"int": v.Inner}
	case *data.ByteString:
		return map[string]any{"bytes": hex.EncodeToString(v.Inner)}
	default:
		t.Fatalf("unexpected Plutus data %T", d)
		return nil
	}
}

// cliProtocolParams is a complete Conway parameter set in cardano-cli's
// format. Only the version, cost models and execution limits affect script
// costs; the rest are mainnet values so the file parses.
func cliProtocolParams(params Params) map[string]any {
	pp := params.ProtocolParams
	thresholds := func(names ...string) map[string]any {
		m := map[string]any{}
		for _, name := range names {
			m[name] = 0.67
		}
		return m
	}
	return map[string]any{
		"collateralPercentage":   150,
		"committeeMaxTermLength": 146,
		"committeeMinSize":       7,
		"costModels":             pp.CostModels,
		"dRepActivity":           20,
		"dRepDeposit":            500000000,
		"dRepVotingThresholds": thresholds("committeeNoConfidence", "committeeNormal", "hardForkInitiation",
			"motionNoConfidence", "ppEconomicGroup", "ppGovGroup", "ppNetworkGroup", "ppTechnicalGroup",
			"treasuryWithdrawal", "updateToConstitution"),
		"executionUnitPrices":        map[string]any{"priceMemory": 0.0577, "priceSteps": 7.21e-5},
		"govActionDeposit":           uint64(100000000000),
		"govActionLifetime":          6,
		"maxBlockBodySize":           90112,
		"maxBlockExecutionUnits":     map[string]any{"memory": 72000000, "steps": uint64(20000000000)},
		"maxBlockHeaderSize":         1100,
		"maxCollateralInputs":        3,
		"maxTxExecutionUnits":        map[string]any{"memory": json.RawMessage(pp.MaxTxExMem), "steps": json.RawMessage(pp.MaxTxExSteps)},
		"maxTxSize":                  16384,
		"maxValueSize":               5000,
		"minFeeRefScriptCostPerByte": 15,
		"minPoolCost":                170000000,
		"monetaryExpansion":          0.003,
		"poolPledgeInfluence":        0.3,
		"poolRetireMaxEpoch":         18,
		"poolVotingThresholds": thresholds("committeeNoConfidence", "committeeNormal", "hardForkInitiation",
			"motionNoConfidence", "ppSecurityGroup"),
		"protocolVersion":     map[string]any{"major": pp.ProtocolMajorVersion, "minor": 0},
		"stakeAddressDeposit": 2000000,
		"stakePoolDeposit":    500000000,
		"stakePoolTargetNum":  500,
		"treasuryCut":         0.2,
		"txFeeFixed":          155381,
		"txFeePerByte":        44,
		"utxoCostPerByte":     4310,
	}
}

package ogmios

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/SundaeSwap-finance/kugo"
	ogmigo "github.com/SundaeSwap-finance/ogmigo/v6"
	"github.com/SundaeSwap-finance/ogmigo/v6/ouroboros/chainsync/num"
	"github.com/SundaeSwap-finance/ogmigo/v6/ouroboros/shared"
	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/babbage"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/mary"
	"github.com/blinklabs-io/gouroboros/ledger/shelley"

	"github.com/Salvionied/apollo/v2/backend"
)

func TestOgmiosCapabilitiesWithoutKupo(t *testing.T) {
	ctx := testChainContext(t, Config{OgmiosEndpoint: testOgmiosEndpoint})
	if !backend.Supports(ctx, backend.CapabilityEvaluateTx|backend.CapabilityUtxoByRef) {
		t.Fatal("expected Ogmios-supported capabilities")
	}
	if backend.Supports(ctx, backend.CapabilityUtxos|backend.CapabilityScriptCbor) {
		t.Fatal("Ogmios without Kupo reported Kupo capabilities")
	}

	tests := []struct {
		name       string
		capability backend.Capability
		call       func() error
	}{
		{"utxos", backend.CapabilityUtxos, func() error { _, err := ctx.Utxos(testAddress(t)); return err }},
		{"script", backend.CapabilityScriptCbor, func() error { _, err := ctx.ScriptCbor(common.Blake2b224{}); return err }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.call()
			if !errors.Is(err, backend.ErrUnsupported) {
				t.Fatalf("expected ErrUnsupported, got %v", err)
			}
			var unsupported *backend.UnsupportedError
			if !errors.As(err, &unsupported) || unsupported.Capability != test.capability {
				t.Fatalf("unexpected unsupported error: %#v", err)
			}
		})
	}
}

func testAddress(t *testing.T) common.Address {
	t.Helper()
	var raw [57]byte
	raw[0] = 0x00
	raw[1] = 0xAA
	raw[29] = 0xBB
	addr, err := common.NewAddressFromBytes(raw[:])
	if err != nil {
		t.Fatal(err)
	}
	return addr
}

func TestSharedValueToUtxoRejectsNegativeLovelace(t *testing.T) {
	value := shared.Value{
		shared.AdaPolicy: {
			shared.AdaAsset: num.Int64(-1),
		},
	}
	if _, err := sharedValueToUtxo(common.Blake2b256{}, 0, value, testAddress(t)); err == nil {
		t.Fatal("expected negative lovelace error")
	}
}

func TestSharedValueToUtxoRejectsNegativeAssetQuantity(t *testing.T) {
	value := shared.Value{
		shared.AdaPolicy: {
			shared.AdaAsset: num.Int64(1000000),
		},
		"00000000000000000000000000000000000000000000000000000001": {
			"544f4b454e": num.Int64(-1),
		},
	}
	if _, err := sharedValueToUtxo(common.Blake2b256{}, 0, value, testAddress(t)); err == nil {
		t.Fatal("expected negative asset quantity error")
	}
}

const testTxHashHex = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func testMatch(datumType, datumHashHex string) kugo.Match {
	return kugo.Match{
		TransactionID: testTxHashHex,
		OutputIndex:   0,
		DatumType:     datumType,
		DatumHash:     datumHashHex,
		Value: kugo.Value{
			shared.AdaPolicy: {
				shared.AdaAsset: num.Int64(1000000),
			},
		},
	}
}

// kupoDatumServer serves the Kupo /v1/datums/{hash} endpoint, returning the
// given datum CBOR hex for any requested hash.
func kupoDatumServer(t *testing.T, datumCborHex string) (*kugo.Client, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"datum": datumCborHex})
	}))
	t.Cleanup(server.Close)
	return kugo.New(kugo.WithEndpoint(server.URL)), server
}

func TestMatchToUtxoFetchesAndVerifiesInlineDatum(t *testing.T) {
	datumCbor := []byte{0x18, 0x2a} // CBOR encoding of integer 42
	datumHash := common.Blake2b256Hash(datumCbor)
	client, _ := kupoDatumServer(t, hex.EncodeToString(datumCbor))

	match := testMatch("inline", hex.EncodeToString(datumHash.Bytes()))
	utxo, err := matchToUtxo(t.Context(), match, testAddress(t), client)
	if err != nil {
		t.Fatal(err)
	}
	output, ok := utxo.Output.(*babbage.BabbageTransactionOutput)
	if !ok {
		t.Fatalf("unexpected output type %T", utxo.Output)
	}
	if output.Datum() == nil {
		t.Fatal("expected inline datum to be populated")
	}
	if got := output.Datum().Cbor(); !bytes.Equal(got, datumCbor) {
		t.Fatalf("inline datum CBOR = %x, want %x", got, datumCbor)
	}
}

func TestMatchToUtxoRejectsInlineDatumHashMismatch(t *testing.T) {
	// Kupo returns datum bytes that do not hash to the claimed datum hash.
	client, _ := kupoDatumServer(t, "182b")
	datumHash := common.Blake2b256Hash([]byte{0x18, 0x2a})

	match := testMatch("inline", hex.EncodeToString(datumHash.Bytes()))
	if _, err := matchToUtxo(t.Context(), match, testAddress(t), client); err == nil {
		t.Fatal("expected inline datum hash mismatch error")
	}
}

func TestMatchToUtxoRejectsMissingInlineDatum(t *testing.T) {
	client, _ := kupoDatumServer(t, "")
	datumHash := common.Blake2b256Hash([]byte{0x18, 0x2a})

	match := testMatch("inline", hex.EncodeToString(datumHash.Bytes()))
	if _, err := matchToUtxo(t.Context(), match, testAddress(t), client); err == nil {
		t.Fatal("expected missing inline datum error")
	}
}

func TestMatchToUtxoHashDatumProducesHashOption(t *testing.T) {
	datumHash := common.Blake2b256Hash([]byte{0x18, 0x2a})
	datumHashHex := hex.EncodeToString(datumHash.Bytes())

	// No datum fetcher needed for hash datums; nil must not be dereferenced.
	match := testMatch("hash", datumHashHex)
	utxo, err := matchToUtxo(t.Context(), match, testAddress(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	output, ok := utxo.Output.(*babbage.BabbageTransactionOutput)
	if !ok {
		t.Fatalf("unexpected output type %T", utxo.Output)
	}
	if output.Datum() != nil {
		t.Fatal("hash datum must not produce an inline datum")
	}
	gotHash := output.DatumHash()
	if gotHash == nil {
		t.Fatal("expected datum hash to be populated")
	}
	if got := hex.EncodeToString(gotHash.Bytes()); got != datumHashHex {
		t.Fatalf("datum hash = %s, want %s", got, datumHashHex)
	}
}

func TestMatchToUtxoRejectsUnknownDatumType(t *testing.T) {
	datumHash := common.Blake2b256Hash([]byte{0x18, 0x2a})
	match := testMatch("bogus", hex.EncodeToString(datumHash.Bytes()))
	if _, err := matchToUtxo(t.Context(), match, testAddress(t), nil); err == nil {
		t.Fatal("expected unsupported datum type error")
	}
}

func TestKupoScriptToScriptRefVerifiesHash(t *testing.T) {
	scriptBytes := []byte{0x01, 0x02, 0x03}
	script := kugo.Script{
		Language: kugo.ScriptLanguagePlutusV2,
		Script:   hex.EncodeToString(scriptBytes),
	}
	correctHash := hex.EncodeToString(common.PlutusV2Script(scriptBytes).Hash().Bytes())

	ref, err := kupoScriptToScriptRef(script, correctHash)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := ref.Script.(common.PlutusV2Script); !ok {
		t.Fatalf("expected PlutusV2 script, got %T", ref.Script)
	}

	wrongHash := hex.EncodeToString(common.PlutusV1Script(scriptBytes).Hash().Bytes())
	if _, err := kupoScriptToScriptRef(script, wrongHash); err == nil {
		t.Fatal("expected script hash mismatch error")
	}
}

func TestKupoScriptToScriptRefPlutusV4(t *testing.T) {
	scriptBytes := []byte{0x01, 0x02, 0x03}
	script := kugo.Script{
		Language: kupoScriptLanguagePlutusV4,
		Script:   hex.EncodeToString(scriptBytes),
	}
	correctHash := hex.EncodeToString(common.PlutusV4Script(scriptBytes).Hash().Bytes())

	ref, err := kupoScriptToScriptRef(script, correctHash)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := ref.Script.(common.PlutusV4Script); !ok {
		t.Fatalf("expected PlutusV4 script, got %T", ref.Script)
	}
}

func TestOgmiosScriptToScriptRefPlutusV4(t *testing.T) {
	scriptBytes := []byte{0x01, 0x02, 0x03}
	ref, err := ogmiosScriptToScriptRef(json.RawMessage(`{"language":"plutus:v4","cbor":"010203"}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := ref.Script.(common.PlutusV4Script); !ok {
		t.Fatalf("expected PlutusV4 script, got %T", ref.Script)
	}
	if got := ref.Script.RawScriptBytes(); !bytes.Equal(got, scriptBytes) {
		t.Fatalf("script bytes = %x, want %x", got, scriptBytes)
	}
}

func TestOgmiosCostModelKeyPlutusV4(t *testing.T) {
	if got := ogmiosCostModelKey("plutus:v4"); got != "PlutusV4" {
		t.Fatalf("cost model key = %q, want PlutusV4", got)
	}
}

func TestEvaluateResponseToExUnitsRejectsZeroResults(t *testing.T) {
	if _, err := evaluateResponseToExUnits(nil); err == nil {
		t.Fatal("expected error for nil response")
	}
	if _, err := evaluateResponseToExUnits(&ogmigo.EvaluateTxResponse{}); err == nil {
		t.Fatal("expected error for zero evaluation results")
	}
}

func TestEvaluateResponseToExUnitsReportsErrors(t *testing.T) {
	resp := &ogmigo.EvaluateTxResponse{
		Error: &ogmigo.EvaluateTxError{Code: 3010, Message: "script failure"},
	}
	if _, err := evaluateResponseToExUnits(resp); err == nil {
		t.Fatal("expected evaluation error")
	}
}

func TestEvaluateResponseToExUnitsConvertsResults(t *testing.T) {
	resp := &ogmigo.EvaluateTxResponse{
		ExUnits: []ogmigo.ExUnits{
			{
				Validator: ogmigo.Validator{Purpose: "spend", Index: 0},
				Budget:    ogmigo.ExUnitsBudget{Memory: 1700, Cpu: 476468},
			},
		},
	}
	result, err := evaluateResponseToExUnits(resp)
	if err != nil {
		t.Fatal(err)
	}
	key := common.RedeemerKey{Tag: common.RedeemerTagSpend, Index: 0}
	if eu := result[key]; eu.Memory != 1700 || eu.Steps != 476468 {
		t.Fatalf("unexpected budget %+v", eu)
	}
}

// sampleCommonUtxo builds a resolved gouroboros UTxO for additional-UTxO
// conversion tests: a known tx ref, address, lovelace coin, one native asset,
// and a PlutusV2 reference script.
func sampleCommonUtxo(t *testing.T) common.Utxo {
	t.Helper()
	var txId common.Blake2b256
	for i := range txId {
		txId[i] = 0x11
	}
	input := shelley.ShelleyTransactionInput{TxId: txId, OutputIndex: 3}

	var policyId common.Blake2b224
	for i := range policyId {
		policyId[i] = 0xAB
	}
	assetName := []byte("TOKEN")
	assetData := map[common.Blake2b224]map[cbor.ByteString]*big.Int{
		policyId: {cbor.NewByteString(assetName): big.NewInt(42)},
	}
	ma := common.NewMultiAsset[common.MultiAssetTypeOutput](assetData)

	output := babbage.BabbageTransactionOutput{
		OutputAddress: testAddress(t),
		OutputAmount: mary.MaryTransactionOutputValue{
			Amount: 1_500_000,
			Assets: &ma,
		},
		TxOutScriptRef: &common.ScriptRef{
			Type:   common.ScriptRefTypePlutusV2,
			Script: common.PlutusV2Script([]byte{0x49, 0x48, 0x01, 0x00}),
		},
	}
	return common.Utxo{Id: input, Output: &output}
}

func TestCommonUtxoToShared(t *testing.T) {
	su, err := commonUtxoToShared(sampleCommonUtxo(t))
	if err != nil {
		t.Fatal(err)
	}

	wantTxID := strings.Repeat("11", 32)
	if su.Transaction.ID != wantTxID {
		t.Fatalf("transaction id = %q, want %q", su.Transaction.ID, wantTxID)
	}
	if su.Index != 3 {
		t.Fatalf("index = %d, want 3", su.Index)
	}
	if su.Address != testAddress(t).String() {
		t.Fatalf("address = %q, want %q", su.Address, testAddress(t).String())
	}

	// ADA must nest under "ada" -> "lovelace".
	ada, ok := su.Value[shared.AdaPolicy]
	if !ok {
		t.Fatalf("value missing %q key: %+v", shared.AdaPolicy, su.Value)
	}
	if got := ada[shared.AdaAsset].String(); got != "1500000" {
		t.Fatalf("lovelace = %s, want 1500000", got)
	}

	// Native asset must nest under policyHex -> assetNameHex.
	policyHex := strings.Repeat("ab", 28)
	assetNameHex := hex.EncodeToString([]byte("TOKEN"))
	assets, ok := su.Value[policyHex]
	if !ok {
		t.Fatalf("value missing policy key %q: %+v", policyHex, su.Value)
	}
	if got := assets[assetNameHex].String(); got != "42" {
		t.Fatalf("asset qty = %s, want 42", got)
	}

	// Reference script must serialize as {"language":"plutus:v2","cbor":...}.
	if len(su.Script) == 0 {
		t.Fatal("expected script to be set")
	}
	var script struct {
		Language string `json:"language"`
		Cbor     string `json:"cbor"`
	}
	if err := json.Unmarshal(su.Script, &script); err != nil {
		t.Fatalf("script JSON unmarshal: %v", err)
	}
	if script.Language != "plutus:v2" {
		t.Fatalf("script language = %q, want plutus:v2", script.Language)
	}
	if script.Cbor != hex.EncodeToString([]byte{0x49, 0x48, 0x01, 0x00}) {
		t.Fatalf("script cbor = %q", script.Cbor)
	}
	if su.Datum != "" || su.DatumHash != "" {
		t.Fatalf("unexpected datum fields: datum=%q datumHash=%q", su.Datum, su.DatumHash)
	}
}

func TestCommonUtxoToSharedRejectsMissingFields(t *testing.T) {
	valid := sampleCommonUtxo(t)
	var typedNilInput *shelley.ShelleyTransactionInput
	var typedNilOutput *babbage.BabbageTransactionOutput

	tests := []struct {
		name    string
		utxo    common.Utxo
		wantErr string
	}{
		{
			name:    "nil transaction input",
			utxo:    common.Utxo{Output: valid.Output},
			wantErr: "missing transaction input",
		},
		{
			name:    "typed nil transaction input",
			utxo:    common.Utxo{Id: typedNilInput, Output: valid.Output},
			wantErr: "missing transaction input",
		},
		{
			name:    "nil transaction output",
			utxo:    common.Utxo{Id: valid.Id},
			wantErr: "missing transaction output",
		},
		{
			name:    "typed nil transaction output",
			utxo:    common.Utxo{Id: valid.Id, Output: typedNilOutput},
			wantErr: "missing transaction output",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := commonUtxoToShared(test.utxo); err == nil {
				t.Fatal("expected malformed UTxO error")
			} else if !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("error = %q, want substring %q", err, test.wantErr)
			}
		})
	}
}

func TestEvaluateTxRejectsMalformedAdditionalUtxos(t *testing.T) {
	valid := sampleCommonUtxo(t)
	tests := []struct {
		name    string
		utxo    common.Utxo
		wantErr string
	}{
		{
			name:    "missing transaction input",
			utxo:    common.Utxo{Output: valid.Output},
			wantErr: "missing transaction input",
		},
		{
			name:    "missing transaction output",
			utxo:    common.Utxo{Id: valid.Id},
			wantErr: "missing transaction output",
		},
	}

	ctx := testChainContext(t, Config{OgmiosEndpoint: testOgmiosEndpoint})
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := ctx.EvaluateTx([]byte{0x84}, []common.Utxo{test.utxo}); err == nil {
				t.Fatal("expected malformed UTxO error")
			} else if !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("error = %q, want substring %q", err, test.wantErr)
			}
		})
	}
}

func TestCommonUtxoToSharedInlineDatum(t *testing.T) {
	var txId common.Blake2b256
	for i := range txId {
		txId[i] = 0x44
	}
	innerDatumCbor := []byte{0x01}
	optCbor, err := cbor.Encode([]any{1, cbor.Tag{Number: 24, Content: innerDatumCbor}})
	if err != nil {
		t.Fatal(err)
	}
	var opt babbage.BabbageTransactionOutputDatumOption
	if err := opt.UnmarshalCBOR(optCbor); err != nil {
		t.Fatal(err)
	}
	output := babbage.BabbageTransactionOutput{
		OutputAddress: testAddress(t),
		OutputAmount:  mary.MaryTransactionOutputValue{Amount: 2_000_000},
		DatumOption:   &opt,
	}
	utxo := common.Utxo{
		Id:     shelley.ShelleyTransactionInput{TxId: txId, OutputIndex: 0},
		Output: &output,
	}

	su, err := commonUtxoToShared(utxo)
	if err != nil {
		t.Fatal(err)
	}
	if su.Datum != hex.EncodeToString(innerDatumCbor) {
		t.Fatalf("datum = %q, want %q", su.Datum, hex.EncodeToString(innerDatumCbor))
	}
	if su.DatumHash != "" {
		t.Fatalf("inline datum must not set DatumHash, got %q", su.DatumHash)
	}
	if len(su.Script) != 0 {
		t.Fatalf("unexpected script: %s", su.Script)
	}
}

func TestCommonUtxoToSharedHashOnlyDatum(t *testing.T) {
	var txId common.Blake2b256
	for i := range txId {
		txId[i] = 0x55
	}
	var datumHash common.Blake2b256
	for i := range datumHash {
		datumHash[i] = 0xEF
	}
	optCbor, err := cbor.Encode([]any{0, datumHash})
	if err != nil {
		t.Fatal(err)
	}
	var opt babbage.BabbageTransactionOutputDatumOption
	if err := opt.UnmarshalCBOR(optCbor); err != nil {
		t.Fatal(err)
	}
	output := babbage.BabbageTransactionOutput{
		OutputAddress: testAddress(t),
		OutputAmount:  mary.MaryTransactionOutputValue{Amount: 2_000_000},
		DatumOption:   &opt,
	}
	utxo := common.Utxo{
		Id:     shelley.ShelleyTransactionInput{TxId: txId, OutputIndex: 0},
		Output: &output,
	}

	su, err := commonUtxoToShared(utxo)
	if err != nil {
		t.Fatal(err)
	}
	if su.DatumHash != strings.Repeat("ef", 32) {
		t.Fatalf("datumHash = %q, want %q", su.DatumHash, strings.Repeat("ef", 32))
	}
	if su.Datum != "" {
		t.Fatalf("hash-only datum must not set Datum, got %q", su.Datum)
	}
}

func TestOgmiosScriptRefJSONLanguageDetection(t *testing.T) {
	raw := []byte{0x49, 0x48, 0x01, 0x00}
	for _, tc := range []struct {
		script   common.Script
		language string
	}{
		{common.PlutusV1Script(raw), "plutus:v1"},
		{common.PlutusV2Script(raw), "plutus:v2"},
		{common.PlutusV3Script(raw), "plutus:v3"},
		{common.PlutusV4Script(raw), "plutus:v4"},
		{common.NativeScript{}, "native"},
	} {
		out, err := ogmiosScriptRefJSON(tc.script)
		if err != nil {
			t.Fatalf("%s: %v", tc.language, err)
		}
		var parsed struct {
			Language string `json:"language"`
			Cbor     string `json:"cbor"`
		}
		if err := json.Unmarshal(out, &parsed); err != nil {
			t.Fatalf("%s: unmarshal: %v", tc.language, err)
		}
		if parsed.Language != tc.language {
			t.Fatalf("language = %q, want %q", parsed.Language, tc.language)
		}
	}
}

func TestProtocolParamsPreserveReferenceScriptFeeParameters(t *testing.T) {
	const body = `{
		"minFeeCoefficient": 44,
		"minFeeConstant": {"ada": {"lovelace": 155381}},
		"minUtxoDepositCoefficient": 4310,
		"stakeCredentialDeposit": {"ada": {"lovelace": 2000000}},
		"stakePoolDeposit": {"ada": {"lovelace": 500000000}},
		"minStakePoolCost": {"ada": {"lovelace": 170000000}},
		"scriptExecutionPrices": {"memory": "1/1", "cpu": "1/1"},
		"minFeeReferenceScripts": {
			"base": 15.125,
			"range": 7,
			"multiplier": 1.2345
		}
	}`

	var raw ogmiosProtocolParams
	if err := json.Unmarshal([]byte(body), &raw); err != nil {
		t.Fatal(err)
	}
	pp, err := raw.toProtocolParams()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := pp.RefScriptFeePerByteRational(), big.NewRat(121, 8); got.Cmp(want) != 0 {
		t.Fatalf("RefScriptFeePerByteRational() = %s, want %s", got, want)
	}
	if got, want := pp.RefScriptMultiplierRational(), big.NewRat(2469, 2000); got.Cmp(want) != 0 {
		t.Fatalf("RefScriptMultiplierRational() = %s, want %s", got, want)
	}
	if got := pp.RefScriptSizeIncrement(); got != 7 {
		t.Fatalf("RefScriptSizeIncrement() = %d, want 7", got)
	}

	want := backend.TierRefScriptFeeRational(53, big.NewRat(121, 8), 7, big.NewRat(2469, 2000))
	if got := backend.TierRefScriptFeeRational(53, pp.RefScriptFeePerByteRational(), pp.RefScriptSizeIncrement(), pp.RefScriptMultiplierRational()); got != want {
		t.Fatalf("reference-script fee = %d, want %d", got, want)
	}
}

// ogmiosV6ProtocolParamsTemplate is a queryLedgerState/protocolParameters
// result as Ogmios v6.x serves it on a Conway-era network. Every lovelace
// parameter is a %s placeholder so the same response can be rendered in both
// encodings Ogmios has used (see adaLovelace and bareLovelace); the remaining
// fields are reproduced as sent, except that the Plutus cost model arrays are
// truncated to keep the fixture readable.
const ogmiosV6ProtocolParamsTemplate = `{
	"minFeeCoefficient": 44,
	"minFeeConstant": %s,
	"minUtxoDepositCoefficient": 4310,
	"minUtxoDepositConstant": %s,
	"maxBlockBodySize": {"bytes": 90112},
	"maxBlockHeaderSize": {"bytes": 1100},
	"maxTransactionSize": {"bytes": 16384},
	"maxValueSize": {"bytes": 5000},
	"maxReferenceScriptsSize": {"bytes": 204800},
	"extraEntropy": "neutral",
	"stakeCredentialDeposit": %s,
	"stakePoolDeposit": %s,
	"stakePoolRetirementEpochBound": 18,
	"stakePoolPledgeInfluence": "3/10",
	"minStakePoolCost": %s,
	"desiredNumberOfStakePools": 500,
	"monetaryExpansion": "3/1000",
	"treasuryExpansion": "1/5",
	"collateralPercentage": 150,
	"maxCollateralInputs": 3,
	"version": {"major": 10, "minor": 0},
	"scriptExecutionPrices": {"memory": "577/10000", "cpu": "721/10000000"},
	"maxExecutionUnitsPerTransaction": {
		"memory": 14000000,
		"cpu": 10000000000
	},
	"maxExecutionUnitsPerBlock": {"memory": 62000000, "cpu": 20000000000},
	"minFeeReferenceScripts": {"base": 15, "range": 25600, "multiplier": 1.2},
	"plutusCostModels": {
		"plutus:v1": [100788, 420, 1, 1, 1000, 173, 0, 1],
		"plutus:v2": [100788, 420, 1, 1, 1000, 173, 0, 1],
		"plutus:v3": [100788, 420, 1, 1, 1000, 173, 0, 1]
	},
	"stakePoolVotingThresholds": {
		"noConfidence": "51/100",
		"constitutionalCommittee": {
			"default": "51/100",
			"stateOfNoConfidence": "51/100"
		},
		"hardForkInitiation": "51/100",
		"protocolParametersUpdate": {"security": "51/100"}
	},
	"delegateRepresentativeVotingThresholds": {
		"noConfidence": "67/100",
		"constitutionalCommittee": {
			"default": "67/100",
			"stateOfNoConfidence": "3/5"
		},
		"constitution": "3/4",
		"hardForkInitiation": "3/5",
		"protocolParametersUpdate": {
			"network": "67/100",
			"economic": "67/100",
			"technical": "67/100",
			"governance": "3/4"
		},
		"treasuryWithdrawals": "67/100"
	},
	"constitutionalCommitteeMinSize": 7,
	"constitutionalCommitteeMaxTermLength": 146,
	"governanceActionLifetime": 6,
	"governanceActionDeposit": %s,
	"delegateRepresentativeDeposit": %s,
	"delegateRepresentativeMaxIdleTime": 20
}`

// adaLovelace renders the Value<AdaOnly> encoding Ogmios has used since v6.1.0.
func adaLovelace(amount int64) string {
	return fmt.Sprintf(`{"ada": {"lovelace": %d}}`, amount)
}

// bareLovelace renders the flat encoding used by Ogmios v6.0.x.
func bareLovelace(amount int64) string {
	return fmt.Sprintf(`{"lovelace": %d}`, amount)
}

// protocolParamsBody renders ogmiosV6ProtocolParamsTemplate with every lovelace
// amount encoded by the given function. The argument order must match the
// placeholder order in the template.
func protocolParamsBody(encode func(int64) string) string {
	return fmt.Sprintf(
		ogmiosV6ProtocolParamsTemplate,
		encode(155381),       // minFeeConstant
		encode(0),            // minUtxoDepositConstant
		encode(2000000),      // stakeCredentialDeposit
		encode(500000000),    // stakePoolDeposit
		encode(170000000),    // minStakePoolCost
		encode(100000000000), // governanceActionDeposit
		encode(500000000),    // delegateRepresentativeDeposit
	)
}

func decodeProtocolParams(
	t *testing.T,
	body string,
) backend.ProtocolParameters {
	t.Helper()
	var raw ogmiosProtocolParams
	if err := json.Unmarshal([]byte(body), &raw); err != nil {
		t.Fatalf("failed to decode protocol params: %v", err)
	}
	pp, err := raw.toProtocolParams()
	if err != nil {
		t.Fatalf("failed to convert protocol params: %v", err)
	}
	return pp
}

// assertOgmiosV6ProtocolParams checks every parameter Apollo reads out of an
// Ogmios protocol-parameters response against the fixture above. The zero
// checks are the point of the test: a lovelace amount whose wire shape is not
// understood decodes to 0 without an error, and a zero minFeeConstant leaves
// every computed fee short by 155381 lovelace.
func assertOgmiosV6ProtocolParams(t *testing.T, pp backend.ProtocolParameters) {
	t.Helper()

	intFields := []struct {
		name string
		got  int64
		want int64
	}{
		{"MinFeeConstant", pp.MinFeeConstant, 155381},
		{"MinFeeCoefficient", pp.MinFeeCoefficient, 44},
		{"CoinsPerUtxoByteValue", pp.CoinsPerUtxoByteValue(), 4310},
		{"MaxBlockSize", int64(pp.MaxBlockSize), 90112},
		{"MaxBlockHeaderSize", int64(pp.MaxBlockHeaderSize), 1100},
		{"MaxTxSize", int64(pp.MaxTxSize), 16384},
		{"CollateralPercent", int64(pp.CollateralPercent), 150},
		{"MaxCollateralInputs", int64(pp.MaxCollateralInputs), 3},
		{
			"MaximumReferenceScriptsSize",
			int64(pp.MaximumReferenceScriptsSize),
			204800,
		},
		{
			"MinFeeReferenceScriptsRange",
			int64(pp.MinFeeReferenceScriptsRange),
			25600,
		},
	}
	for _, field := range intFields {
		if field.got == 0 {
			t.Errorf("%s is zero: value was silently dropped", field.name)
			continue
		}
		if field.got != field.want {
			t.Errorf("%s = %d, want %d", field.name, field.got, field.want)
		}
	}

	stringFields := []struct {
		name string
		got  string
		want string
	}{
		{"KeyDeposits", pp.KeyDeposits, "2000000"},
		{"PoolDeposits", pp.PoolDeposits, "500000000"},
		{"MinPoolCost", pp.MinPoolCost, "170000000"},
		{"CoinsPerUtxoByte", pp.CoinsPerUtxoByte, "4310"},
		{"MaxTxExMem", pp.MaxTxExMem, "14000000"},
		{"MaxTxExSteps", pp.MaxTxExSteps, "10000000000"},
		{"MaxBlockExMem", pp.MaxBlockExMem, "62000000"},
		{"MaxBlockExSteps", pp.MaxBlockExSteps, "20000000000"},
		{"MaxValSize", pp.MaxValSize, "5000"},
	}
	for _, field := range stringFields {
		if field.got == "" || field.got == "0" {
			t.Errorf("%s = %q: value was silently dropped", field.name, field.got)
			continue
		}
		if field.got != field.want {
			t.Errorf("%s = %q, want %q", field.name, field.got, field.want)
		}
	}

	if got, want := pp.PriceMem, 577.0/10000.0; got != want {
		t.Errorf("PriceMem = %v, want %v", got, want)
	}
	if got, want := pp.PriceStep, 721.0/10000000.0; got != want {
		t.Errorf("PriceStep = %v, want %v", got, want)
	}

	base, wantBase := pp.RefScriptFeePerByteRational(), big.NewRat(15, 1)
	if base.Cmp(wantBase) != 0 {
		t.Errorf("RefScriptFeePerByteRational() = %s, want %s", base, wantBase)
	}
	mult, wantMult := pp.RefScriptMultiplierRational(), big.NewRat(6, 5)
	if mult.Cmp(wantMult) != 0 {
		t.Errorf("RefScriptMultiplierRational() = %s, want %s", mult, wantMult)
	}

	for _, language := range []string{"PlutusV1", "PlutusV2", "PlutusV3"} {
		costs, ok := pp.CostModels[language]
		if !ok {
			t.Errorf("cost models missing %s", language)
			continue
		}
		if len(costs) == 0 || costs[0] != 100788 {
			t.Errorf("%s cost model = %v", language, costs)
		}
	}
}

// TestProtocolParamsDecodeLovelaceShapes decodes a realistic Ogmios v6
// protocol-parameters response in both lovelace encodings. Ogmios v6.0.x sent
// a bare {"lovelace": N}; v6.1.0 onward sends {"ada": {"lovelace": N}}. Both
// must yield the real parameter values.
func TestProtocolParamsDecodeLovelaceShapes(t *testing.T) {
	shapes := []struct {
		name   string
		encode func(int64) string
	}{
		{"ada nested (v6.1.0 onward)", adaLovelace},
		{"bare lovelace (v6.0.x)", bareLovelace},
	}
	for _, shape := range shapes {
		t.Run(shape.name, func(t *testing.T) {
			pp := decodeProtocolParams(t, protocolParamsBody(shape.encode))
			assertOgmiosV6ProtocolParams(t, pp)
		})
	}
}

// TestProtocolParamsAcceptRenamedReferenceScriptsSize covers the v7.0 rename of
// maxReferenceScriptsSize to maxReferenceScriptsSizePerTransaction.
func TestProtocolParamsAcceptRenamedReferenceScriptsSize(t *testing.T) {
	body := strings.Replace(
		protocolParamsBody(adaLovelace),
		`"maxReferenceScriptsSize"`,
		`"maxReferenceScriptsSizePerTransaction"`,
		1,
	)
	if body == protocolParamsBody(adaLovelace) {
		t.Fatal("fixture no longer contains maxReferenceScriptsSize")
	}

	pp := decodeProtocolParams(t, body)
	if got, want := pp.MaximumReferenceScriptsSize, 204800; got != want {
		t.Fatalf("MaximumReferenceScriptsSize = %d, want %d", got, want)
	}
}

// TestProtocolParamsLovelaceShapesAgree pins the two encodings to identical
// results, so neither path can drift away from the other.
func TestProtocolParamsLovelaceShapesAgree(t *testing.T) {
	nested := decodeProtocolParams(t, protocolParamsBody(adaLovelace))
	bare := decodeProtocolParams(t, protocolParamsBody(bareLovelace))
	if !reflect.DeepEqual(nested, bare) {
		t.Fatalf("encodings disagree:\nnested = %+v\nbare   = %+v", nested, bare)
	}
}

// TestProtocolParamsProduceLedgerMinFee mirrors the fee formula Apollo applies
// in Complete() to prove the decoded parameters produce the fee the ledger
// expects. With a dropped minFeeConstant this fee is 155381 lovelace short and
// the node rejects the transaction with FeeTooSmallUTxO.
func TestProtocolParamsProduceLedgerMinFee(t *testing.T) {
	pp := decodeProtocolParams(t, protocolParamsBody(adaLovelace))

	const txSize = 1024
	got := int64(txSize)*pp.MinFeeCoefficient + pp.MinFeeConstant
	want := int64(txSize*44 + 155381)
	if got != want {
		t.Fatalf(
			"min fee for a %d-byte tx = %d, want %d (short by %d)",
			txSize, got, want, want-got,
		)
	}
}

// TestOgmiosLovelaceRejectsUnrecognizedShape is the guard against the next
// wire-format change: an amount Apollo cannot interpret must be an error, not
// a zero. Silently decoding to zero is what shipped the fee bug.
func TestOgmiosLovelaceRejectsUnrecognizedShape(t *testing.T) {
	bodies := []string{
		`{}`,
		`null`,
		`{"coin": 155381}`,
		`{"ada": {}}`,
		`{"ada": {"coin": 155381}}`,
		`{"ada": null}`,
		`{"lovelaces": 155381}`,
	}
	for _, body := range bodies {
		t.Run(body, func(t *testing.T) {
			var amount ogmiosLovelace
			err := json.Unmarshal([]byte(body), &amount)
			if err == nil {
				t.Fatalf(
					"%s decoded to %d without error; an unrecognized"+
						" shape must not become a silent zero",
					body, amount.Lovelace,
				)
			}
		})
	}
}

// TestOgmiosLovelacePrefersAdaWhenBothPresent documents the precedence used if
// a response ever carries both encodings.
func TestOgmiosLovelacePrefersAdaWhenBothPresent(t *testing.T) {
	var amount ogmiosLovelace
	body := `{"ada": {"lovelace": 155381}, "lovelace": 7}`
	if err := json.Unmarshal([]byte(body), &amount); err != nil {
		t.Fatal(err)
	}
	if amount.Lovelace != 155381 {
		t.Fatalf("lovelace = %d, want 155381", amount.Lovelace)
	}
}

// TestProtocolParamsRejectUnrecognizedLovelaceShape checks that an unreadable
// amount fails the whole response rather than one field.
func TestProtocolParamsRejectUnrecognizedLovelaceShape(t *testing.T) {
	body := protocolParamsBody(func(amount int64) string {
		return fmt.Sprintf(`{"coin": %d}`, amount)
	})
	var raw ogmiosProtocolParams
	if err := json.Unmarshal([]byte(body), &raw); err == nil {
		t.Fatal("expected an error for an unrecognized lovelace encoding")
	}
}

// TestProtocolParamsRejectMissingRequiredFields covers the other way a
// parameter can silently become zero: the key disappearing from the response.
func TestProtocolParamsRejectMissingRequiredFields(t *testing.T) {
	const body = `{"scriptExecutionPrices": {"memory": "1/1", "cpu": "1/1"}}`

	var raw ogmiosProtocolParams
	if err := json.Unmarshal([]byte(body), &raw); err != nil {
		t.Fatal(err)
	}
	_, err := raw.toProtocolParams()
	if err == nil {
		t.Fatal("expected an error for missing fee and deposit parameters")
	}
	required := []string{
		"minFeeCoefficient",
		"minFeeConstant",
		"minUtxoDepositCoefficient",
		"stakeCredentialDeposit",
		"stakePoolDeposit",
		"minStakePoolCost",
	}
	for _, field := range required {
		if !strings.Contains(err.Error(), field) {
			t.Errorf("error %q does not report missing %s", err, field)
		}
	}
}

// ogmiosV6ShelleyGenesisBody is a queryNetwork/genesisConfiguration result for
// the Shelley era as Ogmios v6.x serves it, carrying the real mainnet values.
// Note the two shapes that are easy to get wrong: activeSlotsCoefficient is an
// exact ratio string, and slotLength is an object in milliseconds. The
// initialParameters/initialFunds members are abridged; Apollo does not read
// them.
const ogmiosV6ShelleyGenesisBody = `{
	"era": "shelley",
	"startTime": "2017-09-23T21:44:51Z",
	"networkMagic": 764824073,
	"network": "mainnet",
	"activeSlotsCoefficient": "1/20",
	"securityParameter": 2160,
	"epochLength": 432000,
	"slotsPerKesPeriod": 129600,
	"maxKesEvolutions": 62,
	"slotLength": {"milliseconds": 1000},
	"updateQuorum": 5,
	"maxLovelaceSupply": 45000000000000000,
	"initialParameters": {
		"minFeeCoefficient": 44,
		"minFeeConstant": {"ada": {"lovelace": 155381}}
	},
	"initialDelegates": [],
	"initialFunds": {},
	"initialStakePools": {"stakePools": {}, "delegators": {}}
}`

func decodeGenesisParams(
	t *testing.T,
	body string,
) backend.GenesisParameters {
	t.Helper()
	var raw ogmiosGenesisConfig
	if err := json.Unmarshal([]byte(body), &raw); err != nil {
		t.Fatalf("failed to decode genesis config: %v", err)
	}
	gp, err := raw.toGenesisParams()
	if err != nil {
		t.Fatalf("failed to convert genesis config: %v", err)
	}
	return gp
}

// TestGenesisParamsDecodeOgmiosV6Response decodes a realistic Ogmios v6 Shelley
// genesis response. activeSlotsCoefficient is a ratio string and slotLength is
// {"milliseconds": N}; decoding either into a plain scalar makes the whole
// GenesisParams call fail.
func TestGenesisParamsDecodeOgmiosV6Response(t *testing.T) {
	gp := decodeGenesisParams(t, ogmiosV6ShelleyGenesisBody)

	if got, want := gp.ActiveSlotsCoefficient, 0.05; got != want {
		t.Errorf("ActiveSlotsCoefficient = %v, want %v", got, want)
	}
	if got, want := gp.SlotLength, 1; got != want {
		t.Errorf("SlotLength = %d seconds, want %d", got, want)
	}
	// 2017-09-23T21:44:51Z, the mainnet system start, in Unix seconds.
	if got, want := gp.SystemStart, int64(1506203091); got != want {
		t.Errorf("SystemStart = %d, want %d", got, want)
	}

	intFields := []struct {
		name string
		got  int
		want int
	}{
		{"NetworkMagic", gp.NetworkMagic, 764824073},
		{"SecurityParam", gp.SecurityParam, 2160},
		{"EpochLength", gp.EpochLength, 432000},
		{"SlotsPerKesPeriod", gp.SlotsPerKesPeriod, 129600},
		{"MaxKesEvolutions", gp.MaxKesEvolutions, 62},
		{"UpdateQuorum", gp.UpdateQuorum, 5},
	}
	for _, field := range intFields {
		if field.got == 0 {
			t.Errorf("%s is zero: value was silently dropped", field.name)
			continue
		}
		if field.got != field.want {
			t.Errorf("%s = %d, want %d", field.name, field.got, field.want)
		}
	}

	if got, want := gp.MaxLovelaceSupply, "45000000000000000"; got != want {
		t.Errorf("MaxLovelaceSupply = %q, want %q", got, want)
	}
}

// TestGenesisParamsAcceptLegacyScalarShapes keeps the older Ogmios encodings
// working: a bare number for the coefficient and whole seconds for the slot
// length.
func TestGenesisParamsAcceptLegacyScalarShapes(t *testing.T) {
	const body = `{
		"startTime": "2017-09-23T21:44:51Z",
		"networkMagic": 764824073,
		"activeSlotsCoefficient": 0.05,
		"securityParameter": 2160,
		"epochLength": 432000,
		"slotsPerKesPeriod": 129600,
		"maxKesEvolutions": 62,
		"slotLength": 1,
		"updateQuorum": 5,
		"maxLovelaceSupply": 45000000000000000
	}`

	gp := decodeGenesisParams(t, body)
	if got, want := gp.ActiveSlotsCoefficient, 0.05; got != want {
		t.Errorf("ActiveSlotsCoefficient = %v, want %v", got, want)
	}
	if got, want := gp.SlotLength, 1; got != want {
		t.Errorf("SlotLength = %d, want %d", got, want)
	}
}

// TestGenesisParamsRejectUnrecognizedShapes holds the genesis decoding to the
// same rule as the protocol parameters: a value Apollo cannot interpret is an
// error, never a zero.
func TestGenesisParamsRejectUnrecognizedShapes(t *testing.T) {
	bodies := map[string]string{
		"ratio object":       `{"activeSlotsCoefficient": {"ratio": "1/20"}}`,
		"ratio not a number": `{"activeSlotsCoefficient": "one twentieth"}`,
		"slot length object": `{"slotLength": {"seconds": 1}}`,
		"slot length string": `{"slotLength": "1000ms"}`,
	}
	for name, body := range bodies {
		t.Run(name, func(t *testing.T) {
			var raw ogmiosGenesisConfig
			if err := json.Unmarshal([]byte(body), &raw); err == nil {
				t.Fatalf(
					"%s decoded without error; an unrecognized"+
						" shape must not become a silent zero",
					body,
				)
			}
		})
	}
}

// TestGenesisParamsRejectInvalidStartTime keeps an unparseable timestamp from
// silently becoming a zero system start.
func TestGenesisParamsRejectInvalidStartTime(t *testing.T) {
	const body = `{"startTime": "yesterday"}`

	var raw ogmiosGenesisConfig
	if err := json.Unmarshal([]byte(body), &raw); err != nil {
		t.Fatal(err)
	}
	if _, err := raw.toGenesisParams(); err == nil {
		t.Fatal("expected an error for an unparseable start time")
	}
}

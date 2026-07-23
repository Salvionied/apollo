package ogmios

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
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
	ctx := NewOgmiosChainContext(nil, nil, 0)
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

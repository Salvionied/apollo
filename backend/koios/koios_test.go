package koios

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/babbage"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/mary"
	"github.com/blinklabs-io/gouroboros/ledger/shelley"

	"github.com/Salvionied/apollo/v2/backend"
)

func testAddress(t *testing.T) common.Address {
	t.Helper()
	var raw [57]byte
	raw[1] = 0xAA
	raw[29] = 0xBB
	addr, err := common.NewAddressFromBytes(raw[:])
	if err != nil {
		t.Fatal(err)
	}
	return addr
}

// testScript is (program 1.1.0 (lam x x)) wrapped in a CBOR byte string, the
// form held by gouroboros Plutus script types.
var testScript = common.PlutusV3Script{0x46, 0x01, 0x01, 0x00, 0x20, 0x01, 0x01}

const testTxHash = "1111111111111111111111111111111111111111111111111111111111111111"

// recordingServer serves canned responses by method and path and records each
// request it receives.
type recordingServer struct {
	t        *testing.T
	mu       sync.Mutex
	handlers map[string]func(r *http.Request, body []byte) (int, string)
	requests []recordedRequest
}

type recordedRequest struct {
	method string
	path   string
	query  string
	header http.Header
	body   []byte
}

func newRecordingServer(t *testing.T) (*recordingServer, *httptest.Server) {
	t.Helper()
	rs := &recordingServer{t: t, handlers: map[string]func(*http.Request, []byte) (int, string){}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		rs.mu.Lock()
		rs.requests = append(rs.requests, recordedRequest{
			method: r.Method, path: r.URL.Path, query: r.URL.RawQuery, header: r.Header.Clone(), body: body,
		})
		handler, ok := rs.handlers[r.Method+" "+r.URL.Path]
		rs.mu.Unlock()
		if !ok {
			http.Error(w, "unexpected request "+r.Method+" "+r.URL.Path, http.StatusNotFound)
			return
		}
		status, response := handler(r, body)
		w.WriteHeader(status)
		_, _ = io.WriteString(w, response)
	}))
	t.Cleanup(server.Close)
	return rs, server
}

func (rs *recordingServer) handle(method, path string, status int, response string) {
	rs.handleFunc(method, path, func(*http.Request, []byte) (int, string) { return status, response })
}

func (rs *recordingServer) handleFunc(method, path string, fn func(r *http.Request, body []byte) (int, string)) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	rs.handlers[method+" "+path] = fn
}

func (rs *recordingServer) recorded() []recordedRequest {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	return append([]recordedRequest(nil), rs.requests...)
}

func TestKoiosImplementsContextAndCapabilities(t *testing.T) {
	ctx := NewKoiosChainContext("https://example.invalid/api/v1", 1, "")
	if !backend.Supports(ctx, backend.AllCapabilities) {
		t.Fatalf("capabilities = %b, want all", ctx.Capabilities())
	}
	if ctx.NetworkId() != 1 {
		t.Fatalf("NetworkId() = %d, want 1", ctx.NetworkId())
	}
}

func TestBaseUrlUsedAsGiven(t *testing.T) {
	for _, basePath := range []string{"/api/v1", "/api/v1/", "/proxy/koios", ""} {
		t.Run(basePath, func(t *testing.T) {
			rs, server := newRecordingServer(t)
			wantPrefix := strings.TrimRight(basePath, "/")
			rs.handle(http.MethodGet, wantPrefix+"/tip", http.StatusOK, `[{"abs_slot":42,"epoch_no":7}]`)
			slot, err := NewKoiosChainContext(server.URL+basePath, 0, "").Tip()
			if err != nil {
				t.Fatalf("Tip() error = %v", err)
			}
			if slot != 42 {
				t.Fatalf("Tip() = %d, want 42", slot)
			}
		})
	}
}

func TestBearerTokenSentOnlyWhenSet(t *testing.T) {
	rs, server := newRecordingServer(t)
	rs.handle(http.MethodGet, "/tip", http.StatusOK, `[{"abs_slot":1,"epoch_no":1}]`)
	if _, err := NewKoiosChainContext(server.URL, 0, "secret-token").Tip(); err != nil {
		t.Fatal(err)
	}
	if _, err := NewKoiosChainContext(server.URL, 0, "").Tip(); err != nil {
		t.Fatal(err)
	}
	requests := rs.recorded()
	if got := requests[0].header.Get("Authorization"); got != "Bearer secret-token" {
		t.Fatalf("Authorization = %q, want bearer token", got)
	}
	if got := requests[1].header.Get("Authorization"); got != "" {
		t.Fatalf("Authorization = %q, want none for the public tier", got)
	}
}

func TestProviderIntegerRejectsLargeExponent(t *testing.T) {
	if _, err := parseNonNegativeInt64("0e4096", "value"); err == nil {
		t.Fatal("expected a large provider exponent to fail before integer conversion")
	}
	for _, input := range []string{"100", "1e2", "1.0e2"} {
		if got, err := parseNonNegativeInt64(input, "value"); err != nil || got != 100 {
			t.Fatalf("parseNonNegativeInt64(%q) = %d, %v; want 100", input, got, err)
		}
	}
}

func TestTipAndCurrentEpoch(t *testing.T) {
	rs, server := newRecordingServer(t)
	rs.handle(http.MethodGet, "/tip", http.StatusOK,
		`[{"hash":"aa","era":"Conway","epoch_no":512,"abs_slot":140000000,"epoch_slot":5,"block_time":1}]`)
	ctx := NewKoiosChainContext(server.URL, 1, "")
	slot, err := ctx.Tip()
	if err != nil || slot != 140000000 {
		t.Fatalf("Tip() = %d, %v; want 140000000", slot, err)
	}
	epoch, err := ctx.CurrentEpoch()
	if err != nil || epoch != 512 {
		t.Fatalf("CurrentEpoch() = %d, %v; want 512", epoch, err)
	}
}

func TestTipRejectsMalformedResponses(t *testing.T) {
	for name, response := range map[string]string{
		"empty":         `[]`,
		"two rows":      `[{"abs_slot":1,"epoch_no":1},{"abs_slot":2,"epoch_no":1}]`,
		"negative slot": `[{"abs_slot":-1,"epoch_no":1}]`,
		"not json":      `<html>`,
	} {
		t.Run(name, func(t *testing.T) {
			rs, server := newRecordingServer(t)
			rs.handle(http.MethodGet, "/tip", http.StatusOK, response)
			if _, err := NewKoiosChainContext(server.URL, 0, "").Tip(); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestAPIErrorIncludesStatusAndBody(t *testing.T) {
	rs, server := newRecordingServer(t)
	rs.handle(http.MethodGet, "/tip", http.StatusTooManyRequests, `{"message":"rate limited"}`)
	_, err := NewKoiosChainContext(server.URL, 0, "").Tip()
	if err == nil || !strings.Contains(err.Error(), "429") || !strings.Contains(err.Error(), "rate limited") {
		t.Fatalf("error = %v, want status and body", err)
	}
}

func TestRequestContextPropagatesCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := NewKoiosChainContext("https://example.invalid/api/v1", 0, "").TipContext(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("TipContext() error = %v, want context.Canceled", err)
	}
}

// epochParamsJSON is a Conway epoch_params row in the Koios v1 shape.
func epochParamsJSON(overrides map[string]any) string {
	row := map[string]any{
		"epoch_no":                         512,
		"min_fee_a":                        44,
		"min_fee_b":                        155381,
		"max_block_size":                   90112,
		"max_tx_size":                      16384,
		"max_bh_size":                      1100,
		"key_deposit":                      "2000000",
		"pool_deposit":                     "500000000",
		"decentralisation":                 0,
		"extra_entropy":                    nil,
		"protocol_major":                   10,
		"protocol_minor":                   0,
		"min_pool_cost":                    "170000000",
		"cost_models":                      map[string]any{"PlutusV1": []int64{100, 200}, "PlutusV2": []int64{300}, "PlutusV3": []int64{400, 500, 600}},
		"price_mem":                        0.0577,
		"price_step":                       0.0000721,
		"max_tx_ex_mem":                    14000000,
		"max_tx_ex_steps":                  int64(10000000000),
		"max_block_ex_mem":                 62000000,
		"max_block_ex_steps":               int64(20000000000),
		"max_val_size":                     5000,
		"collateral_percent":               150,
		"max_collateral_inputs":            3,
		"coins_per_utxo_size":              "4310",
		"min_fee_ref_script_cost_per_byte": 15,
	}
	for k, v := range overrides {
		row[k] = v
	}
	out, _ := json.Marshal([]any{row})
	return string(out)
}

func protocolParamsServer(t *testing.T, params string) (*recordingServer, *KoiosChainContext) {
	t.Helper()
	rs, server := newRecordingServer(t)
	rs.handle(http.MethodGet, "/tip", http.StatusOK, `[{"abs_slot":1,"epoch_no":512}]`)
	rs.handle(http.MethodGet, "/epoch_params", http.StatusOK, params)
	return rs, NewKoiosChainContext(server.URL, 1, "")
}

func TestProtocolParams(t *testing.T) {
	rs, ctx := protocolParamsServer(t, epochParamsJSON(nil))
	pp, err := ctx.ProtocolParams()
	if err != nil {
		t.Fatalf("ProtocolParams() error = %v", err)
	}
	var paramsQuery string
	for _, req := range rs.recorded() {
		if req.path == "/epoch_params" {
			paramsQuery = req.query
		}
	}
	if paramsQuery != "_epoch_no=512" {
		t.Fatalf("epoch_params query = %q, want _epoch_no=512", paramsQuery)
	}
	checks := []struct {
		name      string
		got, want any
	}{
		{"MinFeeCoefficient", pp.MinFeeCoefficient, int64(44)},
		{"MinFeeConstant", pp.MinFeeConstant, int64(155381)},
		{"MaxBlockSize", pp.MaxBlockSize, 90112},
		{"MaxTxSize", pp.MaxTxSize, 16384},
		{"MaxBlockHeaderSize", pp.MaxBlockHeaderSize, 1100},
		{"KeyDeposits", pp.KeyDeposits, "2000000"},
		{"PoolDeposits", pp.PoolDeposits, "500000000"},
		{"ProtocolMajorVersion", pp.ProtocolMajorVersion, 10},
		{"MinPoolCost", pp.MinPoolCost, "170000000"},
		{"PriceMem", pp.PriceMem, 0.0577},
		{"PriceStep", pp.PriceStep, 0.0000721},
		{"MaxTxExMem", pp.MaxTxExMem, "14000000"},
		{"MaxTxExSteps", pp.MaxTxExSteps, "10000000000"},
		{"MaxBlockExSteps", pp.MaxBlockExSteps, "20000000000"},
		{"MaxValSize", pp.MaxValSize, "5000"},
		{"CollateralPercent", pp.CollateralPercent, 150},
		{"MaxCollateralInputs", pp.MaxCollateralInputs, 3},
		{"CoinsPerUtxoByte", pp.CoinsPerUtxoByte, "4310"},
		{"MinFeeRefScriptCostPerByte", pp.MinFeeRefScriptCostPerByte, 15.0},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
		}
	}
	if pp.MinFeeRefScriptCostPerByteRational == nil || pp.MinFeeRefScriptCostPerByteRational.Cmp(big.NewRat(15, 1)) != 0 {
		t.Errorf("MinFeeRefScriptCostPerByteRational = %v, want 15", pp.MinFeeRefScriptCostPerByteRational)
	}
	if got := pp.CostModels["PlutusV3"]; len(got) != 3 || got[2] != 600 {
		t.Errorf("PlutusV3 cost model = %v", got)
	}
	if max, err := ctx.MaxTxFee(); err != nil || max != 44*16384+155381 {
		t.Errorf("MaxTxFee() = %d, %v", max, err)
	}
}

func TestProtocolParamsFractionalRefScriptPrice(t *testing.T) {
	_, ctx := protocolParamsServer(t, epochParamsJSON(map[string]any{"min_fee_ref_script_cost_per_byte": json.Number("12.5")}))
	pp, err := ctx.ProtocolParams()
	if err != nil {
		t.Fatal(err)
	}
	if pp.MinFeeRefScriptCostPerByteRational.Cmp(big.NewRat(25, 2)) != 0 {
		t.Fatalf("ref script price = %v, want 25/2", pp.MinFeeRefScriptCostPerByteRational)
	}
}

func TestProtocolParamsCachedAndCopied(t *testing.T) {
	rs, ctx := protocolParamsServer(t, epochParamsJSON(nil))
	first, err := ctx.ProtocolParams()
	if err != nil {
		t.Fatal(err)
	}
	first.CostModels["PlutusV3"][0] = -1
	second, err := ctx.ProtocolParams()
	if err != nil {
		t.Fatal(err)
	}
	if second.CostModels["PlutusV3"][0] != 400 {
		t.Fatal("mutating a returned cost model changed the cache")
	}
	if n := len(rs.recorded()); n != 2 {
		t.Fatalf("made %d requests, want 2 (tip and epoch_params once)", n)
	}
}

func TestProtocolParamsRejectsMalformed(t *testing.T) {
	tests := map[string]string{
		"missing fee":         epochParamsJSON(map[string]any{"min_fee_a": nil}),
		"missing deposit":     epochParamsJSON(map[string]any{"key_deposit": nil}),
		"missing cost models": epochParamsJSON(map[string]any{"cost_models": nil}),
		"missing utxo cost":   epochParamsJSON(map[string]any{"coins_per_utxo_size": nil}),
		"negative fee":        epochParamsJSON(map[string]any{"min_fee_b": -1}),
		"fractional fee":      epochParamsJSON(map[string]any{"min_fee_a": 44.5}),
		"bad deposit":         epochParamsJSON(map[string]any{"pool_deposit": "lots"}),
		"negative ref price":  epochParamsJSON(map[string]any{"min_fee_ref_script_cost_per_byte": -1}),
		"keyed cost models":   epochParamsJSON(map[string]any{"cost_models": map[string]any{"PlutusV1": map[string]int{"a": 1}}}),
		"duplicate model":     epochParamsJSON(map[string]any{"cost_models": map[string]any{"PlutusV1": []int{1}, "plutus:v1": []int{2}}}),
		"oversized tx size":   epochParamsJSON(map[string]any{"max_tx_size": int64(1) << 40}),
		"no rows":             `[]`,
		"two rows":            "[" + strings.Trim(epochParamsJSON(nil), "[]") + "," + strings.Trim(epochParamsJSON(nil), "[]") + "]",
	}
	for name, response := range tests {
		t.Run(name, func(t *testing.T) {
			_, ctx := protocolParamsServer(t, response)
			if _, err := ctx.ProtocolParams(); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

const genesisJSON = `[{"networkmagic":"764824073","networkid":"Mainnet","epochlength":"432000","slotlength":"1",
"maxlovelacesupply":"45000000000000000","systemstart":1506203091,"activeslotcoeff":"0.05",
"slotsperkesperiod":"129600","maxkesrevolutions":"62","securityparam":"2160","updatequorum":"5","alonzogenesis":"{}"}]`

func TestGenesisParams(t *testing.T) {
	rs, server := newRecordingServer(t)
	rs.handle(http.MethodGet, "/genesis", http.StatusOK, genesisJSON)
	ctx := NewKoiosChainContext(server.URL, 1, "")
	gp, err := ctx.GenesisParams()
	if err != nil {
		t.Fatal(err)
	}
	want := backend.GenesisParameters{
		ActiveSlotsCoefficient: 0.05,
		UpdateQuorum:           5,
		MaxLovelaceSupply:      "45000000000000000",
		NetworkMagic:           764824073,
		EpochLength:            432000,
		SystemStart:            1506203091,
		SlotsPerKesPeriod:      129600,
		SlotLength:             1,
		MaxKesEvolutions:       62,
		SecurityParam:          2160,
	}
	if gp != want {
		t.Fatalf("GenesisParams() = %+v, want %+v", gp, want)
	}
	if _, err := ctx.GenesisParams(); err != nil {
		t.Fatal(err)
	}
	if n := len(rs.recorded()); n != 1 {
		t.Fatalf("made %d requests, want 1 (cached)", n)
	}
}

func TestGenesisParamsRejectsMalformed(t *testing.T) {
	for name, replace := range map[string][2]string{
		"bad magic":          {`"764824073"`, `"mainnet"`},
		"fractional length":  {`"slotlength":"1"`, `"slotlength":"0.2"`},
		"zero coefficient":   {`"0.05"`, `"0"`},
		"bad supply":         {`"45000000000000000"`, `"-1"`},
		"missing systemtime": {`"systemstart":1506203091,`, ``},
	} {
		t.Run(name, func(t *testing.T) {
			rs, server := newRecordingServer(t)
			rs.handle(http.MethodGet, "/genesis", http.StatusOK, strings.Replace(genesisJSON, replace[0], replace[1], 1))
			if _, err := NewKoiosChainContext(server.URL, 1, "").GenesisParams(); err == nil {
				t.Fatal("expected error")
			}
		})
	}
	rs, server := newRecordingServer(t)
	rs.handle(http.MethodGet, "/genesis", http.StatusOK, `[]`)
	if _, err := NewKoiosChainContext(server.URL, 1, "").GenesisParams(); err == nil {
		t.Fatal("expected error for empty genesis")
	}
}

// utxoRow returns a Koios address_utxos / utxo_info row.
func utxoRow(t *testing.T, index int, overrides map[string]any) map[string]any {
	t.Helper()
	row := map[string]any{
		"tx_hash":          testTxHash,
		"tx_index":         index,
		"address":          testAddress(t).String(),
		"value":            "2000000",
		"stake_address":    nil,
		"payment_cred":     "aa",
		"epoch_no":         512,
		"block_height":     1,
		"block_time":       1,
		"datum_hash":       nil,
		"inline_datum":     nil,
		"reference_script": nil,
		"asset_list":       []any{},
		"is_spent":         false,
	}
	for k, v := range overrides {
		row[k] = v
	}
	return row
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	out, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

func TestUtxosParsesOutputs(t *testing.T) {
	policy := strings.Repeat("ab", 28)
	datumCbor := []byte{0xd8, 0x79, 0x80} // Constr 0 []
	datumHash := common.Blake2b256Hash(datumCbor)
	hashOnly := strings.Repeat("cd", 32)
	rows := []any{
		utxoRow(t, 0, map[string]any{
			"value": "1500000",
			"asset_list": []any{
				map[string]any{"policy_id": policy, "asset_name": hex.EncodeToString([]byte("TOKEN")), "quantity": "42"},
				map[string]any{"policy_id": policy, "asset_name": "", "quantity": "18446744073709551616"},
			},
		}),
		utxoRow(t, 1, map[string]any{
			"datum_hash":   hex.EncodeToString(datumHash.Bytes()),
			"inline_datum": map[string]any{"bytes": hex.EncodeToString(datumCbor), "value": map[string]any{"constructor": 0}},
			"reference_script": map[string]any{
				"hash": testScript.Hash().String(), "size": 7, "type": "plutusV3",
				"bytes": hex.EncodeToString(testScript), "value": nil,
			},
		}),
		utxoRow(t, 2, map[string]any{"datum_hash": hashOnly}),
	}
	rs, server := newRecordingServer(t)
	rs.handle(http.MethodPost, "/address_utxos", http.StatusOK, mustJSON(t, rows))

	addr := testAddress(t)
	utxos, err := NewKoiosChainContext(server.URL, 0, "").Utxos(addr)
	if err != nil {
		t.Fatalf("Utxos() error = %v", err)
	}
	if len(utxos) != 3 {
		t.Fatalf("got %d UTxOs, want 3", len(utxos))
	}

	req := rs.recorded()[0]
	var body struct {
		Addresses []string `json:"_addresses"`
		Extended  bool     `json:"_extended"`
	}
	if err := json.Unmarshal(req.body, &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Addresses) != 1 || body.Addresses[0] != addr.String() || !body.Extended {
		t.Fatalf("request body = %s", req.body)
	}
	if req.query != "limit=1000&offset=0&order=tx_hash.asc%2Ctx_index.asc" {
		t.Fatalf("request query = %q", req.query)
	}

	first := utxos[0]
	if first.Id.Index() != 0 || hex.EncodeToString(first.Id.Id().Bytes()) != testTxHash {
		t.Fatalf("first input = %s#%d", first.Id.Id(), first.Id.Index())
	}
	if first.Output.Amount().Uint64() != 1_500_000 {
		t.Fatalf("lovelace = %s", first.Output.Amount())
	}
	var policyId common.Blake2b224
	copy(policyId[:], mustDecodeHex(t, policy))
	assets := first.Output.Assets()
	if got := assets.Asset(policyId, []byte("TOKEN")); got.Int64() != 42 {
		t.Fatalf("TOKEN quantity = %s", got)
	}
	if got := assets.Asset(policyId, []byte{}); got.String() != "18446744073709551616" {
		t.Fatalf("empty-name quantity = %s, want a value beyond uint64", got)
	}

	second := utxos[1]
	if datum := second.Output.Datum(); datum == nil || hex.EncodeToString(datum.Cbor()) != "d87980" {
		t.Fatalf("inline datum = %v", second.Output.Datum())
	}
	ref := second.Output.ScriptRef()
	if ref == nil || ref.Hash() != testScript.Hash() {
		t.Fatalf("reference script = %v", ref)
	}

	third := utxos[2]
	if third.Output.Datum() != nil || third.Output.DatumHash() == nil ||
		hex.EncodeToString(third.Output.DatumHash().Bytes()) != hashOnly {
		t.Fatalf("hash-only datum not preserved")
	}
}

func TestUtxosPaginates(t *testing.T) {
	rs, server := newRecordingServer(t)
	rs.handleFunc(http.MethodPost, "/address_utxos", func(r *http.Request, _ []byte) (int, string) {
		count := koiosPageSize
		start := 0
		if r.URL.Query().Get("offset") == "1000" {
			count, start = 2, koiosPageSize
		}
		rows := make([]any, 0, count)
		for i := range count {
			rows = append(rows, utxoRow(t, start+i, nil))
		}
		return http.StatusOK, mustJSON(t, rows)
	})
	utxos, err := NewKoiosChainContext(server.URL, 0, "").Utxos(testAddress(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(utxos) != koiosPageSize+2 {
		t.Fatalf("got %d UTxOs, want %d", len(utxos), koiosPageSize+2)
	}
	if n := len(rs.recorded()); n != 2 {
		t.Fatalf("made %d requests, want 2 (stop after a short page)", n)
	}
}

func TestUtxosEmptyAddress(t *testing.T) {
	rs, server := newRecordingServer(t)
	rs.handle(http.MethodPost, "/address_utxos", http.StatusOK, `[]`)
	utxos, err := NewKoiosChainContext(server.URL, 0, "").Utxos(testAddress(t))
	if err != nil || len(utxos) != 0 {
		t.Fatalf("Utxos() = %v, %v; want empty", utxos, err)
	}
}

func TestUtxosRejectsMalformedRows(t *testing.T) {
	policy := strings.Repeat("ab", 28)
	otherDatum := strings.Repeat("ee", 32)
	tests := map[string]map[string]any{
		"short tx hash":       {"tx_hash": "abcd"},
		"negative index":      {"tx_index": -1},
		"huge index":          {"tx_index": int64(1) << 33},
		"bad value":           {"value": "1.5"},
		"negative value":      {"value": "-1"},
		"other address":       {"address": "addr_test1vz2fxv2umyhttkxyxp8x0dlpdt3k6cwng5pxj3jhsydzerspjrlsz"},
		"bad policy":          {"asset_list": []any{map[string]any{"policy_id": "zz", "asset_name": "", "quantity": "1"}}},
		"zero quantity":       {"asset_list": []any{map[string]any{"policy_id": policy, "asset_name": "", "quantity": "0"}}},
		"long asset name":     {"asset_list": []any{map[string]any{"policy_id": policy, "asset_name": strings.Repeat("00", 33), "quantity": "1"}}},
		"duplicate asset":     {"asset_list": []any{map[string]any{"policy_id": policy, "asset_name": "", "quantity": "1"}, map[string]any{"policy_id": policy, "asset_name": "", "quantity": "2"}}},
		"bad datum hash":      {"datum_hash": "abcd"},
		"datum mismatch":      {"datum_hash": otherDatum, "inline_datum": map[string]any{"bytes": "d87980"}},
		"bad inline datum":    {"inline_datum": map[string]any{"bytes": "zz"}},
		"script hash differs": {"reference_script": map[string]any{"hash": strings.Repeat("00", 28), "type": "plutusV3", "bytes": hex.EncodeToString(testScript)}},
		"script type wrong":   {"reference_script": map[string]any{"hash": testScript.Hash().String(), "type": "plutusV2", "bytes": hex.EncodeToString(testScript)}},
		"script no bytes":     {"reference_script": map[string]any{"hash": testScript.Hash().String(), "type": "plutusV3", "bytes": nil}},
		"script unknown type": {"reference_script": map[string]any{"hash": testScript.Hash().String(), "type": "plutusV9", "bytes": hex.EncodeToString(testScript)}},
	}
	for name, overrides := range tests {
		t.Run(name, func(t *testing.T) {
			rs, server := newRecordingServer(t)
			rs.handle(http.MethodPost, "/address_utxos", http.StatusOK, mustJSON(t, []any{utxoRow(t, 0, overrides)}))
			if _, err := NewKoiosChainContext(server.URL, 0, "").Utxos(testAddress(t)); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestScriptRefFromKoiosAcceptsBothWrappings(t *testing.T) {
	inner, err := hex.DecodeString("010100200101") // flat program without the CBOR wrapper
	if err != nil {
		t.Fatal(err)
	}
	doubleWrapped, err := cbor.Encode([]byte(testScript))
	if err != nil {
		t.Fatal(err)
	}
	for name, scriptBytes := range map[string][]byte{
		"as stored": testScript,
		"unwrapped": inner,
		"wrapped":   doubleWrapped,
	} {
		t.Run(name, func(t *testing.T) {
			bytesHex := hex.EncodeToString(scriptBytes)
			ref, err := scriptRefFromKoios("plutusV3", &bytesHex, nil, testScript.Hash().String())
			if err != nil {
				t.Fatal(err)
			}
			if ref.Type != common.ScriptRefTypePlutusV3 || string(ref.Script.RawScriptBytes()) != string(testScript) {
				t.Fatalf("script ref = %x (type %d)", ref.Script.RawScriptBytes(), ref.Type)
			}
		})
	}
}

func TestScriptRefFromKoiosNativeScript(t *testing.T) {
	var keyHash common.Blake2b224
	keyHash[0] = 0x01
	native, err := cbor.Encode([]any{0, keyHash})
	if err != nil {
		t.Fatal(err)
	}
	var script common.NativeScript
	if _, err := cbor.Decode(native, &script); err != nil {
		t.Fatal(err)
	}
	nativeHex := hex.EncodeToString(native)
	ref, err := scriptRefFromKoios("timelock", &nativeHex, nil, script.Hash().String())
	if err != nil {
		t.Fatal(err)
	}
	if ref.Type != common.ScriptRefTypeNativeScript {
		t.Fatalf("type = %d, want native", ref.Type)
	}
}

// liveTimelockJSON is the JSON form Koios returns, with null bytes, for the
// timelock reference script 7c169ca1... on preprod (output 2004673b...#0).
const liveTimelockJSON = `{"type": "all", "scripts": [{"type": "any", "scripts": [{"type": "sig",
"keyHash": "da0eb5ed7611482ec5089b69d870e0c56c1c45180256112398e0835b"}]}, {"type": "atLeast",
"scripts": [{"slot": 1000, "type": "after"}, {"slot": 999999999999999, "type": "before"}], "required": 1}]}`

const liveTimelockHash = "7c169ca1dfa6220de314daefc1c49b5111e3eb0eebe150fb2cfce774"

func TestUtxosNativeReferenceScriptFromJSON(t *testing.T) {
	rs, server := newRecordingServer(t)
	rs.handle(http.MethodPost, "/address_utxos", http.StatusOK, mustJSON(t, []any{utxoRow(t, 0, map[string]any{
		"reference_script": map[string]any{
			"hash": liveTimelockHash, "size": nil, "type": "timelock", "bytes": nil,
			"value": json.RawMessage(liveTimelockJSON),
		},
	})}))
	utxos, err := NewKoiosChainContext(server.URL, 0, "").Utxos(testAddress(t))
	if err != nil {
		t.Fatalf("Utxos() error = %v", err)
	}
	ref := utxos[0].Output.ScriptRef()
	if ref == nil || hex.EncodeToString(ref.Hash().Bytes()) != liveTimelockHash {
		t.Fatalf("reference script = %v, want native script %s", ref, liveTimelockHash)
	}
	if _, ok := ref.(common.NativeScript); !ok {
		t.Fatalf("reference script type = %T, want common.NativeScript", ref)
	}
}

func TestNativeScriptFromKoiosJSONRejectsBadInput(t *testing.T) {
	tests := map[string]struct {
		value, hash string
	}{
		"hash mismatch":     {liveTimelockJSON, strings.Repeat("00", 28)},
		"unknown type":      {`{"type":"maybe"}`, liveTimelockHash},
		"bad key hash":      {`{"type":"sig","keyHash":"abcd"}`, liveTimelockHash},
		"missing required":  {`{"type":"atLeast","scripts":[]}`, liveTimelockHash},
		"negative slot":     {`{"type":"after","slot":-1}`, liveTimelockHash},
		"not an object":     {`[1,2]`, liveTimelockHash},
		"null value":        {`null`, liveTimelockHash},
		"required too high": {`{"type":"atLeast","required":2,"scripts":[{"type":"after","slot":1}]}`, liveTimelockHash},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := scriptRefFromKoios("timelock", nil, json.RawMessage(tc.value), tc.hash); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestScriptCborNativeScriptFromJSON(t *testing.T) {
	rs, server := newRecordingServer(t)
	rs.handle(http.MethodPost, "/script_info", http.StatusOK, `[{"script_hash":"`+liveTimelockHash+
		`","creation_tx_hash":"79253d463401982e8ca3271ad5769562f0c7cc5e0dd8c2ce0f05643930fd8880","type":"timelock","value":`+
		liveTimelockJSON+`,"bytes":null,"size":null}]`)
	var hash common.Blake2b224
	copy(hash[:], mustDecodeHex(t, liveTimelockHash))
	scriptCbor, err := NewKoiosChainContext(server.URL, 0, "").ScriptCbor(hash)
	if err != nil {
		t.Fatalf("ScriptCbor() error = %v", err)
	}
	var native common.NativeScript
	if _, err := cbor.Decode(scriptCbor, &native); err != nil || native.Hash() != hash {
		t.Fatalf("ScriptCbor() = %x, want native script CBOR hashing to %s (decode error %v)", scriptCbor, liveTimelockHash, err)
	}
}

func TestUtxoByRef(t *testing.T) {
	rs, server := newRecordingServer(t)
	rs.handle(http.MethodPost, "/utxo_info", http.StatusOK,
		mustJSON(t, []any{utxoRow(t, 3, map[string]any{"is_spent": true})}))
	ctx := NewKoiosChainContext(server.URL, 0, "")
	txHash, err := parseBlake2b256(testTxHash, "tx hash")
	if err != nil {
		t.Fatal(err)
	}
	utxo, err := ctx.UtxoByRef(txHash, 3)
	if err != nil {
		t.Fatalf("UtxoByRef() error = %v", err)
	}
	if utxo.Id.Index() != 3 || utxo.Output.Address().String() != testAddress(t).String() {
		t.Fatalf("UtxoByRef() = %+v", utxo)
	}
	var body struct {
		Refs     []string `json:"_utxo_refs"`
		Extended bool     `json:"_extended"`
	}
	if err := json.Unmarshal(rs.recorded()[0].body, &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Refs) != 1 || body.Refs[0] != testTxHash+"#3" || !body.Extended {
		t.Fatalf("request body = %s", rs.recorded()[0].body)
	}

	if _, err := ctx.UtxoByRef(txHash, 4); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("UtxoByRef(missing) error = %v, want not found", err)
	}
}

func TestScriptCbor(t *testing.T) {
	hash := testScript.Hash()
	rs, server := newRecordingServer(t)
	rs.handle(http.MethodPost, "/script_info", http.StatusOK, mustJSON(t, []any{map[string]any{
		"script_hash": hash.String(), "creation_tx_hash": testTxHash, "type": "plutusV3",
		"value": nil, "bytes": "010100200101", "size": 6,
	}}))
	ctx := NewKoiosChainContext(server.URL, 0, "")
	got, err := ctx.ScriptCbor(hash)
	if err != nil {
		t.Fatalf("ScriptCbor() error = %v", err)
	}
	if string(got) != string(testScript) {
		t.Fatalf("ScriptCbor() = %x, want %x", got, []byte(testScript))
	}
	if body := string(rs.recorded()[0].body); body != `{"_script_hashes":["`+hash.String()+`"]}` {
		t.Fatalf("request body = %s", body)
	}

	var other common.Blake2b224
	if _, err := ctx.ScriptCbor(other); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("ScriptCbor(other) error = %v, want not found", err)
	}
}

func TestScriptCborRejectsMismatchAndMissingBytes(t *testing.T) {
	hash := testScript.Hash()
	for name, row := range map[string]map[string]any{
		"mismatch": {"script_hash": hash.String(), "type": "plutusV3", "bytes": "4401010001"},
		"no bytes": {"script_hash": hash.String(), "type": "plutusV3", "bytes": nil},
	} {
		t.Run(name, func(t *testing.T) {
			rs, server := newRecordingServer(t)
			rs.handle(http.MethodPost, "/script_info", http.StatusOK, mustJSON(t, []any{row}))
			if _, err := NewKoiosChainContext(server.URL, 0, "").ScriptCbor(hash); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestSubmitTx(t *testing.T) {
	txCbor := []byte{0x84, 0xa0, 0xa0, 0xf5, 0xf6}
	rs, server := newRecordingServer(t)
	rs.handle(http.MethodPost, "/submittx", http.StatusAccepted, `"`+testTxHash+`"`)
	hash, err := NewKoiosChainContext(server.URL, 0, "").SubmitTx(txCbor)
	if err != nil {
		t.Fatalf("SubmitTx() error = %v", err)
	}
	if hex.EncodeToString(hash.Bytes()) != testTxHash {
		t.Fatalf("SubmitTx() = %s", hash)
	}
	req := rs.recorded()[0]
	if req.header.Get("Content-Type") != "application/cbor" || string(req.body) != string(txCbor) {
		t.Fatalf("request content-type %q body %x", req.header.Get("Content-Type"), req.body)
	}
}

func TestSubmitTxRejectsBadResponses(t *testing.T) {
	for name, tc := range map[string]struct {
		status int
		body   string
	}{
		"rejected":   {http.StatusBadRequest, `"transaction rejected: BadInputsUTxO"`},
		"short hash": {http.StatusAccepted, `"abcd"`},
		"not json":   {http.StatusAccepted, testTxHash},
	} {
		t.Run(name, func(t *testing.T) {
			rs, server := newRecordingServer(t)
			rs.handle(http.MethodPost, "/submittx", tc.status, tc.body)
			if _, err := NewKoiosChainContext(server.URL, 0, "").SubmitTx([]byte{0x80}); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func sampleUtxo(t *testing.T, txByte byte, index uint32) common.Utxo {
	t.Helper()
	var txId common.Blake2b256
	for i := range txId {
		txId[i] = txByte
	}
	return common.Utxo{
		Id: shelley.ShelleyTransactionInput{TxId: txId, OutputIndex: index},
		Output: &babbage.BabbageTransactionOutput{
			OutputAddress: testAddress(t),
			OutputAmount:  mary.MaryTransactionOutputValue{Amount: 5_000_000},
		},
	}
}

const evaluateResult = `{"jsonrpc":"2.0","method":"evaluateTransaction","result":[
{"validator":{"purpose":"spend","index":0},"budget":{"memory":1700,"cpu":476468}},
{"validator":{"purpose":"mint","index":1},"budget":{"memory":2000,"cpu":500000}}]}`

type evaluateRequest struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  struct {
		Transaction struct {
			Cbor string `json:"cbor"`
		} `json:"transaction"`
		AdditionalUtxo []struct {
			Transaction struct {
				ID string `json:"id"`
			} `json:"transaction"`
			Index uint32 `json:"index"`
		} `json:"additionalUtxo"`
	} `json:"params"`
}

func decodeEvaluateRequest(t *testing.T, body []byte) evaluateRequest {
	t.Helper()
	var req evaluateRequest
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatalf("decode evaluate request: %v", err)
	}
	return req
}

func TestEvaluateTx(t *testing.T) {
	rs, server := newRecordingServer(t)
	rs.handle(http.MethodPost, "/ogmios", http.StatusOK, evaluateResult)
	result, err := NewKoiosChainContext(server.URL, 0, "").EvaluateTx([]byte{0x84, 0x01}, nil)
	if err != nil {
		t.Fatalf("EvaluateTx() error = %v", err)
	}
	want := map[common.RedeemerKey]common.ExUnits{
		{Tag: common.RedeemerTagSpend, Index: 0}: {Memory: 1700, Steps: 476468},
		{Tag: common.RedeemerTagMint, Index: 1}:  {Memory: 2000, Steps: 500000},
	}
	if len(result) != len(want) {
		t.Fatalf("EvaluateTx() = %v, want %v", result, want)
	}
	for key, units := range want {
		if result[key] != units {
			t.Fatalf("EvaluateTx()[%v] = %v, want %v", key, result[key], units)
		}
	}
	req := decodeEvaluateRequest(t, rs.recorded()[0].body)
	if req.JSONRPC != "2.0" || req.Method != "evaluateTransaction" || req.Params.Transaction.Cbor != "8401" {
		t.Fatalf("evaluate request = %s", rs.recorded()[0].body)
	}
	if strings.Contains(string(rs.recorded()[0].body), "additionalUtxo") {
		t.Fatal("additionalUtxo sent without additional UTxOs")
	}
}

func TestEvaluateTxSendsAdditionalUtxos(t *testing.T) {
	rs, server := newRecordingServer(t)
	rs.handle(http.MethodPost, "/ogmios", http.StatusOK, evaluateResult)
	utxos := []common.Utxo{sampleUtxo(t, 0x22, 1)}
	if _, err := NewKoiosChainContext(server.URL, 0, "").EvaluateTx([]byte{0x84}, utxos); err != nil {
		t.Fatal(err)
	}
	req := decodeEvaluateRequest(t, rs.recorded()[0].body)
	if len(req.Params.AdditionalUtxo) != 1 || req.Params.AdditionalUtxo[0].Transaction.ID != strings.Repeat("22", 32) ||
		req.Params.AdditionalUtxo[0].Index != 1 {
		t.Fatalf("additionalUtxo = %+v", req.Params.AdditionalUtxo)
	}
}

func TestEvaluateTxDropsOverlappingAdditionalUtxos(t *testing.T) {
	onChain := sampleUtxo(t, 0x22, 0)
	offChain := sampleUtxo(t, 0x33, 1)
	overlap := fmt.Sprintf(`{"jsonrpc":"2.0","method":"evaluateTransaction","error":{"code":3002,
"message":"Some user-provided additional UTxO entries overlap with those that exist in the ledger.",
"data":{"overlappingOutputReferences":[{"transaction":{"id":"%s"},"index":0}]}}}`, strings.Repeat("22", 32))
	rs, server := newRecordingServer(t)
	calls := 0
	rs.handleFunc(http.MethodPost, "/ogmios", func(*http.Request, []byte) (int, string) {
		calls++
		if calls == 1 {
			return http.StatusOK, overlap
		}
		return http.StatusOK, evaluateResult
	})
	result, err := NewKoiosChainContext(server.URL, 0, "").EvaluateTx([]byte{0x84}, []common.Utxo{onChain, offChain})
	if err != nil {
		t.Fatalf("EvaluateTx() error = %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("EvaluateTx() = %v", result)
	}
	requests := rs.recorded()
	if len(requests) != 2 {
		t.Fatalf("made %d requests, want 2", len(requests))
	}
	retry := decodeEvaluateRequest(t, requests[1].body)
	if len(retry.Params.AdditionalUtxo) != 1 || retry.Params.AdditionalUtxo[0].Transaction.ID != strings.Repeat("33", 32) {
		t.Fatalf("retry additionalUtxo = %+v, want only the off-chain UTxO", retry.Params.AdditionalUtxo)
	}
}

// TestEvaluateTxRetriesOverlapReportedWithHTTP400 mirrors live Koios, which
// returns Ogmios JSON-RPC errors with HTTP status 400.
func TestEvaluateTxRetriesOverlapReportedWithHTTP400(t *testing.T) {
	onChain := sampleUtxo(t, 0x22, 0)
	offChain := sampleUtxo(t, 0x33, 1)
	overlap := fmt.Sprintf(`{"jsonrpc":"2.0","method":"evaluateTransaction","error":{"code":3002,
"message":"Some user-provided additional UTxO entries overlap with those that exist in the ledger.",
"data":{"overlappingOutputReferences":[{"transaction":{"id":"%s"},"index":0}]}},"id":null}`, strings.Repeat("22", 32))
	rs, server := newRecordingServer(t)
	calls := 0
	rs.handleFunc(http.MethodPost, "/ogmios", func(*http.Request, []byte) (int, string) {
		calls++
		if calls == 1 {
			return http.StatusBadRequest, overlap
		}
		return http.StatusOK, evaluateResult
	})
	result, err := NewKoiosChainContext(server.URL, 0, "").EvaluateTx([]byte{0x84}, []common.Utxo{onChain, offChain})
	if err != nil {
		t.Fatalf("EvaluateTx() error = %v", err)
	}
	if len(result) != 2 || len(rs.recorded()) != 2 {
		t.Fatalf("EvaluateTx() = %v after %d requests, want a result after one retry", result, len(rs.recorded()))
	}
}

func TestEvaluateTxParsesErrorReportedWithHTTP400(t *testing.T) {
	scriptFailure := `{"jsonrpc":"2.0","method":"evaluateTransaction","error":{"code":3010,
"message":"Some scripts of the transactions terminated with error(s).",
"data":[{"validator":{"index":1,"purpose":"spend"},"error":{"code":3110,"message":"Extraneous redeemer"}}]},"id":null}`
	rs, server := newRecordingServer(t)
	rs.handle(http.MethodPost, "/ogmios", http.StatusBadRequest, scriptFailure)
	_, err := NewKoiosChainContext(server.URL, 0, "").EvaluateTx([]byte{0x84}, nil)
	var rpcErr *ogmiosRPCError
	if !errors.As(err, &rpcErr) || rpcErr.Code != 3010 {
		t.Fatalf("error = %v, want a parsed Ogmios error with code 3010", err)
	}
	if strings.Contains(err.Error(), "koios API error") {
		t.Fatalf("error = %v, want the Ogmios error rather than the HTTP status", err)
	}
}

func TestEvaluateTxNonJSONRPCErrorKeepsHTTPStatus(t *testing.T) {
	rs, server := newRecordingServer(t)
	rs.handle(http.MethodPost, "/ogmios", http.StatusBadGateway, `<html>bad gateway</html>`)
	_, err := NewKoiosChainContext(server.URL, 0, "").EvaluateTx([]byte{0x84}, nil)
	if err == nil || !strings.Contains(err.Error(), "koios API error 502") {
		t.Fatalf("error = %v, want the HTTP status", err)
	}
}

func TestEvaluateTxOverlapRetryWithoutAdditionalUtxos(t *testing.T) {
	onChain := sampleUtxo(t, 0x22, 0)
	overlap := fmt.Sprintf(`{"error":{"code":3002,"message":"overlap",
"data":{"overlappingOutputReferences":[{"transaction":{"id":"%s"},"index":0}]}}}`, strings.Repeat("22", 32))
	rs, server := newRecordingServer(t)
	calls := 0
	rs.handleFunc(http.MethodPost, "/ogmios", func(*http.Request, []byte) (int, string) {
		calls++
		if calls == 1 {
			return http.StatusOK, overlap
		}
		return http.StatusOK, evaluateResult
	})
	if _, err := NewKoiosChainContext(server.URL, 0, "").EvaluateTx([]byte{0x84}, []common.Utxo{onChain}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(rs.recorded()[1].body), "additionalUtxo") {
		t.Fatal("retry still sent additionalUtxo after every entry overlapped")
	}
}

func TestEvaluateTxErrors(t *testing.T) {
	scriptFailure := `{"jsonrpc":"2.0","error":{"code":3010,"message":"Some scripts of the transactions terminated with error(s).",
"data":[{"validator":{"purpose":"spend","index":0},"error":{"code":3012,"message":"validator crashed"}}]}}`
	unmatchedOverlap := fmt.Sprintf(`{"error":{"code":3002,"message":"overlap",
"data":{"overlappingOutputReferences":[{"transaction":{"id":"%s"},"index":9}]}}}`, strings.Repeat("44", 32))
	tests := map[string]struct {
		response string
		wantErr  string
	}{
		"script failure":       {scriptFailure, "code 3010"},
		"unmatched overlap":    {unmatchedOverlap, "no supplied UTxO matched"},
		"malformed overlap":    {`{"error":{"code":3002,"message":"overlap","data":{}}}`, "malformed overlap data"},
		"empty result":         {`{"jsonrpc":"2.0","result":[]}`, "no results"},
		"missing result":       {`{"jsonrpc":"2.0"}`, "no result or error"},
		"bad purpose":          {`{"result":[{"validator":{"purpose":"dance","index":0},"budget":{"memory":1,"cpu":1}}]}`, "redeemer purpose"},
		"budget out of int64":  {`{"result":[{"validator":{"purpose":"spend","index":0},"budget":{"memory":18446744073709551615,"cpu":1}}]}`, "overflow"},
		"not json":             {`<html>`, "parse"},
		"index past uint32max": {`{"result":[{"validator":{"purpose":"spend","index":4294967296},"budget":{"memory":1,"cpu":1}}]}`, "uint32"},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			rs, server := newRecordingServer(t)
			rs.handle(http.MethodPost, "/ogmios", http.StatusOK, tc.response)
			_, err := NewKoiosChainContext(server.URL, 0, "").EvaluateTx([]byte{0x84}, []common.Utxo{sampleUtxo(t, 0x22, 0)})
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error = %v, want it to contain %q", err, tc.wantErr)
			}
		})
	}
}

func TestEvaluateTxRejectsInvalidAdditionalUtxo(t *testing.T) {
	_, server := newRecordingServer(t)
	_, err := NewKoiosChainContext(server.URL, 0, "").EvaluateTx([]byte{0x84}, []common.Utxo{{}})
	if err == nil || !strings.Contains(err.Error(), "index 0") {
		t.Fatalf("error = %v, want invalid additional UTxO", err)
	}
}

func mustDecodeHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

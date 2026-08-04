package ogmios

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/blinklabs-io/gouroboros/ledger/babbage"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/mary"
	"github.com/blinklabs-io/gouroboros/ledger/shelley"

	"github.com/Salvionied/apollo/v2/backend"
)

// testOgmiosEndpoint is a syntactically valid Ogmios endpoint. Constructing a
// context performs no I/O, so no test that uses it reaches the network.
const testOgmiosEndpoint = "ws://127.0.0.1:1337"

// testChainContext builds a context from cfg, failing the test if the
// configuration is rejected.
func testChainContext(t *testing.T, cfg Config) *OgmiosChainContext {
	t.Helper()
	ctx, err := NewOgmiosChainContext(cfg)
	if err != nil {
		t.Fatalf("NewOgmiosChainContext(%+v): %v", cfg, err)
	}
	return ctx
}

// TestNewOgmiosChainContextRejectsInvalidConfig keeps an unusable
// configuration from producing a context that fails - or panics - later. The
// constructor is the only place that can still report the problem to the
// caller as a configuration error.
func TestNewOgmiosChainContextRejectsInvalidConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr string
	}{
		{
			name:    "zero config",
			cfg:     Config{},
			wantErr: "ogmios endpoint is required",
		},
		{
			name:    "blank ogmios endpoint",
			cfg:     Config{OgmiosEndpoint: "   "},
			wantErr: "ogmios endpoint is required",
		},
		{
			name:    "bare host and port",
			cfg:     Config{OgmiosEndpoint: "127.0.0.1:1337"},
			wantErr: "invalid ogmios endpoint",
		},
		{
			name:    "ogmios endpoint without scheme",
			cfg:     Config{OgmiosEndpoint: "localhost:1337"},
			wantErr: "scheme must be ws, wss, http, or https",
		},
		{
			name:    "ogmios endpoint with unusable scheme",
			cfg:     Config{OgmiosEndpoint: "ftp://127.0.0.1:1337"},
			wantErr: "scheme must be ws, wss, http, or https",
		},
		{
			name:    "ogmios endpoint without host",
			cfg:     Config{OgmiosEndpoint: "ws:///tip"},
			wantErr: "missing host",
		},
		{
			name:    "unparseable ogmios endpoint",
			cfg:     Config{OgmiosEndpoint: "://127.0.0.1:1337"},
			wantErr: "invalid ogmios endpoint",
		},
		{
			name: "kupo endpoint with unusable scheme",
			cfg: Config{
				OgmiosEndpoint: testOgmiosEndpoint,
				KupoEndpoint:   "ws://127.0.0.1:1442",
			},
			wantErr: "scheme must be http or https",
		},
		{
			name: "kupo endpoint without host",
			cfg: Config{
				OgmiosEndpoint: testOgmiosEndpoint,
				KupoEndpoint:   "http:///matches",
			},
			wantErr: "missing host",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, err := NewOgmiosChainContext(test.cfg)
			if err == nil {
				t.Fatalf("expected %+v to be rejected", test.cfg)
			}
			if ctx != nil {
				t.Fatal("a rejected configuration must not yield a context")
			}
			if !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("error = %q, want substring %q", err, test.wantErr)
			}
		})
	}
}

// TestNewOgmiosChainContextAcceptsEndpointSchemes covers the endpoints a
// caller can reasonably supply. Ogmios speaks JSON-RPC over WebSocket, so an
// http URL is dialed as ws rather than rejected.
func TestNewOgmiosChainContextAcceptsEndpointSchemes(t *testing.T) {
	for _, endpoint := range []string{
		"ws://127.0.0.1:1337",
		"wss://ogmios.example:443",
		"http://127.0.0.1:1337",
		"https://ogmios.example",
	} {
		t.Run(endpoint, func(t *testing.T) {
			testChainContext(t, Config{OgmiosEndpoint: endpoint})
		})
	}
}

// TestNewOgmiosChainContextDialsConfiguredEndpoint proves the constructor
// wires its Config through to a working Ogmios client: the query reaches the
// configured host as a WebSocket upgrade even though the endpoint was given
// as an http URL. The server answers with a plain HTTP response, so the
// handshake fails and the query returns an error rather than hanging.
func TestNewOgmiosChainContextDialsConfiguredEndpoint(t *testing.T) {
	var upgrades atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
				upgrades.Add(1)
			}
			w.WriteHeader(http.StatusBadRequest)
		},
	))
	t.Cleanup(server.Close)

	ctx := testChainContext(t, Config{
		OgmiosEndpoint: server.URL,
		NetworkId:      1,
	})
	if got := ctx.NetworkId(); got != 1 {
		t.Fatalf("NetworkId() = %d, want 1", got)
	}
	if _, err := ctx.CurrentEpochContext(t.Context()); err == nil {
		t.Fatal("expected a handshake failure against a plain HTTP server")
	}
	if got := upgrades.Load(); got != 1 {
		t.Fatalf("websocket upgrade attempts = %d, want 1", got)
	}
}

// kupoServer serves the two Kupo endpoints Apollo queries: the match list
// behind Utxos and the script lookup behind ScriptCbor.
func kupoServer(t *testing.T, scriptCborHex string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			switch {
			case strings.HasPrefix(r.URL.Path, "/v1/matches"):
				_, _ = w.Write([]byte(`[{
					"transaction_id": "` + testTxHashHex + `",
					"output_index": 1,
					"value": {"ada": {"lovelace": 2000000}}
				}]`))
			case strings.HasPrefix(r.URL.Path, "/v1/scripts/"):
				_ = json.NewEncoder(w).Encode(map[string]string{
					"language": "plutus:v2",
					"script":   scriptCborHex,
				})
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		},
	))
	t.Cleanup(server.Close)
	return server
}

// TestNewOgmiosChainContextQueriesKupo exercises the Kupo half of a context
// built from a Config end to end: settings in, client built internally,
// responses decoded into gouroboros types.
func TestNewOgmiosChainContextQueriesKupo(t *testing.T) {
	const scriptCborHex = "49480100"
	server := kupoServer(t, scriptCborHex)

	ctx := testChainContext(t, Config{
		OgmiosEndpoint: testOgmiosEndpoint,
		KupoEndpoint:   server.URL,
		KupoTimeout:    5 * time.Second,
	})
	if !backend.Supports(
		ctx, backend.CapabilityUtxos|backend.CapabilityScriptCbor,
	) {
		t.Fatal("a context configured with Kupo must report its capabilities")
	}

	utxos, err := ctx.UtxosContext(t.Context(), testAddress(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(utxos) != 1 {
		t.Fatalf("utxos = %d, want 1", len(utxos))
	}
	if got := utxos[0].Id.Index(); got != 1 {
		t.Fatalf("output index = %d, want 1", got)
	}
	if got := hex.EncodeToString(utxos[0].Id.Id().Bytes()); got != testTxHashHex {
		t.Fatalf("tx hash = %s, want %s", got, testTxHashHex)
	}
	if got := utxos[0].Output.Amount(); got.Int64() != 2_000_000 {
		t.Fatalf("lovelace = %s, want 2000000", got)
	}

	var scriptHash common.Blake2b224
	scriptCbor, err := ctx.ScriptCborContext(t.Context(), scriptHash)
	if err != nil {
		t.Fatal(err)
	}
	if got := hex.EncodeToString(scriptCbor); got != scriptCborHex {
		t.Fatalf("script cbor = %s, want %s", got, scriptCborHex)
	}
}

// stubOgmiosClient is a canned OgmiosClient. Implementing the whole interface
// in a handful of lines is the point: the injection seam stays usable without
// a server, and without naming a type Apollo does not own.
type stubOgmiosClient struct {
	protocolParams json.RawMessage
	genesis        json.RawMessage
	epoch          uint64
	tip            uint64
	txHash         common.Blake2b256
	exUnits        map[common.RedeemerKey]common.ExUnits
	utxo           *common.Utxo

	// Recorded arguments.
	eras            []string
	submitted       []byte
	evaluated       []byte
	additionalUtxos []common.Utxo
	refs            []string
}

var _ OgmiosClient = (*stubOgmiosClient)(nil)

func (s *stubOgmiosClient) ProtocolParameters(
	_ context.Context,
) (json.RawMessage, error) {
	return s.protocolParams, nil
}

func (s *stubOgmiosClient) GenesisConfig(
	_ context.Context,
	era string,
) (json.RawMessage, error) {
	s.eras = append(s.eras, era)
	return s.genesis, nil
}

func (s *stubOgmiosClient) CurrentEpoch(_ context.Context) (uint64, error) {
	return s.epoch, nil
}

func (s *stubOgmiosClient) Tip(_ context.Context) (uint64, error) {
	return s.tip, nil
}

func (s *stubOgmiosClient) SubmitTx(
	_ context.Context,
	txCbor []byte,
) (common.Blake2b256, error) {
	s.submitted = txCbor
	return s.txHash, nil
}

func (s *stubOgmiosClient) EvaluateTx(
	_ context.Context,
	txCbor []byte,
	additionalUtxos []common.Utxo,
) (map[common.RedeemerKey]common.ExUnits, error) {
	s.evaluated = txCbor
	s.additionalUtxos = additionalUtxos
	return s.exUnits, nil
}

func (s *stubOgmiosClient) UtxoByRef(
	_ context.Context,
	txHash common.Blake2b256,
	index uint32,
) (*common.Utxo, error) {
	s.refs = append(
		s.refs,
		hex.EncodeToString(txHash.Bytes())+
			"#"+strconv.FormatUint(uint64(index), 10),
	)
	return s.utxo, nil
}

// stubKupoClient is a canned KupoClient.
type stubKupoClient struct {
	utxos      []common.Utxo
	scriptCbor []byte

	addresses []string
	hashes    []string
}

var _ KupoClient = (*stubKupoClient)(nil)

func (s *stubKupoClient) UtxosAtAddress(
	_ context.Context,
	address common.Address,
) ([]common.Utxo, error) {
	s.addresses = append(s.addresses, address.String())
	return s.utxos, nil
}

func (s *stubKupoClient) ScriptCbor(
	_ context.Context,
	scriptHash common.Blake2b224,
) ([]byte, error) {
	s.hashes = append(s.hashes, hex.EncodeToString(scriptHash.Bytes()))
	return s.scriptCbor, nil
}

// testStubUtxo is a resolved UTxO the stubs can hand back.
func testStubUtxo(t *testing.T) common.Utxo {
	t.Helper()
	var txId common.Blake2b256
	for i := range txId {
		txId[i] = 0x22
	}
	return common.Utxo{
		Id: shelley.ShelleyTransactionInput{TxId: txId, OutputIndex: 7},
		Output: &babbage.BabbageTransactionOutput{
			OutputAddress: testAddress(t),
			OutputAmount:  mary.MaryTransactionOutputValue{Amount: 3_000_000},
		},
	}
}

// TestInjectedClientsServeEveryChainContextMethod walks the whole
// ChainContext surface against the stubs: every method must go through the
// injected clients, and Apollo must still own the decoding of the responses
// it asked for as raw JSON.
func TestInjectedClientsServeEveryChainContextMethod(t *testing.T) {
	var txHash common.Blake2b256
	for i := range txHash {
		txHash[i] = 0x33
	}
	utxo := testStubUtxo(t)
	ogmiosClient := &stubOgmiosClient{
		protocolParams: json.RawMessage(protocolParamsBody(adaLovelace)),
		genesis:        json.RawMessage(ogmiosV6ShelleyGenesisBody),
		epoch:          507,
		tip:            123456,
		txHash:         txHash,
		exUnits: map[common.RedeemerKey]common.ExUnits{
			{Tag: common.RedeemerTagSpend, Index: 0}: {
				Memory: 1700,
				Steps:  476468,
			},
		},
		utxo: &utxo,
	}
	kupoClient := &stubKupoClient{
		utxos:      []common.Utxo{utxo},
		scriptCbor: []byte{0x49, 0x48, 0x01, 0x00},
	}
	ctx, err := NewOgmiosChainContextFromClients(ogmiosClient, kupoClient, 1)
	if err != nil {
		t.Fatal(err)
	}

	if !backend.Supports(
		ctx, backend.CapabilityUtxos|backend.CapabilityScriptCbor,
	) {
		t.Fatal("an injected Kupo client must report its capabilities")
	}
	if got := ctx.NetworkId(); got != 1 {
		t.Fatalf("NetworkId() = %d, want 1", got)
	}

	pp, err := ctx.ProtocolParams()
	if err != nil {
		t.Fatal(err)
	}
	assertOgmiosV6ProtocolParams(t, pp)

	gp, err := ctx.GenesisParams()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := gp.NetworkMagic, 764824073; got != want {
		t.Fatalf("NetworkMagic = %d, want %d", got, want)
	}
	if len(ogmiosClient.eras) != 1 || ogmiosClient.eras[0] != "shelley" {
		t.Fatalf("genesis eras queried = %v, want [shelley]", ogmiosClient.eras)
	}

	if epoch, err := ctx.CurrentEpoch(); err != nil || epoch != 507 {
		t.Fatalf("CurrentEpoch() = %d, %v, want 507, nil", epoch, err)
	}
	if tip, err := ctx.Tip(); err != nil || tip != 123456 {
		t.Fatalf("Tip() = %d, %v, want 123456, nil", tip, err)
	}
	if fee, err := ctx.MaxTxFee(); err != nil || fee == 0 {
		t.Fatalf("MaxTxFee() = %d, %v, want a nonzero fee", fee, err)
	}

	txCbor := []byte{0x84, 0xA0}
	hash, err := ctx.SubmitTx(txCbor)
	if err != nil {
		t.Fatal(err)
	}
	if hash != txHash {
		t.Fatalf("SubmitTx hash = %x, want %x", hash, txHash)
	}
	if string(ogmiosClient.submitted) != string(txCbor) {
		t.Fatalf("submitted CBOR = %x, want %x", ogmiosClient.submitted, txCbor)
	}

	additional := []common.Utxo{sampleCommonUtxo(t)}
	exUnits, err := ctx.EvaluateTx(txCbor, additional)
	if err != nil {
		t.Fatal(err)
	}
	key := common.RedeemerKey{Tag: common.RedeemerTagSpend, Index: 0}
	if eu := exUnits[key]; eu.Memory != 1700 || eu.Steps != 476468 {
		t.Fatalf("unexpected budget %+v", eu)
	}
	if len(ogmiosClient.additionalUtxos) != 1 {
		t.Fatalf(
			"additional UTxOs forwarded = %d, want 1",
			len(ogmiosClient.additionalUtxos),
		)
	}

	byRef, err := ctx.UtxoByRef(txHash, 7)
	if err != nil {
		t.Fatal(err)
	}
	if byRef == nil || byRef.Id.Index() != 7 {
		t.Fatalf("UtxoByRef() = %+v, want the stub UTxO", byRef)
	}

	utxos, err := ctx.Utxos(testAddress(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(utxos) != 1 {
		t.Fatalf("Utxos() returned %d UTxOs, want 1", len(utxos))
	}
	if len(kupoClient.addresses) != 1 ||
		kupoClient.addresses[0] != testAddress(t).String() {
		t.Fatalf("queried addresses = %v", kupoClient.addresses)
	}

	scriptCbor, err := ctx.ScriptCbor(common.Blake2b224{})
	if err != nil {
		t.Fatal(err)
	}
	if len(scriptCbor) != 4 {
		t.Fatalf("ScriptCbor() = %x, want 4 bytes", scriptCbor)
	}
}

// TestEvaluateTxValidatesAdditionalUtxosForEveryClient keeps the malformed
// additional UTxO check in the context, where it holds no matter which client
// is injected.
func TestEvaluateTxValidatesAdditionalUtxosForEveryClient(t *testing.T) {
	ogmiosClient := &stubOgmiosClient{}
	ctx, err := NewOgmiosChainContextFromClients(ogmiosClient, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	malformed := []common.Utxo{{Output: sampleCommonUtxo(t).Output}}
	if _, err := ctx.EvaluateTx([]byte{0x84}, malformed); err == nil {
		t.Fatal("expected malformed UTxO error")
	} else if !strings.Contains(err.Error(), "missing transaction input") {
		t.Fatalf("error = %q, want a missing-input error", err)
	}
	if ogmiosClient.evaluated != nil {
		t.Fatal("a malformed UTxO must not reach the client")
	}
}

// TestNewOgmiosChainContextFromClientsRejectsNilOgmiosClient covers both
// spellings of a missing client: an untyped nil and a nil pointer stored in
// the interface. Neither may produce a context that fails on first use.
func TestNewOgmiosChainContextFromClientsRejectsNilOgmiosClient(t *testing.T) {
	var typedNil *stubOgmiosClient
	clients := []struct {
		name   string
		client OgmiosClient
	}{
		{"untyped nil", nil},
		{"typed nil", typedNil},
	}
	for _, test := range clients {
		t.Run(test.name, func(t *testing.T) {
			ctx, err := NewOgmiosChainContextFromClients(test.client, nil, 0)
			if err == nil {
				t.Fatal("expected a missing Ogmios client to be rejected")
			}
			if ctx != nil {
				t.Fatal("a rejected client must not yield a context")
			}
		})
	}
}

// TestNewOgmiosChainContextFromClientsNormalizesTypedNilKupo keeps a nil
// pointer passed as the Kupo client from being reported as a working one.
func TestNewOgmiosChainContextFromClientsNormalizesTypedNilKupo(t *testing.T) {
	var typedNil *stubKupoClient
	ctx, err := NewOgmiosChainContextFromClients(
		&stubOgmiosClient{}, typedNil, 0,
	)
	if err != nil {
		t.Fatal(err)
	}
	if backend.Supports(
		ctx, backend.CapabilityUtxos|backend.CapabilityScriptCbor,
	) {
		t.Fatal("a nil Kupo client must not report Kupo capabilities")
	}
	if _, err := ctx.Utxos(testAddress(t)); !errors.Is(
		err, backend.ErrUnsupported,
	) {
		t.Fatalf("Utxos() error = %v, want ErrUnsupported", err)
	}
}

// TestZeroValueChainContextReturnsErrors pins the audit fix: a context no
// constructor produced has no client, and every method must report that
// rather than dereference nil.
func TestZeroValueChainContextReturnsErrors(t *testing.T) {
	ctx := &OgmiosChainContext{}

	if got := ctx.NetworkId(); got != 0 {
		t.Fatalf("NetworkId() = %d, want 0", got)
	}
	if got := ctx.Capabilities(); got != 0 {
		t.Fatalf("Capabilities() = %v, want no capabilities", got)
	}

	tests := []struct {
		name string
		want error
		call func() error
	}{
		{"ProtocolParams", errNoOgmiosClient, func() error {
			_, err := ctx.ProtocolParams()
			return err
		}},
		{"GenesisParams", errNoOgmiosClient, func() error {
			_, err := ctx.GenesisParams()
			return err
		}},
		{"CurrentEpoch", errNoOgmiosClient, func() error {
			_, err := ctx.CurrentEpoch()
			return err
		}},
		{"MaxTxFee", errNoOgmiosClient, func() error {
			_, err := ctx.MaxTxFee()
			return err
		}},
		{"Tip", errNoOgmiosClient, func() error {
			_, err := ctx.Tip()
			return err
		}},
		{"SubmitTx", errNoOgmiosClient, func() error {
			_, err := ctx.SubmitTx([]byte{0x84})
			return err
		}},
		{"EvaluateTx", errNoOgmiosClient, func() error {
			_, err := ctx.EvaluateTx([]byte{0x84}, nil)
			return err
		}},
		{"UtxoByRef", errNoOgmiosClient, func() error {
			_, err := ctx.UtxoByRef(common.Blake2b256{}, 0)
			return err
		}},
		{"Utxos", backend.ErrUnsupported, func() error {
			_, err := ctx.Utxos(testAddress(t))
			return err
		}},
		{"ScriptCbor", backend.ErrUnsupported, func() error {
			_, err := ctx.ScriptCbor(common.Blake2b224{})
			return err
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.call()
			if err == nil {
				t.Fatal("expected an error, got nil")
			}
			if !errors.Is(err, test.want) {
				t.Fatalf("error = %v, want %v", err, test.want)
			}
		})
	}
}

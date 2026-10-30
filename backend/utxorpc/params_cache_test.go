package utxorpc

import (
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	alphacardano "github.com/utxorpc/go-codegen/utxorpc/v1alpha/cardano"
	alphaquery "github.com/utxorpc/go-codegen/utxorpc/v1alpha/query"
	cardano "github.com/utxorpc/go-codegen/utxorpc/v1beta/cardano"
	query "github.com/utxorpc/go-codegen/utxorpc/v1beta/query"

	"github.com/Salvionied/apollo/v2/backend"
)

// queryServer is a gRPC server for the query RPCs behind protocol and genesis
// parameters. It answers each RPC from replies, keyed by method name, and
// counts the calls it receives. A method without a reply is UNIMPLEMENTED.
type queryServer struct {
	mu      sync.Mutex
	replies map[string]queryReply
	calls   map[string]int
}

type queryReply struct {
	msg    proto.Message
	status string // gRPC status code; "" means OK
}

func newQueryServer(t *testing.T, replies map[string]queryReply) (*queryServer, *UtxoRpcChainContext) {
	t.Helper()
	s := &queryServer{replies: replies, calls: map[string]int{}}
	srv := httptest.NewUnstartedServer(s)
	protocols := new(http.Protocols)
	protocols.SetUnencryptedHTTP2(true)
	srv.Config.Protocols = protocols
	srv.Start()
	t.Cleanup(srv.Close)
	return s, NewUtxoRpcChainContext(srv.URL, 1, nil)
}

func (s *queryServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	method := req.URL.Path[strings.LastIndex(req.URL.Path, "/")+1:]
	if strings.Contains(req.URL.Path, "utxorpc.v1alpha.") {
		method = "alpha." + method
	}
	s.mu.Lock()
	s.calls[method]++
	reply, ok := s.replies[method]
	s.mu.Unlock()

	w.Header().Set("Content-Type", "application/grpc")
	w.Header().Set("Trailer", "Grpc-Status, Grpc-Message")
	w.WriteHeader(http.StatusOK)
	switch {
	case !ok:
		w.Header().Set("Grpc-Status", "12") // UNIMPLEMENTED
	case reply.status != "":
		w.Header().Set("Grpc-Status", reply.status)
	default:
		encoded, err := proto.Marshal(reply.msg)
		if err != nil {
			w.Header().Set("Grpc-Status", "13") // INTERNAL
			return
		}
		_, _ = w.Write(grpcFrame(encoded))
		w.Header().Set("Grpc-Status", "0")
	}
}

func (s *queryServer) callCount(method string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls[method]
}

func (s *queryServer) setReply(method string, reply queryReply) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.replies[method] = reply
}

func testPParams() *cardano.PParams {
	return &cardano.PParams{
		MinFeeCoefficient: &cardano.BigInt{BigInt: &cardano.BigInt_Int{Int: 44}},
		MinFeeConstant:    &cardano.BigInt{BigInt: &cardano.BigInt_Int{Int: 155381}},
		MaxTxSize:         16384,
		CostModels: &cardano.CostModels{
			PlutusV3: &cardano.CostModel{Values: []int64{100788, 420, 1}},
		},
		MinFeeScriptRefCostPerByte: &cardano.RationalNumber{Numerator: 15, Denominator: 1},
	}
}

func paramsReply() queryReply {
	return queryReply{msg: &query.ReadParamsResponse{
		Values: &query.AnyChainParams{
			Params: &query.AnyChainParams_Cardano{Cardano: testPParams()},
		},
	}}
}

func testGenesis() *cardano.Genesis {
	return &cardano.Genesis{
		ActiveSlotsCoeff:  &cardano.RationalNumber{Numerator: 1, Denominator: 20},
		EpochLength:       432000,
		MaxKesEvolutions:  62,
		MaxLovelaceSupply: &cardano.BigInt{BigInt: &cardano.BigInt_Int{Int: 45000000000000000}},
		NetworkMagic:      764824073,
		SecurityParam:     2160,
		SlotLength:        1000,
		SlotsPerKesPeriod: 129600,
		SystemStart:       "2017-09-23T21:44:51Z",
		UpdateQuorum:      5,
	}
}

func genesisReply(g *cardano.Genesis) queryReply {
	return queryReply{msg: &query.ReadGenesisResponse{
		Config: &query.ReadGenesisResponse_Cardano{Cardano: g},
	}}
}

// TestProtocolParamsCached covers the transaction builder's call pattern: it
// asks for protocol parameters and the max fee many times while building one
// transaction, which must not become one request per call.
func TestProtocolParamsCached(t *testing.T) {
	server, ctx := newQueryServer(t, map[string]queryReply{"ReadParams": paramsReply()})

	for range 5 {
		pp, err := ctx.ProtocolParams()
		if err != nil {
			t.Fatal(err)
		}
		if pp.MinFeeConstant != 155381 {
			t.Fatalf("MinFeeConstant = %d, want 155381", pp.MinFeeConstant)
		}
		if _, err := ctx.MaxTxFee(); err != nil {
			t.Fatal(err)
		}
	}
	if n := server.callCount("ReadParams"); n != 1 {
		t.Fatalf("ReadParams called %d times, want 1", n)
	}
}

func TestProtocolParamsCacheReturnsCopies(t *testing.T) {
	server, ctx := newQueryServer(t, map[string]queryReply{"ReadParams": paramsReply()})

	first, err := ctx.ProtocolParams()
	if err != nil {
		t.Fatal(err)
	}
	first.CostModels["PlutusV3"][0] = -1
	first.CostModels["PlutusV9"] = []int64{1}
	first.MinFeeRefScriptCostPerByteRational.SetInt64(-1)

	second, err := ctx.ProtocolParams()
	if err != nil {
		t.Fatal(err)
	}
	if got := second.CostModels["PlutusV3"][0]; got != 100788 {
		t.Fatalf("cached cost model entry = %d, want 100788", got)
	}
	if _, ok := second.CostModels["PlutusV9"]; ok {
		t.Fatal("a cost model added to a returned value reached the cache")
	}
	if got := second.MinFeeRefScriptCostPerByteRational; got.Cmp(big.NewRat(15, 1)) != 0 {
		t.Fatalf("cached ref script price = %s, want 15", got)
	}
	if n := server.callCount("ReadParams"); n != 1 {
		t.Fatalf("ReadParams called %d times, want 1", n)
	}
}

func TestProtocolParamsCacheExpires(t *testing.T) {
	server, ctx := newQueryServer(t, map[string]queryReply{"ReadParams": paramsReply()})

	if _, err := ctx.ProtocolParams(); err != nil {
		t.Fatal(err)
	}
	ctx.cacheMu.Lock()
	ctx.paramsCacheAt = ctx.paramsCacheAt.Add(-cacheExpiry)
	ctx.cacheMu.Unlock()
	if _, err := ctx.ProtocolParams(); err != nil {
		t.Fatal(err)
	}
	if n := server.callCount("ReadParams"); n != 2 {
		t.Fatalf("ReadParams called %d times, want 2", n)
	}
}

func TestProtocolParamsFailuresNotCached(t *testing.T) {
	tests := map[string]queryReply{
		"server error": {status: "14"}, // UNAVAILABLE
		"no cardano params": {msg: &query.ReadParamsResponse{
			Values: &query.AnyChainParams{},
		}},
	}
	for name, reply := range tests {
		t.Run(name, func(t *testing.T) {
			server, ctx := newQueryServer(t, map[string]queryReply{"ReadParams": reply})
			if _, err := ctx.ProtocolParams(); err == nil {
				t.Fatal("expected error")
			}
			server.setReply("ReadParams", paramsReply())
			if _, err := ctx.ProtocolParams(); err != nil {
				t.Fatal(err)
			}
			if n := server.callCount("ReadParams"); n != 2 {
				t.Fatalf("ReadParams called %d times, want 2", n)
			}
		})
	}
}

func TestGenesisParamsDecodesReadGenesis(t *testing.T) {
	_, ctx := newQueryServer(t, map[string]queryReply{"ReadGenesis": genesisReply(testGenesis())})
	if !backend.Supports(ctx, backend.CapabilityGenesisParams) {
		t.Fatal("UTxO RPC does not report CapabilityGenesisParams")
	}

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
}

func TestGenesisParamsFallsBackToV1Alpha(t *testing.T) {
	alphaGenesis := &alphacardano.Genesis{}
	if err := transcode(testGenesis(), alphaGenesis); err != nil {
		t.Fatal(err)
	}
	server, ctx := newQueryServer(t, map[string]queryReply{
		"alpha.ReadGenesis": {msg: &alphaquery.ReadGenesisResponse{
			Config: &alphaquery.ReadGenesisResponse_Cardano{Cardano: alphaGenesis},
		}},
	})

	gp, err := ctx.GenesisParams()
	if err != nil {
		t.Fatal(err)
	}
	if gp.NetworkMagic != 764824073 || gp.SlotLength != 1 {
		t.Fatalf("GenesisParams() = %+v, want mainnet values", gp)
	}
	if n := server.callCount("ReadGenesis"); n != 1 {
		t.Fatalf("v1beta ReadGenesis called %d times, want 1", n)
	}
}

// TestGenesisParamsIgnoresNegotiatedVersion covers servers whose versions
// differ in which RPCs they implement: once another call has settled on one
// version, ReadGenesis must still reach the other when only it serves
// genesis, and doing so must not move the other calls to that version.
func TestGenesisParamsIgnoresNegotiatedVersion(t *testing.T) {
	alphaGenesis := &alphacardano.Genesis{}
	if err := transcode(testGenesis(), alphaGenesis); err != nil {
		t.Fatal(err)
	}
	alphaParams := &alphacardano.PParams{}
	if err := transcode(testPParams(), alphaParams); err != nil {
		t.Fatal(err)
	}
	tests := map[string]struct {
		replies       map[string]queryReply
		paramsMethod  string
		genesisMethod string
	}{
		"v1beta params, v1alpha genesis": {
			replies: map[string]queryReply{
				"ReadParams": paramsReply(),
				"alpha.ReadGenesis": {msg: &alphaquery.ReadGenesisResponse{
					Config: &alphaquery.ReadGenesisResponse_Cardano{Cardano: alphaGenesis},
				}},
			},
			paramsMethod:  "ReadParams",
			genesisMethod: "alpha.ReadGenesis",
		},
		"v1alpha params, v1beta genesis": {
			replies: map[string]queryReply{
				"alpha.ReadParams": {msg: &alphaquery.ReadParamsResponse{
					Values: &alphaquery.AnyChainParams{
						Params: &alphaquery.AnyChainParams_Cardano{Cardano: alphaParams},
					},
				}},
				"ReadGenesis": genesisReply(testGenesis()),
			},
			paramsMethod:  "alpha.ReadParams",
			genesisMethod: "ReadGenesis",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			server, ctx := newQueryServer(t, test.replies)
			if _, err := ctx.ProtocolParams(); err != nil {
				t.Fatal(err)
			}
			gp, err := ctx.GenesisParams()
			if err != nil {
				t.Fatal(err)
			}
			if gp.NetworkMagic != 764824073 {
				t.Fatalf("GenesisParams() = %+v, want mainnet values", gp)
			}
			if n := server.callCount(test.genesisMethod); n != 1 {
				t.Fatalf("%s called %d times, want 1", test.genesisMethod, n)
			}

			// Parameters still come from the version negotiated for them.
			ctx.cacheMu.Lock()
			ctx.cachedParams = nil
			ctx.cacheMu.Unlock()
			if _, err := ctx.ProtocolParams(); err != nil {
				t.Fatal(err)
			}
			if n := server.callCount(test.paramsMethod); n != 2 {
				t.Fatalf("%s called %d times, want 2", test.paramsMethod, n)
			}
		})
	}
}

func TestGenesisParamsCached(t *testing.T) {
	server, ctx := newQueryServer(t, map[string]queryReply{"ReadGenesis": genesisReply(testGenesis())})

	for range 3 {
		if _, err := ctx.GenesisParams(); err != nil {
			t.Fatal(err)
		}
	}
	if n := server.callCount("ReadGenesis"); n != 1 {
		t.Fatalf("ReadGenesis called %d times, want 1", n)
	}

	ctx.cacheMu.Lock()
	ctx.genesisCacheAt = time.Now().Add(-cacheExpiry)
	ctx.cacheMu.Unlock()
	if _, err := ctx.GenesisParams(); err != nil {
		t.Fatal(err)
	}
	if n := server.callCount("ReadGenesis"); n != 2 {
		t.Fatalf("ReadGenesis called %d times after expiry, want 2", n)
	}
}

func TestGenesisParamsRejectsMalformed(t *testing.T) {
	tests := map[string]func(*cardano.Genesis){
		"missing active slots coefficient": func(g *cardano.Genesis) { g.ActiveSlotsCoeff = nil },
		"zero coefficient denominator": func(g *cardano.Genesis) {
			g.ActiveSlotsCoeff = &cardano.RationalNumber{Numerator: 1}
		},
		"negative coefficient": func(g *cardano.Genesis) {
			g.ActiveSlotsCoeff = &cardano.RationalNumber{Numerator: -1, Denominator: 20}
		},
		"coefficient above one": func(g *cardano.Genesis) {
			g.ActiveSlotsCoeff = &cardano.RationalNumber{Numerator: 21, Denominator: 20}
		},
		"missing epoch length":    func(g *cardano.Genesis) { g.EpochLength = 0 },
		"missing slot length":     func(g *cardano.Genesis) { g.SlotLength = 0 },
		"missing system start":    func(g *cardano.Genesis) { g.SystemStart = "" },
		"malformed system start":  func(g *cardano.Genesis) { g.SystemStart = "yesterday" },
		"missing max lovelace":    func(g *cardano.Genesis) { g.MaxLovelaceSupply = nil },
		"negative max lovelace":   func(g *cardano.Genesis) { g.MaxLovelaceSupply = &cardano.BigInt{BigInt: &cardano.BigInt_Int{Int: -1}} },
		"sub-second slot length":  func(g *cardano.Genesis) { g.SlotLength = 200 },
		"fractional slot seconds": func(g *cardano.Genesis) { g.SlotLength = 1500 },
	}
	for name, corrupt := range tests {
		t.Run(name, func(t *testing.T) {
			g := testGenesis()
			corrupt(g)
			_, ctx := newQueryServer(t, map[string]queryReply{"ReadGenesis": genesisReply(g)})
			if _, err := ctx.GenesisParams(); err == nil {
				t.Fatal("expected error")
			}
		})
	}

	t.Run("no cardano config", func(t *testing.T) {
		_, ctx := newQueryServer(t, map[string]queryReply{
			"ReadGenesis": {msg: &query.ReadGenesisResponse{}},
		})
		if _, err := ctx.GenesisParams(); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestGenesisParamsFailuresNotCached(t *testing.T) {
	g := testGenesis()
	g.SystemStart = ""
	server, ctx := newQueryServer(t, map[string]queryReply{"ReadGenesis": genesisReply(g)})

	if _, err := ctx.GenesisParams(); err == nil {
		t.Fatal("expected error")
	}
	server.setReply("ReadGenesis", genesisReply(testGenesis()))
	if _, err := ctx.GenesisParams(); err != nil {
		t.Fatal(err)
	}
	if n := server.callCount("ReadGenesis"); n != 2 {
		t.Fatalf("ReadGenesis called %d times, want 2", n)
	}
}

// TestGenesisParamsUnimplementedIsUnsupported covers servers that predate
// ReadGenesis: the failure must classify as ErrUnsupported, as it did before
// the backend could ask, rather than as an opaque transport error.
func TestGenesisParamsUnimplementedIsUnsupported(t *testing.T) {
	_, ctx := newQueryServer(t, map[string]queryReply{})

	_, err := ctx.GenesisParams()
	if !errors.Is(err, backend.ErrUnsupported) {
		t.Fatalf("got %v, want ErrUnsupported", err)
	}
	var unsupported *backend.UnsupportedError
	if !errors.As(err, &unsupported) || unsupported.Capability != backend.CapabilityGenesisParams {
		t.Fatalf("unexpected unsupported error: %#v", err)
	}
}

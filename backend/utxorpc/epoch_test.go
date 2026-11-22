package utxorpc

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	alphacardano "github.com/utxorpc/go-codegen/utxorpc/v1alpha/cardano"
	alphaquery "github.com/utxorpc/go-codegen/utxorpc/v1alpha/query"
	cardano "github.com/utxorpc/go-codegen/utxorpc/v1beta/cardano"
	query "github.com/utxorpc/go-codegen/utxorpc/v1beta/query"
	syncpb "github.com/utxorpc/go-codegen/utxorpc/v1beta/sync"

	"github.com/Salvionied/apollo/v2/backend"
)

func boundary(slot, epoch uint64) *cardano.EraBoundary {
	return &cardano.EraBoundary{Slot: slot, Epoch: epoch}
}

// mainnetLikeEras has a Byron era of 4492800 slots (208 epochs of 21600)
// followed by a Shelley-length open-ended era.
func mainnetLikeEras() []*cardano.EraSummary {
	return []*cardano.EraSummary{
		{Name: "byron", Start: boundary(0, 0), End: boundary(4492800, 208)},
		{Name: "shelley", Start: boundary(4492800, 208), End: boundary(16588800, 236)},
		{Name: "conway", Start: boundary(16588800, 236)},
	}
}

func protocolMajor(major uint32) *cardano.PParams {
	return &cardano.PParams{ProtocolVersion: &cardano.ProtocolVersion{Major: major}}
}

func TestCurrentEpochUnnamedShelleyEraByProtocolVersion(t *testing.T) {
	t.Parallel()
	_, ctx := newQueryServer(t, epochReplies(432000*3+7,
		&cardano.EraSummary{Start: boundary(0, 0), ProtocolParams: protocolMajor(2)},
	))
	if got, err := ctx.CurrentEpoch(); err != nil || got != 3 {
		t.Fatalf("CurrentEpoch() = %d, %v, want 3", got, err)
	}
}

func eraReply(eras ...*cardano.EraSummary) queryReply {
	return queryReply{msg: &query.ReadEraSummaryResponse{
		Summary: &query.ReadEraSummaryResponse_Cardano{
			Cardano: &cardano.EraSummaries{Summaries: eras},
		},
	}}
}

func tipReply(slot uint64) queryReply {
	return queryReply{msg: &syncpb.ReadTipResponse{Tip: &syncpb.BlockRef{Slot: slot}}}
}

func epochReplies(tip uint64, eras ...*cardano.EraSummary) map[string]queryReply {
	return map[string]queryReply{
		"ReadEraSummary": eraReply(eras...),
		"ReadGenesis":    genesisReply(testGenesis()),
		"ReadTip":        tipReply(tip),
	}
}

func TestCurrentEpochReportedCapability(t *testing.T) {
	t.Parallel()
	ctx := NewUtxoRpcChainContext("http://localhost", 0, nil)
	if !backend.Supports(ctx, backend.CapabilityCurrentEpoch) {
		t.Fatal("UTxO RPC does not report CapabilityCurrentEpoch")
	}
}

func TestCurrentEpochComputesFromEraSummary(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		tip  uint64
		want uint64
	}{
		{"era start", 4492800, 208},
		{"inside first epoch", 4492800 + 432000 - 1, 208},
		{"second epoch start", 4492800 + 432000, 209},
		{"closed era last slot", 16588800 - 1, 235},
		{"open era start", 16588800, 236},
		{"open era later", 16588800 + 3*432000 + 17, 239},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, ctx := newQueryServer(t, epochReplies(tt.tip, mainnetLikeEras()...))
			got, err := ctx.CurrentEpoch()
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("CurrentEpoch() = %d, want %d", got, tt.want)
			}
		})
	}
}

// The full era tables a server returns for the public networks: every era
// from Byron on, each closed one ending where the next starts. On preview,
// which opens in Alonzo, Byron through Mary are empty eras at slot 0.
func TestCurrentEpochPublicNetworkEraTables(t *testing.T) {
	t.Parallel()
	mainnet := []*cardano.EraSummary{
		{Name: "byron", Start: boundary(0, 0), End: boundary(4492800, 208)},
		{Name: "shelley", Start: boundary(4492800, 208), End: boundary(16588800, 236)},
		{Name: "allegra", Start: boundary(16588800, 236), End: boundary(23068800, 251)},
		{Name: "mary", Start: boundary(23068800, 251), End: boundary(39916800, 290)},
		{Name: "alonzo", Start: boundary(39916800, 290), End: boundary(72316800, 365)},
		{Name: "babbage", Start: boundary(72316800, 365), End: boundary(133660800, 507)},
		{Name: "conway", Start: boundary(133660800, 507)},
	}
	preview := []*cardano.EraSummary{
		{Name: "byron", Start: boundary(0, 0), End: boundary(0, 0)},
		{Name: "shelley", Start: boundary(0, 0), End: boundary(0, 0)},
		{Name: "allegra", Start: boundary(0, 0), End: boundary(0, 0)},
		{Name: "mary", Start: boundary(0, 0), End: boundary(0, 0)},
		{Name: "alonzo", Start: boundary(0, 0), End: boundary(259200, 3)},
		{Name: "babbage", Start: boundary(259200, 3), End: boundary(55814400, 646)},
		{Name: "conway", Start: boundary(55814400, 646)},
	}
	tests := []struct {
		name        string
		eras        []*cardano.EraSummary
		epochLength uint32
		tip         uint64
		want        uint64
	}{
		{"mainnet conway", mainnet, 432000, 133660800 + 10*432000 + 5, 517},
		{"mainnet babbage", mainnet, 432000, 133660800 - 1, 506},
		{"preview conway", preview, 86400, 55814400 + 5*86400 + 1, 651},
		{"preview babbage", preview, 86400, 259200, 3},
		{"preview first slot", preview, 86400, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			replies := epochReplies(tt.tip, tt.eras...)
			g := testGenesis()
			g.EpochLength = tt.epochLength
			replies["ReadGenesis"] = genesisReply(g)
			_, ctx := newQueryServer(t, replies)
			got, err := ctx.CurrentEpoch()
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("CurrentEpoch() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestCurrentEpochFallsBackToV1Alpha(t *testing.T) {
	t.Parallel()
	alphaEras := &alphacardano.EraSummaries{}
	if err := transcode(&cardano.EraSummaries{Summaries: mainnetLikeEras()}, alphaEras); err != nil {
		t.Fatal(err)
	}
	server, ctx := newQueryServer(t, map[string]queryReply{
		"alpha.ReadEraSummary": {msg: &alphaquery.ReadEraSummaryResponse{
			Summary: &alphaquery.ReadEraSummaryResponse_Cardano{Cardano: alphaEras},
		}},
		"ReadGenesis": genesisReply(testGenesis()),
		"ReadTip":     tipReply(16588800 + 432000),
	})
	got, err := ctx.CurrentEpoch()
	if err != nil {
		t.Fatal(err)
	}
	if got != 237 {
		t.Fatalf("CurrentEpoch() = %d, want 237", got)
	}
	if n := server.callCount("ReadEraSummary"); n != 1 {
		t.Fatalf("v1beta ReadEraSummary called %d times, want 1", n)
	}
	if n := server.callCount("alpha.ReadEraSummary"); n != 1 {
		t.Fatalf("v1alpha ReadEraSummary called %d times, want 1", n)
	}
}

func TestCurrentEpochUnimplementedIsUnsupported(t *testing.T) {
	t.Parallel()
	_, ctx := newQueryServer(t, map[string]queryReply{
		"ReadGenesis": genesisReply(testGenesis()),
		"ReadTip":     tipReply(16588800),
	})
	_, err := ctx.CurrentEpoch()
	if !errors.Is(err, backend.ErrUnsupported) {
		t.Fatalf("got %v, want ErrUnsupported", err)
	}
	var unsupported *backend.UnsupportedError
	if !errors.As(err, &unsupported) || unsupported.Capability != backend.CapabilityCurrentEpoch {
		t.Fatalf("unexpected unsupported error: %#v", err)
	}
}

func TestCurrentEpochRejectsMalformedSummary(t *testing.T) {
	t.Parallel()
	shelley := func() *cardano.EraSummary {
		return &cardano.EraSummary{Name: "shelley", Start: boundary(1000, 5)}
	}
	tests := []struct {
		name    string
		tip     uint64
		eras    []*cardano.EraSummary
		wantErr string
	}{
		{"no eras", 5000, nil, "era summary has no eras"},
		{"tip before first era", 999, []*cardano.EraSummary{shelley()}, "no era in the summary contains tip slot 999"},
		{"nil start", 5000, []*cardano.EraSummary{{Name: "shelley"}}, `era "shelley" has no start boundary`},
		{"tip in byron era", 5000, []*cardano.EraSummary{
			{Name: "Byron", Start: boundary(0, 0), End: boundary(432000, 1)},
			{Name: "shelley", Start: boundary(432000, 1)},
		}, `tip slot 5000 is in Byron era "Byron"`},
		{"tip in unnamed byron era", 5000, []*cardano.EraSummary{
			{Start: boundary(0, 0), ProtocolParams: protocolMajor(1)},
		}, `tip slot 5000 is in Byron era ""`},
		{"closed era disagrees with epoch length", 5000, []*cardano.EraSummary{
			{Name: "shelley", Start: boundary(0, 0), End: boundary(10000, 1)},
		}, `era "shelley" spans 10000 slots over 1 epochs, inconsistent with epoch length 432000`},
		{"closed era not a whole number of epochs", 5000, []*cardano.EraSummary{
			{Name: "shelley", Start: boundary(0, 0), End: boundary(432005, 1)},
		}, `era "shelley" spans 432005 slots over 1 epochs, inconsistent with epoch length 432000`},
		{"end before start", 5000, []*cardano.EraSummary{
			{Name: "shelley", Start: boundary(5000, 0), End: boundary(4000, 0)},
			{Name: "conway", Start: boundary(0, 0)},
		}, `era "shelley" ends at slot 4000, before its start at slot 5000`},
		{"overlapping eras", 5000, []*cardano.EraSummary{
			{Name: "a", Start: boundary(0, 0)},
			{Name: "b", Start: boundary(1000, 1)},
		}, `eras "a" and "b" both contain slot 5000`},
		{"gap before tip", 5000, []*cardano.EraSummary{
			{Name: "a", Start: boundary(0, 0), End: boundary(1000, 0)},
		}, "no era in the summary contains tip slot 5000"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, ctx := newQueryServer(t, epochReplies(tt.tip, tt.eras...))
			got, err := ctx.CurrentEpoch()
			if err == nil {
				t.Fatalf("CurrentEpoch() = %d, want error %q", got, tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("CurrentEpoch() error = %v, want %q", err, tt.wantErr)
			}
		})
	}

	t.Run("no cardano summary", func(t *testing.T) {
		t.Parallel()
		replies := epochReplies(5000)
		replies["ReadEraSummary"] = queryReply{msg: &query.ReadEraSummaryResponse{}}
		_, ctx := newQueryServer(t, replies)
		const want = "no cardano era summary in response"
		if got, err := ctx.CurrentEpoch(); err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("CurrentEpoch() = %d, %v, want error %q", got, err, want)
		}
	})
}

// GenesisParamsContext already rejects a zero epoch length, so epochAtSlot's
// own guard against dividing by zero is only reachable directly.
func TestEpochAtSlotRejectsZeroEpochLength(t *testing.T) {
	t.Parallel()
	_, err := epochAtSlot(mainnetLikeEras(), 16588800, 0)
	if err == nil || !strings.Contains(err.Error(), "epoch length is zero") {
		t.Fatalf("epochAtSlot() error = %v, want epoch length is zero", err)
	}
}

// A zero genesis epoch length is reported by the genesis query, before any
// era summary is read.
func TestCurrentEpochZeroGenesisEpochLength(t *testing.T) {
	t.Parallel()
	replies := epochReplies(5000, mainnetLikeEras()...)
	g := testGenesis()
	g.EpochLength = 0
	replies["ReadGenesis"] = genesisReply(g)
	server, ctx := newQueryServer(t, replies)
	_, err := ctx.CurrentEpoch()
	if err == nil || !strings.Contains(err.Error(), "genesis epoch length is missing") {
		t.Fatalf("CurrentEpoch() error = %v, want genesis epoch length is missing", err)
	}
	if n := server.callCount("ReadEraSummary"); n != 0 {
		t.Fatalf("ReadEraSummary called %d times, want 0", n)
	}
}

func TestCurrentEpochEraSummaryCached(t *testing.T) {
	t.Parallel()
	server, ctx := newQueryServer(t, epochReplies(16588800, mainnetLikeEras()...))
	for range 3 {
		if _, err := ctx.CurrentEpoch(); err != nil {
			t.Fatal(err)
		}
	}
	if n := server.callCount("ReadEraSummary"); n != 1 {
		t.Fatalf("ReadEraSummary called %d times, want 1", n)
	}
	ctx.cacheMu.Lock()
	ctx.eraCacheAt = time.Now().Add(-cacheExpiry)
	ctx.cacheMu.Unlock()
	if _, err := ctx.CurrentEpoch(); err != nil {
		t.Fatal(err)
	}
	if n := server.callCount("ReadEraSummary"); n != 2 {
		t.Fatalf("ReadEraSummary called %d times after expiry, want 2", n)
	}
}

func TestCurrentEpochFailureNotCached(t *testing.T) {
	t.Parallel()
	server, ctx := newQueryServer(t, epochReplies(5000))
	if _, err := ctx.CurrentEpoch(); err == nil {
		t.Fatal("expected error for empty summary")
	}
	server.setReply("ReadEraSummary", eraReply(&cardano.EraSummary{Name: "shelley", Start: boundary(0, 0)}))
	if got, err := ctx.CurrentEpoch(); err != nil || got != 0 {
		t.Fatalf("CurrentEpoch() = %d, %v, want 0", got, err)
	}
}

// Overlapping eras pass validateEraSummaries and are only rejected once the
// tip is located, so the rejection has to drop the cached summary as well.
func TestCurrentEpochUnusableSummaryNotCached(t *testing.T) {
	t.Parallel()
	server, ctx := newQueryServer(t, epochReplies(5000,
		&cardano.EraSummary{Name: "a", Start: boundary(0, 0)},
		&cardano.EraSummary{Name: "b", Start: boundary(1000, 1)},
	))
	if _, err := ctx.CurrentEpoch(); err == nil {
		t.Fatal("expected error for overlapping eras")
	}
	server.setReply("ReadEraSummary", eraReply(&cardano.EraSummary{Name: "shelley", Start: boundary(0, 0)}))
	if got, err := ctx.CurrentEpoch(); err != nil || got != 0 {
		t.Fatalf("CurrentEpoch() = %d, %v, want 0", got, err)
	}
}

func TestCurrentEpochHonoursCancelledContext(t *testing.T) {
	t.Parallel()
	server, ctx := newQueryServer(t, epochReplies(16588800, mainnetLikeEras()...))
	cctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := ctx.CurrentEpochContext(cctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
	if n := server.callCount("ReadTip") + server.callCount("ReadEraSummary"); n != 0 {
		t.Fatalf("cancelled context still reached the server %d times", n)
	}
}

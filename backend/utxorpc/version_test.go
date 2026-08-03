package utxorpc

import (
	"encoding/binary"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/babbage"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/mary"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	alphacardano "github.com/utxorpc/go-codegen/utxorpc/v1alpha/cardano"
	alphaquery "github.com/utxorpc/go-codegen/utxorpc/v1alpha/query"
	alphasubmit "github.com/utxorpc/go-codegen/utxorpc/v1alpha/submit"
	alphasync "github.com/utxorpc/go-codegen/utxorpc/v1alpha/sync"
	cardano "github.com/utxorpc/go-codegen/utxorpc/v1beta/cardano"
	query "github.com/utxorpc/go-codegen/utxorpc/v1beta/query"
	submit "github.com/utxorpc/go-codegen/utxorpc/v1beta/submit"
	syncpb "github.com/utxorpc/go-codegen/utxorpc/v1beta/sync"
)

// versionTestAddress builds a synthetic mainnet address for driving Utxos().
func versionTestAddress(t *testing.T) common.Address {
	t.Helper()
	payment := make([]byte, 28)
	stake := make([]byte, 28)
	for i := range payment {
		payment[i] = 0xaa
		stake[i] = 0xbb
	}
	addr, err := common.NewAddressFromParts(
		common.AddressTypeKeyKey, 1, payment, stake,
	)
	if err != nil {
		t.Fatalf("build test address: %v", err)
	}
	return addr
}

// grpcFrame wraps a message in gRPC framing: one compression byte plus a
// four-byte big-endian length.
func grpcFrame(msg []byte) []byte {
	out := make([]byte, 5, 5+len(msg))
	//nolint:gosec // test payloads are a few bytes
	binary.BigEndian.PutUint32(out[1:5], uint32(len(msg)))
	return append(out, msg...)
}

// payloadFor returns a valid response body for the RPC being called, so tests
// exercise a fully successful call rather than only the routing. ReadTip is the
// operation the routing tests drive; anything else gets an empty message.
func (r *versionRouter) payloadFor(path string) []byte {
	if !strings.Contains(path, "ReadTip") {
		return nil
	}
	var (
		encoded []byte
		err     error
	)
	if strings.Contains(path, "utxorpc.v1alpha.") {
		encoded, err = proto.Marshal(&alphasync.ReadTipResponse{
			Tip: &alphasync.BlockRef{Slot: testTipSlot},
		})
	} else {
		encoded, err = proto.Marshal(&syncpb.ReadTipResponse{
			Tip: &syncpb.BlockRef{Slot: testTipSlot},
		})
	}
	if err != nil {
		return nil
	}
	return encoded
}

// testTipSlot is the slot the recording server reports for ReadTip.
const testTipSlot = 4242

// versionRouter serves only the proto packages it is told to, answering
// everything else with gRPC UNIMPLEMENTED, which is how a server that exposes
// one version but not the other behaves.
type versionRouter struct {
	serve string // proto package fragment to accept, e.g. "utxorpc.v1alpha."

	mu    sync.Mutex
	paths []string
}

func (r *versionRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mu.Lock()
	r.paths = append(r.paths, req.URL.Path)
	r.mu.Unlock()

	// gRPC carries its status in trailers, not headers.
	w.Header().Set("Content-Type", "application/grpc")
	w.Header().Set("Trailer", "Grpc-Status, Grpc-Message")
	if !strings.Contains(req.URL.Path, r.serve) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Grpc-Status", "12") // UNIMPLEMENTED
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(grpcFrame(r.payloadFor(req.URL.Path)))
	w.Header().Set("Grpc-Status", "0")
}

func (r *versionRouter) recorded() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, len(r.paths))
	copy(out, r.paths)
	return out
}

func newVersionRouter(t *testing.T, serve string) (*versionRouter, *httptest.Server) {
	t.Helper()
	r := &versionRouter{serve: serve}
	srv := httptest.NewUnstartedServer(r)
	protocols := new(http.Protocols)
	protocols.SetUnencryptedHTTP2(true)
	srv.Config.Protocols = protocols
	srv.Start()
	t.Cleanup(srv.Close)
	return r, srv
}

// TestFallsBackToV1AlphaWhenV1BetaUnimplemented covers the Dolos-style
// deployment that serves only the older services: the v1beta attempt returns
// UNIMPLEMENTED and the call must succeed against v1alpha rather than failing.
func TestFallsBackToV1AlphaWhenV1BetaUnimplemented(t *testing.T) {
	router, srv := newVersionRouter(t, "utxorpc.v1alpha.")
	cc := NewUtxoRpcChainContext(srv.URL, 1, nil)

	slot, err := cc.TipContext(t.Context())
	if err != nil {
		t.Fatalf("expected the v1alpha fallback to answer, got: %v", err)
	}
	if slot != testTipSlot {
		t.Errorf("slot = %d, want %d from the v1alpha server", slot, testTipSlot)
	}

	paths := router.recorded()
	if len(paths) != 2 {
		t.Fatalf("expected a v1beta attempt then a v1alpha retry, got %v", paths)
	}
	if !strings.Contains(paths[0], "utxorpc.v1beta.") {
		t.Errorf("first attempt was not v1beta: %s", paths[0])
	}
	if !strings.Contains(paths[1], "utxorpc.v1alpha.") {
		t.Errorf("fallback was not v1alpha: %s", paths[1])
	}
	if got := cc.negotiatedVersion(); got != protoVersionV1Alpha {
		t.Errorf("negotiated version = %s, want v1alpha", got)
	}
}

// TestNegotiationIsStickyPerContext confirms the fallback probe happens once,
// not on every call.
func TestNegotiationIsStickyPerContext(t *testing.T) {
	router, srv := newVersionRouter(t, "utxorpc.v1alpha.")
	cc := NewUtxoRpcChainContext(srv.URL, 1, nil)

	for range 3 {
		if _, err := cc.TipContext(t.Context()); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	var betaAttempts int
	for _, p := range router.recorded() {
		if strings.Contains(p, "utxorpc.v1beta.") {
			betaAttempts++
		}
	}
	if betaAttempts != 1 {
		t.Errorf(
			"v1beta was attempted %d times across 3 calls, want 1: %v",
			betaAttempts, router.recorded(),
		)
	}
}

// TestPrefersV1BetaWhenAvailable confirms a server offering both is used on
// v1beta, and that no v1alpha request is ever made.
func TestPrefersV1BetaWhenAvailable(t *testing.T) {
	router, srv := newVersionRouter(t, "utxorpc.")
	cc := NewUtxoRpcChainContext(srv.URL, 1, nil)

	slot, err := cc.TipContext(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if slot != testTipSlot {
		t.Errorf("slot = %d, want %d", slot, testTipSlot)
	}
	for _, p := range router.recorded() {
		if strings.Contains(p, "utxorpc.v1alpha.") {
			t.Errorf("v1alpha was contacted although v1beta answered: %s", p)
		}
	}
	if got := cc.negotiatedVersion(); got != protoVersionV1Beta {
		t.Errorf("negotiated version = %s, want v1beta", got)
	}
}

// TestFallbackAppliesToEveryOperation makes sure no call site was left wired
// directly to the v1beta client.
func TestFallbackAppliesToEveryOperation(t *testing.T) {
	addr := versionTestAddress(t)
	ops := map[string]func(*UtxoRpcChainContext) error{
		"ProtocolParams": func(c *UtxoRpcChainContext) error {
			_, err := c.ProtocolParamsContext(t.Context())
			return err
		},
		"Tip": func(c *UtxoRpcChainContext) error {
			_, err := c.TipContext(t.Context())
			return err
		},
		"Utxos": func(c *UtxoRpcChainContext) error {
			_, err := c.UtxosContext(t.Context(), addr)
			return err
		},
		"SubmitTx": func(c *UtxoRpcChainContext) error {
			_, err := c.SubmitTxContext(t.Context(), []byte{0x00})
			return err
		},
	}
	for name, op := range ops {
		t.Run(name, func(t *testing.T) {
			router, srv := newVersionRouter(t, "utxorpc.v1alpha.")
			cc := NewUtxoRpcChainContext(srv.URL, 1, nil)
			// Errors from parsing an empty response body are fine; what matters
			// is that a v1alpha request was attempted at all.
			_ = op(cc)
			var sawAlpha bool
			for _, p := range router.recorded() {
				if strings.Contains(p, "utxorpc.v1alpha.") {
					sawAlpha = true
				}
			}
			if !sawAlpha {
				t.Errorf(
					"%s never fell back to v1alpha; paths: %v",
					name, router.recorded(),
				)
			}
		})
	}
}

// TestNonUnimplementedErrorsDoNotFallBack guards the fallback trigger. A
// transport or application error says nothing about which proto version a
// server speaks, so retrying it against the other version would double the
// failure latency and hide the real cause.
func TestNonUnimplementedErrorsDoNotFallBack(t *testing.T) {
	rec := &versionRouter{serve: "\x00never matches"}
	srv := httptest.NewUnstartedServer(http.HandlerFunc(
		func(w http.ResponseWriter, req *http.Request) {
			rec.mu.Lock()
			rec.paths = append(rec.paths, req.URL.Path)
			rec.mu.Unlock()
			w.Header().Set("Content-Type", "application/grpc")
			w.Header().Set("Trailer", "Grpc-Status, Grpc-Message")
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Grpc-Status", "7") // PERMISSION_DENIED
		},
	))
	protocols := new(http.Protocols)
	protocols.SetUnencryptedHTTP2(true)
	srv.Config.Protocols = protocols
	srv.Start()
	defer srv.Close()

	cc := NewUtxoRpcChainContext(srv.URL, 1, nil)
	if _, err := cc.TipContext(t.Context()); err == nil {
		t.Fatal("expected the permission error to be returned")
	}

	for _, p := range rec.recorded() {
		if strings.Contains(p, "utxorpc.v1alpha.") {
			t.Errorf("fell back to v1alpha on a non-UNIMPLEMENTED error: %s", p)
		}
	}
	if got := cc.negotiatedVersion(); got != protoVersionUnknown {
		t.Errorf(
			"negotiated version = %s after a transport-level failure, want unknown",
			got,
		)
	}
}

// TestTranscodeRejectsUnrepresentableFields proves the runtime guard fires when
// a v1alpha message carries a field the v1beta message cannot represent, rather
// than silently dropping it.
func TestTranscodeRejectsUnrepresentableFields(t *testing.T) {
	src := &alphasync.ReadTipResponse{}
	// Field 900 exists in neither schema, so it survives Marshal as an unknown
	// field and must be reported instead of discarded.
	src.ProtoReflect().SetUnknown(protoreflect.RawFields(
		[]byte{0xe0, 0x38, 0x01}, // field 900, varint, value 1
	))
	if err := transcode(src, &syncpb.ReadTipResponse{}); err == nil {
		t.Fatal("expected transcode to reject unrepresentable fields")
	} else if !strings.Contains(err.Error(), "diverged") {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestTranscodePreservesValues checks the reprojection actually carries data
// across, not merely that it does not error.
func TestTranscodePreservesValues(t *testing.T) {
	src := &alphasync.ReadTipResponse{
		Tip: &alphasync.BlockRef{Slot: 12345, Height: 7},
	}
	dst := &syncpb.ReadTipResponse{}
	if err := transcode(src, dst); err != nil {
		t.Fatal(err)
	}
	if got := dst.GetTip().GetSlot(); got != 12345 {
		t.Errorf("slot = %d, want 12345", got)
	}
	if got := dst.GetTip().GetHeight(); got != 7 {
		t.Errorf("height = %d, want 7", got)
	}
}

// TestProtoPackagesRemainWireCompatible is the guard behind the transcode
// approach: every message Apollo reprojects must assign the same field numbers
// and kinds in both proto packages. If a future codegen release diverges, this
// fails here rather than silently dropping data at runtime.
//
// Only the four responses that are actually reprojected are listed. UTxO
// responses are deliberately absent because they are not wire compatible; see
// TestUtxoValueSchemasDifferBetweenVersions records how the parsed Cardano
// value structures differ, which is why UTxO responses are converted per
// version instead of reprojected.
//
// The difference is not a wire-type change: Asset field 2 is a BigInt message
// in both. v1alpha names it output_coin and adds mint_coin at field 3 for the
// minting case, while v1beta uses one quantity field for both; v1alpha also
// carries Multiasset.redeemer at field 3, which v1beta drops. Those extra
// v1alpha fields have no v1beta counterpart, so reprojecting a message that
// used them would move them into unknown fields.
func TestUtxoValueSchemasDifferBetweenVersions(t *testing.T) {
	alphaAsset := (&alphacardano.Asset{}).ProtoReflect().Descriptor()
	betaAsset := (&cardano.Asset{}).ProtoReflect().Descriptor()

	alphaField2 := alphaAsset.Fields().ByNumber(2)
	betaField2 := betaAsset.Fields().ByNumber(2)
	if alphaField2 == nil || betaField2 == nil {
		t.Fatal("expected both Asset messages to define field 2")
	}
	// Same wire type, different name: the quantity itself does carry across.
	if alphaField2.Kind() != betaField2.Kind() {
		t.Errorf(
			"Asset field 2 kind changed: v1alpha %s, v1beta %s",
			alphaField2.Kind(), betaField2.Kind(),
		)
	}
	if alphaField2.Name() == betaField2.Name() {
		t.Errorf(
			"Asset field 2 now has the same name in both versions (%s)",
			alphaField2.Name(),
		)
	}
	t.Logf(
		"Asset field 2: v1alpha %s, v1beta %s, both %s",
		alphaField2.Name(), betaField2.Name(), alphaField2.Kind(),
	)

	// Fields present only in v1alpha, which a reprojection could not represent.
	for _, only := range []struct {
		msg   protoreflect.MessageDescriptor
		beta  protoreflect.MessageDescriptor
		field protoreflect.FieldNumber
	}{
		{alphaAsset, betaAsset, 3},
		{
			(&alphacardano.Multiasset{}).ProtoReflect().Descriptor(),
			(&cardano.Multiasset{}).ProtoReflect().Descriptor(),
			3,
		},
	} {
		af := only.msg.Fields().ByNumber(only.field)
		if af == nil {
			t.Errorf("%s: expected field %d in v1alpha", only.msg.FullName(), only.field)
			continue
		}
		if only.beta.Fields().ByNumber(only.field) != nil {
			t.Errorf(
				"%s field %d (%s) now exists in v1beta too; the schemas may have converged",
				only.msg.FullName(), only.field, af.Name(),
			)
			continue
		}
		t.Logf("%s field %d (%s) exists only in v1alpha", only.msg.FullName(), only.field, af.Name())
	}
}

// TestUtxoConversionSharedAcrossVersions confirms both versions' converters
// produce the same UTxO from the same native bytes, which is the property that
// makes converting per version safe rather than duplicative.
func TestUtxoConversionSharedAcrossVersions(t *testing.T) {
	addr := versionTestAddress(t)
	output := babbage.BabbageTransactionOutput{
		OutputAddress: addr,
		OutputAmount:  mary.MaryTransactionOutputValue{Amount: 1_500_000},
	}
	nativeBytes, err := cbor.Encode(&output)
	if err != nil {
		t.Fatal(err)
	}
	hash := make([]byte, common.Blake2b256Size)
	for i := range hash {
		hash[i] = 0x11
	}

	betaUtxo, err := utxoFromRpc(&query.AnyUtxoData{
		NativeBytes: nativeBytes,
		TxoRef:      &query.TxoRef{Hash: hash, Index: 3},
	})
	if err != nil {
		t.Fatalf("v1beta conversion: %v", err)
	}
	alphaUtxo, err := utxoFromRpcAlpha(&alphaquery.AnyUtxoData{
		NativeBytes: nativeBytes,
		TxoRef:      &alphaquery.TxoRef{Hash: hash, Index: 3},
	})
	if err != nil {
		t.Fatalf("v1alpha conversion: %v", err)
	}

	if betaUtxo.Id.Id() != alphaUtxo.Id.Id() {
		t.Errorf(
			"tx id differs: v1beta %x, v1alpha %x",
			betaUtxo.Id.Id(), alphaUtxo.Id.Id(),
		)
	}
	if betaUtxo.Id.Index() != alphaUtxo.Id.Index() {
		t.Errorf(
			"index differs: v1beta %d, v1alpha %d",
			betaUtxo.Id.Index(), alphaUtxo.Id.Index(),
		)
	}
	if betaUtxo.Output.Amount().Cmp(alphaUtxo.Output.Amount()) != 0 {
		t.Errorf(
			"amount differs: v1beta %s, v1alpha %s",
			betaUtxo.Output.Amount(), alphaUtxo.Output.Amount(),
		)
	}
	if got := betaUtxo.Output.Amount().Uint64(); got != 1_500_000 {
		t.Errorf("amount = %d, want 1500000", got)
	}
}

// TestProtoPackagesRemainWireCompatible is the guard behind the transcode
// approach: every message Apollo reprojects from v1alpha onto v1beta must
// assign the same field numbers, names and kinds in both proto packages. If a
// future codegen release diverges, this fails here rather than silently
// dropping data at runtime.
//
// Only the four responses that are actually reprojected are listed. UTxO
// responses are deliberately absent because they are converted per version
// instead; TestUtxoValueSchemasDifferBetweenVersions records why.
func TestProtoPackagesRemainWireCompatible(t *testing.T) {
	pairs := []struct {
		name  string
		alpha proto.Message
		beta  proto.Message
	}{
		{
			"ReadParamsResponse",
			&alphaquery.ReadParamsResponse{},
			&query.ReadParamsResponse{},
		},
		{
			"ReadTipResponse",
			&alphasync.ReadTipResponse{},
			&syncpb.ReadTipResponse{},
		},
		{
			"SubmitTxResponse",
			&alphasubmit.SubmitTxResponse{},
			&submit.SubmitTxResponse{},
		},
		{
			"EvalTxResponse",
			&alphasubmit.EvalTxResponse{},
			&submit.EvalTxResponse{},
		},
	}
	for _, p := range pairs {
		t.Run(p.name, func(t *testing.T) {
			assertFieldsCompatible(
				t,
				p.alpha.ProtoReflect().Descriptor(),
				p.beta.ProtoReflect().Descriptor(),
				map[string]struct{}{},
			)
		})
	}
}

// assertFieldsCompatible checks that every field of the v1alpha message exists
// in the v1beta message with the same number, name and kind. v1beta may add
// fields; it may not renumber, rename or retype the ones v1alpha already has.
//
// visited breaks cycles: protobuf messages may be recursive or mutually
// recursive, so descending unconditionally does not terminate.
func assertFieldsCompatible(
	t *testing.T,
	alpha, beta protoreflect.MessageDescriptor,
	visited map[string]struct{},
) {
	t.Helper()
	key := string(alpha.FullName()) + "|" + string(beta.FullName())
	if _, seen := visited[key]; seen {
		return
	}
	visited[key] = struct{}{}

	alphaFields := alpha.Fields()
	betaFields := beta.Fields()
	for i := range alphaFields.Len() {
		af := alphaFields.Get(i)
		bf := betaFields.ByNumber(af.Number())
		if bf == nil {
			t.Errorf(
				"%s: field %s (number %d) is absent from the v1beta message",
				alpha.FullName(), af.Name(), af.Number(),
			)
			continue
		}
		if af.Name() != bf.Name() {
			t.Errorf(
				"%s: field number %d is %s in v1alpha but %s in v1beta",
				alpha.FullName(), af.Number(), af.Name(), bf.Name(),
			)
		}
		if af.Kind() != bf.Kind() {
			t.Errorf(
				"%s: field %s changed kind from %s to %s",
				alpha.FullName(), af.Name(), af.Kind(), bf.Kind(),
			)
		}
		if af.Cardinality() != bf.Cardinality() {
			t.Errorf(
				"%s: field %s changed cardinality from %s to %s",
				alpha.FullName(), af.Name(), af.Cardinality(), bf.Cardinality(),
			)
		}
		if af.Kind() == protoreflect.MessageKind && !af.IsMap() {
			assertFieldsCompatible(t, af.Message(), bf.Message(), visited)
		}
	}
}

package ouroboros

import (
	"context"
	"encoding/hex"
	"errors"
	"io"
	"net"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	gouroboros "github.com/blinklabs-io/gouroboros"
	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/babbage"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/conway"
	"github.com/blinklabs-io/gouroboros/ledger/mary"
	"github.com/blinklabs-io/gouroboros/muxer"
	"github.com/blinklabs-io/gouroboros/protocol"
	pcommon "github.com/blinklabs-io/gouroboros/protocol/common"
	"github.com/blinklabs-io/gouroboros/protocol/localstatequery"
	"github.com/blinklabs-io/gouroboros/protocol/localtxsubmission"
	ouroboros_mock "github.com/blinklabs-io/ouroboros-mock"
)

// mockNode hands out one scripted node-to-client conversation per dial, so a
// test can check both what the context sends and when it reconnects.
type mockNode struct {
	t             *testing.T
	conversations [][]ouroboros_mock.ConversationEntry

	mu    sync.Mutex
	dials int
	errs  []error
	done  []chan struct{}
}

func newMockNode(t *testing.T, conversations ...[]ouroboros_mock.ConversationEntry) (*mockNode, *OuroborosChainContext) {
	t.Helper()
	return newMockNodeWithConfig(t, Config{}, conversations...)
}

// newMockNodeWithConfig is newMockNode with timeouts and other settings from
// cfg. The network magic is always the mock's.
func newMockNodeWithConfig(
	t *testing.T,
	cfg Config,
	conversations ...[]ouroboros_mock.ConversationEntry,
) (*mockNode, *OuroborosChainContext) {
	t.Helper()
	m := &mockNode{t: t, conversations: conversations}
	cfg.NetworkMagic = ouroboros_mock.MockNetworkMagic
	ctx := newContext(m.dial, cfg)
	t.Cleanup(func() {
		// Let each conversation finish before closing: messages that need
		// no reply, such as Release, may still be in flight.
		m.mu.Lock()
		done := m.done
		m.mu.Unlock()
		for _, d := range done {
			select {
			case <-d:
			case <-time.After(2 * time.Second):
				t.Error("mock node: conversation did not finish")
			}
		}
		_ = ctx.Close()
		m.mu.Lock()
		defer m.mu.Unlock()
		for _, err := range m.errs {
			t.Errorf("mock node: %v", err)
		}
	})
	return m, ctx
}

func (m *mockNode) dial(context.Context) (net.Conn, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.dials >= len(m.conversations) {
		m.dials++
		return nil, errors.New("mock node: no conversation left to dial")
	}
	conn := ouroboros_mock.NewConnection(ouroboros_mock.ProtocolRoleClient, m.conversations[m.dials])
	m.dials++
	done := make(chan struct{})
	m.done = append(m.done, done)
	go func() {
		defer close(done)
		for err := range conn.(*ouroboros_mock.Connection).ErrorChan() {
			if err != nil {
				m.mu.Lock()
				m.errs = append(m.errs, err)
				m.mu.Unlock()
			}
		}
	}()
	return conn, nil
}

func (m *mockNode) dialCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.dials
}

var handshake = []ouroboros_mock.ConversationEntry{
	ouroboros_mock.ConversationEntryHandshakeRequestGeneric,
	ouroboros_mock.ConversationEntryHandshakeNtCResponse,
}

// acquire expects a fresh AcquireVolatileTip. A context that failed to
// release its previous acquisition would send a ReAcquire instead, which the
// mock rejects.
var acquire = []ouroboros_mock.ConversationEntry{
	ouroboros_mock.ConversationEntryInput{
		ProtocolId:  localstatequery.ProtocolId,
		MessageType: localstatequery.MessageTypeAcquireVolatileTip,
	},
	ouroboros_mock.ConversationEntryOutput{
		ProtocolId: localstatequery.ProtocolId,
		IsResponse: true,
		Messages:   []protocol.Message{localstatequery.NewMsgAcquired()},
	},
}

var release = []ouroboros_mock.ConversationEntry{
	ouroboros_mock.ConversationEntryInput{
		ProtocolId:  localstatequery.ProtocolId,
		MessageType: localstatequery.MessageTypeRelease,
	},
}

func queryResult(t *testing.T, result any) []ouroboros_mock.ConversationEntry {
	t.Helper()
	encoded, ok := result.([]byte)
	if !ok {
		var err error
		encoded, err = cbor.Encode(result)
		if err != nil {
			t.Fatal(err)
		}
	}
	message, err := cbor.Encode(localstatequery.NewMsgResult(encoded))
	if err != nil {
		t.Fatal(err)
	}
	entries := []ouroboros_mock.ConversationEntry{
		ouroboros_mock.ConversationEntryInput{
			ProtocolId:  localstatequery.ProtocolId,
			MessageType: localstatequery.MessageTypeQuery,
		},
	}
	// The mock emits one muxer segment per output entry. Large results
	// must be fragmented just as they are when a real node sends them.
	for len(message) > 0 {
		size := min(len(message), muxer.SegmentMaxPayloadLength)
		fragment := &protocol.MessageBase{}
		fragment.SetCbor(message[:size])
		entries = append(entries, ouroboros_mock.ConversationEntryOutput{
			ProtocolId: localstatequery.ProtocolId,
			IsResponse: true,
			Messages:   []protocol.Message{fragment},
		})
		message = message[size:]
	}
	return entries
}

func conversation(parts ...[]ouroboros_mock.ConversationEntry) []ouroboros_mock.ConversationEntry {
	var out []ouroboros_mock.ConversationEntry
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

func tipExchange(t *testing.T, slot uint64) []ouroboros_mock.ConversationEntry {
	hash := make([]byte, 32)
	hash[0] = byte(slot & 0xff)
	return conversation(acquire, queryResult(t, pcommon.NewPoint(slot, hash)), release)
}

// conwayEra is the hard-fork combinator's index for Conway, which the node
// reports for its current era.
var conwayEra = []byte{0x06}

func TestTipReusesConnectionAndReleasesState(t *testing.T) {
	node, ctx := newMockNode(t, conversation(handshake, tipExchange(t, 123), tipExchange(t, 124)))

	for _, want := range []uint64{123, 124} {
		got, err := ctx.TipContext(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("Tip() = %d, want %d", got, want)
		}
	}
	if n := node.dialCount(); n != 1 {
		t.Fatalf("dialed %d times, want 1", n)
	}
}

func TestTipAtOriginIsAnError(t *testing.T) {
	_, ctx := newMockNode(t, conversation(handshake, acquire, queryResult(t, pcommon.NewPointOrigin()), release))
	if _, err := ctx.Tip(); err == nil {
		t.Fatal("expected an error for a chain at origin")
	}
}

func TestCurrentEpoch(t *testing.T) {
	_, ctx := newMockNode(t, conversation(
		handshake, acquire,
		queryResult(t, conwayEra),
		queryResult(t, []uint64{512}),
		release,
	))
	got, err := ctx.CurrentEpoch()
	if err != nil {
		t.Fatal(err)
	}
	if got != 512 {
		t.Fatalf("CurrentEpoch() = %d, want 512", got)
	}
}

// TestReconnectsAfterNodeClosesConnection covers a node restart: the next
// operation after the node drops the connection dials a new one.
func TestReconnectsAfterNodeClosesConnection(t *testing.T) {
	node, ctx := newMockNode(t,
		conversation(handshake, tipExchange(t, 10), []ouroboros_mock.ConversationEntry{ouroboros_mock.ConversationEntryClose{}}),
		conversation(handshake, tipExchange(t, 11)),
	)

	if _, err := ctx.Tip(); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool {
		ctx.session.gate <- struct{}{}
		defer func() { <-ctx.session.gate }()
		select {
		case <-ctx.session.health.done:
			return true
		default:
			return false
		}
	})
	got, err := ctx.Tip()
	if err != nil {
		t.Fatal(err)
	}
	if got != 11 {
		t.Fatalf("Tip() after reconnect = %d, want 11", got)
	}
	if n := node.dialCount(); n != 2 {
		t.Fatalf("dialed %d times, want 2", n)
	}
}

// TestCancelUnblocksAndReconnects covers a node that stops answering: the
// caller's context ends the operation, and the abandoned connection is not
// reused.
func TestCancelUnblocksAndReconnects(t *testing.T) {
	stalled := conversation(handshake, []ouroboros_mock.ConversationEntry{
		ouroboros_mock.ConversationEntryInput{
			ProtocolId:  localstatequery.ProtocolId,
			MessageType: localstatequery.MessageTypeAcquireVolatileTip,
		},
	})
	node, ctx := newMockNode(t, stalled, conversation(handshake, tipExchange(t, 7)))

	callCtx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := ctx.TipContext(callCtx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("got %v, want context.DeadlineExceeded", err)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("cancelled call took %v to return", elapsed)
	}

	got, err := ctx.Tip()
	if err != nil {
		t.Fatal(err)
	}
	if got != 7 {
		t.Fatalf("Tip() = %d, want 7", got)
	}
	if n := node.dialCount(); n != 2 {
		t.Fatalf("dialed %d times, want 2", n)
	}
}

func TestDialFailureIsReported(t *testing.T) {
	_, ctx := newMockNode(t)
	if _, err := ctx.Tip(); err == nil {
		t.Fatal("expected a dial error")
	}
}

func testTx() []byte {
	// [body {}, witnesses {}, valid true, auxiliary data null]
	tx, _ := hex.DecodeString("84a0a0f5f6")
	return tx
}

func submitExchange(reply protocol.Message) []ouroboros_mock.ConversationEntry {
	return []ouroboros_mock.ConversationEntry{
		ouroboros_mock.ConversationEntryInput{
			ProtocolId:  localtxsubmission.ProtocolId,
			MessageType: localtxsubmission.MessageTypeSubmitTx,
		},
		ouroboros_mock.ConversationEntryOutput{
			ProtocolId: localtxsubmission.ProtocolId,
			IsResponse: true,
			Messages:   []protocol.Message{reply},
		},
	}
}

func TestSubmitTx(t *testing.T) {
	_, ctx := newMockNode(t, conversation(
		handshake, acquire, queryResult(t, conwayEra),
		submitExchange(localtxsubmission.NewMsgAcceptTx()), release,
	))

	got, err := ctx.SubmitTx(testTx())
	if err != nil {
		t.Fatal(err)
	}
	want := common.Blake2b256Hash([]byte{0xa0})
	if got != want {
		t.Fatalf("SubmitTx() = %s, want the body hash %s", got, want)
	}
}

// TestSubmitTxRejectionKeepsConnection checks that a rejection is reported
// as one, and that the connection survives it: the node decided, nothing
// went wrong with the transport.
func TestSubmitTxRejectionKeepsConnection(t *testing.T) {
	reason, _ := hex.DecodeString("820082018163666f6f")
	node, ctx := newMockNode(t, conversation(
		handshake, acquire, queryResult(t, conwayEra),
		submitExchange(localtxsubmission.NewMsgRejectTx(reason)), release,
		tipExchange(t, 5),
	))

	_, err := ctx.SubmitTx(testTx())
	var rejected localtxsubmission.TransactionRejectedError
	if !errors.As(err, &rejected) {
		t.Fatalf("got %v, want a TransactionRejectedError", err)
	}
	if _, err := ctx.Tip(); err != nil {
		t.Fatal(err)
	}
	if n := node.dialCount(); n != 1 {
		t.Fatalf("dialed %d times, want 1", n)
	}
}

func TestSubmitTxRejectsMalformedTransaction(t *testing.T) {
	node, ctx := newMockNode(t)
	for _, tx := range [][]byte{nil, {0xa0}, {0x81, 0xa0}} {
		if _, err := ctx.SubmitTx(tx); err == nil {
			t.Fatalf("SubmitTx(%x) succeeded, want an error", tx)
		}
	}
	if n := node.dialCount(); n != 0 {
		t.Fatalf("dialed %d times for malformed transactions, want 0", n)
	}
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("condition not met before deadline")
		}
		time.Sleep(time.Millisecond)
	}
}

func TestProtocolParamsOverTheWire(t *testing.T) {
	_, ctx := newMockNode(t, conversation(
		handshake, acquire,
		queryResult(t, conwayEra),
		queryResult(t, []conway.ConwayProtocolParameters{wireConwayParams()}),
		release,
	))
	pp, err := ctx.ProtocolParams()
	if err != nil {
		t.Fatal(err)
	}
	if pp.MinFeeConstant != 155381 || pp.CoinsPerUtxoByte != "4310" || pp.PriceMem != 0.0577 {
		t.Fatalf("unexpected protocol parameters: %+v", pp)
	}
	if len(pp.CostModels["PlutusV3"]) != 3 {
		t.Fatalf("CostModels = %v, want a PlutusV3 model", pp.CostModels)
	}
}

// wireConwayParams fills the governance thresholds testConwayParams leaves
// unset, since a nil rational cannot be encoded.
func wireConwayParams() conway.ConwayProtocolParameters {
	p := testConwayParams()
	fillRats(reflect.ValueOf(&p).Elem())
	return p
}

func fillRats(v reflect.Value) {
	for i := range v.NumField() {
		f := v.Field(i)
		switch {
		case !f.CanSet():
		case f.Type() == reflect.TypeFor[*cbor.Rat]():
			if f.IsNil() {
				f.Set(reflect.ValueOf(rat(1, 2)))
			}
		case f.Type() == reflect.TypeFor[cbor.Rat]():
			if f.Field(0).IsNil() {
				f.Set(reflect.ValueOf(*rat(1, 2)))
			}
		case f.Kind() == reflect.Struct:
			fillRats(f)
		}
	}
}

// utxoResult encodes a UTxO query result: a one-element array holding a
// map from [tx id, index] to output.
type utxoEntry struct {
	txId  []byte
	index uint64
	out   *babbage.BabbageTransactionOutput
}

func utxoResult(t *testing.T, entries ...utxoEntry) []byte {
	t.Helper()
	if len(entries) > 23 {
		t.Fatal("utxoResult encodes at most 23 entries")
	}
	result := []byte{0x81, 0xa0 + byte(len(entries))} //nolint:gosec // bounded above
	for _, e := range entries {
		k, err := cbor.Encode([]any{e.txId, e.index})
		if err != nil {
			t.Fatal(err)
		}
		v, err := cbor.Encode(e.out)
		if err != nil {
			t.Fatal(err)
		}
		result = append(append(result, k...), v...)
	}
	return result
}

func testOutput(t *testing.T, lovelace uint64) *babbage.BabbageTransactionOutput {
	t.Helper()
	addr, err := common.NewAddress(testAddress)
	if err != nil {
		t.Fatal(err)
	}
	return &babbage.BabbageTransactionOutput{
		OutputAddress: addr,
		OutputAmount:  mary.MaryTransactionOutputValue{Amount: lovelace},
	}
}

func TestUtxosOverTheWire(t *testing.T) {
	txId := make([]byte, 32)
	txId[31] = 9
	_, ctx := newMockNode(t, conversation(
		handshake, acquire,
		queryResult(t, conwayEra),
		queryResult(t, utxoResult(t, utxoEntry{txId, 1, testOutput(t, 5_000_000)})),
		release,
	))
	addr, err := common.NewAddress(testAddress)
	if err != nil {
		t.Fatal(err)
	}
	utxos, err := ctx.Utxos(addr)
	if err != nil {
		t.Fatal(err)
	}
	if len(utxos) != 1 {
		t.Fatalf("got %d utxos, want 1", len(utxos))
	}
	u := utxos[0]
	if u.Id.Index() != 1 || u.Id.Id().String() != hex.EncodeToString(txId) {
		t.Fatalf("utxo id = %s, want %x#1", u.Id, txId)
	}
	if u.Output.Amount().Uint64() != 5_000_000 || u.Output.Address().String() != testAddress {
		t.Fatalf("unexpected output: %s %s", u.Output.Address(), u.Output.Amount())
	}
}

func TestUtxoByRefOverTheWire(t *testing.T) {
	var txId common.Blake2b256
	txId[0] = 7
	_, ctx := newMockNode(t, conversation(
		handshake,
		acquire, queryResult(t, conwayEra),
		queryResult(t, utxoResult(t, utxoEntry{txId.Bytes(), 2, testOutput(t, 3_000_000)})),
		release,
		acquire, queryResult(t, conwayEra), queryResult(t, []byte{0x81, 0xa0}), release,
	))
	u, err := ctx.UtxoByRef(txId, 2)
	if err != nil {
		t.Fatal(err)
	}
	if u.Id.Id() != txId || u.Id.Index() != 2 || u.Output.Amount().Uint64() != 3_000_000 {
		t.Fatalf("unexpected utxo %s with %s", u.Id, u.Output.Amount())
	}
	if _, err := ctx.UtxoByRef(txId, 3); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("got %v, want a not-found error", err)
	}
}

func TestUtxosAddressResultLimit(t *testing.T) {
	const limit = 100_000
	for _, count := range []int{limit, limit + 1} {
		t.Run(strconv.Itoa(count), func(t *testing.T) {
			results := make(map[localstatequery.UtxoId]babbage.BabbageTransactionOutput, count)
			output := *testOutput(t, 2_000_000)
			for i := range count {
				results[localstatequery.UtxoId{Idx: i}] = output
			}
			node, ctx := newMockNode(t, conversation(
				handshake, acquire, queryResult(t, conwayEra),
				queryResult(t, localstatequery.UTxOsResult{Results: results}), release,
				tipExchange(t, 5),
			))
			addr, err := common.NewAddress(testAddress)
			if err != nil {
				t.Fatal(err)
			}
			got, err := ctx.UtxosContext(t.Context(), addr)
			if count > limit {
				if err == nil || !strings.Contains(err.Error(), "UTxO count") {
					t.Errorf("UtxosContext() error = %v, want UTxO count limit", err)
				}
				if len(got) != 0 {
					t.Errorf("returned %d UTxOs after rejecting the query", len(got))
				}
			} else if err != nil || len(got) != count {
				t.Fatalf("UtxosContext() returned %d UTxOs, %v; want %d, nil", len(got), err, count)
			}
			if _, err := ctx.Tip(); err != nil {
				t.Fatal(err)
			}
			if node.dialCount() != 1 {
				t.Fatal("completed address query discarded its connection")
			}
		})
	}
}

// TestHandshakeTimeoutUnblocksSilentPeer covers a peer that accepts the
// socket but never answers the version proposal. The node-to-client
// handshake has no timeout of its own, so without one a call with no
// deadline, and Close behind it, would wait forever.
func TestHandshakeTimeoutUnblocksSilentPeer(t *testing.T) {
	var peers []net.Conn
	var mu sync.Mutex
	silent := func(context.Context) (net.Conn, error) {
		client, server := net.Pipe()
		go func() { _, _ = io.Copy(io.Discard, server) }()
		mu.Lock()
		peers = append(peers, server)
		mu.Unlock()
		return client, nil
	}
	ctx := newContext(silent, Config{
		NetworkMagic:     ouroboros_mock.MockNetworkMagic,
		HandshakeTimeout: 50 * time.Millisecond,
	})
	t.Cleanup(func() {
		mu.Lock()
		defer mu.Unlock()
		for _, p := range peers {
			_ = p.Close()
		}
	})

	done := make(chan error, 1)
	go func() {
		_, err := ctx.Tip()
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), "handshake: no answer within 50ms") {
			t.Fatalf("got %v, want a handshake timeout", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Tip() against a silent peer did not return")
	}

	closed := make(chan struct{})
	go func() {
		_ = ctx.Close()
		close(closed)
	}()
	select {
	case <-closed:
	case <-time.After(5 * time.Second):
		t.Fatal("Close() did not return")
	}
}

// TestHandshakeTimeoutStopsAfterHandshake checks that the handshake timeout
// does not also cut off the connection once the handshake has finished.
func TestHandshakeTimeoutStopsAfterHandshake(t *testing.T) {
	node, ctx := newMockNodeWithConfig(t,
		Config{HandshakeTimeout: 20 * time.Millisecond},
		conversation(handshake, tipExchange(t, 5), tipExchange(t, 6)),
	)
	if _, err := ctx.Tip(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(100 * time.Millisecond)
	got, err := ctx.Tip()
	if err != nil {
		t.Fatal(err)
	}
	if got != 6 {
		t.Fatalf("Tip() = %d, want 6", got)
	}
	if n := node.dialCount(); n != 1 {
		t.Fatalf("dialed %d times, want 1", n)
	}
}

// TestQueuedCallGivesUpOnCancellation covers a call waiting behind a slow
// operation: it returns when its own context ends, and it leaves the shared
// connection alone.
func TestQueuedCallGivesUpOnCancellation(t *testing.T) {
	node, ctx := newMockNode(t, conversation(handshake, tipExchange(t, 5), tipExchange(t, 6)))
	if _, err := ctx.Tip(); err != nil {
		t.Fatal(err)
	}

	// Hold the turn as a slow address query would.
	ctx.session.gate <- struct{}{}
	callCtx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := ctx.TipContext(callCtx)
	elapsed := time.Since(start)
	<-ctx.session.gate
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("got %v, want context.DeadlineExceeded", err)
	}
	if elapsed > 5*time.Second {
		t.Fatalf("queued call took %v to give up", elapsed)
	}

	got, err := ctx.Tip()
	if err != nil {
		t.Fatal(err)
	}
	if got != 6 {
		t.Fatalf("Tip() = %d, want 6", got)
	}
	if n := node.dialCount(); n != 1 {
		t.Fatalf("dialed %d times, want 1: the queued call closed the connection", n)
	}
}

// TestConnectionFailureCauseIsReported checks that a call failing because
// its connection broke reports why the connection broke, not only that its
// mini-protocol shut down.
func TestConnectionFailureCauseIsReported(t *testing.T) {
	stalledAcquire := ouroboros_mock.ConversationEntryInput{
		ProtocolId:  localstatequery.ProtocolId,
		MessageType: localstatequery.MessageTypeAcquireVolatileTip,
	}

	t.Run("node closes the socket", func(t *testing.T) {
		_, ctx := newMockNode(t, conversation(handshake, []ouroboros_mock.ConversationEntry{
			stalledAcquire,
			ouroboros_mock.ConversationEntryClose{},
		}))
		_, err := ctx.Tip()
		if !errors.Is(err, protocol.ErrProtocolShuttingDown) {
			t.Fatalf("got %v, want an error wrapping ErrProtocolShuttingDown", err)
		}
		// gouroboros reports the closed socket unless it has already
		// stopped the waiting protocol, which then looks idle; either way
		// the error must say that the node closed the connection.
		var closedErr *muxer.ConnectionClosedError
		if !errors.As(err, &closedErr) && !errors.Is(err, errNodeClosedConnection) {
			t.Fatalf("got %v, want the node closing the connection as the cause", err)
		}
	})

	t.Run("query timeout", func(t *testing.T) {
		_, ctx := newMockNodeWithConfig(t,
			Config{QueryTimeout: 50 * time.Millisecond},
			conversation(handshake, acquire, []ouroboros_mock.ConversationEntry{
				ouroboros_mock.ConversationEntryInput{
					ProtocolId:  localstatequery.ProtocolId,
					MessageType: localstatequery.MessageTypeQuery,
				},
			}),
		)
		_, err := ctx.Tip()
		if !errors.Is(err, protocol.ErrProtocolShuttingDown) || !strings.Contains(err.Error(), "connection failed: protocol error: ") ||
			!strings.Contains(err.Error(), "timeout") {
			t.Fatalf("got %v, want the query timeout behind the shutdown", err)
		}
	})
}

func TestSubmitOutcome(t *testing.T) {
	submitErr := errors.New("submit failed")
	releaseErr := errors.New("release failed")
	if err := submitOutcome(nil, nil); err != nil {
		t.Fatalf("accepted and released: got %v", err)
	}
	if err := submitOutcome(submitErr, releaseErr); !errors.Is(err, submitErr) || errors.Is(err, errReleaseAfterAccept) {
		t.Fatalf("failed submission: got %v, want the submission error", err)
	}
	err := submitOutcome(nil, releaseErr)
	if !errors.Is(err, errReleaseAfterAccept) || !errors.Is(err, releaseErr) {
		t.Fatalf("accepted, release failed: got %v, want errReleaseAfterAccept wrapping the release error", err)
	}
}

// TestReleaseAfterAcceptDiscardsConnection checks how run treats a
// submission the node accepted on a connection whose release then failed:
// the error stays recognisable, so SubmitTxContext can report the
// transaction as submitted, and the connection is not reused.
func TestReleaseAfterAcceptDiscardsConnection(t *testing.T) {
	node, ctx := newMockNode(t,
		conversation(handshake, tipExchange(t, 5)),
		conversation(handshake, tipExchange(t, 6)),
	)
	if _, err := ctx.Tip(); err != nil {
		t.Fatal(err)
	}
	// Let the node read the Release that Tip sent without waiting before
	// the connection is dropped under it.
	node.mu.Lock()
	first := node.done[0]
	node.mu.Unlock()
	<-first
	err := ctx.session.run(t.Context(), func(*gouroboros.Connection) error {
		return submitOutcome(nil, errors.New("release failed"))
	})
	if !errors.Is(err, errReleaseAfterAccept) {
		t.Fatalf("got %v, want errReleaseAfterAccept", err)
	}
	got, err := ctx.Tip()
	if err != nil {
		t.Fatal(err)
	}
	if got != 6 {
		t.Fatalf("Tip() = %d, want 6 from a new connection", got)
	}
	if n := node.dialCount(); n != 2 {
		t.Fatalf("dialed %d times, want 2", n)
	}
}

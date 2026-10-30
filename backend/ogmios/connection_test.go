package ogmios

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// epochServer is an Ogmios stand-in that answers every request with epoch 7.
// It counts the WebSocket connections it accepts and how many are open.
type epochServer struct {
	*httptest.Server
	accepted atomic.Int64
	open     atomic.Int64
}

func newEpochServer(t *testing.T) *epochServer {
	t.Helper()
	s := &epochServer{}
	upgrader := websocket.Upgrader{}
	s.Server = httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				return
			}
			s.accepted.Add(1)
			s.open.Add(1)
			defer s.open.Add(-1)
			defer func() { _ = conn.Close() }()
			for {
				if _, _, err := conn.ReadMessage(); err != nil {
					return
				}
				reply := []byte(`{"jsonrpc":"2.0","result":7}`)
				if err := conn.WriteMessage(
					websocket.TextMessage, reply,
				); err != nil {
					return
				}
			}
		},
	))
	t.Cleanup(s.Close)
	return s
}

func (s *epochServer) queryEpochs(t *testing.T, ctx *OgmiosChainContext, n int) {
	t.Helper()
	for range n {
		epoch, err := ctx.CurrentEpochContext(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if epoch != 7 {
			t.Fatalf("CurrentEpoch() = %d, want 7", epoch)
		}
	}
}

// TestOgmiosReusesConnections covers the field report behind connection
// reuse: every query dialing its own WebSocket made a busy Apollo client
// open dozens of connections per transaction.
func TestOgmiosReusesConnections(t *testing.T) {
	server := newEpochServer(t)
	ctx := testChainContext(t, Config{OgmiosEndpoint: server.URL})
	t.Cleanup(ctx.CloseIdleConnections)

	server.queryEpochs(t, ctx, 5)
	if got := server.accepted.Load(); got != 1 {
		t.Fatalf("accepted %d connections for 5 queries, want 1", got)
	}

	ctx.CloseIdleConnections()
	waitForCondition(t, func() bool { return server.open.Load() == 0 })

	// The context stays usable once its idle connections are closed.
	server.queryEpochs(t, ctx, 1)
	if got := server.accepted.Load(); got != 2 {
		t.Fatalf("accepted %d connections, want 2", got)
	}
}

func TestOgmiosConnectionReuseDisabled(t *testing.T) {
	server := newEpochServer(t)
	ctx := testChainContext(t, Config{
		OgmiosEndpoint:     server.URL,
		OgmiosMaxIdleConns: -1,
	})

	server.queryEpochs(t, ctx, 3)
	if got := server.accepted.Load(); got != 3 {
		t.Fatalf("accepted %d connections for 3 queries, want 3", got)
	}
	waitForCondition(t, func() bool { return server.open.Load() == 0 })
}

func TestOgmiosIdleTimeoutClosesConnections(t *testing.T) {
	server := newEpochServer(t)
	ctx := testChainContext(t, Config{
		OgmiosEndpoint:    server.URL,
		OgmiosIdleTimeout: 10 * time.Millisecond,
	})
	t.Cleanup(ctx.CloseIdleConnections)

	server.queryEpochs(t, ctx, 1)
	waitForCondition(t, func() bool { return server.open.Load() == 0 })
}

func TestNewOgmiosChainContextRejectsNegativeIdleTimeout(t *testing.T) {
	ctx, err := NewOgmiosChainContext(Config{
		OgmiosEndpoint:    testOgmiosEndpoint,
		OgmiosIdleTimeout: -time.Second,
	})
	if err == nil || ctx != nil {
		t.Fatalf("got (%v, %v), want a rejected configuration", ctx, err)
	}
	if !strings.Contains(err.Error(), "idle timeout") {
		t.Fatalf("error = %q, want mention of the idle timeout", err)
	}
}

// TestCloseIdleConnectionsCustomClient covers contexts built from injected
// clients, which may or may not hold connections of their own.
func TestCloseIdleConnectionsCustomClient(t *testing.T) {
	plain, err := NewOgmiosChainContextFromClients(&stubOgmiosClient{}, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	plain.CloseIdleConnections()

	closer := &closingOgmiosClient{}
	ctx, err := NewOgmiosChainContextFromClients(closer, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	ctx.CloseIdleConnections()
	if closer.closed != 1 {
		t.Fatalf("CloseIdleConnections reached the client %d times, want 1",
			closer.closed)
	}
}

type closingOgmiosClient struct {
	stubOgmiosClient
	closed int
}

func (c *closingOgmiosClient) CloseIdleConnections() { c.closed++ }

func waitForCondition(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("condition not met before deadline")
		}
		time.Sleep(time.Millisecond)
	}
}

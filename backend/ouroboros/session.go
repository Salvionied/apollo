package ouroboros

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	gouroboros "github.com/blinklabs-io/gouroboros"
	"github.com/blinklabs-io/gouroboros/protocol/localstatequery"
	"github.com/blinklabs-io/gouroboros/protocol/localtxsubmission"
)

// dialFunc opens the transport to the node. Tests replace it to talk to a
// scripted peer instead of a socket.
type dialFunc func(ctx context.Context) (net.Conn, error)

// session owns at most one node-to-client connection and runs one operation
// on it at a time. A connection is opened by the first operation that needs
// it and is discarded after any failure, so a node restart costs only the
// operations in flight at the time.
type session struct {
	dial          dialFunc
	networkMagic  uint32
	queryTimeout  time.Duration
	submitTimeout time.Duration

	// mu serializes operations. LocalStateQuery keeps the acquired ledger
	// state on the connection, so an acquire and the queries that read it
	// must not interleave with another operation's. Every field below is
	// read and written only by the goroutine holding mu.
	mu     sync.Mutex
	raw    net.Conn
	conn   *gouroboros.Connection
	broken chan struct{} // closed once conn fails or shuts down
}

// errTransactionRejected marks submission failures the node decided on, as
// opposed to transport failures. The connection stays usable after them.
var errTransactionRejected = errors.New("node rejected the transaction")

// run executes op on a live connection, dialing and handshaking first if
// needed. If ctx ends before op returns, the transport is closed to unblock
// op and ctx.Err is returned. Any failure other than a transaction rejection
// discards the connection, since the mini-protocol state it leaves behind is
// unknown; the next operation reconnects.
func (s *session) run(ctx context.Context, op func(*gouroboros.Connection) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.broken != nil {
		select {
		case <-s.broken:
			s.closeLocked()
		default:
		}
	}
	if s.raw == nil {
		raw, err := s.dial(ctx)
		if err != nil {
			return fmt.Errorf("connect to node: %w", err)
		}
		s.raw = raw
	}

	type outcome struct {
		conn   *gouroboros.Connection
		broken chan struct{}
		err    error
	}
	raw, conn := s.raw, s.conn
	done := make(chan outcome, 1)
	go func() {
		o := outcome{conn: conn}
		if o.conn == nil {
			o.conn, o.broken, o.err = s.handshake(raw)
			if o.err != nil {
				done <- o
				return
			}
		}
		o.err = op(o.conn)
		done <- o
	}()

	var o outcome
	select {
	case o = <-done:
	case <-ctx.Done():
		// Closing the transport makes the node calls op is blocked in fail.
		_ = raw.Close()
		o = <-done
		o.err = ctx.Err()
	}
	if o.broken != nil {
		s.conn, s.broken = o.conn, o.broken
	}
	if o.err != nil && (ctx.Err() != nil || !errors.Is(o.err, errTransactionRejected)) {
		s.closeLocked()
	}
	return o.err
}

// handshake opens a node-to-client connection over raw. It touches no
// session state, so it can run while the caller waits on its context.
func (s *session) handshake(raw net.Conn) (*gouroboros.Connection, chan struct{}, error) {
	conn, err := gouroboros.NewConnection(
		gouroboros.WithConnection(raw),
		gouroboros.WithNetworkMagic(s.networkMagic),
		gouroboros.WithNodeToNode(false),
		// Local queries can legitimately take minutes (a UTxO-by-address
		// scan on mainnet), so the muxer must not time out between segments.
		// Callers bound operations with their context instead.
		gouroboros.WithMuxerSegmentReadTimeout(0),
		gouroboros.WithLocalStateQueryConfig(localstatequery.NewConfig(
			localstatequery.WithQueryTimeout(s.queryTimeout),
		)),
		gouroboros.WithLocalTxSubmissionConfig(localtxsubmission.NewConfig(
			localtxsubmission.WithTimeout(s.submitTimeout),
		)),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("node-to-client handshake: %w", err)
	}
	broken := make(chan struct{})
	go func() {
		// The connection owns its error channel and closes it on shutdown.
		// A node that closes the socket while every protocol is idle shuts
		// the connection down without reporting an error, so the close
		// marks the connection broken as well as the first error does.
		defer close(broken)
		for range conn.ErrorChan() {
			return
		}
	}()
	return conn, broken, nil
}

// closeLocked tears down the current connection, if any.
func (s *session) closeLocked() {
	if s.raw != nil {
		_ = s.raw.Close()
	}
	if s.conn != nil {
		_ = s.conn.Close()
	}
	s.raw, s.conn, s.broken = nil, nil, nil
}

// close releases the connection. The session stays usable and reconnects on
// the next operation.
func (s *session) close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closeLocked()
}

// query acquires the node's current ledger state, runs fn against it and
// releases it, so every operation reads the tip as of its own start rather
// than a snapshot left over from an earlier one.
func (s *session) query(ctx context.Context, fn func(*localstatequery.Client) error) error {
	return s.run(ctx, func(conn *gouroboros.Connection) error {
		client := conn.LocalStateQuery().Client
		if err := client.AcquireVolatileTip(); err != nil {
			return fmt.Errorf("acquire ledger state: %w", err)
		}
		err := fn(client)
		if releaseErr := client.Release(); err == nil && releaseErr != nil {
			err = fmt.Errorf("release ledger state: %w", releaseErr)
		}
		return err
	})
}

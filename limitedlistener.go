// Package limitedlistener provides a TCP listener that enforces global and per-connection
// bandwidth limits on data transfer. It uses the `golang.org/x/time/rate` package to
// implement rate limiting and ensures that the bandwidth consumed by all connections
// stays within the specified limits.
package limitedlistener

import (
	"context"
	"fmt"
	"net"
	"sync"

	"golang.org/x/time/rate"
)

var (
	ErrLimitOutOfRange = fmt.Errorf("bandwidth limits must be higher than zero")
	ErrInvalidLimits   = fmt.Errorf("global bandwidth limit must be equal or higher than per conn bandwidth limit")
)

// LimitedConnection wraps a net.Conn and enforces both global and per-connection bandwidth limits on the Read operation.
type LimitedConnection struct {
	net.Conn
	globalLimiter *rate.Limiter
	limiter       *rate.Limiter
}

// NewLimitedConnection creates a new LimitedConnection with the specified global and per-connection bandwidth limits.
//
// Parameters:
//   - conn: The underlying net.Conn to wrap.
//   - globalLimiter: The global rate limiter shared across all connections.
//   - bytesPerSecond: The per-connection bandwidth limit in bytes per second.
func NewLimitedConnection(conn net.Conn, globalLimiter *rate.Limiter, bytesPerSecond int) *LimitedConnection {
	limiter := rate.NewLimiter(rate.Limit(bytesPerSecond), bytesPerSecond)
	return &LimitedConnection{
		Conn:          conn,
		globalLimiter: globalLimiter,
		limiter:       limiter,
	}
}

// Read reads data from the connection while respecting the global and per-connection bandwidth limits.
// It ensures that the data transfer rate does not exceed the specified limits.
func (lc *LimitedConnection) Read(b []byte) (int, error) {
	allowed := len(b)

	if allowed > lc.limiter.Burst() {
		allowed = lc.limiter.Burst()
	}

	ctx := context.Background()

	err := lc.globalLimiter.WaitN(ctx, allowed)
	if err != nil {
		return 0, err
	}
	err = lc.limiter.WaitN(ctx, allowed)
	if err != nil {
		return 0, err
	}

	return lc.Conn.Read(b[:allowed])
}

// LimitedListener wraps a net.Listener and enforces global and per-connection bandwidth limits on all accepted connections.
type LimitedListener struct {
	net.Listener
	globalLimiter         *rate.Limiter
	perConnBandwidthLimit int
	sync.RWMutex
	connections []*LimitedConnection
}

// NewLimitedListener creates a new LimitedListener with the specified global and per-connection bandwidth limits.
//
// Parameters:
//   - listener: The underlying net.Listener to wrap.
//   - globalLimit: The global bandwidth limit in bytes per second.
//   - perConnLimit: The per-connection bandwidth limit in bytes per second.
func NewLimitedListener(listener net.Listener, globalLimit, perConnLimit int) (*LimitedListener, error) {

	if globalLimit <= 0 || perConnLimit <= 0 {
		return nil, ErrLimitOutOfRange
	}
	if globalLimit < perConnLimit {
		return nil, ErrInvalidLimits
	}

	globalLimiter := rate.NewLimiter(rate.Limit(globalLimit), globalLimit)

	return &LimitedListener{
		Listener:              listener,
		globalLimiter:         globalLimiter,
		perConnBandwidthLimit: perConnLimit,
	}, nil
}

// Accept accepts incoming connections and wraps them with a LimitedConnection to enforce bandwidth limits.
func (l *LimitedListener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}

	l.RLock()
	defer l.RUnlock()

	limitedConnection := NewLimitedConnection(conn, l.globalLimiter, l.perConnBandwidthLimit)
	l.connections = append(l.connections, limitedConnection)

	return limitedConnection, nil
}

// SetLimits updates the global and per-connection bandwidth limits for the listener and all active connections.
func (l *LimitedListener) SetLimits(global, perConn int) {
	if global <= 0 || perConn <= 0 || global < perConn {
		return
	}
	l.Lock()
	defer l.Unlock()

	l.globalLimiter.SetLimit(rate.Limit(global))
	l.globalLimiter.SetBurst(global)
	l.perConnBandwidthLimit = perConn

	for _, connection := range l.connections {
		connection.limiter.SetLimit(rate.Limit(perConn))
		connection.limiter.SetBurst(perConn)
	}
}

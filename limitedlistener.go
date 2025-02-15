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

type LimitedConnection struct {
	net.Conn
	globalLimiter *rate.Limiter
	limiter       *rate.Limiter
}

func NewLimitedConnection(conn net.Conn, globalLimiter *rate.Limiter, bytesPerSecond int) *LimitedConnection {
	limiter := rate.NewLimiter(rate.Limit(bytesPerSecond), bytesPerSecond)
	return &LimitedConnection{
		Conn:          conn,
		globalLimiter: globalLimiter,
		limiter:       limiter,
	}
}

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

type LimitedListener struct {
	net.Listener
	globalLimiter         *rate.Limiter
	perConnBandwidthLimit int
	sync.RWMutex
	connections []*LimitedConnection
}

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

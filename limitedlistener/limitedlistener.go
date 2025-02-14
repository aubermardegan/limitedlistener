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
	limiter *rate.Limiter
}

func NewLimitedConnection(conn net.Conn, bytesPerSecond int) *LimitedConnection {
	limiter := rate.NewLimiter(rate.Limit(bytesPerSecond), bytesPerSecond)
	return &LimitedConnection{
		Conn:    conn,
		limiter: limiter,
	}
}

func (lc *LimitedConnection) Write(b []byte) (int, error) {

	err := lc.limiter.WaitN(context.Background(), len(b))
	if err != nil {
		return 0, err
	}

	return lc.Conn.Write(b)
}

type LimitedListener struct {
	net.Listener
	globalBandwidthLimit  int
	perConnBandwidthLimit int
	sync.RWMutex
}

func NewLimitedListener(listener net.Listener, globalLimit, perConnLimit int) (*LimitedListener, error) {

	if globalLimit <= 0 || perConnLimit <= 0 {
		return nil, ErrLimitOutOfRange
	}
	if globalLimit < perConnLimit {
		return nil, ErrInvalidLimits
	}

	return &LimitedListener{
		Listener:              listener,
		globalBandwidthLimit:  globalLimit,
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

	limitedConnection := NewLimitedConnection(conn, l.perConnBandwidthLimit)
	return limitedConnection, nil
}

func (l *LimitedListener) SetLimits(global, perConn int) {
	if global <= 0 || perConn <= 0 || global < perConn {
		return
	}
	l.Lock()
	l.globalBandwidthLimit = global
	l.perConnBandwidthLimit = perConn
	l.Unlock()
}

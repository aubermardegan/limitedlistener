package limitedlistener

import (
	"errors"
	"net"
	"testing"
)

// TestIfSatisfiesLimitListenerInterface verifies that the LimitedListener type implements the LimitListener interface.
func TestIfSatisfiesLimitListenerInterface(t *testing.T) {
	type LimitListener interface {
		net.Listener
		SetLimits(global, perConn int)
	}

	var _ LimitListener = (*LimitedListener)(nil)
}

// TestNewLimitedListenerValidationErrors tests the validation logic in the NewLimitedListener function.
// It ensures that the function returns appropriate errors for invalid inputs, such as zero or negative limits,
// and when the global limit is less than the per-connection limit.
func TestNewLimitedListenerValidationErros(t *testing.T) {
	testCases := []struct {
		test    string
		global  int
		perConn int
		want    error
	}{
		{
			"Value 0 on global limit",
			0,
			10,
			ErrLimitOutOfRange,
		},
		{
			"Value 0 on perConn limit",
			10,
			0,
			ErrLimitOutOfRange,
		},
		{
			"Negative value 0 on global limit",
			-10,
			0,
			ErrLimitOutOfRange,
		},
		{
			"Global limit lower than perConn limit",
			10,
			100,
			ErrInvalidLimits,
		},
		{
			"Valid values (same value)",
			10,
			10,
			nil,
		},
		{
			"Valid values (different value)",
			100,
			10,
			nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.test, func(t *testing.T) {
			listener, _ := net.Listen("tcp", ":8080")
			_, got := NewLimitedListener(listener, tc.global, tc.perConn)

			if !errors.Is(got, tc.want) {
				t.Errorf("expected %s, but got %s", tc.want.Error(), got.Error())
			}
		})
	}
}

// TestSetLimits tests the SetLimits method of the LimitedListener type.
// It verifies that the method correctly updates the global and per-connection bandwidth limits for the listener and all active connections.
// It also ensures that invalid inputs do not change the existing limits.
func TestSetLimits(t *testing.T) {

	testCases := []struct {
		test        string
		global      int
		perConn     int
		wantGlobal  int
		wantPerConn int
	}{
		{
			"Valid parameters should set the new limits",
			20,
			10,
			20,
			10,
		},
		{
			"Invalid params should not change the limits",
			0,
			-10,
			100,
			50,
		},
		{
			"Global limit lesser than PerConn limit should not change the limits",
			50,
			60,
			100,
			50,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.test, func(t *testing.T) {
			listener, _ := net.Listen("tcp", ":8080")
			limitedListener, err := NewLimitedListener(listener, 100, 50)
			if err != nil {
				t.Errorf("didn't expected error but got one")
			}

			limitedListener.SetLimits(tc.global, tc.perConn)
			limitedListener.RLock()
			gotGlobal := int(limitedListener.globalLimiter.Limit())
			gotPerConn := limitedListener.perConnBandwidthLimit
			if gotGlobal != tc.wantGlobal || gotPerConn != tc.wantPerConn {
				t.Errorf("expected: global: %d, perConn %d, but got global: %d, perConn %d", tc.wantGlobal, tc.wantPerConn, gotGlobal, gotPerConn)
			}

			for connection := range limitedListener.connections {
				if int(connection.limiter.Limit()) != tc.wantPerConn || connection.limiter.Burst() != tc.wantPerConn {
					t.Errorf("expected: global: %d, perConn %d, but got global: %d, perConn %d", tc.wantGlobal, tc.wantPerConn, gotGlobal, gotPerConn)
				}
			}

			limitedListener.RUnlock()
		})
	}
}

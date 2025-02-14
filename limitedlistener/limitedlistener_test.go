package limitedlistener

import (
	"errors"
	"net"
	"testing"
)

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
				t.Errorf("Expected %s, but got %s", tc.want.Error(), got.Error())
			}
		})
	}
}

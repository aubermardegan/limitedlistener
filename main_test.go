package main

import (
	"net"
	"testing"
)

func Test_Size(t *testing.T) {
	testCases := []struct {
		test    string
		payload []byte
		want    int
	}{
		{
			"Test1",
			[]byte("hello world"),
			len([]byte("hello world")),
		},
		{
			"Test2",
			make([]byte, 1000),
			1000,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.test, func(t *testing.T) {
			conn, err := net.Dial("tcp", ":8080")
			if err != nil {
				t.Error("could not connect to TCP server: ", err)
			}
			defer conn.Close()

			if _, err := conn.Write(tc.payload); err != nil {
				t.Error("could not write payload to TCP server:", err)
			}

			out := make([]byte, 1024)
			if bytesReceived, err := conn.Read(out); err == nil {
				if bytesReceived != tc.want {
					t.Errorf("response did match expected size: want: %d, got: %d", tc.want, bytesReceived)
				}
			} else {
				t.Error("could not read from connection")
			}
		})
	}
}

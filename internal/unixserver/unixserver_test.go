package unixserver

import (
	"net"
	"path/filepath"
	"testing"
	"time"
)

func TestUnixServer(t *testing.T) {
	for _, count := range []int{0, 10} {
		connected := make(chan struct{})

		s := Server{
			Address: filepath.Join(t.TempDir(), "server.sock"),
			ServeConn: func(net.Conn) {
				connected <- struct{}{}
			},
		}

		if err := s.ListenAndServe(); err != nil {
			t.Errorf("ListenAndServe() failed: %v", err)
		}

		for i := 0; i < count; i++ {
			if conn, err := net.Dial("unix", s.Address); err != nil {
				t.Errorf("Dial() failed: %v", err)
			} else {
				const timeout = 5 * time.Second

				select {
				case <-connected:
				case <-time.After(timeout):
					t.Errorf("Not connected after %v", timeout)
				}

				if err := conn.Close(); err != nil {
					t.Errorf("Close() failed: %v", err)
				}
			}
		}

		if err := s.Close(); err != nil {
			t.Errorf("Close() failed: %v", err)
		}
	}
}

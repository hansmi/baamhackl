package unixserver

import (
	"errors"
	"fmt"
	"net"
	"sync"

	"go.uber.org/zap"
)

type Server struct {
	Logger    *zap.Logger
	Address   string
	ServeConn func(net.Conn)

	mu       sync.Mutex
	listener net.Listener
	quit     chan struct{}
}

func (s *Server) ListenAndServe() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.listener != nil {
		return errors.New("server already started")
	}

	if s.quit == nil {
		s.quit = make(chan struct{})
	}

	logger := s.Logger

	listener, err := net.Listen("unix", s.Address)
	if err != nil {
		return fmt.Errorf("unable to listen at %s: %s", s.Address, err)
	}

	if ul := listener.(*net.UnixListener); ul != nil {
		ul.SetUnlinkOnClose(true)
	}

	s.listener = listener

	go func() {
		if err := s.serve(listener); err != nil {
			if logger == nil {
				logger = zap.L()
			}

			logger.Error("UnixServer failed", zap.Error(err))
		}
	}()

	return nil
}

func (s *Server) serve(listener net.Listener) error {
	for {
		conn, err := listener.Accept()
		if err != nil {
			s.mu.Lock()
			select {
			case <-s.quit:
				err = nil
			default:
				err = fmt.Errorf("listener accept failed: %w", err)
			}
			s.mu.Unlock()

			return err
		}

		go s.ServeConn(conn)
	}
}

// Close immediately closes all listeners. Running connection handlers are not
// affected.
func (s *Server) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.quit == nil {
		return nil
	}

	select {
	case <-s.quit:
	default:
		close(s.quit)
		if s.listener != nil {
			return s.listener.Close()
		}
	}

	return nil
}

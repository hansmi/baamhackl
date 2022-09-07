package service

import (
	"io"
	"net"
	"net/rpc"

	"github.com/hansmi/baamhackl/internal/unixserver"
	"github.com/hansmi/baamhackl/internal/watchman"
)

type FileChangedRequest struct {
	HandlerName string
	RootDir     string
	Change      watchman.FileChange
}

type FileChangedResponse struct {
}

type Callbacks interface {
	FileChanged(FileChangedRequest) error
}

type serviceFunctions struct {
	cb Callbacks
}

func (s *serviceFunctions) FileChanged(req FileChangedRequest, _ *FileChangedResponse) error {
	return s.cb.FileChanged(req)
}

func newService(cb Callbacks) (*rpc.Server, error) {
	srv := rpc.NewServer()

	if err := srv.RegisterName("Service", &serviceFunctions{cb}); err != nil {
		return nil, err
	}

	return srv, nil
}

func ListenAndServe(address string, cb Callbacks) (io.Closer, error) {
	srv, err := newService(cb)
	if err != nil {
		return nil, err
	}

	server := &unixserver.Server{
		Address: address,
		ServeConn: func(conn net.Conn) {
			srv.ServeConn(conn)
		},
	}

	if err := server.ListenAndServe(); err != nil {
		return nil, err
	}

	return server, nil
}

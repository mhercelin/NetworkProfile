package ipc

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"networkprofile/internal/network"
)

// Server answers privileged requests. It runs inside the elevated helper, where
// the real netsh-backed manager lives.
type Server struct {
	manager network.Manager
}

func NewServer(manager network.Manager) *Server {
	return &Server{manager: manager}
}

// ErrShutdown is returned by Serve when the caller asked the helper to stop.
var ErrShutdown = errors.New("arrêt demandé")

// Serve answers requests on one connection until it closes, or until the caller
// asks the helper to shut down.
func (s *Server) Serve(conn io.ReadWriter) error {
	dec := json.NewDecoder(conn)
	enc := json.NewEncoder(conn)

	for {
		var req Request
		if err := dec.Decode(&req); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}

		if req.Op == OpShutdown {
			// Acknowledge before stopping, so the caller knows the helper meant
			// to go rather than having crashed.
			_ = enc.Encode(Response{})
			return ErrShutdown
		}

		if err := enc.Encode(s.Handle(req)); err != nil {
			return err
		}
	}
}

// Handle carries out one request. It is separated from the transport so the
// privileged behaviour can be tested without a pipe or an elevated process.
func (s *Server) Handle(req Request) Response {
	switch req.Op {
	case OpPing:
		return Response{}
	case OpSetStatic:
		return responseFor(s.manager.SetStatic(req.Interface, req.Static))
	case OpSetDHCP:
		return responseFor(s.manager.SetDHCP(req.Interface))
	case OpConnectWiFi:
		return responseFor(s.manager.ConnectWiFi(req.Interface, req.SSID))
	default:
		return Response{Error: fmt.Sprintf("opération inconnue : %q", req.Op)}
	}
}

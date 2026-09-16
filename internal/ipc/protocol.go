// Package ipc carries adapter writes from the unprivileged interface to the
// elevated helper.
//
// The interface itself runs as the operator, so it reads adapter state and
// stores profiles in that operator's own profile directory. Only the writes
// need administrator rights, and on a machine where the operator is not an
// administrator, elevation means a whole separate account — so the two halves
// cannot share memory, only this pipe.
package ipc

import (
	"fmt"

	"networkprofile/internal/network"
)

// Op is the privileged action being asked for.
type Op string

const (
	OpSetStatic   Op = "set-static"
	OpSetDHCP     Op = "set-dhcp"
	OpConnectWiFi Op = "connect-wifi"
	OpPing        Op = "ping"
	OpShutdown    Op = "shutdown"
)

type Request struct {
	Op        Op                   `json:"op"`
	Interface string               `json:"interface,omitempty"`
	Static    network.StaticConfig `json:"static,omitzero"`
	SSID      string               `json:"ssid,omitempty"`
}

type Response struct {
	Error string `json:"error,omitempty"`
}

// Err rebuilds the failure on the caller's side. The helper reports a message
// rather than an error value: nothing survives a pipe but text.
func (r Response) Err() error {
	if r.Error == "" {
		return nil
	}
	return fmt.Errorf("%s", r.Error)
}

func responseFor(err error) Response {
	if err == nil {
		return Response{}
	}
	return Response{Error: err.Error()}
}

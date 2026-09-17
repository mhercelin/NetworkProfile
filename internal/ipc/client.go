package ipc

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"networkprofile/internal/network"
)

// Client speaks to the elevated helper over a single connection. Requests are
// serialised: the helper answers one at a time, and applying a profile issues
// several in a row.
type Client struct {
	mu     sync.Mutex
	conn   io.ReadWriteCloser
	enc    *json.Encoder
	dec    *json.Decoder
	broken bool
}

func NewClient(conn io.ReadWriteCloser) *Client {
	return &Client{
		conn: conn,
		enc:  json.NewEncoder(conn),
		dec:  json.NewDecoder(conn),
	}
}

func (c *Client) Do(req Request) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// A transport failure and a refusal from netsh are very different things:
	// only the first means the helper is gone and has to be started again.
	if err := c.enc.Encode(req); err != nil {
		c.broken = true
		return fmt.Errorf("envoi à l'assistant : %w", err)
	}

	var resp Response
	if err := c.dec.Decode(&resp); err != nil {
		c.broken = true
		return fmt.Errorf("réponse de l'assistant : %w", err)
	}
	return resp.Err()
}

// Broken reports that the connection itself failed, as opposed to the helper
// answering that it could not carry out the request.
func (c *Client) Broken() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.broken
}

func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.broken {
		// Best effort: the helper also exits on its own when this process does.
		_ = c.enc.Encode(Request{Op: OpShutdown})
		var resp Response
		_ = c.dec.Decode(&resp)
	}
	return c.conn.Close()
}

// Manager applies writes through the helper while answering reads locally:
// listing adapters and known networks needs no privileges at all, so routing
// them through an elevated process would buy nothing.
//
// The helper is started on the first write rather than at launch, so opening
// the application merely to look at the current configuration never raises a
// credentials prompt.
type Manager struct {
	local network.Manager

	mu     sync.Mutex
	dial   func() (*Client, error)
	client *Client
}

var _ network.Manager = (*Manager)(nil)

func NewManager(local network.Manager, dial func() (*Client, error)) *Manager {
	return &Manager{local: local, dial: dial}
}

func (m *Manager) Interfaces() ([]network.Interface, error) {
	return m.local.Interfaces()
}

func (m *Manager) KnownWiFiNetworks(iface string) ([]string, error) {
	return m.local.KnownWiFiNetworks(iface)
}

func (m *Manager) SetStatic(iface string, cfg network.StaticConfig) error {
	return m.send(Request{Op: OpSetStatic, Interface: iface, Static: cfg})
}

func (m *Manager) SetDHCP(iface string) error {
	return m.send(Request{Op: OpSetDHCP, Interface: iface})
}

func (m *Manager) ConnectWiFi(iface, ssid string) error {
	return m.send(Request{Op: OpConnectWiFi, Interface: iface, SSID: ssid})
}

func (m *Manager) send(req Request) error {
	client, err := m.helper()
	if err != nil {
		return err
	}

	err = client.Do(req)
	if client.Broken() {
		// The helper died; forget it so the next attempt starts a fresh one.
		m.mu.Lock()
		if m.client == client {
			m.client = nil
		}
		m.mu.Unlock()
	}
	return err
}

func (m *Manager) helper() (*Client, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.client != nil {
		return m.client, nil
	}

	client, err := m.dial()
	if err != nil {
		return nil, err
	}
	m.client = client
	return client, nil
}

// Elevated reports whether the privileged helper is running. Until the first
// change is applied, no privileges have been asked for at all — and saying
// otherwise in the interface would be a plain lie.
func (m *Manager) Elevated() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.client != nil
}

// Close stops the helper if one was ever started.
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.client == nil {
		return nil
	}
	err := m.client.Close()
	m.client = nil
	return err
}

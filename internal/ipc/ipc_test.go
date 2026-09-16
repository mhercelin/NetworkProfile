package ipc

import (
	"errors"
	"net"
	"reflect"
	"strings"
	"testing"

	"networkprofile/internal/network"
)

func TestHandle(t *testing.T) {
	cfg := network.StaticConfig{Address: "10.10.128.20", Mask: "255.255.255.0", Gateway: "10.10.128.1"}

	tests := []struct {
		name      string
		request   Request
		wantCalls []network.Call
		wantError bool
	}{
		{
			name:      "ping touches nothing",
			request:   Request{Op: OpPing},
			wantCalls: nil,
		},
		{
			name:      "static address",
			request:   Request{Op: OpSetStatic, Interface: "Ethernet", Static: cfg},
			wantCalls: []network.Call{{Op: "static", Interface: "Ethernet", Static: cfg}},
		},
		{
			name:      "back to DHCP",
			request:   Request{Op: OpSetDHCP, Interface: "Wi-Fi"},
			wantCalls: []network.Call{{Op: "dhcp", Interface: "Wi-Fi"}},
		},
		{
			name:      "join a network",
			request:   Request{Op: OpConnectWiFi, Interface: "Wi-Fi", SSID: "ATELIER-5G"},
			wantCalls: []network.Call{{Op: "wifi", Interface: "Wi-Fi", SSID: "ATELIER-5G"}},
		},
		{
			name:      "unknown operation is refused, not ignored",
			request:   Request{Op: "reboot"},
			wantCalls: nil,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &network.Fake{}

			resp := NewServer(fake).Handle(tt.request)

			if tt.wantError && resp.Err() == nil {
				t.Fatal("expected the request to be refused")
			}
			if !tt.wantError && resp.Err() != nil {
				t.Fatalf("unexpected error: %v", resp.Err())
			}
			if !reflect.DeepEqual(fake.Calls, tt.wantCalls) {
				t.Fatalf("unexpected calls\ngot:  %+v\nwant: %+v", fake.Calls, tt.wantCalls)
			}
		})
	}
}

func TestHandleReportsAFailureFromTheAdapter(t *testing.T) {
	fake := &network.Fake{FailOn: "Ethernet"}

	resp := NewServer(fake).Handle(Request{Op: OpSetDHCP, Interface: "Ethernet"})

	if resp.Err() == nil {
		t.Fatal("expected the adapter failure to be reported back")
	}
	if !strings.Contains(resp.Error, "Ethernet") {
		t.Fatalf("the message must survive the pipe intact, got: %q", resp.Error)
	}
}

// connected wires a real Client to a real Server over an in-memory pipe, which
// exercises the framing without needing an elevated process.
func connected(t *testing.T, manager network.Manager) *Client {
	t.Helper()

	clientSide, serverSide := net.Pipe()
	server := NewServer(manager)

	go func() {
		_ = server.Serve(serverSide)
		serverSide.Close()
	}()

	client := NewClient(clientSide)
	t.Cleanup(func() { clientSide.Close() })
	return client
}

func TestClientServerRoundTrip(t *testing.T) {
	fake := &network.Fake{}
	client := connected(t, fake)

	if err := client.Do(Request{Op: OpSetDHCP, Interface: "Ethernet"}); err != nil {
		t.Fatalf("Do: %v", err)
	}
	if err := client.Do(Request{Op: OpConnectWiFi, Interface: "Wi-Fi", SSID: "ATELIER-5G"}); err != nil {
		t.Fatalf("Do: %v", err)
	}

	want := []network.Call{
		{Op: "dhcp", Interface: "Ethernet"},
		{Op: "wifi", Interface: "Wi-Fi", SSID: "ATELIER-5G"},
	}
	if !reflect.DeepEqual(fake.Calls, want) {
		t.Fatalf("unexpected calls\ngot:  %+v\nwant: %+v", fake.Calls, want)
	}
}

func TestClientSurfacesAHelperSideFailure(t *testing.T) {
	client := connected(t, &network.Fake{FailOn: "Ethernet"})

	err := client.Do(Request{Op: OpSetDHCP, Interface: "Ethernet"})

	if err == nil {
		t.Fatal("expected the helper-side failure to reach the caller")
	}
	if !strings.Contains(err.Error(), "Ethernet") {
		t.Fatalf("unexpected message: %v", err)
	}
}

func TestServeStopsOnShutdown(t *testing.T) {
	clientSide, serverSide := net.Pipe()
	defer clientSide.Close()

	done := make(chan error, 1)
	go func() { done <- NewServer(&network.Fake{}).Serve(serverSide) }()

	NewClient(clientSide).Do(Request{Op: OpShutdown})

	if err := <-done; !errors.Is(err, ErrShutdown) {
		t.Fatalf("expected ErrShutdown, got: %v", err)
	}
}

// countingDial reports how many times the helper had to be started, which is
// the number of credentials prompts the operator would have seen.
func countingDial(t *testing.T, privileged network.Manager, starts *int) func() (*Client, error) {
	t.Helper()
	return func() (*Client, error) {
		*starts++
		return connected(t, privileged), nil
	}
}

// The manager keeps reads local: an elevated round trip to list adapters would
// buy nothing, since listing them needs no privileges — and it must never cost
// a credentials prompt.
func TestManagerAnswersReadsWithoutStartingTheHelper(t *testing.T) {
	local := &network.Fake{
		Adapters: []network.Interface{{Name: "Ethernet", Kind: network.KindEthernet}},
		WiFi:     map[string][]string{"Wi-Fi": {"ATELIER-5G"}},
	}
	privileged := &network.Fake{}
	starts := 0
	manager := NewManager(local, countingDial(t, privileged, &starts))

	adapters, err := manager.Interfaces()
	if err != nil {
		t.Fatalf("Interfaces: %v", err)
	}
	networks, err := manager.KnownWiFiNetworks("Wi-Fi")
	if err != nil {
		t.Fatalf("KnownWiFiNetworks: %v", err)
	}

	if len(adapters) != 1 || adapters[0].Name != "Ethernet" {
		t.Fatalf("unexpected adapters: %+v", adapters)
	}
	if len(networks) != 1 || networks[0] != "ATELIER-5G" {
		t.Fatalf("unexpected networks: %v", networks)
	}
	if starts != 0 {
		t.Fatalf("reads must not raise an elevation prompt, helper started %d time(s)", starts)
	}
	if len(privileged.Calls) != 0 {
		t.Fatalf("reads must not reach the helper, got: %+v", privileged.Calls)
	}
}

func TestManagerSendsWritesToTheHelper(t *testing.T) {
	local := &network.Fake{}
	privileged := &network.Fake{}
	starts := 0
	manager := NewManager(local, countingDial(t, privileged, &starts))
	cfg := network.StaticConfig{Address: "192.168.1.50", Mask: "255.255.255.0"}

	if err := manager.SetStatic("Ethernet", cfg); err != nil {
		t.Fatalf("SetStatic: %v", err)
	}

	if len(local.Calls) != 0 {
		t.Fatalf("writes must not be applied locally, got: %+v", local.Calls)
	}
	want := []network.Call{{Op: "static", Interface: "Ethernet", Static: cfg}}
	if !reflect.DeepEqual(privileged.Calls, want) {
		t.Fatalf("unexpected calls\ngot:  %+v\nwant: %+v", privileged.Calls, want)
	}
}

// This is the whole point of the design: one prompt, then every later change
// goes through the helper already running.
func TestManagerStartsTheHelperOnlyOnce(t *testing.T) {
	privileged := &network.Fake{}
	starts := 0
	manager := NewManager(&network.Fake{}, countingDial(t, privileged, &starts))

	for range 5 {
		if err := manager.SetDHCP("Ethernet"); err != nil {
			t.Fatalf("SetDHCP: %v", err)
		}
	}

	if starts != 1 {
		t.Fatalf("expected a single elevation, got %d", starts)
	}
	if len(privileged.Calls) != 5 {
		t.Fatalf("expected 5 writes to reach the helper, got %d", len(privileged.Calls))
	}
}

// A refusal from netsh must not be mistaken for a dead helper, or every failed
// change would cost another credentials prompt.
func TestManagerKeepsTheHelperAfterARefusedChange(t *testing.T) {
	privileged := &network.Fake{FailOn: "Ethernet"}
	starts := 0
	manager := NewManager(&network.Fake{}, countingDial(t, privileged, &starts))

	if err := manager.SetDHCP("Ethernet"); err == nil {
		t.Fatal("expected the refusal to surface")
	}
	if err := manager.SetDHCP("Wi-Fi"); err != nil {
		t.Fatalf("SetDHCP: %v", err)
	}

	if starts != 1 {
		t.Fatalf("expected a single elevation, got %d", starts)
	}
}

func TestManagerReportsARefusedElevation(t *testing.T) {
	refused := errors.New("élévation refusée")
	manager := NewManager(&network.Fake{}, func() (*Client, error) { return nil, refused })

	err := manager.SetDHCP("Ethernet")

	if !errors.Is(err, refused) {
		t.Fatalf("expected the refusal to reach the caller, got: %v", err)
	}
}

//go:build windows && integration

package network

import "testing"

// Read-only integration check: it reports what this machine actually says, so
// the reading layer can be compared against reality. It changes nothing.
//
//	go test -tags integration ./internal/network -run TestRealAdapters -v
func TestRealAdapters(t *testing.T) {
	w := NewWindows()

	list, err := w.Interfaces()
	if err != nil {
		t.Fatalf("Interfaces: %v", err)
	}
	if len(list) == 0 {
		t.Fatal("no Ethernet or Wi-Fi adapter reported")
	}

	for _, iface := range list {
		t.Logf("%-38s kind=%-8s virtual=%-5t up=%-5t dhcp=%-5t addr=%s mask=%s gw=%s dns=%v",
			iface.Name, iface.Kind, iface.Virtual, iface.Up, iface.DHCP,
			iface.Address, iface.Mask, iface.Gateway, iface.DNS)

		if iface.Kind != KindWiFi {
			continue
		}
		networks, err := w.KnownWiFiNetworks(iface.Name)
		if err != nil {
			t.Logf("    known networks unavailable: %v", err)
			continue
		}
		t.Logf("    %d known networks: %v", len(networks), networks)
	}
}

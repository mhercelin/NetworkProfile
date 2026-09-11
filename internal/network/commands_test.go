package network

import (
	"reflect"
	"testing"
)

func TestStaticCommands(t *testing.T) {
	tests := []struct {
		name  string
		iface string
		cfg   StaticConfig
		want  [][]string
	}{
		{
			name:  "address, mask, gateway and two DNS servers",
			iface: "Ethernet",
			cfg: StaticConfig{
				Address: "10.10.128.20",
				Mask:    "255.255.255.0",
				Gateway: "10.10.128.1",
				DNS:     []string{"10.10.128.1", "9.9.9.9"},
			},
			want: [][]string{
				{"interface", "ipv4", "set", "address", "name=Ethernet", "static", "10.10.128.20", "255.255.255.0", "10.10.128.1"},
				{"interface", "ipv4", "set", "dnsservers", "name=Ethernet", "static", "10.10.128.1", "primary", "validate=no"},
				{"interface", "ipv4", "add", "dnsservers", "name=Ethernet", "9.9.9.9", "index=2", "validate=no"},
			},
		},
		{
			name:  "no gateway leaves the argument out entirely",
			iface: "Ethernet",
			cfg:   StaticConfig{Address: "192.168.1.50", Mask: "255.255.255.0", DNS: []string{"192.168.1.1"}},
			want: [][]string{
				{"interface", "ipv4", "set", "address", "name=Ethernet", "static", "192.168.1.50", "255.255.255.0"},
				{"interface", "ipv4", "set", "dnsservers", "name=Ethernet", "static", "192.168.1.1", "primary", "validate=no"},
			},
		},
		{
			name:  "no DNS server clears the ones already set",
			iface: "Ethernet",
			cfg:   StaticConfig{Address: "192.168.1.50", Mask: "255.255.255.0", Gateway: "192.168.1.1"},
			want: [][]string{
				{"interface", "ipv4", "set", "address", "name=Ethernet", "static", "192.168.1.50", "255.255.255.0", "192.168.1.1"},
				{"interface", "ipv4", "set", "dnsservers", "name=Ethernet", "source=static", "address=none"},
			},
		},
		{
			name:  "an interface name holding a space stays a single argument",
			iface: "Ethernet 2",
			cfg:   StaticConfig{Address: "172.16.4.10", Mask: "255.255.0.0"},
			want: [][]string{
				{"interface", "ipv4", "set", "address", "name=Ethernet 2", "static", "172.16.4.10", "255.255.0.0"},
				{"interface", "ipv4", "set", "dnsservers", "name=Ethernet 2", "source=static", "address=none"},
			},
		},
		{
			name:  "a third DNS server is indexed after the second",
			iface: "Wi-Fi",
			cfg: StaticConfig{
				Address: "172.16.32.8",
				Mask:    "255.255.0.0",
				DNS:     []string{"172.16.0.1", "9.9.9.9", "1.1.1.1"},
			},
			want: [][]string{
				{"interface", "ipv4", "set", "address", "name=Wi-Fi", "static", "172.16.32.8", "255.255.0.0"},
				{"interface", "ipv4", "set", "dnsservers", "name=Wi-Fi", "static", "172.16.0.1", "primary", "validate=no"},
				{"interface", "ipv4", "add", "dnsservers", "name=Wi-Fi", "9.9.9.9", "index=2", "validate=no"},
				{"interface", "ipv4", "add", "dnsservers", "name=Wi-Fi", "1.1.1.1", "index=3", "validate=no"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := staticCommands(tt.iface, tt.cfg)

			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("unexpected commands\ngot:  %v\nwant: %v", got, tt.want)
			}
		})
	}
}

func TestDHCPCommandsResetAddressAndDNS(t *testing.T) {
	want := [][]string{
		{"interface", "ipv4", "set", "address", "name=Ethernet", "source=dhcp"},
		{"interface", "ipv4", "set", "dnsservers", "name=Ethernet", "source=dhcp"},
	}

	got := dhcpCommands("Ethernet")

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected commands\ngot:  %v\nwant: %v", got, want)
	}
}

func TestConnectWiFiCommand(t *testing.T) {
	want := []string{"wlan", "connect", "name=Salle de formation 2", "interface=Wi-Fi"}

	got := connectWiFiCommand("Wi-Fi", "Salle de formation 2")

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected command\ngot:  %v\nwant: %v", got, want)
	}
}

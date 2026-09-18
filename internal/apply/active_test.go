package apply

import (
	"reflect"
	"testing"

	"networkprofile/internal/network"
	"networkprofile/internal/profile"
)

func staticProfile(id, iface, address, mask, gateway string) profile.Profile {
	return profile.Profile{
		ID:   id,
		Name: id,
		Targets: []profile.Target{{
			Interface: iface, Mode: profile.ModeStatic,
			Address: address, Mask: mask, Gateway: gateway,
		}},
	}
}

func dhcpProfile(id string, interfaces ...string) profile.Profile {
	p := profile.Profile{ID: id, Name: id}
	for _, iface := range interfaces {
		p.Targets = append(p.Targets, profile.Target{Interface: iface, Mode: profile.ModeDHCP})
	}
	return p
}

func TestActiveIDs(t *testing.T) {
	atelier := staticProfile("atelier", "Ethernet", "192.168.1.50", "255.255.255.0", "192.168.1.1")
	automate := staticProfile("automate", "Ethernet", "192.168.0.241", "255.255.255.0", "")
	chantier := staticProfile("chantier", "Wi-Fi", "172.16.32.8", "255.255.0.0", "172.16.0.1")
	defaults := dhcpProfile("defaut", "Ethernet", "Wi-Fi")
	profiles := []profile.Profile{defaults, atelier, automate, chantier}

	tests := []struct {
		name     string
		adapters []network.Interface
		want     []string
	}{
		{
			// The reason this returns a list: two adapters, two profiles, both
			// genuinely in effect.
			name: "an Ethernet profile and a Wi-Fi profile at once",
			adapters: []network.Interface{
				{Name: "Ethernet", Address: "192.168.1.50", Mask: "255.255.255.0", Gateway: "192.168.1.1"},
				{Name: "Wi-Fi", Address: "172.16.32.8", Mask: "255.255.0.0", Gateway: "172.16.0.1"},
			},
			want: []string{"atelier", "chantier"},
		},
		{
			name: "one adapter matching, the other on nothing known",
			adapters: []network.Interface{
				{Name: "Ethernet", Address: "192.168.0.241", Mask: "255.255.255.0"},
				{Name: "Wi-Fi", Address: "10.0.0.5", Mask: "255.255.255.0"},
			},
			want: []string{"automate"},
		},
		{
			name: "both adapters on DHCP",
			adapters: []network.Interface{
				{Name: "Ethernet", DHCP: true},
				{Name: "Wi-Fi", DHCP: true},
			},
			want: []string{"defaut"},
		},
		{
			// The multi-adapter profile needs every one of its targets.
			name: "only one of the two adapters on DHCP",
			adapters: []network.Interface{
				{Name: "Ethernet", DHCP: true},
				{Name: "Wi-Fi", Address: "10.0.0.5", Mask: "255.255.255.0"},
			},
			want: []string{},
		},
		{
			name: "right address, wrong gateway",
			adapters: []network.Interface{
				{Name: "Ethernet", Address: "192.168.1.50", Mask: "255.255.255.0", Gateway: "192.168.1.254"},
			},
			want: []string{},
		},
		{
			// A fixed address that happens to equal the one the lease handed
			// out is still not the same configuration.
			name: "the address matches but the adapter is on DHCP",
			adapters: []network.Interface{
				{Name: "Ethernet", DHCP: true, Address: "192.168.1.50", Mask: "255.255.255.0", Gateway: "192.168.1.1"},
			},
			want: []string{},
		},
		{
			name:     "the adapter is not on this machine",
			adapters: []network.Interface{{Name: "Ethernet 9", DHCP: true}},
			want:     []string{},
		},
		{
			name:     "nothing to compare against",
			adapters: nil,
			want:     []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ActiveIDs(profiles, tt.adapters)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("ActiveIDs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestActiveIDsIgnoresDNS(t *testing.T) {
	p := profile.Profile{
		ID:   "atelier",
		Name: "Atelier",
		Targets: []profile.Target{{
			Interface: "Ethernet", Mode: profile.ModeStatic,
			Address: "192.168.1.50", Mask: "255.255.255.0",
			DNS: []string{"192.168.1.1"},
		}},
	}
	adapters := []network.Interface{
		// Windows supplemented the resolvers on its own.
		{Name: "Ethernet", Address: "192.168.1.50", Mask: "255.255.255.0", DNS: []string{"192.168.1.1", "9.9.9.9"}},
	}

	if got := ActiveIDs([]profile.Profile{p}, adapters); !reflect.DeepEqual(got, []string{"atelier"}) {
		t.Fatalf("expected the profile to still match, got %v", got)
	}
}

// A profile that names a network is only in effect on that network. Reported
// from the field: a Wi-Fi profile showed as applied while the adapter was
// connected to nothing.
func TestActiveIDsChecksTheNetworkAWiFiProfileNames(t *testing.T) {
	p := profile.Profile{
		ID:      "maison",
		Name:    "Maison",
		Targets: []profile.Target{{Interface: "Wi-Fi", Mode: profile.ModeDHCP, SSID: "LIVEBOX-1234"}},
	}

	tests := []struct {
		name    string
		adapter network.Interface
		want    bool
	}{
		{"on the named network", network.Interface{Name: "Wi-Fi", DHCP: true, SSID: "LIVEBOX-1234"}, true},
		{"case differs", network.Interface{Name: "Wi-Fi", DHCP: true, SSID: "livebox-1234"}, true},
		{"connected to nothing", network.Interface{Name: "Wi-Fi", DHCP: true}, false},
		{"connected elsewhere", network.Interface{Name: "Wi-Fi", DHCP: true, SSID: "VOISIN"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := len(ActiveIDs([]profile.Profile{p}, []network.Interface{tt.adapter})) == 1
			if got != tt.want {
				t.Fatalf("active = %t, want %t", got, tt.want)
			}
		})
	}
}

// A profile that names no network only changes the address, and stays in effect
// whatever the adapter is connected to.
func TestActiveIDsIgnoresTheNetworkWhenTheProfileNamesNone(t *testing.T) {
	p := profile.Profile{
		ID:      "adresse-seule",
		Name:    "Adresse seule",
		Targets: []profile.Target{{Interface: "Wi-Fi", Mode: profile.ModeDHCP}},
	}
	adapter := network.Interface{Name: "Wi-Fi", DHCP: true, SSID: "N_IMPORTE_QUOI"}

	if len(ActiveIDs([]profile.Profile{p}, []network.Interface{adapter})) != 1 {
		t.Fatal("expected the profile to be in effect regardless of the network")
	}
}

func TestActiveIDsIgnoresAProfileWithoutTargets(t *testing.T) {
	empty := profile.Profile{ID: "vide", Name: "Vide"}

	got := ActiveIDs([]profile.Profile{empty}, []network.Interface{{Name: "Ethernet", DHCP: true}})
	if len(got) != 0 {
		t.Fatalf("expected no match, got %v", got)
	}
}

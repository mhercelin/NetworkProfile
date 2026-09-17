package apply

import (
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

func TestActive(t *testing.T) {
	atelier := staticProfile("atelier", "Ethernet", "192.168.1.50", "255.255.255.0", "192.168.1.1")
	automate := staticProfile("automate", "Ethernet", "192.168.0.241", "255.255.255.0", "")
	defaults := dhcpProfile("defaut", "Ethernet", "Wi-Fi")
	profiles := []profile.Profile{defaults, atelier, automate}

	tests := []struct {
		name     string
		adapters []network.Interface
		want     string
	}{
		{
			name: "a static profile matching exactly",
			adapters: []network.Interface{
				{Name: "Ethernet", Address: "192.168.1.50", Mask: "255.255.255.0", Gateway: "192.168.1.1"},
				{Name: "Wi-Fi", DHCP: true},
			},
			want: "atelier",
		},
		{
			name: "a profile with no gateway, and none set",
			adapters: []network.Interface{
				{Name: "Ethernet", Address: "192.168.0.241", Mask: "255.255.255.0"},
				{Name: "Wi-Fi", DHCP: true},
			},
			want: "automate",
		},
		{
			// Every target has to match, which is what makes a multi-adapter
			// profile mean something.
			name: "both adapters on DHCP",
			adapters: []network.Interface{
				{Name: "Ethernet", DHCP: true},
				{Name: "Wi-Fi", DHCP: true},
			},
			want: "defaut",
		},
		{
			name: "only one of the two adapters on DHCP",
			adapters: []network.Interface{
				{Name: "Ethernet", DHCP: true},
				{Name: "Wi-Fi", Address: "10.0.0.5", Mask: "255.255.255.0"},
			},
			want: "",
		},
		{
			name: "right address, wrong gateway",
			adapters: []network.Interface{
				{Name: "Ethernet", Address: "192.168.1.50", Mask: "255.255.255.0", Gateway: "192.168.1.254"},
			},
			want: "",
		},
		{
			// A fixed address that happens to equal the one the lease handed
			// out is still not the same configuration.
			name: "the address matches but the adapter is on DHCP",
			adapters: []network.Interface{
				{Name: "Ethernet", DHCP: true, Address: "192.168.1.50", Mask: "255.255.255.0", Gateway: "192.168.1.1"},
			},
			want: "",
		},
		{
			name:     "the adapter is not on this machine",
			adapters: []network.Interface{{Name: "Ethernet 9", DHCP: true}},
			want:     "",
		},
		{
			name:     "nothing to compare against",
			adapters: nil,
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Active(profiles, tt.adapters); got != tt.want {
				t.Fatalf("Active() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestActiveIgnoresDNS(t *testing.T) {
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

	if got := Active([]profile.Profile{p}, adapters); got != "atelier" {
		t.Fatalf("expected the profile to still match, got %q", got)
	}
}

func TestActiveIgnoresAProfileWithoutTargets(t *testing.T) {
	empty := profile.Profile{ID: "vide", Name: "Vide"}

	if got := Active([]profile.Profile{empty}, []network.Interface{{Name: "Ethernet", DHCP: true}}); got != "" {
		t.Fatalf("expected no match, got %q", got)
	}
}

package apply

import (
	"reflect"
	"strings"
	"testing"

	"networkprofile/internal/network"
	"networkprofile/internal/profile"
)

func newFake() *network.Fake {
	return &network.Fake{
		Adapters: []network.Interface{
			{Name: "Ethernet", Kind: network.KindEthernet},
			{Name: "Ethernet 5", Kind: network.KindEthernet},
			{Name: "Wi-Fi", Kind: network.KindWiFi},
		},
		WiFi: map[string][]string{"Wi-Fi": {"ATELIER-5G", "CHANTIER-MOB"}},
	}
}

func TestApplyStaticProfile(t *testing.T) {
	fake := newFake()
	p := profile.Profile{
		ID:   "site-client-b",
		Name: "Site client B",
		Targets: []profile.Target{{
			Interface: "Ethernet",
			Mode:      profile.ModeStatic,
			Address:   "10.10.128.20",
			Mask:      "255.255.255.0",
			Gateway:   "10.10.128.1",
			DNS:       []string{"10.10.128.1"},
		}},
	}

	if err := New(fake).Apply(p); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	want := []network.Call{{
		Op:        "static",
		Interface: "Ethernet",
		Static: network.StaticConfig{
			Address: "10.10.128.20",
			Mask:    "255.255.255.0",
			Gateway: "10.10.128.1",
			DNS:     []string{"10.10.128.1"},
		},
	}}
	if !reflect.DeepEqual(fake.Calls, want) {
		t.Fatalf("unexpected calls\ngot:  %+v\nwant: %+v", fake.Calls, want)
	}
}

func TestApplyDHCPProfileOnSeveralInterfaces(t *testing.T) {
	fake := newFake()
	p := profile.Profile{
		ID:   "defaut-dhcp",
		Name: "Défaut — DHCP",
		Targets: []profile.Target{
			{Interface: "Ethernet", Mode: profile.ModeDHCP},
			{Interface: "Wi-Fi", Mode: profile.ModeDHCP},
		},
	}

	if err := New(fake).Apply(p); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	want := []network.Call{
		{Op: "dhcp", Interface: "Ethernet"},
		{Op: "dhcp", Interface: "Wi-Fi"},
	}
	if !reflect.DeepEqual(fake.Calls, want) {
		t.Fatalf("unexpected calls\ngot:  %+v\nwant: %+v", fake.Calls, want)
	}
}

// Joining a network makes Windows reconfigure the adapter, so an address set
// before the connection would be wiped by it.
func TestApplyConnectsWiFiBeforeSettingTheAddress(t *testing.T) {
	fake := newFake()
	p := profile.Profile{
		ID:   "chantier",
		Name: "Chantier",
		Targets: []profile.Target{{
			Interface: "Wi-Fi",
			Mode:      profile.ModeStatic,
			SSID:      "ATELIER-5G",
			Address:   "172.16.32.8",
			Mask:      "255.255.0.0",
		}},
	}

	if err := New(fake).Apply(p); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	if len(fake.Calls) != 2 {
		t.Fatalf("expected 2 calls, got %d: %+v", len(fake.Calls), fake.Calls)
	}
	if fake.Calls[0].Op != "wifi" || fake.Calls[1].Op != "static" {
		t.Fatalf("expected the Wi-Fi connection first, got: %+v", fake.Calls)
	}
}

func TestApplyWiFiWithoutSSIDLeavesTheCurrentNetwork(t *testing.T) {
	fake := newFake()
	p := profile.Profile{
		ID:   "meme-wifi-autre-ip",
		Name: "Même Wi-Fi, autre IP",
		Targets: []profile.Target{{
			Interface: "Wi-Fi",
			Mode:      profile.ModeStatic,
			Address:   "192.168.1.50",
			Mask:      "255.255.255.0",
		}},
	}

	if err := New(fake).Apply(p); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	if len(fake.Calls) != 1 || fake.Calls[0].Op != "static" {
		t.Fatalf("expected the address change alone, got: %+v", fake.Calls)
	}
}

// Profiles travel between machines, where adapter names differ.
func TestApplyReportsAMissingInterface(t *testing.T) {
	fake := newFake()
	p := profile.Profile{
		ID:      "autre-poste",
		Name:    "Autre poste",
		Targets: []profile.Target{{Interface: "Ethernet 9", Mode: profile.ModeDHCP}},
	}

	err := New(fake).Apply(p)

	if err == nil {
		t.Fatal("expected an error on an interface this machine does not have")
	}
	if !strings.Contains(err.Error(), "Ethernet 9") {
		t.Fatalf("the error must name the missing interface, got: %v", err)
	}
	if len(fake.Calls) != 0 {
		t.Fatalf("nothing must be applied, got: %+v", fake.Calls)
	}
}

// A "back to DHCP" profile must still restore the second adapter when the
// first one refuses.
func TestApplyContinuesAfterAFailingTarget(t *testing.T) {
	fake := newFake()
	fake.FailOn = "Ethernet"
	p := profile.Profile{
		ID:   "defaut-dhcp",
		Name: "Défaut — DHCP",
		Targets: []profile.Target{
			{Interface: "Ethernet", Mode: profile.ModeDHCP},
			{Interface: "Wi-Fi", Mode: profile.ModeDHCP},
		},
	}

	err := New(fake).Apply(p)

	if err == nil {
		t.Fatal("expected the failing target to be reported")
	}
	want := []network.Call{{Op: "dhcp", Interface: "Wi-Fi"}}
	if !reflect.DeepEqual(fake.Calls, want) {
		t.Fatalf("the second target must still be applied\ngot:  %+v\nwant: %+v", fake.Calls, want)
	}
}

func TestApplyTargetAppliesAOneOffChange(t *testing.T) {
	fake := newFake()
	target := profile.Target{
		Interface: "Ethernet",
		Mode:      profile.ModeStatic,
		Address:   "192.168.1.50",
		Mask:      "255.255.255.0",
		Gateway:   "192.168.1.1",
	}

	if err := New(fake).ApplyTarget(target); err != nil {
		t.Fatalf("ApplyTarget: %v", err)
	}

	want := []network.Call{{
		Op:        "static",
		Interface: "Ethernet",
		Static:    network.StaticConfig{Address: "192.168.1.50", Mask: "255.255.255.0", Gateway: "192.168.1.1"},
	}}
	if !reflect.DeepEqual(fake.Calls, want) {
		t.Fatalf("unexpected calls\ngot:  %+v\nwant: %+v", fake.Calls, want)
	}
}

func TestApplyTargetReportsAMissingInterface(t *testing.T) {
	fake := newFake()

	err := New(fake).ApplyTarget(profile.Target{Interface: "Ethernet 9", Mode: profile.ModeDHCP})

	if err == nil {
		t.Fatal("expected an error on an interface this machine does not have")
	}
	if len(fake.Calls) != 0 {
		t.Fatalf("nothing must be applied, got: %+v", fake.Calls)
	}
}

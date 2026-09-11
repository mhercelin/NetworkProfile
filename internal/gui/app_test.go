package gui

import (
	"path/filepath"
	"strings"
	"testing"

	"networkprofile/internal/network"
	"networkprofile/internal/profile"
)

func newApp(t *testing.T) (*App, *network.Fake) {
	t.Helper()

	fake := &network.Fake{
		Adapters: []network.Interface{
			{Name: "Ethernet", Kind: network.KindEthernet},
			{Name: "Wi-Fi", Kind: network.KindWiFi},
		},
		WiFi: map[string][]string{"Wi-Fi": {"ATELIER-5G"}},
	}
	store := profile.NewStore(filepath.Join(t.TempDir(), "profiles.yaml"))
	return New(store, fake), fake
}

func sampleProfile() profile.Profile {
	return profile.Profile{
		ID:   "site-client-b",
		Name: "Site client B",
		Targets: []profile.Target{{
			Interface: "Ethernet",
			Mode:      profile.ModeStatic,
			Address:   "10.10.128.20",
			Mask:      "255.255.255.0",
			Gateway:   "10.10.128.1",
		}},
	}
}

func TestProfilesIsEmptyOnAFreshInstall(t *testing.T) {
	app, _ := newApp(t)

	profiles, err := app.Profiles()

	if err != nil {
		t.Fatalf("Profiles: %v", err)
	}
	if len(profiles) != 0 {
		t.Fatalf("expected no profile, got %d", len(profiles))
	}
}

func TestSaveProfileThenReadItBack(t *testing.T) {
	app, _ := newApp(t)

	if err := app.SaveProfile(sampleProfile()); err != nil {
		t.Fatalf("SaveProfile: %v", err)
	}

	profiles, err := app.Profiles()
	if err != nil {
		t.Fatalf("Profiles: %v", err)
	}
	if len(profiles) != 1 || profiles[0].ID != "site-client-b" {
		t.Fatalf("expected the saved profile, got: %+v", profiles)
	}
}

func TestSaveProfileReplacesTheOneCarryingTheSameID(t *testing.T) {
	app, _ := newApp(t)
	if err := app.SaveProfile(sampleProfile()); err != nil {
		t.Fatalf("SaveProfile: %v", err)
	}

	renamed := sampleProfile()
	renamed.Name = "Site client B — étage 2"
	if err := app.SaveProfile(renamed); err != nil {
		t.Fatalf("SaveProfile: %v", err)
	}

	profiles, err := app.Profiles()
	if err != nil {
		t.Fatalf("Profiles: %v", err)
	}
	if len(profiles) != 1 {
		t.Fatalf("expected a single profile, got %d", len(profiles))
	}
	if profiles[0].Name != "Site client B — étage 2" {
		t.Fatalf("expected the profile to be replaced, got: %+v", profiles[0])
	}
}

func TestSaveProfileRejectsAnInvalidProfile(t *testing.T) {
	app, _ := newApp(t)
	p := sampleProfile()
	p.Targets[0].Gateway = "192.168.1.1" // outside the address subnet

	if err := app.SaveProfile(p); err == nil {
		t.Fatal("SaveProfile must reject an invalid profile")
	}

	profiles, _ := app.Profiles()
	if len(profiles) != 0 {
		t.Fatalf("nothing must be stored, got: %+v", profiles)
	}
}

func TestDeleteProfile(t *testing.T) {
	app, _ := newApp(t)
	if err := app.SaveProfile(sampleProfile()); err != nil {
		t.Fatalf("SaveProfile: %v", err)
	}

	if err := app.DeleteProfile("site-client-b"); err != nil {
		t.Fatalf("DeleteProfile: %v", err)
	}

	profiles, _ := app.Profiles()
	if len(profiles) != 0 {
		t.Fatalf("expected no profile left, got: %+v", profiles)
	}
}

func TestDeleteProfileReportsAnUnknownID(t *testing.T) {
	app, _ := newApp(t)

	err := app.DeleteProfile("jamais-cree")

	if err == nil {
		t.Fatal("expected an error on an unknown id")
	}
	if !strings.Contains(err.Error(), "jamais-cree") {
		t.Fatalf("the error must name the id, got: %v", err)
	}
}

func TestApplyProfile(t *testing.T) {
	app, fake := newApp(t)
	if err := app.SaveProfile(sampleProfile()); err != nil {
		t.Fatalf("SaveProfile: %v", err)
	}

	if err := app.ApplyProfile("site-client-b"); err != nil {
		t.Fatalf("ApplyProfile: %v", err)
	}

	if len(fake.Calls) != 1 || fake.Calls[0].Op != "static" || fake.Calls[0].Interface != "Ethernet" {
		t.Fatalf("unexpected calls: %+v", fake.Calls)
	}
}

func TestApplyProfileReportsAnUnknownID(t *testing.T) {
	app, fake := newApp(t)

	if err := app.ApplyProfile("jamais-cree"); err == nil {
		t.Fatal("expected an error on an unknown id")
	}
	if len(fake.Calls) != 0 {
		t.Fatalf("nothing must be applied, got: %+v", fake.Calls)
	}
}

// The quick change screen skips the store entirely, so validation has to happen
// here rather than on the way to the file.
func TestApplyTargetRejectsAnInvalidConfiguration(t *testing.T) {
	app, fake := newApp(t)

	err := app.ApplyTarget(profile.Target{
		Interface: "Ethernet",
		Mode:      profile.ModeStatic,
		Address:   "192.168.1.300",
		Mask:      "255.255.255.0",
	})

	if err == nil {
		t.Fatal("expected an invalid address to be refused")
	}
	if len(fake.Calls) != 0 {
		t.Fatalf("nothing must reach the adapters, got: %+v", fake.Calls)
	}
}

func TestApplyTargetAppliesAValidConfiguration(t *testing.T) {
	app, fake := newApp(t)

	err := app.ApplyTarget(profile.Target{
		Interface: "Wi-Fi",
		Mode:      profile.ModeDHCP,
	})

	if err != nil {
		t.Fatalf("ApplyTarget: %v", err)
	}
	if len(fake.Calls) != 1 || fake.Calls[0].Op != "dhcp" {
		t.Fatalf("unexpected calls: %+v", fake.Calls)
	}
}

func TestKnownWiFiNetworks(t *testing.T) {
	app, _ := newApp(t)

	networks, err := app.KnownWiFiNetworks("Wi-Fi")

	if err != nil {
		t.Fatalf("KnownWiFiNetworks: %v", err)
	}
	if len(networks) != 1 || networks[0] != "ATELIER-5G" {
		t.Fatalf("unexpected networks: %v", networks)
	}
}

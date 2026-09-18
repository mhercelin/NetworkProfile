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
	return New(store, fake, nil), fake
}

// newAppCountingChanges reports how many times the change hook fired, which is
// what keeps the notification area in step with the stored profiles.
func newAppCountingChanges(t *testing.T, changes *int) *App {
	t.Helper()

	store := profile.NewStore(filepath.Join(t.TempDir(), "profiles.yaml"))
	return New(store, &network.Fake{
		Adapters: []network.Interface{{Name: "Ethernet", Kind: network.KindEthernet}},
	}, func() { *changes++ })
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

// The status line reads this, and a profile applied from the notification area
// never passes through the window — so it cannot be the window that remembers.
func TestLastAppliedRecordsWhicheverSurfaceAsked(t *testing.T) {
	app, _ := newApp(t)
	if err := app.SaveProfile(sampleProfile()); err != nil {
		t.Fatalf("SaveProfile: %v", err)
	}

	if before := app.LastApplied(); before.Name != "" {
		t.Fatalf("nothing applied yet, got %+v", before)
	}

	if err := app.ApplyProfile("site-client-b"); err != nil {
		t.Fatalf("ApplyProfile: %v", err)
	}

	applied := app.LastApplied()
	if applied.ID != "site-client-b" || applied.Name != "Site client B" {
		t.Fatalf("unexpected record: %+v", applied)
	}
	if applied.At == "" {
		t.Fatal("the time of the change is what the status line shows")
	}
}

func TestLastAppliedIgnoresAFailedApplication(t *testing.T) {
	app, fake := newApp(t)
	if err := app.SaveProfile(sampleProfile()); err != nil {
		t.Fatalf("SaveProfile: %v", err)
	}
	fake.FailOn = "Ethernet"

	if err := app.ApplyProfile("site-client-b"); err == nil {
		t.Fatal("expected the application to fail")
	}

	if applied := app.LastApplied(); applied.Name != "" {
		t.Fatalf("a refused change must not be reported as applied: %+v", applied)
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

func TestSetPinned(t *testing.T) {
	app, _ := newApp(t)
	if err := app.SaveProfile(sampleProfile()); err != nil {
		t.Fatalf("SaveProfile: %v", err)
	}

	if err := app.SetPinned("site-client-b", true); err != nil {
		t.Fatalf("SetPinned: %v", err)
	}

	profiles, _ := app.Profiles()
	if len(profiles) != 1 || !profiles[0].Pinned {
		t.Fatalf("expected the profile to be pinned, got: %+v", profiles)
	}

	if err := app.SetPinned("site-client-b", false); err != nil {
		t.Fatalf("SetPinned: %v", err)
	}
	profiles, _ = app.Profiles()
	if profiles[0].Pinned {
		t.Fatal("expected the profile to be unpinned")
	}
}

func TestSetPinnedReportsAnUnknownID(t *testing.T) {
	app, _ := newApp(t)

	if err := app.SetPinned("jamais-cree", true); err == nil {
		t.Fatal("expected an error on an unknown id")
	}
}

// Anything that writes the profile file has to tell the notification area, or
// its menu drifts away from what is stored.
func TestEveryWriteAnnouncesItself(t *testing.T) {
	changes := 0
	app := newAppCountingChanges(t, &changes)

	if err := app.SaveProfile(sampleProfile()); err != nil {
		t.Fatalf("SaveProfile: %v", err)
	}
	if changes != 1 {
		t.Fatalf("saving must announce itself, got %d", changes)
	}

	if err := app.SetPinned("site-client-b", true); err != nil {
		t.Fatalf("SetPinned: %v", err)
	}
	if changes != 2 {
		t.Fatalf("pinning must announce itself, got %d", changes)
	}

	if err := app.DeleteProfile("site-client-b"); err != nil {
		t.Fatalf("DeleteProfile: %v", err)
	}
	if changes != 3 {
		t.Fatalf("deleting must announce itself, got %d", changes)
	}
}

func TestAFailedWriteAnnouncesNothing(t *testing.T) {
	changes := 0
	app := newAppCountingChanges(t, &changes)

	invalid := sampleProfile()
	invalid.Targets[0].Address = "192.168.1.300"
	if err := app.SaveProfile(invalid); err == nil {
		t.Fatal("expected the invalid profile to be refused")
	}

	if changes != 0 {
		t.Fatalf("a refused write must announce nothing, got %d", changes)
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

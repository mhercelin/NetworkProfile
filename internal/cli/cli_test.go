package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"networkprofile/internal/profile"
)

func sampleProfiles() []profile.Profile {
	return []profile.Profile{
		{
			ID:   "ethernet-atelier",
			Name: "Ethernet — Atelier",
			Targets: []profile.Target{{
				Interface: "Ethernet", Mode: profile.ModeStatic,
				Address: "192.168.1.50", Mask: "255.255.255.0",
			}},
		},
		{
			ID:   "defaut-dhcp",
			Name: "Défaut — DHCP",
			Targets: []profile.Target{
				{Interface: "Ethernet", Mode: profile.ModeDHCP},
				{Interface: "Wi-Fi", Mode: profile.ModeDHCP},
			},
		},
	}
}

type run struct {
	handled bool
	code    int
	out     string
	err     string
	applied []string
}

func exec(t *testing.T, args []string, profiles []profile.Profile, applyErr error) run {
	t.Helper()

	var out, errOut bytes.Buffer
	result := run{}

	handled, code := Run(args, Env{
		Profiles: func() ([]profile.Profile, error) { return profiles, nil },
		Apply: func(id string) error {
			result.applied = append(result.applied, id)
			return applyErr
		},
		Out: &out,
		Err: &errOut,
	})

	result.handled = handled
	result.code = code
	result.out = out.String()
	result.err = errOut.String()
	return result
}

// No arguments means the window, and nothing must be printed to a console that
// is not there.
func TestNoArgumentsFallsThroughToTheWindow(t *testing.T) {
	got := exec(t, nil, sampleProfiles(), nil)

	if got.handled {
		t.Fatal("an empty command line must not be handled here")
	}
	if got.out != "" || got.err != "" {
		t.Fatalf("nothing must be written: out=%q err=%q", got.out, got.err)
	}
}

func TestList(t *testing.T) {
	got := exec(t, []string{"--list"}, sampleProfiles(), nil)

	if !got.handled || got.code != 0 {
		t.Fatalf("handled=%t code=%d", got.handled, got.code)
	}
	for _, want := range []string{"ethernet-atelier", "Ethernet — Atelier", "192.168.1.50", "defaut-dhcp", "DHCP"} {
		if !strings.Contains(got.out, want) {
			t.Fatalf("listing is missing %q:\n%s", want, got.out)
		}
	}
}

func TestListWithNoProfiles(t *testing.T) {
	got := exec(t, []string{"--list"}, nil, nil)

	if got.code != 0 {
		t.Fatalf("an empty list is not a failure, got code %d", got.code)
	}
	if !strings.Contains(got.out, "Aucun profil") {
		t.Fatalf("unexpected output: %q", got.out)
	}
}

func TestApplyByID(t *testing.T) {
	got := exec(t, []string{"--profile", "ethernet-atelier"}, sampleProfiles(), nil)

	if got.code != 0 {
		t.Fatalf("code %d, stderr: %s", got.code, got.err)
	}
	if len(got.applied) != 1 || got.applied[0] != "ethernet-atelier" {
		t.Fatalf("unexpected applications: %v", got.applied)
	}
}

// A name typed at a prompt will not carry the accents and capitals of the
// window, so matching ignores case.
func TestApplyByDisplayedNameIgnoringCase(t *testing.T) {
	got := exec(t, []string{"--profile", "ethernet — ATELIER"}, sampleProfiles(), nil)

	if got.code != 0 {
		t.Fatalf("code %d, stderr: %s", got.code, got.err)
	}
	if len(got.applied) != 1 || got.applied[0] != "ethernet-atelier" {
		t.Fatalf("unexpected applications: %v", got.applied)
	}
}

func TestApplyReportsAnUnknownProfile(t *testing.T) {
	got := exec(t, []string{"--profile", "jamais-cree"}, sampleProfiles(), nil)

	if got.code == 0 {
		t.Fatal("an unknown profile must fail")
	}
	if len(got.applied) != 0 {
		t.Fatalf("nothing must be applied: %v", got.applied)
	}
	if !strings.Contains(got.err, "jamais-cree") {
		t.Fatalf("the message must name what was asked for: %q", got.err)
	}
}

// Applying the wrong profile puts the machine on the wrong network, so an
// ambiguous name is refused rather than resolved arbitrarily.
func TestApplyRefusesAnAmbiguousName(t *testing.T) {
	profiles := []profile.Profile{
		{ID: "site-a", Name: "Site", Targets: []profile.Target{{Interface: "Ethernet", Mode: profile.ModeDHCP}}},
		{ID: "site-b", Name: "SITE", Targets: []profile.Target{{Interface: "Wi-Fi", Mode: profile.ModeDHCP}}},
	}

	got := exec(t, []string{"--profile", "site"}, profiles, nil)

	if got.code == 0 {
		t.Fatal("an ambiguous name must fail")
	}
	if len(got.applied) != 0 {
		t.Fatalf("nothing must be applied: %v", got.applied)
	}
	for _, want := range []string{"site-a", "site-b"} {
		if !strings.Contains(got.err, want) {
			t.Fatalf("the message must list the candidates, got: %q", got.err)
		}
	}
}

// An exact identifier wins even when it also matches a name.
func TestApplyPrefersAnExactIdentifier(t *testing.T) {
	profiles := []profile.Profile{
		{ID: "atelier", Name: "Bureau", Targets: []profile.Target{{Interface: "Ethernet", Mode: profile.ModeDHCP}}},
		{ID: "bureau", Name: "Atelier", Targets: []profile.Target{{Interface: "Wi-Fi", Mode: profile.ModeDHCP}}},
	}

	got := exec(t, []string{"--profile", "atelier"}, profiles, nil)

	if got.code != 0 {
		t.Fatalf("code %d, stderr: %s", got.code, got.err)
	}
	if len(got.applied) != 1 || got.applied[0] != "atelier" {
		t.Fatalf("expected the identifier to win, got: %v", got.applied)
	}
}

func TestApplyReportsAFailure(t *testing.T) {
	got := exec(t, []string{"--profile", "defaut-dhcp"}, sampleProfiles(), errors.New("élévation refusée"))

	if got.code == 0 {
		t.Fatal("a refused application must fail")
	}
	if !strings.Contains(got.err, "élévation refusée") {
		t.Fatalf("the cause must reach the console: %q", got.err)
	}
}

func TestProfileWithoutAName(t *testing.T) {
	got := exec(t, []string{"--profile"}, sampleProfiles(), nil)

	if got.code == 0 {
		t.Fatal("a missing argument must fail")
	}
	if len(got.applied) != 0 {
		t.Fatalf("nothing must be applied: %v", got.applied)
	}
}

func TestUnknownArgument(t *testing.T) {
	got := exec(t, []string{"--reboot"}, sampleProfiles(), nil)

	if !got.handled || got.code == 0 {
		t.Fatalf("an unknown argument must be handled and fail: handled=%t code=%d", got.handled, got.code)
	}
	if !strings.Contains(got.err, "--reboot") {
		t.Fatalf("the message must name the argument: %q", got.err)
	}
}

func TestHelp(t *testing.T) {
	got := exec(t, []string{"--help"}, sampleProfiles(), nil)

	if got.code != 0 {
		t.Fatalf("help is not a failure, got code %d", got.code)
	}
	if !strings.Contains(got.out, "--profile") {
		t.Fatalf("usage is missing the commands: %q", got.out)
	}
}

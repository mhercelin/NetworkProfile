package profile

import (
	"errors"
	"testing"
)

// validProfile is the reference each case mutates, so a test states exactly
// what it changed and nothing else.
func validProfile() Profile {
	return Profile{
		ID:   "site-client-b",
		Name: "Site client B",
		Targets: []Target{{
			Interface: "Ethernet",
			Mode:      ModeStatic,
			Address:   "10.10.128.20",
			Mask:      "255.255.255.0",
			Gateway:   "10.10.128.1",
			DNS:       []string{"10.10.128.1", "9.9.9.9"},
		}},
	}
}

func hasFieldError(t *testing.T, err error, field string) bool {
	t.Helper()
	var ve *ValidationError
	if !errors.As(err, &ve) {
		return false
	}
	for _, fe := range ve.Errors {
		if fe.Field == field {
			return true
		}
	}
	return false
}

func TestProfileValidate(t *testing.T) {
	tests := []struct {
		name string
		// mutate adjusts the reference profile; wantField is the field path
		// validation must flag, or "" when the profile must be accepted.
		mutate    func(p *Profile)
		wantField string
	}{
		{"complete static profile", func(*Profile) {}, ""},
		{"empty id", func(p *Profile) { p.ID = "" }, "id"},
		{"id with capitals", func(p *Profile) { p.ID = "Site-Client-B" }, "id"},
		{"id with spaces", func(p *Profile) { p.ID = "site client b" }, "id"},
		{"blank name", func(p *Profile) { p.Name = "  " }, "name"},
		{"no target", func(p *Profile) { p.Targets = nil }, "targets"},
		{"empty interface", func(p *Profile) { p.Targets[0].Interface = "" }, "targets[0].interface"},
		{"unknown mode", func(p *Profile) { p.Targets[0].Mode = "auto" }, "targets[0].mode"},
		{"missing address", func(p *Profile) { p.Targets[0].Address = "" }, "targets[0].address"},
		{"malformed address", func(p *Profile) { p.Targets[0].Address = "192.168.1.300" }, "targets[0].address"},
		{"IPv6 address rejected", func(p *Profile) { p.Targets[0].Address = "2001:db8::1" }, "targets[0].address"},
		{"missing mask", func(p *Profile) { p.Targets[0].Mask = "" }, "targets[0].mask"},
		{"non-contiguous mask", func(p *Profile) { p.Targets[0].Mask = "255.0.255.0" }, "targets[0].mask"},
		{"zero mask", func(p *Profile) { p.Targets[0].Mask = "0.0.0.0" }, "targets[0].mask"},
		{"CIDR notation rejected as mask", func(p *Profile) { p.Targets[0].Mask = "24" }, "targets[0].mask"},
		{"malformed gateway", func(p *Profile) { p.Targets[0].Gateway = "10.10.128" }, "targets[0].gateway"},
		{"gateway outside the subnet", func(p *Profile) { p.Targets[0].Gateway = "192.168.1.1" }, "targets[0].gateway"},
		{"gateway is optional", func(p *Profile) { p.Targets[0].Gateway = "" }, ""},
		{"malformed DNS server", func(p *Profile) { p.Targets[0].DNS = []string{"10.10.128.1", "notanip"} }, "targets[0].dns[1]"},
		{"DNS servers are optional", func(p *Profile) { p.Targets[0].DNS = nil }, ""},
		{"/16 mask accepted", func(p *Profile) {
			p.Targets[0].Address = "172.16.4.10"
			p.Targets[0].Mask = "255.255.0.0"
			p.Targets[0].Gateway = "172.16.0.1"
			p.Targets[0].DNS = nil
		}, ""},
		{"bare DHCP target", func(p *Profile) {
			p.Targets[0] = Target{Interface: "Ethernet", Mode: ModeDHCP}
		}, ""},
		{"DHCP target carrying an address", func(p *Profile) {
			p.Targets[0] = Target{Interface: "Ethernet", Mode: ModeDHCP, Address: "10.10.128.20"}
		}, "targets[0].mode"},
		{"Wi-Fi target with an SSID", func(p *Profile) {
			p.Targets[0].Interface = "Wi-Fi"
			p.Targets[0].SSID = "SiteB-Corp"
		}, ""},
		{"same interface targeted twice", func(p *Profile) {
			p.Targets = append(p.Targets, Target{Interface: "Ethernet", Mode: ModeDHCP})
		}, "targets[1].interface"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := validProfile()
			tt.mutate(&p)

			err := p.Validate()

			if tt.wantField == "" {
				if err != nil {
					t.Fatalf("expected a valid profile, got error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected %q to be flagged, got no error", tt.wantField)
			}
			if !hasFieldError(t, err, tt.wantField) {
				t.Fatalf("expected an error on %q, got: %v", tt.wantField, err)
			}
		})
	}
}

// The "Défaut" profile puts several adapters back on DHCP in one go: this is
// the case that justifies a profile holding a list of targets rather than one.
func TestProfileWithDHCPTargetsOnSeveralInterfaces(t *testing.T) {
	p := Profile{
		ID:   "defaut-dhcp",
		Name: "Défaut — DHCP",
		Targets: []Target{
			{Interface: "Ethernet", Mode: ModeDHCP},
			{Interface: "Wi-Fi", Mode: ModeDHCP},
		},
	}

	if err := p.Validate(); err != nil {
		t.Fatalf("expected a multi-interface DHCP profile to be valid, got: %v", err)
	}
}

func TestValidationErrorReportsEveryProblem(t *testing.T) {
	p := validProfile()
	p.ID = ""
	p.Name = ""
	p.Targets[0].Address = "nonsense"

	err := p.Validate()

	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected a *ValidationError, got %T", err)
	}
	if len(ve.Errors) != 3 {
		t.Fatalf("expected 3 errors, got %d: %v", len(ve.Errors), ve.Errors)
	}
}

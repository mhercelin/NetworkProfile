package profile

import (
	"fmt"
	"math/bits"
	"net/netip"
	"regexp"
	"strings"
)

// Mode is how an interface gets its address.
type Mode string

const (
	ModeDHCP   Mode = "dhcp"
	ModeStatic Mode = "static"
)

// Target is the configuration applied to one network interface. A Wi-Fi target
// may carry an SSID, naming a network Windows already holds credentials for;
// leaving it empty keeps the interface on its current network.
type Target struct {
	Interface string   `yaml:"interface"`
	Mode      Mode     `yaml:"mode"`
	SSID      string   `yaml:"ssid,omitempty"`
	Address   string   `yaml:"address,omitempty"`
	Mask      string   `yaml:"mask,omitempty"`
	Gateway   string   `yaml:"gateway,omitempty"`
	DNS       []string `yaml:"dns,omitempty"`
}

// Profile is a named set of targets applied together.
type Profile struct {
	ID      string   `yaml:"id"`
	Name    string   `yaml:"name"`
	Targets []Target `yaml:"targets"`
}

// FieldError locates a problem at a field path such as "targets[1].gateway".
type FieldError struct {
	Field   string
	Message string
}

// ValidationError carries every problem found, so a form can show them at once.
type ValidationError struct {
	Errors []FieldError
}

func (e *ValidationError) Error() string {
	parts := make([]string, len(e.Errors))
	for i, fe := range e.Errors {
		parts[i] = fe.Field + " : " + fe.Message
	}
	return strings.Join(parts, " ; ")
}

var idPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func (p Profile) Validate() error {
	var errs []FieldError

	if !idPattern.MatchString(p.ID) {
		errs = append(errs, FieldError{"id", "identifiant vide ou invalide (minuscules, chiffres et tirets)"})
	}
	if strings.TrimSpace(p.Name) == "" {
		errs = append(errs, FieldError{"name", "le nom est obligatoire"})
	}
	if len(p.Targets) == 0 {
		errs = append(errs, FieldError{"targets", "au moins une interface cible est requise"})
	}

	seen := make(map[string]bool, len(p.Targets))
	for i, t := range p.Targets {
		path := fmt.Sprintf("targets[%d]", i)
		if t.Interface != "" {
			if seen[t.Interface] {
				errs = append(errs, FieldError{path + ".interface", "interface déjà ciblée par ce profil"})
			}
			seen[t.Interface] = true
		}
		errs = append(errs, t.validate(path)...)
	}

	if len(errs) > 0 {
		return &ValidationError{Errors: errs}
	}
	return nil
}

func (t Target) validate(path string) []FieldError {
	var errs []FieldError

	if strings.TrimSpace(t.Interface) == "" {
		errs = append(errs, FieldError{path + ".interface", "l'interface est obligatoire"})
	}

	switch t.Mode {
	case ModeDHCP:
		if t.Address != "" || t.Mask != "" || t.Gateway != "" || len(t.DNS) > 0 {
			errs = append(errs, FieldError{path + ".mode", "un profil DHCP ne porte ni adresse, ni masque, ni passerelle, ni DNS"})
		}
	case ModeStatic:
		errs = append(errs, t.validateStatic(path)...)
	default:
		errs = append(errs, FieldError{path + ".mode", `mode inconnu (attendu "static" ou "dhcp")`})
	}

	return errs
}

func (t Target) validateStatic(path string) []FieldError {
	var errs []FieldError

	addr, addrOK := parseIPv4(t.Address)
	if !addrOK {
		errs = append(errs, FieldError{path + ".address", "adresse IPv4 invalide"})
	}

	prefixLen, maskOK := maskPrefixLen(t.Mask)
	if !maskOK {
		errs = append(errs, FieldError{path + ".mask", "masque de sous-réseau IPv4 invalide"})
	}

	if t.Gateway != "" {
		gw, gwOK := parseIPv4(t.Gateway)
		switch {
		case !gwOK:
			errs = append(errs, FieldError{path + ".gateway", "passerelle IPv4 invalide"})
		case addrOK && maskOK && !sameSubnet(addr, gw, prefixLen):
			errs = append(errs, FieldError{path + ".gateway", "la passerelle n'appartient pas au sous-réseau de l'adresse"})
		}
	}

	for i, d := range t.DNS {
		if _, ok := parseIPv4(d); !ok {
			errs = append(errs, FieldError{fmt.Sprintf("%s.dns[%d]", path, i), "serveur DNS IPv4 invalide"})
		}
	}

	return errs
}

func parseIPv4(s string) (netip.Addr, bool) {
	a, err := netip.ParseAddr(s)
	if err != nil || !a.Is4() {
		return netip.Addr{}, false
	}
	return a, true
}

// maskPrefixLen converts a dotted-quad mask to a prefix length. Masks whose
// 1-bits are not contiguous (255.0.255.0) are rejected: Windows accepts them
// on some paths and routes unpredictably afterwards.
func maskPrefixLen(s string) (int, bool) {
	a, ok := parseIPv4(s)
	if !ok {
		return 0, false
	}
	b := a.As4()
	v := uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
	inv := ^v
	if inv&(inv+1) != 0 {
		return 0, false
	}
	ones := bits.OnesCount32(v)
	if ones == 0 {
		return 0, false
	}
	return ones, true
}

func sameSubnet(a, b netip.Addr, prefixLen int) bool {
	p, err := a.Prefix(prefixLen)
	if err != nil {
		return false
	}
	return p.Contains(b)
}

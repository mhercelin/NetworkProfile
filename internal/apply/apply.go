package apply

import (
	"errors"
	"fmt"
	"log"

	"networkprofile/internal/network"
	"networkprofile/internal/profile"
)

// Service applies profiles to the machine's adapters.
type Service struct {
	manager network.Manager
}

func New(manager network.Manager) *Service {
	return &Service{manager: manager}
}

// Apply applies every target of a profile. A target that fails does not stop
// the others: a "back to DHCP" profile covering two adapters must still restore
// the second one when the first refuses.
func (s *Service) Apply(p profile.Profile) error {
	present, err := s.presentInterfaces()
	if err != nil {
		return err
	}

	var errs []error
	for _, target := range p.Targets {
		if !present[target.Interface] {
			// Profiles are meant to be carried between machines, where adapter
			// names differ; say so plainly instead of letting netsh fail.
			errs = append(errs, fmt.Errorf("interface %q absente de cette machine", target.Interface))
			continue
		}
		errs = append(errs, s.applyTarget(target))
	}

	return errors.Join(errs...)
}

// ApplyTarget applies a single interface configuration, which is what the quick
// change screen does without saving a profile.
func (s *Service) ApplyTarget(target profile.Target) error {
	present, err := s.presentInterfaces()
	if err != nil {
		return err
	}
	if !present[target.Interface] {
		return fmt.Errorf("interface %q absente de cette machine", target.Interface)
	}
	return s.applyTarget(target)
}

func (s *Service) applyTarget(target profile.Target) error {
	log.Printf("application : %s mode=%s ssid=%q adresse=%q", target.Interface, target.Mode, target.SSID, target.Address)

	if target.SSID != "" {
		// Joining a network makes Windows reconfigure that adapter, so the
		// address has to be set after the connection, never before.
		if err := s.manager.ConnectWiFi(target.Interface, target.SSID); err != nil {
			return fmt.Errorf("%s : connexion au réseau %q : %w", target.Interface, target.SSID, err)
		}
	}

	var err error
	switch target.Mode {
	case profile.ModeDHCP:
		err = s.manager.SetDHCP(target.Interface)
	case profile.ModeStatic:
		err = s.manager.SetStatic(target.Interface, network.StaticConfig{
			Address: target.Address,
			Mask:    target.Mask,
			Gateway: target.Gateway,
			DNS:     target.DNS,
		})
	default:
		return fmt.Errorf("%s : mode %q inconnu", target.Interface, target.Mode)
	}

	if err != nil {
		return fmt.Errorf("%s : %w", target.Interface, err)
	}
	return nil
}

func (s *Service) presentInterfaces() (map[string]bool, error) {
	list, err := s.manager.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("lecture des cartes réseau : %w", err)
	}

	present := make(map[string]bool, len(list))
	for _, iface := range list {
		present[iface.Name] = true
	}
	return present, nil
}

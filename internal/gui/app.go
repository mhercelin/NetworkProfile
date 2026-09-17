package gui

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"networkprofile/internal/apply"
	"networkprofile/internal/network"
	"networkprofile/internal/profile"
)

// App is the surface bound to the user interface: every method here is callable
// from the frontend. It holds no state of its own — profiles are read from the
// file on each call, so hand-editing the YAML is picked up without a restart.
type App struct {
	store   *profile.Store
	manager network.Manager
	applier *apply.Service

	// changed is how the notification area learns that the pinned set moved,
	// rather than polling the file for it. It is a constructor argument rather
	// than a setter because every exported method here is bound to the
	// frontend, and a callback is not something that survives that crossing.
	changed func()
}

func New(store *profile.Store, manager network.Manager, changed func()) *App {
	return &App{
		store:   store,
		manager: manager,
		applier: apply.New(manager),
		changed: changed,
	}
}

func (a *App) notifyChanged() {
	if a.changed != nil {
		a.changed()
	}
}

// DataDir is where this application stores profiles: %AppData%\NetworkProfile.
//
// Per-user is deliberate — two people sharing a machine are configuring it for
// different work — and it is only possible because the interface runs as the
// operator. Elevation is confined to the helper, which stores nothing.
func DataDir() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "NetworkProfile"), nil
}

func DefaultStorePath() (string, error) {
	dir, err := DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "profiles.yaml"), nil
}

func (a *App) Interfaces() ([]network.Interface, error) {
	return a.manager.Interfaces()
}

// Elevated reports whether privileges have actually been obtained. Managers
// that need none — the one used in tests — never claim otherwise.
func (a *App) Elevated() bool {
	reporter, ok := a.manager.(interface{ Elevated() bool })
	return ok && reporter.Elevated()
}

func (a *App) KnownWiFiNetworks(iface string) ([]string, error) {
	return a.manager.KnownWiFiNetworks(iface)
}

func (a *App) Profiles() ([]profile.Profile, error) {
	return a.store.Load()
}

// SaveProfile creates the profile, or replaces the one already carrying its id.
func (a *App) SaveProfile(p profile.Profile) error {
	if err := p.Validate(); err != nil {
		return err
	}

	profiles, err := a.store.Load()
	if err != nil {
		return err
	}

	replaced := false
	for i := range profiles {
		if profiles[i].ID == p.ID {
			profiles[i] = p
			replaced = true
			break
		}
	}
	if !replaced {
		profiles = append(profiles, p)
	}

	if err := a.store.Save(profiles); err != nil {
		return err
	}
	a.notifyChanged()
	return nil
}

func (a *App) DeleteProfile(id string) error {
	profiles, err := a.store.Load()
	if err != nil {
		return err
	}

	for i, p := range profiles {
		if p.ID == id {
			if err := a.store.Save(slices.Delete(profiles, i, i+1)); err != nil {
				return err
			}
			a.notifyChanged()
			return nil
		}
	}
	return fmt.Errorf("profil %q introuvable", id)
}

// SetPinned adds or removes a profile from the notification area menu.
func (a *App) SetPinned(id string, pinned bool) error {
	profiles, err := a.store.Load()
	if err != nil {
		return err
	}

	for i := range profiles {
		if profiles[i].ID == id {
			profiles[i].Pinned = pinned
			if err := a.store.Save(profiles); err != nil {
				return err
			}
			a.notifyChanged()
			return nil
		}
	}
	return fmt.Errorf("profil %q introuvable", id)
}

func (a *App) ApplyProfile(id string) error {
	profiles, err := a.store.Load()
	if err != nil {
		return err
	}

	for _, p := range profiles {
		if p.ID == id {
			err := a.applier.Apply(p)
			// Applied or not, the machine may have moved: both the window and
			// the notification area menu have to be told so neither keeps
			// showing the previous profile as the active one.
			a.notifyChanged()
			return err
		}
	}
	return fmt.Errorf("profil %q introuvable", id)
}

// ActiveProfileID reports which stored profile matches what the adapters
// currently report, or an empty string when none does.
func (a *App) ActiveProfileID() (string, error) {
	profiles, err := a.store.Load()
	if err != nil {
		return "", err
	}
	adapters, err := a.manager.Interfaces()
	if err != nil {
		return "", err
	}
	return apply.Active(profiles, adapters), nil
}

// ApplyTarget applies one interface configuration without saving it, which is
// what the quick change screen does.
func (a *App) ApplyTarget(target profile.Target) error {
	if err := target.Validate(); err != nil {
		return err
	}

	err := a.applier.ApplyTarget(target)
	// A one-off change moves the machine off whatever profile was active, which
	// the notification area menu has to reflect.
	a.notifyChanged()
	return err
}

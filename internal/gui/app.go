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
}

func New(store *profile.Store, manager network.Manager) *App {
	return &App{
		store:   store,
		manager: manager,
		applier: apply.New(manager),
	}
}

// DataDir is where everything this application writes lives.
//
// It is deliberately machine-wide rather than per-user. The application always
// runs elevated, and on a machine where the operator is not a local
// administrator, elevation switches to a different account entirely — so
// %AppData% would point at whichever administrator account happened to answer
// the prompt, and the profiles would appear to vanish from one launch to the
// next.
func DataDir() string {
	programData := os.Getenv("ProgramData")
	if programData == "" {
		programData = `C:\ProgramData`
	}
	return filepath.Join(programData, "NetworkProfile")
}

func DefaultStorePath() string {
	return filepath.Join(DataDir(), "profiles.yaml")
}

func (a *App) Interfaces() ([]network.Interface, error) {
	return a.manager.Interfaces()
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

	return a.store.Save(profiles)
}

func (a *App) DeleteProfile(id string) error {
	profiles, err := a.store.Load()
	if err != nil {
		return err
	}

	for i, p := range profiles {
		if p.ID == id {
			return a.store.Save(slices.Delete(profiles, i, i+1))
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
			return a.applier.Apply(p)
		}
	}
	return fmt.Errorf("profil %q introuvable", id)
}

// ApplyTarget applies one interface configuration without saving it, which is
// what the quick change screen does.
func (a *App) ApplyTarget(target profile.Target) error {
	if err := target.Validate(); err != nil {
		return err
	}
	return a.applier.ApplyTarget(target)
}

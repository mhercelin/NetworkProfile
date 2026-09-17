package profile

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const formatVersion = 1

type document struct {
	Version  int       `yaml:"version"`
	Profiles []Profile `yaml:"profiles"`
}

// Store reads and writes the profile file.
type Store struct {
	path string
}

func NewStore(path string) *Store {
	return &Store{path: path}
}

func (s *Store) Path() string {
	return s.path
}

// Load reads every profile. A missing file is not an error: that is what a
// first run looks like.
func (s *Store) Load() ([]Profile, error) {
	raw, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var doc document
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("%s : YAML illisible : %w", s.path, err)
	}
	if doc.Version != formatVersion {
		return nil, fmt.Errorf("%s : version de format %d non supportée (attendu %d)", s.path, doc.Version, formatVersion)
	}
	if err := validateSet(doc.Profiles); err != nil {
		return nil, fmt.Errorf("%s : %w", s.path, err)
	}

	return doc.Profiles, nil
}

// Save replaces the file with the given profiles. The write goes through a
// temporary file in the same directory: an interrupted save leaves the previous
// profiles intact rather than a half-written file.
func (s *Store) Save(profiles []Profile) error {
	if err := validateSet(profiles); err != nil {
		return err
	}

	out, err := yaml.Marshal(document{Version: formatVersion, Profiles: profiles})
	if err != nil {
		return err
	}

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, ".profiles-*.yaml")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.Write(out); err != nil {
		// Already failing; the close can add nothing to the diagnosis.
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	return os.Rename(tmpName, s.path)
}

func validateSet(profiles []Profile) error {
	seen := make(map[string]bool, len(profiles))
	for i, p := range profiles {
		if err := p.Validate(); err != nil {
			return fmt.Errorf("profil %d (%q) : %w", i, p.ID, err)
		}
		if seen[p.ID] {
			return fmt.Errorf("identifiant de profil en double : %q", p.ID)
		}
		seen[p.ID] = true
	}
	return nil
}

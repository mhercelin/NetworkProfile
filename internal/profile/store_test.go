package profile

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	return NewStore(filepath.Join(t.TempDir(), "profiles.yaml"))
}

func TestStoreRoundTrip(t *testing.T) {
	s := newTestStore(t)
	want := []Profile{
		validProfile(),
		{
			ID:   "defaut-dhcp",
			Name: "Défaut — DHCP",
			Targets: []Target{
				{Interface: "Ethernet", Mode: ModeDHCP},
				{Interface: "Wi-Fi", Mode: ModeDHCP},
			},
		},
		{
			ID:      "chantier-lyon",
			Name:    "Chantier — Lyon",
			Targets: []Target{{Interface: "Wi-Fi", Mode: ModeStatic, SSID: "CHANTIER-MOB", Address: "172.16.32.8", Mask: "255.255.0.0", Gateway: "172.16.0.1"}},
		},
	}

	if err := s.Save(want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := s.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("profiles read back differ from those written\ngot:  %+v\nwant: %+v", got, want)
	}
}

func TestStoreLoadMissingFile(t *testing.T) {
	got, err := newTestStore(t).Load()

	if err != nil {
		t.Fatalf("a first run must not be an error, got: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected an empty list, got %d profiles", len(got))
	}
}

func TestStoreLoadMalformedYAML(t *testing.T) {
	s := newTestStore(t)
	if err := os.WriteFile(s.Path(), []byte("profiles: [oops"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := s.Load(); err == nil {
		t.Fatal("expected an error on malformed YAML")
	}
}

func TestStoreLoadUnknownVersion(t *testing.T) {
	s := newTestStore(t)
	if err := os.WriteFile(s.Path(), []byte("version: 99\nprofiles: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := s.Load()

	if err == nil {
		t.Fatal("expected an error on an unknown format version")
	}
	if !strings.Contains(err.Error(), "99") {
		t.Fatalf("the error must name the version encountered, got: %v", err)
	}
}

// A hand-edited file can hold anything: better to refuse loudly than to load a
// profile that will cut the network when applied.
func TestStoreLoadInvalidProfile(t *testing.T) {
	s := newTestStore(t)
	doc := "version: 1\nprofiles:\n  - id: atelier\n    name: Atelier\n    targets:\n      - interface: Ethernet\n        mode: static\n        address: 192.168.1.999\n        mask: 255.255.255.0\n"
	if err := os.WriteFile(s.Path(), []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := s.Load(); err == nil {
		t.Fatal("expected an error on an invalid profile")
	}
}

func TestStoreSaveRejectsInvalidProfile(t *testing.T) {
	s := newTestStore(t)
	p := validProfile()
	p.Targets[0].Address = "nonsense"

	if err := s.Save([]Profile{p}); err == nil {
		t.Fatal("Save must reject an invalid profile")
	}
	if _, err := os.Stat(s.Path()); !os.IsNotExist(err) {
		t.Fatal("no file must be written when validation fails")
	}
}

func TestStoreSaveRejectsDuplicateID(t *testing.T) {
	s := newTestStore(t)
	p := validProfile()

	err := s.Save([]Profile{p, p})

	if err == nil {
		t.Fatal("Save must reject two profiles sharing an id")
	}
	if !strings.Contains(err.Error(), p.ID) {
		t.Fatalf("the error must name the offending id, got: %v", err)
	}
}

func TestStoreSaveLeavesNoTempFile(t *testing.T) {
	s := newTestStore(t)
	if err := s.Save([]Profile{validProfile()}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	entries, err := os.ReadDir(filepath.Dir(s.Path()))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected a single file in the directory, got: %v", entries)
	}
}

func TestStoreSaveReplacesPreviousContent(t *testing.T) {
	s := newTestStore(t)
	if err := s.Save([]Profile{validProfile()}); err != nil {
		t.Fatalf("first Save: %v", err)
	}

	if err := s.Save(nil); err != nil {
		t.Fatalf("second Save: %v", err)
	}
	got, err := s.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(got) != 0 {
		t.Fatalf("expected an empty list after replacement, got %d profiles", len(got))
	}
}

func makeProfiles(n int) []Profile {
	profiles := make([]Profile, n)
	for i := range profiles {
		profiles[i] = Profile{
			ID:   fmt.Sprintf("profile-%d", i),
			Name: fmt.Sprintf("Profile %d", i),
			Targets: []Target{{
				Interface: "Ethernet",
				Mode:      ModeStatic,
				Address:   fmt.Sprintf("10.%d.%d.20", i/256, i%256),
				Mask:      "255.255.255.0",
				Gateway:   fmt.Sprintf("10.%d.%d.1", i/256, i%256),
				DNS:       []string{"9.9.9.9"},
			}},
		}
	}
	return profiles
}

// The file is re-read on every launch: 200 profiles is a wide margin over the
// few dozen expected.
func BenchmarkStoreLoad(b *testing.B) {
	s := NewStore(filepath.Join(b.TempDir(), "profiles.yaml"))
	if err := s.Save(makeProfiles(200)); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for range b.N {
		if _, err := s.Load(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStoreSave(b *testing.B) {
	s := NewStore(filepath.Join(b.TempDir(), "profiles.yaml"))
	profiles := makeProfiles(200)

	b.ResetTimer()
	for range b.N {
		if err := s.Save(profiles); err != nil {
			b.Fatal(err)
		}
	}
}

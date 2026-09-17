// Package cli handles the command-line forms of the application, so a site can
// have its own desktop shortcut instead of being chosen in the window.
package cli

import (
	"fmt"
	"io"
	"strings"

	"networkprofile/internal/profile"
)

// Env is what the commands need from the rest of the application, supplied by
// the caller so the commands themselves can be tested without touching a
// machine's adapters.
type Env struct {
	Profiles func() ([]profile.Profile, error)
	Apply    func(id string) error
	Out      io.Writer
	Err      io.Writer
}

const usage = `NetworkProfile — bascule entre profils réseau

  NetworkProfile.exe                     ouvre la fenêtre
  NetworkProfile.exe --list              liste les profils enregistrés
  NetworkProfile.exe --profile <nom>     applique un profil, puis quitte

Le nom accepte l'identifiant du profil ou son nom affiché, sans tenir compte
de la casse. Appliquer un profil demande les droits administrateur.
`

// Run interprets the command line. It reports whether the arguments asked for
// a command-line action at all: when they did not, the caller opens the window
// as usual.
func Run(args []string, env Env) (handled bool, code int) {
	switch {
	case len(args) == 0:
		return false, 0

	case matches(args[0], "--help", "-h", "/?"):
		fmt.Fprint(env.Out, usage)
		return true, 0

	case matches(args[0], "--list", "-l"):
		return true, list(env)

	case matches(args[0], "--profile", "-p"):
		if len(args) < 2 || strings.TrimSpace(args[1]) == "" {
			fmt.Fprintln(env.Err, "erreur : --profile attend le nom d'un profil")
			return true, 2
		}
		return true, apply(args[1], env)

	default:
		fmt.Fprintf(env.Err, "erreur : argument inconnu %q\n\n%s", args[0], usage)
		return true, 2
	}
}

func matches(arg string, names ...string) bool {
	for _, name := range names {
		if strings.EqualFold(arg, name) {
			return true
		}
	}
	return false
}

func list(env Env) int {
	profiles, err := env.Profiles()
	if err != nil {
		fmt.Fprintf(env.Err, "erreur : %v\n", err)
		return 1
	}
	if len(profiles) == 0 {
		fmt.Fprintln(env.Out, "Aucun profil enregistré.")
		return 0
	}

	for _, p := range profiles {
		fmt.Fprintf(env.Out, "%-28s %s\n", p.ID, describe(p))
	}
	return 0
}

func describe(p profile.Profile) string {
	parts := make([]string, 0, len(p.Targets))
	for _, target := range p.Targets {
		switch target.Mode {
		case profile.ModeDHCP:
			parts = append(parts, target.Interface+" : DHCP")
		default:
			parts = append(parts, fmt.Sprintf("%s : %s", target.Interface, target.Address))
		}
	}
	return fmt.Sprintf("%s (%s)", p.Name, strings.Join(parts, ", "))
}

func apply(wanted string, env Env) int {
	profiles, err := env.Profiles()
	if err != nil {
		fmt.Fprintf(env.Err, "erreur : %v\n", err)
		return 1
	}

	found, err := resolve(wanted, profiles)
	if err != nil {
		fmt.Fprintf(env.Err, "erreur : %v\n", err)
		return 1
	}

	if err := env.Apply(found.ID); err != nil {
		fmt.Fprintf(env.Err, "erreur : %v\n", err)
		return 1
	}

	fmt.Fprintf(env.Out, "Profil appliqué : %s\n", found.Name)
	return 0
}

// resolve accepts the identifier or the displayed name. A name typed on a
// command line is not going to match the accents and capitals of the window, so
// the comparison ignores case — but an ambiguous answer is refused rather than
// resolved arbitrarily, since the wrong profile means the wrong network.
func resolve(wanted string, profiles []profile.Profile) (profile.Profile, error) {
	wanted = strings.TrimSpace(wanted)

	for _, p := range profiles {
		if p.ID == wanted {
			return p, nil
		}
	}

	var matched []profile.Profile
	for _, p := range profiles {
		if strings.EqualFold(p.Name, wanted) || strings.EqualFold(p.ID, wanted) {
			matched = append(matched, p)
		}
	}

	switch len(matched) {
	case 1:
		return matched[0], nil
	case 0:
		return profile.Profile{}, fmt.Errorf("aucun profil nommé %q — « --list » donne les noms disponibles", wanted)
	default:
		names := make([]string, len(matched))
		for i, p := range matched {
			names[i] = p.ID
		}
		return profile.Profile{}, fmt.Errorf("%q correspond à plusieurs profils (%s) : utilisez l'identifiant", wanted, strings.Join(names, ", "))
	}
}

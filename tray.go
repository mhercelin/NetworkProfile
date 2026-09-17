package main

import (
	_ "embed"
	"sync"

	"github.com/energye/systray"

	"networkprofile/internal/profile"
)

//go:embed build/windows/icon.ico
var trayIcon []byte

// The notification area offers no way to delete a menu item once added, so a
// fixed number of slots is created up front and then renamed or hidden. Twelve
// is more pinned sites than anyone will keep in a menu.
const traySlots = 12

// tray puts the application in the notification area: the window comes back
// from here without another elevation, and the pinned profiles are one click
// away without opening it at all.
//
// Wails v2 has no notification area of its own, so the icon runs its own
// message loop alongside the window's — which is also why every field below is
// touched from two threads and guarded.
type tray struct {
	apply func(id string)

	mu     sync.Mutex
	slots  []*systray.MenuItem
	pinned []profile.Profile
	ready  bool
}

func newTray(apply func(id string)) *tray {
	return &tray{apply: apply}
}

// run blocks on the icon's own message loop; call it in a goroutine.
func (t *tray) run(show, quit func()) {
	systray.Run(func() {
		systray.SetIcon(trayIcon)
		systray.SetTooltip("NetworkProfile")

		systray.AddMenuItem("Afficher", "Afficher la fenêtre").Click(show)

		t.mu.Lock()
		for slot := range traySlots {
			item := systray.AddMenuItem("", "")
			item.Hide()
			item.Click(func() { t.applySlot(slot) })
			t.slots = append(t.slots, item)
		}
		t.ready = true
		// Pins may have been handed over before the menu existed.
		t.refreshLocked()
		t.mu.Unlock()

		systray.AddSeparator()
		systray.AddMenuItem("Quitter", "Fermer NetworkProfile et son assistant élevé").Click(quit)

		// Double-clicking the icon is the habit most people have.
		systray.SetOnDClick(func(systray.IMenu) { show() })
	}, nil)
}

// SetProfiles hands over the current profiles; only the pinned ones reach the
// menu.
func (t *tray) SetProfiles(profiles []profile.Profile) {
	pinned := make([]profile.Profile, 0, traySlots)
	for _, p := range profiles {
		if p.Pinned {
			pinned = append(pinned, p)
		}
		if len(pinned) == traySlots {
			break
		}
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	t.pinned = pinned
	t.refreshLocked()
}

func (t *tray) refreshLocked() {
	if !t.ready {
		return
	}
	for i, item := range t.slots {
		if i >= len(t.pinned) {
			item.Hide()
			continue
		}
		item.SetTitle(t.pinned[i].Name)
		item.Show()
	}
}

// applySlot resolves the slot under the lock but applies outside it: applying
// waits on an elevation prompt, and the menu must not be frozen meanwhile.
func (t *tray) applySlot(slot int) {
	t.mu.Lock()
	if slot >= len(t.pinned) {
		t.mu.Unlock()
		return
	}
	id := t.pinned[slot].ID
	t.mu.Unlock()

	t.apply(id)
}

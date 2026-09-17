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
	header *systray.MenuItem
	slots  []*systray.MenuItem
	pinned []profile.Profile
	active string
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

		systray.AddMenuItem("Afficher la fenêtre", "Ouvrir NetworkProfile").Click(show)

		t.mu.Lock()

		// A disabled caption separates the profiles from the two actions that
		// operate on the application itself; without it the menu is a flat list
		// where "Quitter" sits among the sites.
		systray.AddSeparator()
		t.header = systray.AddMenuItem("Profils épinglés", "")
		t.header.Disable()
		t.header.Hide()

		// Checkboxes rather than plain entries: the tick is how a menu says
		// "this is the one you are on", and it is drawn by Windows itself.
		for slot := range traySlots {
			item := systray.AddMenuItemCheckbox("", "", false)
			item.Hide()
			item.Click(func() { t.applySlot(slot) })
			t.slots = append(t.slots, item)
		}

		t.ready = true
		// Profiles may have been handed over before the menu existed.
		t.refreshLocked()
		t.mu.Unlock()

		systray.AddSeparator()
		systray.AddMenuItem("Quitter", "Fermer NetworkProfile et son assistant élevé").Click(quit)

		// Double-clicking the icon is the habit most people have.
		systray.SetOnDClick(func(systray.IMenu) { show() })
	}, nil)
}

// SetProfiles hands over the current profiles and which one is active. Only the
// pinned ones reach the menu.
func (t *tray) SetProfiles(profiles []profile.Profile, active string) {
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
	t.active = active
	t.refreshLocked()
}

func (t *tray) refreshLocked() {
	if !t.ready {
		return
	}

	if len(t.pinned) == 0 {
		t.header.Hide()
	} else {
		t.header.Show()
	}

	for i, item := range t.slots {
		if i >= len(t.pinned) {
			item.Hide()
			continue
		}

		p := t.pinned[i]
		item.SetTitle(p.Name)
		if p.ID == t.active {
			item.Check()
			// Re-applying the configuration already in place would only cost an
			// elevation prompt for nothing.
			item.Disable()
		} else {
			item.Uncheck()
			item.Enable()
		}
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

package main

import (
	_ "embed"
	"log"
	"slices"
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
// message loop alongside the window's. Two rules follow, and both were learned
// the hard way: never call into the menu while holding the state lock, and
// never let a menu update block the write that triggered it. The menu is a view
// of the profiles; a view that stalls must not stall what it shows.
type tray struct {
	apply func(id string)

	mu     sync.Mutex
	header *systray.MenuItem
	slots  []*systray.MenuItem
	pinned []profile.Profile
	active []string
	ready  bool

	// renderMu serialises menu updates on their own, so two refreshes cannot
	// interleave without any of it happening under mu.
	renderMu sync.Mutex
}

func newTray(apply func(id string)) *tray {
	return &tray{apply: apply}
}

// run blocks on the icon's own message loop; call it in a goroutine.
func (t *tray) run(show, quit func()) {
	systray.Run(func() {
		log.Print("zone de notification : construction du menu")

		systray.SetIcon(trayIcon)
		systray.SetTooltip("NetworkProfile")

		systray.AddMenuItem("Afficher la fenêtre", "Ouvrir NetworkProfile").Click(show)

		// A disabled caption separates the profiles from the actions that
		// operate on the application itself; without it the menu is a flat list
		// where "Quitter" sits among the sites.
		systray.AddSeparator()
		header := systray.AddMenuItem("Profils épinglés", "")
		header.Disable()

		// Checkboxes rather than plain entries: the tick is how a menu says
		// "this one is in effect", and Windows draws it itself.
		slots := make([]*systray.MenuItem, 0, traySlots)
		for slot := range traySlots {
			item := systray.AddMenuItemCheckbox("", "", false)
			item.Click(func() { t.applySlot(slot) })
			slots = append(slots, item)
		}

		systray.AddSeparator()
		systray.AddMenuItem("Quitter", "Fermer NetworkProfile et son assistant élevé").Click(quit)

		// Double-clicking the icon is the habit most people have.
		systray.SetOnDClick(func(systray.IMenu) { show() })

		t.mu.Lock()
		t.header = header
		t.slots = slots
		t.ready = true
		pinned, active := t.pinned, t.active
		t.mu.Unlock()

		log.Printf("zone de notification : menu prêt (%d emplacements)", len(slots))

		// Profiles may have arrived before the menu existed.
		t.render(header, slots, pinned, active)
	}, nil)
}

// SetProfiles hands over the current profiles and which of them are in effect.
// Only the pinned ones reach the menu.
func (t *tray) SetProfiles(profiles []profile.Profile, active []string) {
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
	t.pinned = pinned
	t.active = active
	ready, header, slots := t.ready, t.header, t.slots
	t.mu.Unlock()

	if !ready {
		log.Printf("zone de notification : %d profil(s) épinglé(s) en attente du menu", len(pinned))
		return
	}
	t.render(header, slots, pinned, active)
}

// render touches the menu, and never runs under mu.
func (t *tray) render(header *systray.MenuItem, slots []*systray.MenuItem, pinned []profile.Profile, active []string) {
	t.renderMu.Lock()
	defer t.renderMu.Unlock()

	if len(pinned) == 0 {
		header.Hide()
	} else {
		header.Show()
	}

	for i, item := range slots {
		if i >= len(pinned) {
			item.Hide()
			continue
		}

		p := pinned[i]
		item.SetTitle(p.Name)
		if slices.Contains(active, p.ID) {
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

	log.Printf("zone de notification : %d profil(s) affiché(s), actifs=%v", len(pinned), active)
}

// applySlot resolves the slot under the lock but applies outside it: applying
// waits on an elevation prompt, and nothing else must be held meanwhile.
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

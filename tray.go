package main

import (
	_ "embed"

	"github.com/energye/systray"
)

//go:embed build/windows/icon.ico
var trayIcon []byte

// startTray puts the application in the notification area. This is what makes
// the single credentials prompt worthwhile: closing the window only hides it,
// the process and its elevated helper stay alive, and the window comes back
// from here without another elevation.
//
// Wails v2 has no notification area of its own, so the icon runs its own
// message loop alongside the window's.
func startTray(show, quit func()) {
	systray.Run(func() {
		systray.SetIcon(trayIcon)
		systray.SetTooltip("NetworkProfile")

		open := systray.AddMenuItem("Afficher", "Afficher la fenêtre")
		systray.AddSeparator()
		leave := systray.AddMenuItem("Quitter", "Fermer NetworkProfile et son assistant élevé")

		open.Click(show)
		leave.Click(quit)

		// Double-clicking the icon is the habit most people have.
		systray.SetOnDClick(func(systray.IMenu) { show() })
	}, nil)
}

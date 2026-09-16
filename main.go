package main

import (
	"embed"
	"log"
	"os"
	"path/filepath"

	"networkprofile/internal/gui"
	"networkprofile/internal/network"
	"networkprofile/internal/profile"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// WebView2 defaults its cache to the running account's %AppData%. Elevation
	// can switch accounts, and that profile is then unreachable — so its store
	// is pinned next to ours instead.
	webviewData := filepath.Join(gui.DataDir(), "webview")
	if err := os.MkdirAll(webviewData, 0o755); err != nil {
		log.Fatalf("création de %s : %v", webviewData, err)
	}

	app := gui.New(profile.NewStore(gui.DefaultStorePath()), network.NewWindows())

	err := wails.Run(&options.App{
		Title:            "NetworkProfile",
		Width:            1100,
		Height:           720,
		MinWidth:         900,
		MinHeight:        600,
		AssetServer:      &assetserver.Options{Assets: assets},
		BackgroundColour: &options.RGBA{R: 247, G: 248, B: 250, A: 1},
		Bind:             []any{app},
		Windows: &windows.Options{
			WebviewUserDataPath: webviewData,
		},
	})
	if err != nil {
		log.Fatalf("démarrage de l'interface : %v", err)
	}
}

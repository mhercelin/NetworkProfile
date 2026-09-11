package main

import (
	"embed"
	"log"

	"networkprofile/internal/gui"
	"networkprofile/internal/network"
	"networkprofile/internal/profile"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	storePath, err := gui.DefaultStorePath()
	if err != nil {
		log.Fatalf("emplacement du fichier de profils : %v", err)
	}

	app := gui.New(profile.NewStore(storePath), network.NewWindows())

	err = wails.Run(&options.App{
		Title:            "NetworkProfile",
		Width:            1100,
		Height:           720,
		MinWidth:         900,
		MinHeight:        600,
		AssetServer:      &assetserver.Options{Assets: assets},
		BackgroundColour: &options.RGBA{R: 247, G: 248, B: 250, A: 1},
		Bind:             []any{app},
	})
	if err != nil {
		log.Fatalf("démarrage de l'interface : %v", err)
	}
}

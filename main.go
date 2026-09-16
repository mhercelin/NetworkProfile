package main

import (
	"context"
	"embed"
	"log"
	"os"

	"networkprofile/internal/gui"
	"networkprofile/internal/ipc"
	"networkprofile/internal/network"
	"networkprofile/internal/profile"

	"github.com/energye/systray"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// The same executable is both halves: started plainly it is the interface,
	// started with --helper it is the elevated half the interface spawned.
	if args, ok := ipc.ParseHelperArgs(os.Args[1:]); ok {
		if err := ipc.ServeHelper(args.Pipe, args.Owner, args.Parent, network.NewWindows()); err != nil {
			log.Fatalf("assistant élevé : %v", err)
		}
		return
	}

	if err := runInterface(); err != nil {
		log.Fatal(err)
	}
}

func runInterface() error {
	storePath, err := gui.DefaultStorePath()
	if err != nil {
		return err
	}

	// Reads are answered in this process; only writes cross into the helper,
	// which is started the first time one is attempted.
	manager := ipc.NewManager(network.NewWindows(), ipc.StartHelper)
	defer manager.Close()

	app := gui.New(profile.NewStore(storePath), manager)

	var ctx context.Context

	return wails.Run(&options.App{
		Title:            "NetworkProfile",
		Width:            1100,
		Height:           720,
		MinWidth:         900,
		MinHeight:        600,
		AssetServer:      &assetserver.Options{Assets: assets},
		BackgroundColour: &options.RGBA{R: 247, G: 248, B: 250, A: 1},
		Bind:             []any{app},
		OnStartup: func(c context.Context) {
			ctx = c
			go startTray(
				func() {
					wruntime.WindowUnminimise(c)
					wruntime.WindowShow(c)
				},
				func() {
					systray.Quit()
					wruntime.Quit(c)
				},
			)
		},

		// Closing the window keeps the process — and with it the elevated
		// helper — alive, so the credentials are asked for once a day rather
		// than once per change.
		HideWindowOnClose: true,

		// Launching the executable again raises the running window instead of
		// starting a second copy, which would mean a second elevation.
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: "networkprofile-2f7c41d8",
			OnSecondInstanceLaunch: func(options.SecondInstanceData) {
				if ctx == nil {
					return
				}
				wruntime.WindowUnminimise(ctx)
				wruntime.WindowShow(ctx)
			},
		},
	})
}

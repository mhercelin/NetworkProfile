package main

import (
	"context"
	"embed"
	"log"
	"os"

	"networkprofile/internal/cli"
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
	// The same executable is all three: started plainly it is the interface,
	// with --helper the elevated half it spawns, and with a command it applies
	// a profile and leaves.
	if args, ok := ipc.ParseHelperArgs(os.Args[1:]); ok {
		if err := ipc.ServeHelper(args.Pipe, args.Owner, args.Parent, network.NewWindows()); err != nil {
			log.Fatalf("assistant élevé : %v", err)
		}
		return
	}

	if len(os.Args) > 1 {
		if code := runCommandLine(os.Args[1:]); code >= 0 {
			os.Exit(code)
		}
	}

	if err := runInterface(); err != nil {
		log.Fatal(err)
	}
}

// runCommandLine reports the exit code, or -1 when the arguments turned out not
// to be a command after all and the window should open.
func runCommandLine(args []string) int {
	attachConsole()

	storePath, err := gui.DefaultStorePath()
	if err != nil {
		log.Print(err)
		return 1
	}

	manager := ipc.NewManager(network.NewWindows(), ipc.StartHelper)
	defer func() { _ = manager.Close() }()

	// The same surface the window is bound to, so a profile applied from a
	// shortcut goes through exactly the path a click goes through.
	app := gui.New(profile.NewStore(storePath), manager)

	handled, code := cli.Run(args, cli.Env{
		Profiles: app.Profiles,
		Apply:    app.ApplyProfile,
		Out:      os.Stdout,
		Err:      os.Stderr,
	})
	if !handled {
		return -1
	}
	return code
}

func runInterface() error {
	storePath, err := gui.DefaultStorePath()
	if err != nil {
		return err
	}

	// Reads are answered in this process; only writes cross into the helper,
	// which is started the first time one is attempted.
	manager := ipc.NewManager(network.NewWindows(), ipc.StartHelper)
	defer func() {
		// The helper also exits on its own when this process does, so a failure
		// here leaves nothing behind — but it is worth knowing about.
		if err := manager.Close(); err != nil {
			log.Printf("arrêt de l'assistant élevé : %v", err)
		}
	}()

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

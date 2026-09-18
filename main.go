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
		os.Exit(runHelper(args))
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

// runHelper is the elevated half. It writes to the same log as the interface,
// so a failure that crosses the privilege boundary can be read in one place.
func runHelper(args ipc.HelperArgs) int {
	if args.Log != "" {
		closer := startLogging(args.Log, "[assistant]")
		defer func() { _ = closer.Close() }()
	}

	log.Printf("démarrage (pid %d)", os.Getpid())

	if err := ipc.ServeHelper(args.Pipe, args.Owner, args.Parent, network.NewWindows()); err != nil {
		log.Printf("arrêt sur erreur : %v", err)
		return 1
	}

	log.Print("arrêt normal")
	return 0
}

// applyFromTray applies a pinned profile. The window is usually hidden when
// this runs, so a failure has to announce itself: otherwise the operator is
// left believing the network changed when it did not.
func applyFromTray(ctx context.Context, app *gui.App, id string) {
	log.Printf("zone de notification : application de %q", id)

	err := app.ApplyProfile(id)
	if err == nil {
		log.Printf("zone de notification : %q appliqué", id)
		return
	}

	// Logged as well as shown: a dialog raised from a hidden window is exactly
	// the kind of thing that can fail to appear, and then the failure would
	// leave no trace at all.
	log.Printf("zone de notification : échec de %q : %v", id, err)

	if ctx == nil {
		return
	}
	if _, dialogErr := wruntime.MessageDialog(ctx, wruntime.MessageDialogOptions{
		Type:    wruntime.ErrorDialog,
		Title:   "NetworkProfile",
		Message: err.Error(),
	}); dialogErr != nil {
		log.Printf("zone de notification : impossible d'afficher l'erreur : %v", dialogErr)
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

	journal, err := logPath()
	if err != nil {
		log.Print(err)
		return 1
	}
	closer := startLogging(journal, "[commande]")
	defer func() { _ = closer.Close() }()

	manager := ipc.NewManager(network.NewWindows(), ipc.HelperLauncher(journal))
	defer func() { _ = manager.Close() }()

	// The same surface the window is bound to, so a profile applied from a
	// shortcut goes through exactly the path a click goes through. Nothing
	// watches for changes here: the process ends with the command.
	app := gui.New(profile.NewStore(storePath), manager, nil)

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

	journal, err := logPath()
	if err != nil {
		return err
	}
	trimLog(journal)
	closer := startLogging(journal, "[ihm]")
	defer func() { _ = closer.Close() }()
	log.Printf("démarrage (pid %d)", os.Getpid())

	// Reads are answered in this process; only writes cross into the helper,
	// which is started the first time one is attempted.
	manager := ipc.NewManager(network.NewWindows(), ipc.HelperLauncher(journal))
	defer func() {
		// The helper also exits on its own when this process does, so a failure
		// here leaves nothing behind — but it is worth knowing about.
		if err := manager.Close(); err != nil {
			log.Printf("arrêt de l'assistant élevé : %v", err)
		}
	}()

	var (
		ctx  context.Context
		app  *gui.App
		icon = newTray(func(id string) { applyFromTray(ctx, app, id) })
	)

	// The menu follows the stored profiles instead of polling them, and the
	// window is told so its list stops showing the old active profile.
	// Refreshing the two views runs on its own goroutine, and this matters: the
	// notification area menu is a view of the profiles, and a view that stalls
	// must never stall the write it is reacting to. Saving a profile is not
	// allowed to depend on a menu being well.
	refresh := func() {
		profiles, err := app.Profiles()
		if err != nil {
			log.Printf("relecture des profils : %v", err)
			return
		}
		active, err := app.ActiveProfileIDs()
		if err != nil {
			log.Printf("profils actifs : %v", err)
		}

		icon.SetProfiles(profiles, active)
		if ctx != nil {
			wruntime.EventsEmit(ctx, "profiles:changed")
		}
	}

	app = gui.New(profile.NewStore(storePath), manager, func() { go refresh() })

	return wails.Run(&options.App{
		Title:            "NetworkProfile",
		Width:            1100,
		Height:           720,
		MinWidth:         900,
		MinHeight:        600,
		AssetServer:      &assetserver.Options{Assets: assets},
		BackgroundColour: &options.RGBA{R: 247, G: 248, B: 250, A: 1},
		Bind:             []any{app},
		// Quitting through the runtime rather than only from the menu: the icon
		// was the sole way out, so a menu that fails to open leaves the
		// application running with no way to stop it.
		OnShutdown: func(context.Context) {
			log.Print("arrêt")
			systray.Quit()
		},

		OnStartup: func(c context.Context) {
			ctx = c
			go refresh()

			go icon.run(
				func() {
					wruntime.WindowUnminimise(c)
					wruntime.WindowShow(c)
				},
				func() { wruntime.Quit(c) },
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

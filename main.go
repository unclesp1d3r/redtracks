// Package main is the RedTracks Wails desktop application entry point.
package main

import (
	"embed"
	"log"

	"github.com/unclesp1d3r/redtracks/internal/app"
	"github.com/unclesp1d3r/redtracks/internal/store"
	"github.com/wailsapp/wails/v3/pkg/application"
)

const (
	appName      = "RedTracks"
	windowWidth  = 1400
	windowHeight = 900
	titleBarH    = 50

	// Smallest window that keeps the two-pane Host layout usable beside a VM.
	minWindowWidth  = 900
	minWindowHeight = 600

	// Window background, matches the dark theme --background token.
	bgRed, bgGreen, bgBlue = 14, 17, 27
)

// Files under frontend/dist are embedded into the binary and served to the webview.
//
//go:embed all:frontend/dist
var assets embed.FS

func main() {
	backend, err := app.New(store.RootDir())
	if err != nil {
		log.Fatal(err)
	}

	wailsApp := application.New(application.Options{
		Name:        appName,
		Description: "Offline pentest and CTF engagement tracker",
		Services: []application.Service{
			application.NewService(app.NewEngagementService(backend)),
			application.NewService(app.NewHostService(backend)),
			application.NewService(app.NewImportService(backend)),
			application.NewService(app.NewChecklistService(backend)),
		},
		// Localhost-only and unauthenticated by design: single-operator machine, browser fallback.
		Server: application.ServerOptions{Host: "localhost"},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	backend.SetEmitter(func(name string, data any) { wailsApp.Event.Emit(name, data) })

	wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:     appName,
		Width:     windowWidth,
		Height:    windowHeight,
		MinWidth:  minWindowWidth,
		MinHeight: minWindowHeight,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: titleBarH,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		BackgroundColour: application.NewRGB(bgRed, bgGreen, bgBlue),
		URL:              "/",
	})

	if err := wailsApp.Run(); err != nil {
		log.Fatal(err)
	}
}

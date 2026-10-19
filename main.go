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

	// Window background, the dark theme --background token oklch(0.15 0.02 260) in sRGB.
	bgRed, bgGreen, bgBlue = 7, 11, 20
)

// Files under frontend/dist are embedded into the binary and served to the webview.
//
//go:embed all:frontend/dist
var assets embed.FS

func main() {
	root, err := store.RootDir()
	if err != nil {
		log.Fatal(err)
	}
	backend, err := app.New(root)
	if err != nil {
		log.Fatal(err)
	}
	importSvc := app.NewImportService(backend)

	wailsApp := application.New(application.Options{
		Name:        appName,
		Description: "Offline pentest and CTF engagement tracker",
		Services: []application.Service{
			application.NewService(app.NewEngagementService(backend)),
			application.NewService(app.NewHostService(backend)),
			application.NewService(importSvc),
			application.NewService(app.NewChecklistService(backend)),
			application.NewService(app.NewDiagnosticsService(backend)),
			application.NewService(app.NewEvidenceService(backend)),
			application.NewService(app.NewSessionLogService(backend)),
		},
		// Localhost-only and unauthenticated by design: single-operator machine, browser fallback.
		Server: application.ServerOptions{Host: "localhost"},
		Assets: application.AssetOptions{
			Handler:    application.AssetFileServerFS(assets),
			Middleware: app.EvidenceAssets(backend),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	backend.SetEmitter(func(name string, data any) { wailsApp.Event.Emit(name, data) })

	win := wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
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
		EnableFileDrop:   true,
	})
	wireDesktop(wailsApp, win, backend, app.NewDropper(backend, importSvc))

	if err := wailsApp.Run(); err != nil {
		log.Fatal(err)
	}
}

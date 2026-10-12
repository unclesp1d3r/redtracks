// Package main is the RedTracks Wails desktop application entry point.
package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
)

const (
	appName      = "RedTracks"
	windowWidth  = 1400
	windowHeight = 900
	titleBarH    = 50

	// Window background, matches the dark theme --background token.
	bgRed, bgGreen, bgBlue = 14, 17, 27
)

// Files under frontend/dist are embedded into the binary and served to the webview.
//
//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := application.New(application.Options{
		Name:        appName,
		Description: "Offline pentest and CTF engagement tracker",
		// Register backend services here.
		Services: []application.Service{},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  appName,
		Width:  windowWidth,
		Height: windowHeight,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: titleBarH,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		BackgroundColour: application.NewRGB(bgRed, bgGreen, bgBlue),
		URL:              "/",
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

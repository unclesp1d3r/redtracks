//go:build server

package main

import (
	"errors"

	"github.com/unclesp1d3r/redtracks/internal/app"
	"github.com/wailsapp/wails/v3/pkg/application"
)

var errNoNativeDialog = errors.New("no native file dialog in the server build")

// wireDesktop in the server build reports every native capability as absent;
// the frontend falls back to HTML5 drop, typed paths and copy path.
func wireDesktop(_ *application.App, _ *application.WebviewWindow, backend *app.App, _ *app.Dropper) {
	backend.SetHooks(app.Hooks{
		PickFile: func() (string, error) { return "", errNoNativeDialog },
	})
}

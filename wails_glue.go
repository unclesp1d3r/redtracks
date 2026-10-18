//go:build !server

package main

import (
	"github.com/unclesp1d3r/redtracks/internal/app"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// wireDesktop injects the native integrations (KTD15): capabilities, the
// open-file dialog, the default application menu (so Cmd+V reaches the
// webview on macOS) and the window's file-drop subscription.
func wireDesktop(wailsApp *application.App, win *application.WebviewWindow, backend *app.App, dropper *app.Dropper) {
	backend.SetHooks(app.Hooks{
		Capabilities: app.Capabilities{NativeDrop: true, NativeDialog: true, Reveal: true},
		PickFile:     func() (string, error) { return pickFile(wailsApp) },
	})
	wailsApp.Menu.Set(application.DefaultApplicationMenu())
	win.OnWindowEvent(events.Common.WindowFilesDropped, func(e *application.WindowEvent) {
		onFilesDropped(wailsApp, dropper, e)
	})
}

// pickFile shows the native open-file dialog; a cancel yields "" and no error.
func pickFile(wailsApp *application.App) (string, error) {
	return wailsApp.Dialog.OpenFile().CanChooseFiles(true).CanChooseDirectories(false).PromptForSingleSelection()
}

func onFilesDropped(wailsApp *application.App, dropper *app.Dropper, e *application.WindowEvent) {
	ctx := e.Context()
	paths := ctx.DroppedFiles()
	var attrs map[string]string // a drop outside any target has no details: nil means "no target"
	if details := ctx.DropTargetDetails(); details != nil {
		attrs = details.Attributes
	}
	// Copying can take a while; keep it off the UI thread.
	go func() {
		wailsApp.Event.Emit(app.EventDropped, dropper.HandleDrop(paths, attrs))
	}()
}

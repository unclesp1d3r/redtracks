package app

// Capabilities reports which desktop integrations main.go wired in. The
// server build leaves every field false and the frontend adapts (KTD15).
type Capabilities struct {
	NativeDrop   bool // the window delivers dropped file paths to Go
	NativeDialog bool // PickFile opens a native open-file dialog
	Reveal       bool // a file manager can reveal a path
}

// Hooks are the desktop integrations injected by main.go. PickFile opens the
// native open-file dialog and returns the chosen absolute path, or "" when
// the operator cancelled; it is nil in the server build.
type Hooks struct {
	Capabilities Capabilities
	PickFile     func() (string, error)
}

// SetHooks installs the desktop integrations; main.go calls it before Run.
func (a *App) SetHooks(h Hooks) {
	a.mu.Lock()
	a.hooks = h
	a.mu.Unlock()
}

// Hooks returns the installed desktop integrations.
func (a *App) Hooks() Hooks {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.hooks
}

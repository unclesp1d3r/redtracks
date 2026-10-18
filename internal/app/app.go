// Package app holds the Wails services that expose the engagement store,
// checklists and importers to the frontend, plus the rotating log.
package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/unclesp1d3r/redtracks/checklists"
	"github.com/unclesp1d3r/redtracks/internal/checklist"
	"github.com/unclesp1d3r/redtracks/internal/store"
)

const (
	dirMode          = 0o750
	logFileName      = "redtracks.log"
	checklistsDir    = "checklists"
	engagementsDir   = "engagements"
	eventChanged     = "engagement:changed"
	eventError       = "engagement:error"
	checklistsSource = "checklists"
)

// Poll intervals; variables so tests can shorten them.
var (
	pollInterval      = 2 * time.Second
	checklistInterval = 2 * time.Second
)

var errNoEngagement = errors.New("no engagement open")

// App is the shared state behind the Wails services.
type App struct {
	Root string

	mu             sync.Mutex
	store          *store.Store
	cancelPoll     context.CancelFunc
	cancelWatch    context.CancelFunc
	defs           []checklist.Definition
	defsErr        error
	installErr     error
	installedFiles []string
	onOpen         []func()
	log            *slog.Logger
	logCloser      io.Closer
	emit           func(name string, data any)
}

// New prepares root, opens the log, installs the default checklists, loads
// checklist definitions and starts watching them. An install or load error is
// logged and emitted once an emitter is set (SetEmitter flushes both); neither
// is fatal.
func New(root string) (*App, error) {
	if err := os.MkdirAll(filepath.Join(root, engagementsDir), dirMode); err != nil {
		return nil, fmt.Errorf("create %s: %w", root, err)
	}
	lg, closer, err := OpenLog(filepath.Join(root, logFileName))
	if err != nil {
		return nil, err
	}
	a := &App{Root: root, log: lg, logCloser: closer, defs: []checklist.Definition{}}
	dir := a.checklistsDir()
	added, err := checklist.InstallDefaults(checklists.FS, dir)
	if err != nil {
		a.installErr = err
		lg.Error("install default checklists", "err", err)
	}
	if len(added) > 0 {
		a.installedFiles = added
		lg.Info("installed bundled checklist files", "files", added)
	}
	if err := a.reloadDefs(); err != nil {
		lg.Error("load checklists", "err", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.cancelWatch = cancel
	go checklist.Watch(ctx, dir, checklistInterval, a.onDefsChanged, a.onDefsError)
	return a, nil
}

// reloadDefs loads definitions from disk; on failure the previous set is kept
// and the error is remembered for SetEmitter.
func (a *App) reloadDefs() error {
	defs, err := checklist.LoadDefinitions(a.checklistsDir())
	a.mu.Lock()
	defer a.mu.Unlock()
	if err != nil {
		a.defsErr = err
		return err
	}
	a.defs, a.defsErr = defs, nil
	return nil
}

func (a *App) onDefsChanged(defs []checklist.Definition) {
	a.mu.Lock()
	a.defs, a.defsErr = defs, nil
	a.mu.Unlock()
	a.emitEvent(eventChanged, []string{checklistsSource})
}

func (a *App) onDefsError(err error) {
	a.mu.Lock()
	a.defsErr = err
	a.mu.Unlock()
	a.log.Error("watch checklists", "err", err)
	a.emitError(errorKindChecklists, a.checklistsDir(), err)
}

func (a *App) checklistsDir() string { return filepath.Join(a.Root, checklistsDir) }

// SetEmitter installs the event emitter and flushes pending checklist errors.
func (a *App) SetEmitter(fn func(name string, data any)) {
	a.mu.Lock()
	a.emit = fn
	pending := []error{a.installErr, a.defsErr}
	a.mu.Unlock()
	for _, err := range pending {
		if err != nil {
			a.emitError(errorKindChecklists, a.checklistsDir(), err)
		}
	}
}

// onEngagementOpened registers fn to run whenever another engagement is opened.
func (a *App) onEngagementOpened(fn func()) {
	a.mu.Lock()
	a.onOpen = append(a.onOpen, fn)
	a.mu.Unlock()
}

func (a *App) notifyOpened() {
	a.mu.Lock()
	hooks := slices.Clone(a.onOpen)
	a.mu.Unlock()
	for _, fn := range hooks {
		fn()
	}
}

// LogError writes a frontend-reported problem to redtracks.log.
func (a *App) LogError(where, message string) {
	a.log.Error("frontend", "where", where, "message", message)
}

func (a *App) emitEvent(name string, data any) {
	a.mu.Lock()
	fn := a.emit
	a.mu.Unlock()
	if fn == nil {
		a.log.Warn("event dropped, no emitter", "event", name)
		return
	}
	fn(name, data)
}

// Store returns the open engagement store.
func (a *App) Store() (*store.Store, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.store == nil {
		return nil, errNoEngagement
	}
	return a.store, nil
}

// Definitions returns a copy of the current checklist definitions.
func (a *App) Definitions() []checklist.Definition {
	a.mu.Lock()
	defer a.mu.Unlock()
	return slices.Clone(a.defs)
}

func (a *App) definitionsStatus() DefinitionsStatus {
	a.mu.Lock()
	defer a.mu.Unlock()
	st := DefinitionsStatus{Count: len(a.defs), InstalledFiles: slices.Clone(a.installedFiles)}
	if a.defsErr != nil {
		st.Error = a.defsErr.Error()
	}
	if a.installErr != nil {
		st.InstallError = a.installErr.Error()
	}
	return st
}

// fail logs err under method at Error level and returns it unchanged.
func (a *App) fail(method string, err error) error {
	if err != nil {
		a.log.Error(method, "err", err)
	}
	return err
}

// withStore runs fn against the open store, logging any error under method.
func (a *App) withStore(method string, fn func(*store.Store) error) error {
	st, err := a.Store()
	if err != nil {
		return a.fail(method, err)
	}
	return a.fail(method, fn(st))
}

// Close stops background work and closes the log.
func (a *App) Close() error {
	a.mu.Lock()
	if a.cancelPoll != nil {
		a.cancelPoll()
	}
	a.cancelWatch()
	a.mu.Unlock()
	return a.logCloser.Close()
}

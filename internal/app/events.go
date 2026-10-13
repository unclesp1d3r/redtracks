package app

import (
	"errors"
	"path/filepath"

	"github.com/unclesp1d3r/redtracks/internal/store"
)

// Values of ErrorEvent.Kind.
const (
	errorKindEngagement = "engagement"
	errorKindChecklists = "checklists"
)

// ErrorEvent is the payload of the "engagement:error" event. Kind is
// "engagement" or "checklists". Path is the file, relative to the engagement
// folder, when the store recorded one (see store.ErrorPath), and empty when unknown.
type ErrorEvent struct {
	Kind    string
	Path    string
	Message string
}

// errorEvents turns err into one ErrorEvent per errors.Join child, with Path
// made relative to baseDir when the error names a file.
func errorEvents(kind, baseDir string, err error) []ErrorEvent {
	children := []error{err}
	var multi interface{ Unwrap() []error }
	if errors.As(err, &multi) {
		children = multi.Unwrap()
	}
	out := make([]ErrorEvent, 0, len(children))
	for _, child := range children {
		ev := ErrorEvent{Kind: kind, Message: child.Error()}
		if p := store.ErrorPath(child); p != "" {
			ev.Path = relativeSlash(baseDir, p)
		}
		out = append(out, ev)
	}
	return out
}

func relativeSlash(base, path string) string {
	rel, err := filepath.Rel(base, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

// emitError emits one engagement:error event per child of err.
func (a *App) emitError(kind, baseDir string, err error) {
	for _, ev := range errorEvents(kind, baseDir, err) {
		a.emitEvent(eventError, ev)
	}
}

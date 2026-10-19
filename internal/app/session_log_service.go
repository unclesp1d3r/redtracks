package app

import (
	"errors"

	"github.com/unclesp1d3r/redtracks/internal/model"
	"github.com/unclesp1d3r/redtracks/internal/sessionlogs"
	"github.com/unclesp1d3r/redtracks/internal/store"
)

// ErrNoNativeDialog is returned by PickLogFile when this build has no native
// file dialog (the server build); the frontend falls back to a typed path.
var ErrNoNativeDialog = errors.New("no native file dialog in this build; type the path instead")

// SessionLogService imports terminal session logs into the open engagement
// and serves them in pages, searches, and clips.
type SessionLogService struct {
	app *App
	idx sessionlogs.Cache
}

// NewSessionLogService returns the session log service bound to a.
func NewSessionLogService(a *App) *SessionLogService {
	s := &SessionLogService{app: a}
	// Indexes are keyed by log id inside one engagement; drop them when another opens.
	a.onEngagementOpened(s.idx.Clear)
	return s
}

// PickLogFile opens the native open-file dialog and returns the chosen
// absolute path, or "" when the operator cancelled.
func (s *SessionLogService) PickLogFile() (string, error) {
	h := s.app.Hooks()
	if !h.Capabilities.NativeDialog || h.PickFile == nil {
		return "", s.app.fail("SessionLogService.PickLogFile", ErrNoNativeDialog)
	}
	path, err := h.PickFile()
	return path, s.app.fail("SessionLogService.PickLogFile", err)
}

// ImportPath copies the regular file at the absolute path into the engagement.
func (s *SessionLogService) ImportPath(path string) (sessionlogs.Source, error) {
	var out sessionlogs.Source
	err := s.app.withStore("SessionLogService.ImportPath", func(st *store.Store) error {
		src, err := sessionlogs.Import(st.Engagement().Dir, path)
		out = src
		return err
	})
	return out, err
}

// List returns the engagement's imported logs.
func (s *SessionLogService) List() ([]sessionlogs.Source, error) {
	var out []sessionlogs.Source
	err := s.app.withStore("SessionLogService.List", func(st *store.Store) error {
		var err error
		out, err = sessionlogs.List(st.Engagement().Dir)
		return err
	})
	return out, err
}

// Page returns count display lines of log id starting at 0-based line from.
func (s *SessionLogService) Page(id string, from, count int) ([]string, error) {
	var out []string
	err := s.withLog("SessionLogService.Page", id, func(l *sessionlogs.Log, _ *store.Store) error {
		var err error
		out, err = l.Page(from, count)
		return err
	})
	return out, err
}

// Search returns lines of log id containing query, after afterLine.
func (s *SessionLogService) Search(id, query string, afterLine, limit int) ([]sessionlogs.Match, error) {
	var out []sessionlogs.Match
	err := s.withLog("SessionLogService.Search", id, func(l *sessionlogs.Log, _ *store.Store) error {
		var err error
		out, err = l.Search(query, afterLine, limit)
		return err
	})
	return out, err
}

// Clip saves lines from..to (inclusive, 0-based) of log id as a text evidence item on host ip.
func (s *SessionLogService) Clip(id, ip string, from, to int) (model.EvidenceItem, error) {
	var out model.EvidenceItem
	err := s.withLog("SessionLogService.Clip", id, func(l *sessionlogs.Log, st *store.Store) error {
		var err error
		out, err = l.Clip(st, ip, l.Source(), from, to)
		return err
	})
	return out, err
}

func (s *SessionLogService) withLog(method, id string, fn func(*sessionlogs.Log, *store.Store) error) error {
	return s.app.withStore(method, func(st *store.Store) error {
		l, err := s.idx.Open(st.Engagement().Dir, id)
		if err != nil {
			return err
		}
		return fn(l, st)
	})
}

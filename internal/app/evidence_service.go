package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/unclesp1d3r/redtracks/internal/evidence"
	"github.com/unclesp1d3r/redtracks/internal/model"
	"github.com/unclesp1d3r/redtracks/internal/store"
)

// maxPendingInboxListings bounds the listing tokens held in memory; the oldest is evicted first.
const maxPendingInboxListings = 8

var (
	errStaleInbox   = errors.New("the engagement changed since this inbox listing; reload the inbox")
	errBadInboxName = errors.New("invalid inbox file name")
)

// removeInboxFile deletes an inbox file; a variable so tests can force a failure.
var removeInboxFile = os.Remove

// EvidenceService exposes the inbox and, later, evidence capture to the frontend.
type EvidenceService struct {
	app     *App
	mu      sync.Mutex
	pending map[string]*store.Store // listing token -> the engagement it was taken under
	order   []string                // tokens, oldest first
}

// NewEvidenceService returns the evidence service bound to a.
func NewEvidenceService(a *App) *EvidenceService {
	return &EvidenceService{app: a, pending: map[string]*store.Store{}}
}

// InboxListing is the settled inbox files plus the token Assign must present.
// Token is empty when no engagement is open.
type InboxListing struct {
	Token string
	Files []evidence.InboxFile
}

// Inbox lists the files that have settled in the inbox.
func (s *EvidenceService) Inbox() (InboxListing, error) {
	l, err := s.inbox()
	return l, s.app.fail("EvidenceService.Inbox", err)
}

func (s *EvidenceService) inbox() (InboxListing, error) {
	empty := InboxListing{Files: []evidence.InboxFile{}}
	st, err := s.app.Store()
	if errors.Is(err, errNoEngagement) {
		return empty, nil
	}
	if err != nil {
		return empty, err
	}
	s.app.mu.Lock()
	ready := slices.Clone(s.app.inboxReady)
	s.app.mu.Unlock()
	files, err := evidence.ListInbox(s.app.inboxDir(), ready)
	if err != nil {
		return empty, err
	}
	return InboxListing{Token: s.issue(st), Files: files}, nil
}

func (s *EvidenceService) issue(st *store.Store) string {
	token := model.NewID()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pending[token] = st
	s.order = append(s.order, token)
	for len(s.order) > maxPendingInboxListings {
		delete(s.pending, s.order[0])
		s.order = s.order[1:]
	}
	return token
}

// Assign copies the inbox file name into the host's evidence at ip, indexes it,
// and only then removes the inbox file. token comes from Inbox and is refused
// once another engagement has been opened.
func (s *EvidenceService) Assign(
	token, name, ip, caption string,
	links []model.EvidenceLink,
) (model.EvidenceItem, error) {
	var item model.EvidenceItem
	err := s.app.withStore("EvidenceService.Assign", func(st *store.Store) error {
		if err := validateInboxName(name); err != nil {
			return err
		}
		s.mu.Lock()
		owner, ok := s.pending[token]
		s.mu.Unlock()
		if !ok {
			return fmt.Errorf("%w %q", errUnknownToken, token)
		}
		if owner != st {
			return errStaleInbox
		}
		path := filepath.Join(s.app.inboxDir(), name)
		added, err := st.ImportEvidenceFile(ip, path, caption, links)
		if err != nil {
			return err
		}
		item = added.Item
		// The copy is indexed; a failed removal only leaves a duplicate listing, never a lost file.
		if rerr := removeInboxFile(path); rerr != nil {
			s.app.log.Error("remove assigned inbox file", "name", name, "err", rerr)
		}
		return nil
	})
	return item, err
}

// DiscardInboxFile deletes an inbox file without keeping a copy.
func (s *EvidenceService) DiscardInboxFile(name string) error {
	err := validateInboxName(name)
	if err == nil {
		if err = removeInboxFile(filepath.Join(s.app.inboxDir(), name)); err != nil {
			err = fmt.Errorf("discard %s: %w", name, err)
		}
	}
	return s.app.fail("EvidenceService.DiscardInboxFile", err)
}

// validateInboxName accepts only a plain file name that cannot leave the inbox.
func validateInboxName(name string) error {
	if name == "" || name == "." || strings.ContainsAny(name, `/\`) || strings.Contains(name, "..") ||
		strings.ContainsRune(name, 0) {
		return fmt.Errorf("%w %q", errBadInboxName, name)
	}
	return nil
}

package app

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"slices"

	"github.com/unclesp1d3r/redtracks/internal/model"
	"github.com/unclesp1d3r/redtracks/internal/progress"
	"github.com/unclesp1d3r/redtracks/internal/store"
)

// EngagementService manages engagements, notes, credentials, users, shares, loot and findings.
type EngagementService struct{ app *App }

// NewEngagementService returns the engagement service bound to a.
func NewEngagementService(a *App) *EngagementService { return &EngagementService{app: a} }

// Notes is a host's notes text with the hash to pass back to WriteNotes.
type Notes struct {
	Text string
	Hash string
}

// List returns the engagements under the root, newest first.
func (s *EngagementService) List() ([]store.Summary, error) {
	out, err := store.ListEngagements(s.app.Root)
	return out, s.app.fail("EngagementService.List", err)
}

// Create makes a new engagement folder and returns its path.
func (s *EngagementService) Create(name, domain string, t model.EngagementType) (string, error) {
	dir, err := store.CreateEngagement(s.app.Root, name, domain, t)
	return dir, s.app.fail("EngagementService.Create", err)
}

// Open switches to the engagement in dir, which must be a direct child of root/engagements.
func (s *EngagementService) Open(dir string) (model.Engagement, error) {
	eng, err := s.open(dir)
	return eng, s.app.fail("EngagementService.Open", err)
}

func (s *EngagementService) open(dir string) (model.Engagement, error) {
	a := s.app
	clean := filepath.Clean(dir)
	if filepath.Dir(clean) != filepath.Join(a.Root, engagementsDir) {
		return model.Engagement{}, fmt.Errorf("engagement %q is not under %s", dir, a.Root)
	}
	st, err := store.Open(clean)
	if err != nil {
		return model.Engagement{}, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.mu.Lock()
	if a.cancelPoll != nil {
		a.cancelPoll()
	}
	a.store, a.cancelPoll = st, cancel
	a.mu.Unlock()
	go st.Poll(ctx, pollInterval,
		func(changed []string) { a.emitEvent(eventChanged, changed) },
		func(err error) {
			a.log.Error("poll engagement", "err", err)
			a.emitEvent(eventError, err.Error())
		})
	return st.Engagement(), nil
}

// Current returns the open engagement.
func (s *EngagementService) Current() (model.Engagement, error) {
	st, err := s.app.Store()
	if err != nil {
		return model.Engagement{}, s.app.fail("EngagementService.Current", err)
	}
	return st.Engagement(), nil
}

// Summaries returns the board rows for the open engagement.
func (s *EngagementService) Summaries() ([]progress.HostSummary, error) {
	st, err := s.app.Store()
	if err != nil {
		return nil, s.app.fail("EngagementService.Summaries", err)
	}
	return progress.Summarize(st.Engagement(), s.app.Definitions()), nil
}

// AttackGraph returns the Mermaid attack graph for the open engagement.
func (s *EngagementService) AttackGraph() (string, error) {
	st, err := s.app.Store()
	if err != nil {
		return "", s.app.fail("EngagementService.AttackGraph", err)
	}
	return progress.AttackGraph(st.Engagement()), nil
}

// ReadNotes returns a host's notes and their hash.
func (s *EngagementService) ReadNotes(ip string) (Notes, error) {
	var n Notes
	err := s.app.withStore("EngagementService.ReadNotes", func(st *store.Store) error {
		var rerr error
		n.Text, n.Hash, rerr = st.ReadNotes(ip)
		return rerr
	})
	return n, err
}

// WriteNotes saves a host's notes; a stale baseHash fails with a "conflict" error.
func (s *EngagementService) WriteNotes(ip, text, baseHash string) error {
	return s.app.withStore("EngagementService.WriteNotes", func(st *store.Store) error {
		return st.WriteNotes(ip, text, baseHash)
	})
}

// MarkTried records that a credential was tried on the host with the given IP (stored by host ID), and that it works when works is set.
func (s *EngagementService) MarkTried(credID, hostIP string, works bool) error {
	return s.app.withStore("EngagementService.MarkTried", func(st *store.Store) error {
		host, ok := st.Engagement().Host(hostIP)
		if !ok {
			return fmt.Errorf("host %s: %w", hostIP, errNotFound)
		}
		return st.MutateCreds(func(c *model.Creds) error {
			next, err := updateByID(c.Credentials, credID, credentialID, func(cr model.Credential) model.Credential {
				cr.TriedOn = appendOnce(cr.TriedOn, host.ID)
				if works {
					cr.WorksOn = appendOnce(cr.WorksOn, host.ID)
				}
				return cr
			})
			c.Credentials = next
			return err
		})
	})
}

func appendOnce(list []string, v string) []string {
	if slices.Contains(list, v) {
		return list
	}
	return append(slices.Clone(list), v)
}

var errNotFound = errors.New("not found")

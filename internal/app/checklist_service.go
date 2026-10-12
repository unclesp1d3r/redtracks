package app

import (
	"github.com/unclesp1d3r/redtracks/internal/checklist"
)

// ChecklistService exposes checklist definitions and command substitution.
type ChecklistService struct{ app *App }

// NewChecklistService returns the checklist service bound to a.
func NewChecklistService(a *App) *ChecklistService { return &ChecklistService{app: a} }

// Definitions returns the loaded checklist definitions.
func (s *ChecklistService) Definitions() ([]checklist.Definition, error) {
	return s.app.Definitions(), nil
}

// Reload re-reads the definitions from disk now.
func (s *ChecklistService) Reload() error {
	return s.app.fail("ChecklistService.Reload", s.app.reloadDefs())
}

// Command substitutes $T and $D in cmd.
func (s *ChecklistService) Command(cmd, ip, domain string) (string, error) {
	out, err := checklist.Substitute(cmd, ip, domain)
	return out, s.app.fail("ChecklistService.Command", err)
}

package app

import (
	"github.com/unclesp1d3r/redtracks/internal/checklist"
)

// ChecklistService exposes checklist definitions and command substitution.
type ChecklistService struct{ app *App }

// NewChecklistService returns the checklist service bound to a.
func NewChecklistService(a *App) *ChecklistService { return &ChecklistService{app: a} }

// DefinitionsStatus reports how the checklist definitions loaded. Error is the
// last load or watch failure ("" when the current set loaded cleanly) and
// InstallError is the failure to install the default set at startup.
// InstalledFiles lists the bundled files this launch added to the checklists
// folder (every file on a first run, only the newly shipped ones after).
type DefinitionsStatus struct {
	Count          int
	Error          string
	InstallError   string
	InstalledFiles []string
}

// Status reports the loaded definition count and any load or install error.
func (s *ChecklistService) Status() DefinitionsStatus {
	return s.app.definitionsStatus()
}

// Command substitutes $T and $D in cmd.
func (s *ChecklistService) Command(cmd, ip, domain string) (string, error) {
	out, err := checklist.Substitute(cmd, ip, domain)
	return out, s.app.fail("ChecklistService.Command", err)
}

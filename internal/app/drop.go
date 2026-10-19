package app

import (
	"errors"
	"strings"

	"github.com/unclesp1d3r/redtracks/internal/model"
)

// EventDropped carries a DropResult to the frontend after a native file drop.
const EventDropped = "evidence:dropped"

// Data attributes the frontend sets on a drop target; the window hands them to Go with the dropped paths.
const (
	attrEvidenceHost   = "data-evidence-host"
	attrEvidenceTarget = "data-evidence-target"
	attrEvidenceKind   = "data-evidence-kind"
	dropKindImport     = "import"
	// targetParts is kind, target id and an optional service id.
	targetParts = 3
)

var (
	errOutsideTarget = errors.New("drop landed outside a target")
	errBadTarget     = errors.New("drop target is malformed")
	errOneImport     = errors.New("drop one file at a time onto the import zone")
)

// DropAdded is a dropped file that became an evidence item.
type DropAdded struct {
	Path      string
	Item      model.EvidenceItem
	Duplicate bool
}

// DropRefusal is a dropped path that was not accepted, with the reason.
type DropRefusal struct {
	Path   string
	Reason string
}

// DropResult is the outcome of one native drop.
type DropResult struct {
	Added   []DropAdded
	Refused []DropRefusal
	Preview *PreviewResponse // set when a file was dropped on the import zone
}

// Dropper turns native drops into evidence items or an import preview. It
// holds no Wails types, so tests drive it directly.
type Dropper struct {
	app    *App
	Import *ImportService
}

// NewDropper builds a Dropper over the app and its import service.
func NewDropper(a *App, imp *ImportService) *Dropper { return &Dropper{app: a, Import: imp} }

// HandleDrop copies each dropped path into the target host's evidence (the
// sources are never touched), or previews one path for the import zone.
func (d *Dropper) HandleDrop(paths []string, attrs map[string]string) DropResult {
	if attrs[attrEvidenceKind] == dropKindImport {
		return d.previewImport(paths)
	}
	host := attrs[attrEvidenceHost]
	if host == "" {
		return refuseAll(paths, errOutsideTarget)
	}
	links, err := parseTarget(attrs[attrEvidenceTarget])
	if err != nil {
		return refuseAll(paths, err)
	}
	return d.addEvidence(host, links, paths)
}

func (d *Dropper) previewImport(paths []string) DropResult {
	if len(paths) != 1 {
		return refuseAll(paths, errOneImport)
	}
	resp, err := d.Import.PreviewPath(paths[0])
	if err != nil {
		return DropResult{Refused: []DropRefusal{{Path: paths[0], Reason: err.Error()}}}
	}
	return DropResult{Preview: &resp}
}

func (d *Dropper) addEvidence(host string, links []model.EvidenceLink, paths []string) DropResult {
	st, err := d.app.Store()
	if err != nil {
		return refuseAll(paths, d.app.fail("Dropper.HandleDrop", err))
	}
	var res DropResult
	for _, p := range paths {
		added, err := st.ImportEvidenceFile(host, p, "", links)
		if err != nil {
			res.Refused = append(
				res.Refused,
				DropRefusal{Path: p, Reason: d.app.fail("Dropper.HandleDrop", err).Error()},
			)
			continue
		}
		res.Added = append(res.Added, DropAdded{Path: p, Item: added.Item, Duplicate: added.Duplicate})
	}
	return res
}

// refuseAll answers a drop that cannot be processed at all: one refusal, not one per path.
func refuseAll(paths []string, reason error) DropResult {
	r := DropRefusal{Reason: reason.Error()}
	if len(paths) == 1 {
		r.Path = paths[0]
	}
	return DropResult{Refused: []DropRefusal{r}}
}

// parseTarget reads "<kind>:<targetId>[:<serviceId>]"; empty means host only.
func parseTarget(target string) ([]model.EvidenceLink, error) {
	if target == "" {
		return nil, nil
	}
	parts := strings.SplitN(target, ":", targetParts)
	link := model.EvidenceLink{Kind: model.EvidenceLinkKind(parts[0])}
	if len(parts) < 2 || parts[1] == "" || !link.Kind.IsValid() {
		return nil, errBadTarget
	}
	link.TargetID = parts[1]
	if len(parts) == targetParts {
		link.ServiceID = parts[2]
	}
	return []model.EvidenceLink{link}, nil
}

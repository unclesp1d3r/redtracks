package store

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/unclesp1d3r/redtracks/internal/model"
)

// checkMeta rejects an engagement.json that would break the model's invariants.
func checkMeta(path string, m model.EngagementMeta) error {
	if !m.Type.IsValid() {
		return fmt.Errorf("%s: invalid engagement type %q", path, m.Type)
	}
	seen := map[string]bool{}
	for _, ft := range m.FlagTypes {
		switch {
		case ft.ID == "":
			return fmt.Errorf("%s: flag type %q has an empty id", path, ft.Label)
		case seen[ft.ID]:
			return fmt.Errorf("%s: duplicate flag type id %q", path, ft.ID)
		case ft.Points < 0:
			return fmt.Errorf("%s: flag type %q has negative points %d", path, ft.ID, ft.Points)
		}
		seen[ft.ID] = true
	}
	return nil
}

// checkHost rejects a host.json with bad enums or duplicate identifiers.
func checkHost(path string, h model.Host) error {
	if !h.Status.IsValid() {
		return fmt.Errorf("%s: invalid host status %q", path, h.Status)
	}
	if err := checkItemStates(path, "host checklist", h.Checklist); err != nil {
		return err
	}
	serviceIDs := map[string]bool{}
	for _, svc := range h.Services {
		if serviceIDs[svc.ID] {
			return fmt.Errorf("%s: duplicate service id %q", path, svc.ID)
		}
		serviceIDs[svc.ID] = true
		if err := checkItemStates(path, "service "+svc.ID+" checklist", svc.Checklist); err != nil {
			return err
		}
	}
	flagTypes := map[string]bool{}
	for _, f := range h.Flags {
		if flagTypes[f.TypeID] {
			return fmt.Errorf("%s: duplicate flag for type %q", path, f.TypeID)
		}
		flagTypes[f.TypeID] = true
	}
	return nil
}

func checkItemStates(path, where string, c model.ChecklistState) error {
	for id, st := range c.Items {
		if st.Status != "" && !st.Status.IsValid() {
			return fmt.Errorf("%s: %s item %q has invalid status %q", path, where, id, st.Status)
		}
	}
	return nil
}

// checkEvidence rejects an evidence.json with duplicate ids, a file name that
// is not a bare base name, or an unknown kind. Link targets are not checked:
// a link to a record that no longer exists is tolerated.
func checkEvidence(path string, f model.EvidenceFile) error {
	seen := map[string]bool{}
	for _, it := range f.Items {
		switch {
		case it.ID == "":
			return fmt.Errorf("%s: evidence item %q has an empty id", path, it.File)
		case seen[it.ID]:
			return fmt.Errorf("%s: duplicate evidence id %q", path, it.ID)
		case !isBareName(it.File):
			return fmt.Errorf("%s: evidence %q has file %q, which is not a bare file name", path, it.ID, it.File)
		case !it.Kind.IsValid():
			return fmt.Errorf("%s: evidence %q has invalid kind %q", path, it.ID, it.Kind)
		}
		seen[it.ID] = true
		for _, l := range it.Links {
			if !l.Kind.IsValid() {
				return fmt.Errorf("%s: evidence %q has a link with invalid kind %q", path, it.ID, l.Kind)
			}
		}
	}
	return nil
}

func isBareName(name string) bool {
	return name != "" && name != "." && name != ".." && !strings.ContainsAny(name, `/\`) && filepath.Base(name) == name
}

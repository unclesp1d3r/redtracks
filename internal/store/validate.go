package store

import (
	"fmt"

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

package app

import (
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/unclesp1d3r/redtracks/internal/checklist"
	"github.com/unclesp1d3r/redtracks/internal/model"
	"github.com/unclesp1d3r/redtracks/internal/store"
)

// HostService edits hosts, their services, access paths, flags and checklist state.
type HostService struct{ app *App }

// NewHostService returns the host service bound to a.
func NewHostService(a *App) *HostService { return &HostService{app: a} }

// ChecklistView is one matching definition with the recorded state of each item.
type ChecklistView struct {
	Definition checklist.Definition
	Items      []checklist.ItemView
}

func (s *HostService) mutate(method, ip string, fn func(*model.Host) error) error {
	return s.app.withStore(method, func(st *store.Store) error { return st.MutateHost(ip, fn) })
}

// Add creates a host.
func (s *HostService) Add(ip, hostname string) error {
	h := model.Host{
		ID: model.NewID(), IP: ip, Hostname: hostname, Status: model.HostStatusUntouched,
	}.Normalize()
	return s.app.withStore("HostService.Add", func(st *store.Store) error { return st.AddHost(h) })
}

// Update replaces Hostname and OS only; SetStatus owns Status, and no screen
// edits Tags yet, so a stale client copy must not overwrite them.
func (s *HostService) Update(h model.Host) error {
	return s.mutate("HostService.Update", h.IP, func(cur *model.Host) error {
		cur.Hostname, cur.OS = h.Hostname, h.OS
		return nil
	})
}

// SetStatus sets a host's status.
func (s *HostService) SetStatus(ip string, st model.HostStatus) error {
	if !st.IsValid() {
		return s.app.fail("HostService.SetStatus", fmt.Errorf("invalid host status %q", st))
	}
	return s.mutate("HostService.SetStatus", ip, func(cur *model.Host) error {
		cur.Status = st
		return nil
	})
}

// AddService adds a service, assigning an ID when empty.
func (s *HostService) AddService(ip string, svc model.Service) error {
	svc.ID = withNewID(svc.ID)
	svc = hostWithService(svc).Services[0]
	return s.mutate("HostService.AddService", ip, func(cur *model.Host) error {
		cur.Services = append(slices.Clone(cur.Services), svc)
		return nil
	})
}

// hostWithService normalises a service's nil collections via Host.Normalize.
func hostWithService(svc model.Service) model.Host {
	return model.Host{Services: []model.Service{svc}}.Normalize()
}

// UpdateService replaces scalar fields only; WebPaths and Checklist are kept.
func (s *HostService) UpdateService(ip string, svc model.Service) error {
	return s.mutate("HostService.UpdateService", ip, func(cur *model.Host) error {
		next, err := updateByID(cur.Services, svc.ID, svcID, func(old model.Service) model.Service {
			svc.WebPaths, svc.Checklist = old.WebPaths, old.Checklist
			return svc
		})
		cur.Services = next
		return err
	})
}

func svcID(s model.Service) string           { return s.ID }
func accessPathID(p model.AccessPath) string { return p.ID }

// AddAccessPath adds an access path, assigning an ID when empty.
func (s *HostService) AddAccessPath(ip string, p model.AccessPath) error {
	p.ID = withNewID(p.ID)
	return s.mutate("HostService.AddAccessPath", ip, func(cur *model.Host) error {
		cur.AccessPaths = append(slices.Clone(cur.AccessPaths), p)
		return nil
	})
}

// RemoveAccessPath deletes an access path.
func (s *HostService) RemoveAccessPath(ip, pathID string) error {
	return s.mutate("HostService.RemoveAccessPath", ip, func(cur *model.Host) error {
		next, err := deleteByID(cur.AccessPaths, pathID, accessPathID)
		cur.AccessPaths = next
		return err
	})
}

// SetFlag replaces the flag with the same TypeID or appends it.
func (s *HostService) SetFlag(ip string, f model.Flag) error {
	return s.mutate("HostService.SetFlag", ip, func(cur *model.Host) error {
		next := slices.Clone(cur.Flags)
		if i := slices.IndexFunc(next, func(x model.Flag) bool { return x.TypeID == f.TypeID }); i >= 0 {
			next[i] = f
		} else {
			next = append(next, f)
		}
		cur.Flags = next
		return nil
	})
}

// SetItem records a checklist item's state; serviceID "" targets the host checklist.
func (s *HostService) SetItem(ip, serviceID, itemID string, st model.ItemStatus, note string) error {
	if !st.IsValid() {
		return s.app.fail("HostService.SetItem", fmt.Errorf("invalid item status %q", st))
	}
	state := model.ItemState{Status: st, Note: note, UpdatedAt: time.Now().UTC().Format(time.RFC3339)}
	return s.mutate("HostService.SetItem", ip, func(cur *model.Host) error {
		if serviceID == "" {
			cur.Checklist = withItem(cur.Checklist, itemID, state)
			return nil
		}
		next, err := updateByID(cur.Services, serviceID, svcID, func(old model.Service) model.Service {
			old.Checklist = withItem(old.Checklist, itemID, state)
			return old
		})
		cur.Services = next
		return err
	})
}

func withItem(c model.ChecklistState, itemID string, state model.ItemState) model.ChecklistState {
	items := maps.Clone(c.Items)
	if items == nil {
		items = map[string]model.ItemState{}
	}
	items[itemID] = state
	return model.ChecklistState{Items: items}
}

// Checklists returns the definitions matching the host (serviceID "") or a service, with item state.
func (s *HostService) Checklists(ip, serviceID string) ([]ChecklistView, error) {
	st, err := s.app.Store()
	if err != nil {
		return nil, s.app.fail("HostService.Checklists", err)
	}
	host, ok := st.Engagement().Host(ip)
	if !ok {
		return nil, s.app.fail("HostService.Checklists", fmt.Errorf("host %s: %w", ip, errNotFound))
	}
	defs, state := checklist.ForHost(s.app.Definitions(), host), host.Checklist
	if serviceID != "" {
		svc, found := host.Service(serviceID)
		if !found {
			return nil, s.app.fail("HostService.Checklists", fmt.Errorf("service %s: %w", serviceID, errNotFound))
		}
		defs, state = checklist.ForService(s.app.Definitions(), svc), svc.Checklist
	}
	views := make([]ChecklistView, len(defs))
	for i, d := range defs {
		views[i] = ChecklistView{Definition: d, Items: checklist.Merge(d, state)}
	}
	return views, nil
}

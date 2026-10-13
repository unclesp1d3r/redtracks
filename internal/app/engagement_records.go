package app

import (
	"fmt"
	"slices"
	"strings"

	"github.com/unclesp1d3r/redtracks/internal/model"
	"github.com/unclesp1d3r/redtracks/internal/store"
)

func credentialID(c model.Credential) string { return c.ID }
func userID(u model.User) string             { return u.ID }
func shareID(s model.Share) string           { return s.ID }
func lootID(l model.Loot) string             { return l.ID }
func findingID(f model.Finding) string       { return f.ID }

// updateByID returns a copy of list with the element whose ID is id passed through fn.
func updateByID[T any](list []T, id string, idOf func(T) string, fn func(T) T) ([]T, error) {
	i := slices.IndexFunc(list, func(v T) bool { return idOf(v) == id })
	if i < 0 {
		return list, fmt.Errorf("record %q: %w", id, errNotFound)
	}
	out := slices.Clone(list)
	out[i] = fn(out[i])
	return out, nil
}

func deleteByID[T any](list []T, id string, idOf func(T) string) ([]T, error) {
	i := slices.IndexFunc(list, func(v T) bool { return idOf(v) == id })
	if i < 0 {
		return list, fmt.Errorf("record %q: %w", id, errNotFound)
	}
	return slices.Delete(slices.Clone(list), i, i+1), nil
}

func withNewID(id string) string {
	if id == "" {
		return model.NewID()
	}
	return id
}

func (s *EngagementService) mutateCreds(method string, fn func(*model.Creds) error) error {
	return s.app.withStore(method, func(st *store.Store) error { return st.MutateCreds(fn) })
}

// AddCredential adds a credential, assigning an ID when empty.
func (s *EngagementService) AddCredential(c model.Credential) error {
	c.ID = withNewID(c.ID)
	return s.mutateCreds("EngagementService.AddCredential", func(cs *model.Creds) error {
		cs.Credentials = append(slices.Clone(cs.Credentials), c)
		return nil
	})
}

// UpdateCredential replaces scalar fields only; WorksOn and TriedOn are kept.
func (s *EngagementService) UpdateCredential(c model.Credential) error {
	return s.mutateCreds("EngagementService.UpdateCredential", func(cs *model.Creds) error {
		next, err := updateByID(cs.Credentials, c.ID, credentialID, func(old model.Credential) model.Credential {
			c.WorksOn, c.TriedOn = old.WorksOn, old.TriedOn
			return c
		})
		cs.Credentials = next
		return err
	})
}

// DeleteCredential removes a credential.
func (s *EngagementService) DeleteCredential(id string) error {
	return s.mutateCreds("EngagementService.DeleteCredential", func(cs *model.Creds) error {
		next, err := deleteByID(cs.Credentials, id, credentialID)
		cs.Credentials = next
		return err
	})
}

// AddUser adds a user, assigning an ID when empty.
func (s *EngagementService) AddUser(u model.User) error {
	u.ID = withNewID(u.ID)
	return s.mutateCreds("EngagementService.AddUser", func(cs *model.Creds) error {
		cs.Users = append(slices.Clone(cs.Users), u)
		return nil
	})
}

// UpdateUser replaces a user's fields.
func (s *EngagementService) UpdateUser(u model.User) error {
	return s.mutateCreds("EngagementService.UpdateUser", func(cs *model.Creds) error {
		next, err := updateByID(cs.Users, u.ID, userID, func(model.User) model.User { return u })
		cs.Users = next
		return err
	})
}

// DeleteUser removes a user.
func (s *EngagementService) DeleteUser(id string) error {
	return s.mutateCreds("EngagementService.DeleteUser", func(cs *model.Creds) error {
		next, err := deleteByID(cs.Users, id, userID)
		cs.Users = next
		return err
	})
}

// AddShare adds a share, assigning an ID when empty.
func (s *EngagementService) AddShare(sh model.Share) error {
	sh.ID = withNewID(sh.ID)
	return s.mutateCreds("EngagementService.AddShare", func(cs *model.Creds) error {
		cs.Shares = append(slices.Clone(cs.Shares), sh)
		return nil
	})
}

// UpdateShare replaces a share's fields.
func (s *EngagementService) UpdateShare(sh model.Share) error {
	return s.mutateCreds("EngagementService.UpdateShare", func(cs *model.Creds) error {
		next, err := updateByID(cs.Shares, sh.ID, shareID, func(model.Share) model.Share { return sh })
		cs.Shares = next
		return err
	})
}

// DeleteShare removes a share.
func (s *EngagementService) DeleteShare(id string) error {
	return s.mutateCreds("EngagementService.DeleteShare", func(cs *model.Creds) error {
		next, err := deleteByID(cs.Shares, id, shareID)
		cs.Shares = next
		return err
	})
}

// AddLoot adds a loot entry, assigning an ID when empty.
func (s *EngagementService) AddLoot(l model.Loot) error {
	l.ID = withNewID(l.ID)
	return s.mutateCreds("EngagementService.AddLoot", func(cs *model.Creds) error {
		cs.Loot = append(slices.Clone(cs.Loot), l)
		return nil
	})
}

// UpdateLoot replaces a loot entry's fields.
func (s *EngagementService) UpdateLoot(l model.Loot) error {
	return s.mutateCreds("EngagementService.UpdateLoot", func(cs *model.Creds) error {
		next, err := updateByID(cs.Loot, l.ID, lootID, func(model.Loot) model.Loot { return l })
		cs.Loot = next
		return err
	})
}

// DeleteLoot removes a loot entry.
func (s *EngagementService) DeleteLoot(id string) error {
	return s.mutateCreds("EngagementService.DeleteLoot", func(cs *model.Creds) error {
		next, err := deleteByID(cs.Loot, id, lootID)
		cs.Loot = next
		return err
	})
}

func (s *EngagementService) mutateFindings(method string, fn func(*model.FindingsFile) error) error {
	return s.app.withStore(method, func(st *store.Store) error { return st.MutateFindings(fn) })
}

// checkFinding lowercases the severity and requires the finding's host to exist.
func (s *EngagementService) checkFinding(method string, f model.Finding) (model.Finding, error) {
	f.Severity = strings.ToLower(strings.TrimSpace(f.Severity))
	st, err := s.app.Store()
	if err != nil {
		return f, s.app.fail(method, err)
	}
	if _, ok := st.Engagement().HostByID(f.HostID); !ok {
		return f, s.app.fail(method, fmt.Errorf("finding host %q: %w", f.HostID, errNotFound))
	}
	return f, nil
}

// AddFinding adds a finding, assigning an ID when empty. Severity is lowercased
// and HostID must name a host in the engagement.
func (s *EngagementService) AddFinding(f model.Finding) error {
	f, err := s.checkFinding("EngagementService.AddFinding", f)
	if err != nil {
		return err
	}
	f.ID = withNewID(f.ID)
	return s.mutateFindings("EngagementService.AddFinding", func(ff *model.FindingsFile) error {
		ff.Findings = append(slices.Clone(ff.Findings), f)
		return nil
	})
}

// UpdateFinding replaces a finding's fields, with the same checks as AddFinding.
func (s *EngagementService) UpdateFinding(f model.Finding) error {
	f, err := s.checkFinding("EngagementService.UpdateFinding", f)
	if err != nil {
		return err
	}
	return s.mutateFindings("EngagementService.UpdateFinding", func(ff *model.FindingsFile) error {
		next, err := updateByID(ff.Findings, f.ID, findingID, func(model.Finding) model.Finding { return f })
		ff.Findings = next
		return err
	})
}

// DeleteFinding removes a finding.
func (s *EngagementService) DeleteFinding(id string) error {
	return s.mutateFindings("EngagementService.DeleteFinding", func(ff *model.FindingsFile) error {
		next, err := deleteByID(ff.Findings, id, findingID)
		ff.Findings = next
		return err
	})
}

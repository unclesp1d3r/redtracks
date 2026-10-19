// Package progress computes the "what's left" summaries, board ordering, and
// the Mermaid attack graph from a loaded engagement. Everything here is pure.
package progress

import (
	"slices"
	"sort"

	"github.com/unclesp1d3r/redtracks/internal/checklist"
	"github.com/unclesp1d3r/redtracks/internal/model"
)

// HostSummary is what is left to do on one host.
type HostSummary struct {
	HostID, IP, Hostname string
	Status               model.HostStatus
	MissingFlags         []string // flag type IDs
	UntouchedServices    []string // "80/tcp http"
	OpenItems            int
	UntriedCreds         []string // credential IDs
	NoAccessPath         bool
	FlagPoints           int
	TotalPoints          int
	ItemsDone            int
	ItemsTotal           int
	MissingSlots         []SlotRef // proof slots still owed, host-level first
	OwedShare            float64   // missing flags and slots over expected flags and owed slots; 0 when nothing is owed
}

// SlotRef is one proof slot a host still owes. ServiceID and ServiceLabel are
// empty for host-level slots.
type SlotRef struct {
	DefinitionID string
	SlotID       string
	Title        string
	Hint         string
	ServiceID    string
	ServiceLabel string
}

// itemCounts tallies checklist items by state.
type itemCounts struct {
	todo, done, total int
}

func (c itemCounts) add(o itemCounts) itemCounts {
	return itemCounts{todo: c.todo + o.todo, done: c.done + o.done, total: c.total + o.total}
}

// Summarize returns one summary per host in board order: most proof owed
// first (missing flags and slots as a per-item share), then lowest checklist
// completion, ties broken by IP. The input is
// not modified.
func Summarize(eng model.Engagement, defs []checklist.Definition) []HostSummary {
	out := make([]HostSummary, len(eng.Hosts))
	for i, h := range eng.Hosts {
		out[i] = summarizeHost(eng, defs, h)
	}
	sort.SliceStable(out, func(i, j int) bool { return boardLess(out[i], out[j]) })
	return out
}

func summarizeHost(eng model.Engagement, defs []checklist.Definition, h model.Host) HostSummary {
	counts := countState(checklist.ForHost(defs, h), h.Checklist)
	untouched := []string{}
	for _, s := range h.Services {
		svcCounts := countState(checklist.ForService(defs, s), s.Checklist)
		counts = counts.add(svcCounts)
		if svcCounts.total > 0 && svcCounts.todo == svcCounts.total {
			untouched = append(untouched, s.Label())
		}
	}
	flags := flagProgress(eng.Meta.FlagTypes, h)
	slots := slotProgress(defs, h, eng.Evidence[h.IP].Items)
	return HostSummary{
		HostID:            h.ID,
		IP:                h.IP,
		Hostname:          h.Hostname,
		Status:            h.Status,
		MissingFlags:      flags.missing,
		UntouchedServices: untouched,
		OpenItems:         counts.todo,
		UntriedCreds:      untriedCreds(eng.Creds.Credentials, h.ID),
		NoAccessPath:      h.Status != model.HostStatusUntouched && len(h.AccessPaths) == 0,
		FlagPoints:        flags.points,
		TotalPoints:       flags.total,
		ItemsDone:         counts.done,
		ItemsTotal:        counts.total,
		MissingSlots:      slots.missing,
		OwedShare:         share(len(flags.missing)+len(slots.missing), len(eng.Meta.FlagTypes)+slots.owed),
	}
}

type slotTally struct {
	missing []SlotRef
	owed    int
}

// slotProgress lists the slots h owes that no evidence item fills: host-level
// slots first, then each service's slots once per unfilled service.
func slotProgress(defs []checklist.Definition, h model.Host, items []model.EvidenceItem) slotTally {
	tally := slotTally{missing: []SlotRef{}}
	tally.add(checklist.SlotsForHost(defs, h), checklist.FilledSlots(items, ""), model.Service{})
	for _, s := range h.Services {
		owed := checklist.SlotsForService(defs, s)
		if len(owed) == 0 {
			continue
		}
		tally.add(owed, checklist.FilledSlots(items, s.ID), s)
	}
	return tally
}

func (t *slotTally) add(owed []checklist.OwedSlot, filled map[string]bool, svc model.Service) {
	t.owed += len(owed)
	for _, slot := range owed {
		if filled[slot.ID] {
			continue
		}
		ref := SlotRef{DefinitionID: slot.DefinitionID, SlotID: slot.ID, Title: slot.Title, Hint: slot.Hint}
		if svc.ID != "" {
			ref.ServiceID, ref.ServiceLabel = svc.ID, svc.Label()
		}
		t.missing = append(t.missing, ref)
	}
}

func countState(defs []checklist.Definition, state model.ChecklistState) itemCounts {
	var c itemCounts
	for _, def := range defs {
		if !def.HasItems() {
			continue
		}
		for _, view := range checklist.Merge(def, state) {
			c.total++
			switch view.State.Status {
			case model.ItemTodo:
				c.todo++
			case model.ItemDone:
				c.done++
			case model.ItemSkipped, model.ItemNA:
			}
		}
	}
	return c
}

type flagTally struct {
	missing       []string
	points, total int
}

// flagProgress tallies the flag types with no value on h and the points
// captured versus available.
func flagProgress(types []model.FlagType, h model.Host) flagTally {
	tally := flagTally{missing: []string{}}
	for _, ft := range types {
		tally.total += ft.Points
		if hasFlag(h, ft.ID) {
			tally.points += ft.Points
		} else {
			tally.missing = append(tally.missing, ft.ID)
		}
	}
	return tally
}

func hasFlag(h model.Host, typeID string) bool {
	for _, f := range h.Flags {
		if f.TypeID == typeID && f.Value != "" {
			return true
		}
	}
	return false
}

func untriedCreds(creds []model.Credential, hostID string) []string {
	out := []string{}
	for _, c := range creds {
		if c.FoundOnHostID == hostID || slices.Contains(c.TriedOn, hostID) || slices.Contains(c.WorksOn, hostID) {
			continue
		}
		out = append(out, c.ID)
	}
	return out
}

func boardLess(a, b HostSummary) bool {
	if a.OwedShare != b.OwedShare {
		return a.OwedShare > b.OwedShare
	}
	if ra, rb := ratio(a.ItemsDone, a.ItemsTotal), ratio(b.ItemsDone, b.ItemsTotal); ra != rb {
		return ra < rb
	}
	return a.IP < b.IP
}

// ratio is num/den, with a zero denominator treated as 1: no flags or no
// items means nothing is outstanding, so such hosts sort last on the board.
func ratio(num, den int) float64 {
	if den == 0 {
		return 1
	}
	return float64(num) / float64(den)
}

// share is num/den with a zero denominator treated as 0: a host that owes no
// proof sorts after every host that does.
func share(num, den int) float64 {
	if den == 0 {
		return 0
	}
	return float64(num) / float64(den)
}

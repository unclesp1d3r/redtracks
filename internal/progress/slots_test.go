package progress_test

import (
	"slices"
	"testing"

	"github.com/unclesp1d3r/redtracks/internal/checklist"
	"github.com/unclesp1d3r/redtracks/internal/model"
	"github.com/unclesp1d3r/redtracks/internal/progress"
)

func hostSlotDef() checklist.Definition {
	return checklist.Definition{
		ID:    "proofs",
		Match: checklist.Match{HostOS: []string{"*"}},
		Slots: []checklist.Slot{{ID: "proofs.shell", Title: "Interactive shell", Hint: "prompt and whoami"}},
	}
}

func serviceSlotDef() checklist.Definition {
	return checklist.Definition{
		ID:    "httpslots",
		Match: checklist.Match{ServiceTypes: []string{"http"}},
		Slots: []checklist.Slot{{ID: "httpslots.shell", Title: "Web shell"}},
	}
}

func slotEvidence(serviceID string, slotIDs ...string) model.EvidenceFile {
	items := make([]model.EvidenceItem, len(slotIDs))
	for i, id := range slotIDs {
		items[i] = model.EvidenceItem{ID: "e" + id, Links: []model.EvidenceLink{
			{Kind: model.EvidenceLinkSlot, TargetID: id, ServiceID: serviceID},
		}}
	}
	return model.EvidenceFile{Items: items}
}

func boardIDs(got []progress.HostSummary) []string {
	ids := make([]string, len(got))
	for i, s := range got {
		ids[i] = s.HostID
	}
	return ids
}

// AE4: identical checklist completion, no outstanding flags, one missing slot.
func TestMissingSlotSortsHostFirst(t *testing.T) {
	eng := model.Engagement{
		Hosts: []model.Host{
			{ID: "proven", IP: "10.0.0.1"},
			{ID: "owes", IP: "10.0.0.2"},
		},
		Evidence: map[string]model.EvidenceFile{"10.0.0.1": slotEvidence("", "proofs.shell")},
	}

	got := progress.Summarize(eng, []checklist.Definition{hostSlotDef()})

	if want := []string{"owes", "proven"}; !slices.Equal(boardIDs(got), want) {
		t.Fatalf("order = %v, want %v", boardIDs(got), want)
	}
	owes := summaryFor(t, got, "owes")
	want := []progress.SlotRef{
		{DefinitionID: "proofs", SlotID: "proofs.shell", Title: "Interactive shell", Hint: "prompt and whoami"},
	}
	if !slices.Equal(owes.MissingSlots, want) || owes.OwedShare != 1 {
		t.Fatalf("owes = %+v", owes)
	}
	if proven := summaryFor(t, got, "proven"); len(proven.MissingSlots) != 0 || proven.OwedShare != 0 {
		t.Fatalf("proven = %+v", proven)
	}
}

func TestAllProofCapturedSortsAfterMissingFlag(t *testing.T) {
	eng := model.Engagement{
		Meta: model.EngagementMeta{FlagTypes: []model.FlagType{{ID: "user", Points: 10}, {ID: "root", Points: 20}}},
		Hosts: []model.Host{
			{
				ID:    "complete",
				IP:    "10.0.0.1",
				Flags: []model.Flag{{TypeID: "user", Value: "v"}, {TypeID: "root", Value: "v"}},
			},
			{ID: "oneflag", IP: "10.0.0.2", Flags: []model.Flag{{TypeID: "user", Value: "v"}}},
		},
		Evidence: map[string]model.EvidenceFile{
			"10.0.0.1": slotEvidence("", "proofs.shell"),
			"10.0.0.2": slotEvidence("", "proofs.shell"),
		},
	}

	got := progress.Summarize(eng, []checklist.Definition{hostSlotDef()})

	if want := []string{"oneflag", "complete"}; !slices.Equal(boardIDs(got), want) {
		t.Fatalf("order = %v, want %v", boardIDs(got), want)
	}
	// One of three proofs is missing, counted per item rather than by points.
	if share := summaryFor(t, got, "oneflag").OwedShare; share < 0.33 || share > 0.34 {
		t.Fatalf("owedShare = %v, want 1/3", share)
	}
}

func TestNothingOwedIsZeroShareThenChecklistOrder(t *testing.T) {
	svc := func(c model.ChecklistState) []model.Service {
		return []model.Service{{ID: "s", Type: "http", Checklist: c}}
	}
	done := model.ChecklistState{Items: map[string]model.ItemState{"http-1": {Status: model.ItemDone}}}
	eng := model.Engagement{
		Meta: model.EngagementMeta{FlagTypes: []model.FlagType{{ID: "user", Points: 10}}},
		Hosts: []model.Host{
			{ID: "halfdone", IP: "10.0.0.1", Services: svc(done), Flags: []model.Flag{{TypeID: "user", Value: "v"}}},
			{
				ID:       "fresh",
				IP:       "10.0.0.2",
				Services: svc(model.ChecklistState{}),
				Flags:    []model.Flag{{TypeID: "user", Value: "v"}},
			},
			{ID: "owes", IP: "10.0.0.3", Services: svc(done)},
		},
	}

	got := progress.Summarize(eng, []checklist.Definition{httpDef()})

	if want := []string{"owes", "fresh", "halfdone"}; !slices.Equal(boardIDs(got), want) {
		t.Fatalf("order = %v, want %v", boardIDs(got), want)
	}
	if fresh := summaryFor(t, got, "fresh"); fresh.OwedShare != 0 {
		t.Fatalf("fresh owedShare = %v, want 0", fresh.OwedShare)
	}
}

func TestServiceSlotListedOncePerUnfilledService(t *testing.T) {
	host := model.Host{ID: "h", IP: "10.0.0.1", Services: []model.Service{
		{ID: "web80", Port: 80, Protocol: "tcp", Type: "http"},
		{ID: "web8080", Port: 8080, Protocol: "tcp", Type: "http"},
		{ID: "ssh", Port: 22, Protocol: "tcp", Type: "ssh"},
	}}
	eng := model.Engagement{
		Hosts:    []model.Host{host},
		Evidence: map[string]model.EvidenceFile{"10.0.0.1": slotEvidence("web80", "httpslots.shell")},
	}

	got := progress.Summarize(eng, []checklist.Definition{serviceSlotDef()})

	want := []progress.SlotRef{{
		DefinitionID: "httpslots", SlotID: "httpslots.shell", Title: "Web shell",
		ServiceID: "web8080", ServiceLabel: "8080/tcp http",
	}}
	if s := got[0]; !slices.Equal(s.MissingSlots, want) || s.OwedShare != 0.5 {
		t.Fatalf("summary = %+v, want missing %+v and share 0.5", s, want)
	}
}

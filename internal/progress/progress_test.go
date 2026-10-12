package progress_test

import (
	"slices"
	"testing"

	"github.com/unclesp1d3r/redtracks/internal/checklist"
	"github.com/unclesp1d3r/redtracks/internal/model"
	"github.com/unclesp1d3r/redtracks/internal/progress"
)

func httpDef() checklist.Definition {
	return checklist.Definition{
		ID:    "http",
		Match: checklist.Match{ServiceTypes: []string{"http"}},
		Items: []checklist.Item{{ID: "http-1"}, {ID: "http-2"}},
	}
}

func todoState(ids ...string) model.ChecklistState {
	items := map[string]model.ItemState{}
	for _, id := range ids {
		items[id] = model.ItemState{Status: model.ItemTodo}
	}
	return model.ChecklistState{Items: items}
}

func summaryFor(t *testing.T, got []progress.HostSummary, id string) progress.HostSummary {
	t.Helper()
	for _, s := range got {
		if s.HostID == id {
			return s
		}
	}
	t.Fatalf("no summary for host %q", id)
	return progress.HostSummary{}
}

func TestMissingFlags(t *testing.T) {
	eng := model.Engagement{
		Meta: model.EngagementMeta{FlagTypes: model.DefaultFlagTypes(model.EngagementOSAI)},
		Hosts: []model.Host{{
			ID: "a", IP: "10.0.0.1",
			Flags: []model.Flag{{TypeID: "ai_vector", Value: "abc"}, {TypeID: "proof", Value: ""}},
		}},
	}

	got := summaryFor(t, progress.Summarize(eng, nil), "a")

	if want := []string{"trad_vector", "proof"}; !slices.Equal(got.MissingFlags, want) {
		t.Fatalf("MissingFlags = %v, want %v", got.MissingFlags, want)
	}
	if got.FlagPoints != 15 || got.TotalPoints != 30 {
		t.Fatalf("points = %d/%d, want 15/30", got.FlagPoints, got.TotalPoints)
	}
}

func TestUntouchedServices(t *testing.T) {
	eng := model.Engagement{Hosts: []model.Host{{
		ID: "a", IP: "10.0.0.1",
		Services: []model.Service{
			{ID: "s1", Port: 80, Protocol: "tcp", Type: "http", Checklist: todoState("http-1", "http-2")},
			{ID: "s2", Port: 8080, Protocol: "tcp", Type: "http", Checklist: model.ChecklistState{
				Items: map[string]model.ItemState{"http-1": {Status: model.ItemDone}},
			}},
			{ID: "s3", Port: 22, Protocol: "tcp", Type: "ssh"},
		},
	}}}

	got := summaryFor(t, progress.Summarize(eng, []checklist.Definition{httpDef()}), "a")

	if want := []string{"80/tcp http"}; !slices.Equal(got.UntouchedServices, want) {
		t.Fatalf("UntouchedServices = %v, want %v", got.UntouchedServices, want)
	}
	if got.OpenItems != 3 || got.ItemsDone != 1 || got.ItemsTotal != 4 {
		t.Fatalf("open/done/total = %d/%d/%d, want 3/1/4", got.OpenItems, got.ItemsDone, got.ItemsTotal)
	}
}

func TestOpenItemsIncludeHostChecklists(t *testing.T) {
	def := checklist.Definition{
		ID:    "linux",
		Match: checklist.Match{HostOS: []string{"linux"}},
		Items: []checklist.Item{{ID: "l-1"}, {ID: "l-2"}},
	}
	eng := model.Engagement{Hosts: []model.Host{{
		ID: "a", IP: "10.0.0.1", OS: "Linux",
		Checklist: model.ChecklistState{Items: map[string]model.ItemState{"l-1": {Status: model.ItemDone}}},
	}}}

	got := summaryFor(t, progress.Summarize(eng, []checklist.Definition{def}), "a")

	if got.OpenItems != 1 || got.ItemsDone != 1 || got.ItemsTotal != 2 {
		t.Fatalf("open/done/total = %d/%d/%d, want 1/1/2", got.OpenItems, got.ItemsDone, got.ItemsTotal)
	}
}

func TestUntriedCreds(t *testing.T) {
	eng := model.Engagement{
		Hosts: []model.Host{{ID: "A", IP: "10.0.0.1"}, {ID: "B", IP: "10.0.0.2"}, {ID: "C", IP: "10.0.0.3"}},
		Creds: model.Creds{Credentials: []model.Credential{
			{ID: "cred1", FoundOnHostID: "A", TriedOn: []string{"B"}},
		}},
	}

	got := progress.Summarize(eng, nil)

	if c := summaryFor(t, got, "C").UntriedCreds; !slices.Equal(c, []string{"cred1"}) {
		t.Fatalf("host C UntriedCreds = %v", c)
	}
	for _, id := range []string{"A", "B"} {
		if c := summaryFor(t, got, id).UntriedCreds; len(c) != 0 {
			t.Fatalf("host %s UntriedCreds = %v, want none", id, c)
		}
	}
}

func TestUntriedCredsExcludesWorksOn(t *testing.T) {
	eng := model.Engagement{
		Hosts: []model.Host{{ID: "A", IP: "10.0.0.1"}, {ID: "B", IP: "10.0.0.2"}},
		Creds: model.Creds{Credentials: []model.Credential{{ID: "c", FoundOnHostID: "A", WorksOn: []string{"B"}}}},
	}

	if c := summaryFor(t, progress.Summarize(eng, nil), "B").UntriedCreds; len(c) != 0 {
		t.Fatalf("UntriedCreds = %v, want none", c)
	}
}

func TestNoAccessPath(t *testing.T) {
	path := []model.AccessPath{{ID: "p"}}
	eng := model.Engagement{Hosts: []model.Host{
		{ID: "untouched", IP: "10.0.0.1", Status: model.HostStatusUntouched},
		{ID: "foothold", IP: "10.0.0.2", Status: model.HostStatusFoothold},
		{ID: "pathed", IP: "10.0.0.3", Status: model.HostStatusRoot, AccessPaths: path},
	}}

	got := progress.Summarize(eng, nil)

	for id, want := range map[string]bool{"untouched": false, "foothold": true, "pathed": false} {
		if s := summaryFor(t, got, id); s.NoAccessPath != want {
			t.Fatalf("%s NoAccessPath = %v, want %v", id, s.NoAccessPath, want)
		}
	}
}

func TestBoardOrder(t *testing.T) {
	flagTypes := []model.FlagType{{ID: "x", Points: 10}, {ID: "y", Points: 10}, {ID: "z", Points: 5}}
	captured := func(ids ...string) []model.Flag {
		flags := make([]model.Flag, len(ids))
		for i, id := range ids {
			flags[i] = model.Flag{TypeID: id, Value: "v"}
		}
		return flags
	}
	done := model.ChecklistState{Items: map[string]model.ItemState{"http-1": {Status: model.ItemDone}}}
	svc := func(c model.ChecklistState) []model.Service {
		return []model.Service{{ID: "s", Type: "http", Checklist: c}}
	}
	eng := model.Engagement{
		Meta: model.EngagementMeta{FlagTypes: flagTypes},
		Hosts: []model.Host{
			{ID: "twenty", IP: "10.0.0.9", Flags: captured("x", "y")},
			{ID: "zeroB", IP: "10.0.0.2", Services: svc(done)},
			{ID: "ten", IP: "10.0.0.8", Flags: captured("x")},
			{ID: "zeroA", IP: "10.0.0.1", Services: svc(model.ChecklistState{})},
		},
	}

	got := progress.Summarize(eng, []checklist.Definition{httpDef()})

	ids := make([]string, len(got))
	for i, s := range got {
		ids[i] = s.HostID
	}
	if want := []string{"zeroA", "zeroB", "ten", "twenty"}; !slices.Equal(ids, want) {
		t.Fatalf("order = %v, want %v", ids, want)
	}
}

func TestBoardOrderTiesBreakByIP(t *testing.T) {
	eng := model.Engagement{Hosts: []model.Host{
		{ID: "b", IP: "10.0.0.2"}, {ID: "a", IP: "10.0.0.1"},
	}}

	got := progress.Summarize(eng, nil)

	if got[0].HostID != "a" || got[1].HostID != "b" {
		t.Fatalf("order = %s, %s; want a, b", got[0].HostID, got[1].HostID)
	}
}

func TestSummarizeDoesNotMutateInput(t *testing.T) {
	eng := model.Engagement{Hosts: []model.Host{{ID: "b", IP: "10.0.0.2"}, {ID: "a", IP: "10.0.0.1"}}}

	progress.Summarize(eng, nil)

	if eng.Hosts[0].ID != "b" {
		t.Fatal("Summarize reordered the caller's hosts")
	}
}

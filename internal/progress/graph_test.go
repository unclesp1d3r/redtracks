package progress_test

import (
	"os"
	"strings"
	"testing"

	"github.com/unclesp1d3r/redtracks/internal/model"
	"github.com/unclesp1d3r/redtracks/internal/progress"
)

const goldenPath = "testdata/graph.golden.mmd"

func graphEngagement() model.Engagement {
	return model.Engagement{Hosts: []model.Host{
		{
			ID: "web", IP: "10.0.0.5", Hostname: "web01", Status: model.HostStatusFoothold,
			AccessPaths: []model.AccessPath{{ID: "p1", Method: "sqli"}},
		},
		{
			ID: "app", IP: "10.0.0.6", Hostname: "app01", Status: model.HostStatusUser,
			AccessPaths: []model.AccessPath{{ID: "p2", Method: "ssh", ViaHostID: "web"}},
		},
		{
			ID: "dc", IP: "10.0.0.10", Hostname: "dc01", Status: model.HostStatusRoot,
			AccessPaths: []model.AccessPath{
				{ID: "p3", Method: "winrm", ViaHostID: "app"},
				{ID: "p4", Method: "kerberoast", ViaHostID: "gone"},
			},
		},
		{ID: "lonely", IP: "fe80::1", Status: model.HostStatusUntouched},
	}}
}

func TestAttackGraphGolden(t *testing.T) {
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}

	got := progress.AttackGraph(graphEngagement())

	if got != string(want) {
		t.Fatalf("graph mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestAttackGraphEscapesHostileLabels(t *testing.T) {
	eng := model.Engagement{Hosts: []model.Host{{
		ID: "h", IP: "10.0.0.1", Hostname: `a"];x["b`, Status: model.HostStatusRoot,
		AccessPaths: []model.AccessPath{{ID: "p", Method: "<img src=x>"}},
	}}}

	got := progress.AttackGraph(eng)

	if !strings.HasPrefix(got, "flowchart LR\n") {
		t.Fatalf("missing header: %q", got)
	}
	if strings.Contains(got, "<") {
		t.Fatalf("output contains <:\n%s", got)
	}
	for _, line := range strings.Split(got, "\n") {
		if strings.Count(line, `"`)%2 != 0 || strings.Contains(line, `"]`) && !strings.HasSuffix(line, `"]`) {
			t.Fatalf("unbalanced or escaped quote in %q", line)
		}
	}
}

func TestAttackGraphTruncatesLongLabels(t *testing.T) {
	eng := model.Engagement{Hosts: []model.Host{{
		ID: "h", IP: "10.0.0.1", Hostname: strings.Repeat("x", 200),
	}}}

	got := progress.AttackGraph(eng)

	if strings.Contains(got, strings.Repeat("x", 61)) {
		t.Fatalf("label not cut at 60 chars:\n%s", got)
	}
}

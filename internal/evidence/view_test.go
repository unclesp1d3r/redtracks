package evidence

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/unclesp1d3r/redtracks/internal/model"
)

var t0 = time.Date(2026, 10, 12, 10, 0, 0, 0, time.UTC)

func item(id, file string, at time.Time, links ...model.EvidenceLink) model.EvidenceItem {
	return model.EvidenceItem{ID: id, File: file, Kind: model.EvidenceKindText, CapturedAt: at, Links: links}
}

func TestBuildOrdersByCaptureThenID(t *testing.T) {
	items := []model.EvidenceItem{
		item("b", "b.txt", t0),
		item("c", "c.txt", t0.Add(-time.Hour)),
		item("a", "a.txt", t0),
	}

	v := Build(BuildInput{Items: items})

	got := []string{v.Items[0].ID, v.Items[1].ID, v.Items[2].ID}
	if strings.Join(got, "") != "cab" {
		t.Fatalf("order = %v, want c a b", got)
	}
	if v.NotIndexed == nil {
		t.Error("NotIndexed must be an empty slice, not nil")
	}
}

func TestBuildLabelsAndRemovesLinks(t *testing.T) {
	host := model.Host{
		Flags:       []model.Flag{{TypeID: "user"}},
		AccessPaths: []model.AccessPath{{ID: "ap1", Method: "ssh key"}},
	}
	findings := []model.Finding{{ID: "f1", Title: "SQLi"}}
	slots := []SlotInfo{{ID: "s1", Title: "whoami"}, {ID: "s2", Title: "web shot", ServiceID: "svc"}}
	links := []model.EvidenceLink{
		{Kind: model.EvidenceLinkSlot, TargetID: "s1"},
		{Kind: model.EvidenceLinkSlot, TargetID: "s2", ServiceID: "svc"},
		{Kind: model.EvidenceLinkSlot, TargetID: "s2", ServiceID: "gone"},
		{Kind: model.EvidenceLinkFlag, TargetID: "user"},
		{Kind: model.EvidenceLinkFlag, TargetID: "root"},
		{Kind: model.EvidenceLinkFinding, TargetID: "f1"},
		{Kind: model.EvidenceLinkFinding, TargetID: "f2"},
		{Kind: model.EvidenceLinkAccessPath, TargetID: "ap1"},
	}
	wantLabel := []string{"whoami", "web shot", "", "user", "", "SQLi", "", "ssh key"}
	wantRemoved := []bool{false, false, true, false, true, false, true, false}

	v := Build(
		BuildInput{
			Host:     host,
			Findings: findings,
			Slots:    slots,
			Items:    []model.EvidenceItem{item("i", "i.txt", t0, links...)},
		},
	)

	for i, l := range v.Items[0].Links {
		if l.Label != wantLabel[i] || l.Removed != wantRemoved[i] {
			t.Errorf("link %d = %+v, want label %q removed %v", i, l, wantLabel[i], wantRemoved[i])
		}
	}
}

func TestBuildMarksMissingAndKeepsNotIndexed(t *testing.T) {
	gone := item("g", "g.txt", t0)

	v := Build(BuildInput{
		Items: []model.EvidenceItem{gone, item("k", "k.txt", t0)}, Missing: []model.EvidenceItem{gone},
		NotIndexed: []string{"x.png"},
	})

	if !v.Items[0].Missing || v.Items[1].Missing {
		t.Errorf("missing flags = %v %v", v.Items[0].Missing, v.Items[1].Missing)
	}
	if len(v.NotIndexed) != 1 || v.NotIndexed[0] != "x.png" {
		t.Errorf("NotIndexed = %v", v.NotIndexed)
	}
}

func writeText(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestSearchFindsCaptionAndTextCaseInsensitive(t *testing.T) {
	dir := t.TempDir()
	writeText(t, dir, "a.txt", "line one\nthe Kerberoast hash is here\n")
	writeText(t, dir, "b.txt", "nothing relevant\n")
	hosts := []HostItems{{IP: "10.0.0.1", Dir: dir, Items: []model.EvidenceItem{
		item("a", "a.txt", t0), item("b", "b.txt", t0),
		{ID: "c", File: "c.png", Kind: model.EvidenceKindImage, Caption: "KERBEROAST screenshot"},
	}}}

	hits := Search(hosts, "kerberoast")

	if len(hits) != 2 || hits[0].ItemID != "a" || hits[1].ItemID != "c" {
		t.Fatalf("hits = %+v", hits)
	}
	if !strings.Contains(hits[0].Snippet, "Kerberoast hash") || hits[0].IP != "10.0.0.1" {
		t.Errorf("snippet = %+v", hits[0])
	}
}

func TestSearchEmptyQueryAndUnreadableFiles(t *testing.T) {
	dir := t.TempDir()
	hosts := []HostItems{{IP: "10.0.0.1", Dir: dir, Items: []model.EvidenceItem{
		item("a", "missing.txt", t0), item("esc", "../x.txt", t0),
	}}}

	if hits := Search(hosts, "   "); hits != nil {
		t.Errorf("blank query hits = %v", hits)
	}
	if hits := Search(hosts, "x"); len(hits) != 0 {
		t.Errorf("unreadable files produced hits = %v", hits)
	}
}

func TestSearchSurvivesLongLines(t *testing.T) {
	dir := t.TempDir()
	writeText(t, dir, "a.txt", strings.Repeat("x", 3*maxSearchLineBytes)+"\nneedle here\n")
	hosts := []HostItems{{IP: "10.0.0.1", Dir: dir, Items: []model.EvidenceItem{item("a", "a.txt", t0)}}}

	hits := Search(hosts, "needle")

	if len(hits) != 1 {
		t.Fatalf("hits = %+v", hits)
	}
}

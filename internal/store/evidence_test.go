package store

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/unclesp1d3r/redtracks/internal/model"
)

const validEvidenceJSON = `{
  "items": [
    {"id": "e1", "file": "shot.png", "kind": "image", "capturedAt": "2026-10-12T10:00:00Z", "caption": "login"}
  ]
}
`

func evidenceFilePath(dir string) string {
	return filepath.Join(dir, "hosts", fixtureIP, evidenceFile)
}

func evDirOf(dir string) string {
	return filepath.Join(dir, "hosts", fixtureIP, evidenceDir)
}

func evidenceItems(s *Store, ip string) []model.EvidenceItem {
	return s.Engagement().Evidence[ip].Items
}

func TestOpenWithoutEvidenceHasEmptyList(t *testing.T) {
	s := mustOpen(t, copyFixture(t))
	if got := evidenceItems(s, fixtureIP); len(got) != 0 {
		t.Fatalf("items = %+v, want none", got)
	}
	view, err := s.EvidenceListing(fixtureIP)
	if err != nil || len(view.Items) != 0 || len(view.NotIndexed) != 0 || len(view.Missing) != 0 {
		t.Fatalf("listing = %+v, %v", view, err)
	}
}

func TestReloadReportsHandWrittenEvidenceIndex(t *testing.T) {
	dir := copyFixture(t)
	s := mustOpen(t, dir)
	path := evidenceFilePath(dir)
	mustWrite(t, path, validEvidenceJSON)
	bumpMtime(t, path)
	changed, err := s.Reload()
	want := "hosts/" + fixtureIP + "/evidence.json"
	if err != nil || !slices.Equal(changed, []string{want}) {
		t.Fatalf("Reload = %v, %v; want [%s]", changed, err, want)
	}
	items := evidenceItems(s, fixtureIP)
	if len(items) != 1 || items[0].ID != "e1" || items[0].Links == nil {
		t.Fatalf("items = %+v", items)
	}
}

func TestCorruptEvidenceKeepsLastGoodItems(t *testing.T) {
	dir := copyFixture(t)
	path := evidenceFilePath(dir)
	mustWrite(t, path, validEvidenceJSON)
	s := mustOpen(t, dir)
	mustWrite(t, path, "{not json")
	bumpMtime(t, path)
	_, err := s.Reload()
	if err == nil || !strings.Contains(err.Error(), "evidence.json") {
		t.Fatalf("Reload err = %v, want a ParseError naming evidence.json", err)
	}
	if got := evidenceItems(s, fixtureIP); len(got) != 1 || got[0].ID != "e1" {
		t.Fatalf("items = %+v, want last good copy", got)
	}
}

func TestMutateEvidenceWritesIndentedCamelCaseAndIsNotExternal(t *testing.T) {
	dir := copyFixture(t)
	s := mustOpen(t, dir)
	err := s.MutateEvidence(fixtureIP, func(f *model.EvidenceFile) error {
		f.Items = append(f.Items, model.EvidenceItem{
			ID: "e1", File: "shot.png", Kind: model.EvidenceKindImage,
			CapturedAt: time.Date(2026, 10, 12, 10, 0, 0, 0, time.UTC),
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	got := readFile(t, evidenceFilePath(dir))
	for _, want := range []string{"{\n  \"items\": [\n    {\n      \"id\": \"e1\"", "\"capturedAt\"", "\"links\": []"} {
		if !strings.Contains(got, want) {
			t.Errorf("evidence.json missing %q:\n%s", want, got)
		}
	}
	if changed, err := s.Reload(); err != nil || len(changed) != 0 {
		t.Fatalf("Reload after own write = %v, %v", changed, err)
	}
}

func TestMutateEvidenceUnknownHostFails(t *testing.T) {
	s := mustOpen(t, copyFixture(t))
	err := s.MutateEvidence("10.9.9.9", func(*model.EvidenceFile) error { return nil })
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("err = %v, want not found", err)
	}
}

func TestHandAddedFileIsNotIndexedAndDirReported(t *testing.T) {
	dir := copyFixture(t)
	s := mustOpen(t, dir)
	evDir := evDirOf(dir)
	mustWriteAll(t, filepath.Join(evDir, "dropped.png"), "png")
	bumpMtime(t, filepath.Join(evDir, "dropped.png"))
	changed, err := s.Reload()
	want := "hosts/" + fixtureIP + "/evidence"
	if err != nil || !slices.Equal(changed, []string{want}) {
		t.Fatalf("Reload = %v, %v; want [%s]", changed, err, want)
	}
	view, err := s.EvidenceListing(fixtureIP)
	if err != nil || !slices.Equal(view.NotIndexed, []string{"dropped.png"}) {
		t.Fatalf("listing = %+v, %v", view, err)
	}
	if changed, err := s.Reload(); err != nil || len(changed) != 0 {
		t.Fatalf("idle Reload = %v, %v", changed, err)
	}
}

func TestDeletedFileIsMissingUntilEntryRemoved(t *testing.T) {
	dir := copyFixture(t)
	evDir := evDirOf(dir)
	mustWriteAll(t, filepath.Join(evDir, "shot.png"), "png")
	mustWrite(t, evidenceFilePath(dir), validEvidenceJSON)
	s := mustOpen(t, dir)
	view, err := s.EvidenceListing(fixtureIP)
	if err != nil || len(view.Missing) != 0 || len(view.NotIndexed) != 0 {
		t.Fatalf("listing before delete = %+v (err %v)", view, err)
	}
	if err := os.Remove(filepath.Join(evDir, "shot.png")); err != nil {
		t.Fatal(err)
	}
	view, err = s.EvidenceListing(fixtureIP)
	if err != nil || len(view.Missing) != 1 || view.Missing[0].ID != "e1" {
		t.Fatalf("listing = %+v, %v", view, err)
	}
	err = s.MutateEvidence(fixtureIP, func(f *model.EvidenceFile) error {
		f.Items = nil
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	view, err = s.EvidenceListing(fixtureIP)
	if err != nil || len(view.Items) != 0 || len(view.Missing) != 0 || len(view.NotIndexed) != 0 {
		t.Fatalf("listing after removal = %+v, %v", view, err)
	}
}

func TestEvidenceLinkToUnknownFindingIsValid(t *testing.T) {
	dir := copyFixture(t)
	mustWrite(t, evidenceFilePath(dir), `{"items": [{"id": "e1", "file": "a.txt", "kind": "text",
  "links": [{"kind": "finding", "targetId": "no-such-finding"}]}]}`)
	s := mustOpen(t, dir)
	if got := evidenceItems(s, fixtureIP); len(got) != 1 || len(got[0].Links) != 1 {
		t.Fatalf("items = %+v", got)
	}
}

func TestEvidenceValidationRejectsBadItems(t *testing.T) {
	cases := map[string]string{
		"separator":  `{"id": "e1", "file": "a/b.png", "kind": "image"}`,
		"backslash":  `{"id": "e1", "file": "a\\b.png", "kind": "image"}`,
		"dotdot":     `{"id": "e1", "file": "..", "kind": "image"}`,
		"empty file": `{"id": "e1", "file": "", "kind": "image"}`,
		"bad kind":   `{"id": "e1", "file": "a.png", "kind": "video"}`,
		"bad link":   `{"id": "e1", "file": "a.png", "kind": "image", "links": [{"kind": "host", "targetId": "x"}]}`,
		"empty id":   `{"id": "", "file": "a.png", "kind": "image"}`,
		"duplicate":  `{"id": "e1", "file": "a.png", "kind": "image"}, {"id": "e1", "file": "b.png", "kind": "image"}`,
	}
	for name, item := range cases {
		t.Run(name, func(t *testing.T) {
			dir := copyFixture(t)
			path := evidenceFilePath(dir)
			mustWrite(t, path, `{"items": [`+item+`]}`)
			_, err := Open(dir)
			if err == nil || !strings.Contains(err.Error(), path) {
				t.Fatalf("Open err = %v, want an error naming %s", err, path)
			}
		})
	}
}

func TestEvidenceIPv6UsesUnderscoreDirectory(t *testing.T) {
	dir := t.TempDir()
	s := mustOpen(t, dir)
	if err := s.AddHost(model.Host{IP: "fe80::1"}); err != nil {
		t.Fatal(err)
	}
	err := s.MutateEvidence("FE80::1", func(f *model.EvidenceFile) error {
		f.Items = append(f.Items, model.EvidenceItem{ID: "e1", File: "a.txt", Kind: model.EvidenceKindText})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "hosts", "fe80__1", evidenceFile)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("index not at %s: %v", path, err)
	}
	reopened := mustOpen(t, dir)
	if got := evidenceItems(reopened, "fe80::1"); len(got) != 1 || got[0].ID != "e1" {
		t.Fatalf("items = %+v", got)
	}
}

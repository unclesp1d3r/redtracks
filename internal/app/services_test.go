package app

import (
	"encoding/base64"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/unclesp1d3r/redtracks/checklists"
	"github.com/unclesp1d3r/redtracks/internal/model"
	"github.com/unclesp1d3r/redtracks/internal/progress"
	"github.com/unclesp1d3r/redtracks/internal/store"
)

const nmapFixture = "../importers/testdata/nmap-localhost.xml"

func TestSetItemPersists(t *testing.T) {
	f := openSample(t)
	a, dir := f.app, f.dir
	h := NewHostService(a)

	if err := h.SetItem(sampleHost, "", "recon.ping", model.ItemDone, "ok"); err != nil {
		t.Fatal(err)
	}

	st, err := store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	host, _ := st.Engagement().Host(sampleHost)
	got := host.Checklist.Items["recon.ping"]
	if got.Status != model.ItemDone || got.UpdatedAt == "" || got.Note != "ok" {
		t.Fatalf("got %+v", got)
	}
}

func addSecondHost(t *testing.T, a *App) model.Host {
	t.Helper()
	if err := NewHostService(a).Add("10.10.10.9", ""); err != nil {
		t.Fatal(err)
	}
	h, ok := current(t, a).Host("10.10.10.9")
	if !ok || h.ID == "" {
		t.Fatalf("second host missing: %+v", h)
	}
	return h
}

func TestMarkTriedAppendsOnce(t *testing.T) {
	a := openSample(t).app
	e := NewEngagementService(a)
	host := addSecondHost(t, a)
	id := current(t, a).Creds.Credentials[0].ID

	for range 2 {
		if err := e.MarkTried(id, host.IP, false); err != nil {
			t.Fatal(err)
		}
	}
	if err := e.MarkTried(id, host.IP, true); err != nil {
		t.Fatal(err)
	}

	c := current(t, a).Creds.Credentials[0]
	if !slices.Equal(c.TriedOn, []string{host.ID}) {
		t.Fatalf("TriedOn %v, want [%s]", c.TriedOn, host.ID)
	}
	if !slices.Equal(c.WorksOn, []string{host.ID}) {
		t.Fatalf("WorksOn %v, want [%s]", c.WorksOn, host.ID)
	}
}

func TestMarkTriedRemovesCredFromUntried(t *testing.T) {
	a := openSample(t).app
	host := addSecondHost(t, a)
	id := current(t, a).Creds.Credentials[0].ID
	untriedOn := func() []string {
		for _, s := range progress.Summarize(current(t, a), nil) {
			if s.IP == host.IP {
				return s.UntriedCreds
			}
		}
		t.Fatal("host summary missing")
		return nil
	}

	if !slices.Contains(untriedOn(), id) {
		t.Fatalf("setup: credential should start untried, got %v", untriedOn())
	}
	if err := NewEngagementService(a).MarkTried(id, host.IP, false); err != nil {
		t.Fatal(err)
	}
	if slices.Contains(untriedOn(), id) {
		t.Fatalf("credential still untried after MarkTried: %v", untriedOn())
	}
}

func TestMarkTriedUnknownHostErrors(t *testing.T) {
	a := openSample(t).app
	id := current(t, a).Creds.Credentials[0].ID
	if err := NewEngagementService(a).MarkTried(id, "10.9.9.9", false); !errors.Is(err, errNotFound) {
		t.Fatalf("err = %v, want not found", err)
	}
}

func TestImportPreviewThenApply(t *testing.T) {
	a := openSample(t).app
	raw, err := os.ReadFile(nmapFixture)
	if err != nil {
		t.Fatal(err)
	}
	s := NewImportService(a)

	resp, err := s.PreviewData("nmap.xml", base64.StdEncoding.EncodeToString(raw))
	if err != nil || resp.Token == "" || len(resp.Preview.Rows) == 0 {
		t.Fatalf("preview: %v %+v", err, resp)
	}
	if _, err := s.Apply(resp.Token, map[string]string{"unused.example": "10.0.0.1"}); err != nil {
		t.Fatal(err)
	}
	cur := current(t, a)
	if _, ok := cur.Host("127.0.0.1"); !ok {
		t.Fatal("host 127.0.0.1 missing")
	}
	if _, err := s.Apply(resp.Token, nil); err == nil || !strings.Contains(err.Error(), "unknown token") {
		t.Fatalf("second apply: %v", err)
	}
}

func TestPreviewDataRejectsOversize(t *testing.T) {
	a := openSample(t).app
	big := base64.StdEncoding.EncodeToString(make([]byte, maxImportBytes+1))
	if _, err := NewImportService(a).PreviewData("big", big); err == nil {
		t.Fatal("want size error")
	}
}

func TestUpdateServiceKeepsChecklist(t *testing.T) {
	a := openSample(t).app
	h := NewHostService(a)
	cur := current(t, a)
	host, _ := cur.Host(sampleHost)
	stale := host.Services[0]

	if err := h.SetItem(sampleHost, stale.ID, "http.robots", model.ItemDone, ""); err != nil {
		t.Fatal(err)
	}
	stale.Notes = "edited"
	if err := h.UpdateService(sampleHost, stale); err != nil {
		t.Fatal(err)
	}

	cur = current(t, a)
	host, _ = cur.Host(sampleHost)
	svc := host.Services[0]
	if svc.Checklist.Items["http.robots"].Status != model.ItemDone || svc.Notes != "edited" {
		t.Fatalf("got %+v", svc)
	}
}

func TestOpenTwiceStopsFirstPoller(t *testing.T) {
	f := newTestApp(t)
	a, dirA := f.app, f.dir
	ev := &events{}
	a.SetEmitter(ev.emit)
	e := NewEngagementService(a)
	dirB, err := e.Create("second", "", model.EngagementLab)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.Open(dirA); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Open(dirB); err != nil {
		t.Fatal(err)
	}
	before := ev.count(eventChanged)

	notes := filepath.Join(dirA, "notes.md")
	if err := os.WriteFile(notes, []byte("changed"), logFileMode); err != nil {
		t.Fatal(err)
	}
	future := time.Now().Add(time.Minute)
	if err := os.Chtimes(filepath.Join(dirA, "creds.json"), future, future); err != nil {
		t.Fatal(err)
	}
	time.Sleep(200 * time.Millisecond)

	if got := ev.count(eventChanged); got != before {
		t.Fatalf("changed events %d -> %d", before, got)
	}
}

func TestOpenRejectsOutsideEngagements(t *testing.T) {
	a := newTestApp(t).app
	for _, dir := range []string{a.Root, filepath.Join(a.Root, engagementsDir, "pg-sample", "hosts"), "/tmp"} {
		if _, err := NewEngagementService(a).Open(dir); err == nil {
			t.Fatalf("Open(%q) should fail", dir)
		}
	}
}

func TestWriteNotesConflictMessage(t *testing.T) {
	a := openSample(t).app
	e := NewEngagementService(a)
	_, err := e.WriteNotes(sampleHost, "x", "stale-hash")
	if err == nil || !strings.HasPrefix(err.Error(), "conflict") {
		t.Fatalf("got %v", err)
	}
}

func TestStoreErrorsWithoutEngagement(t *testing.T) {
	a := newTestApp(t).app
	if _, err := a.Store(); err == nil || err.Error() != "no engagement open" {
		t.Fatalf("got %v", err)
	}
	if err := NewHostService(a).Add("10.0.0.1", "x"); err == nil {
		t.Fatal("want error")
	}
}

// shippedCount is the number of definitions embedded in checklists.FS.
func shippedCount(t *testing.T) int {
	t.Helper()
	files, err := fs.Glob(checklists.FS, "*.json")
	if err != nil {
		t.Fatal(err)
	}
	return len(files)
}

func TestNewInstallsDefaults(t *testing.T) {
	root := t.TempDir()
	a, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = a.Close() }()
	files, err := filepath.Glob(filepath.Join(root, checklistsDir, "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	want := shippedCount(t)
	if len(files) != want {
		t.Fatalf("installed %d definition files, want %d", len(files), want)
	}
	if len(a.Definitions()) != want {
		t.Fatalf("loaded %d definitions, want %d", len(a.Definitions()), want)
	}
}

func TestNewSurvivesBadDefinition(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, checklistsDir)
	if err := os.MkdirAll(dir, dirMode); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "bad.json"), []byte("{"), logFileMode); err != nil {
		t.Fatal(err)
	}
	a, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = a.Close() }()
	if len(a.Definitions()) != 0 {
		t.Fatalf("want no definitions, got %d", len(a.Definitions()))
	}
	logged, err := os.ReadFile(filepath.Join(root, logFileName))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(logged), "bad.json") {
		t.Fatalf("error not logged: %s", logged)
	}
	ev := &events{}
	a.SetEmitter(ev.emit)
	if ev.count(eventError) == 0 {
		t.Fatal("definitions error not emitted")
	}
}

package app

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/unclesp1d3r/redtracks/internal/model"
	"github.com/unclesp1d3r/redtracks/internal/store"
)

// logFixture writes a numbered log file and returns its absolute path.
func logFixture(t *testing.T, lines int) string {
	t.Helper()
	var b strings.Builder
	for range lines {
		b.WriteString("\x1b[32mscan output\x1b[0m line\n")
	}
	p := filepath.Join(t.TempDir(), "tmux.log")
	if err := os.WriteFile(p, []byte(b.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestImportPathAcceptsAbsoluteRegularFile(t *testing.T) { // AE16
	f := openSample(t)
	svc := NewSessionLogService(f.app)

	src, err := svc.ImportPath(logFixture(t, 300))
	if err != nil {
		t.Fatal(err)
	}
	if src.Lines != 300 || src.OriginalName != "tmux.log" {
		t.Fatalf("source = %+v", src)
	}
	if _, err := os.Stat(filepath.Join(store.LogsDir(f.dir), src.Name)); err != nil {
		t.Fatalf("copy missing under logs/: %v", err)
	}
	list, err := svc.List()
	if err != nil || len(list) != 1 || list[0].ID != src.ID {
		t.Fatalf("List = %+v, %v", list, err)
	}
}

func TestImportPathRefusesRelativePathAndNoEngagement(t *testing.T) {
	f := openSample(t)
	if _, err := NewSessionLogService(f.app).ImportPath("rel/x.log"); err == nil {
		t.Fatal("relative path: want error")
	}
	closed := newTestApp(t)
	if _, err := NewSessionLogService(closed.app).ImportPath(logFixture(t, 1)); err == nil {
		t.Fatal("no engagement: want error")
	}
}

func TestPickLogFileWithoutNativeDialog(t *testing.T) {
	f := newTestApp(t)
	svc := NewSessionLogService(f.app)
	if _, err := svc.PickLogFile(); !errors.Is(err, ErrNoNativeDialog) {
		t.Fatalf("err = %v, want ErrNoNativeDialog", err)
	}
	f.app.SetHooks(Hooks{Capabilities: Capabilities{NativeDialog: true}}) // PickFile nil
	if _, err := svc.PickLogFile(); !errors.Is(err, ErrNoNativeDialog) {
		t.Fatalf("nil PickFile: err = %v, want ErrNoNativeDialog", err)
	}
}

func TestPickLogFileReturnsChosenPathOrEmptyOnCancel(t *testing.T) {
	f := newTestApp(t)
	svc := NewSessionLogService(f.app)
	for _, want := range []string{"/tmp/chosen.log", ""} {
		f.app.SetHooks(Hooks{
			Capabilities: Capabilities{NativeDialog: true},
			PickFile:     func() (string, error) { return want, nil },
		})
		got, err := svc.PickLogFile()
		if err != nil || got != want {
			t.Fatalf("PickLogFile = %q, %v; want %q", got, err, want)
		}
	}
}

func TestServicePageSearchAndClip(t *testing.T) { // AE6
	f := openSample(t)
	svc := NewSessionLogService(f.app)
	src, err := svc.ImportPath(logFixture(t, 6000))
	if err != nil {
		t.Fatal(err)
	}

	page, err := svc.Page(src.ID, 10, 2)
	if err != nil || len(page) != 2 || page[0] != "scan output line" {
		t.Fatalf("Page = %q, %v", page, err)
	}
	hits, err := svc.Search(src.ID, "SCAN", 4, 3)
	if err != nil || len(hits) != 3 || hits[0].Line != 5 {
		t.Fatalf("Search = %+v, %v", hits, err)
	}

	item, err := svc.Clip(src.ID, sampleHost, 100, 139)
	if err != nil {
		t.Fatal(err)
	}
	want := &model.ClipOf{LogID: src.ID, FirstLine: 100, LastLine: 139}
	if item.Kind != model.EvidenceKindText || item.ClipOf == nil || *item.ClipOf != *want ||
		!strings.Contains(item.Caption, src.ID) || !strings.Contains(item.Caption, "101-140") {
		t.Fatalf("item = %+v", item)
	}
	body, err := os.ReadFile(filepath.Join(f.dir, "hosts", sampleHost, "evidence", item.File))
	if err != nil || strings.Count(string(body), "\n") != 40 || strings.Contains(string(body), "\x1b") {
		t.Fatalf("clip body lines/escapes wrong: %v", err)
	}
	st, err := f.app.Store()
	if err != nil {
		t.Fatal(err)
	}
	listing, err := st.EvidenceListing(sampleHost)
	if err != nil || len(listing.Items) == 0 || listing.Items[len(listing.Items)-1].ClipOf == nil {
		t.Fatalf("clipOf not persisted in the index: %+v, %v", listing, err)
	}

	if _, err := svc.Clip(src.ID, sampleHost, 0, 5000); err == nil {
		t.Fatal("5001-line clip: want refusal")
	}
}

func TestServiceRejectsBadIDs(t *testing.T) {
	f := openSample(t)
	svc := NewSessionLogService(f.app)
	for _, id := range []string{"../x", "a/b", "..", "", "20260101T000000Z-1"} {
		if _, err := svc.Page(id, 0, 1); err == nil {
			t.Errorf("Page(%q): want error", id)
		}
	}
}

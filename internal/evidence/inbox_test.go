package evidence

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/unclesp1d3r/redtracks/internal/model"
)

const (
	testEvery    = 10 * time.Millisecond
	testDeadline = 3 * time.Second
)

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func appendFile(t *testing.T, dir, name, content string) {
	t.Helper()
	f, err := os.OpenFile(filepath.Join(dir, name), os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestScannerFileWrittenInTwoStepsIsNotReadyUntilTicksAgree(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	sc := &inboxScanner{dir: dir}
	writeFile(t, dir, "scan.txt", "part one")
	scanReady := func() []string {
		t.Helper()
		r, err := sc.scan()
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	// Act + Assert
	if r := scanReady(); len(r) != 0 {
		t.Fatalf("first tick ready = %v, want none", r)
	}
	appendFile(t, dir, "scan.txt", " and part two") // size changes between ticks
	if r := scanReady(); len(r) != 0 {
		t.Fatalf("size changed between ticks, ready = %v, want none", r)
	}
	if r := scanReady(); !slices.Equal(r, []string{"scan.txt"}) {
		t.Fatalf("two agreeing ticks, ready = %v, want [scan.txt]", r)
	}
}

func TestScannerIgnoresDotfilesDirsAndTempSuffixes(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".hidden", "x")
	writeFile(t, dir, "a.part", "x")
	writeFile(t, dir, "b.tmp", "x")
	writeFile(t, dir, "c.crdownload", "x")
	writeFile(t, dir, "ok.txt", "x")
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o700); err != nil {
		t.Fatal(err)
	}
	sc := &inboxScanner{dir: dir}
	var ready []string
	var err error
	for range 2 {
		ready, err = sc.scan()
		if err != nil {
			t.Fatal(err)
		}
	}
	if !slices.Equal(ready, []string{"ok.txt"}) {
		t.Fatalf("ready = %v, want only ok.txt", ready)
	}
}

func TestScannerMissingDirIsEmpty(t *testing.T) {
	sc := &inboxScanner{dir: filepath.Join(t.TempDir(), "nope")}
	ready, err := sc.scan()
	if err != nil || len(ready) != 0 {
		t.Fatalf("scan = %v, %v; want empty, nil", ready, err)
	}
}

type changes struct {
	mu    sync.Mutex
	calls [][]string
}

func (c *changes) record(r []string) {
	c.mu.Lock()
	c.calls = append(c.calls, r)
	c.mu.Unlock()
}

func (c *changes) snapshot() [][]string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return slices.Clone(c.calls)
}

func waitUntil(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(testDeadline)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(testEvery)
	}
}

func TestWatchInboxReportsAppearAndDisappearOnly(t *testing.T) {
	dir := t.TempDir()
	var c changes
	ctx := t.Context()
	go WatchInbox(ctx, dir, testEvery, c.record, func(err error) { t.Error(err) })

	writeFile(t, dir, "a.txt", "hello")
	waitUntil(t, "a.txt ready", func() bool { return len(c.snapshot()) == 1 })
	time.Sleep(10 * testEvery) // idle ticks must not report again
	if got := c.snapshot(); len(got) != 1 || !slices.Equal(got[0], []string{"a.txt"}) {
		t.Fatalf("calls = %v, want one call [a.txt]", got)
	}
	if err := os.Remove(filepath.Join(dir, "a.txt")); err != nil {
		t.Fatal(err)
	}
	waitUntil(t, "a.txt gone", func() bool { return len(c.snapshot()) == 2 })
	if got := c.snapshot()[1]; len(got) != 0 {
		t.Fatalf("second call = %v, want empty", got)
	}
}

func TestListInboxSniffsAndPreviews(t *testing.T) {
	dir := t.TempDir()
	var sb strings.Builder
	for range 30 {
		sb.WriteString("short line\n")
	}
	long := strings.Repeat("x", 300) + "\n"
	writeFile(t, dir, "notes.txt", "\x1b[31mred\x1b[0m\n"+long+sb.String())
	writeFile(t, dir, "pic.png", "\x89PNG\r\n\x1a\nrest")
	writeFile(t, dir, "blob.bin", "\x00\x01\x02\x03")
	files, err := ListInbox(dir, []string{"blob.bin", "notes.txt", "pic.png", "gone.txt"})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 3 {
		t.Fatalf("files = %+v, want 3 (gone.txt skipped)", files)
	}
	blob, notes, pic := files[0], files[1], files[2]
	if blob.Kind != "" || blob.Preview != "" {
		t.Errorf("blob = %+v, want unsupported with no preview", blob)
	}
	if pic.Kind != model.EvidenceKindImage || pic.Preview != "" {
		t.Errorf("pic = %+v, want image with no preview", pic)
	}
	lines := strings.Split(notes.Preview, "\n")
	if notes.Kind != model.EvidenceKindText || len(lines) != previewLines || lines[0] != "red" {
		t.Errorf("notes kind=%q lines=%d first=%q", notes.Kind, len(lines), lines[0])
	}
	if len(lines[1]) != previewLineBytes {
		t.Errorf("line 1 len = %d, want %d", len(lines[1]), previewLineBytes)
	}
}

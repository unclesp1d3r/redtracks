//go:build unix

package evidence

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"syscall"
	"testing"

	"github.com/unclesp1d3r/redtracks/internal/model"
)

func TestListInboxSkipsSymlinkAndFifoWithoutOpening(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "real.txt", "hi")
	if err := os.Symlink(filepath.Join(dir, "real.txt"), filepath.Join(dir, "link.txt")); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Mkfifo(filepath.Join(dir, "pipe.txt"), 0o600); err != nil {
		t.Fatal(err)
	}
	// A FIFO opened for reading would block forever; this returning proves it was not opened.
	files, err := ListInbox(dir, []string{"link.txt", "pipe.txt", "real.txt"})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].Name != "real.txt" {
		t.Fatalf("files = %+v, want only real.txt", files)
	}
	sc := &inboxScanner{dir: dir}
	var ready []string
	for range 2 {
		if ready, err = sc.scan(); err != nil {
			t.Fatal(err)
		}
	}
	if len(ready) != 1 || ready[0] != "real.txt" {
		t.Fatalf("scan ready = %v, want only real.txt", ready)
	}
}

func TestListInboxKeepsUnreadableFileListedWithoutKind(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can open mode 000 files")
	}
	dir := t.TempDir()
	writeFile(t, dir, "locked.bin", "x")
	writeFile(t, dir, "ok.png", "\x89PNG\r\n\x1a\n"+strings.Repeat("\x00", 32))
	locked := filepath.Join(dir, "locked.bin")
	if err := os.Chmod(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { restoreMode(t, locked, 0o600) })

	files, err := ListInbox(dir, []string{"locked.bin", "ok.png"})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 || files[0].Name != "locked.bin" || files[0].Kind != "" ||
		files[1].Name != "ok.png" || files[1].Kind != model.EvidenceKindImage {
		t.Fatalf("files = %+v, want locked.bin (no kind) and ok.png (image)", files)
	}
}

func TestWatchInboxReportsRecoveryWithUnchangedFiles(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can read mode 000 directories")
	}
	dir := t.TempDir()
	writeFile(t, dir, "a.txt", "hello")
	var c changes
	var mu sync.Mutex
	var errs int
	go WatchInbox(t.Context(), dir, testEvery, c.record, func(error) {
		mu.Lock()
		errs++
		mu.Unlock()
	})
	waitUntil(t, "a.txt ready", func() bool { return len(c.snapshot()) == 1 })
	t.Cleanup(func() { restoreMode(t, dir, 0o700) })

	if err := os.Chmod(dir, 0); err != nil {
		t.Fatal(err)
	}
	waitUntil(t, "onError", func() bool { mu.Lock(); defer mu.Unlock(); return errs == 1 })
	restoreMode(t, dir, 0o700)

	waitUntil(t, "recovery onChange", func() bool { return len(c.snapshot()) == 2 })
	if got := c.snapshot()[1]; !slices.Equal(got, []string{"a.txt"}) {
		t.Fatalf("recovery call = %v, want [a.txt]", got)
	}
}

func restoreMode(t *testing.T, path string, mode os.FileMode) {
	t.Helper()
	if err := os.Chmod(path, mode); err != nil {
		t.Error(err)
	}
}

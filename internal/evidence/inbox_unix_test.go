//go:build unix

package evidence

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
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

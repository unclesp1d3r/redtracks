//go:build unix

package app

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestInboxNeverListsSymlinkOrFifo(t *testing.T) {
	f := openSample(t)
	svc := NewEvidenceService(f.app)
	target := writeInbox(t, f.app, "real.txt", "alpha")
	if err := os.Symlink(target, filepath.Join(f.app.inboxDir(), "link.txt")); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Mkfifo(filepath.Join(f.app.inboxDir(), "pipe.txt"), 0o600); err != nil {
		t.Fatal(err)
	}

	listedNames(t, svc, "real.txt")
}

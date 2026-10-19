//go:build !windows

package store

import (
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestImportEvidenceFileRefusesNamedPipe(t *testing.T) {
	dir := copyFixture(t)
	s := mustOpen(t, dir)
	fifo := filepath.Join(t.TempDir(), "fifo")
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		t.Skipf("mkfifo unavailable: %v", err)
	}
	_, err := s.ImportEvidenceFile(fixtureIP, fifo, "", nil)
	if err == nil || !strings.Contains(err.Error(), "named pipe") {
		t.Fatalf("err = %v, want a named pipe refusal", err)
	}
	if len(evFiles(t, dir)) != 0 || len(evidenceItems(s, fixtureIP)) != 0 {
		t.Fatal("a refused source wrote evidence")
	}
}

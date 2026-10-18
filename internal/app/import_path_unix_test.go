//go:build unix

package app

import (
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestPreviewPathRefusesFIFO(t *testing.T) {
	f := openSample(t)
	fifo := filepath.Join(t.TempDir(), "pipe.xml")
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := NewImportService(f.app).PreviewPath(fifo)
	if err == nil || !strings.Contains(err.Error(), "regular file") {
		t.Fatalf("got %v, want regular file refusal", err)
	}
}

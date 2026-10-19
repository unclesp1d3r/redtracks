//go:build !windows

package sessionlogs

import (
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestImportRefusesFIFO(t *testing.T) {
	p := filepath.Join(t.TempDir(), "pipe.log")
	if err := syscall.Mkfifo(p, 0o600); err != nil {
		t.Skip("mkfifo unavailable:", err)
	}
	_, err := Import(t.TempDir(), p)
	if err == nil || !strings.Contains(err.Error(), "named pipe") {
		t.Fatalf("err = %v, want named pipe refusal", err)
	}
}

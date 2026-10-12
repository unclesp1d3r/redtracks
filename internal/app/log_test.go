package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenLogRotatesAt5MB(t *testing.T) {
	path := filepath.Join(t.TempDir(), "redtracks.log")
	lg, closer, err := OpenLog(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = closer.Close() }()

	lg.Info(strings.Repeat("x", maxLogBytes))
	lg.Info("after")

	if _, err := os.Stat(path + ".1"); err != nil {
		t.Fatalf("rotated file missing: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() > 1024 {
		t.Fatalf("log not small after rotation: %d", info.Size())
	}
}

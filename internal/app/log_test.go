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

func TestLogRotationFailureKeepsLogging(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "redtracks.log")
	lg, closer, err := OpenLog(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = closer.Close() }()
	lg.Info(strings.Repeat("x", maxLogBytes))

	const readOnlyDir = 0o500
	if err := os.Chmod(dir, readOnlyDir); err != nil { // rename needs write access to the directory
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(dir, dirMode); err != nil {
			t.Error(err)
		}
	})
	lg.Info("first after failed rotation")
	lg.Info("second after failed rotation")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "second after failed rotation") {
		t.Fatal("log stopped accepting writes after a failed rotation")
	}
}

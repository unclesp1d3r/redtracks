package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteAtomicReplacesContent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a.json")
	if err := WriteAtomic(path, []byte("one")); err != nil {
		t.Fatal(err)
	}
	if err := WriteAtomic(path, []byte("two")); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "two" {
		t.Fatalf("got %q, want %q", got, "two")
	}
}

func TestWriteAtomicLeavesNoTempOnSuccess(t *testing.T) {
	dir := t.TempDir()
	if err := WriteAtomic(filepath.Join(dir, "a.json"), []byte("x")); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "a.json" {
		t.Fatalf("unexpected directory contents: %v", entries)
	}
}

func TestWriteAtomicSetsMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a.json")
	if err := WriteAtomic(path, []byte("x")); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if fileMode != 0o600 {
		t.Fatalf("fileMode = %v, want 0600", fileMode)
	}
	if info.Mode().Perm() != fileMode {
		t.Fatalf("mode %v, want %v", info.Mode().Perm(), fileMode)
	}
}

func TestWriteAtomicUnwritableDirKeepsOriginal(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "a.json")
	if err := WriteAtomic(path, []byte("original")); err != nil {
		t.Fatal(err)
	}
	const readOnlyDir = 0o500
	if err := os.Chmod(dir, readOnlyDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(dir, 0o700); err != nil { //nolint:gosec // directories need the execute bit
			t.Errorf("restore permissions: %v", err)
		}
	})

	if err := WriteAtomic(path, []byte("new")); err == nil {
		t.Fatal("expected error writing into read-only directory")
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "original" {
		t.Fatalf("original changed: %q", got)
	}
}

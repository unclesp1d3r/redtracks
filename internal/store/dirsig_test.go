package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDirSignatureTracksFilesAndIgnoresDotfiles(t *testing.T) {
	dir := t.TempDir()
	sig := func() string {
		t.Helper()
		s, err := DirSignature(dir, nil)
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	empty := sig()
	mustWrite(t, filepath.Join(dir, ".hidden"), "x")
	if got := sig(); got != empty {
		t.Fatalf("dotfile changed signature: %q", got)
	}
	file := filepath.Join(dir, "a.png")
	mustWrite(t, file, "1")
	added := sig()
	if added == empty {
		t.Fatal("adding a file did not change the signature")
	}
	mustWrite(t, file, "12")
	grown := sig()
	if grown == added {
		t.Fatal("growing a file did not change the signature")
	}
	bumpMtime(t, file)
	touched := sig()
	if touched == grown {
		t.Fatal("touching a file did not change the signature")
	}
	if err := os.Remove(file); err != nil {
		t.Fatal(err)
	}
	if got := sig(); got != empty {
		t.Fatalf("removing the file left signature %q, want %q", got, empty)
	}
}

func TestDirSignatureKeepFilterAndMissingDir(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "a.json"), "{}")
	mustWrite(t, filepath.Join(dir, "b.txt"), "x")
	got, err := DirSignature(dir, func(name string) bool { return strings.HasSuffix(name, ".json") })
	if err != nil || !strings.Contains(got, "a.json") || strings.Contains(got, "b.txt") {
		t.Fatalf("signature = %q, %v", got, err)
	}
	if got, err := DirSignature(filepath.Join(dir, "nope"), nil); err != nil || got != "" {
		t.Fatalf("missing dir = %q, %v; want empty", got, err)
	}
}

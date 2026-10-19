package sessionlogs

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/unclesp1d3r/redtracks/internal/store"
)

func fileTime(t *testing.T) time.Time {
	t.Helper()
	return time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
}

func TestImportRefusesRelativePath(t *testing.T) {
	_, err := Import(t.TempDir(), "relative/session.log")
	if err == nil || !strings.Contains(err.Error(), "absolute") {
		t.Fatalf("err = %v, want absolute-path refusal", err)
	}
}

func TestImportRefusesDirectory(t *testing.T) {
	_, err := Import(t.TempDir(), t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "directory") {
		t.Fatalf("err = %v, want directory refusal", err)
	}
}

func TestImportRefusesSymlink(t *testing.T) {
	target := writeFile(t, "hello\n")
	link := filepath.Join(t.TempDir(), "link.log")
	if err := os.Symlink(target, link); err != nil {
		t.Skip("symlinks unavailable:", err)
	}
	dir := t.TempDir()
	_, err := Import(dir, link)
	if err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("err = %v, want symbolic link refusal", err)
	}
	entries, readErr := os.ReadDir(store.LogsDir(dir))
	if readErr != nil && !errors.Is(readErr, fs.ErrNotExist) {
		t.Fatal(readErr)
	}
	if len(entries) != 0 {
		t.Fatalf("refused import left files: %v", entries)
	}
}

func TestImportRecordsSourceFacts(t *testing.T) {
	p := writeFile(t, numbered(10))
	want := fileTime(t)
	if err := os.Chtimes(p, want, want); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	src, err := Import(dir, p)
	if err != nil {
		t.Fatal(err)
	}
	if src.OriginalName != "session.log" || src.Lines != 10 || !src.ModTime.Equal(want) ||
		!strings.HasSuffix(src.Name, ".log") || src.ID+".log" != src.Name {
		t.Fatalf("source = %+v", src)
	}
}

func TestListKeepsOriginalNameAfterImport(t *testing.T) {
	dir := t.TempDir()
	src, err := Import(dir, writeFile(t, numbered(3)))
	if err != nil {
		t.Fatal(err)
	}
	listed, err := List(dir)
	if err != nil || len(listed) != 1 {
		t.Fatalf("List = %v, %v; want one source (the sidecar is not a log)", listed, err)
	}
	if listed[0].OriginalName != "session.log" || listed[0].Name != src.Name {
		t.Fatalf("listed = %+v, want original name session.log and stored name %s", listed[0], src.Name)
	}
}

func TestListFallsBackToStoredNameWithoutUsableSidecar(t *testing.T) {
	cases := map[string]func(t *testing.T, sidecarPath string){
		"missing": func(t *testing.T, p string) {
			t.Helper()
			if err := os.Remove(p); err != nil {
				t.Fatal(err)
			}
		},
		"corrupt": func(t *testing.T, p string) {
			t.Helper()
			if err := os.WriteFile(p, []byte("{not json"), 0o600); err != nil {
				t.Fatal(err)
			}
		},
	}
	for name, damage := range cases {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			src, err := Import(dir, writeFile(t, numbered(3)))
			if err != nil {
				t.Fatal(err)
			}
			damage(t, filepath.Join(store.LogsDir(dir), src.ID+".json"))
			listed, err := List(dir)
			if err != nil || len(listed) != 1 {
				t.Fatalf("List = %v, %v; want one source", listed, err)
			}
			if listed[0].OriginalName != src.Name {
				t.Fatalf("OriginalName = %q, want stored name %q", listed[0].OriginalName, src.Name)
			}
		})
	}
}

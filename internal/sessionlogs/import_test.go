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

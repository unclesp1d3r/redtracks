package sessionlogs

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestClipRefusesBadRanges(t *testing.T) {
	l := importText(t, numbered(10))
	for _, c := range []struct{ from, to int }{{-1, 3}, {5, 2}, {0, maxClipLines}, {0, 10}} {
		if _, err := l.clipText(c.from, c.to); err == nil {
			t.Errorf("clipText(%d,%d): want error", c.from, c.to)
		}
	}
}

func TestClipTextJoinsStrippedLines(t *testing.T) {
	l := importText(t, "a\n\x1b[1mb\x1b[0m\nc\nd\n")
	got, err := l.clipText(1, 2)
	if err != nil || got != "b\nc\n" {
		t.Fatalf("clipText = %q, %v", got, err)
	}
	if !strings.HasSuffix(got, "\n") {
		t.Fatal("clip must end with a newline")
	}
}

// A clip keeps a selected line whole past the 64 KiB view limit; only the view is cut.
func TestClipKeepsLongLinesWhilePageTruncates(t *testing.T) {
	dir := t.TempDir()
	long := strings.Repeat("x", maxLineBytes+1024)
	src := filepath.Join(dir, "long.log")
	if err := os.WriteFile(src, []byte("first\n"+long+"\nlast\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	engagement := filepath.Join(dir, "eng")
	s, err := Import(engagement, src)
	if err != nil {
		t.Fatal(err)
	}
	l, err := Open(engagement, s.ID)
	if err != nil {
		t.Fatal(err)
	}
	text, err := l.clipText(1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSuffix(text, "\n"); len(got) != len(long) {
		t.Fatalf("clip kept %d bytes of a %d byte line", len(got), len(long))
	}
	page, err := l.Page(1, 1)
	if err != nil || len(page) != 1 || len(page[0]) != maxLineBytes {
		t.Fatalf(
			"page = %d lines, first %d bytes, err %v; want one line cut at %d",
			len(page),
			len(page[0]),
			err,
			maxLineBytes,
		)
	}
}

// A raw line longer than the clip cap is refused even when stripping its control
// sequences would leave a short result; a truncated line must never be saved as proof.
func TestClipRefusesLineOverCapEvenWhenStrippedShort(t *testing.T) {
	dir := t.TempDir()
	var b strings.Builder
	b.WriteString("first\n")
	for b.Len() < maxClipBytes+4096 {
		b.WriteString("\x1b[31m\x1b[0m")
	}
	b.WriteString("the proof\nlast\n")
	src := filepath.Join(dir, "ansi.log")
	if err := os.WriteFile(src, []byte(b.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	engagement := filepath.Join(dir, "eng")
	s, err := Import(engagement, src)
	if err != nil {
		t.Fatal(err)
	}
	l, err := Open(engagement, s.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := l.clipText(1, 1); !errors.Is(err, errClipTooBig) {
		t.Fatalf("clip of an over-cap line returned %v, want errClipTooBig", err)
	}
}

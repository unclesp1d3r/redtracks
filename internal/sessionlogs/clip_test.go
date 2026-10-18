package sessionlogs

import (
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

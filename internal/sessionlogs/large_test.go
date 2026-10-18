package sessionlogs

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const (
	megabyte        = 1 << 20
	genLine         = "2026-10-12T22:57:00Z INFO scan host=10.10.10.5 port=445 state=open service=microsoft-ds\n"
	largeEnvVar     = "REDTRACKS_LONG_LOGS"
	mediumLogMB     = 50
	largeLogMB      = 300
	needleLineIndex = 123_456
)

// generatedLog names a synthetic log file and its line count.
type generatedLog struct {
	path  string
	lines int
}

// generateLog writes repeated lines totalling at least mb megabytes and
// returns the path and line count. Line needleLineIndex is distinctive.
func generateLog(t *testing.T, mb int) generatedLog {
	t.Helper()
	p := filepath.Join(t.TempDir(), "gen.log")
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	w := bufio.NewWriterSize(f, megabyte)
	lines := 0
	for written := 0; written < mb*megabyte; lines++ {
		line := genLine
		if lines == needleLineIndex {
			line = "NEEDLE here\n"
		}
		n, err := w.WriteString(line)
		if err != nil {
			t.Fatal(err)
		}
		written += n
	}
	if err := w.Flush(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return generatedLog{path: p, lines: lines}
}

func TestLargeLogs(t *testing.T) {
	if testing.Short() {
		t.Skip("long log cases skipped with -short")
	}
	for _, mb := range []int{mediumLogMB, largeLogMB} {
		t.Run(fmt.Sprintf("%dMB", mb), func(t *testing.T) {
			if mb == largeLogMB && os.Getenv(largeEnvVar) == "" {
				t.Skipf("set %s=1 to run the %d MB case (about 20 s under -race)", largeEnvVar, mb)
			}
			runLargeCase(t, mb)
		})
	}
}

func runLargeCase(t *testing.T, mb int) {
	t.Helper()
	gen := generateLog(t, mb)
	dir := t.TempDir()
	start := time.Now()
	src, err := Import(dir, gen.path)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%d MB import (copy + index): %v", mb, time.Since(start))
	if src.Lines != gen.lines {
		t.Fatalf("Lines = %d, want %d", src.Lines, gen.lines)
	}
	start = time.Now()
	l, err := Open(dir, src.ID)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%d MB Open (index build): %v", mb, time.Since(start))
	got, err := l.Page(needleLineIndex, 1)
	if err != nil || len(got) != 1 || got[0] != "NEEDLE here" {
		t.Fatalf("Page = %q, %v", got, err)
	}
	hits, err := l.Search("needle", -1, 5)
	if err != nil || len(hits) != 1 || hits[0].Line != needleLineIndex {
		t.Fatalf("Search = %+v, %v", hits, err)
	}
}

package sessionlogs

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/unclesp1d3r/redtracks/internal/store"
)

// writeFile writes content to a new file under t.TempDir and returns its path.
func writeFile(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "session.log")
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// importText imports content into a fresh engagement directory and opens it.
func importText(t *testing.T, content string) *Log {
	t.Helper()
	dir := t.TempDir()
	src, err := Import(dir, writeFile(t, content))
	if err != nil {
		t.Fatal(err)
	}
	l, err := Open(dir, src.ID)
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func numbered(n int) string {
	var b strings.Builder
	for i := range n {
		fmt.Fprintf(&b, "line %d\n", i)
	}
	return b.String()
}

func TestPageReturnsStrippedTextAndFileKeepsRawBytes(t *testing.T) {
	raw := "\x1b[31mred\x1b[0m text\n\x1b[2Kcursor\x1b[1;1H done\n"
	l := importText(t, raw)

	got, err := l.Page(0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"red text", "cursor done"}; !equalStrings(got, want) {
		t.Fatalf("Page = %q, want %q", got, want)
	}
	onDisk, err := os.ReadFile(l.path)
	if err != nil {
		t.Fatal(err)
	}
	if string(onDisk) != raw {
		t.Fatalf("log on disk was altered: %q", onDisk)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestIndexHandlesCRLFAndLoneCR(t *testing.T) {
	l := importText(t, "one\r\ntwo\rTWO\nthree")
	got, err := l.Page(0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"one", "TWO", "three"}; !equalStrings(got, want) {
		t.Fatalf("Page = %q, want %q", got, want)
	}
	if l.Lines() != 3 {
		t.Fatalf("Lines = %d, want 3 (last line has no newline)", l.Lines())
	}
}

func TestPageAcrossCheckpointsAndTail(t *testing.T) {
	n := checkpointEvery*3 + 5
	l := importText(t, numbered(n))
	if l.Lines() != n {
		t.Fatalf("Lines = %d, want %d", l.Lines(), n)
	}
	got, err := l.Page(checkpointEvery*3+2, 50)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		fmt.Sprintf("line %d", checkpointEvery*3+2), fmt.Sprintf("line %d", checkpointEvery*3+3),
		fmt.Sprintf("line %d", checkpointEvery*3+4),
	}
	if !equalStrings(got, want) {
		t.Fatalf("tail Page = %q, want %q", got, want)
	}
	mid, err := l.Page(checkpointEvery-1, 3)
	if err != nil || !equalStrings(mid, []string{"line 127", "line 128", "line 129"}) {
		t.Fatalf("mid Page = %q, %v", mid, err)
	}
	past, err := l.Page(n+10, 5)
	if err != nil || len(past) != 0 {
		t.Fatalf("past-end Page = %q, %v", past, err)
	}
}

func TestEmptyFileHasZeroLines(t *testing.T) {
	l := importText(t, "")
	got, err := l.Page(0, 10)
	if err != nil || l.Lines() != 0 || len(got) != 0 {
		t.Fatalf("empty: lines=%d page=%q err=%v", l.Lines(), got, err)
	}
}

func TestOverlongLineIsTruncatedInView(t *testing.T) {
	l := importText(t, strings.Repeat("x", maxLineBytes*3)+"\nnext\n")
	got, err := l.Page(0, 2)
	if err != nil || len(got) != 2 || len(got[0]) != maxLineBytes || got[1] != "next" {
		t.Fatalf("Page lens = %v, err %v", lens(got), err)
	}
}

func lens(s []string) []int {
	out := make([]int, len(s))
	for i, v := range s {
		out[i] = len(v)
	}
	return out
}

func TestSearchOrderAfterLineLimitAndCase(t *testing.T) {
	l := importText(t, "Alpha\nbeta\nALPHA two\ngamma\nalpha three\n")
	all, err := l.Search("alpha", -1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 || all[0].Line != 0 || all[1].Line != 2 || all[2].Line != 4 || all[1].Snippet != "ALPHA two" {
		t.Fatalf("matches = %+v", all)
	}
	after, err := l.Search("alpha", 0, 10)
	if err != nil || len(after) != 2 || after[0].Line != 2 {
		t.Fatalf("afterLine matches = %+v", after)
	}
	limited, err := l.Search("alpha", -1, 1)
	if err != nil || len(limited) != 1 || limited[0].Line != 0 {
		t.Fatalf("limited matches = %+v", limited)
	}
	if _, err := l.Search("", -1, 10); err == nil {
		t.Fatal("empty query: want error")
	}
}

// countingReader records the largest single Read request.
type countingReader struct {
	r       io.Reader
	maxRead int
}

func (c *countingReader) Read(p []byte) (int, error) {
	c.maxRead = max(c.maxRead, len(p))
	return c.r.Read(p)
}

// failingReader errors after n bytes.
type failingReader struct {
	r io.Reader
	n int
}

var errBoom = errors.New("boom")

func (f *failingReader) Read(p []byte) (int, error) {
	if f.n <= 0 {
		return 0, errBoom
	}
	p = p[:min(len(p), f.n)]
	n, err := f.r.Read(p)
	f.n -= n
	return n, err
}

func TestCopyReadsAreBounded(t *testing.T) {
	data := bytes.Repeat([]byte("some log line\n"), 200_000) // ~2.8 MB
	cr := &countingReader{r: bytes.NewReader(data)}
	dir := t.TempDir()
	src, err := importReader(dir, cr, "big.log", fileTime(t))
	if err != nil {
		t.Fatal(err)
	}
	if cr.maxRead > maxCopyRead {
		t.Fatalf("largest Read = %d, want <= %d", cr.maxRead, maxCopyRead)
	}
	if !strings.HasPrefix(filepath.Join(store.LogsDir(dir), src.Name), store.LogsDir(dir)) ||
		src.Size != int64(len(data)) {
		t.Fatalf("source = %+v", src)
	}
	copied, err := os.Open(filepath.Join(store.LogsDir(dir), src.Name))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = copied.Close() }()
	h := sha256.New()
	if _, err := io.Copy(h, copied); err != nil {
		t.Fatal(err)
	}
	if want := sha256.Sum256(data); !bytes.Equal(h.Sum(nil), want[:]) {
		t.Fatal("copy differs from source")
	}
}

func TestFailureMidCopyLeavesNoFile(t *testing.T) {
	dir := t.TempDir()
	fr := &failingReader{r: bytes.NewReader(bytes.Repeat([]byte("x\n"), 100_000)), n: 50_000}
	if _, err := importReader(dir, fr, "x.log", fileTime(t)); !errors.Is(err, errBoom) {
		t.Fatalf("err = %v, want errBoom", err)
	}
	entries, err := os.ReadDir(store.LogsDir(dir))
	if err != nil || len(entries) != 0 {
		t.Fatalf("logs dir entries = %v, err %v", entries, err)
	}
}

func TestListRebuildsSourcesFromDirectory(t *testing.T) {
	dir := t.TempDir()
	src, err := Import(dir, writeFile(t, numbered(300)))
	if err != nil {
		t.Fatal(err)
	}
	got, err := List(dir)
	if err != nil || len(got) != 1 || got[0].ID != src.ID || got[0].Lines != 300 {
		t.Fatalf("List = %+v, err %v", got, err)
	}
	if _, err := Open(dir, "../../etc/passwd"); err == nil {
		t.Fatal("Open with traversal id: want error")
	}
}

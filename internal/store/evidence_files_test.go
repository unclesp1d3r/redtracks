package store

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/unclesp1d3r/redtracks/internal/evidence"
	"github.com/unclesp1d3r/redtracks/internal/model"
)

var (
	pngBytes  = append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte{1, 2, 3}, 50)...)
	jpegBytes = append([]byte("\xff\xd8\xff\xe0"), bytes.Repeat([]byte{4}, 40)...)
	gifBytes  = append([]byte("GIF89a"), bytes.Repeat([]byte{5}, 40)...)
	webpBytes = append([]byte("RIFF\x00\x00\x00\x00WEBPVP8 "), bytes.Repeat([]byte{6}, 40)...)
	captured  = time.Date(2026, 10, 12, 10, 30, 5, 0, time.UTC)
)

func addEv(t *testing.T, s *Store, data []byte, at time.Time, caption string) AddedEvidence {
	t.Helper()
	got, err := s.AddEvidenceFile(fixtureIP, bytes.NewReader(data), at, caption, nil)
	if err != nil {
		t.Fatalf("AddEvidenceFile: %v", err)
	}
	return got
}

func evFiles(t *testing.T, dir string) []string {
	t.Helper()
	names, err := listEvidenceFiles(evDirOf(dir))
	if err != nil {
		t.Fatal(err)
	}
	return names
}

func TestAddEvidenceFileKindsAndExtensions(t *testing.T) {
	cases := []struct {
		name string
		data []byte
		kind model.EvidenceKind
		file string
	}{
		{"png", pngBytes, model.EvidenceKindImage, "20261012-103005-image-1.png"},
		{"jpeg", jpegBytes, model.EvidenceKindImage, "20261012-103005-image-1.jpg"},
		{"gif", gifBytes, model.EvidenceKindImage, "20261012-103005-image-1.gif"},
		{"webp", webpBytes, model.EvidenceKindImage, "20261012-103005-image-1.webp"},
		{"text", []byte("whoami\nroot\n"), model.EvidenceKindText, "20261012-103005-text-1.txt"},
	}
	for _, c := range cases {
		dir := copyFixture(t)
		s := mustOpen(t, dir)
		got := addEv(t, s, c.data, captured, "")
		if got.Item.Kind != c.kind || got.Item.File != c.file || got.Duplicate {
			t.Errorf("%s: got %+v", c.name, got)
		}
		onDisk, err := os.ReadFile(filepath.Join(evDirOf(dir), c.file))
		if err != nil || !bytes.Equal(onDisk, c.data) {
			t.Errorf("%s: file bytes differ: %v", c.name, err)
		}
	}
}

func TestAddEvidenceFileRefusesSVGAndPDFWithoutWriting(t *testing.T) {
	dir := copyFixture(t)
	s := mustOpen(t, dir)
	for want, data := range map[string]string{"SVG": "<svg xmlns='x'/>", "PDF": "%PDF-1.4 body"} {
		_, err := s.AddEvidenceFile(fixtureIP, strings.NewReader(data), captured, "", nil)
		var u *evidence.UnsupportedError
		if !errors.As(err, &u) || u.Detected != want {
			t.Fatalf("%s: err = %v", want, err)
		}
	}
	if files := evFiles(t, dir); len(files) != 0 || len(evidenceItems(s, fixtureIP)) != 0 {
		t.Fatalf("files %v items %v written", files, evidenceItems(s, fixtureIP))
	}
}

func TestAddEvidenceFileSameSecondGetsNumberedNames(t *testing.T) {
	s := mustOpen(t, copyFixture(t))
	a := addEv(t, s, pngBytes, captured, "")
	b := addEv(t, s, jpegBytes[:0:0], captured, "") // empty text, same kind differs
	c := addEv(t, s, pngBytes, captured, "")
	if a.Item.File != "20261012-103005-image-1.png" || c.Item.File != "20261012-103005-image-2.png" {
		t.Fatalf("names %q %q", a.Item.File, c.Item.File)
	}
	if b.Item.Kind != model.EvidenceKindText || len(evidenceItems(s, fixtureIP)) != 3 {
		t.Fatalf("items = %+v", evidenceItems(s, fixtureIP))
	}
}

func TestAddEvidenceFileCaptionStaysOutOfFileName(t *testing.T) {
	dir := copyFixture(t)
	s := mustOpen(t, dir)
	caption := `$(id) <img src=x onerror=alert(1)>`
	got := addEv(t, s, pngBytes, captured, caption)
	if got.Item.Caption != caption || evidenceItems(s, fixtureIP)[0].Caption != caption {
		t.Fatalf("caption not stored verbatim: %+v", got.Item)
	}
	for _, f := range evFiles(t, dir) {
		if strings.ContainsAny(f, "$(<>") || strings.Contains(f, "onerror") {
			t.Fatalf("file name %q carries caption text", f)
		}
	}
}

func TestAddEvidenceFileStoresGivenCaptureTime(t *testing.T) {
	dir := copyFixture(t)
	s := mustOpen(t, dir)
	got := addEv(t, s, pngBytes, captured, "")
	info, err := os.Stat(filepath.Join(evDirOf(dir), got.Item.File))
	if err != nil || !info.ModTime().Equal(captured) || !got.Item.CapturedAt.Equal(captured) {
		t.Fatalf("mtime %v capturedAt %v err %v", info.ModTime(), got.Item.CapturedAt, err)
	}
}

func srcFile(t *testing.T, data []byte, mtime time.Time) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "drop.png")
	if err := os.WriteFile(p, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(p, mtime, mtime); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestImportEvidenceFileCopiesAndKeepsSource(t *testing.T) {
	dir := copyFixture(t)
	s := mustOpen(t, dir)
	mtime := time.Now().Add(-48 * time.Hour).UTC().Truncate(time.Second)
	src := srcFile(t, pngBytes, mtime)
	got, err := s.ImportEvidenceFile(fixtureIP, src, "drop", nil)
	if err != nil || !got.Item.CapturedAt.Equal(mtime) {
		t.Fatalf("got %+v, %v; want capturedAt %v", got.Item, err, mtime)
	}
	if _, err := os.Stat(src); err != nil {
		t.Fatalf("source gone: %v", err)
	}
	info, err := os.Stat(filepath.Join(evDirOf(dir), got.Item.File))
	if err != nil || !info.ModTime().Equal(mtime) {
		t.Fatalf("copy mtime %v, %v", info.ModTime(), err)
	}
}

func TestImportEvidenceFileClampsFutureMtime(t *testing.T) {
	s := mustOpen(t, copyFixture(t))
	before := time.Now().Add(-time.Second)
	got, err := s.ImportEvidenceFile(fixtureIP, srcFile(t, pngBytes, time.Now().Add(72*time.Hour)), "", nil)
	if err != nil || got.Item.CapturedAt.Before(before) || got.Item.CapturedAt.After(time.Now().Add(time.Second)) {
		t.Fatalf("capturedAt %v, %v", got.Item.CapturedAt, err)
	}
}

func TestImportEvidenceFileRefusesNonRegularSources(t *testing.T) {
	dir := copyFixture(t)
	s := mustOpen(t, dir)
	tmp := t.TempDir()
	link := filepath.Join(tmp, "link.png")
	if err := os.Symlink(srcFile(t, pngBytes, captured), link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	cases := map[string]string{"symbolic link": link, "directory": tmp}
	for want, path := range cases {
		_, err := s.ImportEvidenceFile(fixtureIP, path, "", nil)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: err = %v", want, err)
		}
	}
	if len(evFiles(t, dir)) != 0 || len(evidenceItems(s, fixtureIP)) != 0 {
		t.Fatal("a refused source wrote evidence")
	}
}

func TestAddEvidenceFileSizeCap(t *testing.T) {
	for _, kind := range []string{"text", "image"} {
		prefix := "a"
		if kind == "image" {
			prefix = "\x89PNG\r\n\x1a\n"
		}
		fill := func(n int) []byte { return append([]byte(prefix), bytes.Repeat([]byte("a"), n-len(prefix))...) }
		dir := copyFixture(t)
		s := mustOpen(t, dir)
		_, err := s.AddEvidenceFile(fixtureIP, bytes.NewReader(fill(maxEvidenceBytes+1)), captured, "", nil)
		if err == nil || !strings.Contains(err.Error(), "larger than 20 MB") {
			t.Fatalf("%s over cap: err = %v", kind, err)
		}
		if len(evFiles(t, dir)) != 0 {
			t.Fatalf("%s over cap left files %v", kind, evFiles(t, dir))
		}
		atCap := bytes.NewReader(fill(maxEvidenceBytes))
		if _, err := s.AddEvidenceFile(fixtureIP, atCap, captured, "", nil); err != nil {
			t.Fatalf("%s at cap: %v", kind, err)
		}
	}
}

func TestAddEvidenceFileReportsDuplicateDigest(t *testing.T) {
	s := mustOpen(t, copyFixture(t))
	first := addEv(t, s, pngBytes, captured, "")
	second := addEv(t, s, pngBytes, captured, "")
	if first.Duplicate || !second.Duplicate || first.Item.SHA256 != second.Item.SHA256 || first.Item.SHA256 == "" {
		t.Fatalf("first %+v second %+v", first, second)
	}
	if len(evidenceItems(s, fixtureIP)) != 2 {
		t.Fatalf("items = %+v", evidenceItems(s, fixtureIP))
	}
}

func setHook(t *testing.T, stage string) {
	t.Helper()
	orig := evidenceHook
	t.Cleanup(func() { evidenceHook = orig })
	evidenceHook = func(got string) error {
		if got == stage {
			return errors.New("injected")
		}
		return nil
	}
}

func TestAddEvidenceFileFailureBeforeIndexRemovesFile(t *testing.T) {
	dir := copyFixture(t)
	s := mustOpen(t, dir)
	setHook(t, hookAfterFile)
	if _, err := s.AddEvidenceFile(fixtureIP, bytes.NewReader(pngBytes), captured, "", nil); err == nil {
		t.Fatal("want error")
	}
	if len(evFiles(t, dir)) != 0 || len(evidenceItems(s, fixtureIP)) != 0 {
		t.Fatalf("leftovers: %v %v", evFiles(t, dir), evidenceItems(s, fixtureIP))
	}
}

func TestAddEvidenceFileFailureAfterIndexKeepsBoth(t *testing.T) {
	dir := copyFixture(t)
	s := mustOpen(t, dir)
	setHook(t, hookAfterIndex)
	if _, err := s.AddEvidenceFile(fixtureIP, bytes.NewReader(pngBytes), captured, "", nil); err == nil {
		t.Fatal("want error")
	}
	if len(evFiles(t, dir)) != 1 || len(evidenceItems(s, fixtureIP)) != 1 {
		t.Fatalf("want file and entry: %v %v", evFiles(t, dir), evidenceItems(s, fixtureIP))
	}
}

func TestEvidenceBytesSurviveMetadataMutations(t *testing.T) {
	dir := copyFixture(t)
	s := mustOpen(t, dir)
	got := addEv(t, s, pngBytes, captured, "first")
	err := s.MutateEvidence(fixtureIP, func(f *model.EvidenceFile) error {
		f.Items[0].Caption = "second"
		f.Items[0].Links = append(f.Items[0].Links, model.EvidenceLink{Kind: model.EvidenceLinkFlag, TargetID: "f1"})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	onDisk, err := os.ReadFile(filepath.Join(evDirOf(dir), got.Item.File))
	if err != nil || !bytes.Equal(onDisk, pngBytes) {
		t.Fatalf("bytes changed: %v", err)
	}
}

func TestAddEvidenceFileStripsTextButNotImages(t *testing.T) {
	dir := copyFixture(t)
	s := mustOpen(t, dir)
	text := addEv(t, s, []byte("\x1b[31mred\x1b[0m\x1b[2K $(id) <img onerror=x>\n"), captured, "")
	got, err := os.ReadFile(filepath.Join(evDirOf(dir), text.Item.File))
	if err != nil || string(got) != "red $(id) <img onerror=x>\n" {
		t.Fatalf("text stored as %q, %v", got, err)
	}
	img := append(append([]byte{}, pngBytes...), "\x1b[31m"...)
	image := addEv(t, s, img, captured, "")
	got, err = os.ReadFile(filepath.Join(evDirOf(dir), image.Item.File))
	if err != nil || !bytes.Equal(got, img) {
		t.Fatalf("image bytes were altered: %v", err)
	}
}

func TestCreateExclusiveConcurrentCallersGetDistinctNames(t *testing.T) {
	dir := t.TempDir()
	const callers = 16
	names := make(chan string, callers)
	var wg sync.WaitGroup
	for range callers {
		wg.Go(func() {
			f, name, err := CreateExclusive(dir, "stem", "txt")
			if err != nil {
				t.Error(err)
				return
			}
			_ = f.Close()
			names <- name
		})
	}
	wg.Wait()
	close(names)
	seen := map[string]bool{}
	for n := range names {
		seen[n] = true
	}
	if len(seen) != callers || !seen["stem-1.txt"] || !seen["stem-16.txt"] {
		t.Fatalf("names = %v", seen)
	}
}

func TestCreateExclusiveNeverTruncates(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "stem-1.txt"), []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	f, name, err := CreateExclusive(dir, "stem", "txt")
	if err != nil || name != "stem-2.txt" {
		t.Fatalf("name %q, %v", name, err)
	}
	_ = f.Close()
	if data, err := os.ReadFile(filepath.Join(dir, "stem-1.txt")); err != nil || string(data) != "keep" {
		t.Fatalf("existing file changed: %q, %v", data, err)
	}
}

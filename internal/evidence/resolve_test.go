package evidence

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/unclesp1d3r/redtracks/internal/model"
)

var pngHead = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR")

func putFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestResolveServesValidFiles(t *testing.T) {
	root := t.TempDir()
	putFile(t, filepath.Join(root, "shot.png"), pngHead)
	putFile(t, filepath.Join(root, "sub", "a.html"), []byte("<html><script>alert(1)</script></html>"))

	img, err := Resolve(root, "shot.png")
	if err != nil {
		t.Fatalf("png: %v", err)
	}
	if img.Kind != model.EvidenceKindImage || img.ContentType != "image/png" || img.Size != int64(len(pngHead)) {
		t.Fatalf("png resolved as %+v", img)
	}
	txt, err := Resolve(root, "sub", "a.html")
	if err != nil {
		t.Fatalf("markup: %v", err)
	}
	if txt.Kind != model.EvidenceKindText || txt.ContentType != "text/plain; charset=utf-8" {
		t.Fatalf("markup text resolved as %+v", txt)
	}
}

func TestResolveRefusesUnsafeSegments(t *testing.T) {
	root := t.TempDir()
	putFile(t, filepath.Join(root, "ok.png"), pngHead)
	outside := filepath.Join(t.TempDir(), "secret.png")
	putFile(t, outside, pngHead)
	cases := map[string][]string{
		"empty":      {""},
		"none":       {},
		"dot":        {"."},
		"dotdot":     {".."},
		"traversal":  {"..", filepath.Base(filepath.Dir(outside)), "secret.png"},
		"absolute":   {outside},
		"separator":  {"sub/ok.png"},
		"backslash":  {`sub\ok.png`},
		"nul":        {"ok.png\x00.txt"},
		"empty-tail": {"ok.png", ""},
	}
	for name, segs := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := Resolve(root, segs...)
			if _, ok := errors.AsType[*RefusedError](err); !ok {
				t.Fatalf("Resolve(%q) = %v, want RefusedError", segs, err)
			}
		})
	}
}

func TestResolveMissingIsNotFound(t *testing.T) {
	root := t.TempDir()
	_, err := Resolve(root, "nope.png")
	var nf *NotFoundError
	if !errors.As(err, &nf) {
		t.Fatalf("got %v, want NotFoundError", err)
	}
	if _, err := Resolve(filepath.Join(root, "missing-root"), "x.png"); !errors.As(err, &nf) {
		t.Fatalf("missing root: got %v, want NotFoundError", err)
	}
}

func TestResolveRefusesUnsupportedTypes(t *testing.T) {
	root := t.TempDir()
	putFile(t, filepath.Join(root, "x.svg"), []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script/></svg>`))
	putFile(t, filepath.Join(root, "x.zip"), []byte("PK\x03\x04\x00\x00binary"))
	for _, name := range []string{"x.svg", "x.zip"} {
		_, err := Resolve(root, name)
		if _, ok := errors.AsType[*RefusedError](err); !ok {
			t.Fatalf("%s: got %v, want RefusedError", name, err)
		}
	}
}

func TestResolveRefusesDirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "d"), 0o750); err != nil {
		t.Fatal(err)
	}
	_, err := Resolve(root, "d")
	if _, ok := errors.AsType[*RefusedError](err); !ok {
		t.Fatalf("got %v, want RefusedError", err)
	}
}

func TestResolveRefusesSymlinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation needs privileges on Windows")
	}
	root := t.TempDir()
	outsideDir := t.TempDir()
	putFile(t, filepath.Join(outsideDir, "secret.png"), pngHead)
	if err := os.Symlink(filepath.Join(outsideDir, "secret.png"), filepath.Join(root, "link.png")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outsideDir, filepath.Join(root, "linkdir")); err != nil {
		t.Fatal(err)
	}
	for _, segs := range [][]string{{"link.png"}, {"linkdir", "secret.png"}} {
		_, err := Resolve(root, segs...)
		if _, ok := errors.AsType[*RefusedError](err); !ok {
			t.Fatalf("Resolve(%q) = %v, want RefusedError", segs, err)
		}
	}
}

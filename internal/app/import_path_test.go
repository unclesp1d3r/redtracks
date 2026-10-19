package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPreviewPathPreviewsNmapFile(t *testing.T) {
	f := openSample(t)
	abs, err := filepath.Abs(nmapFixture)
	if err != nil {
		t.Fatal(err)
	}
	got, err := NewImportService(f.app).PreviewPath(abs)
	if err != nil {
		t.Fatal(err)
	}
	if got.Token == "" {
		t.Fatalf("preview has no token: %+v", got)
	}
}

func TestPreviewPathRefusals(t *testing.T) {
	f := openSample(t)
	svc := NewImportService(f.app)
	dir := t.TempDir()
	big := filepath.Join(dir, "big.xml")
	if err := os.WriteFile(big, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Truncate(big, maxImportBytes+1); err != nil {
		t.Fatal(err)
	}
	cases := map[string]struct{ path, want string }{
		"relative":  {"nmap.xml", "absolute"},
		"directory": {dir, "regular file"},
		"too big":   {big, "larger than 20 MB"},
		"missing":   {filepath.Join(dir, "nope.xml"), "no such file"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := svc.PreviewPath(c.path)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("PreviewPath(%q) = %v, want error containing %q", c.path, err, c.want)
			}
		})
	}
}

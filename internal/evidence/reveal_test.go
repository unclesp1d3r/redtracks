package evidence

import (
	"slices"
	"testing"
)

func TestRevealArgs(t *testing.T) {
	tests := []struct {
		goos string
		want []string
	}{
		{"darwin", []string{"open", "-R", "/e/dir/a.png"}},
		{"windows", []string{"explorer", `/select,/e/dir/a.png`}},
		{"linux", []string{"xdg-open", "/e/dir"}},
	}
	for _, tc := range tests {
		got, err := RevealArgs(tc.goos, "/e/dir/a.png")
		if err != nil || !slices.Equal(got, tc.want) {
			t.Errorf("%s: got %v, %v want %v", tc.goos, got, err, tc.want)
		}
	}
	if _, err := RevealArgs("plan9", "/x"); err == nil {
		t.Error("unsupported OS accepted")
	}
}

func TestUnderRoot(t *testing.T) {
	tests := []struct {
		root, path string
		want       bool
	}{
		{"/eng", "/eng/hosts/a/evidence/x.png", true},
		{"/eng", "/eng", true},
		{"/eng", "/eng/../etc/passwd", false},
		{"/eng", "/other/x", false},
		{"/eng", "/engineer/x", false},
	}
	for _, tc := range tests {
		if got := UnderRoot(tc.root, tc.path); got != tc.want {
			t.Errorf("UnderRoot(%q, %q) = %v, want %v", tc.root, tc.path, got, tc.want)
		}
	}
}

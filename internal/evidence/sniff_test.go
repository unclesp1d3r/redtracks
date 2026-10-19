package evidence

import (
	"errors"
	"strings"
	"testing"

	"github.com/unclesp1d3r/redtracks/internal/model"
)

func TestSniffAcceptsKnownTypes(t *testing.T) {
	cases := []struct {
		name string
		head string
		want Sniffed
	}{
		{"png", "\x89PNG\r\n\x1a\n....", Sniffed{model.EvidenceKindImage, "png"}},
		{"jpeg", "\xff\xd8\xff\xe0....", Sniffed{model.EvidenceKindImage, "jpg"}},
		{"gif", "GIF89a....", Sniffed{model.EvidenceKindImage, "gif"}},
		{"webp", "RIFF\x00\x00\x00\x00WEBPVP8 ", Sniffed{model.EvidenceKindImage, "webp"}},
		{"text", "uid=0(root) $(id) <img onerror=x>\n", Sniffed{model.EvidenceKindText, "txt"}},
		{"text cut mid-rune", "caf\xc3", Sniffed{model.EvidenceKindText, "txt"}},
	}
	for _, c := range cases {
		got, err := Sniff([]byte(c.head))
		if err != nil || got != c.want {
			t.Errorf("%s: got %+v, %v; want %+v", c.name, got, err, c.want)
		}
	}
}

func TestSniffRefusesAndNamesDetectedType(t *testing.T) {
	cases := map[string]string{
		"SVG": `<?xml version="1.0"?><svg xmlns="x"></svg>`,
		"PDF": "%PDF-1.7\n",
	}
	for want, head := range cases {
		_, err := Sniff([]byte(head))
		var u *UnsupportedError
		if !errors.As(err, &u) || u.Detected != want || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: err = %v", want, err)
		}
	}
	if _, err := Sniff([]byte("text\x00binary")); err == nil {
		t.Error("NUL-bearing data accepted as text")
	}
	if _, err := Sniff([]byte{0xff, 0xfe, 'a', 0xff, 0xfe, 0xfd, 'b'}); err == nil {
		t.Error("invalid UTF-8 accepted as text")
	}
}

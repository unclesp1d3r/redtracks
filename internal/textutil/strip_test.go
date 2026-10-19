package textutil

import (
	"strings"
	"testing"
)

func TestStrip(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"sgr colour", "\x1b[1;31mred\x1b[0m ok", "red ok"},
		{"cursor movement", "a\x1b[2Kb\x1b[10;5Hc", "abc"},
		{"bracketed paste", "\x1b[?2004hls\x1b[?2004l\n", "ls\n"},
		{"osc bel", "\x1b]0;title\x07text", "text"},
		{"osc st", "\x1b]0;title\x1b\\text", "text"},
		{"charset", "\x1b(Bx", "x"},
		{"backspace overwrite", "a\bbc", "bc"},
		{"man page bold", "_\bX", "X"},
		{"cr overwrite", "10%\r100%\nnext", "100%\nnext"},
		{"cr partial overwrite", "hello\rhi", "hillo"},
		{"crlf", "a\r\nb", "a\nb"},
		{"tab kept, bell dropped", "a\tb\x07", "a\tb"},
		{"invalid utf8", "a\xffb", "a�b"},
		{"trailing lone esc", "a\x1b", "a"},
		{"shell metachars kept", "$(id) <img onerror=x>", "$(id) <img onerror=x>"},
	}
	for _, c := range cases {
		if got := string(Strip([]byte(c.in))); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}

func TestStripUnterminatedOSCIsBounded(t *testing.T) {
	in := "\x1b]0;" + strings.Repeat("x", maxStringSeq*2) + "tail"
	got := string(Strip([]byte(in)))
	if !strings.HasSuffix(got, "tail") || len(got) >= len(in) || len(got) < maxStringSeq/2 {
		t.Fatalf("got %d bytes, want the tail kept after the bound", len(got))
	}
}

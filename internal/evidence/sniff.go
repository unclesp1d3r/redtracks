// Package evidence decides which files may be stored as evidence.
package evidence

import (
	"bytes"
	"fmt"
	"net/http"
	"unicode/utf8"

	"github.com/unclesp1d3r/redtracks/internal/model"
)

// SniffLen is how many leading bytes Sniff needs.
const SniffLen = 512

// maxRuneLen is the longest UTF-8 sequence; a head cut inside one is still text.
const maxRuneLen = utf8.UTFMax

// Sniffed is the accepted type of an evidence file.
type Sniffed struct {
	Kind model.EvidenceKind
	Ext  string // without the dot
}

// UnsupportedError refuses a file whose content is not an accepted evidence type.
type UnsupportedError struct {
	Detected string
}

func (e *UnsupportedError) Error() string {
	return fmt.Sprintf("unsupported evidence type %s: only PNG, JPEG, GIF, WebP images and UTF-8 text are accepted",
		e.Detected)
}

// Sniff classifies head, the first SniffLen bytes of a file, by magic bytes.
func Sniff(head []byte) (Sniffed, error) {
	switch {
	case bytes.HasPrefix(head, []byte("\x89PNG\r\n\x1a\n")):
		return Sniffed{model.EvidenceKindImage, "png"}, nil
	case bytes.HasPrefix(head, []byte("\xff\xd8\xff")):
		return Sniffed{model.EvidenceKindImage, "jpg"}, nil
	case bytes.HasPrefix(head, []byte("GIF87a")), bytes.HasPrefix(head, []byte("GIF89a")):
		return Sniffed{model.EvidenceKindImage, "gif"}, nil
	case len(head) >= 12 && bytes.Equal(head[:4], []byte("RIFF")) && bytes.Equal(head[8:12], []byte("WEBP")):
		return Sniffed{model.EvidenceKindImage, "webp"}, nil
	case bytes.HasPrefix(head, []byte("%PDF-")):
		return Sniffed{}, &UnsupportedError{Detected: "PDF"}
	case isSVG(head):
		return Sniffed{}, &UnsupportedError{Detected: "SVG"}
	case isText(head):
		return Sniffed{model.EvidenceKindText, "txt"}, nil
	}
	return Sniffed{}, &UnsupportedError{Detected: http.DetectContentType(head)}
}

func isSVG(head []byte) bool {
	trimmed := bytes.TrimLeft(head, "\xef\xbb\xbf \t\r\n")
	return bytes.HasPrefix(trimmed, []byte("<svg")) ||
		(bytes.HasPrefix(trimmed, []byte("<?xml")) && bytes.Contains(head, []byte("<svg")))
}

// isText reports whether head is NUL-free UTF-8, forgiving a rune cut off at the end.
func isText(head []byte) bool {
	if bytes.IndexByte(head, 0) >= 0 {
		return false
	}
	for cut := 0; cut < maxRuneLen && cut <= len(head); cut++ {
		if utf8.Valid(head[:len(head)-cut]) {
			return true
		}
	}
	return false
}

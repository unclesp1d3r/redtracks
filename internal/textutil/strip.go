// Package textutil holds small text-cleaning helpers.
package textutil

import (
	"unicode/utf8"
)

const (
	esc = 0x1b
	bel = 0x07
	bs  = 0x08
	del = 0x7f

	// maxStringSeq bounds an OSC/DCS/APC/PM string so an unterminated one cannot swallow a file.
	maxStringSeq = 4096

	c1Start = 0x80
	c1End   = 0x9f

	introLen   = 2 // ESC plus the introducer byte
	charsetLen = 3 // ESC, the charset designator, and the charset name
)

// Strip returns data as plain text: terminal escape sequences are removed,
// backspace and carriage-return overwrites are resolved within each line, other
// control characters are dropped, and invalid UTF-8 becomes U+FFFD. Tabs and
// newlines are kept.
func Strip(data []byte) []byte {
	out := make([]byte, 0, len(data))
	var ln lineBuf
	for i := 0; i < len(data); {
		b := data[i]
		switch {
		case b == esc:
			i = skipEscape(data, i)
			continue
		case b == '\n':
			out = append(out, string(ln.runes)...)
			out = append(out, '\n')
			ln = lineBuf{runes: ln.runes[:0]}
		case b == '\r':
			ln.col = 0
		case b == bs:
			ln.col = max(ln.col-1, 0)
		case b < ' ' && b != '\t', b == del:
			// dropped control byte
		default:
			r, size := utf8.DecodeRune(data[i:])
			i += size
			if r >= c1Start && r <= c1End {
				continue
			}
			ln.put(r)
			continue
		}
		i++
	}
	return append(out, string(ln.runes)...)
}

// lineBuf is the current output line with a cursor, so overwrites can be resolved.
type lineBuf struct {
	runes []rune
	col   int
}

// put writes r at the cursor, overwriting what is there, and advances the cursor.
func (l *lineBuf) put(r rune) {
	if l.col < len(l.runes) {
		l.runes[l.col] = r
	} else {
		l.runes = append(l.runes, r)
	}
	l.col++
}

// skipEscape returns the index after the escape sequence starting at data[i].
func skipEscape(data []byte, i int) int {
	if i+1 >= len(data) {
		return i + 1
	}
	switch data[i+1] {
	case '[':
		return skipCSI(data, i+introLen)
	case ']', 'P', '_', '^', 'X':
		return skipString(data, i+introLen)
	case '(', ')', '*', '+', '#', '%':
		return min(i+charsetLen, len(data))
	}
	return i + introLen
}

// skipCSI consumes parameter and intermediate bytes up to the final byte. A byte
// outside the CSI alphabet aborts the sequence and is left for the caller.
func skipCSI(data []byte, i int) int {
	for ; i < len(data); i++ {
		switch b := data[i]; {
		case b >= 0x40 && b <= 0x7e:
			return i + 1
		case b < 0x20 || b > 0x3f:
			return i
		}
	}
	return i
}

// skipString consumes a string sequence through BEL or ST (ESC \), giving up
// after maxStringSeq bytes so the rest of the input survives.
func skipString(data []byte, i int) int {
	limit := min(len(data), i+maxStringSeq)
	for j := i; j < limit; j++ {
		switch {
		case data[j] == bel:
			return j + 1
		case data[j] == esc && j+1 < len(data) && data[j+1] == '\\':
			return j + introLen
		}
	}
	return limit
}

package sessionlogs

import (
	"errors"
	"strings"
	"unicode/utf8"
)

const (
	// maxSearchLimit caps the matches one Search call returns.
	maxSearchLimit = 500
	// maxSnippetBytes bounds the text of one match.
	maxSnippetBytes = 256
)

var errEmptyQuery = errors.New("search query is empty")

// Match is one search hit.
type Match struct {
	Line    int // 0-based
	Snippet string
}

// Search returns up to limit (capped at maxSearchLimit) lines after afterLine
// (exclusive; -1 starts at the top) whose stripped text contains query,
// ignoring case, in line order.
func (l *Log) Search(query string, afterLine, limit int) ([]Match, error) {
	if query == "" {
		return nil, errEmptyQuery
	}
	limit = min(max(limit, 0), maxSearchLimit)
	needle := strings.ToLower(query)
	out := []Match{}
	if limit == 0 {
		return out, nil
	}
	err := l.scan(max(afterLine+1, 0), func(n int, line []byte) bool {
		text := view(line)
		if strings.Contains(strings.ToLower(text), needle) {
			out = append(out, Match{Line: n, Snippet: truncateRunes(text, maxSnippetBytes)})
		}
		return len(out) < limit
	})
	return out, err
}

// truncateRunes cuts s to at most limit bytes on a rune boundary.
func truncateRunes(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	cut := limit
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut]
}

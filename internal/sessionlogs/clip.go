package sessionlogs

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/unclesp1d3r/redtracks/internal/model"
	"github.com/unclesp1d3r/redtracks/internal/store"
)

const (
	// maxClipLines caps one clip.
	maxClipLines = 5000
	// maxClipBytes bounds the clip text held in memory; the store enforces its own 20 MB evidence cap.
	maxClipBytes = 20 << 20
)

var errClipTooBig = errors.New("clip is larger than 20 MB; choose fewer lines")

// Clip saves lines from..to (inclusive, 0-based) of the opened log as a text
// evidence item on the host at ip. The log on disk is never changed.
func (l *Log) Clip(st *store.Store, ip string, src Source, from, to int) (model.EvidenceItem, error) {
	text, err := l.clipText(from, to)
	if err != nil {
		return model.EvidenceItem{}, err
	}
	name := src.OriginalName
	if name == "" {
		name = src.Name
	}
	caption := fmt.Sprintf("%s lines %d-%d", name, from+1, to+1)
	clip := model.ClipOf{LogID: src.ID, FirstLine: from, LastLine: to}
	added, err := st.AddEvidenceClip(ip, strings.NewReader(text), time.Now().UTC(), caption, nil, clip)
	if err != nil {
		return model.EvidenceItem{}, err
	}
	return added.Item, nil
}

// clipText returns the stripped lines from..to joined with newlines, ending in one.
func (l *Log) clipText(from, to int) (string, error) {
	switch {
	case from < 0 || to < 0:
		return "", fmt.Errorf("clip range %d-%d has a negative line", from, to)
	case from > to:
		return "", fmt.Errorf("clip range starts at line %d after it ends at %d", from, to)
	case to-from >= maxClipLines:
		return "", fmt.Errorf("clip of %d lines is over the %d line limit", to-from+1, maxClipLines)
	case to >= l.lines:
		return "", fmt.Errorf("clip ends at line %d but the log has %d lines", to, l.lines)
	}
	var buf bytes.Buffer
	var tooBig bool
	err := l.scan(from, func(n int, line []byte) bool {
		buf.WriteString(view(line))
		buf.WriteByte('\n')
		tooBig = buf.Len() > maxClipBytes
		return n < to && !tooBig
	})
	if err == nil && tooBig {
		err = errClipTooBig
	}
	return buf.String(), err
}

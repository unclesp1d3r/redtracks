package sessionlogs

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/unclesp1d3r/redtracks/internal/store"
	"github.com/unclesp1d3r/redtracks/internal/textutil"
)

const (
	// checkpointEvery is the line spacing of the sparse offset table.
	checkpointEvery = 128
	// maxLineBytes bounds one line in a view; the rest of a longer line is dropped.
	maxLineBytes = 64 << 10
	scanBufBytes = 64 << 10
)

// Log is an imported session log with its sparse line index. It is immutable
// after construction and safe for concurrent use.
type Log struct {
	path    string
	size    int64
	modTime time.Time
	lines   int
	offsets []int64 // offsets[k] is the byte offset of line k*checkpointEvery
}

// Open returns the log with the given id, building its index.
func Open(engagementDir, id string) (*Log, error) {
	if !idPattern.MatchString(id) {
		return nil, fmt.Errorf("invalid session log id %q", id)
	}
	return buildLog(filepath.Join(store.LogsDir(engagementDir), id+logFileSufix))
}

// buildLog scans path once on '\n' and records a byte offset every checkpointEvery lines.
func buildLog(path string) (*Log, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open session log: %w", err)
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", path, err)
	}
	l := &Log{path: path, size: info.Size(), modTime: info.ModTime()}
	buf := make([]byte, scanBufBytes)
	var pos int64
	var last byte
	var newlines int
	for {
		n, err := f.Read(buf)
		chunk := buf[:n]
		for off := 0; ; {
			i := bytes.IndexByte(chunk[off:], '\n')
			if i < 0 {
				break
			}
			off += i + 1
			newlines++
			if newlines%checkpointEvery == 0 {
				l.offsets = append(l.offsets, pos+int64(off))
			}
		}
		if len(chunk) > 0 {
			last = chunk[len(chunk)-1]
		}
		pos += int64(n)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("index %s: %w", path, err)
		}
	}
	l.lines = newlines
	if l.size > 0 && last != '\n' {
		l.lines++
	}
	l.offsets = l.trimOffsets()
	return l, nil
}

// trimOffsets prepends line 0 and drops any trailing checkpoint that points
// past the last line (a file ending exactly on a checkpoint boundary).
func (l *Log) trimOffsets() []int64 {
	out := make([]int64, 0, len(l.offsets)+1)
	out = append(out, 0)
	for k, off := range l.offsets {
		if (k+1)*checkpointEvery < l.lines {
			out = append(out, off)
		}
	}
	return out
}

// Lines returns the number of lines, counting a final line with no newline.
func (l *Log) Lines() int { return l.lines }

// Source describes the log. OriginalName is the stored name, which is all a
// restart can know.
func (l *Log) Source() Source {
	name := filepath.Base(l.path)
	stamp, _, _ := strings.Cut(l.id(), "-")
	imported, err := time.Parse(stampLayout, stamp)
	if err != nil {
		imported = time.Time{} // an id that does not parse still lists, with no import time
	}
	return Source{
		ID: l.id(), Name: name, OriginalName: name, Size: l.size, ModTime: l.modTime,
		ImportedAt: imported, Lines: l.lines,
	}
}

func (l *Log) id() string { return strings.TrimSuffix(filepath.Base(l.path), logFileSufix) }

// scan calls fn with each line number and view bytes from line `from` on until
// fn returns false or the log ends. The bytes are only valid during the call.
func (l *Log) scan(from int, fn func(n int, line []byte) bool) error {
	if from < 0 || from >= l.lines {
		return nil
	}
	f, err := os.Open(l.path)
	if err != nil {
		return fmt.Errorf("open session log: %w", err)
	}
	defer func() { _ = f.Close() }()
	cp := from / checkpointEvery
	if _, err := f.Seek(l.offsets[cp], io.SeekStart); err != nil {
		return fmt.Errorf("seek session log: %w", err)
	}
	rd := bufio.NewReaderSize(f, scanBufBytes)
	var line []byte
	for n := cp * checkpointEvery; n < l.lines; n++ {
		var err error
		line, err = readLine(rd, line[:0])
		if err != nil {
			return fmt.Errorf("read session log: %w", err)
		}
		if n >= from && !fn(n, line) {
			return nil
		}
	}
	return nil
}

// readLine appends one line (without its newline, up to maxLineBytes) to dst
// and discards the rest of an overlong line. A final line with no newline is
// returned normally.
func readLine(rd *bufio.Reader, dst []byte) ([]byte, error) {
	for {
		part, err := rd.ReadSlice('\n')
		if room := maxLineBytes - len(dst); room > 0 {
			dst = append(dst, part[:min(len(part), room)]...)
		}
		if errors.Is(err, bufio.ErrBufferFull) {
			continue
		}
		if errors.Is(err, io.EOF) {
			err = nil
		}
		return bytes.TrimSuffix(dst, []byte("\n")), err
	}
}

// view turns raw line bytes into display text: a trailing "\r" from "\r\n" is
// dropped and control sequences are stripped.
func view(raw []byte) string {
	return string(textutil.Strip(bytes.TrimSuffix(raw, []byte("\r"))))
}

// Page returns count stripped lines starting at 0-based line from. A range
// past the end returns the tail, or nothing.
func (l *Log) Page(from, count int) ([]string, error) {
	if from < 0 || count < 0 {
		return nil, fmt.Errorf("invalid page range from=%d count=%d", from, count)
	}
	out := []string{}
	if count == 0 {
		return out, nil
	}
	err := l.scan(from, func(_ int, line []byte) bool {
		out = append(out, view(line))
		return len(out) < count
	})
	return out, err
}

// Cache holds opened logs by id for the open engagement; Clear drops them.
type Cache struct {
	mu   sync.Mutex
	logs map[string]*Log
}

// Open returns the cached log for id, building and caching it on first use.
func (c *Cache) Open(engagementDir, id string) (*Log, error) {
	key := engagementDir + "\x00" + id
	c.mu.Lock()
	l, ok := c.logs[key]
	c.mu.Unlock()
	if ok {
		return l, nil
	}
	l, err := Open(engagementDir, id)
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	if c.logs == nil {
		c.logs = map[string]*Log{}
	}
	c.logs[key] = l
	c.mu.Unlock()
	return l, nil
}

// Clear forgets every cached log.
func (c *Cache) Clear() {
	c.mu.Lock()
	c.logs = nil
	c.mu.Unlock()
}

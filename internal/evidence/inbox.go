package evidence

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/unclesp1d3r/redtracks/internal/model"
	"github.com/unclesp1d3r/redtracks/internal/textutil"
)

const (
	// previewLines and previewLineBytes bound a text preview.
	previewLines     = 20
	previewLineBytes = 200
	// previewReadBytes is the most of a file ListInbox ever reads.
	previewReadBytes = previewLines * previewLineBytes
)

// tempSuffixes mark a file that is still being written by a browser or copier.
var tempSuffixes = []string{".tmp", ".part", ".crdownload"}

// InboxFile is one settled file waiting in the inbox.
type InboxFile struct {
	Name    string
	Size    int64
	ModTime time.Time
	Kind    model.EvidenceKind // empty when the content is not accepted evidence
	Preview string             // first lines of a text file; empty otherwise
}

type fileSig struct {
	size  int64
	mtime time.Time
}

// inboxScanner remembers each file's signature between ticks; a file is ready
// once two consecutive scans agree on it.
type inboxScanner struct {
	dir  string
	seen map[string]fileSig
}

func isTempName(name string) bool {
	return strings.HasPrefix(name, ".") || slices.ContainsFunc(tempSuffixes, func(s string) bool {
		return strings.HasSuffix(name, s)
	})
}

// scan returns the sorted names of ready files. A missing dir is empty.
func (s *inboxScanner) scan() ([]string, error) {
	entries, err := os.ReadDir(s.dir)
	if errors.Is(err, fs.ErrNotExist) {
		entries, err = nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", s.dir, err)
	}
	next := make(map[string]fileSig, len(entries))
	ready := []string{}
	for _, e := range entries {
		if !e.Type().IsRegular() || isTempName(e.Name()) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue // removed between ReadDir and Info
		}
		sig := fileSig{info.Size(), info.ModTime()}
		next[e.Name()] = sig
		if prev, ok := s.seen[e.Name()]; ok && prev == sig {
			ready = append(ready, e.Name())
		}
	}
	s.seen = next
	slices.Sort(ready)
	return ready, nil
}

// WatchInbox polls dir every `every` and calls onChange with the sorted names
// of ready files whenever that set changes. A read error is reported once per
// distinct message. It returns when ctx is done.
func WatchInbox(ctx context.Context, dir string, every time.Duration, onChange func(ready []string),
	onError func(error),
) {
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	sc := &inboxScanner{dir: dir}
	var last []string
	var lastErr string
	check := func() {
		ready, err := sc.scan()
		if err != nil {
			if err.Error() != lastErr {
				lastErr = err.Error()
				onError(err)
			}
			return
		}
		lastErr = ""
		if slices.Equal(ready, last) {
			return
		}
		last = ready
		onChange(slices.Clone(ready))
	}
	check()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			check()
		}
	}
}

// ListInbox describes the ready files in dir. Symlinks and non-regular files are
// skipped and never opened; a file that vanished since the scan is skipped too.
func ListInbox(dir string, ready []string) ([]InboxFile, error) {
	out := make([]InboxFile, 0, len(ready))
	for _, name := range ready {
		f, ok, err := describe(dir, name)
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, f)
		}
	}
	return out, nil
}

func describe(dir, name string) (InboxFile, bool, error) {
	path := filepath.Join(dir, name)
	info, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return InboxFile{}, false, nil
	}
	if err != nil {
		return InboxFile{}, false, fmt.Errorf("stat %s: %w", name, err)
	}
	if !info.Mode().IsRegular() {
		return InboxFile{}, false, nil
	}
	f := InboxFile{Name: name, Size: info.Size(), ModTime: info.ModTime()}
	head, err := readHead(path)
	if err != nil {
		return InboxFile{}, false, err
	}
	if s, err := Sniff(head[:min(len(head), SniffLen)]); err == nil {
		f.Kind = s.Kind
	}
	if f.Kind == model.EvidenceKindText {
		f.Preview = textPreview(head)
	}
	return f, true, nil
}

// readHead reads at most previewReadBytes from the start of a regular file.
func readHead(path string) ([]byte, error) {
	fh, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer func() { _ = fh.Close() }()
	head, err := io.ReadAll(io.LimitReader(fh, previewReadBytes))
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return head, nil
}

// textPreview strips head and keeps the first previewLines lines, each cut to
// previewLineBytes on a rune boundary.
func textPreview(head []byte) string {
	lines := bytes.SplitN(textutil.Strip(head), []byte("\n"), previewLines+1)
	lines = lines[:min(len(lines), previewLines)]
	for i, l := range lines {
		if len(l) > previewLineBytes {
			n := previewLineBytes
			for n > 0 && l[n]&0xC0 == 0x80 { // back off a continuation byte
				n--
			}
			lines[i] = l[:n]
		}
	}
	return string(bytes.Join(lines, []byte("\n")))
}

package evidence

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/unclesp1d3r/redtracks/internal/model"
)

// NotFoundError says the requested file or its root does not exist.
type NotFoundError struct{ Name string }

func (e *NotFoundError) Error() string { return e.Name + " not found" }

// RefusedError says the request was understood but must not be served.
type RefusedError struct{ Reason string }

func (e *RefusedError) Error() string { return e.Reason }

// Resolved describes a file Resolve accepted. ContentType comes from the
// sniffed kind only, never from the file name.
type Resolved struct {
	Path        string
	Kind        model.EvidenceKind
	ContentType string
	Size        int64
	ModTime     time.Time
}

const textContentType = "text/plain; charset=utf-8"

var imageContentTypes = map[string]string{
	"png": "image/png", "jpg": "image/jpeg", "gif": "image/gif", "webp": "image/webp",
}

func refuse(format string, args ...any) error {
	return &RefusedError{Reason: fmt.Sprintf(format, args...)}
}

// checkSegments refuses anything that is not a plain single path element.
func checkSegments(segments []string) error {
	if len(segments) == 0 {
		return refuse("no path given")
	}
	for _, s := range segments {
		switch {
		case s == "" || s == "." || s == "..":
			return refuse("invalid path segment %q", s)
		case strings.ContainsAny(s, "/\\\x00"), filepath.IsAbs(s), filepath.VolumeName(s) != "":
			return refuse("invalid path segment %q", s)
		}
	}
	return nil
}

// confine returns the path of segments under root after checking that its
// symlink-resolved location stays inside root's.
func confine(root string, segments []string) (string, error) {
	full := filepath.Join(append([]string{root}, segments...)...)
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", notFoundOr(err, filepath.Base(root))
	}
	realFull, err := filepath.EvalSymlinks(full)
	if err != nil {
		return "", notFoundOr(err, segments[len(segments)-1])
	}
	rel, err := filepath.Rel(realRoot, realFull)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", refuse("path escapes the evidence root")
	}
	return full, nil
}

func notFoundOr(err error, name string) error {
	if errors.Is(err, os.ErrNotExist) {
		return &NotFoundError{Name: name}
	}
	return fmt.Errorf("resolve %s: %w", name, err)
}

// OpenResolved is Resolve that also hands back the open file, so the caller
// reads the very file that was checked. The caller closes it.
func OpenResolved(root string, segments ...string) (*os.File, Resolved, error) {
	if err := checkSegments(segments); err != nil {
		return nil, Resolved{}, err
	}
	full, err := confine(root, segments)
	if err != nil {
		return nil, Resolved{}, err
	}
	f, err := os.OpenFile(full, os.O_RDONLY|openFlags, 0)
	if err != nil {
		// A symlink at the final component fails the nofollow open (ELOOP/EMLINK).
		return nil, Resolved{}, refuse("%s cannot be opened as evidence", segments[len(segments)-1])
	}
	res, err := classify(f, full)
	if err != nil {
		_ = f.Close()
		return nil, Resolved{}, err
	}
	return f, res, nil
}

func classify(f *os.File, path string) (Resolved, error) {
	info, err := f.Stat()
	if err != nil {
		return Resolved{}, fmt.Errorf("stat %s: %w", filepath.Base(path), err)
	}
	if !info.Mode().IsRegular() {
		return Resolved{}, refuse("%s is not a regular file", filepath.Base(path))
	}
	head := make([]byte, SniffLen)
	n, err := io.ReadFull(f, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return Resolved{}, fmt.Errorf("read %s: %w", filepath.Base(path), err)
	}
	sniffed, err := Sniff(head[:n])
	if err != nil {
		return Resolved{}, &RefusedError{Reason: err.Error()}
	}
	ct := textContentType
	if sniffed.Kind == model.EvidenceKindImage {
		ct = imageContentTypes[sniffed.Ext]
	}
	return Resolved{Path: path, Kind: sniffed.Kind, ContentType: ct, Size: info.Size(), ModTime: info.ModTime()}, nil
}

// Resolve confines segments to root and reports how the file may be served.
// It refuses empty, dot, parent, absolute, separator-bearing and NUL-bearing
// segments, symlinks, non-regular files and types Sniff does not accept.
func Resolve(root string, segments ...string) (Resolved, error) {
	f, res, err := OpenResolved(root, segments...)
	if err != nil {
		return Resolved{}, err
	}
	_ = f.Close()
	return res, nil
}

// OpenRegular opens the absolute path path for reading without following a
// final symlink and refuses anything that is not a regular file, deciding on
// the open handle so there is no gap before the read. The caller closes it.
func OpenRegular(path string) (*os.File, os.FileInfo, error) {
	if !filepath.IsAbs(path) {
		return nil, nil, refuse("%q is not an absolute path", path)
	}
	f, err := os.OpenFile(path, os.O_RDONLY|openFlags, 0)
	if errors.Is(err, syscall.ELOOP) {
		return nil, nil, refuse("%s is a symbolic link, not a regular file", path)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("open %s: %w", path, err)
	}
	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, nil, fmt.Errorf("stat %s: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		_ = f.Close()
		return nil, nil, refuse("%s is a %s, not a regular file", path, describeMode(info.Mode()))
	}
	return f, info, nil
}

// describeMode names the kind of a non-regular file for refusal messages.
func describeMode(m os.FileMode) string {
	switch {
	case m.IsDir():
		return "directory"
	case m&os.ModeNamedPipe != 0:
		return "named pipe"
	case m&os.ModeDevice != 0:
		return "device"
	case m&os.ModeSocket != 0:
		return "socket"
	}
	return "special file"
}

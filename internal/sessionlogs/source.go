// Package sessionlogs imports terminal session logs into an engagement and
// serves them in pages, searches, and clips without ever holding a whole log
// in memory. It is named apart from the rotating application log in
// internal/app.
package sessionlogs

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/unclesp1d3r/redtracks/internal/evidence"
	"github.com/unclesp1d3r/redtracks/internal/store"
)

const (
	logExt       = "log"
	stampLayout  = "20060102T150405Z"
	maxCopyRead  = 64 << 10 // largest single read while copying a source
	logsDirMode  = 0o700
	logFileSufix = "." + logExt
)

// idPattern matches the stem Import generates: <UTC stamp>-<n>.
var idPattern = regexp.MustCompile(`^\d{8}T\d{6}Z-\d+$`)

// Source describes one imported session log.
type Source struct {
	ID           string // file stem under logs/, stable across restarts
	Name         string // file name under logs/
	OriginalName string // base name of the file that was imported; equals Name after a restart
	Size         int64
	ModTime      time.Time // modification time of the original file
	ImportedAt   time.Time
	Lines        int
}

// Import copies the regular file at srcPath into the engagement's logs
// directory and indexes it. The file is opened once and checked with fstat, so
// a symlink, FIFO, directory, or device is refused. A failed copy leaves no
// file behind.
func Import(engagementDir, srcPath string) (Source, error) {
	f, info, err := evidence.OpenRegular(srcPath)
	if err != nil {
		return Source{}, err
	}
	defer func() { _ = f.Close() }()
	return importReader(engagementDir, f, filepath.Base(srcPath), info.ModTime())
}

// importReader streams r into a freshly named file under logs/ and indexes it.
func importReader(engagementDir string, r io.Reader, original string, modTime time.Time) (Source, error) {
	dir := store.LogsDir(engagementDir)
	if err := os.MkdirAll(dir, logsDirMode); err != nil {
		return Source{}, fmt.Errorf("create %s: %w", dir, err)
	}
	now := time.Now().UTC()
	out, name, err := store.CreateExclusive(dir, now.Format(stampLayout), logExt)
	if err != nil {
		return Source{}, err
	}
	path := filepath.Join(dir, name)
	copied, err := copyBounded(out, r)
	if closeErr := out.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Chtimes(path, modTime, modTime)
	}
	if err != nil {
		return Source{}, errors.Join(fmt.Errorf("copy session log: %w", err), removeQuietly(path))
	}
	l, err := buildLog(path)
	if err != nil {
		return Source{}, errors.Join(err, removeQuietly(path))
	}
	return Source{
		ID: strings.TrimSuffix(name, logFileSufix), Name: name, OriginalName: original,
		Size: copied, ModTime: modTime, ImportedAt: now, Lines: l.Lines(),
	}, nil
}

// copyBounded copies r to w through a fixed buffer, hiding ReadFrom/WriteTo so
// no single read is larger than maxCopyRead.
func copyBounded(w io.Writer, r io.Reader) (int64, error) {
	return io.CopyBuffer(struct{ io.Writer }{w}, struct{ io.Reader }{r}, make([]byte, maxCopyRead))
}

func removeQuietly(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("remove %s: %w", path, err)
	}
	return nil
}

// List rebuilds every source from the logs directory, oldest first.
func List(engagementDir string) ([]Source, error) {
	entries, err := os.ReadDir(store.LogsDir(engagementDir))
	if errors.Is(err, fs.ErrNotExist) {
		return []Source{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("list session logs: %w", err)
	}
	out := []Source{}
	for _, e := range entries {
		id := strings.TrimSuffix(e.Name(), logFileSufix)
		if e.IsDir() || !strings.HasSuffix(e.Name(), logFileSufix) || !idPattern.MatchString(id) {
			continue
		}
		l, err := Open(engagementDir, id)
		if err != nil {
			return nil, err
		}
		out = append(out, l.Source())
	}
	slices.SortFunc(out, func(a, b Source) int { return strings.Compare(a.ID, b.ID) })
	return out, nil
}

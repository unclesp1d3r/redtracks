// Package app holds the Wails services that expose the engagement store,
// checklists and importers to the frontend, plus the rotating log.
package app

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
)

const (
	// maxLogBytes is the size above which the log is rotated before the next write.
	maxLogBytes = 5 << 20
	logFileMode = 0o600
)

// rotatingWriter appends to a file and renames it to path+".1" once it has
// grown past maxLogBytes.
type rotatingWriter struct {
	mu   sync.Mutex
	path string
	file *os.File
	size int64
}

// OpenLog returns a text slog logger appending to path. Before a write that
// finds the file larger than 5 MB, the file is renamed to path+".1" and a new
// one is started. The closer closes the underlying file.
func OpenLog(path string) (*slog.Logger, io.Closer, error) {
	w := &rotatingWriter{path: path}
	if err := w.open(); err != nil {
		return nil, nil, err
	}
	return slog.New(slog.NewTextHandler(w, nil)), w, nil
}

func (w *rotatingWriter) open() error {
	f, err := os.OpenFile(w.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, logFileMode)
	if err != nil {
		return fmt.Errorf("open log %s: %w", w.path, err)
	}
	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return fmt.Errorf("stat log %s: %w", w.path, err)
	}
	w.file, w.size = f, info.Size()
	return nil
}

func (w *rotatingWriter) rotate() error {
	if err := w.file.Close(); err != nil {
		return fmt.Errorf("close log %s: %w", w.path, err)
	}
	if err := os.Rename(w.path, w.path+".1"); err != nil {
		return fmt.Errorf("rotate log %s: %w", w.path, err)
	}
	return w.open()
}

// Write implements io.Writer.
func (w *rotatingWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.size > maxLogBytes {
		if err := w.rotate(); err != nil {
			return 0, err
		}
	}
	n, err := w.file.Write(p)
	w.size += int64(n)
	if err != nil {
		return n, fmt.Errorf("write log %s: %w", w.path, err)
	}
	return n, nil
}

// Close implements io.Closer.
func (w *rotatingWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.file.Close()
}

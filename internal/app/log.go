package app

import (
	"errors"
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
// finds the file larger than maxLogBytes, the file is renamed to path+".1" and a new
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

// rotate moves the full log aside and starts a new one. It always reopens the
// log, even when the rename fails, so logging keeps working; the returned
// error joins every step that failed.
func (w *rotatingWriter) rotate() error {
	var errs []error
	if err := w.file.Close(); err != nil {
		errs = append(errs, fmt.Errorf("close log %s: %w", w.path, err))
	}
	if err := os.Rename(w.path, w.path+".1"); err != nil {
		errs = append(errs, fmt.Errorf("rotate log %s: %w", w.path, err))
	}
	if err := w.open(); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

// Write implements io.Writer. slog discards Write errors, so a failure is
// also noted on stderr.
func (w *rotatingWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.size > maxLogBytes {
		if err := w.rotate(); err != nil {
			noteLogFailure(err)
		}
	}
	n, err := w.file.Write(p)
	w.size += int64(n)
	if err != nil {
		err = fmt.Errorf("write log %s: %w", w.path, err)
		noteLogFailure(err)
		return n, err
	}
	return n, nil
}

func noteLogFailure(err error) {
	_, _ = fmt.Fprintf(os.Stderr, "redtracks: log failure: %v\n", err)
}

// Close implements io.Closer.
func (w *rotatingWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.file.Close()
}

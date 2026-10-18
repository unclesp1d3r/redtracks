package store

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// DirSignature returns a string that changes whenever the name, size, or mtime
// of a file directly inside dir changes, or a file is added or removed. Dotfiles
// and subdirectories are ignored, and keep (when non-nil) filters the rest by
// name. A missing dir has the empty signature.
func DirSignature(dir string, keep func(name string) bool) (string, error) {
	entries, err := os.ReadDir(dir) // sorted by name
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read %s: %w", dir, err)
	}
	var sb strings.Builder
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || strings.HasPrefix(name, ".") || (keep != nil && !keep(name)) {
			continue
		}
		info, err := e.Info()
		if errors.Is(err, os.ErrNotExist) {
			continue // removed between ReadDir and Info
		}
		if err != nil {
			return "", fmt.Errorf("stat %s: %w", name, err)
		}
		fmt.Fprintf(&sb, "%s:%d:%d;", name, info.Size(), info.ModTime().UnixNano())
	}
	return sb.String(), nil
}

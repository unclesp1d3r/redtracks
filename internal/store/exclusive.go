package store

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// maxExclusiveTry bounds the numbering search in CreateExclusive.
const maxExclusiveTry = 10000

// CreateExclusive creates dir/<stem>-<n>.<ext> with the smallest n >= 1 that
// does not exist yet, using O_EXCL so an existing file is never truncated and
// concurrent callers get distinct names. It returns the open file and its base
// name. dir is created if missing.
func CreateExclusive(dir, stem, ext string) (*os.File, string, error) {
	if err := os.MkdirAll(dir, dirMode); err != nil {
		return nil, "", fmt.Errorf("create %s: %w", dir, err)
	}
	for n := 1; n <= maxExclusiveTry; n++ {
		name := fmt.Sprintf("%s-%d.%s", stem, n, ext)
		f, err := os.OpenFile(filepath.Join(dir, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, fileMode)
		if err == nil {
			return f, name, nil
		}
		if !errors.Is(err, fs.ErrExist) {
			return nil, "", fmt.Errorf("create %s in %s: %w", name, dir, err)
		}
	}
	return nil, "", fmt.Errorf("no free name for %s.%s in %s", stem, ext, dir)
}

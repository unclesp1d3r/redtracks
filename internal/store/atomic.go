// Package store persists an engagement as a folder of plain files.
package store

import (
	"fmt"
	"os"
	"path/filepath"
)

// fileMode is the permission set applied to every file the store writes.
const fileMode os.FileMode = 0o644

// WriteAtomic replaces path with data. It writes a temp file in the same
// directory, fsyncs, then renames over path, so readers see the old or the new
// content and never a partial file. The temp file is removed on any failure.
func WriteAtomic(path string, data []byte) (err error) {
	dir, base := filepath.Split(path)
	if dir == "" {
		dir = "."
	}
	tmp, err := os.CreateTemp(dir, "."+base+".*.tmp")
	if err != nil {
		return fmt.Errorf("create temp for %s: %w", path, err)
	}
	tmpName := tmp.Name()
	defer func() {
		if err != nil {
			_ = os.Remove(tmpName)
		}
	}()

	if _, err = tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write %s: %w", tmpName, err)
	}
	if err = tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync %s: %w", tmpName, err)
	}
	if err = tmp.Close(); err != nil {
		return fmt.Errorf("close %s: %w", tmpName, err)
	}
	if err = os.Chmod(tmpName, fileMode); err != nil {
		return fmt.Errorf("chmod %s: %w", tmpName, err)
	}
	if err = os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("rename to %s: %w", path, err)
	}
	return nil
}

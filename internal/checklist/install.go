package checklist

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/unclesp1d3r/redtracks/internal/store"
)

// installDirMode is the permission set applied to a freshly created
// checklist directory.
const installDirMode os.FileMode = 0o700

// tmpSuffix names the staging directory next to dst; it is renamed into place
// only once complete, so a crash never leaves a half-populated dst.
const tmpSuffix = ".tmp"

// InstallDefaults copies every file in src into dst, creating dst if it does
// not already exist. It is a no-op, returning false, when dst already exists:
// an operator's edits are never overwritten.
func InstallDefaults(src fs.FS, dst string) (bool, error) {
	if _, statErr := os.Stat(dst); statErr == nil {
		return false, nil
	} else if !os.IsNotExist(statErr) {
		return false, fmt.Errorf("stat %s: %w", dst, statErr)
	}

	tmp := dst + tmpSuffix
	if err := os.RemoveAll(tmp); err != nil {
		return false, fmt.Errorf("remove stale %s: %w", tmp, err)
	}
	if err := writeDefaults(src, tmp); err != nil {
		_ = os.RemoveAll(tmp)
		return false, err
	}
	if err := os.Rename(tmp, dst); err != nil {
		_ = os.RemoveAll(tmp)
		return false, fmt.Errorf("install %s: %w", dst, err)
	}
	return true, nil
}

// writeDefaults copies every file in src into the new directory dir.
func writeDefaults(src fs.FS, dir string) error {
	if err := os.MkdirAll(dir, installDirMode); err != nil {
		return fmt.Errorf("create %s: %w", dir, err)
	}
	entries, err := fs.ReadDir(src, ".")
	if err != nil {
		return fmt.Errorf("read embedded checklists: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := fs.ReadFile(src, entry.Name())
		if err != nil {
			return fmt.Errorf("read embedded %s: %w", entry.Name(), err)
		}
		if err := store.WriteAtomic(filepath.Join(dir, entry.Name()), data); err != nil {
			return fmt.Errorf("write %s: %w", entry.Name(), err)
		}
	}
	return nil
}

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

// InstallDefaults copies every file in src into dst, creating dst if it does
// not already exist. It is a no-op, returning installed=false, when dst
// already exists: an operator's edits are never overwritten.
//
//nolint:nonamedreturns // signature fixed by the Task 6 plan: InstallDefaults(src fs.FS, dst string) (installed bool, err error)
func InstallDefaults(src fs.FS, dst string) (installed bool, err error) {
	if _, statErr := os.Stat(dst); statErr == nil {
		return false, nil
	} else if !os.IsNotExist(statErr) {
		return false, fmt.Errorf("stat %s: %w", dst, statErr)
	}

	if err := os.MkdirAll(dst, installDirMode); err != nil {
		return false, fmt.Errorf("create %s: %w", dst, err)
	}

	entries, err := fs.ReadDir(src, ".")
	if err != nil {
		return false, fmt.Errorf("read embedded checklists: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := fs.ReadFile(src, entry.Name())
		if err != nil {
			return false, fmt.Errorf("read embedded %s: %w", entry.Name(), err)
		}
		if err := store.WriteAtomic(filepath.Join(dst, entry.Name()), data); err != nil {
			return false, fmt.Errorf("write %s: %w", entry.Name(), err)
		}
	}
	return true, nil
}

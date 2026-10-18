package checklist

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/unclesp1d3r/redtracks/internal/store"
)

// installDirMode is the permission set applied to a freshly created
// checklist directory.
const installDirMode os.FileMode = 0o700

// manifestName is the dotfile in the checklists folder listing every bundled
// file name ever installed, one per line. A name listed there is never added
// again, so an operator's deletion sticks.
const manifestName = ".installed"

// preManifestBundled is the bundled set that shipped before the manifest existed.
// A checklists folder with no manifest is assumed to have received exactly
// these, so only files added after the manifest was introduced are added to it.
var preManifestBundled = []string{
	"_readme.md", "http.json", "kerberos.json", "ldap.json", "linux-privesc.json",
	"llm-chat.json", "mcp-server.json", "rag-api.json", "smb.json", "ssh.json", "windows-privesc.json",
}

// InstallDefaults adds every file in src that is absent from both dst and the
// dst manifest, creating dst if needed, and returns the names added. A present
// file is never touched, and a name the manifest lists is never re-added.
func InstallDefaults(src fs.FS, dst string) ([]string, error) {
	listed, err := readManifest(dst)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dst, installDirMode); err != nil {
		return nil, fmt.Errorf("create %s: %w", dst, err)
	}
	entries, err := fs.ReadDir(src, ".")
	if err != nil {
		return nil, fmt.Errorf("read embedded checklists: %w", err)
	}
	added := []string{}
	known := slices.Clone(listed)
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || slices.Contains(listed, name) {
			continue
		}
		known = append(known, name)
		wrote, err := copyIfAbsent(src, dst, name)
		if err != nil {
			return added, err
		}
		if wrote {
			added = append(added, name)
		}
	}
	if len(known) == len(listed) {
		return added, nil
	}
	if err := store.WriteAtomic(filepath.Join(dst, manifestName), []byte(strings.Join(known, "\n")+"\n")); err != nil {
		return added, fmt.Errorf("write manifest: %w", err)
	}
	return added, nil
}

// copyIfAbsent writes the bundled file name into dst unless it is already there.
func copyIfAbsent(src fs.FS, dst, name string) (bool, error) {
	path := filepath.Join(dst, name)
	if _, err := os.Stat(path); err == nil {
		return false, nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return false, fmt.Errorf("stat %s: %w", path, err)
	}
	data, err := fs.ReadFile(src, name)
	if err != nil {
		return false, fmt.Errorf("read embedded %s: %w", name, err)
	}
	if err := store.WriteAtomic(path, data); err != nil {
		return false, fmt.Errorf("write %s: %w", name, err)
	}
	return true, nil
}

// readManifest returns the names listed in dst's manifest. With no manifest it
// returns the slice-one set when dst already exists, and nothing when it does not.
func readManifest(dst string) ([]string, error) {
	data, err := os.ReadFile(filepath.Join(dst, manifestName))
	if err == nil {
		return strings.Fields(string(data)), nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("read manifest: %w", err)
	}
	if _, statErr := os.Stat(dst); statErr == nil {
		return slices.Clone(preManifestBundled), nil
	} else if !errors.Is(statErr, fs.ErrNotExist) {
		return nil, fmt.Errorf("stat %s: %w", dst, statErr)
	}
	return nil, nil
}

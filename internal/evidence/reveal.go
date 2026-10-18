package evidence

import (
	"fmt"
	"path/filepath"
	"strings"
)

// RevealArgs returns the argument list that shows path in the platform file
// manager. It is a fixed list, never a shell string.
func RevealArgs(goos, path string) ([]string, error) {
	switch goos {
	case "darwin":
		return []string{"open", "-R", path}, nil
	case "windows":
		return []string{"explorer", "/select," + path}, nil
	case "linux":
		return []string{"xdg-open", filepath.Dir(path)}, nil
	}
	return nil, fmt.Errorf("no file manager known for %s", goos)
}

// UnderRoot reports whether path is root or lies inside it, without touching the disk.
func UnderRoot(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

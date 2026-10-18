//go:build unix

package evidence

import (
	"errors"
	"path/filepath"
	"syscall"
	"testing"
)

func TestResolveRefusesFIFOWithoutBlocking(t *testing.T) {
	root := t.TempDir()
	if err := syscall.Mkfifo(filepath.Join(root, "pipe.png"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Resolve(root, "pipe.png")
	if _, ok := errors.AsType[*RefusedError](err); !ok {
		t.Fatalf("got %v, want RefusedError", err)
	}
}

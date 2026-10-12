package store

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

const (
	pollEvery = 20 * time.Millisecond
	pollWait  = 500 * time.Millisecond
)

// bumpMtime pushes a file's mtime forward so Reload sees a change even on
// coarse-resolution filesystems.
func bumpMtime(t *testing.T, path string) {
	t.Helper()
	future := time.Now().Add(time.Second)
	if err := os.Chtimes(path, future, future); err != nil {
		t.Fatal(err)
	}
}

func TestPollCallsOnChange(t *testing.T) {
	dir := copyFixture(t)
	s := mustOpen(t, dir)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	changed := make(chan []string, 4)
	done := make(chan struct{})
	go func() {
		s.Poll(ctx, pollEvery, func(c []string) { changed <- c }, func(error) {})
		close(done)
	}()

	path := filepath.Join(dir, hostRel)
	mustWrite(t, path, readFile(t, path)+"\n")
	bumpMtime(t, path)

	select {
	case got := <-changed:
		if len(got) != 1 || got[0] != hostRel {
			t.Fatalf("changed = %v", got)
		}
	case <-time.After(pollWait):
		t.Fatal("onChange not called")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(pollWait):
		t.Fatal("Poll did not stop after cancel")
	}
}

func TestPollReportsParseErrors(t *testing.T) {
	dir := copyFixture(t)
	s := mustOpen(t, dir)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var mu sync.Mutex
	var onChangeCalls int
	errCh := make(chan error, 4)
	go s.Poll(ctx, pollEvery,
		func([]string) { mu.Lock(); onChangeCalls++; mu.Unlock() },
		func(err error) { errCh <- err })

	path := filepath.Join(dir, hostRel)
	mustWrite(t, path, "{")
	bumpMtime(t, path)

	select {
	case err := <-errCh:
		var pe *ParseError
		if !errors.As(err, &pe) {
			t.Fatalf("err = %v, want *ParseError", err)
		}
	case <-time.After(pollWait):
		t.Fatal("onError not called")
	}
	mu.Lock()
	defer mu.Unlock()
	if onChangeCalls != 0 {
		t.Fatalf("onChange called %d times", onChangeCalls)
	}
}

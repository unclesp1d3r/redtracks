package app

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/unclesp1d3r/redtracks/internal/model"
)

const (
	fixtureDir  = "../store/testdata/pg-sample"
	sampleHost  = "10.10.10.5"
	testPollGap = 20 * time.Millisecond
)

// events records emitted events for assertions.
type events struct {
	mu       sync.Mutex
	list     []string
	payloads []any
}

func (e *events) emit(name string, data any) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.list = append(e.list, name)
	e.payloads = append(e.payloads, data)
}

// errorEvents returns the ErrorEvent payloads emitted so far.
func (e *events) errorEvents() []ErrorEvent {
	e.mu.Lock()
	defer e.mu.Unlock()
	var out []ErrorEvent
	for i, name := range e.list {
		if ev, ok := e.payloads[i].(ErrorEvent); ok && name == eventError {
			out = append(out, ev)
		}
	}
	return out
}

// changedPaths returns every path listed in engagement:changed payloads so far.
func (e *events) changedPaths() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	var out []string
	for i, name := range e.list {
		if paths, ok := e.payloads[i].([]string); ok && name == eventChanged {
			out = append(out, paths...)
		}
	}
	return out
}

func (e *events) count(name string) int {
	e.mu.Lock()
	defer e.mu.Unlock()
	n := 0
	for _, v := range e.list {
		if v == name {
			n++
		}
	}
	return n
}

func copyTree(t *testing.T, src, dst string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(dst), dirMode); err != nil {
		t.Fatal(err)
	}
	if err := os.CopyFS(dst, os.DirFS(src)); err != nil {
		t.Fatal(err)
	}
}

// current returns the open engagement or fails the test.
func current(t *testing.T, a *App) model.Engagement {
	t.Helper()
	eng, err := NewEngagementService(a).Current()
	if err != nil {
		t.Fatal(err)
	}
	return eng
}

// fixture is an App over a temp root plus the path of its copied engagement.
type fixture struct {
	app *App
	dir string
}

// newTestApp builds an App over a temp root holding a copy of pg-sample.
func newTestApp(t *testing.T) fixture {
	t.Helper()
	pollInterval, checklistInterval = testPollGap, testPollGap
	root := t.TempDir()
	dir := filepath.Join(root, engagementsDir, "pg-sample")
	copyTree(t, fixtureDir, dir)
	a, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = a.Close() })
	return fixture{app: a, dir: dir}
}

func openSample(t *testing.T) fixture {
	t.Helper()
	f := newTestApp(t)
	if _, err := NewEngagementService(f.app).Open(f.dir); err != nil {
		t.Fatal(err)
	}
	return f
}

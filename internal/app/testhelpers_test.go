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
	mu   sync.Mutex
	list []string
}

func (e *events) emit(name string, _ any) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.list = append(e.list, name)
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

// newTestApp builds an App over a temp root holding a copy of pg-sample.
// fixture is an App over a temp root plus the path of its copied engagement.
type fixture struct {
	app *App
	dir string
}

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

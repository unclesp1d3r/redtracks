package app

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/unclesp1d3r/redtracks/internal/model"
)

func writeInbox(t *testing.T, a *App, name, content string) string {
	t.Helper()
	p := filepath.Join(a.inboxDir(), name)
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// listedNames waits until the inbox lists exactly want and returns that listing.
func listedNames(t *testing.T, s *EvidenceService, want ...string) InboxListing {
	t.Helper()
	var got InboxListing
	waitFor(t, "inbox listing "+strings.Join(want, ","), func() bool {
		l, err := s.Inbox()
		if err != nil {
			t.Fatal(err)
		}
		names := make([]string, len(l.Files))
		for i, f := range l.Files {
			names[i] = f.Name
		}
		slices.Sort(names)
		got = l
		return slices.Equal(names, want)
	})
	return got
}

func hostEvidenceFiles(t *testing.T, dir string) []string {
	t.Helper()
	return globFiles(t, filepath.Join(dir, "hosts", sampleHost, "evidence", "*"))
}

func TestInboxAssignMovesFileUnderHostAndSurvivesRestart(t *testing.T) {
	f := openSample(t)
	svc := NewEvidenceService(f.app)
	writeInbox(t, f.app, "a.txt", "alpha")
	writeInbox(t, f.app, "b.txt", "bravo")
	writeInbox(t, f.app, "c.txt", "charlie")
	l := listedNames(t, svc, "a.txt", "b.txt", "c.txt")

	item, err := svc.Assign(l.Token, "a.txt", sampleHost, "the proof", nil)
	if err != nil {
		t.Fatal(err)
	}

	if item.Kind != model.EvidenceKindText || item.Caption != "the proof" {
		t.Fatalf("item = %+v", item)
	}
	if got := hostEvidenceFiles(t, f.dir); len(got) != 1 {
		t.Fatalf("host evidence files = %v, want 1", got)
	}
	host, _ := current(t, f.app).Host(sampleHost)
	if host.ID == "" || len(current(t, f.app).Evidence[sampleHost].Items) != 1 {
		t.Fatalf("evidence index = %+v", current(t, f.app).Evidence[sampleHost])
	}
	root := f.app.Root
	if err := f.app.Close(); err != nil {
		t.Fatal(err)
	}
	a2, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = a2.Close() })
	if _, err := NewEngagementService(a2).Open(f.dir); err != nil {
		t.Fatal(err)
	}
	listedNames(t, NewEvidenceService(a2), "b.txt", "c.txt")
}

func TestInboxTokenRefusedAfterEngagementSwitch(t *testing.T) {
	f := openSample(t)
	svc := NewEvidenceService(f.app)
	writeInbox(t, f.app, "a.txt", "alpha")
	l := listedNames(t, svc, "a.txt")
	other := filepath.Join(f.app.Root, engagementsDir, "other")
	copyTree(t, fixtureDir, other)
	if _, err := NewEngagementService(f.app).Open(other); err != nil {
		t.Fatal(err)
	}

	_, err := svc.Assign(l.Token, "a.txt", sampleHost, "", nil)

	if !errors.Is(err, errStaleInbox) {
		t.Fatalf("err = %v, want errStaleInbox", err)
	}
	if _, serr := os.Stat(filepath.Join(f.app.inboxDir(), "a.txt")); serr != nil {
		t.Fatalf("inbox file must remain: %v", serr)
	}
}

func TestInboxNamesWithSeparatorsAreRefusedBeforeFileAccess(t *testing.T) {
	f := openSample(t)
	svc := NewEvidenceService(f.app)
	outside := filepath.Join(f.app.Root, "outside.txt")
	if err := os.WriteFile(outside, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	writeInbox(t, f.app, "a.txt", "alpha")
	l := listedNames(t, svc, "a.txt")

	for _, name := range []string{"../outside.txt", "sub/a.txt", `sub\a.txt`, "..", "", "a..b/../x"} {
		if _, err := svc.Assign(l.Token, name, sampleHost, "", nil); err == nil {
			t.Errorf("Assign(%q) succeeded, want refusal", name)
		}
		if err := svc.DiscardInboxFile(name); err == nil {
			t.Errorf("DiscardInboxFile(%q) succeeded, want refusal", name)
		}
	}

	if _, err := os.Stat(outside); err != nil {
		t.Fatalf("outside file was touched: %v", err)
	}
	if got := hostEvidenceFiles(t, f.dir); len(got) != 0 {
		t.Fatalf("evidence files created: %v", got)
	}
}

func TestInboxRemovalFailureAfterImportIsNotAnError(t *testing.T) {
	f := openSample(t)
	svc := NewEvidenceService(f.app)
	writeInbox(t, f.app, "a.txt", "alpha")
	l := listedNames(t, svc, "a.txt")
	old := removeInboxFile
	removeInboxFile = func(string) error { return errors.New("disk says no") }
	t.Cleanup(func() { removeInboxFile = old })

	item, err := svc.Assign(l.Token, "a.txt", sampleHost, "", nil)

	if err != nil || item.ID == "" {
		t.Fatalf("Assign = %+v, %v; want success", item, err)
	}
	if len(hostEvidenceFiles(t, f.dir)) != 1 {
		t.Fatal("item should exist")
	}
	listedNames(t, svc, "a.txt")
}

func TestDiscardInboxFileRemovesIt(t *testing.T) {
	f := openSample(t)
	svc := NewEvidenceService(f.app)
	writeInbox(t, f.app, "a.txt", "alpha")
	listedNames(t, svc, "a.txt")

	if err := svc.DiscardInboxFile("a.txt"); err != nil {
		t.Fatal(err)
	}

	listedNames(t, svc)
}

func TestInboxSentinelEmittedOnAppearAndDisappearOnly(t *testing.T) {
	f := newTestApp(t)
	rec := &events{}
	f.app.SetEmitter(rec.emit)
	count := func() int {
		n := 0
		for _, p := range rec.changedPaths() {
			if p == inboxSource {
				n++
			}
		}
		return n
	}
	writeInbox(t, f.app, "a.txt", "alpha")
	waitFor(t, "sentinel on appear", func() bool { return count() == 1 })

	for range 10 {
		time.Sleep(testPollGap)
	}
	if got := count(); got != 1 {
		t.Fatalf("sentinel count after idle ticks = %d, want 1", got)
	}
	if err := os.Remove(filepath.Join(f.app.inboxDir(), "a.txt")); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "sentinel on disappear", func() bool { return count() == 2 })
}

func TestInboxWorksWithoutEngagementAndListsOnceOpened(t *testing.T) {
	f := newTestApp(t)
	svc := NewEvidenceService(f.app)
	writeInbox(t, f.app, "a.txt", "alpha")
	waitFor(t, "ready file tracked", func() bool {
		f.app.mu.Lock()
		defer f.app.mu.Unlock()
		return len(f.app.inboxReady) == 1
	})

	l, err := svc.Inbox()
	if err != nil || l.Token != "" || len(l.Files) != 0 {
		t.Fatalf("no engagement: %+v, %v; want empty listing", l, err)
	}
	if _, err := NewEngagementService(f.app).Open(f.dir); err != nil {
		t.Fatal(err)
	}

	got := listedNames(t, svc, "a.txt")
	if got.Token == "" {
		t.Fatal("want a token once an engagement is open")
	}
}

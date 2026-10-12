package store

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/unclesp1d3r/redtracks/internal/model"
)

const credsRel = "creds.json"

func skipIfPermissionsIgnored(t *testing.T, path string) {
	t.Helper()
	if _, err := os.ReadDir(path); err == nil {
		t.Skip("permissions are not enforced (running as root?)")
	}
}

func TestReloadKeepsLastGoodWhenFileDeleted(t *testing.T) {
	dir := copyFixture(t)
	s := mustOpen(t, dir)
	if err := os.Remove(filepath.Join(dir, credsRel)); err != nil {
		t.Fatal(err)
	}
	_, err := s.Reload()
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Reload err = %v, want fs.ErrNotExist", err)
	}
	if len(s.Engagement().Creds.Credentials) == 0 {
		t.Fatal("credential dropped after the file was deleted")
	}
	if _, err := s.Reload(); err != nil {
		t.Fatalf("repeat Reload err = %v, want deduped nil", err)
	}
}

func TestReloadDedupesReadErrors(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores file permissions")
	}
	dir := copyFixture(t)
	s := mustOpen(t, dir)
	path := filepath.Join(dir, credsRel)
	if err := os.Chmod(path, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(path, fileMode); err != nil {
			t.Error(err)
		}
	})
	shiftMtime(t, path, time.Minute)
	if _, err := s.Reload(); err == nil {
		t.Fatal("first Reload should report the read error")
	}
	if _, err := s.Reload(); err != nil {
		t.Fatalf("second Reload err = %v, want deduped nil", err)
	}
}

func TestReloadReportsUnreadableHostDir(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	dir := copyFixture(t)
	s := mustOpen(t, dir)
	hostDirPath := filepath.Join(dir, "hosts", fixtureIP)
	if err := os.Chmod(hostDirPath, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(hostDirPath, dirMode); err != nil {
			t.Error(err)
		}
	})
	skipIfPermissionsIgnored(t, hostDirPath)
	if _, err := s.Reload(); err == nil || !strings.Contains(err.Error(), fixtureIP) {
		t.Fatalf("Reload err = %v, want a stat error naming the host", err)
	}
}

func TestMutateHostRejectsInvalidResult(t *testing.T) {
	dir := copyFixture(t)
	s := mustOpen(t, dir)
	before := readFile(t, filepath.Join(dir, hostRel))
	err := s.MutateHost(fixtureIP, func(h *model.Host) error {
		h.IP = ""
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), hostRel) {
		t.Fatalf("err = %v, want a rejection naming host.json", err)
	}
	if got := readFile(t, filepath.Join(dir, hostRel)); got != before {
		t.Fatal("host.json changed on disk after a rejected mutation")
	}
}

func TestChecksRejectBadValues(t *testing.T) {
	cases := []struct {
		name string
		do   func(*Store) error
		want string
	}{
		{"meta type", func(s *Store) error {
			return s.MutateMeta(func(m *model.EngagementMeta) error { m.Type = "bogus"; return nil })
		}, "engagement.json"},
		{"meta empty flag id", func(s *Store) error {
			return s.MutateMeta(func(m *model.EngagementMeta) error {
				m.FlagTypes = append(m.FlagTypes, model.FlagType{ID: "", Points: 1})
				return nil
			})
		}, "empty id"},
		{"meta duplicate flag id", func(s *Store) error {
			return s.MutateMeta(func(m *model.EngagementMeta) error {
				m.FlagTypes = []model.FlagType{{ID: "a"}, {ID: "a"}}
				return nil
			})
		}, "duplicate flag type"},
		{"meta negative points", func(s *Store) error {
			return s.MutateMeta(func(m *model.EngagementMeta) error {
				m.FlagTypes = []model.FlagType{{ID: "a", Points: -1}}
				return nil
			})
		}, "negative points"},
		{"host status", func(s *Store) error {
			return s.MutateHost(fixtureIP, func(h *model.Host) error { h.Status = "pwned"; return nil })
		}, "invalid host status"},
		{"item status", func(s *Store) error {
			return s.MutateHost(fixtureIP, func(h *model.Host) error {
				h.Checklist.Items = map[string]model.ItemState{"x": {Status: "maybe"}}
				return nil
			})
		}, "invalid status"},
		{"duplicate service", func(s *Store) error {
			return s.MutateHost(fixtureIP, func(h *model.Host) error {
				h.Services = append(h.Services, h.Services[0])
				return nil
			})
		}, "duplicate service"},
		{"duplicate flag", func(s *Store) error {
			return s.MutateHost(fixtureIP, func(h *model.Host) error {
				h.Flags = []model.Flag{{TypeID: "user"}, {TypeID: "user"}}
				return nil
			})
		}, "duplicate flag"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := mustOpen(t, copyFixture(t))
			err := tc.do(s)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want it to contain %q", err, tc.want)
			}
			if !strings.Contains(err.Error(), ".json") {
				t.Fatalf("err = %v, want it to name the path", err)
			}
		})
	}
}

func TestChecksAcceptEmptyItemStatus(t *testing.T) {
	s := mustOpen(t, copyFixture(t))
	err := s.MutateHost(fixtureIP, func(h *model.Host) error {
		h.Checklist.Items = map[string]model.ItemState{"x": {Status: ""}}
		return nil
	})
	if err != nil {
		t.Fatalf("empty item status rejected: %v", err)
	}
}

func TestReloadRejectsInvalidFileOnDisk(t *testing.T) {
	dir := copyFixture(t)
	s := mustOpen(t, dir)
	path := filepath.Join(dir, hostRel)
	edited := strings.Replace(readFile(t, path), `"status": "`, `"status": "bogus`, 1)
	mustWrite(t, path, edited)
	shiftMtime(t, path, time.Minute)
	if _, err := s.Reload(); err == nil || !strings.Contains(err.Error(), "invalid host status") {
		t.Fatalf("Reload err = %v, want invalid host status", err)
	}
}

func TestReloadReportsNotesEdits(t *testing.T) {
	dir := copyFixture(t)
	notes := filepath.Join(dir, "hosts", fixtureIP, "notes.md")
	mustWrite(t, notes, "first")
	s := mustOpen(t, dir)
	if changed, err := s.Reload(); err != nil || len(changed) != 0 {
		t.Fatalf("idle Reload = %v, %v", changed, err)
	}
	mustWrite(t, notes, "second")
	shiftMtime(t, notes, time.Minute)
	changed, err := s.Reload()
	want := "hosts/" + fixtureIP + "/notes.md"
	if err != nil || len(changed) != 1 || changed[0] != want {
		t.Fatalf("Reload = %v, %v; want [%s]", changed, err, want)
	}
	engNotes := filepath.Join(dir, "notes.md")
	mustWrite(t, engNotes, "eng")
	if changed, err := s.Reload(); err != nil || len(changed) != 1 || changed[0] != "notes.md" {
		t.Fatalf("Reload = %v, %v; want [notes.md]", changed, err)
	}
}

func TestWriteNotesIsNotAnExternalChange(t *testing.T) {
	dir := copyFixture(t)
	s := mustOpen(t, dir)
	_, hash, err := s.ReadNotes("")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.WriteNotes("", "mine", hash); err != nil {
		t.Fatal(err)
	}
	if changed, err := s.Reload(); err != nil || len(changed) != 0 {
		t.Fatalf("Reload = %v, %v; want no change", changed, err)
	}
}

func TestWriteAtomicJoinsCleanupError(t *testing.T) {
	boom := errors.New("cannot remove")
	orig := removeFile
	removeFile = func(string) error { return boom }
	t.Cleanup(func() { removeFile = orig })
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.MkdirAll(filepath.Join(target, "child"), dirMode); err != nil {
		t.Fatal(err)
	}
	err := WriteAtomic(target, []byte("x")) // renaming a file over a non-empty directory fails
	if err == nil || !errors.Is(err, boom) {
		t.Fatalf("err = %v, want it to wrap the cleanup error", err)
	}
}

func TestReloadRejectsEmptiedHostFile(t *testing.T) {
	dir := copyFixture(t)
	s := mustOpen(t, dir)
	path := filepath.Join(dir, hostRel)
	mustWrite(t, path, "")
	shiftMtime(t, path, time.Minute)
	changed, err := s.Reload()
	if err == nil || !strings.Contains(err.Error(), hostRel) {
		t.Fatalf("Reload err = %v, want error naming %s", err, hostRel)
	}
	for _, c := range changed {
		if c == hostRel {
			t.Fatalf("emptied host reported as changed: %v", changed)
		}
	}
	if h, ok := s.Engagement().Host(fixtureIP); !ok || h.Hostname != "nibbles" {
		t.Fatalf("last good host lost: %+v ok=%v", h, ok)
	}
}

func TestMutateUnknownHostDoesNotInsertPhantom(t *testing.T) {
	s := mustOpen(t, copyFixture(t))
	before := len(s.Engagement().Hosts)
	err := s.MutateHost("10.99.99.99", func(*model.Host) error { return nil })
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("err = %v, want not found", err)
	}
	if got := len(s.Engagement().Hosts); got != before {
		t.Fatalf("hosts = %d, want %d", got, before)
	}
}

func TestIPv6HostIsCanonicalised(t *testing.T) {
	dir := copyFixture(t)
	s := mustOpen(t, dir)
	if err := s.AddHost(model.Host{ID: "V6", IP: "FE80::1", Status: model.HostStatusUntouched}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "hosts", "fe80::1", "host.json")); err != nil {
		t.Fatal(err)
	}
	if h, ok := s.Engagement().Host("fe80::1"); !ok || h.IP != "fe80::1" {
		t.Fatalf("host = %+v ok=%v", h, ok)
	}
	err := s.MutateHost("FE80::1", func(h *model.Host) error { h.Hostname = "v6"; return nil })
	if err != nil {
		t.Fatal(err)
	}
	if h, _ := s.Engagement().Host("fe80::1"); h.Hostname != "v6" {
		t.Fatalf("hostname = %q", h.Hostname)
	}
	if err := s.AddHost(model.Host{ID: "V6b", IP: "fe80::1"}); !errors.Is(err, ErrHostExists) {
		t.Fatalf("duplicate err = %v", err)
	}
}

func TestMutateCredsCreatesNormalisedFile(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "engagement.json"), `{"id":"X","name":"x","type":"lab"}`)
	s := mustOpen(t, dir)
	if err := s.MutateCreds(func(*model.Creds) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, filepath.Join(dir, credsRel)); !strings.Contains(got, `"credentials": []`) {
		t.Fatalf("creds.json not normalised:\n%s", got)
	}
}

func TestReloadErrorsCarryTheFilePath(t *testing.T) {
	dir := copyFixture(t)
	s := mustOpen(t, dir)
	path := filepath.Join(dir, "hosts", fixtureIP, "host.json")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	bumpMtime(t, path)
	_, err := s.Reload()
	if err == nil || ErrorPath(err) != path {
		t.Fatalf("Reload err = %v, ErrorPath = %q, want %q", err, ErrorPath(err), path)
	}
	if ErrorPath(errors.New("plain")) != "" {
		t.Fatal("a plain error has no path")
	}
}

func TestHandWrittenHostWithoutStatusLoadsAsUntouched(t *testing.T) {
	dir := copyFixture(t)
	ip := "10.10.10.9"
	hostDir := filepath.Join(dir, "hosts", ip)
	if err := os.MkdirAll(hostDir, 0o700); err != nil {
		t.Fatal(err)
	}
	minimal := "{\n  \"ip\": \"" + ip + "\"\n}\n"
	if err := os.WriteFile(filepath.Join(hostDir, "host.json"), []byte(minimal), 0o600); err != nil {
		t.Fatal(err)
	}
	s := mustOpen(t, dir)
	h, ok := s.Engagement().Host(ip)
	if !ok || h.Status != model.HostStatusUntouched {
		t.Fatalf("host = %+v (found %v), want status untouched", h, ok)
	}
}

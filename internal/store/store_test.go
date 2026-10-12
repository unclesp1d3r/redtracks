package store

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/unclesp1d3r/redtracks/internal/model"
)

const (
	fixtureDir = "testdata/pg-sample"
	fixtureIP  = "10.10.10.5"
	hostRel    = "hosts/10.10.10.5/host.json"
)

// copyFixture copies the pg-sample engagement into a temp dir.
func copyFixture(t *testing.T) string {
	t.Helper()
	dst := t.TempDir()
	err := filepath.WalkDir(fixtureDir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(fixtureDir, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o750)
		}
		data, err := os.ReadFile(p) //nolint:gosec // test fixture path
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, fileMode) //nolint:gosec // target is inside t.TempDir
	})
	if err != nil {
		t.Fatal(err)
	}
	return dst
}

func mustOpen(t *testing.T, dir string) *Store {
	t.Helper()
	s, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return s
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), fileMode); err != nil {
		t.Fatal(err)
	}
}

func TestOpenLoadsFixture(t *testing.T) {
	s := mustOpen(t, fixtureDir)
	e := s.Engagement()
	if len(e.Hosts) != 1 || e.Hosts[0].IP != fixtureIP {
		t.Fatalf("hosts = %+v", e.Hosts)
	}
	if len(e.Hosts[0].Services) != 1 || e.Hosts[0].Services[0].Port != 80 {
		t.Fatalf("services = %+v", e.Hosts[0].Services)
	}
	if len(e.Meta.FlagTypes) != 2 || e.Meta.FlagTypes[0].ID != "local" || e.Meta.FlagTypes[1].ID != "proof" {
		t.Fatalf("flag types = %+v", e.Meta.FlagTypes)
	}
	if len(e.Creds.Credentials) != 1 || e.Dir != fixtureDir {
		t.Fatalf("creds = %+v dir = %q", e.Creds.Credentials, e.Dir)
	}
}

func TestOpenMissingFilesIsFine(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "engagement.json"), `{"id":"X","name":"x","type":"lab"}`)
	e := mustOpen(t, dir).Engagement()
	if len(e.Hosts) != 0 || len(e.Creds.Credentials) != 0 || len(e.Findings) != 0 {
		t.Fatalf("expected empty engagement, got %+v", e)
	}
}

func TestMutateHostRoundTrip(t *testing.T) {
	dir := copyFixture(t)
	s := mustOpen(t, dir)
	err := s.MutateHost(fixtureIP, func(h *model.Host) error {
		h.Status = model.HostStatusFoothold
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	raw := readFile(t, filepath.Join(dir, hostRel))
	if !strings.Contains(raw, "\n  \"status\": \"foothold\"") || !strings.HasSuffix(raw, "}\n") {
		t.Fatalf("not two-space indented with trailing newline:\n%s", raw)
	}
	h, _ := mustOpen(t, dir).Engagement().Host(fixtureIP)
	if h.Status != model.HostStatusFoothold {
		t.Fatalf("status = %q", h.Status)
	}
}

func TestMutateHostAppliesOnTopOfExternalEdit(t *testing.T) {
	dir := copyFixture(t)
	s := mustOpen(t, dir)
	path := filepath.Join(dir, hostRel)
	raw := readFile(t, path)
	edited := strings.Replace(raw, `"chain-a"`, `"chain-a", "external"`, 1)
	mustWrite(t, path, edited)
	err := s.MutateHost(fixtureIP, func(h *model.Host) error {
		h.Hostname = "app"
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	h, _ := mustOpen(t, dir).Engagement().Host(fixtureIP)
	if h.Hostname != "app" || len(h.Tags) != 2 || h.Tags[1] != "external" {
		t.Fatalf("host = %+v", h)
	}
}

func TestMutateCorruptFileRefusesToWrite(t *testing.T) {
	dir := copyFixture(t)
	s := mustOpen(t, dir)
	path := filepath.Join(dir, "creds.json")
	mustWrite(t, path, "{")
	err := s.MutateCreds(func(c *model.Creds) error {
		c.Users = append(c.Users, model.User{ID: "U"})
		return nil
	})
	var pe *ParseError
	if !errors.As(err, &pe) {
		t.Fatalf("want *ParseError, got %v", err)
	}
	if raw := readFile(t, path); raw != "{" {
		t.Fatalf("file was rewritten: %q", raw)
	}
}

func TestHostIPMismatchIsParseError(t *testing.T) {
	dir := t.TempDir()
	hostDir := filepath.Join(dir, "hosts", "10.0.0.9")
	if err := os.MkdirAll(hostDir, 0o750); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(hostDir, "host.json"), `{"id":"A","ip":"10.0.0.8"}`)
	_, err := Open(dir)
	if err == nil {
		t.Fatal("want error")
	}
	for _, want := range []string{"10.0.0.9", "10.0.0.8", "host.json"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q missing %q", err, want)
		}
	}
}

func TestWriteNotesConflict(t *testing.T) {
	dir := copyFixture(t)
	s := mustOpen(t, dir)
	_, h1, err := s.ReadNotes(fixtureIP)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "hosts", fixtureIP, "notes.md")
	mustWrite(t, path, "edited in vim\n")
	text, h2, err := s.ReadNotes(fixtureIP)
	if err != nil || text != "edited in vim\n" || h1 == h2 {
		t.Fatalf("refetch: %q %q %q %v", text, h1, h2, err)
	}
	err = s.WriteNotes(fixtureIP, "stale write", h1)
	if !errors.Is(err, ErrConflict) || !strings.Contains(err.Error(), path) {
		t.Fatalf("want ErrConflict with path, got %v", err)
	}
	if raw := readFile(t, path); raw != "edited in vim\n" {
		t.Fatalf("external text clobbered: %q", raw)
	}
	if err := s.WriteNotes(fixtureIP, "merged", h2); err != nil {
		t.Fatal(err)
	}
	if raw := readFile(t, path); raw != "merged" {
		t.Fatalf("notes = %q", raw)
	}
}

func TestReadNotesMissingFile(t *testing.T) {
	dir := copyFixture(t)
	s := mustOpen(t, dir)
	text, _, err := s.ReadNotes("")
	if err != nil || text == "" {
		t.Fatalf("engagement notes: %q %v", text, err)
	}
	if err := os.Remove(filepath.Join(dir, "notes.md")); err != nil {
		t.Fatal(err)
	}
	text, hash, err := s.ReadNotes("")
	const emptySHA = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if err != nil || text != "" || hash != emptySHA {
		t.Fatalf("missing notes: %q %q %v", text, hash, err)
	}
	if err := s.WriteNotes("", "fresh", hash); err != nil {
		t.Fatalf("write over missing: %v", err)
	}
}

func TestHostPathRejectsTraversal(t *testing.T) {
	dir := copyFixture(t)
	s := mustOpen(t, dir)
	before := listTree(t, dir)
	for _, ip := range []string{"../x", "a/b", "/etc", "10.0.0.1/..", "fe80::1%a/../b", ""} {
		if err := s.AddHost(model.Host{ID: "X", IP: ip}); err == nil {
			t.Errorf("AddHost(%q) succeeded", ip)
		}
		if _, err := s.SaveImport(ip, "nmap", []byte("<x/>")); err == nil {
			t.Errorf("SaveImport(%q) succeeded", ip)
		}
		if err := s.MutateHost(ip, func(*model.Host) error { return nil }); err == nil {
			t.Errorf("MutateHost(%q) succeeded", ip)
		}
	}
	if _, err := s.SaveImport(fixtureIP, "../evil", []byte("x")); err == nil {
		t.Error("SaveImport accepted a traversing tool name")
	}
	if after := listTree(t, dir); strings.Join(before, "|") != strings.Join(after, "|") {
		t.Fatalf("tree changed:\n%v\n%v", before, after)
	}
}

func listTree(t *testing.T, dir string) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(dir, func(p string, _ os.DirEntry, err error) error {
		out = append(out, p)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestReloadKeepsLastGoodOnParseError(t *testing.T) {
	dir := copyFixture(t)
	s := mustOpen(t, dir)
	mustWrite(t, filepath.Join(dir, hostRel), "{")
	future := time.Now().Add(time.Minute)
	if err := os.Chtimes(filepath.Join(dir, hostRel), future, future); err != nil {
		t.Fatal(err)
	}
	_, err := s.Reload()
	var pe *ParseError
	if !errors.As(err, &pe) {
		t.Fatalf("want *ParseError, got %v", err)
	}
	if h, ok := s.Engagement().Host(fixtureIP); !ok || h.Hostname != "nibbles" {
		t.Fatalf("last good host lost: %+v", h)
	}
}

func TestRefreshSkipsReadWhenMtimeUnchanged(t *testing.T) {
	dir := copyFixture(t)
	s := mustOpen(t, dir)
	path := filepath.Join(dir, hostRel)
	if err := os.Chmod(path, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(path, 0o600); err != nil {
			t.Error(err)
		}
	})
	if _, err := os.ReadFile(path); err == nil {
		t.Skip("file stays readable (running as root?)")
	}
	changed, err := s.Reload()
	if err != nil || len(changed) != 0 {
		t.Fatalf("Reload = %v, %v; want no change and no error", changed, err)
	}
}

func TestAddHostCreatesFolder(t *testing.T) {
	dir := copyFixture(t)
	s := mustOpen(t, dir)
	if err := s.AddHost(model.Host{ID: "N", IP: "10.0.0.2"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "hosts", "10.0.0.2", "host.json")); err != nil {
		t.Fatal(err)
	}
	if err := s.AddHost(model.Host{ID: "M", IP: "10.0.0.2"}); err == nil {
		t.Fatal("duplicate AddHost succeeded")
	}
	if _, ok := mustOpen(t, dir).Engagement().Host("10.0.0.2"); !ok {
		t.Fatal("added host not persisted")
	}
}

func TestSaveImportNamesFile(t *testing.T) {
	dir := copyFixture(t)
	s := mustOpen(t, dir)
	raw := []byte("<nmaprun/>")
	rel, err := s.SaveImport(fixtureIP, "nmap", raw)
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`^hosts/10\.10\.10\.5/imports/nmap-\d{8}-\d{6}\.xml$`).MatchString(rel) {
		t.Fatalf("rel = %q", rel)
	}
	got := readFile(t, filepath.Join(dir, filepath.FromSlash(rel)))
	if got != string(raw) {
		t.Fatalf("bytes = %q", got)
	}
	rel2, err := s.SaveImport(fixtureIP, "nmap", raw)
	if err != nil || rel2 == rel {
		t.Fatalf("second import must not overwrite: %q %v", rel2, err)
	}
	for tool, wantExt := range map[string]string{"ffuf": ".json", "curl": ".txt"} {
		body := map[string]string{"ffuf": `{"a":1}`, "curl": "plain"}[tool]
		relExt, err := s.SaveImport(fixtureIP, tool, []byte(body))
		if err != nil || !strings.HasSuffix(relExt, wantExt) {
			t.Fatalf("%s: %q %v", tool, relExt, err)
		}
	}
}

func TestReloadReportsChangedFiles(t *testing.T) {
	dir := copyFixture(t)
	s := mustOpen(t, dir)
	path := filepath.Join(dir, hostRel)
	raw := readFile(t, path)
	mustWrite(t, path, strings.Replace(raw, `"nibbles"`, `"renamed"`, 1))
	future := time.Now().Add(time.Minute)
	if err := os.Chtimes(path, future, future); err != nil {
		t.Fatal(err)
	}
	changed, err := s.Reload()
	if err != nil {
		t.Fatal(err)
	}
	if len(changed) != 1 || changed[0] != hostRel {
		t.Fatalf("changed = %v", changed)
	}
	if h, _ := s.Engagement().Host(fixtureIP); h.Hostname != "renamed" {
		t.Fatalf("hostname = %q", h.Hostname)
	}
	if again, err := s.Reload(); err != nil || len(again) != 0 {
		t.Fatalf("second reload changed = %v", again)
	}
}

func TestEngagementReturnsDeepCopy(t *testing.T) {
	s := mustOpen(t, fixtureDir)
	e := s.Engagement()
	e.Hosts[0].Hostname = "mutated"
	e.Hosts[0].Services[0].Port = 1
	e.Meta.FlagTypes[0].ID = "mutated"
	e.Creds.Credentials[0].Username = "mutated"
	again := s.Engagement()
	if again.Hosts[0].Hostname != "nibbles" || again.Hosts[0].Services[0].Port != 80 ||
		again.Meta.FlagTypes[0].ID != "local" || again.Creds.Credentials[0].Username != "admin" {
		t.Fatalf("snapshot aliased store state: %+v", again)
	}
}

func TestMutateRollsBackOnFnError(t *testing.T) {
	dir := copyFixture(t)
	s := mustOpen(t, dir)
	boom := errors.New("boom")
	err := s.MutateHost(fixtureIP, func(h *model.Host) error {
		h.Hostname = "half"
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v", err)
	}
	if h, _ := s.Engagement().Host(fixtureIP); h.Hostname != "nibbles" {
		t.Fatalf("failed mutation leaked: %q", h.Hostname)
	}
}

func TestMutateMetaAndFindingsNormalize(t *testing.T) {
	dir := copyFixture(t)
	s := mustOpen(t, dir)
	if err := s.MutateMeta(func(m *model.EngagementMeta) error { m.FlagTypes = nil; return nil }); err != nil {
		t.Fatal(err)
	}
	if err := s.MutateFindings(func(f *model.FindingsFile) error { f.Findings = nil; return nil }); err != nil {
		t.Fatal(err)
	}
	meta := readFile(t, filepath.Join(dir, "engagement.json"))
	finds := readFile(t, filepath.Join(dir, "findings.json"))
	if !strings.Contains(meta, `"flagTypes": []`) || !strings.Contains(finds, `"findings": []`) {
		t.Fatalf("nil slices not normalised:\n%s\n%s", meta, finds)
	}
}

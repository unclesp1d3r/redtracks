package app

import (
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/unclesp1d3r/redtracks/internal/model"
	"github.com/unclesp1d3r/redtracks/internal/store"
)

const waitTimeout = 3 * time.Second

func TestErrorEventsSplitJoinedErrorsAndRelativisePaths(t *testing.T) {
	base := filepath.Join("eng", "dir")
	parse := &store.ParseError{
		Path: filepath.Join(base, "hosts", "10.0.0.1", "host.json"), Line: 1, Col: 1, Err: errors.New("bad"),
	}
	joined := errors.Join(parse, errors.New("plain"))

	got := errorEvents(errorKindEngagement, base, joined)

	if len(got) != 2 {
		t.Fatalf("events = %+v, want one per joined error", got)
	}
	first := got[0]
	if first.Kind != errorKindEngagement || first.Path != "hosts/10.0.0.1/host.json" ||
		!strings.Contains(first.Message, "bad") {
		t.Fatalf("parse event = %+v", got[0])
	}
	if got[1].Path != "" || got[1].Message != "plain" {
		t.Fatalf("plain event = %+v", got[1])
	}
}

func TestPollErrorEmitsStructuredEvent(t *testing.T) {
	f := newTestApp(t)
	ev := &events{}
	f.app.SetEmitter(ev.emit)
	if _, err := NewEngagementService(f.app).Open(f.dir); err != nil {
		t.Fatal(err)
	}
	host := filepath.Join(f.dir, "hosts", sampleHost, "host.json")
	if err := os.WriteFile(host, []byte("{"), logFileMode); err != nil {
		t.Fatal(err)
	}
	future := time.Now().Add(time.Minute)
	if err := os.Chtimes(host, future, future); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(waitTimeout)
	for len(ev.errorEvents()) == 0 && time.Now().Before(deadline) {
		time.Sleep(testPollGap)
	}
	got := ev.errorEvents()
	if len(got) == 0 {
		t.Fatal("no structured error event emitted")
	}
	if got[0].Kind != "engagement" || got[0].Path != "hosts/"+sampleHost+"/host.json" {
		t.Fatalf("event = %+v", got[0])
	}
}

func TestChecklistStatusReportsLoadError(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, checklistsDir)
	if err := os.MkdirAll(dir, dirMode); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "bad.json"), []byte("{"), logFileMode); err != nil {
		t.Fatal(err)
	}
	a, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = a.Close() }()
	st := NewChecklistService(a).Status()
	if st.Count != 0 || !strings.Contains(st.Error, "bad.json") || st.InstallError != "" {
		t.Fatalf("status = %+v", st)
	}
}

func TestChecklistStatusKeepsInstallError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, engagementsDir), dirMode); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, logFileName), nil, logFileMode); err != nil {
		t.Fatal(err)
	}
	const readOnlyDir = 0o500
	if err := os.Chmod(root, readOnlyDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(root, dirMode); err != nil {
			t.Error(err)
		}
	})
	a, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = a.Close() }()
	st := NewChecklistService(a).Status()
	if st.InstallError == "" {
		t.Fatalf("status = %+v, want the install error to survive reloadDefs", st)
	}
	if st.Error != "" {
		t.Fatalf("load error = %q, want none (the empty set loaded)", st.Error)
	}
}

func TestNewInstallsDefaultsStatusCount(t *testing.T) {
	a, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = a.Close() }()
	if st := NewChecklistService(a).Status(); st.Count != shippedCount(t) || st.Error != "" || st.InstallError != "" {
		t.Fatalf("status = %+v", st)
	}
}

func TestListReturnsBrokenEngagementsAndRoot(t *testing.T) {
	f := newTestApp(t)
	bad := filepath.Join(f.app.Root, engagementsDir, "bad")
	if err := os.MkdirAll(bad, dirMode); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bad, "engagement.json"), []byte("{"), logFileMode); err != nil {
		t.Fatal(err)
	}
	e := NewEngagementService(f.app)
	got, err := e.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Engagements) != 1 || len(got.Broken) != 1 || got.Broken[0].Dir != bad {
		t.Fatalf("List = %+v", got)
	}
	if e.Root() != f.app.Root {
		t.Fatalf("Root = %q, want %q", e.Root(), f.app.Root)
	}
}

func TestCreateRejectsInvalidType(t *testing.T) {
	a := newTestApp(t).app
	_, err := NewEngagementService(a).Create("x", "", "bogus")
	if err == nil || !strings.Contains(err.Error(), "bogus") {
		t.Fatalf("err = %v, want one naming the bad type", err)
	}
}

func TestSetStatusAndSetItemRejectBadValues(t *testing.T) {
	a := openSample(t).app
	h := NewHostService(a)
	if err := h.SetStatus(sampleHost, "pwned"); err == nil || !strings.Contains(err.Error(), "pwned") {
		t.Fatalf("SetStatus err = %v", err)
	}
	err := h.SetItem(sampleHost, "", "recon.ping", "maybe", "")
	if err == nil || !strings.Contains(err.Error(), "maybe") {
		t.Fatalf("SetItem err = %v", err)
	}
}

func TestUpdateDoesNotWriteStatus(t *testing.T) {
	a := openSample(t).app
	h := NewHostService(a)
	if err := h.SetStatus(sampleHost, model.HostStatusFoothold); err != nil {
		t.Fatal(err)
	}
	if err := h.Update(model.Host{IP: sampleHost, Hostname: "renamed", Status: model.HostStatusUntouched}); err != nil {
		t.Fatal(err)
	}
	got, _ := current(t, a).Host(sampleHost)
	if got.Hostname != "renamed" || got.Status != model.HostStatusFoothold {
		t.Fatalf("host = %+v, want renamed and status untouched by Update", got)
	}
}

func TestAddFindingLowercasesSeverityAndChecksHost(t *testing.T) {
	a := openSample(t).app
	e := NewEngagementService(a)
	host, _ := current(t, a).Host(sampleHost)

	if err := e.AddFinding(model.Finding{HostID: host.ID, Title: "t", Severity: " HIGH "}); err != nil {
		t.Fatal(err)
	}
	findings := current(t, a).Findings
	if len(findings) != 1 || findings[0].Severity != "high" {
		t.Fatalf("findings = %+v", findings)
	}
	if err := e.AddFinding(model.Finding{HostID: "nope", Title: "t"}); !errors.Is(err, errNotFound) {
		t.Fatalf("AddFinding unknown host err = %v", err)
	}
	upd := findings[0]
	upd.Severity, upd.HostID = "CRITICAL", "nope"
	if err := e.UpdateFinding(upd); !errors.Is(err, errNotFound) {
		t.Fatalf("UpdateFinding unknown host err = %v", err)
	}
	upd.HostID = host.ID
	if err := e.UpdateFinding(upd); err != nil {
		t.Fatal(err)
	}
	if got := current(t, a).Findings[0].Severity; got != "critical" {
		t.Fatalf("severity = %q", got)
	}
}

// previewNmap previews the nmap fixture through PreviewData.
func previewNmap(t *testing.T, imp *ImportService) PreviewResponse {
	t.Helper()
	raw, err := os.ReadFile(nmapFixture)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := imp.PreviewData("nmap.xml", base64.StdEncoding.EncodeToString(raw))
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func TestPendingImportsClearedOnOpen(t *testing.T) {
	f := openSample(t)
	imp := NewImportService(f.app)
	resp := previewNmap(t, imp)
	if _, err := NewEngagementService(f.app).Open(f.dir); err != nil {
		t.Fatal(err)
	}
	if _, err := imp.Apply(resp.Token, nil); !errors.Is(err, errUnknownToken) {
		t.Fatalf("Apply after Open err = %v, want unknown token", err)
	}
}

func TestPendingImportsAreCapped(t *testing.T) {
	f := openSample(t)
	imp := NewImportService(f.app)
	first := previewNmap(t, imp)
	last := first
	for range maxPendingImports {
		last = previewNmap(t, imp)
	}
	if err := imp.Discard(first.Token); !errors.Is(err, errUnknownToken) {
		t.Fatalf("oldest token survived the cap: %v", err)
	}
	if err := imp.Discard(last.Token); err != nil {
		t.Fatalf("newest token lost: %v", err)
	}
}

func TestDiagnosticsLogErrorWritesLog(t *testing.T) {
	a := newTestApp(t).app
	NewDiagnosticsService(a).LogError("HostPage", "render exploded")
	data, err := os.ReadFile(filepath.Join(a.Root, logFileName))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "HostPage") || !strings.Contains(string(data), "render exploded") {
		t.Fatalf("log = %s", data)
	}
}

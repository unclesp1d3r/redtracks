package app

import (
	"bytes"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/unclesp1d3r/redtracks/internal/model"
)

// waitFor polls cond until it holds or waitTimeout passes.
func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(waitTimeout)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(testPollGap)
	}
}

func globFiles(t *testing.T, pattern string) []string {
	t.Helper()
	files, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func writeWithMtime(t *testing.T, path, content string, mtime time.Time) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), logFileMode); err != nil { //nolint:gosec // path is inside t.TempDir
		t.Fatal(err)
	}
	if err := os.Chtimes(path, mtime, mtime); err != nil {
		t.Fatal(err)
	}
}

func TestPollReportsExternalEditThenCorruption(t *testing.T) {
	f := newTestApp(t)
	ev := &events{}
	f.app.SetEmitter(ev.emit)
	if _, err := NewEngagementService(f.app).Open(f.dir); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(f.dir, "hosts", sampleHost, "host.json")
	orig, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	rel := "hosts/" + sampleHost + "/host.json"

	edited := strings.Replace(string(orig), `"hostname": "nibbles"`, `"hostname": "edited"`, 1)
	writeWithMtime(t, path, edited, time.Now().Add(time.Minute))
	// The checklist watcher also raises engagement:changed on its first tick, so wait
	// for the event that names this file rather than for any event at all.
	waitFor(t, "engagement:changed for "+rel, func() bool { return slices.Contains(ev.changedPaths(), rel) })
	if h, _ := current(t, f.app).Host(sampleHost); h.Hostname != "edited" {
		t.Fatalf("hostname = %q, want the external edit", h.Hostname)
	}

	writeWithMtime(t, path, "{", time.Now().Add(2*time.Minute))
	waitFor(t, "engagement:error", func() bool { return len(ev.errorEvents()) > 0 })
	got := ev.errorEvents()[0]
	if got.Kind != errorKindEngagement || got.Path != rel || got.Message == "" {
		t.Fatalf("error event = %+v", got)
	}
}

func TestUpdateWithStaleSnapshotKeepsOtherFields(t *testing.T) {
	a := openSample(t).app
	h := NewHostService(a)
	host, _ := current(t, a).Host(sampleHost)
	stale := host

	if err := h.SetStatus(sampleHost, model.HostStatusRoot); err != nil {
		t.Fatal(err)
	}
	if err := h.SetFlag(sampleHost, model.Flag{TypeID: "local", Value: "fresh"}); err != nil {
		t.Fatal(err)
	}
	if err := h.SetItem(sampleHost, "", "recon.ping", model.ItemDone, ""); err != nil {
		t.Fatal(err)
	}
	live, _ := current(t, a).Host(sampleHost)

	stale.Hostname, stale.OS = "renamed", "windows"
	stale.Tags, stale.Services, stale.Flags, stale.AccessPaths = nil, nil, nil, nil
	stale.Checklist, stale.Status, stale.ID = model.ChecklistState{}, model.HostStatusUntouched, "other"
	if err := h.Update(stale); err != nil {
		t.Fatal(err)
	}

	got, _ := current(t, a).Host(sampleHost)
	if got.Hostname != "renamed" || got.OS != "windows" {
		t.Fatalf("hostname/os = %q/%q, want the update applied", got.Hostname, got.OS)
	}
	if got.ID != live.ID || got.Status != model.HostStatusRoot || !slices.Equal(got.Tags, live.Tags) ||
		len(got.Tags) == 0 || len(got.Services) != len(live.Services) || len(got.Flags) != len(live.Flags) ||
		len(got.AccessPaths) != len(live.AccessPaths) || got.Checklist.Items["recon.ping"].Status != model.ItemDone {
		t.Fatalf("host = %+v, want everything but hostname and OS kept from %+v", got, live)
	}
}

func TestUpdateCredentialKeepsWorksOnAndTriedOn(t *testing.T) {
	a := openSample(t).app
	e := NewEngagementService(a)
	host := addSecondHost(t, a)
	cred := current(t, a).Creds.Credentials[0]
	if err := e.MarkTried(cred.ID, host.IP, true); err != nil {
		t.Fatal(err)
	}

	stale := cred
	stale.WorksOn, stale.TriedOn, stale.Notes = nil, nil, "edited"
	if err := e.UpdateCredential(stale); err != nil {
		t.Fatal(err)
	}

	got := current(t, a).Creds.Credentials[0]
	if got.Notes != "edited" || !slices.Equal(got.WorksOn, []string{host.ID}) ||
		!slices.Equal(got.TriedOn, []string{host.ID}) {
		t.Fatalf("credential = %+v, want notes edited and WorksOn/TriedOn kept", got)
	}
}

func TestSetFlagReplacesByTypeID(t *testing.T) {
	a := openSample(t).app
	h := NewHostService(a)
	before, _ := current(t, a).Host(sampleHost)
	for _, v := range []string{"first", "second"} {
		if err := h.SetFlag(sampleHost, model.Flag{TypeID: "local", Value: v}); err != nil {
			t.Fatal(err)
		}
	}
	got, _ := current(t, a).Host(sampleHost)
	n := 0
	for _, f := range got.Flags {
		if f.TypeID == "local" {
			n++
			if f.Value != "second" {
				t.Fatalf("flag = %+v, want the second value", f)
			}
		}
	}
	if n != 1 || len(got.Flags) != len(before.Flags) {
		t.Fatalf("flags = %+v, want one local flag and no extras (was %d)", got.Flags, len(before.Flags))
	}
}

func TestHostChecklists(t *testing.T) {
	a := openSample(t).app
	h := NewHostService(a)
	host, _ := current(t, a).Host(sampleHost)

	hostViews, err := h.Checklists(sampleHost, "")
	if err != nil || len(hostViews) == 0 || len(hostViews[0].Items) == 0 {
		t.Fatalf("host checklists = %+v, err %v; want the linux definitions", hostViews, err)
	}

	svcViews, err := h.Checklists(sampleHost, host.Services[0].ID)
	if err != nil || len(svcViews) == 0 {
		t.Fatalf("service checklists = %+v, err %v", svcViews, err)
	}
	var robots model.ItemStatus
	for _, v := range svcViews {
		for _, it := range v.Items {
			if it.ID == "http.robots" {
				robots = it.State.Status
			}
		}
	}
	if robots != model.ItemDone {
		t.Fatalf("http.robots status = %q, want the recorded done", robots)
	}

	if _, err := h.Checklists("10.9.9.9", ""); !errors.Is(err, errNotFound) {
		t.Fatalf("unknown ip err = %v", err)
	}
	if _, err := h.Checklists(sampleHost, "nope"); !errors.Is(err, errNotFound) {
		t.Fatalf("unknown service err = %v", err)
	}
}

func TestDeleteAndUpdateUnknownRecordsLeaveFileUnchanged(t *testing.T) {
	f := openSample(t)
	e := NewEngagementService(f.app)
	path := filepath.Join(f.dir, "creds.json")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := e.DeleteCredential("nope"); !errors.Is(err, errNotFound) {
		t.Fatalf("DeleteCredential err = %v", err)
	}
	if err := e.UpdateShare(model.Share{ID: "nope", Name: "x"}); !errors.Is(err, errNotFound) {
		t.Fatalf("UpdateShare err = %v", err)
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(after, before) {
		t.Fatalf("creds.json changed (err %v)", err)
	}
}

func TestDiscardUnknownTokenErrors(t *testing.T) {
	a := openSample(t).app
	if err := NewImportService(a).Discard("nope"); !errors.Is(err, errUnknownToken) {
		t.Fatalf("err = %v, want unknown token", err)
	}
}

const twoHostNmap = `<nmaprun>
<host><status state="up"/><address addr="10.10.10.5" addrtype="ipv4"/><ports>
<port protocol="tcp" portid="22"><state state="open"/><service name="ssh"/></port></ports></host>
<host><status state="up"/><address addr="10.10.10.9" addrtype="ipv4"/><ports>
<port protocol="tcp" portid="22"><state state="open"/><service name="ssh"/></port></ports></host>
</nmaprun>`

func TestApplyPartialFailureKeepsTokenAndSavesNoRaw(t *testing.T) {
	f := openSample(t)
	imp := NewImportService(f.app)
	resp, err := imp.PreviewData("scan.xml", base64.StdEncoding.EncodeToString([]byte(twoHostNmap)))
	if err != nil {
		t.Fatal(err)
	}
	bad := filepath.Join(f.dir, "hosts", "10.10.10.9", "host.json")
	if err := os.MkdirAll(filepath.Dir(bad), dirMode); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bad, []byte("{"), logFileMode); err != nil {
		t.Fatal(err)
	}
	rawGlob := filepath.Join(f.dir, "hosts", "*", "imports", "*")

	if _, err := imp.Apply(resp.Token, nil); err == nil {
		t.Fatal("Apply succeeded despite the corrupt host")
	}
	if raws := globFiles(t, rawGlob); len(raws) != 0 {
		t.Fatalf("raw files %v saved after a partial failure", raws)
	}

	if err := os.Remove(bad); err != nil {
		t.Fatal(err)
	}
	if _, err := imp.Apply(resp.Token, nil); err != nil {
		t.Fatalf("retry with the same token: %v", err)
	}
	if raws := globFiles(t, rawGlob); len(raws) != 1 {
		t.Fatalf("raw files %v, want exactly one after the retry", raws)
	}
}

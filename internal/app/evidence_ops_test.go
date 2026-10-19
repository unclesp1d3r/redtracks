package app

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/unclesp1d3r/redtracks/internal/model"
)

const (
	slotLocal   = "exam-proofs.local"
	accessPath  = "H8T2WQ4N1B"
	flagLocal   = "local"
	hostEvidDir = "hosts/10.10.10.5/evidence"
)

var opsPNG = append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte{1, 2, 3}, 50)...)

func b64(b []byte) string { return base64.StdEncoding.EncodeToString(b) }

func addText(t *testing.T, s *EvidenceService, text string, links ...model.EvidenceLink) AddResult {
	t.Helper()
	res, err := s.AddFromData(sampleHost, "n.txt", b64([]byte(text)), 0, "", links)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func listView(t *testing.T, s *EvidenceService) []string {
	t.Helper()
	v, err := s.List(sampleHost)
	if err != nil {
		t.Fatal(err)
	}
	ids := make([]string, len(v.Items))
	for i, it := range v.Items {
		ids[i] = it.ID
	}
	return ids
}

func fileHash(t *testing.T, path string) [32]byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return sha256.Sum256(data)
}

func TestAddFromDataKeepsOnlyTheGivenLink(t *testing.T) { // AE7
	f := openSample(t)
	s := NewEvidenceService(f.app)
	link := model.EvidenceLink{Kind: model.EvidenceLinkFinding, TargetID: "F1"}
	stamp := time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)

	res, err := s.AddFromData(sampleHost, "shot.png", b64(opsPNG), stamp.UnixMilli(), "", []model.EvidenceLink{link})
	if err != nil {
		t.Fatal(err)
	}

	if res.Duplicate || res.Item.Kind != model.EvidenceKindImage || !res.Item.CapturedAt.Equal(stamp) {
		t.Errorf("item = %+v", res)
	}
	if !slices.Equal(res.Item.Links, []model.EvidenceLink{link}) || res.Item.Caption != "" {
		t.Errorf("links/caption = %+v %q", res.Item.Links, res.Item.Caption)
	}
	if again, err := s.AddFromData(sampleHost, "shot.png", b64(opsPNG), 0, "", nil); err != nil || !again.Duplicate {
		t.Errorf("second add duplicate = %v, %v", again.Duplicate, err)
	}
}

func TestAddFromDataValidatesAtTheBoundary(t *testing.T) {
	s := NewEvidenceService(openSample(t).app)
	bad := []model.EvidenceLink{{Kind: "nope", TargetID: "x"}}
	empty := []model.EvidenceLink{{Kind: model.EvidenceLinkFlag}}

	if _, err := s.AddFromData(sampleHost, "n", b64(opsPNG), 0, "", bad); err == nil {
		t.Error("bad link kind accepted")
	}
	if _, err := s.AddFromData(sampleHost, "n", b64(opsPNG), 0, "", empty); err == nil {
		t.Error("empty link target accepted")
	}
	if _, err := s.AddFromData("../x", "n", b64(opsPNG), 0, "", nil); err == nil {
		t.Error("bad ip accepted")
	}
	huge := base64.StdEncoding.EncodeToString(make([]byte, maxImportBytes+1))
	if _, err := s.AddFromData(
		sampleHost,
		"big",
		huge,
		0,
		"",
		nil,
	); err == nil ||
		!strings.Contains(err.Error(), "larger than") {
		t.Errorf("oversize err = %v", err)
	}
	if ids := listView(t, s); len(ids) != 0 {
		t.Errorf("failed adds left items: %v", ids)
	}
}

func TestAddFromDataStripsTerminalSequences(t *testing.T) {
	f := openSample(t)
	s := NewEvidenceService(f.app)

	res := addText(t, s, "\x1b[31mroot\x1b[0m flag\x1b[2K\n")

	data, err := os.ReadFile(filepath.Join(f.dir, hostEvidDir, res.Item.File))
	if err != nil || string(data) != "root flag\n" {
		t.Fatalf("stored %q, %v", data, err)
	}
}

func TestListShowsOneItemWithLabelledLinks(t *testing.T) { // AE1, AE10
	f := openSample(t)
	s := NewEvidenceService(f.app)
	res := addText(t, s, "id\nuid=0", []model.EvidenceLink{
		{Kind: model.EvidenceLinkSlot, TargetID: slotLocal},
		{Kind: model.EvidenceLinkFlag, TargetID: flagLocal},
		{Kind: model.EvidenceLinkAccessPath, TargetID: accessPath},
	}...)

	v, err := s.List(sampleHost)
	if err != nil {
		t.Fatal(err)
	}

	if len(v.Items) != 1 || v.Items[0].ID != res.Item.ID {
		t.Fatalf("items = %+v", v.Items)
	}
	got := v.Items[0].Links
	if got[0].Label != "local.txt proof" || got[1].Label != flagLocal || got[2].Label != "direct" {
		t.Errorf("labels = %+v", got)
	}
	for _, l := range got {
		if l.Removed {
			t.Errorf("link %+v marked removed", l)
		}
	}
}

func TestDeletedFindingShowsRemovedLinkAndUnlinkKeepsItem(t *testing.T) { // AE14
	f := openSample(t)
	s := NewEvidenceService(f.app)
	st, err := f.app.Store()
	if err != nil {
		t.Fatal(err)
	}
	if err := st.MutateFindings(func(ff *model.FindingsFile) error {
		ff.Findings = append(ff.Findings, model.Finding{ID: "FND1", Title: "SQLi"})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	link := model.EvidenceLink{Kind: model.EvidenceLinkFinding, TargetID: "FND1"}
	res := addText(t, s, "proof", link)
	if err := st.MutateFindings(func(ff *model.FindingsFile) error { ff.Findings = nil; return nil }); err != nil {
		t.Fatal(err)
	}

	v := mustList(t, s)
	if l := v.Items[0].Links[0]; !l.Removed {
		t.Fatalf("link = %+v, want removed", l)
	}
	if err := s.Unlink(sampleHost, res.Item.ID, link); err != nil {
		t.Fatal(err)
	}
	v = mustList(t, s)
	if len(v.Items) != 1 || len(v.Items[0].Links) != 0 {
		t.Errorf("after unlink = %+v", v.Items)
	}
}

func TestLinkAndCaptionLeaveFileBytesAlone(t *testing.T) {
	f := openSample(t)
	s := NewEvidenceService(f.app)
	res := addText(t, s, "hello")
	path := filepath.Join(f.dir, hostEvidDir, res.Item.File)
	before := fileHash(t, path)
	link := model.EvidenceLink{Kind: model.EvidenceLinkFlag, TargetID: flagLocal}

	for range 2 { // the second Link is an identical no-op
		if err := s.Link(sampleHost, res.Item.ID, link); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.SetCaption(sampleHost, res.Item.ID, "the caption"); err != nil {
		t.Fatal(err)
	}

	v := mustList(t, s)
	if len(v.Items[0].Links) != 1 || v.Items[0].Caption != "the caption" {
		t.Errorf("item = %+v", v.Items[0])
	}
	if fileHash(t, path) != before {
		t.Error("file bytes changed")
	}
	if err := s.Link(sampleHost, res.Item.ID, model.EvidenceLink{Kind: "x", TargetID: "y"}); err == nil {
		t.Error("bad kind accepted")
	}
	if err := s.Link(sampleHost, "nope", link); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("missing id err = %v", err)
	}
}

func TestDropRemovesEntryOfMissingFile(t *testing.T) { // AE2 path
	f := openSample(t)
	s := NewEvidenceService(f.app)
	res := addText(t, s, "gone soon")
	if err := os.Remove(filepath.Join(f.dir, hostEvidDir, res.Item.File)); err != nil {
		t.Fatal(err)
	}
	if v := mustList(t, s); !v.Items[0].Missing {
		t.Fatalf("item not marked missing: %+v", v.Items[0])
	}

	if err := s.Drop(sampleHost, res.Item.ID); err != nil {
		t.Fatal(err)
	}

	if ids := listView(t, s); len(ids) != 0 {
		t.Errorf("items after drop = %v", ids)
	}
	if err := s.Drop(sampleHost, res.Item.ID); err == nil {
		t.Error("second drop succeeded")
	}
}

func TestAdoptIndexesAnUnindexedFile(t *testing.T) {
	f := openSample(t)
	s := NewEvidenceService(f.app)
	dir := filepath.Join(f.dir, hostEvidDir)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "stray.png")
	if err := os.WriteFile(path, opsPNG, 0o600); err != nil {
		t.Fatal(err)
	}
	mtime := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	if err := os.Chtimes(path, mtime, mtime); err != nil {
		t.Fatal(err)
	}

	item, err := s.Adopt(sampleHost, "stray.png")
	if err != nil {
		t.Fatal(err)
	}

	if item.File != "stray.png" || item.Kind != model.EvidenceKindImage || !item.CapturedAt.Equal(mtime) ||
		len(item.Links) != 0 {
		t.Errorf("item = %+v", item)
	}
	if want := sha256.Sum256(opsPNG); item.SHA256 != hexOf(want) {
		t.Errorf("sha = %s", item.SHA256)
	}
	if _, err := s.Adopt(sampleHost, "stray.png"); err == nil {
		t.Error("re-adopt accepted")
	}
	for _, name := range []string{"../stray.png", "a/b.png", ".hidden", ""} {
		if _, err := s.Adopt(sampleHost, name); err == nil {
			t.Errorf("name %q accepted", name)
		}
	}
	if _, err := s.Adopt(sampleHost, "absent.png"); err == nil {
		t.Error("absent file accepted")
	}
	if err := os.WriteFile(filepath.Join(dir, "x.bin"), []byte{0, 1, 2, 0xff, 0}, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Adopt(sampleHost, "x.bin"); err == nil {
		t.Error("unsupported type adopted")
	}
}

func TestListAllReturnsEveryHostInOrder(t *testing.T) {
	f := openSample(t)
	s := NewEvidenceService(f.app)
	addText(t, s, "x")
	st, err := f.app.Store()
	if err != nil {
		t.Fatal(err)
	}
	hosts := st.Engagement().Hosts

	all, err := s.ListAll()
	if err != nil {
		t.Fatal(err)
	}

	if len(all) != len(hosts) {
		t.Fatalf("got %d entries for %d hosts", len(all), len(hosts))
	}
	for i, h := range hosts {
		if all[i].IP != h.IP {
			t.Errorf("entry %d = %s, want %s", i, all[i].IP, h.IP)
		}
	}
	if i := slices.IndexFunc(
		all,
		func(e HostEvidence) bool { return e.IP == sampleHost },
	); len(
		all[i].View.Items,
	) != 1 {
		t.Errorf("sample host view = %+v", all[i].View)
	}
}

func TestSearchFindsTextEvidenceEngagementWide(t *testing.T) { // AE8
	s := NewEvidenceService(openSample(t).app)
	hit := addText(t, s, "found the Needle here\n")
	addText(t, s, "unrelated\n")

	hits, err := s.Search("needle")
	if err != nil {
		t.Fatal(err)
	}

	if len(hits) != 1 || hits[0].ItemID != hit.Item.ID || hits[0].IP != sampleHost ||
		!strings.Contains(hits[0].Snippet, "Needle") {
		t.Errorf("hits = %+v", hits)
	}
}

func TestRevealHonoursCapabilityAndRunsFixedArgs(t *testing.T) { // AE16
	f := openSample(t)
	s := NewEvidenceService(f.app)
	res := addText(t, s, "x")
	var got []string
	t.Cleanup(swapReveal(func(args []string) error { got = args; return nil }))

	if err := s.Reveal(sampleHost, res.Item.ID); !errors.Is(err, ErrNoReveal) || got != nil {
		t.Fatalf("no capability: err = %v args = %v", err, got)
	}
	f.app.SetHooks(Hooks{Capabilities: Capabilities{Reveal: true}})
	if !s.Capabilities().Reveal {
		t.Fatal("Capabilities does not reflect hooks")
	}
	if err := s.Reveal(sampleHost, res.Item.ID); err != nil {
		t.Fatal(err)
	}
	if len(got) < 2 || !strings.Contains(strings.Join(got, " "), filepath.Join(f.dir, hostEvidDir)) {
		t.Errorf("args = %v", got)
	}
	if err := s.Reveal(sampleHost, "nope"); err == nil {
		t.Error("unknown item revealed")
	}
}

func TestCopyPathReturnsAbsoluteFilePath(t *testing.T) {
	f := openSample(t)
	s := NewEvidenceService(f.app)
	res := addText(t, s, "x")

	p, err := s.CopyPath(sampleHost, res.Item.ID)

	if err != nil || p != filepath.Join(f.dir, hostEvidDir, res.Item.File) || !filepath.IsAbs(p) {
		t.Errorf("path = %q, %v", p, err)
	}
}

func TestEvidencePathRefusesEscapingNames(t *testing.T) {
	root := t.TempDir()

	for _, name := range []string{"../x", "a/b", "", ".."} {
		if _, err := evidenceFilePath(root, sampleHost, name); err == nil {
			t.Errorf("name %q accepted", name)
		}
	}
	if _, err := evidenceFilePath(root, sampleHost, "ok.png"); err != nil {
		t.Errorf("plain name refused: %v", err)
	}
}

func hexOf(sum [32]byte) string { return hex.EncodeToString(sum[:]) }

// swapReveal replaces the reveal seam and returns the restore func.
func swapReveal(fn func([]string) error) func() {
	old := runReveal
	runReveal = fn
	return func() { runReveal = old }
}

func mustList(t *testing.T, s *EvidenceService) EvidenceView {
	t.Helper()
	v, err := s.List(sampleHost)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

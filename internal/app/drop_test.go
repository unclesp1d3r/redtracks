package app

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/unclesp1d3r/redtracks/internal/model"
)

func pngBytes(tag string) []byte {
	return append([]byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR"), []byte(tag)...)
}

func dropFile(t *testing.T, dir, name string, data []byte) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func newDropper(f fixture) *Dropper { return NewDropper(f.app, NewImportService(f.app)) }

func slotAttrs() map[string]string {
	return map[string]string{attrEvidenceHost: sampleHost, attrEvidenceTarget: "slot:web-shot"}
}

func TestHandleDropCreatesLinkedItemsAndKeepsSources(t *testing.T) {
	f := openSample(t)
	dir := t.TempDir()
	a := dropFile(t, dir, "a.png", pngBytes("a"))
	b := dropFile(t, dir, "b.png", pngBytes("b"))

	res := newDropper(f).HandleDrop([]string{a, b}, slotAttrs())

	if len(res.Added) != 2 || len(res.Refused) != 0 {
		t.Fatalf("result = %+v", res)
	}
	for _, added := range res.Added {
		links := added.Item.Links
		if len(links) != 1 || links[0].Kind != model.EvidenceLinkSlot || links[0].TargetID != "web-shot" {
			t.Fatalf("links = %+v", links)
		}
	}
	for _, p := range []string{a, b} {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("source %s removed: %v", p, err)
		}
	}
}

func TestHandleDropParsesServiceTarget(t *testing.T) {
	f := openSample(t)
	p := dropFile(t, t.TempDir(), "a.png", pngBytes("a"))
	attrs := map[string]string{attrEvidenceHost: sampleHost, attrEvidenceTarget: "finding:f1:svc9"}

	res := newDropper(f).HandleDrop([]string{p}, attrs)

	want := model.EvidenceLink{Kind: model.EvidenceLinkFinding, TargetID: "f1", ServiceID: "svc9"}
	if len(res.Added) != 1 || res.Added[0].Item.Links[0] != want {
		t.Fatalf("result = %+v", res)
	}
}

func TestHandleDropHostOnlyHasNoLinks(t *testing.T) {
	f := openSample(t)
	p := dropFile(t, t.TempDir(), "a.png", pngBytes("a"))

	res := newDropper(f).HandleDrop([]string{p}, map[string]string{attrEvidenceHost: sampleHost})

	if len(res.Added) != 1 || len(res.Added[0].Item.Links) != 0 {
		t.Fatalf("result = %+v", res)
	}
}

func TestHandleDropRefusesUnsupportedType(t *testing.T) {
	f := openSample(t)
	dir := t.TempDir()
	zip := dropFile(t, dir, "x.zip", []byte("PK\x03\x04\x00\x00binary"))
	ok := dropFile(t, dir, "ok.png", pngBytes("ok"))

	res := newDropper(f).HandleDrop([]string{zip, ok}, slotAttrs())

	if len(res.Added) != 1 || len(res.Refused) != 1 {
		t.Fatalf("result = %+v", res)
	}
	if res.Refused[0].Path != zip || !strings.Contains(res.Refused[0].Reason, "unsupported evidence type") {
		t.Fatalf("refusal = %+v", res.Refused[0])
	}
}

func TestHandleDropReportsDuplicate(t *testing.T) {
	f := openSample(t)
	dir := t.TempDir()
	same := pngBytes("same")
	a := dropFile(t, dir, "a.png", same)
	b := dropFile(t, dir, "b.png", bytes.Clone(same))

	res := newDropper(f).HandleDrop([]string{a, b}, slotAttrs())

	if len(res.Added) != 2 || res.Added[0].Duplicate || !res.Added[1].Duplicate {
		t.Fatalf("result = %+v", res)
	}
}

func TestHandleDropWithoutTargetIsOneRefusal(t *testing.T) {
	f := openSample(t)
	d := newDropper(f)
	for name, attrs := range map[string]map[string]string{
		"nil":          nil,
		"empty":        {},
		"no host":      {attrEvidenceTarget: "slot:x"},
		"unknown kind": {attrEvidenceHost: sampleHost, attrEvidenceTarget: "bogus:x"},
		"bad target":   {attrEvidenceHost: sampleHost, attrEvidenceTarget: "slot"},
	} {
		t.Run(name, func(t *testing.T) {
			res := d.HandleDrop([]string{"/a", "/b"}, attrs)
			if len(res.Added) != 0 || len(res.Refused) != 1 || res.Preview != nil {
				t.Fatalf("result = %+v", res)
			}
		})
	}
	if res := d.HandleDrop([]string{"/a"}, nil); res.Refused[0].Reason != "drop landed outside a target" {
		t.Fatalf("reason = %q", res.Refused[0].Reason)
	}
}

func TestHandleDropImportPreviewsOnePath(t *testing.T) {
	f := openSample(t)
	abs, err := filepath.Abs(nmapFixture)
	if err != nil {
		t.Fatal(err)
	}
	attrs := map[string]string{attrEvidenceKind: dropKindImport}

	res := newDropper(f).HandleDrop([]string{abs}, attrs)

	if res.Preview == nil || res.Preview.Token == "" || len(res.Refused) != 0 {
		t.Fatalf("result = %+v", res)
	}
	two := newDropper(f).HandleDrop([]string{abs, abs}, attrs)
	if two.Preview != nil || len(two.Refused) != 1 {
		t.Fatalf("two paths = %+v", two)
	}
}

func TestHandleDropImportRefusalNamesReason(t *testing.T) {
	f := openSample(t)
	p := dropFile(t, t.TempDir(), "junk.xml", []byte("not a scan"))

	res := newDropper(f).HandleDrop([]string{p}, map[string]string{attrEvidenceKind: dropKindImport})

	if res.Preview != nil || len(res.Refused) != 1 || res.Refused[0].Path != p || res.Refused[0].Reason == "" {
		t.Fatalf("result = %+v", res)
	}
}

package importers

import (
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"testing"

	"github.com/unclesp1d3r/redtracks/internal/model"
	"github.com/unclesp1d3r/redtracks/internal/store"
)

const sampleIP = "10.10.10.5"

func openSample(t *testing.T) *store.Store {
	t.Helper()
	dir := t.TempDir()
	if err := os.CopyFS(dir, os.DirFS(filepath.Join("..", "store", "testdata", "pg-sample"))); err != nil {
		t.Fatalf("copy fixture: %v", err)
	}
	st, err := store.Open(dir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	return st
}

func snapshot(t *testing.T, st *store.Store) map[string]string {
	t.Helper()
	root := os.DirFS(st.Engagement().Dir)
	out := map[string]string{}
	err := fs.WalkDir(root, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, rerr := fs.ReadFile(root, p)
		out[p] = string(data)
		return rerr
	})
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	return out
}

func findRow(p Preview, kind string, badge Badge) (Row, bool) {
	for _, r := range p.Rows {
		if r.Kind == kind && r.Badge == badge {
			return r, true
		}
	}
	return Row{}, false
}

func requireRow(t *testing.T, p Preview, kind string, badge Badge) {
	t.Helper()
	if _, ok := findRow(p, kind, badge); !ok {
		t.Errorf("no %s row with badge %s in %+v", kind, badge, p.Rows)
	}
}

func sampleService(t *testing.T, st *store.Store, port int) model.Service {
	t.Helper()
	h, _ := st.Engagement().Host(sampleIP)
	for _, s := range h.Services {
		if s.Port == port {
			return s
		}
	}
	t.Fatalf("no service on port %d", port)
	return model.Service{}
}

func TestPreviewNewHost(t *testing.T) {
	st := openSample(t)
	res := Result{Tool: ToolNmap, Hosts: []HostResult{{
		IP: "10.10.10.99", Hostname: "fresh",
		Services: []ServiceResult{{Port: 22, Protocol: "tcp", Type: "ssh"}},
	}}}
	p := BuildPreview(st.Engagement(), res, nil)
	requireRow(t, p, "host", BadgeNew)
	requireRow(t, p, "service", BadgeNew)
	if len(p.Unresolved) != 0 || p.Tool != ToolNmap {
		t.Errorf("preview %+v", p)
	}
}

func TestPreviewFillsEmptyFields(t *testing.T) {
	st := openSample(t)
	if err := st.MutateHost(sampleIP, func(h *model.Host) error {
		h.Services[0].Version = ""
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	res := Result{Tool: ToolNmap, Hosts: []HostResult{{
		IP:       sampleIP,
		Services: []ServiceResult{{Port: 80, Protocol: "tcp", Type: "http-alt", Product: "other", Version: "2.4.18"}},
	}}}
	p := BuildPreview(st.Engagement(), res, nil)
	requireRow(t, p, "service", BadgeFills)
	if _, err := Apply(st, res, nil, []byte("<nmaprun/>")); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	s := sampleService(t, st, 80)
	if s.Type != "http" || s.Product != "Apache httpd" || s.Version != "2.4.18" {
		t.Errorf("service %+v, want type and product kept, version filled", s)
	}
}

func TestPreviewDuplicateWebPath(t *testing.T) {
	st := openSample(t)
	res := Result{Tool: ToolFfuf, Hosts: []HostResult{{
		IP: sampleIP, URL: "http://10.10.10.5",
		WebPaths: []WebPathResult{{Path: "/nibbleblog/", Status: 200}, {Path: "/admin", Status: 301}},
	}}}
	p := BuildPreview(st.Engagement(), res, nil)
	requireRow(t, p, "webpath", BadgeDuplicate)
	requireRow(t, p, "webpath", BadgeNew)
}

func webResult(url string) Result {
	return Result{Tool: ToolFfuf, Hosts: []HostResult{{URL: url, WebPaths: []WebPathResult{{Path: "/x", Status: 200}}}}}
}

func TestResolveURLByHostname(t *testing.T) {
	st := openSample(t)
	p := BuildPreview(st.Engagement(), webResult("http://nibbles/"), nil)
	if len(p.Unresolved) != 0 {
		t.Fatalf("unresolved %+v", p.Unresolved)
	}
	r, ok := findRow(p, "webpath", BadgeNew)
	if !ok || r.HostIP != sampleIP {
		t.Errorf("rows %+v, want webpath on %s", p.Rows, sampleIP)
	}
}

func TestUnresolvedWritesNothing(t *testing.T) {
	st := openSample(t)
	before := snapshot(t, st)
	p, err := Apply(st, webResult("http://unknown.local/"), nil, []byte(`{"x":1}`))
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if len(p.Unresolved) != 1 || p.Unresolved[0].Key != "unknown.local" {
		t.Fatalf("unresolved %+v", p.Unresolved)
	}
	if after := snapshot(t, st); !maps.Equal(before, after) {
		t.Errorf("files changed: before %d after %d", len(before), len(after))
	}
}

func TestOverrideResolves(t *testing.T) {
	st := openSample(t)
	res := webResult("http://unknown.local/")
	if _, err := Apply(st, res, map[string]string{"unknown.local": sampleIP}, []byte("{}")); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	s := sampleService(t, st, 80)
	found := false
	for _, w := range s.WebPaths {
		found = found || w.Path == "/x"
	}
	if !found {
		t.Errorf("web paths %+v, want /x on port 80", s.WebPaths)
	}
}

func TestGobusterResolvesByOverrideWithPort(t *testing.T) {
	st := openSample(t)
	res, err := Parse(readFixture(t, "gobuster-dir.txt"))
	if err != nil {
		t.Fatal(err)
	}
	before := snapshot(t, st)
	p, err := Apply(st, res, nil, []byte("raw"))
	if err != nil || len(p.Unresolved) != 1 || p.Unresolved[0].Key != ToolGobuster {
		t.Fatalf("no override: err=%v unresolved=%+v", err, p.Unresolved)
	}
	if !maps.Equal(before, snapshot(t, st)) {
		t.Fatal("unresolved gobuster wrote files")
	}
	if _, err := Apply(st, res, map[string]string{ToolGobuster: sampleIP + ":8080"}, []byte("raw")); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	s := sampleService(t, st, 8080)
	if s.Type != "http" || len(s.WebPaths) != len(res.Hosts[0].WebPaths) {
		t.Errorf("service %+v", s)
	}
}

func counts(st *store.Store) [3]int {
	e := st.Engagement()
	return [3]int{len(e.Findings), len(e.Creds.Users), len(e.Creds.Shares)}
}

func TestApplyDedupesFindingsAndUsers(t *testing.T) {
	st := openSample(t)
	nuclei, err := Parse(readFixture(t, "nuclei.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	enum, err := Parse(readFixture(t, "enum4linux-ng.json"))
	if err != nil {
		t.Fatal(err)
	}
	overrides := map[string]string{ToolEnum4linuxNG: sampleIP, "app.example.test": "10.10.10.7"}
	apply := func() {
		for _, r := range []Result{nuclei, enum} {
			if _, err := Apply(st, r, overrides, []byte("{}")); err != nil {
				t.Fatalf("Apply %s: %v", r.Tool, err)
			}
		}
	}
	apply()
	once := counts(st)
	if once[0] == 0 || once[1] == 0 {
		t.Fatalf("expected findings and users after first apply, got %v", once)
	}
	apply()
	if twice := counts(st); twice != once {
		t.Errorf("counts after second apply %v, want %v", twice, once)
	}
	for _, f := range st.Engagement().Findings {
		if f.Status != "noted" || f.Source != ToolNuclei || f.ServiceID == "" {
			t.Errorf("finding %+v", f)
		}
	}
}

func TestApplySavesRaw(t *testing.T) {
	st := openSample(t)
	dir := st.Engagement().Dir
	res := Result{Tool: ToolNmap, Hosts: []HostResult{{
		IP: sampleIP, Services: []ServiceResult{{Port: 22, Protocol: "tcp"}},
	}}}
	if _, err := Apply(st, res, nil, []byte("<nmaprun/>")); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	files, err := filepath.Glob(filepath.Join(dir, "hosts", sampleIP, "imports", "*"))
	if err != nil || len(files) != 1 {
		t.Errorf("import files %v err %v, want exactly one", files, err)
	}
}

func TestWriteTargetFallsBackWhenHostExists(t *testing.T) {
	st := openSample(t)
	svc := ServiceResult{Port: 8443, Protocol: "tcp", Type: "https"}
	tg := target{ip: sampleIP, host: HostResult{Services: []ServiceResult{svc}}}
	// exists=false is a stale snapshot: the host is already in the store.
	if err := writeTarget(st, tg, ToolNmap, false); err != nil {
		t.Fatalf("writeTarget: %v", err)
	}
	h, ok := st.Engagement().Host(sampleIP)
	if !ok {
		t.Fatal("host missing")
	}
	if serviceIndex(h.Services, 8443, "tcp") < 0 || serviceIndex(h.Services, 80, "tcp") < 0 {
		t.Fatalf("services = %+v, want merged 80 and 8443", h.Services)
	}
}

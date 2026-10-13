package importers

import (
	"errors"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"strings"
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

func findRow(p Preview, kind RowKind, badge Badge) (Row, bool) {
	for _, r := range p.Rows {
		if r.Kind == kind && r.Badge == badge {
			return r, true
		}
	}
	return Row{}, false
}

func requireRow(t *testing.T, p Preview, kind RowKind, badge Badge) {
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
	// enum4linux-ng's target is the DNS name dc01.corp.local; resolve it by hostname.
	err := st.AddHost(
		model.Host{ID: "dc", IP: "10.10.10.10", Hostname: "DC01.corp.local", Status: model.HostStatusUntouched},
	)
	if err != nil {
		t.Fatal(err)
	}
	nuclei, err := Parse(readFixture(t, "nuclei.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	enum, err := Parse(readFixture(t, "enum4linux-ng.json"))
	if err != nil {
		t.Fatal(err)
	}
	overrides := map[string]string{"app.example.test": "10.10.10.7"}
	apply := func() {
		for _, r := range []Result{nuclei, enum} {
			p, err := Apply(st, r, overrides, []byte("{}"))
			if err != nil || len(p.Unresolved) != 0 {
				t.Fatalf("Apply %s: err=%v unresolved=%+v", r.Tool, err, p.Unresolved)
			}
		}
	}
	apply()
	once := counts(st)
	if once[0] == 0 || once[1] == 0 {
		t.Fatalf("expected findings and users after first apply, got %v", once)
	}
	if h, _ := st.Engagement().Host("10.10.10.10"); h.IP == "" {
		t.Fatal("dc host missing")
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

func TestHostnameOnlyResultUsesOverrideKeyedByHostname(t *testing.T) {
	st := openSample(t)
	enum, err := Parse(readFixture(t, "enum4linux-ng.json"))
	if err != nil {
		t.Fatal(err)
	}
	p := BuildPreview(st.Engagement(), enum, nil)
	if len(p.Unresolved) != 1 || p.Unresolved[0].Key != "dc01.corp.local" {
		t.Fatalf("unresolved %+v, want key dc01.corp.local", p.Unresolved)
	}
	if _, err := Apply(st, enum, map[string]string{"dc01.corp.local": sampleIP}, []byte("{}")); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if len(st.Engagement().Creds.Users) == 0 {
		t.Fatal("override did not place the users")
	}
}

func TestMergeFillsServiceWithEmptyProtocol(t *testing.T) {
	h := newHost(sampleIP)
	h.Services = []model.Service{{ID: "s", Port: 80, Type: "http"}} // hand-edited, no protocol
	tg := target{ip: sampleIP, port: 80, host: HostResult{WebPaths: []WebPathResult{{Path: "/x", Status: 200}}}}
	got, _ := mergeHost(h, tg, ToolFfuf, false)
	if len(got.Services) != 1 || len(got.Services[0].WebPaths) != 1 {
		t.Fatalf("services = %+v, want the existing service to gain the path", got.Services)
	}
}

func TestNucleiIPHostKeepsSchemeAndPort(t *testing.T) {
	st := openSample(t)
	in := `{"template-id":"x","info":{"name":"Thing","severity":"low"},"host":"https://10.10.10.5:8443"}` + "\n"
	res, err := Parse([]byte(in))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(st, res, nil, []byte(in)); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if s := sampleService(t, st, 8443); s.Type != "https" {
		t.Errorf("service %+v, want https on 8443", s)
	}
	h, _ := st.Engagement().Host(sampleIP)
	if serviceIndex(h.Services, 80, "tcp") < 0 || len(h.Services) != 2 {
		t.Errorf("services %+v, want only the original 80 plus 8443", h.Services)
	}
}

func TestNucleiBareHostOn443IsHTTPS(t *testing.T) {
	hr, err := nucleiHost("10.10.10.5:443")
	if err != nil || hr.URL != "https://10.10.10.5:443" {
		t.Fatalf("nucleiHost = %+v, %v; want https", hr, err)
	}
	if hr, err = nucleiHost("10.10.10.5:8080"); err != nil || hr.URL != "http://10.10.10.5:8080" {
		t.Fatalf("nucleiHost = %+v, %v; want http", hr, err)
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

func TestPreviewCarriesDroppedAndSkipped(t *testing.T) {
	st := openSample(t)
	res := Result{Tool: ToolNmap, Dropped: 1, DroppedIPs: []string{"bogus"}, Skipped: 3}
	p := BuildPreview(st.Engagement(), res, nil)
	if len(p.Dropped) != 1 || p.Dropped[0] != "bogus" || p.Skipped != 3 {
		t.Fatalf("Dropped=%v Skipped=%d", p.Dropped, p.Skipped)
	}
	if empty := BuildPreview(st.Engagement(), Result{Tool: ToolNmap}, nil); empty.Dropped == nil {
		t.Fatal("Dropped should be an empty slice, not nil, so it serialises as []")
	}
}

func TestApplyMidImportFailureReportsProgress(t *testing.T) {
	st := openSample(t)
	badHost := filepath.Join(st.Engagement().Dir, "hosts", "10.10.10.9", "host.json")
	if err := os.MkdirAll(filepath.Dir(badHost), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(badHost, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	res := Result{Tool: ToolNmap, Hosts: []HostResult{
		{IP: sampleIP, Services: []ServiceResult{{Port: 22, Protocol: "tcp"}}},
		{IP: "10.10.10.9", Services: []ServiceResult{{Port: 22, Protocol: "tcp"}}},
	}}
	_, err := Apply(st, res, nil, []byte("raw"))
	if err == nil {
		t.Fatal("expected a failure on the corrupt host")
	}
	if sampleService(t, st, 22).Port != 22 {
		t.Error("host A was not written")
	}
	if got, rerr := os.ReadFile(badHost); rerr != nil || string(got) != "{" {
		t.Errorf("corrupt host B was touched: %q %v", got, rerr)
	}
	raws, gerr := filepath.Glob(filepath.Join(st.Engagement().Dir, "hosts", "*", "imports", "*"))
	if gerr != nil || len(raws) != 0 {
		t.Errorf("raw files %v saved after a partial failure (err %v)", raws, gerr)
	}
	if _, ok := errors.AsType[*store.ParseError](err); !ok {
		t.Errorf("err = %v, want the parse error to be wrapped", err)
	}
	for _, want := range []string{"10.10.10.9", "after writing 1 of 2", "retries safely"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q lacks %q", err, want)
		}
	}
}

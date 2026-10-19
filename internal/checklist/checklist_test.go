package checklist_test

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/unclesp1d3r/redtracks/checklists"
	"github.com/unclesp1d3r/redtracks/internal/checklist"
	"github.com/unclesp1d3r/redtracks/internal/model"
)

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

const httpDef = `{
  "id": "http",
  "title": "HTTP",
  "match": {"serviceTypes": ["http", "https"]},
  "items": [{"id": "http.robots", "title": "robots", "hint": "", "commands": ["curl $T"]}]
}`

const sshDef = `{
  "id": "ssh",
  "title": "SSH",
  "match": {"serviceTypes": ["ssh"]},
  "items": [{"id": "ssh.banner", "title": "banner", "hint": "", "commands": ["nc $T 22"]}]
}`

func TestLoadDefinitionsReadsAll(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "http.json", httpDef)
	writeFile(t, dir, "ssh.json", sshDef)

	defs, err := checklist.LoadDefinitions(dir)
	if err != nil {
		t.Fatalf("LoadDefinitions: %v", err)
	}
	if len(defs) != 2 {
		t.Fatalf("got %d definitions, want 2", len(defs))
	}
	for _, def := range defs {
		if def.File == "" {
			t.Errorf("definition %s has no File", def.ID)
		}
	}
}

func TestLoadDefinitionsDuplicateItemID(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.json", `{"id":"a","title":"A","match":{"serviceTypes":["a"]},
		"items":[{"id":"http.robots","title":"x","hint":"","commands":[]}]}`)
	writeFile(t, dir, "b.json", `{"id":"b","title":"B","match":{"serviceTypes":["b"]},
		"items":[{"id":"http.robots","title":"y","hint":"","commands":[]}]}`)

	_, err := checklist.LoadDefinitions(dir)
	if err == nil {
		t.Fatal("expected error for duplicate item id")
	}
	if !strings.Contains(err.Error(), "a.json") || !strings.Contains(err.Error(), "b.json") {
		t.Fatalf("error %q does not name both files", err.Error())
	}
}

func TestLoadDefinitionsInvalidJSONNamesFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "broken.json", "{")

	_, err := checklist.LoadDefinitions(dir)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "broken.json") {
		t.Fatalf("error %q does not name the broken file", err.Error())
	}
}

func TestForServiceMatchesByType(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "http.json", httpDef)
	writeFile(t, dir, "mcp.json", `{"id":"mcp-server","title":"MCP",
		"match":{"serviceTypes":["mcp-server"]},
		"items":[{"id":"mcp.tools","title":"tools","hint":"","commands":[]}]}`)
	defs, err := checklist.LoadDefinitions(dir)
	if err != nil {
		t.Fatalf("LoadDefinitions: %v", err)
	}

	matches := checklist.ForService(defs, model.Service{Type: "https"})
	if len(matches) != 1 || matches[0].ID != "http" {
		t.Fatalf("https matched %v, want only http", matches)
	}

	matches = checklist.ForService(defs, model.Service{Type: "mcp-server"})
	if len(matches) != 1 || matches[0].ID != "mcp-server" {
		t.Fatalf("mcp-server matched %v, want only mcp-server", matches)
	}
}

func TestForHostMatchesByOS(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "linux-privesc.json", `{"id":"linux-privesc","title":"Linux",
		"match":{"hostOS":["linux"]},
		"items":[{"id":"linux.sudo","title":"sudo -l","hint":"","commands":["sudo -l"]}]}`)
	writeFile(t, dir, "windows-privesc.json", `{"id":"windows-privesc","title":"Windows",
		"match":{"hostOS":["windows"]},
		"items":[{"id":"windows.whoami","title":"whoami","hint":"","commands":["whoami /priv"]}]}`)
	defs, err := checklist.LoadDefinitions(dir)
	if err != nil {
		t.Fatalf("LoadDefinitions: %v", err)
	}

	matches := checklist.ForHost(defs, model.Host{OS: "Linux"})
	if len(matches) != 1 || matches[0].ID != "linux-privesc" {
		t.Fatalf("linux host matched %v, want only linux-privesc", matches)
	}
}

func TestMergeDefaultsToTodo(t *testing.T) {
	def := checklist.Definition{
		ID: "x",
		Items: []checklist.Item{
			{ID: "x.a", Title: "A"},
			{ID: "x.b", Title: "B"},
			{ID: "x.c", Title: "C"},
		},
	}
	state := model.ChecklistState{Items: map[string]model.ItemState{
		"x.a": {Status: model.ItemDone},
	}}

	views := checklist.Merge(def, state)
	if len(views) != 3 {
		t.Fatalf("got %d views, want 3", len(views))
	}
	todoCount := 0
	for _, v := range views {
		if v.State.Status == model.ItemTodo {
			todoCount++
		}
	}
	if todoCount != 2 {
		t.Fatalf("got %d todo views, want 2", todoCount)
	}
}

func TestMergeIgnoresOrphanState(t *testing.T) {
	def := checklist.Definition{
		ID:    "x",
		Items: []checklist.Item{{ID: "x.a", Title: "A"}},
	}
	state := model.ChecklistState{Items: map[string]model.ItemState{
		"x.a":        {Status: model.ItemDone},
		"x.orphaned": {Status: model.ItemDone},
	}}

	views := checklist.Merge(def, state)
	if len(views) != len(def.Items) {
		t.Fatalf("got %d views, want %d", len(views), len(def.Items))
	}
}

func TestSubstitute(t *testing.T) {
	got, err := checklist.Substitute("curl http://$T/$D", "10.0.0.1", "corp.local")
	if err != nil {
		t.Fatalf("Substitute: %v", err)
	}
	want := "curl http://10.0.0.1/corp.local"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestSubstituteRejectsShellMetacharacters(t *testing.T) {
	cases := []struct {
		name   string
		ip     string
		domain string
	}{
		{"domain metacharacters", "10.0.0.1", "corp.local;curl x|sh"},
		{"domain newline", "10.0.0.1", "corp.local\nrm -rf /"},
		{"ip metacharacters", "10.0.0.1;id", "corp.local"},
		{"ip zone", "::1%x;id", "corp.local"},
		{"domain leading dash", "10.0.0.1", "-evil.local"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := checklist.Substitute("curl $T/$D", tc.ip, tc.domain); err == nil {
				t.Fatalf("expected error for ip=%q domain=%q", tc.ip, tc.domain)
			}
		})
	}
}

func TestWatchReloadsOnChange(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "http.json", httpDef)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	changes := make(chan []checklist.Definition, 16)
	errs := make(chan error, 16)
	go checklist.Watch(ctx, dir, 20*time.Millisecond,
		func(defs []checklist.Definition) { changes <- defs },
		func(err error) { errs <- err },
	)

	waitForCount(t, changes, 1, 500*time.Millisecond)

	writeFile(t, dir, "ssh.json", sshDef)
	last := waitForCount(t, changes, 2, 500*time.Millisecond)
	if len(last) != 2 {
		t.Fatalf("got %d definitions, want 2", len(last))
	}

	writeFile(t, dir, "ssh.json", "{")
	select {
	case err := <-errs:
		if err == nil {
			t.Fatal("expected non-nil error")
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("onError did not fire within 500ms")
	}

	select {
	case defs := <-changes:
		if len(defs) != 2 {
			t.Fatalf("onChange fired with %d definitions after a load error, want 2", len(defs))
		}
		t.Fatal("onChange should not fire again while the file is broken")
	case <-time.After(100 * time.Millisecond):
		// expected: no further onChange call
	}
}

// waitForCount drains changes until it has one with the given length, or
// fails the test after timeout.
func waitForCount(
	t *testing.T, changes <-chan []checklist.Definition, n int, timeout time.Duration,
) []checklist.Definition {
	t.Helper()
	deadline := time.After(timeout)
	for {
		select {
		case defs := <-changes:
			if len(defs) == n {
				return defs
			}
		case <-deadline:
			t.Fatalf("did not observe %d definitions within %s", n, timeout)
		}
	}
}

// shippedCount is the number of definitions embedded in checklists.FS.
func shippedCount(t *testing.T) int {
	t.Helper()
	files, err := fs.Glob(checklists.FS, "*.json")
	if err != nil {
		t.Fatal(err)
	}
	return len(files)
}

func TestInstallDefaultsOnlyOnce(t *testing.T) {
	dst := filepath.Join(t.TempDir(), "checklists")

	added, err := checklist.InstallDefaults(checklists.FS, dst)
	if err != nil {
		t.Fatalf("InstallDefaults: %v", err)
	}
	if len(added) == 0 {
		t.Fatal("expected files added on first call")
	}

	entries, err := os.ReadDir(dst)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	jsonCount := 0
	hasReadme := false
	for _, e := range entries {
		if e.Name() == "_readme.md" {
			hasReadme = true
		}
		if strings.HasSuffix(e.Name(), ".json") {
			jsonCount++
		}
	}
	if want := shippedCount(t); jsonCount != want {
		t.Fatalf("got %d json files, want %d", jsonCount, want)
	}
	if !hasReadme {
		t.Fatal("missing _readme.md")
	}

	modified := filepath.Join(dst, "http.json")
	if err := os.WriteFile(modified, []byte("modified"), 0o600); err != nil {
		t.Fatalf("modify: %v", err)
	}

	added, err = checklist.InstallDefaults(checklists.FS, dst)
	if err != nil {
		t.Fatalf("InstallDefaults second call: %v", err)
	}
	if len(added) != 0 {
		t.Fatalf("second call added %v, want nothing", added)
	}
	data, err := os.ReadFile(modified)
	if err != nil {
		t.Fatalf("read modified: %v", err)
	}
	if string(data) != "modified" {
		t.Fatal("second InstallDefaults overwrote an operator edit")
	}
}

func TestShippedSetLoads(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "checklists")
	if _, err := checklist.InstallDefaults(checklists.FS, dir); err != nil {
		t.Fatalf("InstallDefaults: %v", err)
	}

	defs, err := checklist.LoadDefinitions(dir)
	if err != nil {
		t.Fatalf("LoadDefinitions: %v", err)
	}
	if want := shippedCount(t); len(defs) != want {
		t.Fatalf("got %d definitions, want %d", len(defs), want)
	}

	ids := make([]string, 0, len(defs))
	for _, def := range defs {
		ids = append(ids, def.ID)
		if !def.HasItems() {
			continue // slots-only proof definitions have no checklist to size
		}
		if len(def.Items) < 5 || len(def.Items) > 10 {
			t.Errorf("definition %s has %d items, want 5-10", def.ID, len(def.Items))
		}
		for _, item := range def.Items {
			prefix := def.ID + "."
			if !strings.HasPrefix(item.ID, prefix) {
				t.Errorf("item %s in %s lacks prefix %q", item.ID, def.ID, prefix)
			}
		}
	}

	sort.Strings(ids)
	want := []string{
		"exam-proofs", "http", "kerberos", "ldap", "linux-privesc", "llm-chat",
		"mcp-server", "rag-api", "smb", "ssh", "windows-privesc",
	}
	sort.Strings(want)
	if strings.Join(ids, ",") != strings.Join(want, ",") {
		t.Fatalf("got ids %v, want %v", ids, want)
	}
}

func TestLoadDefinitionsRejectsBadDefinitions(t *testing.T) {
	cases := map[string]struct {
		files map[string]string
		want  string
	}{
		"empty definition id": {
			map[string]string{"x.json": `{"id":"","match":{"serviceTypes":["a"]},"items":[]}`}, "x.json",
		},
		"duplicate definition id": {
			map[string]string{
				"a.json": `{"id":"d","match":{"serviceTypes":["a"]},"items":[]}`,
				"b.json": `{"id":"d","match":{"serviceTypes":["b"]},"items":[]}`,
			}, "b.json",
		},
		"empty item id": {
			map[string]string{
				"x.json": `{"id":"x","match":{"hostOS":["linux"]},"items":[{"id":"","title":"t"}]}`,
			}, "x.json",
		},
		"empty match": {
			map[string]string{"x.json": `{"id":"x","match":{"serviceTypes":[],"hostOS":[]},"items":[]}`}, "x.json",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			for f, c := range tc.files {
				writeFile(t, dir, f, c)
			}
			_, err := checklist.LoadDefinitions(dir)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want one naming %s", err, tc.want)
			}
		})
	}
}

func TestForServiceIsCaseInsensitive(t *testing.T) {
	defs := []checklist.Definition{{ID: "http", Match: checklist.Match{ServiceTypes: []string{"HTTP"}}}}
	if got := checklist.ForService(defs, model.Service{Type: "http"}); len(got) != 1 {
		t.Fatalf("matched %v, want http", got)
	}
}

func TestSubstituteRequiresDomainForDollarD(t *testing.T) {
	if _, err := checklist.Substitute("kerbrute -d $D $T", "10.0.0.1", ""); err == nil {
		t.Fatal("expected an error for $D without a domain")
	}
	if got, err := checklist.Substitute("nmap $T", "10.0.0.1", ""); err != nil || got != "nmap 10.0.0.1" {
		t.Fatalf("got %q, %v", got, err)
	}
}

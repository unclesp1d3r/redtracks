package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/unclesp1d3r/redtracks/internal/model"
)

func TestRootDirHonoursEnv(t *testing.T) {
	t.Setenv("REDTRACKS_HOME", "/tmp/x")
	if got := RootDir(); got != "/tmp/x" {
		t.Fatalf("RootDir = %q, want /tmp/x", got)
	}
}

func TestRootDirDefaultsToHome(t *testing.T) {
	t.Setenv("REDTRACKS_HOME", "")
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home dir")
	}
	if got, want := RootDir(), filepath.Join(home, "RedTracks"); got != want {
		t.Fatalf("RootDir = %q, want %q", got, want)
	}
}

func TestCreateEngagementWritesLayout(t *testing.T) {
	root := t.TempDir()
	dir, err := CreateEngagement(root, "My Lab #1", "corp.local", model.EngagementOSAI)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(root, "engagements", "my-lab-1"); dir != want {
		t.Fatalf("dir = %q, want %q", dir, want)
	}
	for _, p := range []string{"notes.md", "hosts"} {
		if _, err := os.Stat(filepath.Join(dir, p)); err != nil {
			t.Errorf("missing %s: %v", p, err)
		}
	}
	for _, p := range []string{".", "hosts"} {
		info, err := os.Stat(filepath.Join(dir, p))
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != dirMode || dirMode != 0o700 {
			t.Errorf("%s mode %v, want 0700", p, info.Mode().Perm())
		}
	}
	var meta model.EngagementMeta
	if err := json.Unmarshal([]byte(readFile(t, filepath.Join(dir, "engagement.json"))), &meta); err != nil {
		t.Fatal(err)
	}
	if meta.Domain != "corp.local" || meta.Name != "My Lab #1" || meta.Type != model.EngagementOSAI || meta.ID == "" {
		t.Fatalf("meta = %+v", meta)
	}
	var points []int
	for _, f := range meta.FlagTypes {
		points = append(points, f.Points)
	}
	if len(points) != 3 || points[0] != 15 || points[1] != 10 || points[2] != 5 {
		t.Fatalf("points = %v, want [15 10 5]", points)
	}
	if _, err := CreateEngagement(root, "my lab 1", "", model.EngagementLab); err == nil {
		t.Fatal("duplicate slug accepted")
	}
}

func TestCreateEngagementRejectsBadNames(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"", "../x", "a/b", `a\b`, "a..b", "!!!"} {
		if _, err := CreateEngagement(root, name, "", model.EngagementLab); err == nil {
			t.Errorf("name %q accepted", name)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "engagements")); err == nil {
		entries, rerr := os.ReadDir(filepath.Join(root, "engagements"))
		if rerr != nil || len(entries) != 0 {
			t.Fatalf("created %d entries", len(entries))
		}
	}
}

func TestListEngagementsSkipsNonEngagementDirs(t *testing.T) {
	root := t.TempDir()
	if got, err := ListEngagements(root); err != nil || len(got) != 0 {
		t.Fatalf("missing dir: got %v, %v", got, err)
	}
	older, err := CreateEngagement(root, "older", "", model.EngagementLab)
	if err != nil {
		t.Fatal(err)
	}
	newer, err := CreateEngagement(root, "newer", "", model.EngagementOSCP)
	if err != nil {
		t.Fatal(err)
	}
	mustWriteAll(t, filepath.Join(root, "engagements", "stray", "x.txt"), "x")
	// Backdate "older" so the sort order is deterministic.
	path := filepath.Join(older, "engagement.json")
	var meta model.EngagementMeta
	if err := json.Unmarshal([]byte(readFile(t, path)), &meta); err != nil {
		t.Fatal(err)
	}
	meta.CreatedAt = meta.CreatedAt.AddDate(0, 0, -1)
	data, err := json.Marshal(meta)
	if err != nil {
		t.Fatal(err)
	}
	mustWrite(t, path, string(data))
	mustWriteAll(t, filepath.Join(newer, "hosts", "10.0.0.1", "host.json"), "{}")

	got, err := ListEngagements(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Name != "newer" || got[1].Name != "older" {
		t.Fatalf("got %+v", got)
	}
	if got[0].Dir != newer || got[0].Hosts != 1 || got[0].Type != model.EngagementOSCP {
		t.Fatalf("summary = %+v", got[0])
	}
}

func mustWriteAll(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), dirMode); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, path, content)
}

package checklist_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/unclesp1d3r/redtracks/checklists"
	"github.com/unclesp1d3r/redtracks/internal/checklist"
	"github.com/unclesp1d3r/redtracks/internal/model"
)

const proofsDef = `{
  "id": "proofs",
  "title": "Proofs",
  "match": {"hostOS": ["*"]},
  "slots": [{"id": "proofs.local", "title": "local.txt", "hint": "one frame"}]
}`

const httpSlotDef = `{
  "id": "httpslots",
  "title": "HTTP proofs",
  "match": {"serviceTypes": ["http"]},
  "slots": [{"id": "httpslots.shell", "title": "web shell", "hint": ""}]
}`

func loadDefs(t *testing.T, files map[string]string) []checklist.Definition {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		writeFile(t, dir, name, content)
	}
	defs, err := checklist.LoadDefinitions(dir)
	if err != nil {
		t.Fatalf("LoadDefinitions: %v", err)
	}
	return defs
}

func slotIDs(owed []checklist.OwedSlot) []string {
	ids := make([]string, len(owed))
	for i, o := range owed {
		ids[i] = o.ID
	}
	return ids
}

func TestSlotsDefinitionAndLegacyFileBothLoad(t *testing.T) {
	defs := loadDefs(t, map[string]string{"proofs.json": proofsDef, "http.json": httpDef})

	if len(defs) != 2 {
		t.Fatalf("got %d definitions, want 2", len(defs))
	}
	if len(defs[1].Slots) != 1 || defs[1].Slots[0].Hint != "one frame" || defs[1].HasItems() {
		t.Fatalf("proofs def = %+v, want one slot and no items", defs[1])
	}
	if !defs[0].HasItems() || len(defs[0].Slots) != 0 {
		t.Fatalf("legacy def = %+v, want items and no slots", defs[0])
	}
}

func TestSlotIDCollisionNamesBothFiles(t *testing.T) {
	cases := map[string]string{
		"item":  `{"id":"b","title":"B","match":{"hostOS":["*"]},"slots":[{"id":"http.robots","title":"x"}]}`,
		"slot":  `{"id":"b","title":"B","match":{"hostOS":["*"]},"slots":[{"id":"proofs.local","title":"x"}]}`,
		"empty": `{"id":"b","title":"B","match":{"hostOS":["*"]},"slots":[{"id":"","title":"x"}]}`,
	}
	for name, collide := range cases {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, dir, "a.json", httpDef)
			writeFile(t, dir, "a2.json", proofsDef)
			writeFile(t, dir, "b.json", collide)

			_, err := checklist.LoadDefinitions(dir)
			if err == nil {
				t.Fatal("want an error")
			}
			if name == "empty" {
				if !strings.Contains(err.Error(), "b.json") {
					t.Fatalf("err = %v, want b.json named", err)
				}
				return
			}
			prior := map[string]string{"item": "a.json", "slot": "a2.json"}[name]
			if !strings.Contains(err.Error(), prior) || !strings.Contains(err.Error(), "b.json") {
				t.Fatalf("err = %v, want %s and b.json named", err, prior)
			}
		})
	}
}

func TestSlotsForServiceOwedOncePerMatchingService(t *testing.T) {
	defs := loadDefs(t, map[string]string{"httpslots.json": httpSlotDef, "proofs.json": proofsDef})
	svcA := model.Service{ID: "svc-a", Type: "http"}
	svcB := model.Service{ID: "svc-b", Type: "http"}
	items := []model.EvidenceItem{{ID: "e1", Links: []model.EvidenceLink{
		{Kind: model.EvidenceLinkSlot, TargetID: "httpslots.shell", ServiceID: "svc-a"},
	}}}

	for _, tc := range []struct {
		svc    model.Service
		filled bool
	}{{svcA, true}, {svcB, false}} {
		owed := checklist.SlotsForService(defs, tc.svc)
		if want := []string{
			"httpslots.shell",
		}; !slices.Equal(slotIDs(owed), want) ||
			owed[0].DefinitionID != "httpslots" {
			t.Fatalf("service %s owes %+v, want %v", tc.svc.ID, owed, want)
		}
		if got := checklist.FilledSlots(items, tc.svc.ID)["httpslots.shell"]; got != tc.filled {
			t.Fatalf("service %s filled = %v, want %v", tc.svc.ID, got, tc.filled)
		}
	}
}

func TestHostLevelSlotFilledRegardlessOfServiceID(t *testing.T) {
	defs := loadDefs(t, map[string]string{"proofs.json": proofsDef})
	owed := checklist.SlotsForHost(defs, model.Host{OS: "linux"})
	if want := []string{"proofs.local"}; !slices.Equal(slotIDs(owed), want) {
		t.Fatalf("owed = %v, want %v", slotIDs(owed), want)
	}
	items := []model.EvidenceItem{{Links: []model.EvidenceLink{
		{Kind: model.EvidenceLinkSlot, TargetID: "proofs.local", ServiceID: "svc-a"},
		{Kind: model.EvidenceLinkFinding, TargetID: "proofs.other"},
	}}}

	filled := checklist.FilledSlots(items, "")
	if !filled["proofs.local"] || filled["proofs.other"] {
		t.Fatalf("filled = %v, want only the slot link", filled)
	}
}

func TestWildcardMatchesEmptyOSButLinuxOnlyDoesNot(t *testing.T) {
	defs := loadDefs(t, map[string]string{"proofs.json": proofsDef, "linux.json": `{
  "id": "linux", "title": "L", "match": {"hostOS": ["linux"]},
  "slots": [{"id": "linux.s", "title": "s"}]}`})

	got := slotIDs(checklist.SlotsForHost(defs, model.Host{OS: ""}))
	if want := []string{"proofs.local"}; !slices.Equal(got, want) {
		t.Fatalf("empty-OS host owes %v, want %v", got, want)
	}
	if n := len(checklist.ForHost(defs, model.Host{OS: "windows"})); n != 1 {
		t.Fatalf("windows host matched %d definitions, want just the wildcard one", n)
	}
}

func TestShippedExamProofsDeclareHostSlots(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "checklists")
	if _, err := checklist.InstallDefaults(checklists.FS, dir); err != nil {
		t.Fatal(err)
	}
	defs, err := checklist.LoadDefinitions(dir)
	if err != nil {
		t.Fatalf("LoadDefinitions: %v", err)
	}
	got := slotIDs(checklist.SlotsForHost(defs, model.Host{}))
	if len(got) != 4 {
		t.Fatalf("shipped host slots = %v, want 4", got)
	}
	if _, err := fs.Stat(checklists.FS, "exam-proofs.json"); err != nil {
		t.Fatal(err)
	}
}

// seedSliceOne installs every bundled file except name into dir.
func seedSliceOne(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.CopyFS(dir, checklists.FS); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, name)); err != nil {
		t.Fatal(err)
	}
}

func TestInstallAddsOnlyMissingBundledFileOnce(t *testing.T) {
	dst := filepath.Join(t.TempDir(), "checklists")
	seedSliceOne(t, dst, "exam-proofs.json")
	writeFile(t, dst, "http.json", "operator edit")

	added, err := checklist.InstallDefaults(checklists.FS, dst)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"exam-proofs.json"}; !slices.Equal(added, want) {
		t.Fatalf("added = %v, want %v", added, want)
	}
	data, err := os.ReadFile(filepath.Join(dst, "http.json"))
	if err != nil || string(data) != "operator edit" {
		t.Fatalf("present file = %q, %v; want the operator edit kept", data, err)
	}
	again, err := checklist.InstallDefaults(checklists.FS, dst)
	if err != nil || len(again) != 0 {
		t.Fatalf("second run = %v, %v; want nothing added", again, err)
	}
}

func TestInstallIntoEmptyRootAddsEverything(t *testing.T) {
	dst := filepath.Join(t.TempDir(), "checklists")

	added, err := checklist.InstallDefaults(checklists.FS, dst)
	if err != nil {
		t.Fatal(err)
	}
	if want := shippedCount(t) + 1; len(added) != want {
		t.Fatalf("added %d files, want %d (every json plus the readme)", len(added), want)
	}
}

func TestInstallRespectsDeletionButAddsNewBundledName(t *testing.T) {
	dst := filepath.Join(t.TempDir(), "checklists")
	first, err := checklist.InstallDefaults(checklists.FS, dst)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dst, "exam-proofs.json")); err != nil {
		t.Fatal(err)
	}
	added, err := checklist.InstallDefaults(checklists.FS, dst)
	if err != nil || len(added) != 0 {
		t.Fatalf("after deletion added = %v, %v; want nothing", added, err)
	}

	// A name the manifest has never seen is still added: rewrite the manifest
	// from the first install's names with ssh.json dropped.
	without := slices.DeleteFunc(slices.Clone(first), func(n string) bool { return n == "ssh.json" })
	if len(without) != len(first)-1 {
		t.Fatalf("first install added %v, want ssh.json among them", first)
	}
	writeFile(t, dst, ".installed", strings.Join(without, "\n")+"\n")
	if err := os.Remove(filepath.Join(dst, "ssh.json")); err != nil {
		t.Fatal(err)
	}
	added, err = checklist.InstallDefaults(checklists.FS, dst)
	if err != nil || !slices.Equal(added, []string{"ssh.json"}) {
		t.Fatalf("added = %v, %v; want ssh.json", added, err)
	}
}

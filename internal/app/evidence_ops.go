package app

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/netip"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/unclesp1d3r/redtracks/internal/checklist"
	"github.com/unclesp1d3r/redtracks/internal/evidence"
	"github.com/unclesp1d3r/redtracks/internal/model"
	"github.com/unclesp1d3r/redtracks/internal/store"
)

// ErrNoReveal is returned by Reveal when this build has no file manager to open.
var ErrNoReveal = errors.New("no file manager in this build; copy the path instead")

var (
	errItemNotFound   = errors.New("evidence item not found")
	errBadEvidenceRef = errors.New("invalid evidence file name")
	errAlreadyIndexed = errors.New("evidence file is already indexed")
	errEvidenceEscape = errors.New("evidence path is outside the engagement")
)

// runReveal starts the file manager; a variable so tests never launch one.
var runReveal = func(args []string) error {
	// A fixed argument list plus a path validated under the engagement; never a shell string.
	name, rest := args[0], args[1:]
	cmd := exec.Command(name, rest...)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start %s: %w", args[0], err)
	}
	go func() { _ = cmd.Wait() }() //nolint:errcheck // reaps the child; its exit status is irrelevant
	return nil
}

// AddResult is a newly stored evidence item; Duplicate says another item on the host has identical bytes.
type AddResult struct {
	Item      model.EvidenceItem
	Duplicate bool
}

// EvidenceView is one host's evidence with derived states.
type EvidenceView = evidence.View

// SearchHit is one evidence search match.
type SearchHit = evidence.Hit

// HostEvidence is one host's evidence view.
type HostEvidence struct {
	IP   string
	View EvidenceView
}

// Capabilities returns the desktop integrations main.go wired in.
func (s *EvidenceService) Capabilities() Capabilities { return s.app.Hooks().Capabilities }

// AddFromData stores standard-base64 file contents (paste or HTML5 drop) as
// new evidence on ip. lastModifiedMillis of zero means now. name only labels
// errors; it is not used as a caption.
func (s *EvidenceService) AddFromData(
	ip, name, dataBase64 string, lastModifiedMillis int64, caption string, links []model.EvidenceLink,
) (AddResult, error) {
	var res AddResult
	err := s.app.withStore("EvidenceService.AddFromData", func(st *store.Store) error {
		if err := validateLinks(links); err != nil {
			return err
		}
		raw, err := decodeCapped(name, dataBase64, maxImportBytes)
		if err != nil {
			return err
		}
		added, err := st.AddEvidenceFile(ip, bytes.NewReader(raw), capturedAt(lastModifiedMillis), caption, links)
		if err != nil {
			return fmt.Errorf("add %s: %w", name, err)
		}
		res = AddResult{Item: added.Item, Duplicate: added.Duplicate}
		return nil
	})
	return res, err
}

func capturedAt(millis int64) time.Time {
	now := time.Now().UTC()
	if millis == 0 {
		return now
	}
	if at := time.UnixMilli(millis).UTC(); at.Before(now) {
		return at
	}
	return now
}

func validateLink(l model.EvidenceLink) error {
	if !l.Kind.IsValid() {
		return fmt.Errorf("invalid evidence link kind %q", l.Kind)
	}
	if l.TargetID == "" {
		return errors.New("evidence link needs a target id")
	}
	return nil
}

func validateLinks(links []model.EvidenceLink) error {
	for _, l := range links {
		if err := validateLink(l); err != nil {
			return err
		}
	}
	return nil
}

// findItem returns the index of the item with id.
func findItem(f *model.EvidenceFile, id string) (int, error) {
	i := slices.IndexFunc(f.Items, func(it model.EvidenceItem) bool { return it.ID == id })
	if i < 0 {
		return 0, fmt.Errorf("%w: %q", errItemNotFound, id)
	}
	return i, nil
}

// mutateItem applies fn to the item with id on ip, resolving the id inside the store's closure.
func (s *EvidenceService) mutateItem(method, ip, id string, fn func(*model.EvidenceItem) error) error {
	return s.app.withStore(method, func(st *store.Store) error {
		return st.MutateEvidence(ip, func(f *model.EvidenceFile) error {
			i, err := findItem(f, id)
			if err != nil {
				return err
			}
			return fn(&f.Items[i])
		})
	})
}

// Link adds link to the item; an identical link already present is a no-op.
// The target is not checked: links are never enforced.
func (s *EvidenceService) Link(ip, itemID string, link model.EvidenceLink) error {
	if err := validateLink(link); err != nil {
		return s.app.fail("EvidenceService.Link", err)
	}
	return s.mutateItem("EvidenceService.Link", ip, itemID, func(it *model.EvidenceItem) error {
		if !slices.Contains(it.Links, link) {
			it.Links = append(slices.Clone(it.Links), link)
		}
		return nil
	})
}

// Unlink removes the link that matches kind, target and service exactly; the item stays.
func (s *EvidenceService) Unlink(ip, itemID string, link model.EvidenceLink) error {
	return s.mutateItem("EvidenceService.Unlink", ip, itemID, func(it *model.EvidenceItem) error {
		it.Links = slices.DeleteFunc(slices.Clone(it.Links), func(l model.EvidenceLink) bool { return l == link })
		return nil
	})
}

// SetCaption replaces the item's caption.
func (s *EvidenceService) SetCaption(ip, itemID, caption string) error {
	return s.mutateItem("EvidenceService.SetCaption", ip, itemID, func(it *model.EvidenceItem) error {
		it.Caption = caption
		return nil
	})
}

// Drop removes the item's index entry only; the file is never touched. It is
// how an entry whose file is gone is cleared.
func (s *EvidenceService) Drop(ip, itemID string) error {
	return s.app.withStore("EvidenceService.Drop", func(st *store.Store) error {
		return st.MutateEvidence(ip, func(f *model.EvidenceFile) error {
			i, err := findItem(f, itemID)
			if err != nil {
				return err
			}
			f.Items = slices.Delete(slices.Clone(f.Items), i, i+1)
			return nil
		})
	})
}

// evidenceDir returns hosts/<dir>/evidence under engagementDir for a plain
// IP. IPv6 colons become underscores, the same rule as the store's host
// directory names.
func evidenceDir(engagementDir, ip string) (string, error) {
	addr, err := netip.ParseAddr(ip)
	if err != nil || addr.Zone() != "" {
		return "", fmt.Errorf("invalid host ip %q", ip)
	}
	name := strings.ReplaceAll(addr.String(), ":", "_")
	return filepath.Join(engagementDir, "hosts", name, "evidence"), nil
}

// evidenceFilePath returns the absolute path of the evidence file name on ip,
// refusing anything but a plain name that stays under the engagement.
func evidenceFilePath(engagementDir, ip, name string) (string, error) {
	if name == "" || name != filepath.Base(name) || name == ".." || strings.HasPrefix(name, ".") {
		return "", fmt.Errorf("%w %q", errBadEvidenceRef, name)
	}
	dir, err := evidenceDir(engagementDir, ip)
	if err != nil {
		return "", err
	}
	path, err := filepath.Abs(filepath.Join(dir, name))
	if err != nil {
		return "", fmt.Errorf("resolve %s: %w", name, err)
	}
	root, err := filepath.Abs(engagementDir)
	if err != nil {
		return "", fmt.Errorf("resolve %s: %w", engagementDir, err)
	}
	if !evidence.UnderRoot(root, path) {
		return "", fmt.Errorf("%w: %s", errEvidenceEscape, path)
	}
	return path, nil
}

// adoptable describes a file that Adopt may index.
type adoptable struct {
	kind   model.EvidenceKind
	mtime  time.Time
	digest string
}

// inspectFile checks that path is a small regular file of an accepted type and hashes it.
func inspectFile(path string) (adoptable, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return adoptable{}, fmt.Errorf("stat %s: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		return adoptable{}, fmt.Errorf("%s is not a regular file", path)
	}
	f, err := os.Open(path)
	if err != nil {
		return adoptable{}, fmt.Errorf("open %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	data, err := io.ReadAll(io.LimitReader(f, maxImportBytes+1))
	if err != nil {
		return adoptable{}, fmt.Errorf("read %s: %w", path, err)
	}
	if len(data) > maxImportBytes {
		return adoptable{}, fmt.Errorf("%s is larger than %d MB", filepath.Base(path), maxImportMB)
	}
	sniffed, err := evidence.Sniff(data[:min(len(data), evidence.SniffLen)])
	if err != nil {
		return adoptable{}, err
	}
	sum := sha256.Sum256(data)
	return adoptable{
		kind:   sniffed.Kind,
		mtime:  minTime(info.ModTime().UTC(), time.Now().UTC()),
		digest: hex.EncodeToString(sum[:]),
	}, nil
}

func minTime(a, b time.Time) time.Time {
	if a.After(b) {
		return b
	}
	return a
}

// Adopt indexes a file that sits in the host's evidence directory with no
// entry. The file is left as it is (not stripped or renamed); its mtime
// becomes the capture time and it starts with no links.
func (s *EvidenceService) Adopt(ip, fileName string) (model.EvidenceItem, error) {
	var item model.EvidenceItem
	err := s.app.withStore("EvidenceService.Adopt", func(st *store.Store) error {
		path, err := evidenceFilePath(st.Engagement().Dir, ip, fileName)
		if err != nil {
			return err
		}
		info, err := inspectFile(path)
		if err != nil {
			return err
		}
		item = model.EvidenceItem{
			ID: model.NewID(), File: fileName, Kind: info.kind, CapturedAt: info.mtime, SHA256: info.digest,
			Links: []model.EvidenceLink{},
		}
		return st.MutateEvidence(ip, func(f *model.EvidenceFile) error {
			if slices.ContainsFunc(f.Items, func(it model.EvidenceItem) bool { return it.File == fileName }) {
				return fmt.Errorf("%w: %s", errAlreadyIndexed, fileName)
			}
			f.Items = append(slices.Clone(f.Items), item)
			return nil
		})
	})
	return item, err
}

// slotInfos lists every slot h and its services define.
func slotInfos(defs []checklist.Definition, h model.Host) []evidence.SlotInfo {
	infos := []evidence.SlotInfo{}
	for _, o := range checklist.SlotsForHost(defs, h) {
		infos = append(infos, evidence.SlotInfo{ID: o.ID, Title: o.Title})
	}
	for _, svc := range h.Services {
		for _, o := range checklist.SlotsForService(defs, svc) {
			infos = append(infos, evidence.SlotInfo{ID: o.ID, Title: o.Title, ServiceID: svc.ID})
		}
	}
	return infos
}

func (s *EvidenceService) hostView(st *store.Store, eng model.Engagement, h model.Host) (EvidenceView, error) {
	listing, err := st.EvidenceListing(h.IP)
	if err != nil {
		return EvidenceView{}, err
	}
	return evidence.Build(evidence.BuildInput{
		Host: h, Findings: eng.Findings, Slots: slotInfos(s.app.Definitions(), h),
		Items: listing.Items, NotIndexed: listing.NotIndexed, Missing: listing.Missing,
	}), nil
}

// List returns the host's evidence in capture order with derived states.
func (s *EvidenceService) List(ip string) (EvidenceView, error) {
	var v EvidenceView
	err := s.app.withStore("EvidenceService.List", func(st *store.Store) error {
		eng := st.Engagement()
		h, ok := eng.Host(ip)
		if !ok {
			return fmt.Errorf("host %s not found", ip)
		}
		var err error
		v, err = s.hostView(st, eng, h)
		return err
	})
	return v, err
}

// ListAll returns every host's evidence view in engagement host order.
func (s *EvidenceService) ListAll() ([]HostEvidence, error) {
	out := []HostEvidence{}
	err := s.app.withStore("EvidenceService.ListAll", func(st *store.Store) error {
		eng := st.Engagement()
		for _, h := range eng.Hosts {
			v, err := s.hostView(st, eng, h)
			if err != nil {
				return err
			}
			out = append(out, HostEvidence{IP: h.IP, View: v})
		}
		return nil
	})
	return out, err
}

// Search matches query against captions and text evidence across the engagement.
func (s *EvidenceService) Search(query string) ([]SearchHit, error) {
	hits := []SearchHit{}
	err := s.app.withStore("EvidenceService.Search", func(st *store.Store) error {
		eng := st.Engagement()
		hosts := make([]evidence.HostItems, 0, len(eng.Hosts))
		for _, h := range eng.Hosts {
			dir, err := evidenceDir(eng.Dir, h.IP)
			if err != nil {
				return err
			}
			hosts = append(hosts, evidence.HostItems{IP: h.IP, Dir: dir, Items: eng.Evidence[h.IP].Items})
		}
		hits = append(hits, evidence.Search(hosts, query)...)
		return nil
	})
	return hits, err
}

// itemPath returns the absolute path of an indexed item's file.
func (s *EvidenceService) itemPath(st *store.Store, ip, itemID string) (string, error) {
	eng := st.Engagement()
	if _, ok := eng.Host(ip); !ok {
		return "", fmt.Errorf("host %s not found", ip)
	}
	file := eng.Evidence[ip]
	i, err := findItem(&file, itemID)
	if err != nil {
		return "", err
	}
	return evidenceFilePath(eng.Dir, ip, file.Items[i].File)
}

// CopyPath returns the absolute path of the item's file for the no-reveal fallback.
func (s *EvidenceService) CopyPath(ip, itemID string) (string, error) {
	var path string
	err := s.app.withStore("EvidenceService.CopyPath", func(st *store.Store) error {
		var err error
		path, err = s.itemPath(st, ip, itemID)
		return err
	})
	return path, err
}

// Reveal shows the item's file in the platform file manager, or returns
// ErrNoReveal when this build cannot.
func (s *EvidenceService) Reveal(ip, itemID string) error {
	return s.app.withStore("EvidenceService.Reveal", func(st *store.Store) error {
		path, err := s.itemPath(st, ip, itemID)
		if err != nil {
			return err
		}
		if !s.Capabilities().Reveal {
			return ErrNoReveal
		}
		args, err := evidence.RevealArgs(runtime.GOOS, path)
		if err != nil {
			return err
		}
		return runReveal(args)
	})
}

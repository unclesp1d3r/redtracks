package store

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/unclesp1d3r/redtracks/internal/model"
)

// EvidenceListing is one host's evidence index next to what is actually on disk.
type EvidenceListing struct {
	Items      []model.EvidenceItem `json:"items"`
	NotIndexed []string             `json:"notIndexed"` // files in evidence/ with no entry
	Missing    []model.EvidenceItem `json:"missing"`    // entries whose file is gone
}

// evidenceSpec returns the spec for the evidence index of the host at ip.
func (s *Store) evidenceSpec(ip string) (fileSpec[model.EvidenceFile], error) {
	ip, err := validateIP(ip)
	if err != nil {
		return fileSpec[model.EvidenceFile]{}, err
	}
	path, err := evidencePath(s.dir, ip)
	if err != nil {
		return fileSpec[model.EvidenceFile]{}, err
	}
	return fileSpec[model.EvidenceFile]{
		path: path,
		get: func() (model.EvidenceFile, error) {
			if _, ok := s.eng.Host(ip); !ok {
				return model.EvidenceFile{}, fmt.Errorf("host %s not found", ip)
			}
			return s.eng.Evidence[ip], nil
		},
		set: func(f model.EvidenceFile) {
			next := make(map[string]model.EvidenceFile, len(s.eng.Evidence)+1)
			maps.Copy(next, s.eng.Evidence)
			next[ip] = f
			s.eng.Evidence = next
		},
		check:     func(f model.EvidenceFile) error { return checkEvidence(path, f) },
		normalize: model.EvidenceFile.Normalize,
	}, nil
}

// MutateEvidence applies fn to the evidence index of the host at ip and writes evidence.json.
func (s *Store) MutateEvidence(ip string, fn func(*model.EvidenceFile) error) error {
	sp, err := s.evidenceSpec(ip)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return mutate(s, sp, fn)
}

// EvidenceListing returns the host's indexed items plus the two derived lists:
// files with no entry and entries with no file. It reads the evidence
// directory without touching the files themselves.
func (s *Store) EvidenceListing(ip string) (EvidenceListing, error) {
	canon, err := validateIP(ip)
	if err != nil {
		return EvidenceListing{}, err
	}
	dir, err := evidenceDirPath(s.dir, canon)
	if err != nil {
		return EvidenceListing{}, err
	}
	s.mu.Lock()
	items := slices.Clone(s.eng.Evidence[canon].Items)
	s.mu.Unlock()
	files, err := listEvidenceFiles(dir)
	if err != nil {
		return EvidenceListing{}, err
	}
	listing := EvidenceListing{
		Items:      orEmptyItems(items),
		NotIndexed: []string{},
		Missing:    []model.EvidenceItem{},
	}
	indexed := make(map[string]bool, len(items))
	for _, it := range items {
		indexed[it.File] = true
		if !slices.Contains(files, it.File) {
			listing.Missing = append(listing.Missing, it)
		}
	}
	for _, f := range files {
		if !indexed[f] {
			listing.NotIndexed = append(listing.NotIndexed, f)
		}
	}
	return listing, nil
}

func orEmptyItems(items []model.EvidenceItem) []model.EvidenceItem {
	if items == nil {
		return []model.EvidenceItem{}
	}
	return items
}

// listEvidenceFiles returns the base names of the regular, non-dot files in dir.
func listEvidenceFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", dir, err)
	}
	var names []string
	for _, e := range entries {
		if e.Type().IsRegular() && !strings.HasPrefix(e.Name(), ".") {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

// trackEvidence refreshes the evidence index for sp's host and reports a
// changed evidence directory when its signature moved since the last Reload.
func (s *Store) trackEvidence(ip string, note func(path string, changed bool, err error)) {
	sp, err := s.evidenceSpec(ip)
	if err != nil {
		return // ip already validated by the caller
	}
	_, statErr := os.Stat(sp.path)
	if !errors.Is(statErr, os.ErrNotExist) || !s.mtimes[sp.path].IsZero() {
		ch, err := refresh(s, sp)
		note(sp.path, ch, err)
	}
	dir := filepath.Join(filepath.Dir(sp.path), evidenceDir)
	sig, err := DirSignature(dir, nil)
	if err != nil {
		note(dir, false, s.dedupeError(dir, err))
		return
	}
	delete(s.lastErrs, dir)
	if sig != s.dirSigs[dir] {
		s.dirSigs[dir] = sig
		note(dir, true, nil)
	}
}

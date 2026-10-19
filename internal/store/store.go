// Package store persists an engagement as a folder of plain files.
package store

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/unclesp1d3r/redtracks/internal/model"
)

// ErrConflict means a file changed on disk since the caller last read it.
// Callers wrap it with %w and the path; the frontend matches on the prefix
// "conflict".
var ErrConflict = errors.New("conflict")

// ErrHostExists means AddHost found the host already present.
var ErrHostExists = errors.New("already exists")

// Store holds one engagement folder in memory and keeps it in step with disk.
type Store struct {
	dir    string
	eng    model.Engagement
	hashes map[string][sha256.Size]byte // SHA-256 of the bytes last read or written, per path
	mtimes map[string]time.Time         // mtime seen at that moment, per path
	// failedMtimes holds the mtime of a file whose parse error was already reported.
	failedMtimes map[string]time.Time
	// lastErrs holds the text of the last stat or read error reported per path.
	lastErrs map[string]string
	// dirSigs holds the last DirSignature seen per watched directory.
	dirSigs map[string]string
	mu      sync.Mutex
}

// Open loads engagement.json, creds.json, findings.json and every
// hosts/*/host.json under dir. Missing files are not errors.
func Open(dir string) (*Store, error) {
	s := &Store{
		dir:    dir,
		eng:    model.Engagement{Dir: dir},
		hashes: map[string][sha256.Size]byte{},
		mtimes: map[string]time.Time{},

		failedMtimes: map[string]time.Time{},
		lastErrs:     map[string]string{},
		dirSigs:      map[string]string{},
	}
	if _, err := s.Reload(); err != nil {
		return nil, err
	}
	return s, nil
}

// Engagement returns a deep copy of the current state.
func (s *Store) Engagement() model.Engagement {
	s.mu.Lock()
	defer s.mu.Unlock()
	return mustClone(s.eng)
}

// mustClone deep-copies plain data through JSON. Marshal cannot fail for the
// model types, so a failure is a programming error.
func mustClone[T any](v T) T {
	out, err := clone(v)
	if err != nil {
		panic(fmt.Sprintf("store: clone %T: %v", v, err))
	}
	return out
}

func clone[T any](v T) (T, error) {
	var out T
	data, err := json.Marshal(v)
	if err != nil {
		return out, err
	}
	err = json.Unmarshal(data, &out)
	return out, err
}

// MutateMeta applies fn to engagement.json and writes the result.
func (s *Store) MutateMeta(fn func(*model.EngagementMeta) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return mutate(s, s.metaSpec(), fn)
}

// MutateCreds applies fn to creds.json and writes the result.
func (s *Store) MutateCreds(fn func(*model.Creds) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return mutate(s, s.credsSpec(), fn)
}

// MutateFindings applies fn to findings.json and writes the result.
func (s *Store) MutateFindings(fn func(*model.FindingsFile) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return mutate(s, s.findingsSpec(), fn)
}

// MutateHost applies fn to the host with the given IP and writes host.json.
func (s *Store) MutateHost(ip string, fn func(*model.Host) error) error {
	sp, err := s.hostSpec(ip)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return mutate(s, sp, fn)
}

// AddHost creates hosts/<ip>/host.json. It fails if the host already exists.
func (s *Store) AddHost(h model.Host) error {
	canon, err := validateIP(h.IP)
	if err != nil {
		return err
	}
	h.IP = canon
	sp, err := s.hostSpec(h.IP)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.eng.Host(h.IP); ok {
		return fmt.Errorf("host %s %w", h.IP, ErrHostExists)
	}
	if _, err := os.Stat(sp.path); err == nil {
		return fmt.Errorf("host %s %w: %s", h.IP, ErrHostExists, sp.path)
	}
	if err := os.MkdirAll(filepath.Dir(sp.path), dirMode); err != nil {
		return fmt.Errorf("create %s: %w", filepath.Dir(sp.path), err)
	}
	return persist(s, sp, h)
}

// Reload re-reads every file whose mtime changed and returns their paths
// relative to the engagement dir. A file that fails to parse or validate
// (an emptied host.json included) keeps its last good in-memory copy; the error is returned after the rest are processed.
func (s *Store) Reload() ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var changed []string
	var errs []error
	note := func(path string, didChange bool, err error) {
		if err != nil {
			errs = append(errs, withPath(path, err))
		}
		if didChange {
			changed = append(changed, s.rel(path))
		}
	}
	ch, err := refresh(s, s.metaSpec())
	note(metaPath(s.dir), ch, err)
	ch, err = refresh(s, s.credsSpec())
	note(credsPath(s.dir), ch, err)
	ch, err = refresh(s, s.findingsSpec())
	note(findingsPath(s.dir), ch, err)
	s.trackNotes(filepath.Join(s.dir, notesFile), note)
	s.reloadHosts(note)
	switch len(errs) {
	case 0:
		return changed, nil
	case 1:
		return changed, errs[0]
	default:
		return changed, errors.Join(errs...)
	}
}

// reloadHosts refreshes every hosts/*/host.json.
// ponytail: a host folder deleted while open stays in memory until the engagement is reopened;
// drop hosts whose folder is gone at the end of reloadHosts if that ever bites.
func (s *Store) reloadHosts(note func(path string, changed bool, err error)) {
	root := filepath.Join(s.dir, hostsDir)
	entries, err := os.ReadDir(root)
	if errors.Is(err, os.ErrNotExist) {
		return
	}
	if err != nil {
		note(root, false, fmt.Errorf("read %s: %w", root, err))
		return
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		sp, err := s.hostSpec(ipFromDirName(e.Name()))
		if err != nil {
			note(filepath.Join(root, e.Name()), false, err)
			continue
		}
		s.trackNotes(filepath.Join(filepath.Dir(sp.path), notesFile), note)
		s.trackEvidence(ipFromDirName(e.Name()), note)
		_, statErr := os.Stat(sp.path)
		if errors.Is(statErr, fs.ErrNotExist) && s.mtimes[sp.path].IsZero() {
			continue // a folder with only notes.md is a host with no structured state
		}
		ch, err := refresh(s, sp)
		note(sp.path, ch, err)
	}
}

// trackNotes records a notes.md mtime so an external edit shows up in the
// changed list. The notes text itself is not held in memory.
func (s *Store) trackNotes(path string, note func(path string, changed bool, err error)) {
	mtime, err := statMtime(path)
	if err != nil {
		note(path, false, s.dedupeError(path, err))
		return
	}
	delete(s.lastErrs, path)
	known, seen := s.mtimes[path]
	if (seen && known.Equal(mtime)) || (!seen && mtime.IsZero()) {
		return
	}
	s.mtimes[path] = mtime
	note(path, true, nil)
}

// rel returns path relative to the engagement dir, slash-separated.
func (s *Store) rel(path string) string {
	r, err := filepath.Rel(s.dir, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(r)
}

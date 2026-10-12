package store

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"slices"
	"time"

	"github.com/unclesp1d3r/redtracks/internal/model"
)

// fileSpec describes one JSON file the store owns and where its parsed value
// lives in memory.
type fileSpec[T any] struct {
	path      string
	get       func() (T, error)
	set       func(T)
	check     func(T) error // validates a freshly decoded value
	normalize func(T) T     // fills nil slices so they marshal as []
}

func (s *Store) metaSpec() fileSpec[model.EngagementMeta] {
	return fileSpec[model.EngagementMeta]{
		path:  metaPath(s.dir),
		get:   func() (model.EngagementMeta, error) { return s.eng.Meta, nil },
		set:   func(m model.EngagementMeta) { s.eng.Meta = m },
		check: func(m model.EngagementMeta) error { return checkMeta(metaPath(s.dir), m) },
		normalize: func(m model.EngagementMeta) model.EngagementMeta {
			if m.FlagTypes == nil {
				m.FlagTypes = []model.FlagType{}
			}
			return m
		},
	}
}

func (s *Store) credsSpec() fileSpec[model.Creds] {
	return fileSpec[model.Creds]{
		path:      credsPath(s.dir),
		get:       func() (model.Creds, error) { return s.eng.Creds, nil },
		set:       func(c model.Creds) { s.eng.Creds = c },
		normalize: model.Creds.Normalize,
	}
}

func (s *Store) findingsSpec() fileSpec[model.FindingsFile] {
	return fileSpec[model.FindingsFile]{
		path: findingsPath(s.dir),
		get:  func() (model.FindingsFile, error) { return model.FindingsFile{Findings: s.eng.Findings}, nil },
		set:  func(f model.FindingsFile) { s.eng.Findings = f.Findings },
		normalize: func(f model.FindingsFile) model.FindingsFile {
			if f.Findings == nil {
				f.Findings = []model.Finding{}
			}
			return f
		},
	}
}

// hostSpec returns the spec for the host at ip, or an error if ip is invalid.
func (s *Store) hostSpec(ip string) (fileSpec[model.Host], error) {
	ip, err := validateIP(ip) // canonical, so "FE80::1" and "fe80::1" are one host
	if err != nil {
		return fileSpec[model.Host]{}, err
	}
	path, err := hostPath(s.dir, ip)
	if err != nil {
		return fileSpec[model.Host]{}, err
	}
	index := func() int {
		return slices.IndexFunc(s.eng.Hosts, func(h model.Host) bool { return h.IP == ip })
	}
	return fileSpec[model.Host]{
		path: path,
		get: func() (model.Host, error) {
			i := index()
			if i < 0 {
				return model.Host{}, fmt.Errorf("host %s not found", ip)
			}
			return s.eng.Hosts[i], nil
		},
		set: func(h model.Host) {
			hosts := append([]model.Host(nil), s.eng.Hosts...)
			if i := index(); i >= 0 {
				hosts[i] = h
			} else {
				hosts = append(hosts, h)
			}
			s.eng.Hosts = hosts
		},
		check: func(h model.Host) error {
			if h.IP != ip {
				return fmt.Errorf("%s: host.json ip %q does not match folder name %q", path, h.IP, ip)
			}
			return checkHost(path, h)
		},
		normalize: model.Host.Normalize,
	}, nil
}

// statMtime returns the file's mtime, or the zero time if the file is missing.
func statMtime(path string) (time.Time, error) {
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, fmt.Errorf("stat %s: %w", path, err)
	}
	return info.ModTime(), nil
}

// readState returns the file bytes (nil if missing) and its mtime (zero if missing).
func readState(path string) ([]byte, time.Time, error) {
	mtime, err := statMtime(path)
	if err != nil {
		return nil, time.Time{}, err
	}
	data, err := readOptional(path)
	return data, mtime, err
}

// decode parses data and runs check. A missing file (nil data) decodes to the
// zero value unchecked; an existing but empty file is checked like any other
// content, so an emptied host.json is rejected rather than adopted as a zero host.
func (sp fileSpec[T]) decode(data []byte) (T, error) {
	var v T
	if data == nil {
		return v, nil
	}
	if len(data) > 0 {
		if err := decodeJSON(sp.path, data, &v); err != nil {
			return v, err
		}
	}
	if sp.normalize != nil {
		v = sp.normalize(v) // a hand-written file may omit defaults the model fills in
	}
	if sp.check != nil {
		if err := sp.check(v); err != nil {
			return v, err
		}
	}
	return v, nil
}

// record remembers the bytes and mtime that the in-memory copy corresponds to.
func (s *Store) record(path string, data []byte, mtime time.Time) {
	s.hashes[path] = sha256.Sum256(data)
	s.mtimes[path] = mtime
}

// refresh re-reads the file when its mtime moved and adopts the parsed value.
// On a parse or validation failure (an emptied file included) the in-memory
// copy and recorded hash stay untouched; the failing mtime is remembered so
// the error surfaces once per on-disk change.
// Stat and read errors are reported once until their text changes.
func refresh[T any](s *Store, sp fileSpec[T]) (bool, error) {
	changed, err := refreshFile(s, sp)
	return changed, s.dedupeError(sp.path, err)
}

// dedupeError returns err only when it differs from the last error reported
// for path, and forgets the path's last error on success.
func (s *Store) dedupeError(path string, err error) error {
	if err == nil {
		delete(s.lastErrs, path)
		return nil
	}
	if s.lastErrs[path] == err.Error() {
		return nil
	}
	s.lastErrs[path] = err.Error()
	return err
}

func refreshFile[T any](s *Store, sp fileSpec[T]) (bool, error) {
	mtime, err := statMtime(sp.path)
	if err != nil {
		return false, err
	}
	known, seen := s.mtimes[sp.path]
	if mtime.IsZero() && seen && !known.IsZero() {
		return false, fmt.Errorf("%s was deleted while open, keeping the last copy: %w", sp.path, fs.ErrNotExist)
	}
	if seen && known.Equal(mtime) {
		return false, nil
	}
	if failed, ok := s.failedMtimes[sp.path]; ok && failed.Equal(mtime) {
		return false, nil // already reported this corrupt revision
	}
	data, err := readOptional(sp.path)
	if err != nil {
		return false, err
	}
	v, err := sp.decode(data)
	if err != nil {
		s.failedMtimes[sp.path] = mtime
		return false, err
	}
	delete(s.failedMtimes, sp.path)
	sp.set(v)
	s.record(sp.path, data, mtime)
	return true, nil
}

// mutate is the hash-check-and-reapply write path; the caller holds s.mu.
// If the file changed since the last read it is reparsed first, so fn always
// runs against what is on disk. A corrupt file is never overwritten. A file
// that is missing and was never loaded is not adopted, so a mutation on an
// unknown host fails "not found" instead of inserting a zero value; a file
// emptied or removed after loading is refused.
func mutate[T any](s *Store, sp fileSpec[T], fn func(*T) error) error {
	data, _, err := readState(sp.path)
	if err != nil {
		return err
	}
	if len(data) == 0 && !s.mtimes[sp.path].IsZero() {
		return fmt.Errorf("%s vanished or was emptied while open: %w", sp.path, os.ErrNotExist)
	}
	if data != nil && s.hashes[sp.path] != sha256.Sum256(data) {
		fresh, err := sp.decode(data)
		if err != nil {
			return err
		}
		sp.set(fresh)
		// Only the hash is updated: the stale mtime makes the next Reload
		// re-read the file and report it, even if fn fails below.
		s.hashes[sp.path] = sha256.Sum256(data)
	}
	cur, err := sp.get()
	if err != nil {
		return err
	}
	work, err := clone(cur)
	if err != nil {
		return fmt.Errorf("copy %s: %w", sp.path, err)
	}
	if err := fn(&work); err != nil {
		return err
	}
	return persist(s, sp, work)
}

// marshalIndentNL marshals v as two-space-indented JSON with a trailing newline.
func marshalIndentNL(v any) ([]byte, error) {
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}

// persist normalises v, writes it atomically, and adopts it in memory.
func persist[T any](s *Store, sp fileSpec[T], v T) error {
	if sp.normalize != nil {
		v = sp.normalize(v)
	}
	if sp.check != nil {
		if err := sp.check(v); err != nil {
			return err
		}
	}
	out, err := marshalIndentNL(v)
	if err != nil {
		return fmt.Errorf("marshal %s: %w", sp.path, err)
	}
	if err := WriteAtomic(sp.path, out); err != nil {
		return err
	}
	info, err := os.Stat(sp.path)
	if err != nil {
		return fmt.Errorf("stat %s: %w", sp.path, err)
	}
	s.record(sp.path, out, info.ModTime())
	sp.set(v)
	return nil
}

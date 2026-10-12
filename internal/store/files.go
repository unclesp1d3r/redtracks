package store

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
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
		path: metaPath(s.dir),
		get:  func() (model.EngagementMeta, error) { return s.eng.Meta, nil },
		set:  func(m model.EngagementMeta) { s.eng.Meta = m },
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
	path, err := hostPath(s.dir, ip)
	if err != nil {
		return fileSpec[model.Host]{}, err
	}
	index := func() int {
		for i, h := range s.eng.Hosts {
			if h.IP == ip {
				return i
			}
		}
		return -1
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
			return nil
		},
		normalize: model.Host.Normalize,
	}, nil
}

// readState returns the file bytes (nil if missing) and its mtime (zero if missing).
func readState(path string) ([]byte, time.Time, error) {
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, time.Time{}, nil
	}
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("stat %s: %w", path, err)
	}
	data, err := readOptional(path)
	return data, info.ModTime(), err
}

func (sp fileSpec[T]) decode(data []byte) (T, error) {
	var v T
	if len(data) == 0 {
		return v, nil
	}
	if err := decodeJSON(sp.path, data, &v); err != nil {
		return v, err
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
// On a parse failure the in-memory copy and recorded hash stay untouched.
func refresh[T any](s *Store, sp fileSpec[T]) (bool, error) {
	data, mtime, err := readState(sp.path)
	if err != nil {
		return false, err
	}
	if known, ok := s.mtimes[sp.path]; ok && known.Equal(mtime) {
		return false, nil
	}
	v, err := sp.decode(data)
	if err != nil {
		return false, err
	}
	sp.set(v)
	s.record(sp.path, data, mtime)
	return true, nil
}

// mutate is the hash-check-and-reapply write path; the caller holds s.mu.
// If the file changed since the last read it is reparsed first, so fn always
// runs against what is on disk. A corrupt file is never overwritten.
func mutate[T any](s *Store, sp fileSpec[T], fn func(*T) error) error {
	data, mtime, err := readState(sp.path)
	if err != nil {
		return err
	}
	if s.hashes[sp.path] != sha256.Sum256(data) {
		fresh, err := sp.decode(data)
		if err != nil {
			return err
		}
		sp.set(fresh)
		s.record(sp.path, data, mtime)
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

// persist normalises v, writes it atomically, and adopts it in memory.
func persist[T any](s *Store, sp fileSpec[T], v T) error {
	if sp.normalize != nil {
		v = sp.normalize(v)
	}
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", sp.path, err)
	}
	out = append(out, '\n')
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

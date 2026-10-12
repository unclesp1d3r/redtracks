package store

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	dirMode      os.FileMode = 0o700
	importStamp              = "20060102-150405"
	maxImportTry             = 3600
)

// toolName limits the tool prefix of an import file name to a safe alphabet.
var toolName = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

func hashHex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// readOptional returns the file bytes, or nil when the file does not exist.
func readOptional(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return data, nil
}

// ReadNotes returns the notes text and the hex SHA-256 of its bytes. An empty
// ip selects the engagement notes. A missing file reads as "" with the hash of
// empty content.
//
//nolint:gocritic // results are text, hash; nonamedreturns bans naming them
func (s *Store) ReadNotes(ip string) (string, string, error) {
	path, err := notesPath(s.dir, ip)
	if err != nil {
		return "", "", err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := readOptional(path)
	if err != nil {
		return "", "", err
	}
	return string(data), hashHex(data), nil
}

// WriteNotes replaces the notes only if the file still hashes to baseHash;
// otherwise it returns an error wrapping ErrConflict.
func (s *Store) WriteNotes(ip, text, baseHash string) error {
	path, err := notesPath(s.dir, ip)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	current, err := readOptional(path)
	if err != nil {
		return err
	}
	if hashHex(current) != baseHash {
		return fmt.Errorf("%w: %s changed on disk", ErrConflict, path)
	}
	if err := os.MkdirAll(filepath.Dir(path), dirMode); err != nil {
		return fmt.Errorf("create %s: %w", filepath.Dir(path), err)
	}
	return WriteAtomic(path, []byte(text))
}

func importExt(raw []byte) string {
	switch {
	case strings.HasPrefix(string(raw), "<"):
		return "xml"
	case strings.HasPrefix(string(raw), "{"), strings.HasPrefix(string(raw), "["):
		return "json"
	default:
		return "txt"
	}
}

// SaveImport stores raw tool output unchanged under hosts/<ip>/imports/ and
// returns its slash-separated path relative to the engagement dir.
func (s *Store) SaveImport(ip, tool string, raw []byte) (string, error) {
	dir, err := hostDir(s.dir, ip)
	if err != nil {
		return "", err
	}
	if !toolName.MatchString(tool) {
		return "", fmt.Errorf("invalid tool name %q", tool)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	impDir := filepath.Join(dir, importsDir)
	if err := os.MkdirAll(impDir, dirMode); err != nil {
		return "", fmt.Errorf("create %s: %w", impDir, err)
	}
	path, err := freeImportPath(impDir, tool, importExt(raw))
	if err != nil {
		return "", err
	}
	if err := WriteAtomic(path, raw); err != nil {
		return "", err
	}
	return s.rel(path), nil
}

// freeImportPath picks <tool>-<timestamp>.<ext>, advancing the timestamp by a
// second while the name is taken so two imports never overwrite each other.
func freeImportPath(dir, tool, ext string) (string, error) {
	now := time.Now().UTC()
	for i := range maxImportTry {
		name := fmt.Sprintf("%s-%s.%s", tool, now.Add(time.Duration(i)*time.Second).Format(importStamp), ext)
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
			return path, nil
		} else if err != nil {
			return "", fmt.Errorf("stat %s: %w", path, err)
		}
	}
	return "", fmt.Errorf("no free import name in %s", dir)
}

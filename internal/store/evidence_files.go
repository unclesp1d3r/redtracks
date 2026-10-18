package store

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/unclesp1d3r/redtracks/internal/evidence"
	"github.com/unclesp1d3r/redtracks/internal/model"
	"github.com/unclesp1d3r/redtracks/internal/textutil"
)

const (
	// maxEvidenceMB is the evidence size cap; it equals the import cap in internal/app.
	maxEvidenceMB    = 20
	maxEvidenceBytes = maxEvidenceMB << 20

	evidenceStamp = "20060102-150405"

	hookAfterFile  = "file"  // the file is complete, the index is not yet written
	hookAfterIndex = "index" // the index entry is written
)

// evidenceHook is a test seam called at the stages above; a non-nil error aborts the add.
var evidenceHook = func(string) error { return nil }

// AddedEvidence is the result of storing one evidence file.
type AddedEvidence struct {
	Item      model.EvidenceItem
	Duplicate bool // another item on the host has identical stored bytes
}

// AddEvidenceFile stores src as a new evidence file on the host at ip and
// indexes it. Images are written byte for byte; text is stripped of terminal
// control sequences first. The file is written once under a generated name and
// never rewritten. It is removed again if the index entry was not written.
func (s *Store) AddEvidenceFile(ip string, src io.Reader, capturedAt time.Time, caption string,
	links []model.EvidenceLink,
) (AddedEvidence, error) {
	canon, err := validateIP(ip)
	if err != nil {
		return AddedEvidence{}, err
	}
	dir, err := evidenceDirPath(s.dir, canon)
	if err != nil {
		return AddedEvidence{}, err
	}
	if err := s.requireHost(canon); err != nil {
		return AddedEvidence{}, err
	}
	sniffed, body, err := sniffAndBound(src)
	if err != nil {
		return AddedEvidence{}, err
	}
	capturedAt = capturedAt.UTC()
	stem := capturedAt.Format(evidenceStamp) + "-" + string(sniffed.Kind)
	written, err := writeEvidenceFile(dir, stem, sniffed.Ext, body, capturedAt)
	if err != nil {
		return AddedEvidence{}, err
	}
	added, err := s.indexEvidence(canon, model.EvidenceItem{
		ID: model.NewID(), File: written.name, Kind: sniffed.Kind, CapturedAt: capturedAt,
		Caption: caption, Links: slices.Clone(links), SHA256: written.digest,
	})
	if err != nil {
		return AddedEvidence{}, errors.Join(err, s.discardUnindexed(canon, dir, written.name))
	}
	return added, nil
}

func (s *Store) requireHost(ip string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.eng.Host(ip); !ok {
		return fmt.Errorf("host %s not found", ip)
	}
	return nil
}

// sniffAndBound classifies src and returns a reader of the bytes to store, at
// most maxEvidenceBytes of them. Text is read whole and stripped.
func sniffAndBound(src io.Reader) (evidence.Sniffed, io.Reader, error) {
	head := make([]byte, evidence.SniffLen)
	n, err := io.ReadFull(src, head)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return evidence.Sniffed{}, nil, fmt.Errorf("read evidence: %w", err)
	}
	sniffed, err := evidence.Sniff(head[:n])
	if err != nil {
		return evidence.Sniffed{}, nil, err
	}
	whole := io.LimitReader(io.MultiReader(bytes.NewReader(head[:n]), src), maxEvidenceBytes+1)
	if sniffed.Kind != model.EvidenceKindText {
		return sniffed, whole, nil
	}
	raw, err := io.ReadAll(whole)
	if err != nil {
		return evidence.Sniffed{}, nil, fmt.Errorf("read evidence: %w", err)
	}
	return sniffed, bytes.NewReader(textutil.Strip(raw)), checkEvidenceSize(int64(len(raw)))
}

func checkEvidenceSize(n int64) error {
	if n > maxEvidenceBytes {
		return fmt.Errorf("evidence is larger than %d MB", maxEvidenceMB)
	}
	return nil
}

// writtenEvidence names a file just written and the hex SHA-256 of its bytes.
type writtenEvidence struct{ name, digest string }

// writeEvidenceFile streams body into a fresh file, fsyncs it, and stamps its
// mtime. A
// failed write leaves no file behind.
func writeEvidenceFile(dir, stem, ext string, body io.Reader, capturedAt time.Time) (writtenEvidence, error) {
	f, name, err := CreateExclusive(dir, stem, ext)
	if err != nil {
		return writtenEvidence{}, err
	}
	path := filepath.Join(dir, name)
	sum := sha256.New()
	n, err := io.Copy(io.MultiWriter(f, sum), body)
	if err == nil {
		err = checkEvidenceSize(n)
	}
	if err == nil {
		err = f.Sync()
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Chtimes(path, capturedAt, capturedAt)
	}
	if err != nil {
		return writtenEvidence{}, errors.Join(fmt.Errorf("write %s: %w", path, err), removeIgnoringMissing(path))
	}
	return writtenEvidence{name: name, digest: hex.EncodeToString(sum.Sum(nil))}, nil
}

// indexEvidence appends item to the host's index, reporting whether its digest was already present.
func (s *Store) indexEvidence(ip string, item model.EvidenceItem) (AddedEvidence, error) {
	if err := evidenceHook(hookAfterFile); err != nil {
		return AddedEvidence{}, err
	}
	var duplicate bool
	err := s.MutateEvidence(ip, func(f *model.EvidenceFile) error {
		duplicate = slices.ContainsFunc(f.Items, func(it model.EvidenceItem) bool { return it.SHA256 == item.SHA256 })
		f.Items = append(f.Items, item)
		return nil
	})
	if err != nil {
		return AddedEvidence{}, err
	}
	return AddedEvidence{Item: item, Duplicate: duplicate}, evidenceHook(hookAfterIndex)
}

// discardUnindexed removes name unless evidence.json on disk already names it
// (a write can fail after the rename that published the entry).
func (s *Store) discardUnindexed(ip, dir, name string) error {
	path, err := evidencePath(s.dir, ip)
	if err != nil {
		return err
	}
	data, err := readOptional(path)
	if err != nil || bytes.Contains(data, []byte(`"`+name+`"`)) {
		return err
	}
	return removeIgnoringMissing(filepath.Join(dir, name))
}

func removeIgnoringMissing(path string) error {
	if err := removeFile(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove %s: %w", path, err)
	}
	return nil
}

// ImportEvidenceFile copies the regular file at path into the host's evidence
// as AddEvidenceFile does, with the file's mtime (never later than now) as the
// capture time. The source is only read, never moved or changed.
func (s *Store) ImportEvidenceFile(ip, path, caption string, links []model.EvidenceLink) (AddedEvidence, error) {
	f, info, err := evidence.OpenRegular(path)
	if err != nil {
		return AddedEvidence{}, err
	}
	defer func() { _ = f.Close() }()
	return s.AddEvidenceFile(ip, f, minTime(info.ModTime(), time.Now()), caption, links)
}

func minTime(a, b time.Time) time.Time {
	if a.After(b) {
		return b
	}
	return a
}

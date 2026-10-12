package app

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/unclesp1d3r/redtracks/internal/importers"
	"github.com/unclesp1d3r/redtracks/internal/model"
	"github.com/unclesp1d3r/redtracks/internal/store"
)

// maxImportBytes caps the size of an import file.
const (
	maxImportMB    = 20
	maxImportBytes = maxImportMB << 20
)

var errUnknownToken = errors.New("unknown token")

type pendingImport struct {
	result importers.Result
	raw    []byte
}

// ImportService previews tool output and merges it into the open engagement.
type ImportService struct {
	app     *App
	mu      sync.Mutex
	pending map[string]pendingImport
}

// NewImportService returns the import service bound to a.
func NewImportService(a *App) *ImportService {
	return &ImportService{app: a, pending: map[string]pendingImport{}}
}

// PreviewResponse is a parsed import awaiting Apply or Discard.
type PreviewResponse struct {
	Token   string
	Preview importers.Preview
}

// PreviewFile reads and previews the file at path.
func (s *ImportService) PreviewFile(path string) (PreviewResponse, error) {
	resp, err := s.previewFile(path)
	return resp, s.app.fail("ImportService.PreviewFile", err)
}

func (s *ImportService) previewFile(path string) (PreviewResponse, error) {
	f, err := os.Open(path)
	if err != nil {
		return PreviewResponse{}, fmt.Errorf("open %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	raw, err := io.ReadAll(io.LimitReader(f, maxImportBytes+1))
	if err != nil {
		return PreviewResponse{}, fmt.Errorf("read %s: %w", path, err)
	}
	if len(raw) > maxImportBytes {
		return PreviewResponse{}, fmt.Errorf("%s is larger than %d MB", path, maxImportMB)
	}
	return s.preview(raw)
}

// PreviewData previews standard-base64 file contents (drag and drop).
func (s *ImportService) PreviewData(name, dataBase64 string) (PreviewResponse, error) {
	resp, err := s.previewData(name, dataBase64)
	return resp, s.app.fail("ImportService.PreviewData", err)
}

func (s *ImportService) previewData(name, dataBase64 string) (PreviewResponse, error) {
	if base64.StdEncoding.DecodedLen(len(dataBase64)) > maxImportBytes+maxImportBytes/2 {
		return PreviewResponse{}, fmt.Errorf("%s is larger than %d MB", name, maxImportMB)
	}
	raw, err := base64.StdEncoding.DecodeString(dataBase64)
	if err != nil {
		return PreviewResponse{}, fmt.Errorf("decode %s: %w", name, err)
	}
	if len(raw) > maxImportBytes {
		return PreviewResponse{}, fmt.Errorf("%s is larger than %d MB", name, maxImportMB)
	}
	return s.preview(raw)
}

func (s *ImportService) preview(raw []byte) (PreviewResponse, error) {
	st, err := s.app.Store()
	if err != nil {
		return PreviewResponse{}, err
	}
	res, err := importers.Parse(raw)
	if err != nil {
		return PreviewResponse{}, err
	}
	token := model.NewID()
	s.mu.Lock()
	s.pending[token] = pendingImport{result: res, raw: raw}
	s.mu.Unlock()
	return PreviewResponse{Token: token, Preview: importers.BuildPreview(st.Engagement(), res, nil)}, nil
}

// Apply merges a previewed import into the engagement; the token is spent on success.
func (s *ImportService) Apply(token string, overrides map[string]string) (importers.Preview, error) {
	var out importers.Preview
	err := s.app.withStore("ImportService.Apply", func(st *store.Store) error {
		p, ok := s.lookup(token)
		if !ok {
			return fmt.Errorf("%w %q", errUnknownToken, token)
		}
		var aerr error
		if out, aerr = importers.Apply(st, p.result, overrides, p.raw); aerr != nil {
			return aerr
		}
		s.drop(token)
		return nil
	})
	return out, err
}

// Discard forgets a previewed import.
func (s *ImportService) Discard(token string) error {
	if _, ok := s.lookup(token); !ok {
		return s.app.fail("ImportService.Discard", fmt.Errorf("%w %q", errUnknownToken, token))
	}
	s.drop(token)
	return nil
}

func (s *ImportService) lookup(token string) (pendingImport, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.pending[token]
	return p, ok
}

func (s *ImportService) drop(token string) {
	s.mu.Lock()
	delete(s.pending, token)
	s.mu.Unlock()
}

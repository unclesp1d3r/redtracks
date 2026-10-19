package app

import (
	"fmt"
	"io"

	"github.com/unclesp1d3r/redtracks/internal/evidence"
)

// PreviewPath previews the file at an absolute path (a native drop onto the
// import zone), through the same cap as PreviewData.
func (s *ImportService) PreviewPath(path string) (PreviewResponse, error) {
	raw, err := readImportFile(path)
	if err != nil {
		return PreviewResponse{}, s.app.fail("ImportService.PreviewPath", err)
	}
	resp, err := s.preview(raw)
	return resp, s.app.fail("ImportService.PreviewPath", err)
}

func readImportFile(path string) ([]byte, error) {
	f, info, err := evidence.OpenRegular(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	if info.Size() > maxImportBytes {
		return nil, fmt.Errorf("%s is larger than %d MB", info.Name(), maxImportMB)
	}
	// Bound the read too: the file may grow after the stat.
	raw, err := io.ReadAll(io.LimitReader(f, maxImportBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	if len(raw) > maxImportBytes {
		return nil, fmt.Errorf("%s is larger than %d MB", info.Name(), maxImportMB)
	}
	return raw, nil
}

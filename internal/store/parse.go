package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
)

// ParseError reports a malformed file with a 1-based line and column.
type ParseError struct {
	Path      string
	Line, Col int
	Err       error
}

// Error formats as "<path>:<line>:<col>: <err>".
func (e *ParseError) Error() string {
	return fmt.Sprintf("%s:%d:%d: %v", e.Path, e.Line, e.Col, e.Err)
}

// Unwrap returns the underlying decoder error.
func (e *ParseError) Unwrap() error { return e.Err }

// FileError ties an error to the file it concerns without changing its
// message, so callers can group errors per file.
type FileError struct {
	Path string
	Err  error
}

func (e *FileError) Error() string { return e.Err.Error() }

// Unwrap returns the underlying error.
func (e *FileError) Unwrap() error { return e.Err }

// ErrorPath returns the file err concerns, or "" when none is recorded.
func ErrorPath(err error) string {
	if pe, ok := errors.AsType[*ParseError](err); ok {
		return pe.Path
	}
	if fe, ok := errors.AsType[*FileError](err); ok {
		return fe.Path
	}
	return ""
}

// withPath attaches path to err unless it already names a file.
func withPath(path string, err error) error {
	if err == nil || ErrorPath(err) != "" {
		return err
	}
	return &FileError{Path: path, Err: err}
}

// decodeJSON unmarshals data into v. Syntax and type errors become a
// *ParseError with the position derived from the error's byte offset; any
// other error is wrapped with path.
func decodeJSON(path string, data []byte, v any) error {
	err := json.Unmarshal(data, v)
	if err == nil {
		return nil
	}
	var offset int64
	var syntaxErr *json.SyntaxError
	var typeErr *json.UnmarshalTypeError
	switch {
	case errors.As(err, &syntaxErr):
		offset = syntaxErr.Offset
	case errors.As(err, &typeErr):
		offset = typeErr.Offset
	default:
		return fmt.Errorf("%s: %w", path, err)
	}
	return newParseError(path, data, offset, err)
}

// newParseError converts a byte offset into a 1-based line and column.
func newParseError(path string, data []byte, offset int64, err error) *ParseError {
	end := int(min(max(offset, 0), int64(len(data))))
	head := data[:end]
	return &ParseError{
		Path: path,
		Line: bytes.Count(head, []byte{'\n'}) + 1,
		Col:  end - bytes.LastIndexByte(head, '\n'),
		Err:  err,
	}
}

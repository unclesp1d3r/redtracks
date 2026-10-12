package store

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestDecodeJSONReportsLineAndCol(t *testing.T) {
	var v struct {
		IP string `json:"ip"`
	}
	err := decodeJSON("host.json", []byte("{\n  \"ip\": 1,\n  bad\n}"), &v)
	var pe *ParseError
	if !errors.As(err, &pe) {
		t.Fatalf("want *ParseError, got %v", err)
	}
	if pe.Line != 3 {
		t.Fatalf("Line = %d, want 3", pe.Line)
	}
	if pe.Col < 1 {
		t.Fatalf("Col = %d, want >= 1", pe.Col)
	}
	if !strings.HasPrefix(pe.Error(), "host.json:3:") {
		t.Fatalf("Error() = %q", pe.Error())
	}
}

func TestDecodeJSONTypeMismatch(t *testing.T) {
	var v struct {
		Port int `json:"port"`
	}
	err := decodeJSON("svc.json", []byte(`{"port":"eighty"}`), &v)
	var pe *ParseError
	if !errors.As(err, &pe) {
		t.Fatalf("want *ParseError, got %v", err)
	}
	if pe.Line == 0 {
		t.Fatal("Line should be nonzero")
	}
}

func TestDecodeJSONFirstLineColumn(t *testing.T) {
	var v map[string]any
	err := decodeJSON("x.json", []byte(`{`), &v)
	var pe *ParseError
	if !errors.As(err, &pe) || pe.Line != 1 || pe.Col != 2 {
		t.Fatalf("got %v, want line 1 col 2 (EOF after one byte)", err)
	}
}

func TestDecodeJSONSuccess(t *testing.T) {
	var v struct{ A int }
	if err := decodeJSON("x.json", []byte(`{"A":2}`), &v); err != nil || v.A != 2 {
		t.Fatalf("err=%v v=%+v", err, v)
	}
}

func TestDecodeJSONOtherErrorWrappedWithPath(t *testing.T) {
	err := decodeJSON("x.json", []byte(`{}`), nil)
	if err == nil || !strings.Contains(err.Error(), "x.json") {
		t.Fatalf("want error naming path, got %v", err)
	}
	var pe *ParseError
	if errors.As(err, &pe) {
		t.Fatal("non-positional error must not be a ParseError")
	}
}

func TestParseErrorRecoveredThroughWrapping(t *testing.T) {
	var v map[string]any
	inner := decodeJSON("x.json", []byte("{\n bad"), &v)
	wrapped := fmt.Errorf("open: %w", fmt.Errorf("load: %w", inner))
	var pe *ParseError
	if !errors.As(wrapped, &pe) || pe.Path != "x.json" || pe.Line != 2 {
		t.Fatalf("errors.As failed: %v", wrapped)
	}
	if pe.Unwrap() == nil {
		t.Fatal("Unwrap returned nil")
	}
}

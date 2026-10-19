package app

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestDecodeCappedRefusesOversizeBeforeDecoding(t *testing.T) {
	// Not valid base64, so only the length check can produce the "larger than" error.
	huge := strings.Repeat("!", 4*(maxImportBytes+maxImportBytes/2)/3+16)

	_, err := decodeCapped("big.bin", huge, maxImportBytes)

	if err == nil || !strings.Contains(err.Error(), "big.bin is larger than 20 MB") {
		t.Fatalf("err = %v", err)
	}
}

func TestDecodeCappedChecksDecodedLength(t *testing.T) {
	raw := strings.Repeat("a", 11)
	enc := base64.StdEncoding.EncodeToString([]byte(raw))

	if _, err := decodeCapped("n", enc, 10); err == nil || !strings.Contains(err.Error(), "larger than") {
		t.Errorf("over-cap err = %v", err)
	}
	got, err := decodeCapped("n", enc, 11)
	if err != nil || string(got) != raw {
		t.Errorf("got %q, %v", got, err)
	}
	if _, err := decodeCapped("n", "***", 10); err == nil || !strings.Contains(err.Error(), "decode n") {
		t.Errorf("bad base64 err = %v", err)
	}
}

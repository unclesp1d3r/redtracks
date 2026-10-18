package app

import (
	"encoding/base64"
	"fmt"
)

const bytesPerMB = 1 << 20

// decodeCapped decodes standard base64 and refuses a payload over maxBytes. The
// length is checked before decoding (with slack for padding) so an oversize
// string never allocates its decoded copy.
func decodeCapped(name, dataBase64 string, maxBytes int) ([]byte, error) {
	if base64.StdEncoding.DecodedLen(len(dataBase64)) > maxBytes+maxBytes/2 {
		return nil, fmt.Errorf("%s is larger than %d MB", name, maxBytes/bytesPerMB)
	}
	raw, err := base64.StdEncoding.DecodeString(dataBase64)
	if err != nil {
		return nil, fmt.Errorf("decode %s: %w", name, err)
	}
	if len(raw) > maxBytes {
		return nil, fmt.Errorf("%s is larger than %d MB", name, maxBytes/bytesPerMB)
	}
	return raw, nil
}

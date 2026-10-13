package model

import "crypto/rand"

// NewID returns a new opaque record ID.
func NewID() string {
	return rand.Text()
}

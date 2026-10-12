// Package model defines the RedTracks domain types and their on-disk JSON shape.
package model

import "crypto/rand"

// NewID returns a new opaque record ID.
func NewID() string {
	return rand.Text()
}

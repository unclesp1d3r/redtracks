//go:build windows

package store

// openFlags has no equivalent on Windows; the regular-file check still applies.
const openFlags = 0

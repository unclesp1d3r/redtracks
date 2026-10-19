//go:build windows

package evidence

// openFlags has no equivalent on Windows; the regular-file check still applies.
const openFlags = 0

//go:build windows

package sessionlogs

// openFlags has no equivalent on Windows; the regular-file check still applies.
const openFlags = 0

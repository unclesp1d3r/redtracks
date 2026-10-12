package store

import (
	"fmt"
	"net/netip"
	"path/filepath"
)

const (
	metaFile     = "engagement.json"
	credsFile    = "creds.json"
	findingsFile = "findings.json"
	hostFile     = "host.json"
	notesFile    = "notes.md"
	hostsDir     = "hosts"
	importsDir   = "imports"
)

func metaPath(dir string) string     { return filepath.Join(dir, metaFile) }
func credsPath(dir string) string    { return filepath.Join(dir, credsFile) }
func findingsPath(dir string) string { return filepath.Join(dir, findingsFile) }

// validateIP rejects anything that is not a plain IP address. A zone ("%eth0")
// is rejected too because it can contain path separators.
func validateIP(ip string) error {
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return fmt.Errorf("invalid host ip %q: %w", ip, err)
	}
	if addr.Zone() != "" {
		return fmt.Errorf("invalid host ip %q: zones are not allowed", ip)
	}
	return nil
}

// hostDir returns hosts/<ip> under dir, or an error if ip is not an address.
func hostDir(dir, ip string) (string, error) {
	if err := validateIP(ip); err != nil {
		return "", err
	}
	return filepath.Join(dir, hostsDir, ip), nil
}

// hostPath returns the host.json path for ip.
func hostPath(dir, ip string) (string, error) {
	d, err := hostDir(dir, ip)
	if err != nil {
		return "", err
	}
	return filepath.Join(d, hostFile), nil
}

// notesPath returns the notes.md path for ip; an empty ip means the
// engagement-level notes.
func notesPath(dir, ip string) (string, error) {
	if ip == "" {
		return filepath.Join(dir, notesFile), nil
	}
	d, err := hostDir(dir, ip)
	if err != nil {
		return "", err
	}
	return filepath.Join(d, notesFile), nil
}

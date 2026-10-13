package store

import (
	"fmt"
	"net/netip"
	"path/filepath"
	"strings"
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

// validateIP rejects anything that is not a plain IP address and returns its
// canonical text form ("FE80::1" becomes "fe80::1"). A zone ("%eth0") is
// rejected because it can contain path separators.
func validateIP(ip string) (string, error) {
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return "", fmt.Errorf("invalid host ip %q: %w", ip, err)
	}
	if addr.Zone() != "" {
		return "", fmt.Errorf("invalid host ip %q: zones are not allowed", ip)
	}
	return addr.String(), nil
}

// hostDirName turns a canonical IP into its directory name. IPv6 colons
// become underscores because ":" is not allowed in Windows file names, and the
// same name is used on every platform so an engagement folder can move between
// machines. IPv4 text never contains "_", so ipFromDirName reverses it exactly.
func hostDirName(canon string) string { return strings.ReplaceAll(canon, ":", "_") }

// ipFromDirName reverses hostDirName.
func ipFromDirName(name string) string { return strings.ReplaceAll(name, "_", ":") }

// hostDir returns hosts/<hostDirName(ip)> under dir, or an error if ip is not an address.
func hostDir(dir, ip string) (string, error) {
	canon, err := validateIP(ip)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, hostsDir, hostDirName(canon)), nil
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

// Package importers detects scanner output formats and normalises them into
// a single Result type.
package importers

// ServiceResult is one open port reported by a scanner.
type ServiceResult struct {
	Port     int
	Protocol string
	Type     string
	Product  string
	Version  string
}

// WebPathResult is one discovered web path.
type WebPathResult struct {
	Path   string
	Status int
	Size   int
}

// FindingResult is one vulnerability or misconfiguration finding.
type FindingResult struct {
	Title    string
	Severity string
	Evidence string
	Port     int
}

// UserResult is one enumerated account.
type UserResult struct {
	Name   string
	Domain string
}

// ShareResult is one enumerated SMB share.
type ShareResult struct {
	Name   string
	Access string
}

// HostResult is everything a tool learned about one host.
type HostResult struct {
	IP       string // may be empty when the tool only knows a URL
	Hostname string
	OS       string
	URL      string // for web tools, used to resolve host and service
	Services []ServiceResult
	WebPaths []WebPathResult
	Findings []FindingResult
	Users    []UserResult
	Shares   []ShareResult
}

// Result is the normalised output of any importer.
type Result struct {
	Tool  string
	Hosts []HostResult
	// Dropped counts HostResults removed because a non-empty IP failed netip.ParseAddr.
	Dropped int
}

// Tool names, as returned by Detect and stored in Result.Tool.
const (
	ToolNmap         = "nmap"
	ToolMasscan      = "masscan"
	ToolFfuf         = "ffuf"
	ToolFeroxbuster  = "feroxbuster"
	ToolGobuster     = "gobuster"
	ToolNikto        = "nikto"
	ToolNuclei       = "nuclei"
	ToolEnum4linuxNG = "enum4linux-ng"
	ToolNetexec      = "netexec"
)

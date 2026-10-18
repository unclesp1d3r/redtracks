// Package model defines the RedTracks domain types and their on-disk JSON shape.
package model

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

// EngagementType selects the default flag set.
type EngagementType string

// Engagement types.
const (
	EngagementLab    EngagementType = "lab"
	EngagementOSCP   EngagementType = "oscp"
	EngagementOSAI   EngagementType = "osai"
	EngagementCustom EngagementType = "custom"
)

// IsValid reports whether t is one of the known engagement types.
func (t EngagementType) IsValid() bool {
	switch t {
	case EngagementLab, EngagementOSCP, EngagementOSAI, EngagementCustom:
		return true
	default:
		return false
	}
}

// HostStatus is how far the operator has progressed on a host.
type HostStatus string

// Host statuses.
const (
	HostStatusUntouched   HostStatus = "untouched"
	HostStatusEnumerating HostStatus = "enumerating"
	HostStatusFoothold    HostStatus = "foothold"
	HostStatusUser        HostStatus = "user"
	HostStatusRoot        HostStatus = "root"
	HostStatusDone        HostStatus = "done"
)

// IsValid reports whether s is one of the known host statuses.
func (s HostStatus) IsValid() bool {
	switch s {
	case HostStatusUntouched, HostStatusEnumerating, HostStatusFoothold, HostStatusUser, HostStatusRoot,
		HostStatusDone:
		return true
	default:
		return false
	}
}

// ItemStatus is the state of one checklist item.
type ItemStatus string

// Checklist item statuses.
const (
	ItemTodo    ItemStatus = "todo"
	ItemDone    ItemStatus = "done"
	ItemSkipped ItemStatus = "skipped"
	ItemNA      ItemStatus = "na"
)

// IsValid reports whether s is one of the known item statuses.
func (s ItemStatus) IsValid() bool {
	switch s {
	case ItemTodo, ItemDone, ItemSkipped, ItemNA:
		return true
	default:
		return false
	}
}

// Finding severities. Finding.Severity stays a string in the file contract.
const (
	SeverityInfo     = "info"
	SeverityLow      = "low"
	SeverityMedium   = "medium"
	SeverityHigh     = "high"
	SeverityCritical = "critical"
)

// Finding statuses. Finding.Status stays a string in the file contract.
const (
	FindingNoted         = "noted"
	FindingConfirmed     = "confirmed"
	FindingExploited     = "exploited"
	FindingFalsePositive = "false-positive"
)

// Credential kinds. Credential.Kind stays a string in the file contract.
const (
	CredKindPassword = "password"
	CredKindHash     = "hash"
	CredKindKey      = "key"
	CredKindToken    = "token"
)

// FlagType is a kind of flag an engagement awards points for.
type FlagType struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Points int    `json:"points"`
}

// EngagementMeta is the content of engagement.json. Domain is optional (e.g. corp.local).
type EngagementMeta struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Type      EngagementType `json:"type"`
	Domain    string         `json:"domain"`
	FlagTypes []FlagType     `json:"flagTypes"`
	CreatedAt time.Time      `json:"createdAt"`
}

// AccessPath records how a host is reached.
type AccessPath struct {
	ID           string `json:"id"`
	Method       string `json:"method"`
	ViaHostID    string `json:"viaHostId"`
	CredentialID string `json:"credentialId"`
	Command      string `json:"command"`
	Notes        string `json:"notes"`
}

// WebPath is one discovered web path.
type WebPath struct {
	Path   string `json:"path"`
	Status int    `json:"status"`
	Size   int    `json:"size"`
	Source string `json:"source"`
}

// ItemState is the recorded state of one checklist item. UpdatedAt is RFC3339,
// or "" when unset; a time.Time zero value would change the file contract.
type ItemState struct {
	Status    ItemStatus `json:"status"`
	Note      string     `json:"note"`
	UpdatedAt string     `json:"updatedAt"`
}

// ChecklistState maps checklist item IDs to their state.
type ChecklistState struct {
	Items map[string]ItemState `json:"items"`
}

// ProtocolTCP is the default transport for a service.
const ProtocolTCP = "tcp"

// Service is a network service on a host.
type Service struct {
	ID        string         `json:"id"`
	Port      int            `json:"port"`
	Protocol  string         `json:"protocol"`
	Type      string         `json:"type"`
	Product   string         `json:"product"`
	Version   string         `json:"version"`
	Notes     string         `json:"notes"`
	WebPaths  []WebPath      `json:"webPaths"`
	Checklist ChecklistState `json:"checklist"`
}

// Label renders the service as "port/proto type", without a trailing space
// when the type is unknown.
func (s Service) Label() string {
	return strings.TrimSpace(fmt.Sprintf("%d/%s %s", s.Port, s.Protocol, s.Type))
}

// Flag is a captured (or pending) flag.
type Flag struct {
	TypeID     string `json:"typeId"`
	Value      string `json:"value"`
	CapturedAt string `json:"capturedAt"` // RFC3339, or "" when unset
	Submitted  bool   `json:"submitted"`
}

// Host is the content of host.json.
type Host struct {
	ID          string         `json:"id"`
	IP          string         `json:"ip"`
	Hostname    string         `json:"hostname"`
	OS          string         `json:"os"`
	Status      HostStatus     `json:"status"`
	Tags        []string       `json:"tags"`
	AccessPaths []AccessPath   `json:"accessPaths"`
	Services    []Service      `json:"services"`
	Flags       []Flag         `json:"flags"`
	Checklist   ChecklistState `json:"checklist"`
}

// Finding is a vulnerability or observation.
type Finding struct {
	ID        string `json:"id"`
	HostID    string `json:"hostId"`
	ServiceID string `json:"serviceId"`
	Title     string `json:"title"`
	Severity  string `json:"severity"`
	Source    string `json:"source"`
	Evidence  string `json:"evidence"`
	Status    string `json:"status"`
}

// Credential is a username with a secret (password, hash, key or token).
type Credential struct {
	ID            string   `json:"id"`
	Username      string   `json:"username"`
	Secret        string   `json:"secret"`
	Kind          string   `json:"kind"`
	Domain        string   `json:"domain"`
	FoundOnHostID string   `json:"foundOnHostId"`
	WorksOn       []string `json:"worksOn"`
	TriedOn       []string `json:"triedOn"`
	Notes         string   `json:"notes"`
}

// User is an enumerated account.
type User struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Domain string `json:"domain"`
	HostID string `json:"hostId"`
	Notes  string `json:"notes"`
}

// Share is an enumerated network share.
type Share struct {
	ID     string `json:"id"`
	HostID string `json:"hostId"`
	Name   string `json:"name"`
	Access string `json:"access"`
	Notes  string `json:"notes"`
}

// Loot is a collected file or artifact.
type Loot struct {
	ID     string `json:"id"`
	HostID string `json:"hostId"`
	Label  string `json:"label"`
	Path   string `json:"path"`
	Notes  string `json:"notes"`
}

// Creds is the content of creds.json.
type Creds struct {
	Credentials []Credential `json:"credentials"`
	Users       []User       `json:"users"`
	Shares      []Share      `json:"shares"`
	Loot        []Loot       `json:"loot"`
}

// FindingsFile is the content of findings.json.
type FindingsFile struct {
	Findings []Finding `json:"findings"`
}

// Engagement is the in-memory view of an engagement folder.
type Engagement struct {
	Dir      string
	Meta     EngagementMeta
	Hosts    []Host
	Creds    Creds
	Findings []Finding
	// Evidence holds each host's evidence index, keyed by host IP.
	Evidence map[string]EvidenceFile
}

// Service returns the service with the given ID.
func (h Host) Service(id string) (Service, bool) {
	i := slices.IndexFunc(h.Services, func(s Service) bool { return s.ID == id })
	if i < 0 {
		return Service{}, false
	}
	return h.Services[i], true
}

// Host returns the host with the given IP.
func (e Engagement) Host(ip string) (Host, bool) {
	i := slices.IndexFunc(e.Hosts, func(h Host) bool { return h.IP == ip })
	if i < 0 {
		return Host{}, false
	}
	return e.Hosts[i], true
}

// HostByID returns the host with the given ID.
func (e Engagement) HostByID(id string) (Host, bool) {
	i := slices.IndexFunc(e.Hosts, func(h Host) bool { return h.ID == id })
	if i < 0 {
		return Host{}, false
	}
	return e.Hosts[i], true
}

func orEmpty[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

func (c ChecklistState) normalize() ChecklistState {
	if c.Items == nil {
		return ChecklistState{Items: map[string]ItemState{}}
	}
	return c
}

// Normalize returns a copy with nil slices and maps replaced by empty ones so they marshal as [] and {},
// an empty status defaulted to untouched, and an empty service protocol defaulted to tcp.
func (h Host) Normalize() Host {
	if h.Status == "" {
		h.Status = HostStatusUntouched // a hand-written host.json may omit it
	}
	h.Tags = orEmpty(h.Tags)
	h.AccessPaths = orEmpty(h.AccessPaths)
	h.Flags = orEmpty(h.Flags)
	h.Checklist = h.Checklist.normalize()
	services := make([]Service, len(h.Services))
	for i, s := range h.Services {
		if s.Protocol == "" {
			s.Protocol = ProtocolTCP // a hand-edited service may omit it; imports match on port and protocol
		}
		s.WebPaths = orEmpty(s.WebPaths)
		s.Checklist = s.Checklist.normalize()
		services[i] = s
	}
	h.Services = services
	return h
}

// Normalize returns a copy with nil slices replaced by empty ones so they marshal as [].
func (c Creds) Normalize() Creds {
	creds := make([]Credential, len(c.Credentials))
	for i, cr := range c.Credentials {
		cr.WorksOn = orEmpty(cr.WorksOn)
		cr.TriedOn = orEmpty(cr.TriedOn)
		creds[i] = cr
	}
	c.Credentials = creds
	c.Users = orEmpty(c.Users)
	c.Shares = orEmpty(c.Shares)
	c.Loot = orEmpty(c.Loot)
	return c
}

package model

import "time"

// EvidenceKind says how an evidence file is shown.
type EvidenceKind string

// Evidence kinds.
const (
	EvidenceKindImage EvidenceKind = "image"
	EvidenceKindText  EvidenceKind = "text"
)

// IsValid reports whether k is a known evidence kind.
func (k EvidenceKind) IsValid() bool { return k == EvidenceKindImage || k == EvidenceKindText }

// EvidenceLinkKind says what an evidence link points at.
type EvidenceLinkKind string

// Evidence link kinds.
const (
	EvidenceLinkSlot       EvidenceLinkKind = "slot"
	EvidenceLinkFinding    EvidenceLinkKind = "finding"
	EvidenceLinkFlag       EvidenceLinkKind = "flag"
	EvidenceLinkAccessPath EvidenceLinkKind = "accessPath"
)

// IsValid reports whether k is a known link kind.
func (k EvidenceLinkKind) IsValid() bool {
	switch k {
	case EvidenceLinkSlot, EvidenceLinkFinding, EvidenceLinkFlag, EvidenceLinkAccessPath:
		return true
	}
	return false
}

// EvidenceLink ties an evidence item to a record. TargetID may name a record
// that no longer exists; links are never enforced.
type EvidenceLink struct {
	Kind      EvidenceLinkKind `json:"kind"`
	TargetID  string           `json:"targetId"`
	ServiceID string           `json:"serviceId,omitempty"`
}

// ClipOf records which session log lines a clipped text item came from.
type ClipOf struct {
	LogID     string `json:"logId"`
	FirstLine int    `json:"firstLine"`
	LastLine  int    `json:"lastLine"`
}

// EvidenceItem indexes one file in a host's evidence directory.
type EvidenceItem struct {
	ID         string         `json:"id"`
	File       string         `json:"file"` // base name inside the evidence directory
	Kind       EvidenceKind   `json:"kind"`
	CapturedAt time.Time      `json:"capturedAt"`
	Caption    string         `json:"caption"`
	SHA256     string         `json:"sha256,omitempty"` // hex digest of the stored bytes
	Links      []EvidenceLink `json:"links"`
	ClipOf     *ClipOf        `json:"clipOf,omitempty"`
}

// EvidenceFile is the content of hosts/<ip>/evidence.json.
type EvidenceFile struct {
	Items []EvidenceItem `json:"items"`
}

// Normalize returns a copy with nil slices replaced by empty ones so they marshal as [].
func (f EvidenceFile) Normalize() EvidenceFile {
	items := make([]EvidenceItem, len(f.Items))
	for i, it := range f.Items {
		it.Links = orEmpty(it.Links)
		items[i] = it
	}
	f.Items = items
	return f
}

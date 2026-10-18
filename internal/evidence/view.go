package evidence

import (
	"bufio"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/unclesp1d3r/redtracks/internal/model"
)

const (
	// maxSearchLineBytes bounds one scanned chunk; a longer line is searched chunk by chunk.
	maxSearchLineBytes = 64 << 10
	// maxSnippetRunes is the longest snippet returned with a hit.
	maxSnippetRunes = 160
	// snippetLead is how many bytes of context precede the match in a snippet.
	snippetLead = 40
)

// SlotInfo is one proof slot that exists for a host: ServiceID is empty for host-level slots.
type SlotInfo struct {
	ID        string
	Title     string
	ServiceID string
}

// LinkView is an evidence link with its target resolved against the current host.
type LinkView struct {
	model.EvidenceLink

	Removed bool   `json:"removed"` // the target no longer exists
	Label   string `json:"label"`   // slot or finding title, flag type id, access path method
}

// ItemView is an evidence item with derived state.
type ItemView struct {
	model.EvidenceItem

	Missing bool       `json:"missing"` // the file is gone from the evidence directory
	Links   []LinkView `json:"links"`   // shadows EvidenceItem.Links
}

// View is one host's evidence in capture order plus files with no entry.
type View struct {
	Items      []ItemView `json:"items"`
	NotIndexed []string   `json:"notIndexed"`
}

// BuildInput is everything Build derives a View from.
type BuildInput struct {
	Host       model.Host
	Findings   []model.Finding
	Slots      []SlotInfo
	Items      []model.EvidenceItem
	NotIndexed []string
	Missing    []model.EvidenceItem
}

// Build resolves link labels and removed targets and orders items oldest first, ties by ID.
func Build(in BuildInput) View {
	missing := make(map[string]bool, len(in.Missing))
	for _, m := range in.Missing {
		missing[m.ID] = true
	}
	items := slices.Clone(in.Items)
	slices.SortStableFunc(items, func(a, b model.EvidenceItem) int {
		if c := a.CapturedAt.Compare(b.CapturedAt); c != 0 {
			return c
		}
		return strings.Compare(a.ID, b.ID)
	})
	views := make([]ItemView, len(items))
	for i, it := range items {
		links := make([]LinkView, len(it.Links))
		for j, l := range it.Links {
			links[j] = resolveLink(in, l)
		}
		views[i] = ItemView{EvidenceItem: it, Missing: missing[it.ID], Links: links}
	}
	return View{Items: views, NotIndexed: append([]string{}, in.NotIndexed...)}
}

func resolveLink(in BuildInput, l model.EvidenceLink) LinkView {
	label, ok := "", false
	switch l.Kind {
	case model.EvidenceLinkSlot:
		if i := slices.IndexFunc(in.Slots, func(s SlotInfo) bool {
			return s.ID == l.TargetID && (s.ServiceID == l.ServiceID || s.ServiceID == "")
		}); i >= 0 {
			label, ok = in.Slots[i].Title, true
		}
	case model.EvidenceLinkFinding:
		if i := slices.IndexFunc(in.Findings, func(f model.Finding) bool { return f.ID == l.TargetID }); i >= 0 {
			label, ok = in.Findings[i].Title, true
		}
	case model.EvidenceLinkFlag:
		if slices.ContainsFunc(in.Host.Flags, func(f model.Flag) bool { return f.TypeID == l.TargetID }) {
			label, ok = l.TargetID, true
		}
	case model.EvidenceLinkAccessPath:
		if i := slices.IndexFunc(
			in.Host.AccessPaths,
			func(p model.AccessPath) bool { return p.ID == l.TargetID },
		); i >= 0 {
			label, ok = in.Host.AccessPaths[i].Method, true
		}
	}
	return LinkView{EvidenceLink: l, Removed: !ok, Label: label}
}

// HostItems is one host's indexed items and where their files live.
type HostItems struct {
	IP    string
	Dir   string // the host's evidence directory
	Items []model.EvidenceItem
}

// Hit is one search match.
type Hit struct {
	IP      string `json:"ip"`
	ItemID  string `json:"itemId"`
	Snippet string `json:"snippet"`
}

// Search finds items whose caption or, for text items, file contents contain
// query, ignoring case. A blank query matches nothing; unreadable files are skipped.
func Search(hosts []HostItems, query string) []Hit {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return nil
	}
	var hits []Hit
	for _, h := range hosts {
		for _, it := range h.Items {
			if snippet, ok := matchItem(h.Dir, it, query); ok {
				hits = append(hits, Hit{IP: h.IP, ItemID: it.ID, Snippet: snippet})
			}
		}
	}
	return hits
}

func matchItem(dir string, it model.EvidenceItem, query string) (string, bool) {
	if strings.Contains(strings.ToLower(it.Caption), query) {
		return snippetAround(it.Caption, query), true
	}
	if it.Kind != model.EvidenceKindText || it.File != filepath.Base(it.File) || strings.HasPrefix(it.File, ".") {
		return "", false
	}
	return searchFile(filepath.Join(dir, it.File), query)
}

func searchFile(path, query string) (string, bool) {
	if info, err := os.Lstat(path); err != nil || !info.Mode().IsRegular() {
		return "", false
	}
	f, err := os.Open(path)
	if err != nil {
		return "", false
	}
	defer func() { _ = f.Close() }()
	r := bufio.NewReaderSize(f, maxSearchLineBytes)
	for {
		chunk, _, err := r.ReadLine()
		if strings.Contains(strings.ToLower(string(chunk)), query) {
			return snippetAround(string(chunk), query), true
		}
		if err != nil { // io.EOF ends the scan; any other read error skips the file
			return "", false
		}
	}
}

// snippetAround trims line and keeps a window of it around the first match of query (already lower case).
func snippetAround(line, query string) string {
	line = strings.TrimSpace(line)
	lower := strings.ToLower(line)
	start := 0
	if len(lower) == len(line) {
		start = max(strings.Index(lower, query)-snippetLead, 0)
		for start > 0 && !utf8.RuneStart(line[start]) {
			start--
		}
	}
	runes := []rune(line[start:])
	if len(runes) > maxSnippetRunes {
		runes = runes[:maxSnippetRunes]
	}
	return string(runes)
}

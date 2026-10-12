package progress

import (
	"regexp"
	"strings"

	"github.com/unclesp1d3r/redtracks/internal/model"
)

const (
	// classPrefix keeps status class names clear of Mermaid's own classes such as
	// "root", whose top-level group would otherwise inherit a status fill.
	classPrefix = "st_"
	// maxLabelRunes is the longest label emitted into the graph.
	maxLabelRunes = 60
	attackerNode  = "attacker"
	nodePrefix    = "h_"
)

// statusFills lists every host status with its fill colour, in output order.
var statusFills = []struct {
	status model.HostStatus
	fill   string
}{
	{model.HostStatusUntouched, "#3b3f4a"},
	{model.HostStatusEnumerating, "#1d4ed8"},
	{model.HostStatusFoothold, "#b45309"},
	{model.HostStatusUser, "#a16207"},
	{model.HostStatusRoot, "#b91c1c"},
	{model.HostStatusDone, "#15803d"},
}

var (
	nonIDChars    = regexp.MustCompile(`[^A-Za-z0-9]`)
	labelStripper = strings.NewReplacer("<", "", ">", "", "\r", "", "\n", "")
	quoteEscaper  = strings.NewReplacer(`"`, "#quot;")
)

// AttackGraph renders the engagement's access paths as a Mermaid flowchart.
// Hosts without an access path appear unconnected.
func AttackGraph(eng model.Engagement) string {
	var sb strings.Builder
	sb.WriteString("flowchart LR\n")
	sb.WriteString("  " + attackerNode + "[" + quote(attackerNode) + "]\n")

	nodeByHostID := make(map[string]string, len(eng.Hosts))
	for _, h := range eng.Hosts {
		id := nodeID(h.IP)
		nodeByHostID[h.ID] = id
		sb.WriteString("  " + id + "[" + quote(hostLabel(h)) + "]\n")
	}
	for _, h := range eng.Hosts {
		for _, p := range h.AccessPaths {
			from, ok := nodeByHostID[p.ViaHostID]
			if !ok {
				from = attackerNode
			}
			sb.WriteString("  " + from + edge(p.Method) + nodeID(h.IP) + "\n")
		}
	}
	for _, h := range eng.Hosts {
		if h.Status != "" {
			sb.WriteString("  class " + nodeID(h.IP) + " " + classPrefix + string(h.Status) + "\n")
		}
	}
	for _, sf := range statusFills {
		sb.WriteString("  classDef " + classPrefix + string(sf.status) + " fill:" + sf.fill + "\n")
	}
	return sb.String()
}

func nodeID(ip string) string {
	return nodePrefix + nonIDChars.ReplaceAllString(ip, "_")
}

func hostLabel(h model.Host) string {
	if h.Hostname == "" {
		return h.IP
	}
	return h.IP + " (" + h.Hostname + ")"
}

func edge(method string) string {
	if strings.TrimSpace(method) == "" {
		return " --> "
	}
	return " -->|" + quote(method) + "| "
}

// quote makes s safe as a Mermaid quoted label: angle brackets and newlines
// are dropped, the text is cut to maxLabelRunes, and double quotes become #quot;.
func quote(s string) string {
	s = labelStripper.Replace(s)
	if r := []rune(s); len(r) > maxLabelRunes {
		s = string(r[:maxLabelRunes])
	}
	return `"` + quoteEscaper.Replace(s) + `"`
}

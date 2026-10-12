package importers

import (
	"fmt"
	"net/netip"
	"regexp"
	"slices"
)

// ansiEscape matches terminal colour sequences; text tools may emit them.
var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)

// parsers maps a tool name to its parser. Adding a tool means a sniff rule in
// detect.go and an entry here.
var parsers = map[string]func([]byte) (Result, error){
	ToolNmap:         parseNmap,
	ToolMasscan:      parseMasscan,
	ToolGobuster:     parseGobuster,
	ToolFfuf:         parseFfuf,
	ToolFeroxbuster:  parseFeroxbuster,
	ToolNikto:        parseNikto,
	ToolNuclei:       parseNuclei,
	ToolEnum4linuxNG: parseEnum4linux,
	ToolNetexec:      parseNetexec,
}

// Parse strips ANSI colour from data, detects its format, runs the matching
// parser, and drops hosts whose non-empty IP is not a valid address (zoned
// IPv6 included); kept IPs are canonicalised.
func Parse(data []byte) (Result, error) {
	data = ansiEscape.ReplaceAll(data, nil)
	tool, err := Detect(data)
	if err != nil {
		return Result{}, err
	}
	parse, ok := parsers[tool]
	if !ok {
		return Result{}, fmt.Errorf("detected %s but no parser is registered", tool)
	}
	res, err := parse(data)
	if err != nil {
		return Result{}, fmt.Errorf("parse %s: %w", tool, err)
	}
	return dropInvalidHosts(res), nil
}

func dropInvalidHosts(res Result) Result {
	kept := make([]HostResult, 0, len(res.Hosts))
	dropped := res.Dropped
	droppedIPs := slices.Clone(res.DroppedIPs)
	for _, h := range res.Hosts {
		if h.IP != "" {
			addr, err := netip.ParseAddr(h.IP)
			if err != nil || addr.Zone() != "" {
				dropped++
				droppedIPs = append(droppedIPs, h.IP)
				continue
			}
			h.IP = addr.String()
		}
		kept = append(kept, h)
	}
	return Result{Tool: res.Tool, Hosts: kept, Dropped: dropped, DroppedIPs: droppedIPs, Skipped: res.Skipped}
}

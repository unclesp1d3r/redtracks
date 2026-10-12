package importers

import (
	"fmt"
	"net/netip"
)

// parsers maps a tool name to its parser; each tool's file registers itself here.
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

// Parse detects the format of data, runs the matching parser, and drops hosts
// whose non-empty IP is not a valid address.
func Parse(data []byte) (Result, error) {
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
	for _, h := range res.Hosts {
		if h.IP != "" {
			if _, err := netip.ParseAddr(h.IP); err != nil {
				dropped++
				continue
			}
		}
		kept = append(kept, h)
	}
	return Result{Tool: res.Tool, Hosts: kept, Dropped: dropped}
}

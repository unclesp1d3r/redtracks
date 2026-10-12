package importers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
)

type feroxLine struct {
	Type          string `json:"type"`
	URL           string `json:"url"`
	Status        int    `json:"status"`
	ContentLength int    `json:"content_length"`
}

// parseFeroxbuster reads JSONL, keeping only "response" lines and grouping
// them by web base, in first-seen order.
func parseFeroxbuster(data []byte) (Result, error) {
	index := map[string]int{}
	var hosts []HostResult
	for n, raw := range bytes.Split(data, []byte("\n")) {
		if len(bytes.TrimSpace(raw)) == 0 {
			continue
		}
		var l feroxLine
		if err := json.Unmarshal(raw, &l); err != nil {
			return Result{}, fmt.Errorf("decode feroxbuster line %d: %w", n+1, err)
		}
		if l.Type != "response" {
			continue
		}
		u, err := url.Parse(l.URL)
		if err != nil {
			return Result{}, fmt.Errorf("feroxbuster line %d: %w", n+1, err)
		}
		base, err := webBase(l.URL)
		if err != nil {
			return Result{}, fmt.Errorf("feroxbuster line %d: %w", n+1, err)
		}
		i, seen := index[base.URL]
		if !seen {
			i = len(hosts)
			index[base.URL] = i
			hosts = append(hosts, base)
		}
		hosts[i].WebPaths = append(
			hosts[i].WebPaths,
			WebPathResult{Path: u.Path, Status: l.Status, Size: l.ContentLength},
		)
	}
	return Result{Tool: ToolFeroxbuster, Hosts: hosts}, nil
}

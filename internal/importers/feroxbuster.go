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

// parseFeroxbuster reads JSONL, keeping only "response" lines. The host URL
// comes from the first response.
func parseFeroxbuster(data []byte) (Result, error) {
	var host HostResult
	found := false
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
		if !found {
			if host, err = webBase(l.URL); err != nil {
				return Result{}, fmt.Errorf("feroxbuster line %d: %w", n+1, err)
			}
			found = true
		}
		host.WebPaths = append(host.WebPaths, WebPathResult{Path: u.Path, Status: l.Status, Size: l.ContentLength})
	}
	if !found {
		return Result{Tool: ToolFeroxbuster}, nil
	}
	return Result{Tool: ToolFeroxbuster, Hosts: []HostResult{host}}, nil
}

package importers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

type nucleiLine struct {
	Info struct {
		Name     string `json:"name"`
		Severity string `json:"severity"`
	} `json:"info"`
	Host      string      `json:"host"`
	MatchedAt string      `json:"matched-at"`
	Port      json.Number `json:"port"`
}

// parseNuclei reads JSONL and groups findings by host, in first-seen order.
func parseNuclei(data []byte) (Result, error) {
	index := map[string]int{}
	var hosts []HostResult
	for n, raw := range bytes.Split(data, []byte("\n")) {
		if len(bytes.TrimSpace(raw)) == 0 {
			continue
		}
		var l nucleiLine
		if err := json.Unmarshal(raw, &l); err != nil {
			return Result{}, fmt.Errorf("decode nuclei line %d: %w", n+1, err)
		}
		port, err := numberToInt(l.Port)
		if err != nil {
			return Result{}, fmt.Errorf("nuclei line %d: %w", n+1, err)
		}
		base, err := nucleiHost(l.Host)
		if err != nil {
			return Result{}, fmt.Errorf("nuclei line %d: %w", n+1, err)
		}
		key := base.IP + base.URL
		i, seen := index[key]
		if !seen {
			i = len(hosts)
			index[key] = i
			hosts = append(hosts, base)
		}
		hosts[i].Findings = append(hosts[i].Findings, FindingResult{
			Title: l.Info.Name, Severity: strings.ToLower(l.Info.Severity), Evidence: l.MatchedAt, Port: port,
		})
	}
	return Result{Tool: ToolNuclei, Hosts: hosts}, nil
}

// nucleiHost resolves a nuclei host (URL, host:port, or bare host) to a
// HostResult with IP for IP literals and URL otherwise.
func nucleiHost(host string) (HostResult, error) {
	if !strings.Contains(host, "://") {
		host = "http://" + host
	}
	h, err := webBase(host)
	if err != nil {
		return HostResult{}, err
	}
	if h.IP != "" {
		h.URL = ""
	}
	return h, nil
}

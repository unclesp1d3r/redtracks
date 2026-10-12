package importers

import (
	"encoding/json"
	"fmt"
)

const severityInfo = "info"

type niktoOutput struct {
	Host            string      `json:"host"`
	IP              string      `json:"ip"`
	Port            json.Number `json:"port"` // nikto emits the port as a string
	Vulnerabilities []niktoVuln `json:"vulnerabilities"`
}

type niktoVuln struct {
	URL string `json:"url"`
	Msg string `json:"msg"`
}

func parseNikto(data []byte) (Result, error) {
	var out niktoOutput
	if err := json.Unmarshal(data, &out); err != nil {
		return Result{}, fmt.Errorf("decode nikto JSON: %w", err)
	}
	port, err := numberToInt(out.Port)
	if err != nil {
		return Result{}, fmt.Errorf("nikto port: %w", err)
	}
	host := HostResult{IP: out.IP}
	for _, v := range out.Vulnerabilities {
		host.Findings = append(host.Findings, FindingResult{
			Title: v.Msg, Severity: severityInfo, Evidence: v.URL, Port: port,
		})
	}
	return Result{Tool: ToolNikto, Hosts: []HostResult{host}}, nil
}

// numberToInt converts an optional JSON number; an absent value is 0.
func numberToInt(n json.Number) (int, error) {
	if n == "" {
		return 0, nil
	}
	v, err := n.Int64()
	if err != nil {
		return 0, fmt.Errorf("invalid number %q: %w", n, err)
	}
	return int(v), nil
}

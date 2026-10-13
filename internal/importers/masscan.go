package importers

import (
	"encoding/json"
	"fmt"
)

type masscanEntry struct {
	IP    string        `json:"ip"`
	Ports []masscanPort `json:"ports"`
}

type masscanPort struct {
	Port  int    `json:"port"`
	Proto string `json:"proto"`
}

// parseMasscan merges entries by IP, since masscan emits one entry per port.
func parseMasscan(data []byte) (Result, error) {
	var entries []masscanEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return Result{}, fmt.Errorf("decode masscan JSON: %w", err)
	}
	index := map[string]int{}
	hosts := make([]HostResult, 0, len(entries))
	for _, e := range entries {
		i, seen := index[e.IP]
		if !seen {
			i = len(hosts)
			index[e.IP] = i
			hosts = append(hosts, HostResult{IP: e.IP})
		}
		for _, p := range e.Ports {
			hosts[i].Services = append(hosts[i].Services, ServiceResult{Port: p.Port, Protocol: p.Proto})
		}
	}
	return Result{Tool: ToolMasscan, Hosts: hosts}, nil
}

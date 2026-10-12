package importers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"slices"
	"strings"
	"unicode"
)

type enum4linuxUser struct {
	Username string `json:"username"`
}

type enum4linuxOutput struct {
	Target struct {
		Host string `json:"host"`
	} `json:"target"`
	Domain string `json:"domain"`
	OSInfo struct {
		OS string `json:"os"`
	} `json:"os_info"`
	Users  map[string]enum4linuxUser `json:"users"`
	Shares map[string]struct {
		Access struct {
			Mapping string `json:"mapping"`
			Listing string `json:"listing"`
		} `json:"access"`
	} `json:"shares"`
}

func parseEnum4linux(data []byte) (Result, error) {
	var out enum4linuxOutput
	if err := json.Unmarshal(data, &out); err != nil {
		return Result{}, fmt.Errorf("decode enum4linux-ng JSON: %w", err)
	}
	if out.Target.Host == "" {
		return Result{}, errors.New("enum4linux-ng output has no target.host")
	}
	host := HostResult{OS: guessOS(out.OSInfo.OS)}
	if _, err := netip.ParseAddr(out.Target.Host); err == nil {
		host.IP = out.Target.Host
	} else {
		host.Hostname = out.Target.Host
	}
	domain := cleanDomain(out.Domain)
	for _, rid := range sortedKeys(out.Users, ridOrder) {
		host.Users = append(host.Users, UserResult{Name: out.Users[rid].Username, Domain: domain})
	}
	for _, name := range sortedKeys(out.Shares, strings.Compare) {
		a := out.Shares[name].Access
		host.Shares = append(host.Shares, ShareResult{Name: name, Access: a.Mapping + "/" + a.Listing})
	}
	return Result{Tool: ToolEnum4linuxNG, Hosts: []HostResult{host}}, nil
}

// sortedKeys gives map iteration a stable order so imports are deterministic.
func sortedKeys[V any](m map[string]V, order func(a, b string) int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.SortFunc(keys, order)
	return keys
}

// cleanDomain returns "" for a domain containing whitespace or control characters.
func cleanDomain(d string) string {
	for _, r := range d {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return ""
		}
	}
	return d
}

// ridOrder sorts numeric RIDs numerically ("500" before "1103").
func ridOrder(a, b string) int {
	if len(a) != len(b) {
		return len(a) - len(b)
	}
	return strings.Compare(a, b)
}

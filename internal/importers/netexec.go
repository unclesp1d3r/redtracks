package importers

import (
	"errors"
	"regexp"
	"strings"
)

var (
	ansiEscape  = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)
	netexecRow  = regexp.MustCompile(`^(?:SMB|LDAP)\s+(\S+)\s+\d+\s+(\S+)\s+(.*)$`)
	netexecName = regexp.MustCompile(`\(name:([^)\s]+)\)`)
	netexecPerm = regexp.MustCompile(`^(?:READ|WRITE)(?:,(?:READ|WRITE))*$`)
)

const (
	netexecCredPrefix   = "[+] "
	netexecBannerPrefix = "[*] "
	netexecMinShareCols = 2
)

// parseNetexec reads SMB/LDAP text output, grouping rows by IP in first-seen order.
func parseNetexec(data []byte) (Result, error) {
	index := map[string]int{}
	var hosts []HostResult
	for _, line := range strings.Split(string(ansiEscape.ReplaceAll(data, nil)), "\n") {
		m := netexecRow.FindStringSubmatch(strings.TrimRight(line, "\r"))
		if m == nil {
			continue
		}
		ip, name, rest := m[1], m[2], strings.TrimSpace(m[3])
		i, seen := index[ip]
		if !seen {
			i = len(hosts)
			index[ip] = i
			hosts = append(hosts, HostResult{IP: ip, Hostname: name})
		}
		hosts[i] = applyNetexecRow(hosts[i], rest)
	}
	if len(hosts) == 0 {
		return Result{}, errors.New("no netexec SMB/LDAP result lines found")
	}
	return Result{Tool: ToolNetexec, Hosts: hosts}, nil
}

// applyNetexecRow returns h updated with whatever the row's text reveals.
func applyNetexecRow(h HostResult, rest string) HostResult {
	switch {
	case strings.HasPrefix(rest, netexecBannerPrefix):
		if m := netexecName.FindStringSubmatch(rest); m != nil {
			h.Hostname = m[1]
		}
		if os := guessOS(rest); os != "" {
			h.OS = os
		}
	case strings.HasPrefix(rest, netexecCredPrefix):
		if u, ok := netexecUser(strings.TrimPrefix(rest, netexecCredPrefix)); ok {
			h.Users = append(h.Users, u)
		}
	case !strings.HasPrefix(rest, "["):
		f := strings.Fields(rest)
		if len(f) >= netexecMinShareCols && netexecPerm.MatchString(f[1]) {
			h.Shares = append(h.Shares, ShareResult{Name: f[0], Access: f[1]})
		}
	}
	return h
}

// netexecUser parses "domain\user:password" and discards the password.
func netexecUser(s string) (UserResult, bool) {
	cred, _, _ := strings.Cut(s, ":")
	domain, user, hasDomain := strings.Cut(strings.Fields(cred + " ")[0], `\`)
	if !hasDomain {
		domain, user = "", domain
	}
	if user == "" {
		return UserResult{}, false
	}
	return UserResult{Name: user, Domain: cleanDomain(domain)}, true
}

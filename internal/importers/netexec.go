package importers

import (
	"errors"
	"regexp"
	"strings"
)

var (
	netexecRow  = regexp.MustCompile(`^(SMB|LDAP)\s+(\S+)\s+\d+\s+(\S+)\s+(.*)$`)
	netexecName = regexp.MustCompile(`\(name:([^)\s]+)\)`)
	netexecPerm = regexp.MustCompile(`^(?:READ|WRITE)(?:,(?:READ|WRITE))*$`)
)

const (
	netexecCredPrefix   = "[+] "
	netexecBannerPrefix = "[*] "
	netexecProtoSMB     = "SMB"
	netexecShareHeader  = "Share"
)

// parseNetexec reads SMB/LDAP text output, grouping rows by IP in first-seen order.
func parseNetexec(data []byte) (Result, error) {
	index := map[string]int{}
	var hosts []HostResult
	for line := range strings.SplitSeq(string(data), "\n") {
		m := netexecRow.FindStringSubmatch(strings.TrimRight(line, "\r"))
		if m == nil {
			continue
		}
		proto, ip, name, rest := m[1], m[2], m[3], strings.TrimSpace(m[4])
		i, seen := index[ip]
		if !seen {
			i = len(hosts)
			index[ip] = i
			hosts = append(hosts, HostResult{IP: ip, Hostname: name})
		}
		hosts[i] = applyNetexecRow(hosts[i], proto == netexecProtoSMB, rest)
	}
	if len(hosts) == 0 {
		return Result{}, errors.New("no netexec SMB/LDAP result lines found")
	}
	return Result{Tool: ToolNetexec, Hosts: hosts}, nil
}

// applyNetexecRow returns h updated with whatever the row's text reveals.
func applyNetexecRow(h HostResult, isSMB bool, rest string) HostResult {
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
	case isSMB && !strings.HasPrefix(rest, "["):
		if sh, ok := netexecShare(rest); ok {
			h.Shares = append(h.Shares, sh)
		}
	}
	return h
}

// netexecShare parses a share table row. A share whose permission column is
// empty (the next word is part of the remark) is kept with Access "".
func netexecShare(rest string) (ShareResult, bool) {
	f := strings.Fields(rest)
	if len(f) == 0 || strings.Trim(f[0], "-") == "" || f[0] == netexecShareHeader {
		return ShareResult{}, false // blank row, "-----" rule or the table header
	}
	if len(f) > 1 && netexecPerm.MatchString(f[1]) {
		return ShareResult{Name: f[0], Access: f[1]}, true
	}
	return ShareResult{Name: f[0]}, true
}

// netexecUser parses "domain\user:password" and discards the password.
func netexecUser(s string) (UserResult, bool) {
	cred, _, _ := strings.Cut(s, ":")
	fields := strings.Fields(cred)
	if len(fields) == 0 {
		return UserResult{}, false
	}
	domain, user, hasDomain := strings.Cut(fields[0], `\`)
	if !hasDomain {
		domain, user = "", domain
	}
	if user == "" {
		return UserResult{}, false
	}
	return UserResult{Name: user, Domain: cleanDomain(domain)}, true
}

package importers

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// sniffBytes is how much of a file Detect examines.
const sniffBytes = 8 * 1024

// ErrUnknownFormat is returned when no sniff rule matches.
var ErrUnknownFormat = errors.New("unknown format")

type sniffRule struct {
	tool  string
	match func(head []byte) bool
}

var (
	gobusterLine = regexp.MustCompile(`(?m)^/\S+\s+\(Status: \d+\)`)
	netexecLine  = regexp.MustCompile(`(?m)^(?:SMB|LDAP)\s+\S+\s+\d+`)
	feroxType    = regexp.MustCompile(`"type"\s*:\s*"response"`)
)

// sniffRules is ordered; the first match wins.
var sniffRules = []sniffRule{
	{ToolNmap, sniffNmap},
	{ToolMasscan, sniffMasscan},
	{ToolFfuf, sniffFfuf},
	{ToolFeroxbuster, sniffFeroxbuster},
	{ToolGobuster, sniffGobuster},
	{ToolNikto, sniffNikto},
	{ToolNuclei, sniffNuclei},
	{ToolEnum4linuxNG, sniffEnum4linux},
	{ToolNetexec, sniffNetexec},
}

// Detect names the tool that produced head, the first few KB of a file.
func Detect(head []byte) (string, error) {
	if len(head) > sniffBytes {
		head = head[:sniffBytes]
	}
	tried := make([]string, 0, len(sniffRules))
	for _, r := range sniffRules {
		if r.match(head) {
			return r.tool, nil
		}
		tried = append(tried, r.tool)
	}
	return "", fmt.Errorf("%w: tried %s", ErrUnknownFormat, strings.Join(tried, ", "))
}

func sniffNmap(head []byte) bool { return bytes.Contains(head, []byte("<nmaprun")) }

func sniffMasscan(head []byte) bool {
	return bytes.HasPrefix(bytes.TrimSpace(head), []byte("[")) && bytes.Contains(head, []byte(`"ports"`))
}

func sniffFfuf(head []byte) bool {
	return bytes.Contains(head, []byte(`"commandline"`)) && bytes.Contains(head, []byte(`"results"`))
}

func sniffFeroxbuster(head []byte) bool { return feroxType.Match(head) }

func sniffGobuster(head []byte) bool { return gobusterLine.Match(head) }

func sniffNikto(head []byte) bool {
	return bytes.Contains(head, []byte(`"vulnerabilities"`)) && bytes.Contains(head, []byte(`"host"`))
}

func sniffNuclei(head []byte) bool { return bytes.Contains(head, []byte(`"template-id"`)) }

func sniffEnum4linux(head []byte) bool {
	return bytes.Contains(head, []byte(`"target"`)) &&
		(bytes.Contains(head, []byte(`"users"`)) || bytes.Contains(head, []byte(`"shares"`)))
}

func sniffNetexec(head []byte) bool { return netexecLine.Match(head) }

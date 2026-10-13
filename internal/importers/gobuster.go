package importers

import (
	"errors"
	"regexp"
	"strconv"
)

var gobusterResult = regexp.MustCompile(`(?m)^(/\S*)\s+\(Status: (\d+)\)(?:\s+\[Size: (\d+)\])?`)

// parseGobuster reads dir-mode text output. The output carries no base URL,
// so the host has neither IP nor URL and is resolved by the operator.
func parseGobuster(data []byte) (Result, error) {
	matches := gobusterResult.FindAllSubmatch(data, -1)
	if len(matches) == 0 {
		return Result{}, errors.New("no gobuster result lines found")
	}
	paths := make([]WebPathResult, 0, len(matches))
	for _, m := range matches {
		paths = append(paths, WebPathResult{Path: string(m[1]), Status: atoiOrZero(m[2]), Size: atoiOrZero(m[3])})
	}
	return Result{Tool: ToolGobuster, Hosts: []HostResult{{WebPaths: paths}}}, nil
}

// atoiOrZero maps an absent (empty) capture to 0; the regexp guarantees digits otherwise.
func atoiOrZero(b []byte) int {
	n, err := strconv.Atoi(string(b))
	if err != nil {
		return 0
	}
	return n
}

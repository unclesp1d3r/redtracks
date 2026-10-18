// Package checklist loads per-service and per-host enumeration checklists,
// matches them against hosts and services, merges them with recorded state,
// and substitutes target placeholders into their commands.
package checklist

import (
	"context"
	"encoding/json"
	"fmt"
	"net/netip"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/unclesp1d3r/redtracks/internal/model"
	"github.com/unclesp1d3r/redtracks/internal/store"
)

// Item is one checklist step.
type Item struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Hint     string   `json:"hint"`
	Commands []string `json:"commands"`
}

// Match describes which services or hosts a definition applies to.
type Match struct {
	ServiceTypes []string `json:"serviceTypes"`
	HostOS       []string `json:"hostOS"`
}

// Definition is one checklist file's content, plus the file it came from.
type Definition struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Match Match  `json:"match"`
	Items []Item `json:"items"`
	File  string `json:"-"`
}

// ItemView pairs a checklist item with its recorded state.
type ItemView struct {
	Item

	State model.ItemState
}

// domainPattern restricts substituted domains to hostname-safe characters.
var domainPattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

// LoadDefinitions reads every *.json file in dir, sorted by file name, and
// parses it into a Definition. An item ID shared by two files is an error
// naming both files. Invalid JSON reports the file it came from.
func LoadDefinitions(dir string) ([]Definition, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, fmt.Errorf("glob %s: %w", dir, err)
	}
	sort.Strings(paths)

	defs := make([]Definition, 0, len(paths))
	seen := make(map[string]string, len(paths))
	defFiles := make(map[string]string, len(paths))
	for _, path := range paths {
		def, err := loadOne(path)
		if err != nil {
			return nil, err
		}
		base := filepath.Base(path)
		if err := validateDefinition(def, base); err != nil {
			return nil, err
		}
		if prior, ok := defFiles[def.ID]; ok {
			return nil, fmt.Errorf("duplicate definition id %q in %s and %s", def.ID, prior, base)
		}
		defFiles[def.ID] = base
		for _, item := range def.Items {
			if prior, ok := seen[item.ID]; ok {
				return nil, fmt.Errorf("duplicate item id %q in %s and %s", item.ID, prior, base)
			}
			seen[item.ID] = base
		}
		defs = append(defs, def)
	}
	return defs, nil
}

// validateDefinition rejects a definition that could never be matched or
// whose items cannot be addressed.
func validateDefinition(def Definition, file string) error {
	if def.ID == "" {
		return fmt.Errorf("%s: definition has an empty id", file)
	}
	if len(def.Match.ServiceTypes) == 0 && len(def.Match.HostOS) == 0 {
		return fmt.Errorf("%s: definition %q matches nothing (serviceTypes and hostOS are both empty)", file, def.ID)
	}
	for i, item := range def.Items {
		if item.ID == "" {
			return fmt.Errorf("%s: definition %q item %d has an empty id", file, def.ID, i+1)
		}
	}
	return nil
}

func loadOne(path string) (Definition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Definition{}, fmt.Errorf("read %s: %w", path, err)
	}
	var def Definition
	if err := json.Unmarshal(data, &def); err != nil {
		return Definition{}, fmt.Errorf("parse %s: %w", path, err)
	}
	def.File = filepath.Base(path)
	return def, nil
}

// ForService returns the definitions whose match.serviceTypes include s.Type,
// compared case-insensitively.
func ForService(defs []Definition, s model.Service) []Definition {
	out := make([]Definition, 0, len(defs))
	for _, def := range defs {
		if slices.ContainsFunc(def.Match.ServiceTypes, func(v string) bool { return strings.EqualFold(v, s.Type) }) {
			out = append(out, def)
		}
	}
	return out
}

// ForHost returns the definitions whose match.hostOS include h.OS, compared
// case-insensitively.
func ForHost(defs []Definition, h model.Host) []Definition {
	out := make([]Definition, 0, len(defs))
	for _, def := range defs {
		if slices.ContainsFunc(def.Match.HostOS, func(v string) bool { return strings.EqualFold(v, h.OS) }) {
			out = append(out, def)
		}
	}
	return out
}

// Merge returns one ItemView per item in def, in order. An item with no
// recorded state is todo. State for IDs not present in def is ignored.
func Merge(def Definition, state model.ChecklistState) []ItemView {
	views := make([]ItemView, len(def.Items))
	for i, item := range def.Items {
		itemState, ok := state.Items[item.ID]
		if !ok {
			itemState = model.ItemState{Status: model.ItemTodo}
		}
		views[i] = ItemView{Item: item, State: itemState}
	}
	return views
}

// Substitute replaces $T with targetIP and $D with domain in cmd. targetIP
// must parse as an IP address. domain, if non-empty, must match
// ^[A-Za-z0-9._-]+$ and must not start with "-" (it would read as a flag).
// An IPv6 zone is rejected. A cmd that uses $D with an empty domain is an
// error rather than a command with a hole in it.
func Substitute(cmd, targetIP, domain string) (string, error) {
	addr, err := netip.ParseAddr(targetIP)
	if err != nil {
		return "", fmt.Errorf("invalid target IP %q: %w", targetIP, err)
	}
	if addr.Zone() != "" {
		return "", fmt.Errorf("invalid target IP %q: zones are not allowed", targetIP)
	}
	if domain != "" && (strings.HasPrefix(domain, "-") || !domainPattern.MatchString(domain)) {
		return "", fmt.Errorf("invalid domain %q", domain)
	}
	if domain == "" && strings.Contains(cmd, "$D") {
		return "", fmt.Errorf("command uses $D but the engagement has no domain: %q", cmd)
	}
	out := strings.ReplaceAll(cmd, "$T", targetIP)
	out = strings.ReplaceAll(out, "$D", domain)
	return out, nil
}

// Watch polls dir for changes to its *.json files every `every` and calls
// onChange with the freshly loaded set whenever it changes. A load error
// keeps the previous set and calls onError instead. Watch returns when ctx
// is done.
func Watch(ctx context.Context, dir string, every time.Duration, onChange func([]Definition), onError func(error)) {
	ticker := time.NewTicker(every)
	defer ticker.Stop()

	var lastSignature string
	check := func() {
		signature, err := dirSignature(dir)
		if err != nil {
			onError(err)
			return
		}
		if signature == lastSignature {
			return
		}
		defs, err := LoadDefinitions(dir)
		if err != nil {
			lastSignature = signature
			onError(err)
			return
		}
		lastSignature = signature
		onChange(defs)
	}

	check()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			check()
		}
	}
}

// dirSignature builds a string that changes whenever the set of *.json files
// in dir, their sizes, or their modification times change.
func dirSignature(dir string) (string, error) {
	return store.DirSignature(dir, func(name string) bool { return strings.HasSuffix(name, ".json") })
}

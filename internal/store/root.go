package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/unclesp1d3r/redtracks/internal/model"
)

const (
	rootEnv         = "REDTRACKS_HOME"
	rootDefaultName = "RedTracks"
	engagementsDir  = "engagements"
)

// nonSlug matches each run of characters that cannot appear in a folder slug.
var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

// RootDir returns $REDTRACKS_HOME, or ~/RedTracks when it is unset. It fails
// when there is no home directory rather than guessing a relative path.
func RootDir() (string, error) {
	if v := os.Getenv(rootEnv); v != "" {
		return v, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find home directory (set %s to choose a root): %w", rootEnv, err)
	}
	return filepath.Join(home, rootDefaultName), nil
}

// Summary describes one engagement folder for the picker.
type Summary struct {
	Dir       string
	Name      string
	Type      model.EngagementType
	Hosts     int
	CreatedAt time.Time
}

// BrokenEngagement is an engagement folder that could not be summarised.
type BrokenEngagement struct {
	Dir   string
	Error string
}

// ListEngagements scans root/engagements, newest first. A missing directory is
// an empty list; folders without engagement.json are skipped. Folders that
// cannot be read or parsed are left out and reported in the error, alongside
// the rest of the list.
func ListEngagements(root string) ([]Summary, error) {
	sums, broken, err := ListEngagementsDetailed(root)
	if err != nil {
		return nil, err
	}
	errs := make([]error, len(broken))
	for i, b := range broken {
		errs[i] = errors.New(b.Error)
	}
	return sums, errors.Join(errs...)
}

// ListEngagementsDetailed is ListEngagements with the unreadable folders
// returned as data. The error is only for a failure to read the engagements
// directory itself.
func ListEngagementsDetailed(root string) ([]Summary, []BrokenEngagement, error) {
	base := filepath.Join(root, engagementsDir)
	entries, err := os.ReadDir(base)
	if errors.Is(err, os.ErrNotExist) {
		return []Summary{}, []BrokenEngagement{}, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("read %s: %w", base, err)
	}
	out := []Summary{}
	broken := []BrokenEngagement{}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(base, e.Name())
		sum, ok, err := summarize(dir)
		if err != nil {
			broken = append(broken, BrokenEngagement{Dir: dir, Error: err.Error()})
			continue
		}
		if ok {
			out = append(out, sum)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, broken, nil
}

// summarize reads dir's engagement.json; ok is false when there is none.
func summarize(dir string) (Summary, bool, error) {
	path := metaPath(dir)
	data, err := readOptional(path)
	if err != nil || data == nil {
		return Summary{}, false, err
	}
	var meta model.EngagementMeta
	if err := decodeJSON(path, data, &meta); err != nil {
		return Summary{}, false, err
	}
	hosts, err := countHosts(dir)
	if err != nil {
		return Summary{}, false, err
	}
	return Summary{
		Dir:       dir,
		Name:      meta.Name,
		Type:      meta.Type,
		Hosts:     hosts,
		CreatedAt: meta.CreatedAt,
	}, true, nil
}

// countHosts counts hosts/*/host.json files.
func countHosts(dir string) (int, error) {
	root := filepath.Join(dir, hostsDir)
	entries, err := os.ReadDir(root)
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("read %s: %w", root, err)
	}
	n := 0
	for _, e := range entries {
		path := filepath.Join(root, e.Name(), hostFile)
		_, err := os.Stat(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return 0, fmt.Errorf("stat %s: %w", path, err)
		}
		n++
	}
	return n, nil
}

// slugify lowercases name and collapses runs of other characters to one dash.
func slugify(name string) string {
	return strings.Trim(nonSlug.ReplaceAllString(strings.ToLower(name), "-"), "-")
}

// CreateEngagement makes root/engagements/<slug>/ with engagement.json, an
// empty notes.md and hosts/, and returns the folder path.
func CreateEngagement(root, name, domain string, t model.EngagementType) (string, error) {
	if strings.ContainsAny(name, `/\`) || strings.Contains(name, "..") {
		return "", fmt.Errorf("invalid engagement name %q", name)
	}
	slug := slugify(name)
	if slug == "" {
		return "", fmt.Errorf("invalid engagement name %q: nothing left after cleaning", name)
	}
	dir := filepath.Join(root, engagementsDir, slug)
	if _, err := os.Stat(dir); err == nil {
		return "", fmt.Errorf("engagement %q already exists: %s", slug, dir)
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("stat %s: %w", dir, err)
	}
	if err := os.MkdirAll(filepath.Join(dir, hostsDir), dirMode); err != nil {
		return "", fmt.Errorf("create %s: %w", dir, err)
	}
	meta := model.EngagementMeta{
		ID:        model.NewID(),
		Name:      name,
		Type:      t,
		Domain:    domain,
		FlagTypes: model.DefaultFlagTypes(t),
		CreatedAt: time.Now().UTC(),
	}
	if meta.FlagTypes == nil {
		meta.FlagTypes = []model.FlagType{}
	}
	data, err := marshalIndentNL(meta)
	if err != nil {
		return "", fmt.Errorf("marshal %s: %w", metaPath(dir), err)
	}
	if err := WriteAtomic(metaPath(dir), data); err != nil {
		return "", err
	}
	if err := WriteAtomic(filepath.Join(dir, notesFile), nil); err != nil {
		return "", err
	}
	return dir, nil
}

package importers

import (
	"cmp"
	"errors"
	"fmt"
	"net/netip"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/unclesp1d3r/redtracks/internal/model"
	"github.com/unclesp1d3r/redtracks/internal/store"
)

// Badge says what an import would do to one record.
type Badge string

// Row badges.
const (
	BadgeNew       Badge = "new"
	BadgeFills     Badge = "fills"
	BadgeDuplicate Badge = "duplicate"
)

// RowKind is what sort of record a preview Row describes.
type RowKind string

// Row kinds.
const (
	RowKindHost    RowKind = "host"
	RowKindService RowKind = "service"
	RowKindWebPath RowKind = "webpath"
	RowKindFinding RowKind = "finding"
	RowKindUser    RowKind = "user"
	RowKindShare   RowKind = "share"
)

const (
	defaultHTTPPort  = 80
	defaultHTTPSPort = 443
	schemeHTTP       = "http"
	schemeHTTPS      = "https"
	protocolTCP      = "tcp"
)

// Row is one line of the import preview.
type Row struct {
	Kind   RowKind
	HostIP string
	Badge  Badge
	Label  string
}

// Unresolved is a result the importer could not place on a host. Key is what
// the overrides map of Apply uses.
type Unresolved struct {
	Key      string
	Hostname string
	URL      string
}

// Preview is everything an import would do, plus what it cannot place.
type Preview struct {
	Tool       string
	Rows       []Row
	Unresolved []Unresolved
	// Dropped lists the invalid IP strings the parser output carried; those
	// hosts are not imported.
	Dropped []string
	// Skipped counts rows a parser could not read (for example a non-numeric port).
	Skipped int
}

// target is a HostResult placed on an engagement host IP and web port.
type target struct {
	ip     string
	port   int // port web paths attach to, and the default for findings
	scheme string
	host   HostResult
}

// serviceType is the type given to a service created for a web path or finding.
func (t target) serviceType(port int) string {
	if t.scheme != "" && port == t.port {
		return t.scheme
	}
	if port == defaultHTTPSPort {
		return schemeHTTPS
	}
	return schemeHTTP
}

func (t target) findingPort(f FindingResult) int {
	if f.Port > 0 {
		return f.Port
	}
	return t.port
}

// baseURL is the parsed scheme, host and port of a HostResult.URL.
type baseURL struct {
	scheme, host string
	port         int
}

func parseBase(raw string) baseURL {
	u, err := url.Parse(raw)
	if raw == "" || err != nil {
		return baseURL{}
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil {
		port = 0 // absent or invalid port
	}
	return baseURL{scheme: u.Scheme, host: u.Hostname(), port: port}
}

func (b baseURL) webPort() int {
	switch {
	case b.port > 0:
		return b.port
	case b.scheme == schemeHTTPS:
		return defaultHTTPSPort
	default:
		return defaultHTTPPort
	}
}

// BuildPreview computes the rows an import of res would produce against eng.
func BuildPreview(eng model.Engagement, res Result, overrides map[string]string) Preview {
	targets, unresolved := resolveAll(eng, res, overrides)
	return buildPreview(eng, res, targets, unresolved)
}

func buildPreview(eng model.Engagement, res Result, targets []target, unresolved []Unresolved) Preview {
	tool := res.Tool
	work := eng
	work.Hosts = slices.Clone(eng.Hosts)
	rows := []Row{}
	for _, t := range targets {
		var r []Row
		work, r = previewTarget(work, t, tool)
		rows = append(rows, r...)
	}
	dropped := slices.Clone(res.DroppedIPs)
	if dropped == nil {
		dropped = []string{}
	}
	return Preview{Tool: tool, Rows: rows, Unresolved: unresolved, Dropped: dropped, Skipped: res.Skipped}
}

// previewTarget merges one target into a working copy of the engagement.
func previewTarget(work model.Engagement, t target, tool string) (model.Engagement, []Row) {
	idx := slices.IndexFunc(work.Hosts, func(h model.Host) bool { return h.IP == t.ip })
	cur, isNew := newHost(t.ip), idx < 0
	if !isNew {
		cur = work.Hosts[idx]
	}
	merged, rows := mergeHost(cur, t, tool, isNew)
	if isNew {
		work.Hosts = append(work.Hosts, merged)
	} else {
		work.Hosts[idx] = merged
	}
	var more []Row
	work.Creds, more = mergeCreds(work.Creds, merged.ID, t)
	rows = append(rows, more...)
	work.Findings, more = mergeFindings(work.Findings, merged, t, tool)
	return work, append(rows, more...)
}

func newHost(ip string) model.Host {
	return model.Host{ID: model.NewID(), IP: ip, Status: model.HostStatusUntouched}
}

// Apply writes res into st and returns the preview computed beforehand.
// Unresolved hosts are reported and never written. The raw file is saved once,
// under the first resolved host.
func Apply(st *store.Store, res Result, overrides map[string]string, raw []byte) (Preview, error) {
	eng := st.Engagement()
	targets, unresolved := resolveAll(eng, res, overrides)
	preview := buildPreview(eng, res, targets, unresolved)
	for i, t := range targets {
		_, exists := eng.Host(t.ip)
		if err := writeTarget(st, t, res.Tool, exists); err != nil {
			return preview, fmt.Errorf("import %s into %s failed after writing %d of %d hosts "+
				"(confirming again retries safely): %w", res.Tool, t.ip, i, len(targets), err)
		}
	}
	if len(targets) > 0 && len(raw) > 0 {
		if _, err := st.SaveImport(targets[0].ip, res.Tool, raw); err != nil {
			return preview, fmt.Errorf("all %d hosts were written but saving the raw %s output failed "+
				"(confirming again retries safely): %w", len(targets), res.Tool, err)
		}
	}
	return preview, nil
}

// writeTarget merges t into its host. exists comes from the snapshot taken at
// the start of Apply; a host created meanwhile (by an earlier target or
// another writer) makes AddHost fail with ErrHostExists and is merged instead.
func writeTarget(st *store.Store, t target, tool string, exists bool) error {
	mutate := func() error {
		return st.MutateHost(t.ip, func(h *model.Host) error {
			*h, _ = mergeHost(*h, t, tool, false)
			return nil
		})
	}
	if exists {
		if err := mutate(); err != nil {
			return err
		}
	} else {
		h, _ := mergeHost(newHost(t.ip), t, tool, true)
		err := st.AddHost(h)
		if errors.Is(err, store.ErrHostExists) {
			err = mutate()
		}
		if err != nil {
			return err
		}
	}
	host, ok := st.Engagement().Host(t.ip)
	if !ok {
		return fmt.Errorf("host %s not found after writing it", t.ip)
	}
	if len(t.host.Users)+len(t.host.Shares) > 0 {
		err := st.MutateCreds(func(c *model.Creds) error {
			*c, _ = mergeCreds(*c, host.ID, t)
			return nil
		})
		if err != nil {
			return err
		}
	}
	if len(t.host.Findings) == 0 {
		return nil
	}
	return st.MutateFindings(func(f *model.FindingsFile) error {
		f.Findings, _ = mergeFindings(f.Findings, host, t, tool)
		return nil
	})
}

// resolveAll places every HostResult on a host IP or reports it unresolved.
func resolveAll(eng model.Engagement, res Result, overrides map[string]string) ([]target, []Unresolved) {
	var targets []target
	unresolved := []Unresolved{}
	for _, hr := range res.Hosts {
		t, miss := resolveHost(eng, res.Tool, hr, overrides)
		if miss == nil {
			targets = append(targets, t)
			continue
		}
		if !slices.ContainsFunc(unresolved, func(u Unresolved) bool { return u.Key == miss.Key }) {
			unresolved = append(unresolved, *miss)
		}
	}
	return targets, unresolved
}

// resolveHost returns the target, or the Unresolved entry when no host fits.
func resolveHost(eng model.Engagement, tool string, hr HostResult, overrides map[string]string) (target, *Unresolved) {
	u := parseBase(hr.URL)
	t := target{port: u.webPort(), scheme: u.scheme, host: hr}
	if hr.IP != "" {
		t.ip = hr.IP
		return t, nil
	}
	key := tool
	name := cmp.Or(u.host, hr.Hostname)
	if name != "" {
		if ip, ok := engagementHostIP(eng, name); ok {
			t.ip = ip
			return t, nil
		}
		key = name
	}
	if ip, port, ok := parseOverride(overrides[key]); ok {
		t.ip = ip
		if port > 0 {
			t.port = port
		}
		return t, nil
	}
	return target{}, &Unresolved{Key: key, Hostname: hr.Hostname, URL: hr.URL}
}

func engagementHostIP(eng model.Engagement, name string) (string, bool) {
	for _, h := range eng.Hosts {
		if h.IP == name || (h.Hostname != "" && strings.EqualFold(h.Hostname, name)) {
			return h.IP, true
		}
	}
	return "", false
}

// parseOverride reads "ip" or "ip:port"; port is 0 when absent. An invalid
// value is not ok, which leaves the result unresolved.
//
//nolint:gocritic // results are ip, port, ok; nonamedreturns bans naming them
func parseOverride(v string) (string, int, bool) {
	if ap, err := netip.ParseAddrPort(v); err == nil {
		return ap.Addr().String(), int(ap.Port()), true
	}
	if a, err := netip.ParseAddr(v); err == nil {
		return a.String(), 0, true
	}
	return "", 0, false
}

// fillEmpty sets *dst to src when *dst is empty and src is not, reporting whether it did.
func fillEmpty(dst *string, src string) bool {
	if *dst != "" || src == "" {
		return false
	}
	*dst = src
	return true
}

func badgeFor(isNew, filled bool) Badge {
	switch {
	case isNew:
		return BadgeNew
	case filled:
		return BadgeFills
	default:
		return BadgeDuplicate
	}
}

// mergeHost returns a copy of h with t's host fields, services, web paths and
// finding services merged in, and the rows describing the change.
func mergeHost(h model.Host, t target, tool string, isNew bool) (model.Host, []Row) {
	out := h
	filled := fillEmpty(&out.Hostname, t.host.Hostname)
	filled = fillEmpty(&out.OS, t.host.OS) || filled
	label := out.IP
	if out.Hostname != "" {
		label += " (" + out.Hostname + ")"
	}
	rows := []Row{{Kind: RowKindHost, HostIP: out.IP, Badge: badgeFor(isNew, filled), Label: label}}
	out.Services = slices.Clone(h.Services)
	for _, sr := range t.host.Services {
		var row Row
		out.Services, row = upsertService(out.Services, sr)
		rows = append(rows, withHost(row, out.IP))
	}
	rows = append(rows, ensureWebServices(&out, t)...)
	rows = append(rows, mergeWebPaths(&out, t, tool)...)
	return out, rows
}

func withHost(r Row, ip string) Row {
	r.HostIP = ip
	return r
}

// ensureWebServices creates the services that web paths and findings attach to.
func ensureWebServices(h *model.Host, t target) []Row {
	ports := []int{}
	if len(t.host.WebPaths) > 0 {
		ports = append(ports, t.port)
	}
	for _, f := range t.host.Findings {
		ports = append(ports, t.findingPort(f))
	}
	rows := []Row{}
	for _, p := range slices.Compact(slices.Sorted(slices.Values(ports))) {
		var row Row
		sr := ServiceResult{Port: p, Protocol: protocolTCP, Type: t.serviceType(p)}
		h.Services, row = upsertService(h.Services, sr)
		if row.Badge != BadgeDuplicate {
			rows = append(rows, withHost(row, h.IP))
		}
	}
	return rows
}

func serviceIndex(svcs []model.Service, port int, proto string) int {
	return slices.IndexFunc(svcs, func(s model.Service) bool {
		return s.Port == port && cmp.Or(s.Protocol, protocolTCP) == proto // hand-edited services may omit protocol
	})
}

// upsertService matches by port and protocol and fills empty fields only.
func upsertService(svcs []model.Service, sr ServiceResult) ([]model.Service, Row) {
	proto := sr.Protocol
	if proto == "" {
		proto = protocolTCP
	}
	out := slices.Clone(svcs)
	i := serviceIndex(out, sr.Port, proto)
	if i < 0 {
		s := model.Service{
			ID: model.NewID(), Port: sr.Port, Protocol: proto, Type: sr.Type,
			Product: sr.Product, Version: sr.Version, WebPaths: []model.WebPath{},
		}
		return append(out, s), Row{Kind: RowKindService, Badge: BadgeNew, Label: s.Label()}
	}
	s := out[i]
	filled := fillEmpty(&s.Type, sr.Type)
	filled = fillEmpty(&s.Product, sr.Product) || filled
	filled = fillEmpty(&s.Version, sr.Version) || filled
	out[i] = s
	return out, Row{Kind: RowKindService, Badge: badgeFor(false, filled), Label: s.Label()}
}

// mergeWebPaths adds t's web paths to the service on t.port, deduplicated by path and status.
func mergeWebPaths(h *model.Host, t target, tool string) []Row {
	if len(t.host.WebPaths) == 0 {
		return nil
	}
	i := serviceIndex(h.Services, t.port, protocolTCP)
	svc := h.Services[i]
	paths := slices.Clone(svc.WebPaths)
	rows := make([]Row, 0, len(t.host.WebPaths))
	for _, w := range t.host.WebPaths {
		badge := BadgeDuplicate
		if !slices.ContainsFunc(paths, func(p model.WebPath) bool { return p.Path == w.Path && p.Status == w.Status }) {
			paths = append(paths, model.WebPath{Path: w.Path, Status: w.Status, Size: w.Size, Source: tool})
			badge = BadgeNew
		}
		label := fmt.Sprintf("%s (%d)", w.Path, w.Status)
		rows = append(rows, Row{Kind: RowKindWebPath, HostIP: h.IP, Badge: badge, Label: label})
	}
	svc.WebPaths = paths
	h.Services = slices.Clone(h.Services)
	h.Services[i] = svc
	return rows
}

// mergeFindings adds t's findings, deduplicated by source and title per service.
// h must already hold the services the findings attach to.
func mergeFindings(all []model.Finding, h model.Host, t target, tool string) ([]model.Finding, []Row) {
	out := slices.Clone(all)
	rows := make([]Row, 0, len(t.host.Findings))
	for _, f := range t.host.Findings {
		serviceID := ""
		if i := serviceIndex(h.Services, t.findingPort(f), protocolTCP); i >= 0 {
			serviceID = h.Services[i].ID
		}
		isDup := slices.ContainsFunc(out, func(e model.Finding) bool {
			return e.HostID == h.ID && e.ServiceID == serviceID && e.Source == tool && e.Title == f.Title
		})
		if !isDup {
			out = append(out, model.Finding{
				ID: model.NewID(), HostID: h.ID, ServiceID: serviceID, Title: f.Title,
				Severity: f.Severity, Source: tool, Evidence: f.Evidence, Status: model.FindingNoted,
			})
		}
		rows = append(rows, Row{Kind: RowKindFinding, HostIP: h.IP, Badge: badgeFor(!isDup, false), Label: f.Title})
	}
	return out, rows
}

// mergeCreds adds t's users and shares to c, deduplicated by name per host.
func mergeCreds(c model.Creds, hostID string, t target) (model.Creds, []Row) {
	out := c
	out.Users, out.Shares = slices.Clone(c.Users), slices.Clone(c.Shares)
	rows := []Row{}
	for _, u := range t.host.Users {
		i := slices.IndexFunc(out.Users, func(e model.User) bool { return e.HostID == hostID && e.Name == u.Name })
		filled := false
		if i < 0 {
			out.Users = append(out.Users, model.User{ID: model.NewID(), Name: u.Name, Domain: u.Domain, HostID: hostID})
		} else {
			filled = fillEmpty(&out.Users[i].Domain, u.Domain)
		}
		rows = append(rows, Row{Kind: RowKindUser, HostIP: t.ip, Badge: badgeFor(i < 0, filled), Label: u.Name})
	}
	for _, s := range t.host.Shares {
		i := slices.IndexFunc(out.Shares, func(e model.Share) bool { return e.HostID == hostID && e.Name == s.Name })
		filled := false
		if i < 0 {
			share := model.Share{ID: model.NewID(), HostID: hostID, Name: s.Name, Access: s.Access}
			out.Shares = append(out.Shares, share)
		} else {
			filled = fillEmpty(&out.Shares[i].Access, s.Access)
		}
		rows = append(rows, Row{Kind: RowKindShare, HostIP: t.ip, Badge: badgeFor(i < 0, filled), Label: s.Name})
	}
	return out, rows
}

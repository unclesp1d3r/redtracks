package importers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"net/url"
	"strings"
)

type ffufConfig struct {
	URL string `json:"url"`
}

type ffufOutput struct {
	Config  ffufConfig `json:"config"`
	Results []struct {
		Input  map[string]string `json:"input"`
		Status int               `json:"status"`
		Length int               `json:"length"`
	} `json:"results"`
}

// webBase returns a HostResult whose URL is the scheme, host, and port of
// rawURL, with IP set when the host is an IP literal.
func webBase(rawURL string) (HostResult, error) {
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return HostResult{}, fmt.Errorf("invalid base URL %q", rawURL)
	}
	h := HostResult{URL: u.Scheme + "://" + u.Host}
	if addr, perr := netip.ParseAddr(u.Hostname()); perr == nil {
		h.IP = addr.String()
	}
	return h, nil
}

func parseFfuf(data []byte) (Result, error) {
	var out ffufOutput
	if err := json.Unmarshal(data, &out); err != nil {
		return Result{}, fmt.Errorf("decode ffuf JSON: %w", err)
	}
	if out.Config.URL == "" {
		return Result{}, errors.New("ffuf output has no config.url")
	}
	host, err := webBase(strings.ReplaceAll(out.Config.URL, "FUZZ", ""))
	if err != nil {
		return Result{}, err
	}
	for _, r := range out.Results {
		host.WebPaths = append(host.WebPaths, WebPathResult{
			Path: "/" + r.Input["FUZZ"], Status: r.Status, Size: r.Length,
		})
	}
	return Result{Tool: ToolFfuf, Hosts: []HostResult{host}}, nil
}

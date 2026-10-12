package importers

import (
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
)

type nmapRun struct {
	Hosts []nmapHost `xml:"host"`
}

type nmapHost struct {
	Status    nmapStatus     `xml:"status"`
	Addresses []nmapAddress  `xml:"address"`
	Hostnames []nmapHostname `xml:"hostnames>hostname"`
	Ports     []nmapPort     `xml:"ports>port"`
	OSMatches []nmapOSMatch  `xml:"os>osmatch"`
}

type nmapStatus struct {
	State string `xml:"state,attr"`
}

type nmapAddress struct {
	Addr     string `xml:"addr,attr"`
	AddrType string `xml:"addrtype,attr"`
}

type nmapHostname struct {
	Name string `xml:"name,attr"`
}

type nmapPort struct {
	Protocol string      `xml:"protocol,attr"`
	PortID   string      `xml:"portid,attr"`
	State    nmapStatus  `xml:"state"`
	Service  nmapService `xml:"service"`
}

type nmapService struct {
	Name    string `xml:"name,attr"`
	Product string `xml:"product,attr"`
	Version string `xml:"version,attr"`
}

type nmapOSMatch struct {
	Name string `xml:"name,attr"`
}

func parseNmap(data []byte) (Result, error) {
	var run nmapRun
	if err := xml.Unmarshal(data, &run); err != nil {
		return Result{}, fmt.Errorf("decode nmap XML: %w", err)
	}
	hosts := make([]HostResult, 0, len(run.Hosts))
	for _, h := range run.Hosts {
		if h.Status.State == "down" {
			continue
		}
		hosts = append(hosts, nmapHostResult(h))
	}
	return Result{Tool: ToolNmap, Hosts: hosts}, nil
}

func nmapHostResult(h nmapHost) HostResult {
	out := HostResult{IP: nmapIP(h.Addresses)}
	if len(h.Hostnames) > 0 {
		out.Hostname = h.Hostnames[0].Name
	}
	if len(h.OSMatches) > 0 {
		out.OS = guessOS(h.OSMatches[0].Name)
	}
	for _, p := range h.Ports {
		port, err := strconv.Atoi(p.PortID)
		if p.State.State != "open" || err != nil {
			continue
		}
		out.Services = append(out.Services, ServiceResult{
			Port:     port,
			Protocol: p.Protocol,
			Type:     p.Service.Name,
			Product:  p.Service.Product,
			Version:  p.Service.Version,
		})
	}
	return out
}

// nmapIP returns the first ipv4/ipv6 address, ignoring MAC addresses.
func nmapIP(addrs []nmapAddress) string {
	for _, a := range addrs {
		if a.AddrType == "ipv4" || a.AddrType == "ipv6" {
			return a.Addr
		}
	}
	return ""
}

// guessOS maps an nmap osmatch name to "linux", "windows", or "".
func guessOS(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.Contains(lower, "linux"):
		return "linux"
	case strings.Contains(lower, "windows"):
		return "windows"
	default:
		return ""
	}
}

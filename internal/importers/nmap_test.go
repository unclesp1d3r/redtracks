package importers

import (
	"strings"
	"testing"
)

func TestParseNmapLocalhost(t *testing.T) {
	res, err := Parse(readFixture(t, "nmap-localhost.xml"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if res.Tool != "nmap" || len(res.Hosts) != 1 {
		t.Fatalf("tool=%q hosts=%d, want nmap and 1", res.Tool, len(res.Hosts))
	}
	h := res.Hosts[0]
	if h.IP != "127.0.0.1" {
		t.Errorf("IP = %q", h.IP)
	}
	if h.Hostname != "localhost" {
		t.Errorf("Hostname = %q", h.Hostname)
	}
	if h.OS != "" {
		t.Errorf("OS = %q, want empty", h.OS)
	}
	if len(h.Services) == 0 {
		t.Fatal("no services")
	}
	for _, s := range h.Services {
		if s.Protocol != "tcp" || s.Port == 0 {
			t.Errorf("bad service %+v", s)
		}
		if s.Product != "" && s.Type == "" {
			t.Errorf("service %+v has product but no type", s)
		}
	}
}

func TestParseNmapSkipsDownHosts(t *testing.T) {
	res, err := Parse(readFixture(t, "nmap-subnet-down.xml"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(res.Hosts) != 1 || res.Dropped != 1 {
		t.Fatalf("hosts=%d dropped=%d, want 1 and 1", len(res.Hosts), res.Dropped)
	}
	h := res.Hosts[0]
	if h.IP != "10.10.10.5" || len(h.Services) != 1 || h.Services[0].Port != 80 {
		t.Errorf("unexpected host %+v", h)
	}
	if h.Services[0].Type != "http" || h.Services[0].Product != "nginx" || h.Services[0].Version != "1.18.0" {
		t.Errorf("unexpected service %+v", h.Services[0])
	}
}

func TestParseNmapOSGuess(t *testing.T) {
	res, err := Parse(readFixture(t, "nmap-subnet-down.xml"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := res.Hosts[0].OS; got != "linux" {
		t.Errorf("OS = %q, want linux", got)
	}
	if got := guessOS("Microsoft Windows 10"); got != "windows" {
		t.Errorf("guessOS windows = %q", got)
	}
	if got := guessOS("FreeBSD 13"); got != "" {
		t.Errorf("guessOS other = %q, want empty", got)
	}
}

func TestParseNmapMalformedXML(t *testing.T) {
	if _, err := parseNmap([]byte("<nmaprun><host>")); err == nil {
		t.Fatal("want error for truncated XML")
	}
}

func TestParseNmapCountsNonNumericPortAsSkipped(t *testing.T) {
	const doc = `<nmaprun><host><status state="up"/><address addr="10.0.0.1" addrtype="ipv4"/><ports>
<port protocol="tcp" portid="http"><state state="open"/><service name="http"/></port>
<port protocol="tcp" portid="22"><state state="open"/><service name="ssh"/></port>
</ports></host></nmaprun>`
	res, err := parseNmap([]byte(doc))
	if err != nil {
		t.Fatal(err)
	}
	if res.Skipped != 1 || len(res.Hosts[0].Services) != 1 {
		t.Fatalf("Skipped=%d services=%+v, want 1 skipped and ssh kept", res.Skipped, res.Hosts[0].Services)
	}
}

func TestParseNmapSkipsOutOfRangePorts(t *testing.T) {
	var ports strings.Builder
	for _, id := range []string{"-1", "0", "70000", "65535", "1"} {
		ports.WriteString(`<port protocol="tcp" portid="` + id + `"><state state="open"/></port>`)
	}
	doc := `<nmaprun><host><status state="up"/><address addr="10.0.0.1" addrtype="ipv4"/><ports>` +
		ports.String() + `</ports></host></nmaprun>`
	res, err := parseNmap([]byte(doc))
	if err != nil {
		t.Fatal(err)
	}
	if res.Skipped != 3 || len(res.Hosts[0].Services) != 2 {
		t.Fatalf(
			"Skipped=%d services=%+v, want 3 skipped and ports 65535 and 1 kept",
			res.Skipped,
			res.Hosts[0].Services,
		)
	}
}

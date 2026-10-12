package importers

import "testing"

func TestParseNikto(t *testing.T) {
	res, err := Parse(readFixture(t, "nikto.json"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if res.Tool != ToolNikto || len(res.Hosts) != 1 {
		t.Fatalf("tool=%q hosts=%d", res.Tool, len(res.Hosts))
	}
	h := res.Hosts[0]
	if h.IP != "10.10.10.5" || len(h.Findings) != 3 {
		t.Fatalf("host %+v, want IP 10.10.10.5 and 3 findings", h)
	}
	first := h.Findings[0]
	if first.Port != 80 || first.Severity != "info" ||
		first.Title != "The anti-clickjacking X-Frame-Options header is not present." || first.Evidence != "/" {
		t.Errorf("finding[0] = %+v", first)
	}
	for _, f := range h.Findings {
		if f.Port != 80 {
			t.Errorf("finding %+v port != 80", f)
		}
	}
}

func TestParseNuclei(t *testing.T) {
	res, err := Parse(readFixture(t, "nuclei.jsonl"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if res.Tool != ToolNuclei || len(res.Hosts) != 2 {
		t.Fatalf("tool=%q hosts=%d, want nuclei and 2", res.Tool, len(res.Hosts))
	}
	ip := res.Hosts[0]
	if ip.IP != "10.10.10.5" || ip.URL != "" || len(ip.Findings) != 2 {
		t.Fatalf("ip host %+v", ip)
	}
	want := FindingResult{
		Title: "Apache 2.4.49 Path Traversal", Severity: "critical",
		Evidence: "http://10.10.10.5/cgi-bin/.%2e/.%2e/etc/passwd", Port: 80,
	}
	if ip.Findings[0] != want {
		t.Errorf("finding[0] = %+v, want %+v", ip.Findings[0], want)
	}
	named := res.Hosts[1]
	if named.IP != "" || named.URL != "https://app.example.test:8443" || len(named.Findings) != 1 {
		t.Fatalf("named host %+v", named)
	}
	if named.Findings[0].Severity != "medium" || named.Findings[0].Port != 8443 {
		t.Errorf("named finding %+v", named.Findings[0])
	}
}

func TestVulnParsersRejectMalformedInput(t *testing.T) {
	table := map[string]struct {
		parse func([]byte) (Result, error)
		input string
	}{
		"nikto bad json":      {parseNikto, `{"host":`},
		"nikto bad port":      {parseNikto, `{"host":"x","ip":"1.2.3.4","port":"http","vulnerabilities":[]}`},
		"nuclei bad line":     {parseNuclei, `{"template-id":` + "\n"},
		"nuclei bad port":     {parseNuclei, `{"template-id":"x","host":"http://x","port":"zz"}` + "\n"},
		"nuclei bad host url": {parseNuclei, `{"template-id":"x","host":"http://","port":"80"}` + "\n"},
	}
	for name, tc := range table {
		t.Run(name, func(t *testing.T) {
			if _, err := tc.parse([]byte(tc.input)); err == nil {
				t.Fatal("want error")
			}
		})
	}
}

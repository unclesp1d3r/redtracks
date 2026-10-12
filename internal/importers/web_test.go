package importers

import "testing"

func TestParseMasscan(t *testing.T) {
	res, err := Parse(readFixture(t, "masscan.json"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if res.Tool != ToolMasscan || len(res.Hosts) != 2 {
		t.Fatalf("tool=%q hosts=%d, want masscan and 2", res.Tool, len(res.Hosts))
	}
	h := res.Hosts[0]
	if h.IP != "10.10.10.5" || len(h.Services) != 2 {
		t.Fatalf("host %+v, want 10.10.10.5 with 2 services", h)
	}
	if h.Services[0].Port != 80 || h.Services[1].Port != 22 || h.Services[0].Protocol != "tcp" {
		t.Errorf("services %+v", h.Services)
	}
	for _, s := range h.Services {
		if s.Type != "" {
			t.Errorf("service %+v should have empty Type", s)
		}
	}
	if res.Hosts[1].IP != "10.10.10.6" || res.Hosts[1].Services[0].Port != 445 {
		t.Errorf("host 2 %+v", res.Hosts[1])
	}
}

func TestParseGobuster(t *testing.T) {
	res, err := Parse(readFixture(t, "gobuster-dir.txt"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if res.Tool != ToolGobuster || len(res.Hosts) != 1 {
		t.Fatalf("tool=%q hosts=%d", res.Tool, len(res.Hosts))
	}
	h := res.Hosts[0]
	if h.IP != "" || h.URL != "" {
		t.Errorf("IP=%q URL=%q, want both empty", h.IP, h.URL)
	}
	want := []WebPathResult{{"/admin", 301, 169}, {"/index.html", 200, 1234}, {"/secret", 403, 277}}
	if len(h.WebPaths) != len(want) {
		t.Fatalf("paths = %+v", h.WebPaths)
	}
	for i, w := range want {
		if h.WebPaths[i] != w {
			t.Errorf("path[%d] = %+v, want %+v", i, h.WebPaths[i], w)
		}
	}
}

func TestParseFfuf(t *testing.T) {
	res, err := Parse(readFixture(t, "ffuf.json"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if res.Tool != ToolFfuf || len(res.Hosts) != 1 {
		t.Fatalf("tool=%q hosts=%d", res.Tool, len(res.Hosts))
	}
	h := res.Hosts[0]
	if h.URL != "http://10.10.10.5" || h.IP != "10.10.10.5" {
		t.Errorf("URL=%q IP=%q", h.URL, h.IP)
	}
	want := []WebPathResult{{"/admin", 301, 169}, {"/login", 200, 4021}}
	if len(h.WebPaths) != 2 || h.WebPaths[0] != want[0] || h.WebPaths[1] != want[1] {
		t.Errorf("paths = %+v, want %+v", h.WebPaths, want)
	}
}

func TestParseFeroxbuster(t *testing.T) {
	res, err := Parse(readFixture(t, "feroxbuster.jsonl"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if res.Tool != ToolFeroxbuster || len(res.Hosts) != 1 {
		t.Fatalf("tool=%q hosts=%d", res.Tool, len(res.Hosts))
	}
	h := res.Hosts[0]
	if h.URL != "http://app.example.test:8080" || h.IP != "" {
		t.Errorf("URL=%q IP=%q", h.URL, h.IP)
	}
	want := []WebPathResult{{"/admin", 301, 169}, {"/images/logo.png", 200, 5120}}
	if len(h.WebPaths) != 2 || h.WebPaths[0] != want[0] || h.WebPaths[1] != want[1] {
		t.Errorf("paths = %+v, want %+v", h.WebPaths, want)
	}
}

func TestWebParsersRejectMalformedInput(t *testing.T) {
	table := map[string]struct {
		parse func([]byte) (Result, error)
		input string
	}{
		"masscan bad json":         {parseMasscan, `[{"ip":`},
		"masscan bad port":         {parseMasscan, `[{"ip":"1.2.3.4","ports":[{"port":"x"}]}]`},
		"gobuster no result lines": {parseGobuster, "Starting gobuster\nFinished\n"},
		"ffuf bad json":            {parseFfuf, `{"commandline":`},
		"ffuf missing url":         {parseFfuf, `{"commandline":"x","results":[],"config":{}}`},
		"feroxbuster bad line":     {parseFeroxbuster, `{"type":"response","url":` + "\n"},
		"feroxbuster bad url":      {parseFeroxbuster, `{"type":"response","url":"::nope","status":200}` + "\n"},
	}
	for name, tc := range table {
		t.Run(name, func(t *testing.T) {
			if _, err := tc.parse([]byte(tc.input)); err == nil {
				t.Fatal("want error")
			}
		})
	}
}

package importers

import (
	"strings"
	"testing"
)

func TestParseEnum4linux(t *testing.T) {
	res, err := Parse(readFixture(t, "enum4linux-ng.json"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if res.Tool != ToolEnum4linuxNG || len(res.Hosts) != 1 {
		t.Fatalf("tool=%q hosts=%d", res.Tool, len(res.Hosts))
	}
	h := res.Hosts[0]
	if h.Hostname != "dc01.corp.local" || h.IP != "" || h.OS != "windows" {
		t.Errorf("Hostname=%q IP=%q OS=%q", h.Hostname, h.IP, h.OS)
	}
	wantUsers := []UserResult{{"Administrator", "corp.local"}, {"svc_backup", "corp.local"}}
	if len(h.Users) != 2 || h.Users[0] != wantUsers[0] || h.Users[1] != wantUsers[1] {
		t.Errorf("users = %+v, want %+v", h.Users, wantUsers)
	}
	wantShares := []ShareResult{{"ADMIN$", "DENIED/N/A"}, {"SYSVOL", "OK/OK"}}
	if len(h.Shares) != 2 || h.Shares[0] != wantShares[0] || h.Shares[1] != wantShares[1] {
		t.Errorf("shares = %+v, want %+v", h.Shares, wantShares)
	}
}

func TestParseEnum4linuxIPTargetAndBadDomain(t *testing.T) {
	in := `{"target":{"host":"10.10.10.5"},"domain":"evil domain","users":{"1":{"username":"bob"}}}`
	res, err := parseEnum4linux([]byte(in))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	h := res.Hosts[0]
	if h.IP != "10.10.10.5" || h.Hostname != "" {
		t.Errorf("IP=%q Hostname=%q", h.IP, h.Hostname)
	}
	if len(h.Users) != 1 || h.Users[0].Domain != "" {
		t.Errorf("users = %+v, want empty domain", h.Users)
	}
}

func TestParseNetexec(t *testing.T) {
	res, err := Parse(readFixture(t, "netexec-smb.txt"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	assertNetexecFixture(t, res)
}

func assertNetexecFixture(t *testing.T, res Result) {
	t.Helper()
	if res.Tool != ToolNetexec || len(res.Hosts) != 1 {
		t.Fatalf("tool=%q hosts=%d", res.Tool, len(res.Hosts))
	}
	h := res.Hosts[0]
	if h.IP != "10.10.10.5" || h.Hostname != "DC01" || !strings.Contains(h.OS, "windows") {
		t.Errorf("IP=%q Hostname=%q OS=%q", h.IP, h.Hostname, h.OS)
	}
	if len(h.Users) != 1 || h.Users[0] != (UserResult{Name: "svc_backup", Domain: "corp.local"}) {
		t.Errorf("users = %+v", h.Users)
	}
	want := []ShareResult{{"backups", "READ,WRITE"}, {"IPC$", "READ"}}
	if len(h.Shares) != 2 || h.Shares[0] != want[0] || h.Shares[1] != want[1] {
		t.Errorf("shares = %+v, want %+v", h.Shares, want)
	}
}

func TestParseNetexecStripsANSI(t *testing.T) {
	plain := readFixture(t, "netexec-smb.txt")
	lines := strings.Split(strings.TrimRight(string(plain), "\n"), "\n")
	for i, l := range lines {
		lines[i] = "\x1b[1;32m" + l + "\x1b[0m"
	}
	res, err := Parse([]byte(strings.Join(lines, "\n") + "\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	assertNetexecFixture(t, res)
}

func TestParseNetexecDropsBadDomain(t *testing.T) {
	in := "SMB 10.10.10.5 445 DC01 [+] bad\x01dom\\bob:pw\n"
	res, err := parseNetexec([]byte(in))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	u := res.Hosts[0].Users
	if len(u) != 1 || u[0].Name != "bob" || u[0].Domain != "" {
		t.Errorf("users = %+v, want bob with empty domain", u)
	}
}

func TestSMBParsersRejectMalformedInput(t *testing.T) {
	table := map[string]struct {
		parse func([]byte) (Result, error)
		input string
	}{
		"enum4linux bad json":     {parseEnum4linux, `{"target":`},
		"enum4linux no target":    {parseEnum4linux, `{"users":{}}`},
		"netexec no result lines": {parseNetexec, "just some text\n"},
	}
	for name, tc := range table {
		t.Run(name, func(t *testing.T) {
			if _, err := tc.parse([]byte(tc.input)); err == nil {
				t.Fatal("want error")
			}
		})
	}
}

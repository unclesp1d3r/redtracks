package importers

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return data
}

func TestDetectAllFixtures(t *testing.T) {
	// Extended by later units as they add fixtures.
	table := map[string]string{
		"nmap-localhost.xml":   "nmap",
		"nmap-subnet-down.xml": "nmap",
	}
	for file, want := range table {
		t.Run(file, func(t *testing.T) {
			got, err := Detect(readFixture(t, file))
			if err != nil {
				t.Fatalf("Detect: %v", err)
			}
			if got != want {
				t.Errorf("Detect = %q, want %q", got, want)
			}
		})
	}
}

func TestDetectSniffRules(t *testing.T) {
	table := map[string]string{
		"masscan":       `[{"ip":"1.2.3.4","ports":[{"port":80,"proto":"tcp"}]}]`,
		"ffuf":          `{"commandline":"ffuf -u x","results":[]}`,
		"feroxbuster":   `{"type":"response","url":"http://x/a","status":200}` + "\n",
		"gobuster":      "/admin                (Status: 301) [Size: 169]\n",
		"nikto":         `{"host":"1.2.3.4","vulnerabilities":[]}`,
		"nuclei":        `{"template-id":"x","host":"http://x"}` + "\n",
		"enum4linux-ng": `{"target":{"host":"1.2.3.4"},"users":{}}`,
		"netexec":       "SMB         10.0.0.5   445    DC01   [*] Windows Server\n",
	}
	for want, sample := range table {
		t.Run(want, func(t *testing.T) {
			got, err := Detect([]byte(sample))
			if err != nil {
				t.Fatalf("Detect: %v", err)
			}
			if got != want {
				t.Errorf("Detect = %q, want %q", got, want)
			}
		})
	}
}

func TestDetectGarbage(t *testing.T) {
	_, err := Detect(readFixture(t, "garbage.bin"))
	if !errors.Is(err, ErrUnknownFormat) {
		t.Fatalf("err = %v, want ErrUnknownFormat", err)
	}
	if !strings.Contains(err.Error(), "nmap") {
		t.Errorf("error %q should list tools tried", err)
	}
}

func TestParseGarbageReturnsDetectionError(t *testing.T) {
	_, err := Parse(readFixture(t, "garbage.bin"))
	if !errors.Is(err, ErrUnknownFormat) {
		t.Fatalf("err = %v, want ErrUnknownFormat", err)
	}
}

func TestParseDetectedToolWithoutParser(t *testing.T) {
	saved := parsers
	parsers = map[string]func([]byte) (Result, error){}
	t.Cleanup(func() { parsers = saved })
	_, err := Parse([]byte(`{"template-id":"x","host":"http://x"}` + "\n"))
	if err == nil || errors.Is(err, ErrUnknownFormat) {
		t.Fatalf("err = %v, want non-nil error that is not ErrUnknownFormat", err)
	}
}

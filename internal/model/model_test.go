package model_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/unclesp1d3r/redtracks/internal/model"
)

func eq(t *testing.T, want, got any) {
	t.Helper()
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("want %#v, got %#v", want, got)
	}
}

func check(t *testing.T, cond bool, msg ...any) {
	t.Helper()
	if !cond {
		t.Fatal(msg...)
	}
}

func noErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

const idSampleCount = 1000

const sampleHostJSON = `{
  "id": "K3M7Q2ZX9A",
  "ip": "10.10.10.5",
  "hostname": "nibbles",
  "os": "linux",
  "status": "foothold",
  "tags": [
    "chain-a"
  ],
  "accessPaths": [
    {
      "id": "H8T2WQ4N1B",
      "method": "direct",
      "viaHostId": "",
      "credentialId": "",
      "command": "",
      "notes": "public IP, reachable from attacker box"
    }
  ],
  "services": [
    {
      "id": "P1R9X3LM6C",
      "port": 80,
      "protocol": "tcp",
      "type": "http",
      "product": "Apache httpd",
      "version": "2.4.18",
      "notes": "",
      "webPaths": [
        {
          "path": "/nibbleblog/",
          "status": 200,
          "size": 2986,
          "source": "gobuster"
        }
      ],
      "checklist": {
        "items": {
          "http.robots": {
            "status": "done",
            "note": "",
            "updatedAt": "2026-10-12T14:08:11Z"
          },
          "http.vhosts": {
            "status": "todo",
            "note": "",
            "updatedAt": ""
          }
        }
      }
    }
  ],
  "flags": [
    {
      "typeId": "local",
      "value": "f2a1...",
      "capturedAt": "2026-10-12T15:02:00Z",
      "submitted": true
    },
    {
      "typeId": "proof",
      "value": "",
      "capturedAt": "",
      "submitted": false
    }
  ],
  "checklist": {
    "items": {}
  }
}`

func TestDefaultFlagTypes(t *testing.T) {
	tests := []struct {
		name string
		typ  model.EngagementType
		want []model.FlagType
	}{
		{
			"lab",
			model.EngagementLab,
			[]model.FlagType{{ID: "local", Label: "Local", Points: 10}, {ID: "proof", Label: "Proof", Points: 10}},
		},
		{
			"oscp",
			model.EngagementOSCP,
			[]model.FlagType{{ID: "local", Label: "Local", Points: 10}, {ID: "proof", Label: "Proof", Points: 10}},
		},
		{"osai", model.EngagementOSAI, []model.FlagType{
			{ID: "ai_vector", Label: "AI vector", Points: 15},
			{ID: "trad_vector", Label: "Traditional vector", Points: 10},
			{ID: "proof", Label: "Proof", Points: 5},
		}},
		{"custom", model.EngagementCustom, []model.FlagType{}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			eq(t, tc.want, model.DefaultFlagTypes(tc.typ))
		})
	}
}

func TestDefaultFlagTypesReturnsFreshSlice(t *testing.T) {
	first := model.DefaultFlagTypes(model.EngagementLab)
	first[0].Points = 99
	eq(t, 10, model.DefaultFlagTypes(model.EngagementLab)[0].Points)
}

func TestNewIDIsUniqueAndOpaque(t *testing.T) {
	seen := make(map[string]struct{}, idSampleCount)
	for range idSampleCount {
		id := model.NewID()
		check(t, id != "", "empty id")
		_, dup := seen[id]
		check(t, !dup, "duplicate id", id)
		seen[id] = struct{}{}
	}
}

func TestHostJSONFieldOrder(t *testing.T) {
	out, err := json.MarshalIndent(model.Host{ID: "x"}, "", "  ")
	noErr(t, err)
	s := string(out)
	check(t, strings.HasPrefix(s, "{\n  \"id\""), s)
	ap, sv, fl := strings.Index(s, `"accessPaths"`), strings.Index(s, `"services"`), strings.Index(s, `"flags"`)
	check(t, ap >= 0 && ap < sv && sv < fl, s)
}

func TestHostSampleRoundTrip(t *testing.T) {
	var h model.Host
	noErr(t, json.Unmarshal([]byte(sampleHostJSON), &h))
	eq(t, "nibbles", h.Hostname)
	eq(t, model.HostStatusFoothold, h.Status)
	eq(t, model.ItemDone, h.Services[0].Checklist.Items["http.robots"].Status)

	out, err := json.MarshalIndent(h, "", "  ")
	noErr(t, err)
	eq(t, sampleHostJSON, string(out))
}

func TestHostNormalizeFillsNilCollections(t *testing.T) {
	got := model.Host{Services: []model.Service{{}}}.Normalize()
	out, err := json.Marshal(got)
	noErr(t, err)
	s := string(out)
	for _, want := range []string{`"tags":[]`, `"accessPaths":[]`, `"flags":[]`, `"webPaths":[]`, `"checklist":{"items":{}}`} {
		check(t, strings.Contains(s, want), want, s)
	}
	check(t, !strings.Contains(s, "null"), s)
}

func TestHostNormalizeDefaultsServiceProtocol(t *testing.T) {
	got := model.Host{Services: []model.Service{{Port: 80}, {Port: 53, Protocol: "udp"}}}.Normalize()
	eq(t, "tcp", got.Services[0].Protocol)
	eq(t, "udp", got.Services[1].Protocol)
}

func TestHostNormalizeDoesNotMutateReceiver(t *testing.T) {
	h := model.Host{Services: []model.Service{{}}}
	_ = h.Normalize()
	check(t, h.Tags == nil)
	check(t, h.Services[0].WebPaths == nil)
}

func TestCredsNormalizeFillsNilCollections(t *testing.T) {
	got := model.Creds{Credentials: []model.Credential{{}}}.Normalize()
	out, err := json.Marshal(got)
	noErr(t, err)
	s := string(out)
	for _, want := range []string{`"users":[]`, `"shares":[]`, `"loot":[]`, `"worksOn":[]`, `"triedOn":[]`} {
		check(t, strings.Contains(s, want), want, s)
	}
	check(t, !strings.Contains(s, "null"), s)
}

func TestLookupHelpers(t *testing.T) {
	h := model.Host{IP: "10.0.0.1", Services: []model.Service{{ID: "s1", Port: 22}}}
	svc, found := h.Service("s1")
	check(t, found)
	eq(t, 22, svc.Port)
	_, found = h.Service("nope")
	check(t, !found)

	e := model.Engagement{Hosts: []model.Host{h}}
	got, found := e.Host("10.0.0.1")
	check(t, found)
	eq(t, "10.0.0.1", got.IP)
	_, found = e.Host("10.0.0.2")
	check(t, !found)
}

func TestServiceLabel(t *testing.T) {
	eq(t, "80/tcp http", model.Service{Port: 80, Protocol: "tcp", Type: "http"}.Label())
	eq(t, "22/tcp", model.Service{Port: 22, Protocol: "tcp"}.Label())
}

func TestEnumValidators(t *testing.T) {
	check(t, model.EngagementOSCP.IsValid())
	check(t, !model.EngagementType("").IsValid() && !model.EngagementType("x").IsValid())
	check(t, model.HostStatusRoot.IsValid())
	check(t, !model.HostStatus("").IsValid() && !model.HostStatus("pwned").IsValid())
	check(t, model.ItemNA.IsValid())
	check(t, !model.ItemStatus("").IsValid() && !model.ItemStatus("maybe").IsValid())
}

func TestHostByID(t *testing.T) {
	e := model.Engagement{Hosts: []model.Host{{ID: "h1", IP: "10.0.0.1"}}}
	got, found := e.HostByID("h1")
	check(t, found)
	eq(t, "10.0.0.1", got.IP)
	_, found = e.HostByID("h2")
	check(t, !found)
}

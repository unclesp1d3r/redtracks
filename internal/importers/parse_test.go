package importers

import "testing"

func TestDropInvalidHostsZonedIPv6IsDropped(t *testing.T) {
	res := dropInvalidHosts(Result{Hosts: []HostResult{{IP: "fe80::1%eth0"}, {IP: "10.10.10.5"}}})
	if len(res.Hosts) != 1 || res.Hosts[0].IP != "10.10.10.5" || res.Dropped != 1 {
		t.Fatalf("hosts=%+v dropped=%d, want only 10.10.10.5 and 1 dropped", res.Hosts, res.Dropped)
	}
}

func TestDropInvalidHostsCanonicalisesIPv6(t *testing.T) {
	res := dropInvalidHosts(Result{Hosts: []HostResult{{IP: "FE80::1"}, {IP: "fe80:0:0:0:0:0:0:1"}}})
	for _, h := range res.Hosts {
		if h.IP != "fe80::1" {
			t.Errorf("IP = %q, want fe80::1", h.IP)
		}
	}
}

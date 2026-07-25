package nettraffic

import (
	"testing"
	"time"
)

func TestParseFlows(t *testing.T) {
	t.Parallel()

	input := `IP 192.168.1.1.1234 > 8.8.8.8.53: udp 50
IP 8.8.8.8.53 > 192.168.1.1.1234: udp 100
IP 192.168.1.1.1234 > 8.8.8.8.53: udp 50
IP 10.0.0.1.80 > 10.0.0.2.5678: tcp 1400
ARP, Request who-has 192.168.1.1 tell 192.168.1.2, length 28
`
	got := parseFlows(input, 5*time.Second)
	if len(got) == 0 {
		t.Fatal("expected at least one flow, got none")
	}

	var dnsFlow *FlowSample
	for i := range got {
		if got[i].SrcAddr == "192.168.1.1" && got[i].DstAddr == "8.8.8.8" {
			dnsFlow = &got[i]
		}
	}
	if dnsFlow == nil {
		t.Fatal("expected to find flow 192.168.1.1 > 8.8.8.8")
		return
	}
	if dnsFlow.Protocol != "udp" {
		t.Errorf("want protocol=udp, got %s", dnsFlow.Protocol)
	}
	// 2 packets × 50 bytes = 100 bytes / 5s = 20 BPS
	if dnsFlow.Bps != 20 {
		t.Errorf("want Bps=20, got %v", dnsFlow.Bps)
	}
}

func TestStripPort(t *testing.T) {
	t.Parallel()

	cases := []struct{ input, want string }{
		{"192.168.1.1.80", "192.168.1.1"},
		{"10.0.0.1.12345", "10.0.0.1"},
		{"noporthere", "noporthere"},
	}
	for _, tc := range cases {
		if got := stripPort(tc.input); got != tc.want {
			t.Errorf("stripPort(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

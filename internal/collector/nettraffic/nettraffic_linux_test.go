//go:build linux

package nettraffic

import "testing"

// netDevFixture is a /proc/net/dev excerpt (iface: rx_bytes rx_pkts …×7 tx_bytes tx_pkts …×7).
const netDevFixture = "Inter-|Receive|Transmit\n" +
	" face |bytes pkts errs drop fifo frame comp mcast|bytes pkts errs drop fifo colls carr comp\n" +
	"    lo:  100000 1000 0 0 0 0 0 0  100000 1000 0 0 0 0 0 0\n" +
	"  eth0: 5000000 50000 0 0 0 0 0 0 1000000 10000 0 0 0 0 0 0\n" +
	"  eth1:       0 0 0 0 0 0 0 0       0 0 0 0 0 0 0 0\n"

func TestParseNetDev(t *testing.T) {
	t.Parallel()

	got, err := parseNetDev(netDevFixture)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("want 3 entries, got %d: %+v", len(got), got)
	}

	cases := []struct {
		iface   string
		rxBytes uint64
		txBytes uint64
	}{
		{"lo", 100000, 100000},
		{"eth0", 5000000, 1000000},
		{"eth1", 0, 0},
	}
	for i, tc := range cases {
		if got[i].iface != tc.iface || got[i].rxBytes != tc.rxBytes || got[i].txBytes != tc.txBytes {
			t.Errorf("entry %d: want {%s %d %d}, got {%s %d %d}",
				i, tc.iface, tc.rxBytes, tc.txBytes,
				got[i].iface, got[i].rxBytes, got[i].txBytes)
		}
	}
}

func TestParseNetDev_InvalidField(t *testing.T) {
	t.Parallel()

	// rx_bytes replaced with "notanumber" to trigger parse error.
	input := "    lo: notanumber 0 0 0 0 0 0 0  100000 0 0 0 0 0 0 0\n"
	_, err := parseNetDev(input)
	if err == nil {
		t.Fatal("expected error for invalid field, got nil")
	}
}

func TestDiffNetDev(t *testing.T) {
	t.Parallel()

	prev := []netStat{
		{iface: "eth0", rxBytes: 1000, txBytes: 2000},
		{iface: "lo", rxBytes: 500, txBytes: 500},
	}
	curr := []netStat{
		{iface: "eth0", rxBytes: 2500, txBytes: 3500}, // dRx=1500, dTx=1500 → total=3000
		{iface: "lo", rxBytes: 600, txBytes: 600},     // dRx=100, dTx=100 → total=200
	}

	got := diffNetDev(prev, curr)
	if len(got) != 2 {
		t.Fatalf("want 2 samples, got %d", len(got))
	}

	byIface := make(map[string]ProtocolSample, len(got))
	for _, s := range got {
		byIface[s.Interface] = s
	}

	if byIface["eth0"].BytesPerSec != 3000 {
		t.Errorf("eth0: want BytesPerSec=3000, got %d", byIface["eth0"].BytesPerSec)
	}
	if byIface["lo"].BytesPerSec != 200 {
		t.Errorf("lo: want BytesPerSec=200, got %d", byIface["lo"].BytesPerSec)
	}
}

func TestDiffNetDev_MissingInterface(t *testing.T) {
	t.Parallel()

	prev := []netStat{{iface: "eth0", rxBytes: 1000, txBytes: 500}}
	curr := []netStat{{iface: "eth1", rxBytes: 2000, txBytes: 1000}}

	got := diffNetDev(prev, curr)
	if len(got) != 0 {
		t.Errorf("want 0 samples for missing interface, got %d", len(got))
	}
}

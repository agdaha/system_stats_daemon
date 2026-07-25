//go:build darwin

package nettraffic

import "testing"

const netstatIBFixture = `Name       Mtu   Network       Address            Ipkts Ierrs     Ibytes    Opkts Oerrs     Obytes  Coll
lo0       16384 <Link#1>       lo0              100000     0  10000000   100000     0  10000000     0
lo0       16384 127            127.0.0.1        100000     -  10000000   100000     -  10000000     -
en0        1500 <Link#6>       aa:bb:cc:dd:ee:ff 200000     0   5000000   150000     0   3000000     0
en0        1500 10.0.0/24      10.0.0.1         200000     -   5000000   150000     -   3000000     -
`

func TestParseNetstatIB(t *testing.T) {
	t.Parallel()

	got, err := parseNetstatIB(netstatIBFixture)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 Link entries, got %d: %+v", len(got), got)
	}

	cases := []struct {
		iface   string
		rxBytes uint64
		txBytes uint64
	}{
		{"lo0", 10000000, 10000000},
		{"en0", 5000000, 3000000},
	}
	for i, tc := range cases {
		if got[i].iface != tc.iface {
			t.Errorf("[%d] want iface=%s, got %s", i, tc.iface, got[i].iface)
		}
		if got[i].rxBytes != tc.rxBytes {
			t.Errorf("[%d] %s: want rxBytes=%d, got %d", i, tc.iface, tc.rxBytes, got[i].rxBytes)
		}
		if got[i].txBytes != tc.txBytes {
			t.Errorf("[%d] %s: want txBytes=%d, got %d", i, tc.iface, tc.txBytes, got[i].txBytes)
		}
	}
}

func TestParseNetstatIB_InvalidField(t *testing.T) {
	t.Parallel()
	input := "lo0 16384 <Link#1> lo0 100 0 notanumber 100 0 100 0\n"
	_, err := parseNetstatIB(input)
	if err == nil {
		t.Fatal("expected error for invalid rx bytes")
	}
}

func TestDiffNetStats(t *testing.T) {
	t.Parallel()

	prev := []netStat{
		{iface: "en0", rxBytes: 1000, txBytes: 2000},
		{iface: "lo0", rxBytes: 500, txBytes: 500},
	}
	curr := []netStat{
		{iface: "en0", rxBytes: 2500, txBytes: 3500},
		{iface: "lo0", rxBytes: 600, txBytes: 600},
	}

	got := diffNetStats(prev, curr)
	if len(got) != 2 {
		t.Fatalf("want 2 samples, got %d", len(got))
	}

	byIface := make(map[string]ProtocolSample, len(got))
	for _, s := range got {
		byIface[s.Interface] = s
	}
	if byIface["en0"].BytesPerSec != 3000 {
		t.Errorf("en0: want BytesPerSec=3000, got %d", byIface["en0"].BytesPerSec)
	}
	if byIface["lo0"].BytesPerSec != 200 {
		t.Errorf("lo0: want BytesPerSec=200, got %d", byIface["lo0"].BytesPerSec)
	}
}

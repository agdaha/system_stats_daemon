//go:build darwin

package diskio

import "testing"

const iostatFixture = `              disk0       disk1
    KB/t  tps  MB/s      KB/t  tps  MB/s
   64.45  17   1.07     91.88  11   1.00
    0.00   2   0.50      4.00   1   0.25
`

func TestParseIOStat(t *testing.T) {
	t.Parallel()

	got, err := parseIOStat(iostatFixture)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 devices, got %d: %+v", len(got), got)
	}

	cases := []struct {
		device string
		tps    float32
		kbps   float32
	}{
		{"disk0", 2, 0.50 * 1024},
		{"disk1", 1, 0.25 * 1024},
	}
	for i, tc := range cases {
		if got[i].Device != tc.device {
			t.Errorf("[%d] want device=%s, got %s", i, tc.device, got[i].Device)
		}
		if got[i].TPS != tc.tps {
			t.Errorf("[%d] want TPS=%.2f, got %.2f", i, tc.tps, got[i].TPS)
		}
		if got[i].KBps != tc.kbps {
			t.Errorf("[%d] want KBps=%.2f, got %.2f", i, tc.kbps, got[i].KBps)
		}
	}
}

func TestParseIOStat_TooFewLines(t *testing.T) {
	t.Parallel()
	_, err := parseIOStat("disk0\nKB/t tps MB/s\n")
	if err == nil {
		t.Fatal("expected error for too few lines")
	}
}

func TestParseIOStat_TooFewFields(t *testing.T) {
	t.Parallel()
	input := "disk0\nKB/t tps MB/s\n1.00 2\n0.00 1\n"
	_, err := parseIOStat(input)
	if err == nil {
		t.Fatal("expected error for too few fields")
	}
}

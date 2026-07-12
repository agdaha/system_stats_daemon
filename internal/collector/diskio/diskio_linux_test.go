//go:build linux

package diskio

import "testing"

func TestParseDiskStats(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		input   string
		want    []diskStat
		wantErr bool
	}{
		{
			name: "nominal",
			input: "   8       0 sda 1000 200 5000 300 2000 400 8000 600 0 100 900 0 0 0 0\n" +
				"   8       1 sda1 800 100 4000 200 1000 200 6000 400 0 50 600 0 0 0 0\n",
			want: []diskStat{
				{device: "sda", readsCompleted: 1000, sectorsRead: 5000, writesCompleted: 2000, sectorsWritten: 8000},
				{device: "sda1", readsCompleted: 800, sectorsRead: 4000, writesCompleted: 1000, sectorsWritten: 6000},
			},
		},
		{
			name: "skips ram and loop",
			input: "   1   0 ram0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0\n" +
				"   7   0 loop0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0\n",
			want: nil,
		},
		{
			name:  "empty input",
			input: "",
			want:  nil,
		},
		{
			name:    "invalid reads field",
			input:   "   8   0 sda foo 200 5000 300 2000 400 8000 600 0 100 900\n",
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseDiskStats(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("want %d entries, got %d: %+v", len(tc.want), len(got), got)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("entry %d: want %+v, got %+v", i, tc.want[i], got[i])
				}
			}
		})
	}
}

func TestComputeDiff(t *testing.T) {
	t.Parallel()

	// dOps = (1010+2020) - (1000+2000) = 30, dSectors = (5100+8200) - (5000+8000) = 300 → KBps = 150
	prev := []diskStat{
		{device: "sda", readsCompleted: 1000, sectorsRead: 5000, writesCompleted: 2000, sectorsWritten: 8000},
	}
	curr := []diskStat{
		{device: "sda", readsCompleted: 1010, sectorsRead: 5100, writesCompleted: 2020, sectorsWritten: 8200},
	}

	got := computeDiff(prev, curr)
	if len(got) != 1 {
		t.Fatalf("want 1 sample, got %d", len(got))
	}
	if got[0].TPS != 30 {
		t.Errorf("want TPS=30, got %v", got[0].TPS)
	}
	if got[0].KBps != 150 {
		t.Errorf("want KBps=150, got %v", got[0].KBps)
	}
}

func TestComputeDiff_MissingDevice(t *testing.T) {
	t.Parallel()

	prev := []diskStat{{device: "sda", readsCompleted: 100}}
	curr := []diskStat{{device: "sdb", readsCompleted: 100}}

	got := computeDiff(prev, curr)
	if len(got) != 0 {
		t.Errorf("want 0 samples for missing prev device, got %d", len(got))
	}
}

func TestComputeDiff_NegativeClamped(t *testing.T) {
	t.Parallel()

	// Counter wrap (or clock drift): curr < prev → clamp to 0.
	prev := []diskStat{{device: "sda", readsCompleted: 1000, sectorsRead: 5000, writesCompleted: 0, sectorsWritten: 0}}
	curr := []diskStat{{device: "sda", readsCompleted: 500, sectorsRead: 4000, writesCompleted: 0, sectorsWritten: 0}}

	got := computeDiff(prev, curr)
	if len(got) != 1 {
		t.Fatalf("want 1 sample, got %d", len(got))
	}
	if got[0].TPS != 0 || got[0].KBps != 0 {
		t.Errorf("want TPS=0 KBps=0 after clamp, got TPS=%v KBps=%v", got[0].TPS, got[0].KBps)
	}
}

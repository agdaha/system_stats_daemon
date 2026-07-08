//go:build linux

package cpu

import "testing"

func TestParseCPUStat(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		input   string
		want    cpuStat
		wantErr bool
	}{
		{
			name:  "nominal",
			input: "cpu  1234 56 789 90000 100 10 20 5 0 0\ncpu0 617 28 394 45000 50 5 10 2 0 0\n",
			want: cpuStat{
				user: 1234, nice: 56, system: 789, idle: 90000,
				iowait: 100, irq: 10, softirq: 20, steal: 5,
			},
		},
		{
			name:    "no cpu aggregate line",
			input:   "intr 1234\nbtime 5678\n",
			wantErr: true,
		},
		{
			name:    "too few fields",
			input:   "cpu 1234 56\n",
			wantErr: true,
		},
		{
			name:    "invalid value",
			input:   "cpu  foo 56 789 90000 0 0 0 0\n",
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseCPUStat(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("want %+v, got %+v", tc.want, got)
			}
		})
	}
}

func TestComputeDiff(t *testing.T) {
	t.Parallel()

	// prev total = 10000, curr total = 10200, delta = 200
	// dUser = 100, dSystem = 20, dIdle = 80
	prev := cpuStat{user: 1000, system: 200, idle: 8800}
	curr := cpuStat{user: 1100, system: 220, idle: 8880}

	s, ok := computeDiff(prev, curr)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if s.User != 50 {
		t.Errorf("want User=50, got %v", s.User)
	}
	if s.System != 10 {
		t.Errorf("want System=10, got %v", s.System)
	}
	if s.Idle != 40 {
		t.Errorf("want Idle=40, got %v", s.Idle)
	}
}

func TestComputeDiff_ZeroDelta(t *testing.T) {
	t.Parallel()
	s := cpuStat{user: 1000, system: 200, idle: 8800}
	_, ok := computeDiff(s, s)
	if ok {
		t.Error("expected ok=false when total delta is zero")
	}
}

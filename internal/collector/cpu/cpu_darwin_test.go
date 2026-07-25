//go:build darwin

package cpu

import "testing"

const topFixture = `Processes: 398 total, 2 running, 396 sleeping, 1716 threads
2024/07/25 10:15:35
Load Avg: 1.23, 0.45, 0.12
CPU usage: 5.12% user, 2.34% sys, 92.54% idle
SharedLibs: 450M resident, 96M data, 58M linkedit.
`

func TestParseCPU(t *testing.T) {
	t.Parallel()

	got, err := parseCPU(topFixture)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := Sample{User: 5.12, System: 2.34, Idle: 92.54}
	if got != want {
		t.Errorf("want %+v, got %+v", want, got)
	}
}

func TestParseCPULine(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		input   string
		want    Sample
		wantErr bool
	}{
		{
			name:  "nominal",
			input: "CPU usage: 5.12% user, 2.34% sys, 92.54% idle",
			want:  Sample{User: 5.12, System: 2.34, Idle: 92.54},
		},
		{
			name:  "zero",
			input: "CPU usage: 0.00% user, 0.00% sys, 100.00% idle",
			want:  Sample{User: 0, System: 0, Idle: 100},
		},
		{
			name:    "too few fields",
			input:   "CPU usage: 5.12% user",
			wantErr: true,
		},
		{
			name:    "invalid user",
			input:   "CPU usage: foo% user, 2.34% sys, 92.54% idle",
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseCPULine(tc.input)
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

func TestParseCPU_NoLine(t *testing.T) {
	t.Parallel()
	_, err := parseCPU("no relevant lines here\n")
	if err == nil {
		t.Fatal("expected error for missing CPU usage line")
	}
}

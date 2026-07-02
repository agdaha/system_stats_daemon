//go:build linux

package loadavg

import "testing"

func TestParseLoadAvg(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		input   string
		want    Sample
		wantErr bool
	}{
		{
			name:  "nominal",
			input: "0.52 0.58 0.59 1/432 12345\n",
			want:  Sample{One: 0.52, Five: 0.58, Fifteen: 0.59},
		},
		{
			name:  "high load",
			input: "4.00 3.50 2.00 5/100 999\n",
			want:  Sample{One: 4.00, Five: 3.50, Fifteen: 2.00},
		},
		{
			name:  "zero load",
			input: "0.00 0.00 0.00 0/1 1\n",
			want:  Sample{},
		},
		{
			name:    "empty input",
			input:   "",
			wantErr: true,
		},
		{
			name:    "too few fields",
			input:   "0.52 0.58",
			wantErr: true,
		},
		{
			name:    "invalid 1m",
			input:   "foo 0.58 0.59 1/1 1",
			wantErr: true,
		},
		{
			name:    "invalid 5m",
			input:   "0.52 bar 0.59 1/1 1",
			wantErr: true,
		},
		{
			name:    "invalid 15m",
			input:   "0.52 0.58 baz 1/1 1",
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseLoadAvg(tc.input)
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

//go:build darwin

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
			input: "{ 1.23 0.45 0.12 }\n",
			want:  Sample{One: 1.23, Five: 0.45, Fifteen: 0.12},
		},
		{
			name:  "zero load",
			input: "{ 0.00 0.00 0.00 }",
			want:  Sample{},
		},
		{
			name:    "empty braces",
			input:   "{  }",
			wantErr: true,
		},
		{
			name:    "invalid 1m",
			input:   "{ foo 0.45 0.12 }",
			wantErr: true,
		},
		{
			name:    "invalid 5m",
			input:   "{ 1.23 bar 0.12 }",
			wantErr: true,
		},
		{
			name:    "invalid 15m",
			input:   "{ 1.23 0.45 baz }",
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

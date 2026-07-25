//go:build darwin

package netsockets

import "testing"

const tcpFixture = `Active Internet connections (including servers)
Proto Recv-Q Send-Q  Local Address          Foreign Address        (state)
tcp4       0      0  127.0.0.1.8080         *.*                    LISTEN
tcp46      0      0  *.5432                 *.*                    LISTEN
tcp4       0      0  10.0.0.1.12345         10.0.0.2.67890        ESTABLISHED
tcp4       0      0  10.0.0.1.54321         10.0.0.2.80           TIME_WAIT
tcp4       0      0  10.0.0.1.54322         10.0.0.2.80           TIME_WAIT
`

const udpFixture = `Active Internet connections (including servers)
Proto Recv-Q Send-Q  Local Address          Foreign Address
udp4       0      0  *.*                    *.*
udp4       0      0  *.53                   *.*
`

func TestParseTCPStates(t *testing.T) {
	t.Parallel()

	got := parseTCPStates(tcpFixture)
	if len(got) == 0 {
		t.Fatal("expected at least one state")
	}

	byState := make(map[string]uint32, len(got))
	for _, s := range got {
		byState[s.State] = s.Count
	}

	if byState["LISTEN"] != 2 {
		t.Errorf("want LISTEN=2, got %d", byState["LISTEN"])
	}
	if byState["ESTABLISHED"] != 1 {
		t.Errorf("want ESTABLISHED=1, got %d", byState["ESTABLISHED"])
	}
	if byState["TIME_WAIT"] != 2 {
		t.Errorf("want TIME_WAIT=2, got %d", byState["TIME_WAIT"])
	}
}

func TestParseTCPListeners(t *testing.T) {
	t.Parallel()

	got := parseTCPListeners(tcpFixture)
	if len(got) != 2 {
		t.Fatalf("want 2 listeners, got %d: %+v", len(got), got)
	}

	byPort := make(map[uint32]ListenSample, len(got))
	for _, s := range got {
		byPort[s.Port] = s
	}

	if _, ok := byPort[8080]; !ok {
		t.Error("expected port 8080 in listeners")
	}
	if _, ok := byPort[5432]; !ok {
		t.Error("expected port 5432 in listeners")
	}
}

func TestParseUDPListeners(t *testing.T) {
	t.Parallel()

	got := parseUDPListeners(udpFixture)
	if len(got) != 1 {
		t.Fatalf("want 1 udp listener (port 53), got %d: %+v", len(got), got)
	}
	if got[0].Port != 53 {
		t.Errorf("want port 53, got %d", got[0].Port)
	}
}

func TestExtractPort(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input   string
		want    uint32
		wantErr bool
	}{
		{"127.0.0.1.8080", 8080, false},
		{"*.5432", 5432, false},
		{"*.*", 0, true},
		{"*.* ", 0, true},
		{"noport", 0, true},
	}
	for _, tc := range cases {
		got, err := extractPort(tc.input)
		if tc.wantErr {
			if err == nil {
				t.Errorf("extractPort(%q): expected error", tc.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("extractPort(%q): unexpected error: %v", tc.input, err)
			continue
		}
		if got != tc.want {
			t.Errorf("extractPort(%q) = %d, want %d", tc.input, got, tc.want)
		}
	}
}

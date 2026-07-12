//go:build linux

package netsockets

import "testing"

const sstaFixture = `State     Recv-Q Send-Q Local Address:Port   Peer Address:Port
LISTEN    0      200        127.0.0.1:54339            0.0.0.0:*
LISTEN    0      4096         0.0.0.0:1433             0.0.0.0:*
ESTAB     0      0          127.0.0.1:10808          127.0.0.1:36060
ESTAB     0      0          127.0.0.1:10808          127.0.0.1:36652
TIME-WAIT 0      0          127.0.0.1:36058          127.0.0.1:10808
`

const netstatFixture = `Active Internet connections (only servers)
Proto Recv-Q Send-Q Local Address               Foreign Address             State       PID/Program name
tcp        0      0 127.0.0.1:8828              0.0.0.0:*                   LISTEN      1140481/VSCodium
tcp        0      0 0.0.0.0:5432                0.0.0.0:*                   LISTEN      -
tcp        0      0 0.0.0.0:1433                0.0.0.0:*                   LISTEN      1234/sqlserver
udp        0      0 0.0.0.0:43094               0.0.0.0:*                               149063/python3
`

func TestParseSSTA(t *testing.T) {
	t.Parallel()

	got := parseSSTA(sstaFixture)
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
	if byState["ESTAB"] != 2 {
		t.Errorf("want ESTAB=2, got %d", byState["ESTAB"])
	}
	if byState["TIME-WAIT"] != 1 {
		t.Errorf("want TIME-WAIT=1, got %d", byState["TIME-WAIT"])
	}
}

func TestParseNetstat(t *testing.T) {
	t.Parallel()

	got := parseNetstat(netstatFixture)
	if len(got) != 4 {
		t.Fatalf("want 4 entries, got %d: %+v", len(got), got)
	}

	byPort := make(map[uint32]ListenSample, len(got))
	for _, s := range got {
		byPort[s.Port] = s
	}

	vscodium := byPort[8828]
	if vscodium.Protocol != "tcp" || vscodium.PID != 1140481 || vscodium.Command != "VSCodium" {
		t.Errorf("port 8828: want {tcp 1140481 VSCodium}, got %+v", vscodium)
	}

	postgres := byPort[5432]
	if postgres.Protocol != "tcp" || postgres.PID != 0 || postgres.Command != "" {
		t.Errorf("port 5432: want {tcp 0 ''}, got %+v", postgres)
	}

	python := byPort[43094]
	if python.Protocol != "udp" || python.PID != 149063 || python.Command != "python3" {
		t.Errorf("port 43094: want {udp 149063 python3}, got %+v", python)
	}
}

func TestExtractPort(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  uint32
	}{
		{"127.0.0.1:8828", 8828},
		{"0.0.0.0:5432", 5432},
		{"[::1]:631", 631},
		{"0.0.0.0:*", 0}, // wildcard fails gracefully (err ignored by caller)
	}
	for _, tc := range cases {
		got, _ := extractPort(tc.input)
		if got != tc.want {
			t.Errorf("extractPort(%q) = %d, want %d", tc.input, got, tc.want)
		}
	}
}

func TestParsePIDProg(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input   string
		wantPID uint32
		wantCmd string
	}{
		{"1140481/VSCodium", 1140481, "VSCodium"},
		{"149063/python3", 149063, "python3"},
		{"-", 0, ""},
		{"invalidpid/prog", 0, "prog"},
	}
	for _, tc := range cases {
		pid, cmd := parsePIDProg(tc.input)
		if pid != tc.wantPID || cmd != tc.wantCmd {
			t.Errorf("parsePIDProg(%q) = {%d %q}, want {%d %q}",
				tc.input, pid, cmd, tc.wantPID, tc.wantCmd)
		}
	}
}

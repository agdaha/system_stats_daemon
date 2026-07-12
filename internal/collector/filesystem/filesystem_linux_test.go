//go:build linux

package filesystem

import "testing"

const dfkFixture = `Filesystem     1024-blocks      Used Available Capacity Mounted on
udevfs                5120         0      5120        0% /dev
/dev/sda1        110326172  51200000  59126172       47% /
tmpfs              8052332     27024   8025308        1% /tmp
`

const dfiFixture = `Filesystem      Inodes   IUsed   IFree IUse% Mounted on
udevfs         1995960     627 1995333    1% /dev
/dev/sda1      7045120 1073710 5971410   16% /
efivarfs             0       0       0     - /boot/efi
tmpfs          2013083    1007 2012076    1% /tmp
`

func TestParseDFk(t *testing.T) {
	t.Parallel()

	got, err := parseDFk(dfkFixture)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("want 3 entries, got %d", len(got))
	}

	cases := []struct {
		idx        int
		filesystem string
		usedKB     uint64
		pct        float32
		mount      string
	}{
		{0, "udevfs", 0, 0, "/dev"},
		{1, "/dev/sda1", 51200000, 47, "/"},
		{2, "tmpfs", 27024, 1, "/tmp"},
	}
	for _, tc := range cases {
		e := got[tc.idx]
		if e.filesystem != tc.filesystem || e.usedKB != tc.usedKB || e.usedPercent != tc.pct || e.mountPoint != tc.mount {
			t.Errorf("entry %d: want {%s %d %g %s}, got {%s %d %g %s}",
				tc.idx, tc.filesystem, tc.usedKB, tc.pct, tc.mount,
				e.filesystem, e.usedKB, e.usedPercent, e.mountPoint)
		}
	}
}

func TestParseDFi(t *testing.T) {
	t.Parallel()

	got, err := parseDFi(dfiFixture)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 4 {
		t.Fatalf("want 4 entries, got %d", len(got))
	}

	efi := got[2]
	if efi.mountPoint != "/boot/efi" || efi.usedInodes != 0 || efi.usedInodesPercent != 0 {
		t.Errorf("efi entry: want {/boot/efi 0 0}, got {%s %d %g}",
			efi.mountPoint, efi.usedInodes, efi.usedInodesPercent)
	}

	root := got[1]
	if root.mountPoint != "/" || root.usedInodes != 1073710 || root.usedInodesPercent != 16 {
		t.Errorf("root entry: want {/ 1073710 16}, got {%s %d %g}",
			root.mountPoint, root.usedInodes, root.usedInodesPercent)
	}
}

func TestMerge(t *testing.T) {
	t.Parallel()

	space := []dfkEntry{
		{filesystem: "/dev/sda1", usedKB: 2048, usedPercent: 50, mountPoint: "/"},
		{filesystem: "tmpfs", usedKB: 1024, usedPercent: 10, mountPoint: "/tmp"},
	}
	inodes := []dfiEntry{
		{mountPoint: "/", usedInodes: 1000, usedInodesPercent: 20},
	}

	got := merge(space, inodes)
	if len(got) != 2 {
		t.Fatalf("want 2 samples, got %d", len(got))
	}

	root := got[0]
	if root.UsedMB != 2 || root.UsedPercent != 50 || root.UsedInodes != 1000 || root.UsedInodesPercent != 20 {
		t.Errorf("root: %+v", root)
	}

	tmp := got[1]
	if tmp.UsedMB != 1 || tmp.UsedInodes != 0 {
		t.Errorf("tmp: %+v", tmp)
	}
}

func TestParsePercent(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input   string
		want    float32
		wantErr bool
	}{
		{"42%", 42, false},
		{"0%", 0, false},
		{"100%", 100, false},
		{"-", 0, false},
		{"abc%", 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			got, err := parsePercent(tc.input)
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
				t.Errorf("want %v, got %v", tc.want, got)
			}
		})
	}
}

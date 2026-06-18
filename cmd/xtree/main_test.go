package main

import "testing"

func TestConnector(t *testing.T) {
	if got := connector(true); got != "└── " {
		t.Errorf("connector(true) = %q", got)
	}
	if got := connector(false); got != "├── " {
		t.Errorf("connector(false) = %q", got)
	}
}

func TestTreePrefix(t *testing.T) {
	cases := []struct {
		name         string
		ancestorLast []bool
		want         string
	}{
		{"top level", nil, ""},
		{"under continuing parent", []bool{false}, "│   "},
		{"under last parent", []bool{true}, "    "},
		{"two levels mixed", []bool{false, true}, "│       "},
		{"two levels last,continuing", []bool{true, false}, "    │   "},
	}
	for _, c := range cases {
		if got := treePrefix(c.ancestorLast); got != c.want {
			t.Errorf("%s: treePrefix(%v) = %q, want %q", c.name, c.ancestorLast, got, c.want)
		}
	}
}

func TestCountDirsFiles(t *testing.T) {
	cases := []struct {
		n            int
		wantD, wantF string
	}{
		{0, "0 directories", "0 files"},
		{1, "1 directory", "1 file"},
		{2, "2 directories", "2 files"},
	}
	for _, c := range cases {
		if got := countDirs(c.n); got != c.wantD {
			t.Errorf("countDirs(%d) = %q, want %q", c.n, got, c.wantD)
		}
		if got := countFiles(c.n); got != c.wantF {
			t.Errorf("countFiles(%d) = %q, want %q", c.n, got, c.wantF)
		}
	}
}

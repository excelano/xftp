package main

import (
	"testing"

	"github.com/excelano/xfiles/internal/drive"
)

func TestCriteriaMatch(t *testing.T) {
	file := drive.Item{Name: "Q1 Report.xlsx"}
	folder := drive.Item{Name: "Reports", IsFolder: true}
	cases := []struct {
		name    string
		c       criteria
		it      drive.Item
		want    bool
		wantErr bool
	}{
		{"empty matches file", criteria{}, file, true, false},
		{"empty matches folder", criteria{}, folder, true, false},
		{"type f keeps file", criteria{typ: "f"}, file, true, false},
		{"type f drops folder", criteria{typ: "f"}, folder, false, false},
		{"type d keeps folder", criteria{typ: "d"}, folder, true, false},
		{"type d drops file", criteria{typ: "d"}, file, false, false},
		{"name glob hit", criteria{namePat: "*.xlsx"}, file, true, false},
		{"name glob miss", criteria{namePat: "*.docx"}, file, false, false},
		{"name is case-sensitive", criteria{namePat: "*.XLSX"}, file, false, false},
		{"iname ignores case", criteria{namePat: "*.XLSX", iname: true}, file, true, false},
		{"name and type together", criteria{namePat: "*.xlsx", typ: "f"}, file, true, false},
		{"type wins over name", criteria{namePat: "*.xlsx", typ: "d"}, file, false, false},
		{"bad pattern errors", criteria{namePat: "[a-"}, file, false, true},
	}
	for _, c := range cases {
		got, err := c.c.match(c.it)
		if (err != nil) != c.wantErr {
			t.Errorf("%s: match err = %v, wantErr %v", c.name, err, c.wantErr)
			continue
		}
		if err == nil && got != c.want {
			t.Errorf("%s: match = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestRelTo(t *testing.T) {
	cases := []struct {
		name, root, full, want string
	}{
		{"root library", "", "Reports/Q1.xlsx", "Reports/Q1.xlsx"},
		{"strips folder root", "Reports", "Reports/Q1.xlsx", "Q1.xlsx"},
		{"strips nested root", "Shared Documents/General", "Shared Documents/General/Phase 2/x.txt", "Phase 2/x.txt"},
		{"item equal to root", "Reports", "Reports", ""},
	}
	for _, c := range cases {
		if got := relTo(c.root, c.full); got != c.want {
			t.Errorf("%s: relTo(%q,%q) = %q, want %q", c.name, c.root, c.full, got, c.want)
		}
	}
}

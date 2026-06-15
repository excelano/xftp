package main

import (
	"reflect"
	"testing"
)

func TestResolveRemote(t *testing.T) {
	cases := []struct {
		cwd, arg, want string
	}{
		{"", "", ""},
		{"Docs", "", "Docs"},
		{"", "Docs", "Docs"},
		{"Docs", "Reports", "Docs/Reports"},
		{"Docs", "/Reports", "Reports"},
		{"Docs/Sub", "..", "Docs"},
		{"Docs/Sub", "../..", ""},
		{"Docs", ".", "Docs"},
		{"Docs", "/", ""},
		{"Docs", "Sub/../Other", "Docs/Other"},
		{"", "/a/b/c", "a/b/c"},
	}
	for _, c := range cases {
		if got := resolveRemote(c.cwd, c.arg); got != c.want {
			t.Errorf("resolveRemote(%q, %q) = %q, want %q", c.cwd, c.arg, got, c.want)
		}
	}
}

func TestTokenize(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"ls", []string{"ls"}},
		{"  ls   Docs ", []string{"ls", "Docs"}},
		{`get "Q1 Plan.xlsx"`, []string{"get", "Q1 Plan.xlsx"}},
		{`mv "a b" "c d"`, []string{"mv", "a b", "c d"}},
		{`put report.txt "Shared Docs/report.txt"`, []string{"put", "report.txt", "Shared Docs/report.txt"}},
	}
	for _, c := range cases {
		got := tokenize(c.in)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("tokenize(%q) = %#v, want %#v", c.in, got, c.want)
		}
	}
}

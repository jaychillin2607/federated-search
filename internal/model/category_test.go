package model

import "testing"

func TestIsValid(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"turf", true},
		{"gym", true},
		{"coach", true},
		{"class", true},
		{"feature", true},
		{"", false},
		{"unknown", false},
		{"TURF", false},
	}
	for _, c := range cases {
		if got := IsValid(c.in); got != c.want {
			t.Errorf("IsValid(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestAll(t *testing.T) {
	got := All()
	if len(got) != 5 {
		t.Fatalf("All() len = %d, want 5", len(got))
	}
	// mutating the returned slice should not affect internal state
	got[0] = "mutated"
	if All()[0] == "mutated" {
		t.Fatalf("All() returned a slice backed by internal state")
	}
}

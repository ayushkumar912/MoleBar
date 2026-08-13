package main

import "testing"

func TestNormalizeDisplayMode(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "default", input: "", want: "sys"},
		{name: "system", input: "sys", want: "sys"},
		{name: "network", input: "net", want: "net"},
		{name: "both", input: "both", want: "both"},
		{name: "invalid", input: "bogus", want: "sys"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeDisplayMode(tt.input); got != tt.want {
				t.Fatalf("normalizeDisplayMode(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

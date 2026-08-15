package config

import "testing"

func TestParseDisplayMode(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    DisplayMode
		wantErr bool
	}{
		{name: "default empty", input: "", want: DisplayModeSys},
		{name: "sys", input: "sys", want: DisplayModeSys},
		{name: "system alias", input: "system", want: DisplayModeSys},
		{name: "net", input: "net", want: DisplayModeNet},
		{name: "network alias", input: "network", want: DisplayModeNet},
		{name: "both", input: "both", want: DisplayModeBoth},
		{name: "whitespace and case", input: "  BOTH  ", want: DisplayModeBoth},
		{name: "invalid", input: "bogus", want: DisplayModeSys, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDisplayMode(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseDisplayMode(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("ParseDisplayMode(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeDisplayModeDefault(t *testing.T) {
	if got := NormalizeDisplayMode("nope"); got != DefaultDisplayMode {
		t.Fatalf("NormalizeDisplayMode(invalid) = %q, want %q", got, DefaultDisplayMode)
	}
	if DefaultDisplayMode != DisplayModeSys {
		t.Fatalf("DefaultDisplayMode = %q, want sys", DefaultDisplayMode)
	}
}

func TestDisplayModeStringAndLabel(t *testing.T) {
	if DisplayModeNet.String() != "net" {
		t.Fatalf("String() = %q", DisplayModeNet.String())
	}
	if DisplayModeNet.Label() != "Network" {
		t.Fatalf("Label() = %q", DisplayModeNet.Label())
	}
	if DisplayMode("").Label() != "System" {
		t.Fatalf("empty Label() = %q", DisplayMode("").Label())
	}
}

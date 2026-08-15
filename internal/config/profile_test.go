package config

import "testing"

func TestBuiltInProfileLabels(t *testing.T) {
	want := []string{"Minimal", "Developer", "Network", "Battery", "Full"}
	got := BuiltInProfiles()
	if len(got) != len(want) {
		t.Fatalf("len = %d", len(got))
	}
	for i, p := range got {
		if p.Label != want[i] {
			t.Fatalf("label[%d] = %q", i, p.Label)
		}
		if len(NormalizeLayout(p.Layout).Metrics) == 0 {
			t.Fatalf("empty layout for %s", p.ID)
		}
	}
}

func TestParseProfileID(t *testing.T) {
	id, err := ParseProfileID(" Developer ")
	if err != nil || id != ProfileDeveloper {
		t.Fatalf("id=%q err=%v", id, err)
	}
	id, err = ParseProfileID("nope")
	if err == nil || id != ProfileCustom {
		t.Fatalf("invalid id=%q err=%v", id, err)
	}
}

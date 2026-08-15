package config

import "testing"

func TestNormalizeLayoutValidMetrics(t *testing.T) {
	got := NormalizeLayout(TrayLayout{Metrics: []Metric{MetricCPU, MetricRX}})
	if len(got.Metrics) != 2 || got.Metrics[0] != MetricCPU || got.Metrics[1] != MetricRX {
		t.Fatalf("got %+v", got)
	}
	if got.Separator != defaultSeparator {
		t.Fatalf("separator = %q", got.Separator)
	}
}

func TestNormalizeLayoutInvalidAndDuplicateMetrics(t *testing.T) {
	got := NormalizeLayout(TrayLayout{Metrics: []Metric{MetricCPU, "nope", MetricCPU, MetricMemory}})
	if len(got.Metrics) != 2 || got.Metrics[0] != MetricCPU || got.Metrics[1] != MetricMemory {
		t.Fatalf("got %+v", got)
	}
}

func TestNormalizeLayoutEmptyBecomesDefault(t *testing.T) {
	got := NormalizeLayout(TrayLayout{})
	if !got.Equal(LayoutFromDisplayMode(DefaultDisplayMode)) {
		t.Fatalf("empty layout = %+v", got)
	}
}

func TestLayoutFromDisplayModeMigration(t *testing.T) {
	sys := LayoutFromDisplayMode(DisplayModeSys)
	if !sys.Equal(TrayLayout{Metrics: []Metric{MetricCPU, MetricMemory}}) {
		t.Fatalf("sys = %+v", sys)
	}
	net := LayoutFromDisplayMode(DisplayModeNet)
	if !net.Equal(TrayLayout{Metrics: []Metric{MetricRX, MetricTX}}) {
		t.Fatalf("net = %+v", net)
	}
	both := LayoutFromDisplayMode(DisplayModeBoth)
	if !both.Equal(TrayLayout{Metrics: []Metric{MetricCPU, MetricMemory, MetricRX, MetricTX}}) {
		t.Fatalf("both = %+v", both)
	}
}

func TestToggleMetricKeepsAtLeastOne(t *testing.T) {
	l := TrayLayout{Metrics: []Metric{MetricCPU}}
	got := l.ToggleMetric(MetricCPU)
	if !got.Equal(l) {
		t.Fatalf("removed last metric: %+v", got)
	}
	got = l.ToggleMetric(MetricMemory)
	if !got.Contains(MetricCPU) || !got.Contains(MetricMemory) {
		t.Fatalf("add failed: %+v", got)
	}
}

func TestProfileResolution(t *testing.T) {
	tests := []struct {
		id   string
		want []Metric
	}{
		{"minimal", []Metric{MetricCPU}},
		{"developer", []Metric{MetricCPU, MetricMemory, MetricRX, MetricTX}},
		{"network", []Metric{MetricRX, MetricTX}},
		{"battery", []Metric{MetricBattery, MetricTemperature, MetricCPU}},
		{"full", []Metric{MetricHealth, MetricCPU, MetricMemory, MetricRX, MetricTX}},
	}
	for _, tt := range tests {
		layout, ok := ResolveProfileLayout(tt.id)
		if !ok {
			t.Fatalf("profile %s missing", tt.id)
		}
		if !layout.Equal(TrayLayout{Metrics: tt.want}) {
			t.Fatalf("%s = %+v", tt.id, layout)
		}
	}
	if _, ok := ResolveProfileLayout("nope"); ok {
		t.Fatal("invalid profile should not resolve")
	}
}

func TestMatchingProfile(t *testing.T) {
	if got := MatchingProfile(LayoutFromDisplayMode(DisplayModeNet)); got != ProfileNetwork {
		t.Fatalf("net profile = %q", got)
	}
	if got := MatchingProfile(TrayLayout{Metrics: []Metric{MetricCPU, MetricDisk}}); got != ProfileCustom {
		t.Fatalf("custom = %q", got)
	}
}

func TestPreferencesApplyProfileAndToggle(t *testing.T) {
	p := DefaultPreferences()
	if !p.ApplyProfile("developer") {
		t.Fatal("developer")
	}
	if p.Profile != "developer" {
		t.Fatalf("profile = %q", p.Profile)
	}
	p.ApplyMetricToggle(MetricHealth)
	if p.Profile != "custom" {
		t.Fatalf("after toggle profile = %q", p.Profile)
	}
	if !p.Layout.Contains(MetricHealth) {
		t.Fatal("health not added")
	}
}

package diagnostics

import (
	"strings"
	"testing"
	"time"

	"github.com/ayush-kumar912/molebar/internal/history"
	"github.com/ayush-kumar912/molebar/internal/molestatus"
	"github.com/ayush-kumar912/molebar/internal/session"
)

func sampleStatus() *molestatus.Status {
	score := 92
	s := &molestatus.Status{
		HealthScore: &score,
		HealthMsg:   "Excellent",
		Network:     []molestatus.Network{{Name: "en0", RxRateMBs: 1.5, TxRateMBs: 0.25, IP: "10.0.0.8"}},
		TopProcesses: []molestatus.ProcessInfo{
			{PID: 1, Name: "Cursor", CPU: 40, MemoryBytes: 1},
			{PID: 2, Name: "Chrome", CPU: 20, MemoryBytes: 2},
		},
	}
	s.CPU.Usage = 31
	s.Memory.UsedPercent = 62
	s.Disks = []molestatus.Disk{{Mount: "/", UsedPercent: 71}}
	s.Batteries = []molestatus.Battery{{Percent: 83, Status: "AC", Health: "Good", CycleCount: 120}}
	s.Thermal.CPUTemp = 58
	return s
}

func sampleInput() Input {
	return Input{
		MoleBarVersion: "1.2.3",
		MoleVersion:    "mo 0.9",
		OSName:         "darwin",
		OSVersion:      "25.6.0",
		Arch:           "arm64",
		Strategy:       "watch",
		Profile:        "developer",
		Interval:       5 * time.Second,
		Capabilities:   molestatus.Capabilities{Version: "mo 0.9", SupportsJSON: true, SupportsWatch: true},
		Status:         sampleStatus(),
		Session:        session.Snapshot{RXBytes: 100, TXBytes: 20, Duration: time.Minute, PeakRX: 4, PeakTX: 1, AvgRX: 0.1, AvgTX: 0.02},
		History:        history.Summary{Samples: 12, CPUMax: 80, MemoryMax: 70, RXMax: 5, TXMax: 1},
		LastError:      "timeout",
		GeneratedAt:    time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC),
	}
}

func TestRenderContainsExpectedValues(t *testing.T) {
	got := Render(sampleInput())
	want := []string{
		"molebar_version: 1.2.3",
		"mole_version: mo 0.9",
		"os: darwin 25.6.0",
		"arch: arm64",
		"strategy: watch",
		"profile: developer",
		"refresh_interval: 5s",
		"supports_watch: true",
		"last_error_category: timeout",
		"health: 92 (Excellent)",
		"cpu_percent: 31.0",
		"memory_percent: 62.0",
		"disk_percent: 71.0",
		"battery_percent: 83.0",
		"temperature_c: 58.0",
		"peak_rx_mbs: 4.000",
		"samples: 12",
		"Cursor cpu=40.0",
	}
	for _, w := range want {
		if !strings.Contains(got, w) {
			t.Fatalf("missing %q in\n%s", w, got)
		}
	}
}

func TestRenderDeterministic(t *testing.T) {
	in := sampleInput()
	if Render(in) != Render(in) {
		t.Fatal("Render is not deterministic")
	}
}

func TestRenderDoesNotLeakSensitiveFields(t *testing.T) {
	in := sampleInput()
	got := Render(in)
	forbidden := []string{
		"10.0.0.8",
		"PATH=",
		"HOME=",
		"SSH",
		"password",
		"token",
		"command",
		"/Users/",
	}
	for _, f := range forbidden {
		if strings.Contains(got, f) {
			t.Fatalf("leaked %q in\n%s", f, got)
		}
	}
}

func TestSummaryOmitsMissingOptionalValues(t *testing.T) {
	in := sampleInput()
	in.Status = &molestatus.Status{}
	got := Summary(in)
	if strings.Contains(got, "Health") {
		t.Fatalf("fake health: %s", got)
	}
	if !strings.Contains(got, "CPU 0.0%") {
		t.Fatalf("summary = %s", got)
	}
}

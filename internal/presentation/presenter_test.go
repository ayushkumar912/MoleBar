package presentation

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/ayush-kumar912/molebar/internal/alerts"
	"github.com/ayush-kumar912/molebar/internal/config"
	"github.com/ayush-kumar912/molebar/internal/molestatus"
	"github.com/ayush-kumar912/molebar/internal/session"
)

func intPtr(v int) *int { return &v }

func sampleStatus() *molestatus.Status {
	s := &molestatus.Status{
		HealthScore: intPtr(100),
		HealthMsg:   "Excellent",
	}
	s.CPU.Usage = 12.2
	s.CPU.Load1 = 2.94
	s.Memory.UsedPercent = 67
	s.Memory.SwapUsed = 25
	s.Memory.SwapTotal = 100
	s.Disks = []molestatus.Disk{{Mount: "/", UsedPercent: 39.5}}
	s.Batteries = []molestatus.Battery{{Percent: 80, Status: "AC"}}
	s.Network = []molestatus.Network{{Name: "en0", RxRateMBs: 1.2, TxRateMBs: 0.33203125}}
	s.Thermal.CPUTemp = 58
	s.TopProcesses = []molestatus.ProcessInfo{
		{PID: 1, Name: "Cursor", CPU: 68},
		{PID: 2, Name: "Chrome", CPU: 41},
		{PID: 3, Name: "docker", CPU: 27},
	}
	return s
}

func TestPresentSys(t *testing.T) {
	vm := Present(State{Mode: config.DisplayModeSys, Status: sampleStatus(), Updated: time.Date(2026, 1, 1, 14, 32, 7, 0, time.UTC)})
	if vm.Title != "CPU 12% | RAM 67%" {
		t.Fatalf("title = %q", vm.Title)
	}
	if !vm.ModeSys || vm.ModeNet || vm.ModeBoth {
		t.Fatalf("checks sys=%v net=%v both=%v", vm.ModeSys, vm.ModeNet, vm.ModeBoth)
	}
	if vm.DisplayLabel != "Display: System" {
		t.Fatalf("label = %q", vm.DisplayLabel)
	}
}

func TestPresentNet(t *testing.T) {
	vm := Present(State{Mode: config.DisplayModeNet, Status: sampleStatus()})
	if !strings.HasPrefix(vm.Title, "↓") || !strings.Contains(vm.Title, "↑") {
		t.Fatalf("title = %q", vm.Title)
	}
	if !vm.ModeNet {
		t.Fatal("net check")
	}
}

func TestPresentBoth(t *testing.T) {
	vm := Present(State{Mode: config.DisplayModeBoth, Status: sampleStatus()})
	if !strings.Contains(vm.Title, "CPU") || !strings.Contains(vm.Title, "↓") {
		t.Fatalf("title = %q", vm.Title)
	}
	if !vm.ModeBoth {
		t.Fatal("both check")
	}
}

func TestPresentEachProfile(t *testing.T) {
	s := sampleStatus()
	tests := []struct {
		id   string
		want []string
	}{
		{"minimal", []string{"CPU 12%"}},
		{"developer", []string{"CPU 12%", "RAM 67%", "↓", "↑"}},
		{"network", []string{"↓", "↑"}},
		{"battery", []string{"80%", "58°C", "CPU 12%"}},
		{"full", []string{"100", "CPU 12%", "RAM 67%", "↓", "↑"}},
	}
	for _, tt := range tests {
		layout, ok := config.ResolveProfileLayout(tt.id)
		if !ok {
			t.Fatalf("profile %s", tt.id)
		}
		vm := Present(State{Layout: layout, Profile: tt.id, Status: s})
		for _, w := range tt.want {
			if !strings.Contains(vm.Title, w) {
				t.Fatalf("%s title %q missing %q", tt.id, vm.Title, w)
			}
		}
		found := false
		for _, row := range vm.ProfileRows {
			if row.ID == tt.id && row.Checked {
				found = true
			}
		}
		if !found {
			t.Fatalf("%s not checked", tt.id)
		}
	}
}

func TestPresentEachMetric(t *testing.T) {
	s := sampleStatus()
	cases := map[config.Metric]string{
		config.MetricCPU:         "CPU 12%",
		config.MetricMemory:      "RAM 67%",
		config.MetricRX:          "↓",
		config.MetricTX:          "↑",
		config.MetricBattery:     "80%",
		config.MetricTemperature: "58°C",
		config.MetricHealth:      "100",
		config.MetricDisk:        "Disk 40%",
	}
	for m, want := range cases {
		vm := Present(State{Layout: config.TrayLayout{Metrics: []config.Metric{m}}, Status: s})
		if !strings.Contains(vm.Title, want) {
			t.Fatalf("%s title %q missing %q", m, vm.Title, want)
		}
	}
}

func TestPresentMissingOptionalValues(t *testing.T) {
	s := sampleStatus()
	s.Batteries = nil
	s.Disks = nil
	s.Thermal = molestatus.Thermal{}
	s.HealthScore = nil
	s.HealthMsg = ""
	vm := Present(State{
		Layout: config.TrayLayout{Metrics: []config.Metric{config.MetricHealth, config.MetricBattery, config.MetricTemperature, config.MetricCPU}},
		Status: s,
	})
	if vm.Battery != "Battery: n/a" {
		t.Fatalf("battery = %q", vm.Battery)
	}
	if vm.Disk != "Disk: n/a" {
		t.Fatalf("disk = %q", vm.Disk)
	}
	if vm.Temperature != "Temperature: n/a" {
		t.Fatalf("temp = %q", vm.Temperature)
	}
	if vm.Health != "Health: n/a" {
		t.Fatalf("health = %q", vm.Health)
	}
	if vm.Title != "CPU 12%" {
		t.Fatalf("title should skip missing optionals, got %q", vm.Title)
	}
}

func TestPresentHealthUnavailable(t *testing.T) {
	s := sampleStatus()
	s.HealthScore = nil
	vm := Present(State{Layout: config.TrayLayout{Metrics: []config.Metric{config.MetricHealth, config.MetricCPU}}, Status: s})
	if strings.Contains(vm.Title, "100") || vm.Title != "CPU 12%" {
		t.Fatalf("title = %q", vm.Title)
	}
}

func TestPresentZeroTraffic(t *testing.T) {
	s := sampleStatus()
	s.Network = nil
	vm := Present(State{Mode: config.DisplayModeNet, Status: s})
	if vm.Down != "↓ 0 KB/s" || vm.Up != "↑ 0 KB/s" {
		t.Fatalf("down=%q up=%q", vm.Down, vm.Up)
	}
	if vm.Session != "Session: ↓0 B ↑0 B" {
		t.Fatalf("session = %q", vm.Session)
	}
}

func TestPresentPopulatedSessionTotals(t *testing.T) {
	vm := Present(State{
		Mode:   config.DisplayModeSys,
		Status: sampleStatus(),
		Session: session.Snapshot{
			RXBytes: 842.1 * (1 << 20),
			TXBytes: 112.4 * (1 << 20),
			PeakRX:  42,
			PeakTX:  8,
		},
	})
	if !strings.Contains(vm.Session, "MB") {
		t.Fatalf("session = %q", vm.Session)
	}
	if !strings.Contains(vm.PeakRX, "MB/s") {
		t.Fatalf("peak = %q", vm.PeakRX)
	}
}

func TestPresentFractionalBattery(t *testing.T) {
	s := sampleStatus()
	s.Batteries[0].Percent = 87.6
	vm := Present(State{Mode: config.DisplayModeSys, Status: s})
	if vm.Battery != "Battery: 87.6% (AC)" {
		t.Fatalf("battery = %q", vm.Battery)
	}
}

func TestPresentProcessRows(t *testing.T) {
	vm := Present(State{Mode: config.DisplayModeSys, Status: sampleStatus()})
	if len(vm.ProcessRows) != 3 {
		t.Fatalf("rows = %#v", vm.ProcessRows)
	}
	if !strings.Contains(vm.ProcessRows[0].Title, "Cursor") {
		t.Fatalf("first = %q", vm.ProcessRows[0].Title)
	}
}

func TestPresentActiveAlerts(t *testing.T) {
	vm := Present(State{
		Mode: config.DisplayModeSys,
		Alerts: []alerts.Alert{{
			Rule:  alerts.Rule{Metric: alerts.MetricCPU, Operator: alerts.OpGT, Value: 90},
			State: alerts.StateFiring,
			Value: 95,
		}},
		AlertsEnabled: true,
	})
	if len(vm.AlertRows) != 1 || !strings.Contains(vm.AlertRows[0].Title, "cpu") {
		t.Fatalf("alerts = %#v", vm.AlertRows)
	}
	if !vm.AlertsEnabled {
		t.Fatal("alerts enabled")
	}
}

func TestPresentErrorState(t *testing.T) {
	vm := Present(State{
		Mode:   config.DisplayModeSys,
		Status: sampleStatus(),
		Err:    errors.New("boom"),
	})
	if vm.Title != "mo: err" {
		t.Fatalf("title = %q", vm.Title)
	}
	if vm.CPU == "CPU: —" {
		t.Fatal("error should keep last-good menu text")
	}
}

func TestPresentErrorWithoutStatus(t *testing.T) {
	vm := Present(State{Mode: config.DisplayModeSys, Err: errors.New("boom")})
	if vm.Title != "mo: err" {
		t.Fatalf("title = %q", vm.Title)
	}
	if vm.CPU != "CPU: —" {
		t.Fatalf("cpu = %q", vm.CPU)
	}
}

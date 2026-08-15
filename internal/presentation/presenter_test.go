package presentation

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/ayush-kumar912/molebar/internal/config"
	"github.com/ayush-kumar912/molebar/internal/molestatus"
)

func sampleStatus() *molestatus.Status {
	s := &molestatus.Status{
		HealthScore: 100,
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
	return s
}

func TestPresentSys(t *testing.T) {
	vm := Present(State{Mode: config.DisplayModeSys, Status: sampleStatus(), Updated: time.Date(2026, 1, 1, 14, 32, 7, 0, time.UTC)})
	if vm.Title != "CPU 12% MEM 67%" {
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

func TestPresentMissingBatteryAndDisk(t *testing.T) {
	s := sampleStatus()
	s.Batteries = nil
	s.Disks = nil
	vm := Present(State{Mode: config.DisplayModeSys, Status: s})
	if vm.Battery != "Battery: n/a" {
		t.Fatalf("battery = %q", vm.Battery)
	}
	if vm.Disk != "Disk: n/a" {
		t.Fatalf("disk = %q", vm.Disk)
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
		Mode:      config.DisplayModeSys,
		Status:    sampleStatus(),
		SessionRx: 842.1 * (1 << 20),
		SessionTx: 112.4 * (1 << 20),
	})
	if !strings.Contains(vm.Session, "MB") {
		t.Fatalf("session = %q", vm.Session)
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

func TestPresentFractionalBattery(t *testing.T) {
	s := sampleStatus()
	s.Batteries[0].Percent = 87.6
	vm := Present(State{Mode: config.DisplayModeSys, Status: s})
	if vm.Battery != "Battery: 87.6% (AC)" {
		t.Fatalf("battery = %q", vm.Battery)
	}
}

// Package presentation formats application state into a tray ViewModel.
// Functions here are pure: no systray, processes, files, or mutation.
package presentation

import (
	"fmt"
	"math"
	"time"

	"github.com/ayush-kumar912/molebar/internal/config"
	"github.com/ayush-kumar912/molebar/internal/molestatus"
)

// ViewModel is everything the tray needs to paint one frame.
type ViewModel struct {
	Title   string
	Tooltip string

	CPU     string
	Memory  string
	Swap    string
	Disk    string
	Battery string

	Down    string
	Up      string
	Session string

	Health  string
	Updated string

	DisplayLabel string
	ModeSys      bool
	ModeNet      bool
	ModeBoth     bool
}

// State is the snapshot the presenter formats. All fields are inputs.
type State struct {
	Mode      config.DisplayMode
	Status    *molestatus.Status
	SessionRx float64
	SessionTx float64
	Updated   time.Time
	Err       error
}

// Present builds a ViewModel from domain state.
func Present(in State) ViewModel {
	mode := config.NormalizeDisplayMode(string(in.Mode))
	vm := ViewModel{
		Title:        "mo …",
		Tooltip:      "Mole system status",
		CPU:          "CPU: —",
		Memory:       "Memory: —",
		Swap:         "Swap: —",
		Disk:         "Disk: —",
		Battery:      "Battery: —",
		Down:         "↓ —",
		Up:           "↑ —",
		Session:      sessionLine(in.SessionRx, in.SessionTx),
		Health:       "Health: —",
		Updated:      "Updated: —",
		DisplayLabel: "Display: " + mode.Label(),
		ModeSys:      mode == config.DisplayModeSys,
		ModeNet:      mode == config.DisplayModeNet,
		ModeBoth:     mode == config.DisplayModeBoth,
	}

	if !in.Updated.IsZero() {
		vm.Updated = "Updated: " + in.Updated.Format("15:04:05")
	}

	if in.Status != nil {
		fillStatus(&vm, mode, in.Status, in.SessionRx, in.SessionTx)
	}

	if in.Err != nil {
		vm.Title = "mo: err"
	}
	return vm
}

func fillStatus(vm *ViewModel, mode config.DisplayMode, s *molestatus.Status, sessionRx, sessionTx float64) {
	rx, tx := s.TotalNetRates()
	switch mode {
	case config.DisplayModeNet:
		vm.Title = fmt.Sprintf("↓%s ↑%s", FormatRate(rx), FormatRate(tx))
	case config.DisplayModeBoth:
		vm.Title = fmt.Sprintf("CPU %.0f%% MEM %.0f%%  ↓%s ↑%s",
			s.CPU.Usage, s.Memory.UsedPercent, FormatRate(rx), FormatRate(tx))
	default:
		vm.Title = fmt.Sprintf("CPU %.0f%% MEM %.0f%%", s.CPU.Usage, s.Memory.UsedPercent)
	}
	vm.Tooltip = fmt.Sprintf("Health %d (%s)", s.HealthScore, s.HealthMsg)

	vm.CPU = fmt.Sprintf("CPU: %.1f%%  (load1 %.2f)", s.CPU.Usage, s.CPU.Load1)
	vm.Memory = fmt.Sprintf("Memory: %.1f%%", s.Memory.UsedPercent)
	vm.Swap = fmt.Sprintf("Swap: %.1f%%", s.SwapPercent())

	if pct := s.PrimaryDiskPercent(); pct >= 0 {
		vm.Disk = fmt.Sprintf("Disk: %.1f%%", pct)
	} else {
		vm.Disk = "Disk: n/a"
	}

	if pct, status, ok := s.PrimaryBattery(); ok {
		vm.Battery = fmt.Sprintf("Battery: %s (%s)", formatBatteryPercent(pct), status)
	} else {
		vm.Battery = "Battery: n/a"
	}

	vm.Down = "↓ " + FormatRate(rx)
	vm.Up = "↑ " + FormatRate(tx)
	vm.Session = sessionLine(sessionRx, sessionTx)
	vm.Health = fmt.Sprintf("Health: %d (%s)", s.HealthScore, s.HealthMsg)
}

func sessionLine(rx, tx float64) string {
	return fmt.Sprintf("Session: ↓%s ↑%s", FormatBytes(rx), FormatBytes(tx))
}

func formatBatteryPercent(pct float64) string {
	if math.IsNaN(pct) || math.IsInf(pct, 0) {
		return "n/a"
	}
	if pct == math.Trunc(pct) {
		return fmt.Sprintf("%.0f%%", pct)
	}
	return fmt.Sprintf("%.1f%%", pct)
}

// FormatRate renders a MB/s value the way most bandwidth-monitor menu bar
// apps do: sub-1-MB/s rates in KB/s for readability, everything else in
// MB/s with one decimal place.
func FormatRate(mbs float64) string {
	if mbs < 1 {
		return fmt.Sprintf("%.0f KB/s", mbs*1024)
	}
	return fmt.Sprintf("%.1f MB/s", mbs)
}

// FormatBytes renders a byte count as B/KB/MB/GB.
func FormatBytes(bytes float64) string {
	if bytes < 0 {
		bytes = 0
	}
	switch {
	case bytes >= 1<<30:
		return fmt.Sprintf("%.2f GB", bytes/(1<<30))
	case bytes >= 1<<20:
		return fmt.Sprintf("%.1f MB", bytes/(1<<20))
	case bytes >= 1<<10:
		return fmt.Sprintf("%.0f KB", bytes/(1<<10))
	default:
		return fmt.Sprintf("%.0f B", bytes)
	}
}

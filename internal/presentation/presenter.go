package presentation

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/ayush-kumar912/molebar/internal/alerts"
	"github.com/ayush-kumar912/molebar/internal/config"
	"github.com/ayush-kumar912/molebar/internal/diagnostics"
	"github.com/ayush-kumar912/molebar/internal/molestatus"
	"github.com/ayush-kumar912/molebar/internal/session"
)

// State is the snapshot the presenter formats. All fields are inputs.
type State struct {
	Layout                 config.TrayLayout
	Mode                   config.DisplayMode
	Profile                string
	Status                 *molestatus.Status
	Session                session.Snapshot
	Alerts                 []alerts.Alert
	AlertsEnabled          bool
	LaunchAtLogin          bool
	LaunchAtLoginSupported bool
	Updated                time.Time
	Err                    error
	Diag                   diagnostics.Input
}

// Present builds a ViewModel from domain state.
func Present(in State) ViewModel {
	layout := resolveLayout(in)
	mode := layout.DisplayMode()
	if len(in.Layout.Metrics) == 0 && in.Mode != "" {
		mode = config.NormalizeDisplayMode(string(in.Mode))
	}
	vm := ViewModel{
		Title:                  "mo …",
		Tooltip:                "Mole system status",
		CPU:                    "CPU: —",
		Memory:                 "Memory: —",
		Swap:                   "Swap: —",
		Disk:                   "Disk: —",
		Battery:                "Battery: —",
		Temperature:            "Temperature: —",
		Health:                 "Health: —",
		Down:                   "↓ —",
		Up:                     "↑ —",
		Session:                sessionLine(in.Session.RXBytes, in.Session.TXBytes),
		SessionRX:              "Session RX: " + FormatBytes(in.Session.RXBytes),
		SessionTX:              "Session TX: " + FormatBytes(in.Session.TXBytes),
		PeakRX:                 "Peak RX: " + FormatRate(in.Session.PeakRX),
		PeakTX:                 "Peak TX: " + FormatRate(in.Session.PeakTX),
		AvgRX:                  "Avg RX: " + FormatRate(in.Session.AvgRX),
		AvgTX:                  "Avg TX: " + FormatRate(in.Session.AvgTX),
		SessionDuration:        "Duration: " + formatDuration(in.Session.Duration),
		Updated:                "Updated: —",
		DisplayLabel:           "Display: " + mode.Label(),
		ModeSys:                mode == config.DisplayModeSys,
		ModeNet:                mode == config.DisplayModeNet,
		ModeBoth:               mode == config.DisplayModeBoth,
		AlertsEnabled:          in.AlertsEnabled,
		LaunchAtLogin:          in.LaunchAtLogin,
		LaunchAtLoginSupported: in.LaunchAtLoginSupported,
		ProfileRows:            profileRows(in.Profile, layout),
		MetricRows:             metricRows(layout),
	}

	if !in.Updated.IsZero() {
		vm.Updated = "Updated: " + in.Updated.Format("15:04:05")
	}

	if in.Status != nil {
		fillStatus(&vm, layout, in.Status, in.Session)
	}

	vm.AlertRows = alertRows(in.Alerts)
	vm.SystemSummary = diagnostics.Summary(in.Diag)
	vm.Diagnostics = diagnostics.Render(in.Diag)

	if in.Err != nil {
		vm.Title = "mo: err"
	}
	return vm
}

func resolveLayout(in State) config.TrayLayout {
	if len(in.Layout.Metrics) > 0 {
		return config.NormalizeLayout(in.Layout)
	}
	if in.Mode != "" {
		return config.LayoutFromDisplayMode(in.Mode)
	}
	return config.LayoutFromDisplayMode(config.DefaultDisplayMode)
}

func fillStatus(vm *ViewModel, layout config.TrayLayout, s *molestatus.Status, sess session.Snapshot) {
	rx, tx := s.TotalNetRates()
	vm.Title = formatTitle(layout, s)
	if score, msg, ok := s.Health(); ok {
		vm.Tooltip = fmt.Sprintf("Health %d (%s)", score, msg)
	} else {
		vm.Tooltip = "Mole system status"
	}

	vm.CPU = fmt.Sprintf("CPU: %.1f%%  (load1 %.2f)", s.CPU.Usage, s.CPU.Load1)
	vm.Memory = fmt.Sprintf("Memory: %.1f%%", s.Memory.UsedPercent)
	vm.Swap = fmt.Sprintf("Swap: %.1f%%", s.SwapPercent())

	if pct := s.PrimaryDiskPercent(); pct >= 0 {
		vm.Disk = fmt.Sprintf("Disk: %.1f%%", pct)
	} else {
		vm.Disk = "Disk: n/a"
	}

	if info, ok := s.PrimaryBatteryInfo(); ok {
		vm.Battery = fmt.Sprintf("Battery: %s (%s)", formatBatteryPercent(info.Percent), info.Status)
	} else {
		vm.Battery = "Battery: n/a"
	}

	if temp, ok := s.CPUTemperature(); ok {
		vm.Temperature = fmt.Sprintf("Temperature: %.0f°C", temp)
	} else {
		vm.Temperature = "Temperature: n/a"
	}

	if score, msg, ok := s.Health(); ok {
		vm.Health = fmt.Sprintf("Health: %d (%s)", score, msg)
	} else {
		vm.Health = "Health: n/a"
	}

	vm.Down = "↓ " + FormatRate(rx)
	vm.Up = "↑ " + FormatRate(tx)
	vm.Session = sessionLine(sess.RXBytes, sess.TXBytes)
	vm.ProcessRows = processRows(s)
}

func formatTitle(layout config.TrayLayout, s *molestatus.Status) string {
	parts := make([]string, 0, len(layout.Metrics))
	for _, m := range layout.Metrics {
		if part, ok := formatMetric(m, s); ok {
			parts = append(parts, part)
		}
	}
	if len(parts) == 0 {
		return "mo …"
	}
	sep := layout.Separator
	if sep == "" {
		sep = " | "
	}
	return strings.Join(parts, sep)
}

func formatMetric(m config.Metric, s *molestatus.Status) (string, bool) {
	switch m {
	case config.MetricCPU:
		return fmt.Sprintf("CPU %.0f%%", s.CPU.Usage), true
	case config.MetricMemory:
		return fmt.Sprintf("RAM %.0f%%", s.Memory.UsedPercent), true
	case config.MetricRX:
		rx, _ := s.TotalNetRates()
		return "↓" + FormatRate(rx), true
	case config.MetricTX:
		_, tx := s.TotalNetRates()
		return "↑" + FormatRate(tx), true
	case config.MetricBattery:
		pct, _, ok := s.PrimaryBattery()
		if !ok {
			return "", false
		}
		return formatBatteryPercent(pct), true
	case config.MetricTemperature:
		temp, ok := s.CPUTemperature()
		if !ok {
			return "", false
		}
		return fmt.Sprintf("%.0f°C", temp), true
	case config.MetricHealth:
		score, _, ok := s.Health()
		if !ok {
			return "", false
		}
		return fmt.Sprintf("%d", score), true
	case config.MetricDisk:
		pct := s.PrimaryDiskPercent()
		if pct < 0 {
			return "", false
		}
		return fmt.Sprintf("Disk %.0f%%", pct), true
	default:
		return "", false
	}
}

func processRows(s *molestatus.Status) []ProcessRow {
	procs := s.TopCPUProcesses(5)
	if len(procs) == 0 {
		return nil
	}
	out := make([]ProcessRow, 0, len(procs))
	for _, p := range procs {
		name := p.Name
		if name == "" {
			name = fmt.Sprintf("pid %d", p.PID)
		}
		out = append(out, ProcessRow{Title: fmt.Sprintf("%s  %.0f%%", name, p.CPUPercent)})
	}
	return out
}

func alertRows(items []alerts.Alert) []AlertRow {
	if len(items) == 0 {
		return nil
	}
	out := make([]AlertRow, 0, len(items))
	for _, a := range items {
		out = append(out, AlertRow{
			Title: fmt.Sprintf("%s %s %.0f (%.1f)", a.Rule.Metric, a.Rule.Operator, a.Rule.Value, a.Value),
		})
	}
	return out
}

func profileRows(selected string, layout config.TrayLayout) []ProfileRow {
	if selected == "" {
		selected = string(config.MatchingProfile(layout))
		if selected == string(config.ProfileCustom) {
			selected = ""
		}
	}
	var rows []ProfileRow
	for _, p := range config.BuiltInProfiles() {
		rows = append(rows, ProfileRow{
			ID:      string(p.ID),
			Label:   p.Label,
			Checked: selected == string(p.ID),
		})
	}
	return rows
}

func metricRows(layout config.TrayLayout) []MetricRow {
	var rows []MetricRow
	for _, m := range config.AllMetrics() {
		rows = append(rows, MetricRow{
			ID:      string(m),
			Label:   m.Label(),
			Checked: layout.Contains(m),
		})
	}
	return rows
}

func sessionLine(rx, tx float64) string {
	return fmt.Sprintf("Session: ↓%s ↑%s", FormatBytes(rx), FormatBytes(tx))
}

func formatDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	d = d.Round(time.Second)
	return d.String()
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

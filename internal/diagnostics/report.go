package diagnostics

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/ayush-kumar912/molebar/internal/history"
	"github.com/ayush-kumar912/molebar/internal/molestatus"
	"github.com/ayush-kumar912/molebar/internal/session"
)

// Input is the explicit data the report generator may use.
// Nothing is read from the filesystem or environment here.
type Input struct {
	MoleBarVersion string
	MoleVersion    string
	OSName         string
	OSVersion      string
	Arch           string
	Strategy       string
	Profile        string
	Interval       time.Duration
	Capabilities   molestatus.Capabilities
	Status         *molestatus.Status
	Session        session.Snapshot
	History        history.Summary
	LastError      string
	GeneratedAt    time.Time
}

// Render builds a full diagnostics report. It is deterministic for a
// given Input and never includes environment variables, tokens, IPs,
// command lines, or home-directory contents.
func Render(in Input) string {
	var b strings.Builder
	writeHeader(&b, in)
	writeCollector(&b, in)
	writeStatus(&b, in)
	writeSession(&b, in)
	writeHistory(&b, in)
	writeProcesses(&b, in)
	return b.String()
}

// Summary is a short clipboard-friendly system summary.
func Summary(in Input) string {
	var b strings.Builder
	fmt.Fprintf(&b, "MoleBar %s\n", dash(in.MoleBarVersion))
	if in.Status != nil {
		if score, msg, ok := in.Status.Health(); ok {
			fmt.Fprintf(&b, "Health %d (%s)\n", score, msg)
		}
		fmt.Fprintf(&b, "CPU %.1f%%  Memory %.1f%%\n", in.Status.CPU.Usage, in.Status.Memory.UsedPercent)
		if pct := in.Status.PrimaryDiskPercent(); pct >= 0 {
			fmt.Fprintf(&b, "Disk %.1f%%\n", pct)
		}
		if pct, status, ok := in.Status.PrimaryBattery(); ok {
			fmt.Fprintf(&b, "Battery %s (%s)\n", formatPct(pct), status)
		}
		rx, tx := in.Status.TotalNetRates()
		fmt.Fprintf(&b, "Network ↓%.3f MB/s ↑%.3f MB/s\n", rx, tx)
	} else {
		b.WriteString("No status sample\n")
	}
	if in.LastError != "" {
		fmt.Fprintf(&b, "Last error: %s\n", in.LastError)
	}
	return b.String()
}

func writeHeader(b *strings.Builder, in Input) {
	b.WriteString("MoleBar diagnostics\n")
	fmt.Fprintf(b, "generated_at: %s\n", formatTime(in.GeneratedAt))
	fmt.Fprintf(b, "molebar_version: %s\n", dash(in.MoleBarVersion))
	fmt.Fprintf(b, "mole_version: %s\n", dash(in.MoleVersion))
	osName := in.OSName
	if osName == "" {
		osName = runtime.GOOS
	}
	arch := in.Arch
	if arch == "" {
		arch = runtime.GOARCH
	}
	fmt.Fprintf(b, "os: %s %s\n", osName, dash(in.OSVersion))
	fmt.Fprintf(b, "arch: %s\n", arch)
}

func writeCollector(b *strings.Builder, in Input) {
	b.WriteString("\n[collector]\n")
	fmt.Fprintf(b, "strategy: %s\n", dash(in.Strategy))
	fmt.Fprintf(b, "profile: %s\n", dash(in.Profile))
	fmt.Fprintf(b, "refresh_interval: %s\n", in.Interval)
	fmt.Fprintf(b, "supports_json: %t\n", in.Capabilities.SupportsJSON)
	fmt.Fprintf(b, "supports_watch: %t\n", in.Capabilities.SupportsWatch)
	fmt.Fprintf(b, "last_error_category: %s\n", dash(in.LastError))
}

func writeStatus(b *strings.Builder, in Input) {
	b.WriteString("\n[status]\n")
	if in.Status == nil {
		b.WriteString("available: false\n")
		return
	}
	s := in.Status
	if score, msg, ok := s.Health(); ok {
		fmt.Fprintf(b, "health: %d (%s)\n", score, msg)
	} else {
		b.WriteString("health: unavailable\n")
	}
	fmt.Fprintf(b, "cpu_percent: %.1f\n", s.CPU.Usage)
	fmt.Fprintf(b, "memory_percent: %.1f\n", s.Memory.UsedPercent)
	if pct := s.PrimaryDiskPercent(); pct >= 0 {
		fmt.Fprintf(b, "disk_percent: %.1f\n", pct)
	} else {
		b.WriteString("disk_percent: unavailable\n")
	}
	if info, ok := s.PrimaryBatteryInfo(); ok {
		fmt.Fprintf(b, "battery_percent: %s\n", formatPct(info.Percent))
		fmt.Fprintf(b, "battery_status: %s\n", dash(info.Status))
		if info.Health != nil {
			fmt.Fprintf(b, "battery_health: %s\n", *info.Health)
		}
		if info.CycleCount != nil {
			fmt.Fprintf(b, "battery_cycles: %d\n", *info.CycleCount)
		}
	} else {
		b.WriteString("battery: unavailable\n")
	}
	if temp, ok := s.CPUTemperature(); ok {
		fmt.Fprintf(b, "temperature_c: %.1f\n", temp)
	} else {
		b.WriteString("temperature_c: unavailable\n")
	}
	rx, tx := s.TotalNetRates()
	fmt.Fprintf(b, "rx_mbs: %.3f\n", rx)
	fmt.Fprintf(b, "tx_mbs: %.3f\n", tx)
}

func writeSession(b *strings.Builder, in Input) {
	b.WriteString("\n[session]\n")
	fmt.Fprintf(b, "duration: %s\n", in.Session.Duration)
	fmt.Fprintf(b, "rx_bytes: %.0f\n", in.Session.RXBytes)
	fmt.Fprintf(b, "tx_bytes: %.0f\n", in.Session.TXBytes)
	fmt.Fprintf(b, "peak_rx_mbs: %.3f\n", in.Session.PeakRX)
	fmt.Fprintf(b, "peak_tx_mbs: %.3f\n", in.Session.PeakTX)
	fmt.Fprintf(b, "avg_rx_mbs: %.3f\n", in.Session.AvgRX)
	fmt.Fprintf(b, "avg_tx_mbs: %.3f\n", in.Session.AvgTX)
}

func writeHistory(b *strings.Builder, in Input) {
	b.WriteString("\n[history]\n")
	fmt.Fprintf(b, "samples: %d\n", in.History.Samples)
	fmt.Fprintf(b, "cpu_max: %.1f\n", in.History.CPUMax)
	fmt.Fprintf(b, "memory_max: %.1f\n", in.History.MemoryMax)
	fmt.Fprintf(b, "rx_max_mbs: %.3f\n", in.History.RXMax)
	fmt.Fprintf(b, "tx_max_mbs: %.3f\n", in.History.TXMax)
}

func writeProcesses(b *strings.Builder, in Input) {
	b.WriteString("\n[top_processes]\n")
	if in.Status == nil {
		b.WriteString("available: false\n")
		return
	}
	procs := in.Status.TopCPUProcesses(5)
	if len(procs) == 0 {
		b.WriteString("available: false\n")
		return
	}
	for _, p := range procs {
		fmt.Fprintf(b, "%s cpu=%.1f\n", sanitizeName(p.Name), p.CPUPercent)
	}
}

func sanitizeName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, "\n", " ")
	if name == "" {
		return "unknown"
	}
	return name
}

func dash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return s
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.UTC().Format(time.RFC3339)
}

func formatPct(pct float64) string {
	return fmt.Sprintf("%.1f", pct)
}

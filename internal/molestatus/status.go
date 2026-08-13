// Package molestatus fetches and parses the JSON output of `mo status --json`
// (the Mole CLI: https://github.com/tw93/Mole) into typed Go structs.
package molestatus

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// Status is a partial mirror of `mo status --json`. Only the fields the
// widget actually displays are modeled; unknown fields are ignored by
// encoding/json automatically, so this stays forward-compatible with new
// keys Mole adds later.
type Status struct {
	CollectedAt string `json:"collected_at"`
	HealthScore int    `json:"health_score"`
	HealthMsg   string `json:"health_score_msg"`

	CPU struct {
		Usage float64   `json:"usage"`
		Load1 float64   `json:"load1"`
		Cores []float64 `json:"per_core"`
	} `json:"cpu"`

	Memory struct {
		UsedPercent float64 `json:"used_percent"`
		Total       int64   `json:"total"`
		Used        int64   `json:"used"`
		Available   int64   `json:"available"`
		SwapUsed    int64   `json:"swap_used"`
		SwapTotal   int64   `json:"swap_total"`
	} `json:"memory"`

	Disks []struct {
		Mount       string  `json:"mount"`
		UsedPercent float64 `json:"used_percent"`
		Total       int64   `json:"total"`
		Used        int64   `json:"used"`
		Purgeable   int64   `json:"purgeable"`
	} `json:"disks"`

	Batteries []struct {
		Percent    int    `json:"percent"`
		Status     string `json:"status"`
		Health     string `json:"health"`
		CycleCount int    `json:"cycle_count"`
	} `json:"batteries"`

	// Network holds one entry per interface Mole reports (Wi-Fi, Ethernet,
	// VPN tunnels, etc). Inactive interfaces are still present with zero
	// rates, so callers should sum/filter rather than assume index 0 is
	// the active link.
	Network []struct {
		Name      string  `json:"name"`
		RxRateMBs float64 `json:"rx_rate_mbs"` // MB/s, matches mo's text UI ("R 0.1 · W 0 MB/s")
		TxRateMBs float64 `json:"tx_rate_mbs"`
		IP        string  `json:"ip"`
	} `json:"network"`

	Procs int `json:"procs"`
}

// SwapPercent returns used/total swap as a percentage, or 0 if no swap
// is configured (avoids a divide-by-zero on machines with swap disabled).
func (s *Status) SwapPercent() float64 {
	if s.Memory.SwapTotal == 0 {
		return 0
	}
	return float64(s.Memory.SwapUsed) / float64(s.Memory.SwapTotal) * 100
}

// PrimaryDiskPercent returns the used_percent of the first reported disk
// (typically "/"), or -1 if Mole reported no disks.
func (s *Status) PrimaryDiskPercent() float64 {
	if len(s.Disks) == 0 {
		return -1
	}
	return s.Disks[0].UsedPercent
}

// PrimaryBattery returns the first battery entry and true, or a zero value
// and false on desktop Macs / machines with no battery reported.
func (s *Status) PrimaryBattery() (percent int, status string, ok bool) {
	if len(s.Batteries) == 0 {
		return 0, "", false
	}
	return s.Batteries[0].Percent, s.Batteries[0].Status, true
}

// TotalNetRates sums rx/tx across every interface Mole reports, in MB/s.
// Summing (rather than picking one interface) is safe because inactive
// interfaces report 0 — but note it can double-count traffic on machines
// running a VPN tunnel on top of Wi-Fi/Ethernet, since the tunnel and the
// underlying physical interface both carry (roughly) the same bytes. If
// you only care about the physical link, filter s.Network by name instead.
func (s *Status) TotalNetRates() (rxMBs, txMBs float64) {
	for _, n := range s.Network {
		rxMBs += n.RxRateMBs
		txMBs += n.TxRateMBs
	}
	return rxMBs, txMBs
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

// FormatBytes renders a byte count as B/KB/MB/GB, matching common
// menu-bar-app conventions (used for session cumulative totals).
func FormatBytes(bytes float64) string {
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

// Fetcher runs `mo status --json` and parses the result. It shells out
// rather than linking Mole as a library because Mole ships as a CLI, not
// a Go package with a stable public API.
type Fetcher struct {
	// BinPath is the path to the `mo` executable. Defaults to "mo" (resolved
	// via $PATH) when empty.
	BinPath string
	// Timeout bounds how long a single `mo status --json` call may run
	// before it's killed. Defaults to 5s when zero.
	Timeout time.Duration
}

func (f *Fetcher) bin() string {
	if f.BinPath != "" {
		return f.BinPath
	}
	if p, err := exec.LookPath("mo"); err == nil {
		return p
	}
	// GUI apps launched from Finder/Applications don't inherit the shell's
	// $PATH, so "mo" often isn't resolvable there even though it works
	// fine from a terminal. Fall back to the standard Homebrew locations.
	for _, p := range []string{"/opt/homebrew/bin/mo", "/usr/local/bin/mo"} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return "mo" // let exec.Command fail with a clear "not found" error
}

func (f *Fetcher) timeout() time.Duration {
	if f.Timeout <= 0 {
		return 5 * time.Second
	}
	return f.Timeout
}

// Fetch runs `mo status --json` and returns the parsed status. Callers
// should treat a non-nil error as "skip this refresh cycle" rather than
// fatal — a transient failure (Mole mid-update, brief CPU spike) shouldn't
// crash a long-running menu-bar process.
func Fetch() (*Status, error) {
	f := &Fetcher{}
	return f.Fetch()
}

// Fetch is the instance-method form of Fetch, allowing a custom BinPath
// or Timeout (mainly useful for tests, or if `mo` isn't on $PATH — e.g.
// launchd-started processes often have a minimal PATH).
func (f *Fetcher) Fetch() (*Status, error) {
	cmd := exec.Command(f.bin(), "status", "--json")

	done := make(chan struct {
		out []byte
		err error
	}, 1)

	go func() {
		out, err := cmd.Output()
		done <- struct {
			out []byte
			err error
		}{out, err}
	}()

	select {
	case r := <-done:
		if r.err != nil {
			return nil, fmt.Errorf("run %q: %w", f.bin(), r.err)
		}
		var s Status
		if err := json.Unmarshal(r.out, &s); err != nil {
			return nil, fmt.Errorf("parse %q output: %w", f.bin(), err)
		}
		return &s, nil
	case <-time.After(f.timeout()):
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		return nil, fmt.Errorf("%q timed out after %s", f.bin(), f.timeout())
	}
}

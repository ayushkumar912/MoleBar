// Package molestatus fetches and parses Mole (`mo status`) JSON.
package molestatus

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// Parse decodes a single Mole status JSON document.
func Parse(data []byte) (*Status, error) {
	var s Status
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMalformedJSON, err)
	}
	return &s, nil
}

// SwapPercent returns used/total swap as a percentage, or 0 if no swap
// is configured (avoids a divide-by-zero on machines with swap disabled).
func (s *Status) SwapPercent() float64 {
	if s == nil || s.Memory.SwapTotal == 0 {
		return 0
	}
	return float64(s.Memory.SwapUsed) / float64(s.Memory.SwapTotal) * 100
}

// PrimaryDiskPercent returns used_percent of the volume mounted at "/".
// If Mole omitted the root filesystem, it returns -1 rather than guessing
// from array order (other mounts such as /Volumes/... may be listed first).
func (s *Status) PrimaryDiskPercent() float64 {
	if s == nil {
		return -1
	}
	for _, d := range s.Disks {
		if d.Mount == "/" {
			return d.UsedPercent
		}
	}
	return -1
}

// Health returns Mole's health score when the field was present in JSON.
// A missing key is unavailable; a present zero is a real score.
func (s *Status) Health() (score int, msg string, ok bool) {
	if s == nil || s.HealthScore == nil {
		return 0, "", false
	}
	return *s.HealthScore, s.HealthMsg, true
}

// PrimaryBattery returns the first battery entry and true, or a zero value
// and false on desktop Macs / machines with no battery reported.
// Percent is the upstream floating-point value; callers must not truncate it.
func (s *Status) PrimaryBattery() (percent float64, status string, ok bool) {
	info, ok := s.PrimaryBatteryInfo()
	return info.Percent, info.Status, ok
}

// PrimaryBatteryInfo returns optional battery fields without inventing values.
func (s *Status) PrimaryBatteryInfo() (BatteryInfo, bool) {
	if s == nil || len(s.Batteries) == 0 {
		return BatteryInfo{}, false
	}
	b := s.Batteries[0]
	info := BatteryInfo{
		Percent: b.Percent,
		Status:  b.Status,
	}
	if charging, ok := chargingFromStatus(b.Status); ok {
		info.Charging = &charging
	}
	if b.Health != "" {
		h := b.Health
		info.Health = &h
	}
	if b.CycleCount != 0 {
		c := b.CycleCount
		info.CycleCount = &c
	}
	if b.Capacity != 0 {
		c := b.Capacity
		info.Capacity = &c
	}
	return info, true
}

func chargingFromStatus(status string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "ac", "charging", "charged", "finishing charge":
		return true, true
	case "battery", "discharging", "running on battery":
		return false, true
	default:
		return false, false
	}
}

// CPUTemperature returns Mole's CPU temperature in Celsius when present.
// Non-positive values are treated as unavailable rather than invented.
func (s *Status) CPUTemperature() (celsius float64, ok bool) {
	if s == nil || s.Thermal.CPUTemp <= 0 {
		return 0, false
	}
	return s.Thermal.CPUTemp, true
}

// TotalNetRates sums rx/tx across the network records Mole actually supplied,
// in MB/s. Mole decides which interfaces appear; MoleBar does not assume it
// receives every physical or VPN interface.
func (s *Status) TotalNetRates() (rxMBs, txMBs float64) {
	if s == nil {
		return 0, 0
	}
	for _, n := range s.Network {
		rxMBs += n.RxRateMBs
		txMBs += n.TxRateMBs
	}
	return rxMBs, txMBs
}

const defaultTopN = 5

// TopCPUProcesses returns up to n processes sorted by CPU descending,
// then name, then PID. n <= 0 uses 5. Command lines are not included.
func (s *Status) TopCPUProcesses(n int) []ProcessStat {
	if s == nil || len(s.TopProcesses) == 0 {
		return nil
	}
	if n <= 0 {
		n = defaultTopN
	}
	out := make([]ProcessStat, 0, len(s.TopProcesses))
	for _, p := range s.TopProcesses {
		out = append(out, ProcessStat{
			PID:        p.PID,
			Name:       p.Name,
			CPUPercent: p.CPU,
			Memory:     p.MemoryBytes,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].CPUPercent != out[j].CPUPercent {
			return out[i].CPUPercent > out[j].CPUPercent
		}
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].PID < out[j].PID
	})
	if len(out) > n {
		out = out[:n]
	}
	return out
}

// MaxProcessCPU is the highest CPU percent among top processes, if any.
func (s *Status) MaxProcessCPU() (float64, bool) {
	procs := s.TopCPUProcesses(0)
	if len(procs) == 0 {
		return 0, false
	}
	return procs[0].CPUPercent, true
}

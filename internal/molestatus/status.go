// Package molestatus fetches and parses Mole (`mo status`) JSON.
package molestatus

import (
	"encoding/json"
	"fmt"
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

// PrimaryBattery returns the first battery entry and true, or a zero value
// and false on desktop Macs / machines with no battery reported.
// Percent is the upstream floating-point value; callers must not truncate it.
func (s *Status) PrimaryBattery() (percent float64, status string, ok bool) {
	if s == nil || len(s.Batteries) == 0 {
		return 0, "", false
	}
	return s.Batteries[0].Percent, s.Batteries[0].Status, true
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

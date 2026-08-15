// Package session accounts estimated network transfer while MoleBar is sampling.
package session

import "time"

const (
	// DefaultMaxGap is the largest interval that may be integrated. A longer
	// pause (sleep/wake, a hung collector) is treated as a discontinuity:
	// the arriving sample primes the meter and adds nothing.
	DefaultMaxGap = 60 * time.Second

	bytesPerMB = 1 << 20
)

// Meter integrates Mole's instantaneous MB/s rates into session byte totals.
//
// Integration rule (right Riemann: the newly observed rate × elapsed):
//
//   - the first valid sample only primes the meter (adds zero)
//   - a sample after Reset or Invalidate behaves like a first sample
//   - a sample after an elapsed interval <= 0 is ignored
//   - a sample after a gap >= MaxGap re-primes and adds zero
//   - totals are clamped so they never go negative
//
// Callers pass explicit timestamps so tests do not sleep.
type Meter struct {
	maxGap  time.Duration
	rx      float64
	tx      float64
	lastAt  time.Time
	hasPrev bool
}

// New returns a meter that refuses to integrate across gaps of maxGap or more.
// A non-positive maxGap uses DefaultMaxGap.
func New(maxGap time.Duration) *Meter {
	if maxGap <= 0 {
		maxGap = DefaultMaxGap
	}
	return &Meter{maxGap: maxGap}
}

// Observe records a successful sample at at with instantaneous rates in MB/s.
func (m *Meter) Observe(at time.Time, rxRateMBs, txRateMBs float64) {
	if m == nil {
		return
	}
	if !m.hasPrev {
		m.lastAt = at
		m.hasPrev = true
		return
	}
	elapsed := at.Sub(m.lastAt)
	if elapsed <= 0 {
		return
	}
	if elapsed >= m.maxGap {
		m.lastAt = at
		return
	}
	sec := elapsed.Seconds()
	m.rx = clampNonNegative(m.rx + rxRateMBs*bytesPerMB*sec)
	m.tx = clampNonNegative(m.tx + txRateMBs*bytesPerMB*sec)
	m.lastAt = at
}

// Invalidate breaks integration continuity without clearing totals.
// The next Observe primes only. Use this when a collection interval failed
// so a later instantaneous rate is not applied across the outage.
func (m *Meter) Invalidate() {
	if m == nil {
		return
	}
	m.hasPrev = false
	m.lastAt = time.Time{}
}

// Reset clears totals and sampling state. The next Observe primes only.
func (m *Meter) Reset() {
	if m == nil {
		return
	}
	m.rx = 0
	m.tx = 0
	m.Invalidate()
}

// Totals returns accumulated RX/TX bytes. Values are never negative.
func (m *Meter) Totals() (rxBytes, txBytes float64) {
	if m == nil {
		return 0, 0
	}
	return m.rx, m.tx
}

func clampNonNegative(v float64) float64 {
	if v < 0 {
		return 0
	}
	return v
}

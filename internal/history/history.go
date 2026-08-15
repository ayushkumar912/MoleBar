package history

import "time"

const (
	// DefaultWindow is the amount of recent history to retain.
	DefaultWindow = 10 * time.Minute
	maxCapacity   = 1024
	minCapacity   = 8
)

// Point is one recorded metric snapshot. Optional fields are nil when
// the upstream collector did not supply them.
type Point struct {
	CPU         float64
	Memory      float64
	RX          float64
	TX          float64
	Temperature *float64
	Health      *float64
}

// History is a bounded in-memory series of Points. It does not touch
// the UI, disk, or Mole.
type History struct {
	ring *Ring[Point]
}

// CapacityFor returns how many samples fit in window at the given
// refresh interval. The result is clamped to a sane fixed bound.
func CapacityFor(interval, window time.Duration) int {
	if interval <= 0 {
		interval = 5 * time.Second
	}
	if window <= 0 {
		window = DefaultWindow
	}
	n := int(window / interval)
	if n < minCapacity {
		return minCapacity
	}
	if n > maxCapacity {
		return maxCapacity
	}
	return n
}

// New constructs a history buffer with a fixed sample capacity.
func New(capacity int) *History {
	return &History{ring: NewRing[Point](capacity)}
}

// Add records a point at an explicit timestamp.
func (h *History) Add(at time.Time, p Point) {
	if h == nil || h.ring == nil {
		return
	}
	h.ring.Append(Sample[Point]{At: at, Value: p})
}

// Samples returns chronological copies of stored points.
func (h *History) Samples() []Sample[Point] {
	if h == nil || h.ring == nil {
		return nil
	}
	return h.ring.Slice()
}

// Len is the number of stored points.
func (h *History) Len() int {
	if h == nil || h.ring == nil {
		return 0
	}
	return h.ring.Len()
}

// Cap is the fixed capacity.
func (h *History) Cap() int {
	if h == nil || h.ring == nil {
		return 0
	}
	return h.ring.Cap()
}

// Summary is a compact view of the retained window for diagnostics.
type Summary struct {
	Samples   int
	CPUMax    float64
	MemoryMax float64
	RXMax     float64
	TXMax     float64
}

// Summarize computes maxima over the retained samples.
func (h *History) Summarize() Summary {
	var out Summary
	if h == nil {
		return out
	}
	samples := h.Samples()
	out.Samples = len(samples)
	for _, s := range samples {
		if s.Value.CPU > out.CPUMax {
			out.CPUMax = s.Value.CPU
		}
		if s.Value.Memory > out.MemoryMax {
			out.MemoryMax = s.Value.Memory
		}
		if s.Value.RX > out.RXMax {
			out.RXMax = s.Value.RX
		}
		if s.Value.TX > out.TXMax {
			out.TXMax = s.Value.TX
		}
	}
	return out
}

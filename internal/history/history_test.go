package history

import (
	"testing"
	"time"
)

func TestCapacityFor(t *testing.T) {
	if got := CapacityFor(5*time.Second, 10*time.Minute); got != 120 {
		t.Fatalf("5s/10m = %d", got)
	}
	if got := CapacityFor(0, 0); got < minCapacity {
		t.Fatalf("defaults = %d", got)
	}
	if got := CapacityFor(time.Hour, time.Second); got != minCapacity {
		t.Fatalf("tiny window = %d", got)
	}
	if got := CapacityFor(time.Millisecond, 24*time.Hour); got != maxCapacity {
		t.Fatalf("huge window = %d", got)
	}
}

func TestHistoryAddAndSummarize(t *testing.T) {
	h := New(4)
	if h.Len() != 0 || h.Cap() != 4 {
		t.Fatalf("len=%d cap=%d", h.Len(), h.Cap())
	}
	temp := 58.0
	health := 90.0
	h.Add(ts(1), Point{CPU: 10, Memory: 20, RX: 1, TX: 0.5, Temperature: &temp, Health: &health})
	h.Add(ts(2), Point{CPU: 40, Memory: 15, RX: 3, TX: 0.2})
	if h.Len() != 2 {
		t.Fatalf("len = %d", h.Len())
	}
	sum := h.Summarize()
	if sum.Samples != 2 || sum.CPUMax != 40 || sum.MemoryMax != 20 || sum.RXMax != 3 || sum.TXMax != 0.5 {
		t.Fatalf("summary = %+v", sum)
	}
	samples := h.Samples()
	if samples[0].Value.Temperature == nil || *samples[0].Value.Temperature != 58 {
		t.Fatalf("temp not preserved")
	}
}

func TestHistoryWrapsWithoutGrowth(t *testing.T) {
	h := New(2)
	for i := 0; i < 10; i++ {
		h.Add(ts(i), Point{CPU: float64(i)})
	}
	if h.Len() != 2 || h.Cap() != 2 {
		t.Fatalf("len=%d cap=%d", h.Len(), h.Cap())
	}
	got := h.Samples()
	if got[0].Value.CPU != 8 || got[1].Value.CPU != 9 {
		t.Fatalf("samples = %#v", got)
	}
}

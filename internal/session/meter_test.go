package session

import (
	"testing"
	"time"
)

func t0() time.Time {
	return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
}

func TestFirstSampleAddsZero(t *testing.T) {
	m := New(DefaultMaxGap)
	m.Observe(t0(), 10, 4)
	rx, tx := m.Totals()
	if rx != 0 || tx != 0 {
		t.Fatalf("first sample totals = %v/%v, want 0/0", rx, tx)
	}
}

func TestSecondValidSampleIntegrates(t *testing.T) {
	m := New(DefaultMaxGap)
	m.Observe(t0(), 1, 0.5)
	m.Observe(t0().Add(2*time.Second), 1, 0.5)
	rx, tx := m.Totals()
	wantRx := 1 * float64(1<<20) * 2
	wantTx := 0.5 * float64(1<<20) * 2
	if rx != wantRx || tx != wantTx {
		t.Fatalf("totals = %v/%v, want %v/%v", rx, tx, wantRx, wantTx)
	}
}

func TestResetClearsTotalsAndTiming(t *testing.T) {
	m := New(DefaultMaxGap)
	m.Observe(t0(), 1, 1)
	m.Observe(t0().Add(time.Second), 1, 1)
	m.Reset()
	rx, tx := m.Totals()
	if rx != 0 || tx != 0 {
		t.Fatalf("after reset totals = %v/%v", rx, tx)
	}
	m.Observe(t0().Add(2*time.Second), 8, 8)
	rx, tx = m.Totals()
	if rx != 0 || tx != 0 {
		t.Fatalf("first sample after reset = %v/%v, want 0/0", rx, tx)
	}
}

func TestInvalidateBreaksContinuity(t *testing.T) {
	m := New(DefaultMaxGap)
	m.Observe(t0(), 1, 1)
	m.Observe(t0().Add(time.Second), 1, 1)
	beforeRx, _ := m.Totals()
	m.Invalidate()
	m.Observe(t0().Add(30*time.Second), 50, 50)
	afterRx, afterTx := m.Totals()
	if afterRx != beforeRx || afterTx != beforeRx {
		t.Fatalf("sample after invalidate should add zero: before=%v after=%v/%v", beforeRx, afterRx, afterTx)
	}
}

func TestZeroAndNegativeElapsed(t *testing.T) {
	m := New(DefaultMaxGap)
	m.Observe(t0(), 1, 1)
	m.Observe(t0(), 1, 1)
	rx, tx := m.Totals()
	if rx != 0 || tx != 0 {
		t.Fatalf("zero elapsed integrated: %v/%v", rx, tx)
	}
	m.Observe(t0().Add(-time.Second), 1, 1)
	rx, tx = m.Totals()
	if rx != 0 || tx != 0 {
		t.Fatalf("negative elapsed integrated: %v/%v", rx, tx)
	}
}

func TestExcessiveSampleGap(t *testing.T) {
	m := New(DefaultMaxGap)
	m.Observe(t0(), 1, 1)
	m.Observe(t0().Add(DefaultMaxGap), 1, 1)
	rx, tx := m.Totals()
	if rx != 0 || tx != 0 {
		t.Fatalf("gap == max should not integrate: %v/%v", rx, tx)
	}
	m.Observe(t0().Add(DefaultMaxGap+2*time.Second), 1, 1)
	rx, tx = m.Totals()
	want := 1 * float64(1<<20) * 2
	if rx != want || tx != want {
		t.Fatalf("after re-prime totals = %v/%v, want %v", rx, tx, want)
	}
}

func TestRxAndTxIndependent(t *testing.T) {
	m := New(DefaultMaxGap)
	m.Observe(t0(), 2, 0)
	m.Observe(t0().Add(time.Second), 2, 0)
	rx, tx := m.Totals()
	if tx != 0 {
		t.Fatalf("tx should stay 0, got %v", tx)
	}
	if rx != 2*float64(1<<20) {
		t.Fatalf("rx = %v", rx)
	}
	m.Reset()
	m.Observe(t0(), 0, 3)
	m.Observe(t0().Add(time.Second), 0, 3)
	rx, tx = m.Totals()
	if rx != 0 {
		t.Fatalf("rx should stay 0, got %v", rx)
	}
	if tx != 3*float64(1<<20) {
		t.Fatalf("tx = %v", tx)
	}
}

func TestNoNegativeAccumulatedResult(t *testing.T) {
	m := New(DefaultMaxGap)
	m.Observe(t0(), 1, 1)
	m.Observe(t0().Add(time.Second), -100, -100)
	rx, tx := m.Totals()
	if rx < 0 || tx < 0 {
		t.Fatalf("negative totals %v/%v", rx, tx)
	}
}

func TestPeakRatesIncludeFirstSample(t *testing.T) {
	m := New(DefaultMaxGap)
	m.Observe(t0(), 4, 1)
	snap := m.Snapshot()
	if snap.PeakRX != 4 || snap.PeakTX != 1 {
		t.Fatalf("first-sample peaks = %+v", snap)
	}
	m.Observe(t0().Add(time.Second), 2, 8)
	snap = m.Snapshot()
	if snap.PeakRX != 4 || snap.PeakTX != 8 {
		t.Fatalf("peaks = %+v", snap)
	}
}

func TestAveragesAndDuration(t *testing.T) {
	m := New(DefaultMaxGap)
	m.Observe(t0(), 1, 0.5)
	if m.Snapshot().Duration != 0 {
		t.Fatalf("first sample duration = %s", m.Snapshot().Duration)
	}
	m.Observe(t0().Add(2*time.Second), 1, 0.5)
	snap := m.Snapshot()
	if snap.Duration != 2*time.Second {
		t.Fatalf("duration = %s", snap.Duration)
	}
	if snap.AvgRX != 1 || snap.AvgTX != 0.5 {
		t.Fatalf("averages = %v/%v", snap.AvgRX, snap.AvgTX)
	}
}

func TestResetClearsPeaksAndDuration(t *testing.T) {
	m := New(DefaultMaxGap)
	m.Observe(t0(), 5, 5)
	m.Observe(t0().Add(time.Second), 5, 5)
	m.Reset()
	snap := m.Snapshot()
	if snap.PeakRX != 0 || snap.PeakTX != 0 || snap.Duration != 0 || snap.AvgRX != 0 {
		t.Fatalf("after reset %+v", snap)
	}
}

func TestInvalidatePreservesPeaksAndDuration(t *testing.T) {
	m := New(DefaultMaxGap)
	m.Observe(t0(), 3, 1)
	m.Observe(t0().Add(2*time.Second), 3, 1)
	before := m.Snapshot()
	m.Invalidate()
	after := m.Snapshot()
	if after.PeakRX != before.PeakRX || after.Duration != before.Duration {
		t.Fatalf("invalidate changed snapshot: before=%+v after=%+v", before, after)
	}
	m.Observe(t0().Add(10*time.Second), 9, 1)
	if m.Snapshot().PeakRX != 9 {
		t.Fatalf("peak after re-prime = %v", m.Snapshot().PeakRX)
	}
	if m.Snapshot().Duration != 10*time.Second {
		t.Fatalf("duration after re-prime = %s", m.Snapshot().Duration)
	}
}

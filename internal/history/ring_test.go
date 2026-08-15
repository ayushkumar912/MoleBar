package history

import (
	"testing"
	"time"
)

func ts(i int) time.Time {
	return time.Date(2026, 1, 1, 0, 0, i, 0, time.UTC)
}

func TestRingEmpty(t *testing.T) {
	r := NewRing[int](4)
	if r.Len() != 0 {
		t.Fatalf("len = %d", r.Len())
	}
	if r.Slice() != nil {
		t.Fatalf("empty slice = %#v", r.Slice())
	}
	if r.Cap() != 4 {
		t.Fatalf("cap = %d", r.Cap())
	}
}

func TestRingAppendOrder(t *testing.T) {
	r := NewRing[int](4)
	r.Append(Sample[int]{At: ts(1), Value: 1})
	r.Append(Sample[int]{At: ts(2), Value: 2})
	got := r.Slice()
	if len(got) != 2 || got[0].Value != 1 || got[1].Value != 2 {
		t.Fatalf("got %#v", got)
	}
	if !got[0].At.Equal(ts(1)) || !got[1].At.Equal(ts(2)) {
		t.Fatalf("timestamps %#v", got)
	}
}

func TestRingExactCapacity(t *testing.T) {
	r := NewRing[int](3)
	for i := 1; i <= 3; i++ {
		r.Append(Sample[int]{At: ts(i), Value: i})
	}
	if r.Len() != 3 || r.Cap() != 3 {
		t.Fatalf("len=%d cap=%d", r.Len(), r.Cap())
	}
	got := r.Slice()
	if len(got) != 3 || got[0].Value != 1 || got[2].Value != 3 {
		t.Fatalf("got %#v", got)
	}
}

func TestRingWraparound(t *testing.T) {
	r := NewRing[int](3)
	for i := 1; i <= 5; i++ {
		r.Append(Sample[int]{At: ts(i), Value: i})
	}
	if r.Len() != 3 {
		t.Fatalf("len after wrap = %d", r.Len())
	}
	got := r.Slice()
	if len(got) != 3 || got[0].Value != 3 || got[1].Value != 4 || got[2].Value != 5 {
		t.Fatalf("wrap order %#v", got)
	}
	if !got[0].At.Equal(ts(3)) || !got[2].At.Equal(ts(5)) {
		t.Fatalf("wrap timestamps %#v", got)
	}
}

func TestRingCapacityOne(t *testing.T) {
	r := NewRing[int](1)
	r.Append(Sample[int]{At: ts(1), Value: 10})
	r.Append(Sample[int]{At: ts(2), Value: 20})
	got := r.Slice()
	if r.Len() != 1 || len(got) != 1 || got[0].Value != 20 {
		t.Fatalf("cap-1 %#v len=%d", got, r.Len())
	}
}

func TestRingNoUnboundedGrowth(t *testing.T) {
	r := NewRing[int](2)
	for i := 0; i < 100; i++ {
		r.Append(Sample[int]{At: ts(i), Value: i})
		if r.Len() > r.Cap() {
			t.Fatalf("len %d exceeded cap %d", r.Len(), r.Cap())
		}
		if len(r.items) != 2 {
			t.Fatalf("backing store grew to %d", len(r.items))
		}
	}
}

func TestRingZeroCapacityBecomesOne(t *testing.T) {
	r := NewRing[int](0)
	if r.Cap() != 1 {
		t.Fatalf("cap = %d", r.Cap())
	}
	r.Append(Sample[int]{Value: 7})
	if r.Len() != 1 || r.Slice()[0].Value != 7 {
		t.Fatalf("zero-cap ring %#v", r.Slice())
	}
}

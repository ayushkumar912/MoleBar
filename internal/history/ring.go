package history

import "time"

// Sample is a timestamped value.
type Sample[T any] struct {
	At    time.Time
	Value T
}

// Ring is a bounded circular buffer. Append is O(1) and does not grow
// past the capacity fixed at construction.
type Ring[T any] struct {
	items []Sample[T]
	cap   int
	head  int
	len   int
}

// NewRing allocates a ring with a fixed capacity. Capacity is at least 1.
func NewRing[T any](capacity int) *Ring[T] {
	if capacity < 1 {
		capacity = 1
	}
	return &Ring[T]{
		items: make([]Sample[T], capacity),
		cap:   capacity,
	}
}

// Append stores s, overwriting the oldest entry when full.
func (r *Ring[T]) Append(s Sample[T]) {
	if r == nil || r.cap == 0 {
		return
	}
	r.items[r.head] = s
	r.head = (r.head + 1) % r.cap
	if r.len < r.cap {
		r.len++
	}
}

// Len is the number of stored samples.
func (r *Ring[T]) Len() int {
	if r == nil {
		return 0
	}
	return r.len
}

// Cap is the fixed capacity.
func (r *Ring[T]) Cap() int {
	if r == nil {
		return 0
	}
	return r.cap
}

// Slice returns samples in chronological order (oldest first).
// The returned slice is a copy and may be empty but never nil-grown.
func (r *Ring[T]) Slice() []Sample[T] {
	if r == nil || r.len == 0 {
		return nil
	}
	out := make([]Sample[T], r.len)
	start := 0
	if r.len == r.cap {
		start = r.head
	}
	for i := 0; i < r.len; i++ {
		out[i] = r.items[(start+i)%r.cap]
	}
	return out
}

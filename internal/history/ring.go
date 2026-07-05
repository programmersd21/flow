// internal/history/ring.go — fixed-size ring buffer for sparkline data.
// Pre-allocated; no heap allocations after construction.

package history

import "time"

var nowFunc = time.Now

type Ring struct {
	buf  []float64
	size int
	head int // index of the oldest element
	n    int // number of elements currently stored
}

func New(cap int) *Ring {
	if cap < 1 {
		cap = 1
	}
	return &Ring{buf: make([]float64, cap), size: cap}
}

func (r *Ring) Push(v float64) {
	if r.n < r.size {
		r.buf[(r.head+r.n)%r.size] = v
		r.n++
	} else {
		r.buf[r.head] = v
		r.head = (r.head + 1) % r.size
	}
}

func (r *Ring) Slice() []float64 {
	out := make([]float64, r.n)
	for i := 0; i < r.n; i++ {
		out[i] = r.buf[(r.head+i)%r.size]
	}
	return out
}

func (r *Ring) Len() int { return r.n }

func (r *Ring) Cap() int { return r.size }

func (r *Ring) Reset() {
	r.head = 0
	r.n = 0
}

// Last returns the most recently pushed value, or 0 if empty.
func (r *Ring) Last() float64 {
	if r.n == 0 {
		return 0
	}
	idx := (r.head + r.n - 1) % r.size
	return r.buf[idx]
}

// Tracker keeps session peak and today totals.
type Tracker struct {
	PeakDown  float64
	PeakUp    float64
	TodayDown float64 // bytes
	TodayUp   float64 // bytes
	Year      int
	Month     time.Month
	Day       int
}

func NewTracker() *Tracker {
	y, m, d := nowFunc().Date()
	return &Tracker{Year: y, Month: m, Day: d}
}

// Record updates peaks and today totals. intervalSecs is the sample interval.
func (t *Tracker) Record(downBps, upBps, intervalSecs float64) {
	now := nowFunc()
	y, m, d := now.Date()
	if y != t.Year || m != t.Month || d != t.Day {
		t.TodayDown = 0
		t.TodayUp = 0
		t.Year = y
		t.Month = m
		t.Day = d
	}
	if downBps > t.PeakDown {
		t.PeakDown = downBps
	}
	if upBps > t.PeakUp {
		t.PeakUp = upBps
	}
	t.TodayDown += downBps * intervalSecs
	t.TodayUp += upBps * intervalSecs
}

func (t *Tracker) ResetPeaks() {
	t.PeakDown = 0
	t.PeakUp = 0
}

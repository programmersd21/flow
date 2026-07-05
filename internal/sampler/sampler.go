// internal/sampler/sampler.go
//
// Polls OS byte counters, accumulates deltas in a sliding window,
// emits bytes/sec averaged over the window.
//
// Windows note: GetIfTable2 counters update roughly every 1 s regardless of
// read rate. The sliding window (4 slots × 250 ms = 1 s) smooths this out.

package sampler

import (
	"context"
	"time"

	"github.com/programmersd21/flow/internal/collector"
)

type Sample struct {
	DownBps   float64
	UpBps     float64
	Interface string
	At        time.Time
	Err       error
}

type entry struct {
	rx uint64
	tx uint64
	dt float64 // elapsed seconds for this slot
}

// windowSlots: targets ~1 s window. At 250 ms default = 4 slots.
const windowSlots = 4

type Sampler struct {
	col      *collector.Collector
	interval time.Duration
	Out      chan Sample

	ring  [windowSlots]entry
	head  int
	full  bool
	rxSum uint64
	txSum uint64
	dtSum float64
}

func New(col *collector.Collector, interval time.Duration) *Sampler {
	return &Sampler{
		col:      col,
		interval: interval,
		Out:      make(chan Sample, 8),
	}
}

func (s *Sampler) Run(ctx context.Context) {
	prev, err := s.col.Read()
	if err != nil {
		select {
		case s.Out <- Sample{Err: err, At: time.Now()}:
		case <-ctx.Done():
		}
		return
	}
	var prevTime time.Time

	// Prime: take two quick reads so the first emitted sample is non-zero.
	// Windows counters need at least one full second to populate.
	primeTimer := time.NewTimer(10 * time.Millisecond)
	select {
	case <-ctx.Done():
		return
	case <-primeTimer.C:
		if snap, err2 := s.col.Read(); err2 != nil {
			select {
			case s.Out <- Sample{Err: err2, At: time.Now()}:
			case <-ctx.Done():
			}
			return
		} else {
			prev = snap
			prevTime = time.Now()
		}
	}

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case t := <-ticker.C:
			snap, err := s.col.Read()
			if err != nil {
				select {
				case s.Out <- Sample{Err: err, At: t}:
				case <-ctx.Done():
				}
				return
			}

			dt := t.Sub(prevTime).Seconds()
			if dt <= 0 {
				dt = s.interval.Seconds()
			}

			var rxDelta, txDelta uint64
			if snap.RxBytes >= prev.RxBytes {
				rxDelta = snap.RxBytes - prev.RxBytes
			}
			if snap.TxBytes >= prev.TxBytes {
				txDelta = snap.TxBytes - prev.TxBytes
			}

			// Update the sliding window.
			slot := &s.ring[s.head]
			if s.full {
				s.rxSum -= slot.rx
				s.txSum -= slot.tx
				s.dtSum -= slot.dt
			}
			slot.rx = rxDelta
			slot.tx = txDelta
			slot.dt = dt
			s.rxSum += rxDelta
			s.txSum += txDelta
			s.dtSum += dt
			s.head = (s.head + 1) % windowSlots
			if s.head == 0 {
				s.full = true
			}

			windowDt := s.dtSum
			if windowDt <= 0 {
				windowDt = float64(windowSlots) * s.interval.Seconds()
			}

			sample := Sample{
				DownBps:   float64(s.rxSum) / windowDt,
				UpBps:     float64(s.txSum) / windowDt,
				Interface: snap.Interface,
				At:        t,
			}

			select {
			case s.Out <- sample:
			default:
			}

			prev = snap
			prevTime = t
		}
	}
}

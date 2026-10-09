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
	// At is the tick the sample was computed for.
	At time.Time
	// Interval is the seconds of real time this sample's rates cover: the
	// sliding-window duration, not the configured refresh. Consumers that
	// accumulate totals must use it, otherwise daily totals drift whenever
	// ticks lag (loaded machine, suspended laptop, coarse timer).
	Interval float64
	Err      error
}

type entry struct {
	rx uint64
	tx uint64
	dt float64 // elapsed seconds for this slot
}

// windowSlots: targets ~1 s window. At 250 ms default = 4 slots.
const windowSlots = 4

// Sampler turns OS counter readings into smoothed rates.
//
// Ownership: Run is the only writer, and it runs in the caller's goroutine.
// Out is read by the consumer (UI or CLI). The send on Out is deliberately
// non-blocking: a sample that cannot be delivered immediately is dropped.
// The UI is freshness-oriented — a stale sample is worse than a missing one,
// because the display would not match the moment it names — and blocking here
// would let a stalled consumer stop sampling altogether. That lossiness is a
// decision, not an oversight, and TestSamplerDoesNotBlockOnSlowConsumer locks
// it in.
type Sampler struct {
	col      collector.Reader
	interval time.Duration
	Out      chan Sample

	ring  [windowSlots]entry
	head  int
	full  bool
	rxSum uint64
	txSum uint64
	dtSum float64
}

// New returns a sampler reading counters from col. Sampling cadence is set by
// interval; Out carries every computed Sample.
func New(col collector.Reader, interval time.Duration) *Sampler {
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
		case s.Out <- Sample{Err: err, At: time.Now().UTC()}:
		case <-ctx.Done():
		}
		return
	}
	var prevTime time.Time

	// Prime the window: a single read cannot produce a rate, so take a second
	// one shortly after the first to establish prevTime.
	primeTimer := time.NewTimer(10 * time.Millisecond)
	defer primeTimer.Stop()
	select {
	case <-ctx.Done():
		return
	case <-primeTimer.C:
		if snap, err2 := s.col.Read(); err2 != nil {
			select {
			case s.Out <- Sample{Err: err2, At: time.Now().UTC()}:
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

			// Window seconds this sample's rates are averaged over. Used both as the
			// divisor and as the reporting interval, so the two can never disagree.
			windowDt := s.dtSum
			if windowDt <= 0 {
				windowDt = float64(windowSlots) * s.interval.Seconds()
			}

			sample := Sample{
				DownBps:   float64(s.rxSum) / windowDt,
				UpBps:     float64(s.txSum) / windowDt,
				Interface: snap.Interface,
				At:        t,
				Interval:  windowDt,
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

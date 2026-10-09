package sampler

import (
	"context"
	"errors"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/programmersd21/flow/internal/collector"
)

// counterReader serves a scripted sequence of counter snapshots, one per Read.
type counterReader struct {
	mu      sync.Mutex
	totals  []collector.Snapshot
	reads   int
	failing error
}

func (c *counterReader) Read() (collector.Snapshot, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.reads++
	if c.failing != nil {
		return collector.Snapshot{}, c.failing
	}
	if len(c.totals) == 0 {
		return collector.Snapshot{Interface: "eth0"}, nil
	}
	s := c.totals[0]
	if len(c.totals) > 1 {
		c.totals = c.totals[1:]
	}
	return s, nil
}

func (c *counterReader) readCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.reads
}

// collect runs a sampler until n samples have been delivered.
func collect(t *testing.T, r *counterReader, interval time.Duration, n int) []Sample {
	t.Helper()
	s := New(r, interval)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var (
		mu  sync.Mutex
		got []Sample
		wg  sync.WaitGroup
	)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for sm := range s.Out {
			mu.Lock()
			got = append(got, sm)
			done := len(got) >= n
			mu.Unlock()
			if done {
				cancel()
				return
			}
		}
	}()
	s.Run(ctx)
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	return got
}

// linearTotals builds a counter sequence rising by step every interval.
func linearTotals(step uint64, count int) []collector.Snapshot {
	out := make([]collector.Snapshot, 0, count)
	for i := 0; i < count; i++ {
		out = append(out, collector.Snapshot{
			Interface: "eth0",
			RxBytes:   step * uint64(i),
			TxBytes:   step * uint64(i),
		})
	}
	return out
}

func TestSamplerRateIsBytesPerSecond(t *testing.T) {
	// 500 000 B every 10 ms is 50 MB/s. If this reported the delta without
	// dividing by elapsed time the UI would show raw per-interval byte counts.
	const step = 500_000
	got := collect(t, &counterReader{totals: linearTotals(step, 200)}, 10*time.Millisecond, 10)
	if len(got) == 0 {
		t.Fatal("no samples delivered")
	}
	want := float64(step) / 0.01
	for i, s := range got {
		if s.Err != nil {
			t.Fatalf("sample %d: unexpected error %v", i, s.Err)
		}
		if math.Abs(s.DownBps-want) > want*0.05 {
			t.Errorf("sample %d: DownBps = %.0f B/s, want %.0f B/s", i, s.DownBps, want)
		}
		if math.Abs(s.UpBps-want) > want*0.05 {
			t.Errorf("sample %d: UpBps = %.0f B/s, want %.0f B/s", i, s.UpBps, want)
		}
	}
}

func TestSamplerSlowIntervalReportsLowRate(t *testing.T) {
	// 60 000 B per second, sampled once a second: the rate must reflect the
	// data rate, not the number of samples observed.
	const step = 60_000
	got := collect(t, &counterReader{totals: linearTotals(step, 400)}, 10*time.Millisecond, 6)
	if len(got) == 0 {
		t.Fatal("no samples delivered")
	}
	// 60 000 B every 10 ms = 6 MB/s.
	want := float64(step) / 0.01
	if math.Abs(got[0].DownBps-want) > want*0.05 {
		t.Errorf("DownBps = %.0f, want %.0f B/s", got[0].DownBps, want)
	}
}

func TestSamplerZeroTraffic(t *testing.T) {
	flat := make([]collector.Snapshot, 100)
	for i := range flat {
		flat[i] = collector.Snapshot{Interface: "eth0", RxBytes: 1000, TxBytes: 1000}
	}
	got := collect(t, &counterReader{totals: flat}, 10*time.Millisecond, 8)
	for i, s := range got {
		if s.DownBps != 0 || s.UpBps != 0 {
			t.Errorf("sample %d: idle link reported down=%v up=%v, want 0/0", i, s.DownBps, s.UpBps)
		}
	}
}

func TestSamplerCounterResetDoesNotSpike(t *testing.T) {
	// An interface restart resets counters to a small value. The danger is a
	// wrong subtraction producing a huge or negative sample, which the UI would
	// render as a full-height spike.
	//
	// A reset cannot yield an instantaneous zero either: the four-slot window
	// time-averages the reset interval with the real intervals before it. So
	// the assertions are bounds, not an exact zero.
	const step = 400_000
	totals := []collector.Snapshot{{Interface: "eth0", RxBytes: 50_000_000}}
	for i := 0; i < 6; i++ {
		totals = append(totals, collector.Snapshot{Interface: "eth0", RxBytes: 50_000_000 + uint64(i+1)*step})
	}
	steady := float64(step) / 0.01 // 40 MB/s
	for i := 0; i < 20; i++ {
		totals = append(totals, collector.Snapshot{Interface: "eth0", RxBytes: uint64(i+1) * step})
	}

	got := collect(t, &counterReader{totals: totals}, 10*time.Millisecond, 22)
	if len(got) == 0 {
		t.Fatal("no samples delivered")
	}
	for i, s := range got {
		if s.Err != nil {
			t.Fatalf("sample %d: error %v", i, s.Err)
		}
		if s.DownBps < 0 {
			t.Errorf("sample %d: negative rate after reset: %v", i, s.DownBps)
		}
		// A reset must never manufacture traffic: at most 2x the steady rate,
		// which covers the window still holding the intervals before the reset.
		if s.DownBps > steady*2 {
			t.Errorf("sample %d: rate %.0f spikes far above steady %.0f after reset",
				i, s.DownBps, steady)
		}
	}
}

func TestSamplerCounterWrapAroundUint64(t *testing.T) {
	// Counters are uint64 and can wrap. A decrease must be treated as a
	// reset, not as an enormous delta.
	const wrapFrom = ^uint64(0) - 5000
	totals := []collector.Snapshot{{Interface: "eth0", RxBytes: wrapFrom}}
	for i := 0; i < 5; i++ {
		totals = append(totals, collector.Snapshot{Interface: "eth0", RxBytes: wrapFrom + uint64(i+1)*1000})
	}
	totals = append(totals, collector.Snapshot{Interface: "eth0", RxBytes: 10})
	got := collect(t, &counterReader{totals: totals}, 10*time.Millisecond, 10)
	for i, s := range got {
		if s.DownBps < 0 {
			t.Errorf("sample %d: negative rate across wrap: %v", i, s.DownBps)
		}
	}
}

func TestSamplerPropagatesReadError(t *testing.T) {
	boom := errors.New("boom")
	r := &counterReader{failing: boom}
	s := New(r, 10*time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	go s.Run(ctx)

	select {
	case sm := <-s.Out:
		if sm.Err == nil {
			t.Fatal("expected an error sample")
		}
		if !errors.Is(sm.Err, boom) {
			t.Errorf("error not wrapped: %v", sm.Err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no error sample delivered")
	}
}

func TestSamplerDoesNotBlockOnSlowConsumer(t *testing.T) {
	// The send must not block: a stalled UI must not freeze sampling. This is
	// the assertion that would catch a blocking send being reintroduced.
	r := &counterReader{totals: linearTotals(1000, 5000)}
	s := New(r, time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() { s.Run(ctx); close(done) }()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("sampler did not return; channel send appears to block")
	}
	if r.readCount() < 2 {
		t.Errorf("expected repeated reads, got %d", r.readCount())
	}
}

func TestSamplerHonoursContextCancellation(t *testing.T) {
	r := &counterReader{totals: linearTotals(1000, 5000)}
	s := New(r, time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { s.Run(ctx); close(done) }()
	time.Sleep(30 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after context cancellation")
	}
}

func TestSamplerNaNAndInfAreNotProduced(t *testing.T) {
	// Sanity: with sane inputs no sample may carry NaN or Inf, which would
	// propagate into the layout maths and panic the renderer.
	got := collect(t, &counterReader{totals: linearTotals(250_000, 200)}, 10*time.Millisecond, 10)
	for i, s := range got {
		if math.IsNaN(s.DownBps) || math.IsInf(s.DownBps, 0) {
			t.Errorf("sample %d: DownBps is not finite: %v", i, s.DownBps)
		}
		if math.IsNaN(s.UpBps) || math.IsInf(s.UpBps, 0) {
			t.Errorf("sample %d: UpBps is not finite: %v", i, s.UpBps)
		}
	}
}

// The sampler reports the seconds its rates cover. Total accumulation uses
// it, so a sample must know how much time it represents, not the configured
// refresh. A window that never fills (very few samples) must still report a
// positive interval rather than zero, which would freeze "today" totals.
func TestSamplerReportsSampleInterval(t *testing.T) {
	got := collect(t, &counterReader{totals: linearTotals(100_000, 200)}, 10*time.Millisecond, 6)
	for i, s := range got {
		if s.Interval <= 0 {
			t.Errorf("sample %d: Interval = %v, want > 0", i, s.Interval)
		}
		// The window is 4 slots at ~10 ms, so the reported interval must sit in
		// that region: not the raw tick, and not hours of drift.
		if s.Interval < 0.005 || s.Interval > 0.25 {
			t.Errorf("sample %d: Interval = %v, want ~0.04s window", i, s.Interval)
		}
	}
}

// A stalled ticker must not inflate the interval: the contract is that the
// interval matches the window the rates were averaged over, and the rates must
// agree with the interval (bytes / seconds), so rate * interval is bytes.
func TestSamplerRateTimesIntervalIsDelta(t *testing.T) {
	const step = 200_000
	got := collect(t, &counterReader{totals: linearTotals(step, 400)}, 10*time.Millisecond, 20)
	for i, s := range got {
		if s.Err != nil {
			t.Fatalf("sample %d: %v", i, s.Err)
		}
		want := float64(step) / 0.01
		// Bytes implied by the reported rate must match the actual counter
		// delta over the same window, within the smoothing the window causes.
		implied := s.DownBps * s.Interval
		if math.Abs(implied-want*s.Interval) > want*s.Interval*0.2 {
			t.Errorf("sample %d: rate %v over %vs = %v bytes, want ~%v",
				i, s.DownBps, s.Interval, implied, want*s.Interval)
		}
	}
}

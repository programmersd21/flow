package ui

import (
	"flag"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/programmersd21/flow/internal/history"
	"github.com/programmersd21/flow/internal/theme"
)

var updateGolden = flag.Bool("update-golden", false, "rewrite golden frame files")

// goldenSamples is deterministic synthetic traffic: a slow oscillation with a
// burst near the end, so goldens exercise idle, plateau, and peak rendering.
func goldenSamples(n int, base, amp int, burstAt int, burstAmp float64) []float64 {
	out := make([]float64, n)
	for i := range out {
		t := float64(i) / 10.0
		v := float64(base) + float64(amp)*(0.5+0.5*math.Sin(t*0.5)) + float64(amp)*0.2*math.Sin(t*1.7+1.0)
		if i >= burstAt && i < burstAt+20 {
			v += burstAmp * (1 - float64(i-burstAt)/20.0)
		}
		if v < 0 {
			v = 0
		}
		out[i] = v
	}
	return out
}

// goldenModel builds a deterministic hero model from synthetic data.
func goldenModel(w, h int) Model {
	theme.SetTheme("default")
	dh := history.New(36000)
	uh := history.New(36000)
	var lastDown, lastUp, peakDown, peakUp float64
	down := goldenSamples(480, 1_200_000, 800_000, 380, 3_000_000)
	up := goldenSamples(480, 180_000, 120_000, 400, 500_000)
	for i := range down {
		dh.Push(down[i])
		uh.Push(up[i])
		lastDown, lastUp = down[i], up[i]
		if down[i] > peakDown {
			peakDown = down[i]
		}
		if up[i] > peakUp {
			peakUp = up[i]
		}
	}
	tr := history.NewTracker()
	tr.PeakDown = peakDown
	tr.PeakUp = peakUp
	tr.TodayDown = 16_900_000
	tr.TodayUp = 70_300_000
	return Model{
		width:  w,
		height: h,
		rates: rateState{
			lastSample:     time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC),
			pingLatency:    46 * time.Millisecond,
			animDown:       lastDown,
			animUp:         lastUp,
			rollingMaxDown: peakDown,
			rollingMaxUp:   peakUp,
		},
		downHist:        dh,
		upHist:          uh,
		tracker:         tr,
		refreshInterval: 100 * time.Millisecond,

		nowOverride: time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC),
		viewMode:    ViewHero,
		iface:       ifaceState{name: "wlan0"},

		windowSecs: 60,
		showGrid:   true,
		noAnim:     true, // static frames for goldens
		launchDone: true,
	}
}

// TestGoldenHeroFrames snapshots the hero screen at 4 fixed sizes.
// Regenerate with: go test ./internal/ui -run TestGoldenHeroFrames -update-golden
func TestGoldenHeroFrames(t *testing.T) {
	sizes := [][2]int{{60, 20}, {80, 24}, {100, 30}, {140, 40}}
	for _, sz := range sizes {
		w, h := sz[0], sz[1]
		name := filepath.Join("testdata", goldenName(w, h))
		m := goldenModel(w, h)
		lines := dashboardContentLines(m, ViewHero)
		frame := stripANSI(strings.Join(lines, "\n")) + "\n"

		if *updateGolden {
			if err := os.WriteFile(name, []byte(frame), 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		want, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("missing golden %s (run with -update-golden): %v", name, err)
		}
		if frame != string(want) {
			t.Errorf("golden mismatch at %dx%d\n--- got ---\n%s\n--- want ---\n%s", w, h, frame, want)
		}
	}
}

func goldenName(w, h int) string {
	return "hero_" + itoa(w) + "x" + itoa(h) + ".golden"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [8]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

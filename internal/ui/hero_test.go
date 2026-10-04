package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/programmersd21/flow/internal/history"
	"github.com/programmersd21/flow/internal/theme"
)

func baseModel(w, h int) Model {
	theme.SetTheme("default")
	tr := history.NewTracker()
	tr.PeakDown = 4_000_000
	tr.PeakUp = 900_000
	tr.TodayDown = 16_900_000
	tr.TodayUp = 70_300_000
	return Model{
		width:           w,
		height:          h,
		animDown:        2_000_000,
		animUp:          195_000,
		rollingMaxDown:  4_000_000,
		rollingMaxUp:    900_000,
		downHist:        history.New(60),
		upHist:          history.New(60),
		tracker:         tr,
		refreshInterval: 100 * time.Millisecond,
		lastSampleTime:  time.Now(),
		ifaceName:       "wlan0",
		windowSecs:      60,
		noAnim:          true,
		launchDone:      true,
	}
}

func TestWindowCycling(t *testing.T) {
	m := baseModel(80, 24)
	if m.windowSecs != 60 {
		t.Fatalf("default window should be 60s, got %d", m.windowSecs)
	}
	m.windowIdx = (m.windowIdx + 1) % len(timeWindows)
	m.windowSecs = timeWindows[m.windowIdx]
	if m.windowSecs != 300 {
		t.Errorf("after one cycle expected 300s, got %d", m.windowSecs)
	}
	m2 := m.WithWindow("1h")
	if m2.windowSecs != 3600 {
		t.Errorf("WithWindow(1h) = %d, want 3600", m2.windowSecs)
	}
	m3 := m.WithWindow("bogus")
	if m3.windowSecs != m.windowSecs {
		t.Error("WithWindow with invalid value should keep current window")
	}
}

func TestHeroNarrowStacksDigits(t *testing.T) {
	m := baseModel(60, 24)
	lines := dashboardContentLines(m, ViewHero)
	content := stripANSI(strings.Join(lines, "\n"))
	if !strings.Contains(content, "peak") || !strings.Contains(content, "session") {
		t.Error("narrow hero missing captions")
	}
}

func TestWindowedSamples(t *testing.T) {
	m := baseModel(80, 24)
	r := history.New(1000)
	for i := 0; i < 500; i++ {
		r.Push(float64(i))
	}
	m.windowSecs = 1 // 1s at 100ms = 10 samples
	got := m.windowedSamples(r)
	if len(got) != 10 {
		t.Errorf("windowedSamples for 1s window = %d samples, want 10", len(got))
	}
	m.windowSecs = 86400
	got = m.windowedSamples(r)
	if len(got) != 500 {
		t.Errorf("24h window should return all 500 samples, got %d", len(got))
	}
}

func TestScaleCycling(t *testing.T) {
	m := baseModel(80, 24)
	if scaleModeName(m.scaleMode) != "auto" {
		t.Errorf("default scale = %q, want auto", scaleModeName(m.scaleMode))
	}
	if scaleExponent(1) != 1.0 {
		t.Error("linear exponent should be 1.0")
	}
	if scaleExponent(2) >= scaleExponent(0) {
		t.Error("sqrt should compress more than auto")
	}
}

// TestGraphScaleNeverClipsSteadyTraffic is a regression test: an earlier
// ceiling of peak*0.6 sat BELOW typical values whenever traffic was steady,
// so the whole waveform clamped to solid blocks and lost all shape.
func TestGraphScaleNeverClipsSteadyTraffic(t *testing.T) {
	// Steady traffic: nothing near zero, peak only ~1.7x typical.
	samples := make([]float64, 200)
	for i := range samples {
		samples[i] = 10_000 + float64(i%37)*100 // ~10.0-13.6 KB/s
	}
	ceil := graphScale(samples, 13_600) // rollingMax == peak
	peak := samples[len(samples)-1]
	for _, v := range samples {
		if normVal(v, ceil, scaleExponent(0)) >= 0.999 {
			t.Fatalf("value %v clips against ceiling %v (peak %v)", v, ceil, peak)
		}
	}
}

// TestGraphScaleUsesVisiblePeakNotRollingMax: a burst that already scrolled
// out of the window must not squash the traffic still on screen.
func TestGraphScaleUsesVisiblePeakNotRollingMax(t *testing.T) {
	steady := make([]float64, 200)
	for i := range steady {
		steady[i] = 8_000 + float64(i%13)*10
	}
	ceil := graphScale(steady, 50_000_000) // rolling max from an old burst
	peak := steady[len(steady)-1]
	if normVal(peak, ceil, scaleExponent(0)) < 0.6 {
		t.Errorf("visible peak should stay tall on screen, got %.2f (ceiling %v, peak %v)",
			normVal(peak, ceil, scaleExponent(0)), ceil, peak)
	}
}

// TestGraphScaleFloorForIdle: an idle link must not magnify noise to full height.
func TestGraphScaleFloorForIdle(t *testing.T) {
	noise := []float64{200, 300, 150, 250}
	ceil := graphScale(noise, 100)
	if ceil < 10_000 {
		t.Errorf("idle ceiling should respect the 10 KB/s floor, got %v", ceil)
	}
	if got := normVal(300, ceil, scaleExponent(0)); got > 0.11 {
		t.Errorf("idle noise should stay near the axis, got %.3f", got)
	}
}

// The centering contract: each hero block has a fixed width, every row is
// centered inside that width, and the two blocks are centered as one group.
// So changing the value (0 vs 248000 vs 2410000) must not move anything.
func TestJoinCenteredStableAcrossValues(t *testing.T) {
	theme.SetTheme("default")
	width := 96
	// Vary only the download side (upload fixed at 197 B/s): same-unit values
	// must produce byte-identical row widths. Crossing a unit boundary
	// (B/s -> KB/s) legitimately changes the caption width, so each case is
	// compared against the first value in its own unit group.
	groups := [][]float64{{0, 4, 9}, {9_880, 11_200, 40_000}, {248_000, 500_000}, {2_410_000, 5_000_000}}
	for _, group := range groups {
		var baseline []int
		for _, down := range group {
			pair := [2]float64{down, 197}
			dn := renderHeroDigits(pair[0], false, true)
			up := renderHeroDigits(pair[1], false, false)
			dn = append(dn, renderHeroCaption(pair[0], false, true, "peak 4.5 MB/s"))
			up = append(up, renderHeroCaption(pair[1], false, false, "session 1.1 GB"))
			rows := joinCentered(dn, up, width)
			widths := make([]int, len(rows))
			for i, r := range rows {
				w := lipgloss.Width(r)
				widths[i] = w
				if w > width {
					t.Errorf("value %v row %d: width %d exceeds %d", pair, i, w, width)
				}
			}
			if baseline == nil {
				baseline = widths
				continue
			}
			// Every row must be the same width for every value: nothing moves.
			for i := range widths {
				if widths[i] != baseline[i] {
					t.Errorf("value %v row %d: width %d, want %d (layout moved)",
						pair, i, widths[i], baseline[i])
				}
			}
			// The pair must also sit centered, not pinned left: the first
			// content column has to be well clear of column 0. (A past
			// regression dropped the group offset and every row started
			// at the left edge.)
			runes := []rune(stripANSI(rows[0]))
			lm := 0
			for lm < len(runes) && runes[lm] == ' ' {
				lm++
			}
			if lm < 4 {
				t.Errorf("value %v: group starts at col %d, expected centered (lm>=4)",
					pair, lm)
			}
			// Centering itself is centerInline's job downstream; what this
			// unit must guarantee is that every row has the same width so
			// the block centers as one rigid unit instead of drifting.
			first := widths[0]
			for i, w := range widths[1:] {
				if w != first {
					t.Errorf("value %v row %d: width %d != first row %d (block not rigid)",
						pair, i+1, w, first)
				}
			}
		}
	}
}

package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/programmersd21/flow/internal/history"
	"github.com/programmersd21/flow/internal/theme"
)

// baseModel returns a ready-to-render model for view tests.
func baseModel(t testing.TB, w, h int) Model {
	t.Helper()
	m := newTestModel(t, w, h)
	m.rates.animDown = 2_000_000
	m.rates.animUp = 195_000
	return m
}

func TestWindowCycling(t *testing.T) {
	m := baseModel(t, 80, 24)
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
	m := baseModel(t, 60, 24)
	lines := dashboardContentLines(m, ViewHero)
	content := stripANSI(strings.Join(lines, "\n"))
	if !strings.Contains(content, "peak") || !strings.Contains(content, "session") {
		t.Error("narrow hero missing captions")
	}
}

func TestWindowedSamples(t *testing.T) {
	m := baseModel(t, 80, 24)
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
	m := baseModel(t, 80, 24)
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

// The centering contract, in two halves:
//
//  1. joinCentered returns rows that are ALL the same width (a rigid unit),
//     so the downstream centerFrame treats the pair as one block. Pre-padding
//     here would be centered a second time and shift the pair right.
//  2. The full View() output centers that unit in the terminal.
//
// Same-unit values must produce byte-identical row widths: nothing moves while
// the value stays in its unit. Crossing a unit boundary (B/s -> KB/s)
// legitimately changes the caption width.
func TestJoinCenteredStableAcrossValues(t *testing.T) {
	theme.SetTheme("default")
	width := 96
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
			for i := range widths {
				if widths[i] != baseline[i] {
					t.Errorf("value %v row %d: width %d, want %d (layout moved)",
						pair, i, widths[i], baseline[i])
				}
			}
			// Rigid unit: every row identical width, so centering moves them together.
			for i := 1; i < len(widths); i++ {
				if widths[i] != widths[0] {
					t.Errorf("value %v: row %d width %d != row 0 width %d (unit not rigid)",
						pair, i, widths[i], widths[0])
				}
			}
		}
	}
}

// TestHeroPairCenteredOnScreen: through the full View(), the digit pair sits
// centered in the terminal.
func TestHeroPairCenteredOnScreen(t *testing.T) {
	theme.SetTheme("default")
	for _, w := range []int{80, 96, 120} {
		m := goldenModel(w, 30)
		m.rates.animDown, m.rates.animUp = 248_000, 197
		out := stripANSI(m.View())
		var first string
		for _, line := range strings.Split(out, "\n") {
			if strings.Contains(line, "█████") {
				first = line
				break
			}
		}
		if first == "" {
			t.Fatalf("w=%d: no digit row found", w)
		}
		runes := []rune(first)
		lm := 0
		for lm < len(runes) && runes[lm] == ' ' {
			lm++
		}
		rm := len(runes)
		for rm > 0 && runes[rm-1] == ' ' {
			rm--
		}
		if d := lm - (w - rm); d < -2 || d > 2 {
			t.Errorf("w=%d: digit pair not centered (margins %d/%d)", w, lm, w-rm)
		}
	}
}

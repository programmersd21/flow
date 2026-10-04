package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/programmersd21/flow/internal/history"
)

func TestCompactViewRendersMinimalRows(t *testing.T) {
	tracker := history.NewTracker()
	tracker.PeakDown = 7 * 1024
	tracker.PeakUp = 9 * 1024
	m := Model{
		width:           80,
		height:          24,
		animDown:        3 * 1024,
		animUp:          5 * 1024,
		rollingMaxDown:  3 * 1024,
		rollingMaxUp:    5 * 1024,
		downHist:        history.New(60),
		upHist:          history.New(60),
		tracker:         tracker,
		refreshInterval: time.Second,
		lastSampleTime:  time.Now(),
		viewMode:        ViewCompact,
		displayFilter:   DisplayBoth,
	}

	content := strings.Join(dashboardContentLines(m, ViewCompact), "\n")
	for _, text := range []string{"download", "upload", "3 KB/s", "5 KB/s", "7 KB/s", "9 KB/s"} {
		if count := strings.Count(content, text); count != 1 {
			t.Errorf("Compact view contains %q %d times; expected once", text, count)
		}
	}
	if strings.ContainsAny(content, "╭╮╰╯│") {
		t.Errorf("Compact view contains box-drawing chrome: %q", content)
	}
	if got := lipgloss.Width(content); got > 80 {
		t.Errorf("Compact view width = %d; expected at most 80", got)
	}
}

// Regression test: ensure view never exceeds terminal height
func TestViewNeverExceedsTerminalHeight(t *testing.T) {
	widths := []int{30, 40, 50, 60, 70, 80, 100, 120}
	heights := []int{4, 6, 10, 14, 16, 18, 20, 22, 24, 26, 28, 30, 32, 36, 40, 50}

	for _, w := range widths {
		for _, h := range heights {
			m := Model{
				width:           w,
				height:          h,
				tracker:         history.NewTracker(),
				downHist:        history.New(60),
				upHist:          history.New(60),
				refreshInterval: time.Second,
				lastSampleTime:  time.Now(),
			}

			mode, lines := pickViewModeAndContent(m)
			if mode == ViewTiny {
				continue // single line, always fits
			}

			rawLines := len(lines)
			if rawLines > h {
				t.Errorf("w=%d h=%d mode=%v: %d lines exceeds terminal height %d", w, h, mode, rawLines, h)
			}
		}
	}
}

// Regression test: help overlay must fit any terminal — no line wider
// than w, no more lines than h, and the title + close hint always visible.
func TestHelpOverlayFitsTerminal(t *testing.T) {
	sizes := [][2]int{
		{80, 24}, {100, 30}, {140, 40},
		{60, 20}, {60, 15}, {40, 12}, {50, 10},
		{120, 30}, {100, 18}, {70, 16},
	}
	for _, sz := range sizes {
		w, h := sz[0], sz[1]
		m := Model{width: w, height: h, showHelp: true}
		out := renderHelp(m)
		lines := strings.Split(out, "\n")
		if len(lines) > h {
			t.Errorf("%dx%d: help frame has %d lines, exceeds height %d", w, h, len(lines), h)
		}
		for i, ln := range lines {
			if got := lipgloss.Width(ln); got > w {
				t.Errorf("%dx%d: line %d width %d exceeds %d: %q", w, h, i, got, w, ln)
			}
		}
		if !strings.Contains(out, "keyboard shortcuts") {
			t.Errorf("%dx%d: missing title", w, h)
		}
		if !strings.Contains(out, "esc") {
			t.Errorf("%dx%d: missing close hint", w, h)
		}
	}
}

// Test adaptive mode selection
func TestEffectiveViewModeFitsBeforeClamping(t *testing.T) {
	widths := []int{60, 70, 80, 100, 120}
	heights := []int{16, 18, 20, 22, 24, 26, 28, 30, 32, 36, 40, 50}

	for _, w := range widths {
		for _, h := range heights {
			m := Model{
				width:           w,
				height:          h,
				tracker:         history.NewTracker(),
				downHist:        history.New(60),
				upHist:          history.New(60),
				refreshInterval: time.Second,
				lastSampleTime:  time.Now(),
			}

			mode, lines := pickViewModeAndContent(m)
			if mode == ViewTiny {
				continue // single line, always fits
			}

			content := strings.Join(lines, "\n")
			got := strings.Count(content, "\n") + 1
			if got > h {
				t.Errorf("w=%d h=%d: mode=%v needs %d lines, exceeds height %d", w, h, mode, got, h)
			}
		}
	}
}

// Regression: the theme picker overflowed the terminal, clipping the footer.
// The row budget ignored the "more" indicators, and the description width was
// measured against the terminal instead of the box, so long descriptions
// wrapped and consumed extra rows.
func TestThemeMenuNeverOverflows(t *testing.T) {
	sizes := [][2]int{
		{80, 24}, {60, 20}, {100, 40}, {50, 14},
		{120, 30}, {70, 18}, {40, 12}, {200, 60}, {30, 10},
	}
	for _, sz := range sizes {
		w, h := sz[0], sz[1]
		for _, sel := range []int{0, 3, 7, 10} {
			m := Model{
				width: w, height: h,
				tracker:           history.NewTracker(),
				downHist:          history.New(60),
				upHist:            history.New(60),
				showThemes:        true,
				themeSelectionIdx: sel,
				launchDone:        true,
			}
			out := renderThemes(m)
			lines := strings.Split(out, "\n")
			if len(lines) > h {
				t.Errorf("%dx%d sel=%d: menu is %d lines, exceeds height %d",
					w, h, sel, len(lines), h)
			}
			if got := lipgloss.Width(out); got > w {
				t.Errorf("%dx%d sel=%d: menu width %d exceeds %d", w, h, sel, got, w)
			}
			// the cancel hint must always be reachable
			if !strings.Contains(stripANSI(out), "esc") {
				t.Errorf("%dx%d sel=%d: footer hint missing (menu clipped)", w, h, sel)
			}
		}
	}
}

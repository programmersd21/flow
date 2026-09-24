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

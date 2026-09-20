package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/programmersd21/flow/internal/history"
)

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

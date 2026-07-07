package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/programmersd21/flow/internal/history"
)

// Regression test for a bug where shrinking the terminal produced a
// dashboard taller than the actual viewport, causing the terminal to
// scroll and clip the top of the TUI (title/graphs) instead of adapting
// the view mode.
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

			expectedMode := m.effectiveViewMode()
			rawLines := dashboardLineCount(m, expectedMode)
			if rawLines > h && expectedMode != ViewTiny {
				t.Errorf("w=%d h=%d mode=%v: untruncated lines %d exceeds terminal height %d", w, h, expectedMode, rawLines, h)
			}
		}
	}
}

// TestEffectiveViewModeFitsBeforeClamping guards against a regression where
// the height measurement used to pick a view mode read centerFrame's
// already-clamped output instead of the raw content: since the clamp
// truncates everything to the terminal height anyway, that made every
// candidate mode look like it "fit", so effectiveViewMode always picked
// Hero and just relied on the clamp to hide the overflow — defeating
// adaptive mode selection while still passing TestViewNeverExceedsTerminalHeight
// above. This test deliberately uses dashboardContentLines directly (not
// dashboardLineCount) as an independent ground truth, so it still catches
// the bug even if dashboardLineCount itself is the thing that regresses.
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

			mode := m.effectiveViewMode()
			if mode == ViewTiny {
				continue // single centered line, always fits trivially
			}
			content := strings.Join(dashboardContentLines(m, mode), "\n")
			if got := strings.Count(content, "\n") + 1; got > h {
				t.Errorf("w=%d h=%d: effectiveViewMode chose mode=%v needing %d pre-clamp lines, which doesn't fit (mode selection is not actually adaptive)", w, h, mode, got)
			}
		}
	}
}

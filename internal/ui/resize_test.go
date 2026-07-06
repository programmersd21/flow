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

			out := m.View()
			lines := strings.Count(out, "\n") + 1
			if lines > h {
				t.Errorf("w=%d h=%d mode=%v: rendered %d lines, exceeds terminal height", w, h, m.effectiveViewMode(), lines)
			}
		}
	}
}

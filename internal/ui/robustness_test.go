package ui

import (
	"math"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
)

// The invariant under test: for any terminal geometry and any model state,
// rendering must not panic, must not exceed the terminal height, and must not
// emit invalid UTF-8. A TUI that draws outside its own frame corrupts the
// terminal, so these are checked for every mode rather than for a few sizes.

// sizesUnderTest spans 1 column to wide, and 1 row to tall.
var widthsUnderTest = []int{1, 2, 20, 30, 40, 50, 60, 80, 100, 120, 160, 200, 400}
var heightsUnderTest = []int{1, 2, 3, 5, 8, 10, 20, 24, 40, 80}

var allModes = []ViewMode{ViewHero, ViewCompact, ViewMini, ViewTiny}

// TestRenderingNeverExceedsTerminal covers the whole grid in one pass. It is a
// table walk rather than a golden per size: the point is the absence of
// crashes and overflow, not byte-exact output.
func TestRenderingNeverExceedsTerminal(t *testing.T) {
	for _, w := range widthsUnderTest {
		for _, h := range heightsUnderTest {
			m := goldenModel(w, h)
			for _, mode := range allModes {
				_, lines := pickViewModeAndContent(m)
				if len(lines) > h {
					t.Errorf("%dx%d mode=%v: %d lines exceeds height %d",
						w, h, mode, len(lines), h)
				}
				for i, line := range lines {
					if got := lipgloss.Width(line); got > w {
						t.Errorf("%dx%d mode=%v: line %d is %d cols, terminal is %d: %q",
							w, h, mode, i, got, w, truncate(line, 60))
					}
					if !utf8.ValidString(line) {
						t.Errorf("%dx%d mode=%v: line %d is not valid UTF-8", w, h, mode, i)
					}
				}
			}
		}
	}
}

// TestRenderingSurvivesHostileState renders every mode with values that have
// historically caused trouble: no data, zero traffic, NaN, infinities,
// enormous rates, negative values, and very long names.
func TestRenderingSurvivesHostileState(t *testing.T) {
	long := strings.Repeat("verylonginterfacename", 8)

	mutators := map[string]func(m *Model){
		"no data":        func(m *Model) {},
		"zero rates":     func(m *Model) { m.rates.animDown, m.rates.animUp = 0, 0 },
		"huge rates":     func(m *Model) { m.rates.animDown, m.rates.animUp = 1e18, 1e18 },
		"NaN rates":      func(m *Model) { m.rates.animDown, m.rates.animUp = math.NaN(), math.NaN() },
		"Inf rates":      func(m *Model) { m.rates.animDown, m.rates.animUp = math.Inf(1), math.Inf(-1) },
		"negative rates": func(m *Model) { m.rates.animDown, m.rates.animUp = -5000, -1 },
		"tiny rates":     func(m *Model) { m.rates.animDown, m.rates.animUp = 0.004, 0 },
		"zero window":    func(m *Model) { m.windowSecs = 0 },
		"huge window":    func(m *Model) { m.windowSecs = 1 << 30 },
		"bad interval":   func(m *Model) { m.refreshInterval = 0 },
		"long iface":     func(m *Model) { m.iface.name = long },
		"unicode iface":  func(m *Model) { m.iface.name = "wlän0-日本語-🎉" },
		"no iface":       func(m *Model) { m.iface.name = "" },
		"stale sample":   func(m *Model) { m.rates.lastSample = time.Now().Add(-time.Hour) },
		"future sample":  func(m *Model) { m.rates.lastSample = time.Now().Add(time.Hour) },
		"paused":         func(m *Model) { m.paused = true },
		"bits":           func(m *Model) { m.bitsMode = true },
		"reset armed":    func(m *Model) { m.resetConfirm = true },
		"launch running": func(m *Model) { m.launchDone = false },
		"grid on":        func(m *Model) { m.showGrid = true },
		"scale linear":   func(m *Model) { m.scaleMode = 1 },
		"scale sqrt":     func(m *Model) { m.scaleMode = 2 },
	}

	for name, mutate := range mutators {
		for _, w := range []int{20, 60, 80, 120} {
			for _, h := range []int{3, 10, 24, 40} {
				m := goldenModel(w, h)
				mutate(&m)
				for _, mode := range allModes {
					_, lines := pickViewModeAndContent(m)
					if len(lines) > h {
						t.Errorf("%s %dx%d mode=%v: %d lines exceeds %d",
							name, w, h, mode, len(lines), h)
					}
					for i, line := range lines {
						if got := lipgloss.Width(line); got > w {
							t.Errorf("%s %dx%d mode=%v: line %d is %d cols: %q",
								name, w, h, mode, i, got, truncate(line, 60))
						}
					}
				}
			}
		}
	}
}

// TestOverlaysNeverExceedTerminal covers the modal overlays, which have their
// own box chrome and are the most likely to overflow.
func TestOverlaysNeverExceedTerminal(t *testing.T) {
	overlays := map[string]func(m *Model){
		"help":      func(m *Model) { m.over.help = true },
		"processes": func(m *Model) { m.over.processes = true },
		"themes":    func(m *Model) { m.over.themes = true; m.over.themeIndex = 4 },
		"interface": func(m *Model) { m.over.iface = true },
		"toast":     func(m *Model) { m.over.toast = "a message"; m.over.toastAt = time.Now() },
	}
	for name, set := range overlays {
		for _, w := range widthsUnderTest {
			for _, h := range heightsUnderTest {
				m := goldenModel(w, h)
				set(&m)
				var out string
				switch name {
				case "help":
					out = renderHelp(m)
				case "processes":
					out = renderProcesses(m)
				case "themes":
					out = renderThemes(m)
				case "interface":
					out = renderIfaceDetails(m)
				case "toast":
					// A toast borrows the dashboard status row, so exercise the
					// full View path rather than the raw content lines.
					out = m.View()
				}
				lines := strings.Split(out, "\n")
				if len(lines) > h {
					t.Errorf("%s %dx%d: %d lines exceeds %d", name, w, h, len(lines), h)
				}
				for i, line := range lines {
					if got := lipgloss.Width(line); got > w {
						t.Errorf("%s %dx%d: line %d is %d cols: %q",
							name, w, h, i, got, truncate(line, 60))
					}
				}
			}
		}
	}
}

// TestNoColorAndReducedMotionMustNotPanic covers the two accessibility modes.
func TestNoColorAndReducedMotionMustNotPanic(t *testing.T) {
	for _, noAnim := range []bool{false, true} {
		for _, showGrid := range []bool{false, true} {
			m := goldenModel(80, 24)
			m.noAnim = noAnim
			m.showGrid = showGrid
			m.colorCap = 0 // CapNone
			for _, mode := range allModes {
				out := m.View()
				if strings.Contains(out, "\x1b[38;2;") {
					t.Errorf("noAnim=%v grid=%v mode=%v emitted truecolor",
						noAnim, showGrid, mode)
				}
			}
		}
	}
}

// TestDisplayFilterNeverPanics exercises the d filter across every mode.
func TestDisplayFilterNeverPanics(t *testing.T) {
	for _, f := range []DisplayFilter{DisplayBoth, DisplayDownOnly, DisplayUpOnly} {
		for _, mode := range allModes {
			m := goldenModel(100, 30)
			m.displayFilter = f
			m.viewMode = mode
			if _, lines := pickViewModeAndContent(m); len(lines) > 30 {
				t.Errorf("filter=%v mode=%v: %d lines exceeds 30", f, mode, len(lines))
			}
		}
	}
}

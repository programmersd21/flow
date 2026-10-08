package ui

import (
	"testing"
	"time"

	"github.com/programmersd21/flow/internal/config"
	"github.com/programmersd21/flow/internal/history"
	"github.com/programmersd21/flow/internal/render"
	"github.com/programmersd21/flow/internal/theme"
)

// newTestModel returns a fully populated, deterministic Model for tests, so
// individual tests never hand-build a Model literal and drift out of sync with
// the real fields.
func newTestModel(t testing.TB, w, h int) Model {
	t.Helper()
	theme.SetTheme("default")
	tr := history.NewTracker()
	tr.PeakDown = 4_000_000
	tr.PeakUp = 900_000
	tr.TodayDown = 16_900_000
	tr.TodayUp = 70_300_000
	return Model{
		width:           w,
		height:          h,
		cfg:             testConfig(),
		downHist:        history.New(36000),
		upHist:          history.New(36000),
		tracker:         tr,
		refreshInterval: 100 * time.Millisecond,
		viewMode:        ViewHero,
		windowSecs:      60,
		showGrid:        true,
		noAnim:          true,
		launchDone:      true,
		glyphSet:        render.GlyphBraille,
		colorCap:        render.CapTrue,
		rates: rateState{
			lastSample:     time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC),
			pingLatency:    46 * time.Millisecond,
			animDown:       248_000,
			animUp:         197,
			rollingMaxDown: 4_000_000,
			rollingMaxUp:   900_000,
			samplePulse:    1.0,
		},
		iface:       ifaceState{name: "wlan0"},
		nowOverride: time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC),
	}
}

// testConfig returns a default config without touching the user's filesystem.
func testConfig() config.Config {
	c := config.Defaults()
	c.History = 60
	c.PingTarget = "1.1.1.1"
	return c
}

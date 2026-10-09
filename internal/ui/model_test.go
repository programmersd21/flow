package ui

import (
	"math"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestFormatBpsExt(t *testing.T) {
	tests := []struct {
		bps      float64
		unit     UnitMode
		bits     bool
		expected string
	}{
		{0, UnitAuto, false, "0 B/s"},
		{0, UnitAuto, true, "0 b/s"},
		{1024, UnitAuto, false, "1 KB/s"},
		{1024, UnitAuto, true, "8 Kb/s"},
		{1048576, UnitAuto, false, "1.0 MB/s"},
		{1048576, UnitAuto, true, "8.0 Mb/s"},
		{1073741824, UnitAuto, false, "1.00 GB/s"},
		{1073741824, UnitAuto, true, "8.00 Gb/s"},
		{1024, UnitKB, false, "1.0 KB/s"},
		{1024, UnitKB, true, "8.0 Kb/s"},
		{500, UnitAuto, false, "500 B/s"},
		{500, UnitAuto, true, "4 Kb/s"},
		{0.5, UnitAuto, false, "0 B/s"},
		{1.5, UnitAuto, false, "2 B/s"},
		{-100, UnitAuto, false, "0 B/s"},
		{100 * 1024 * 1024, UnitMB, false, "100.0 MB/s"},
		{100 * 1024 * 1024 * 1024, UnitGB, false, "100.000 GB/s"},
		{1024, UnitKB, false, "1.0 KB/s"},
	}

	for _, tt := range tests {
		actual := FormatBpsExt(tt.bps, tt.unit, tt.bits)
		if actual != tt.expected {
			t.Errorf("FormatBpsExt(%f, %v, %v) = %q; expected %q", tt.bps, tt.unit, tt.bits, actual, tt.expected)
		}
	}
}

func TestFormatBpsExt_EdgeCases(t *testing.T) {
	if got := FormatBpsExt(math.NaN(), UnitAuto, false); got != "0 B/s" {
		t.Errorf("FormatBpsExt(NaN) = %q; want '0 B/s'", got)
	}
	if got := FormatBpsExt(math.Inf(1), UnitAuto, false); got != "0 B/s" {
		t.Errorf("FormatBpsExt(+Inf) = %q; want '0 B/s'", got)
	}
	if got := FormatBpsExt(math.Inf(-1), UnitAuto, false); got != "0 B/s" {
		t.Errorf("FormatBpsExt(-Inf) = %q; want '0 B/s'", got)
	}
}

func TestFormatBpsFixedWidth(t *testing.T) {
	tests := []struct {
		bps  float64
		unit UnitMode
		bits bool
	}{
		{0, UnitAuto, false},
		{50, UnitAuto, true},
		{1024, UnitAuto, false},
		{12345, UnitAuto, true},
		{1000000, UnitAuto, false},
		{100000000, UnitAuto, true},
	}

	for _, tt := range tests {
		actual := FormatBpsFixedWidth(tt.bps, tt.unit, tt.bits)
		if len(actual) != 10 {
			t.Errorf("FormatBpsFixedWidth(%f, %v, %v) length = %d (%q); expected 10", tt.bps, tt.unit, tt.bits, len(actual), actual)
		}
		if tt.bits {
			if !strings.HasSuffix(actual, "b/s") && !strings.HasSuffix(actual, "Kb/s") && !strings.HasSuffix(actual, "Mb/s") && !strings.HasSuffix(actual, "Gb/s") {
				t.Errorf("FormatBpsFixedWidth(%f, %v, true) = %q does not end with bits unit", tt.bps, tt.unit, actual)
			}
		} else {
			if !strings.HasSuffix(actual, "B/s") && !strings.HasSuffix(actual, "KB/s") && !strings.HasSuffix(actual, "MB/s") && !strings.HasSuffix(actual, "GB/s") {
				t.Errorf("FormatBpsFixedWidth(%f, %v, false) = %q does not end with bytes unit", tt.bps, tt.unit, actual)
			}
		}
	}
}

func TestFormatBps_UnitModes(t *testing.T) {
	bps := float64(5 * 1024 * 1024) // 5 MB/s
	if got := FormatBps(bps, UnitKB); got != "5120.0 KB/s" {
		t.Errorf("FormatBps(5MB, KB) = %q; expected 5120.0 KB/s", got)
	}
	if got := FormatBps(bps, UnitMB); got != "5.0 MB/s" {
		t.Errorf("FormatBps(5MB, MB) = %q; expected 5.0 MB/s", got)
	}
	if got := FormatBps(bps, UnitGB); got != "0.005 GB/s" {
		t.Errorf("FormatBps(5MB, GB) = %q; expected 0.005 GB/s", got)
	}
}

func TestFormatBps_Precision(t *testing.T) {
	bps := 1.5 * 1024 * 1024 * 1024
	got := FormatBps(bps, UnitGB)
	if !strings.Contains(got, "1.500") {
		t.Errorf("FormatBps(1.5GB, GB) = %q; expected 1.500 GB/s", got)
	}
}

func TestHelpOverlayKeyFlow(t *testing.T) {
	m := newTestModel(t, 80, 24)
	m.keys = DefaultKeyMap()

	press := func(key string) Model {
		var msg tea.KeyMsg
		switch key {
		case "?":
			msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}}
		case "q":
			msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
		case "m":
			msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}}
		case "esc":
			msg = tea.KeyMsg{Type: tea.KeyEsc}
		default:
			t.Fatalf("unknown key %q", key)
		}
		nm, _ := m.Update(msg)
		return nm.(Model)
	}

	m = press("?")
	if !m.over.help {
		t.Fatal("? should open the help overlay")
	}
	m = press("m")
	if m.viewMode != ViewHero {
		t.Errorf("help overlay must swallow 'm'; viewMode = %v", m.viewMode)
	}
	if !m.over.help {
		t.Error("'m' should not close the help overlay")
	}
	m = press("q")
	if m.over.help {
		t.Error("'q' should close the help overlay, not quit")
	}
	m = press("?")
	m = press("esc")
	if m.over.help {
		t.Error("esc should close the help overlay")
	}
	m = press("?")
	if !m.over.help {
		t.Fatal("? should reopen the help overlay")
	}
	m = press("?")
	if m.over.help {
		t.Error("? should toggle the help overlay closed")
	}
}

func TestPeakPulseFiresAfterReset(t *testing.T) {
	m := newTestModel(t, 80, 24)
	// Reset peaks to 0
	m.tracker.PeakDown = 0
	m.tracker.PeakUp = 0
	m.rates.downPulse = 0

	// Deliver a positive sample; because PeakDown was 0, it must trigger a pulse
	// (previously the `> 0` guard swallowed it).
	sm := sampleMsg{DownBps: 5_000_000, UpBps: 1_000_000, Interface: "eth0", Interval: 1.0}
	nm, _ := m.Update(sm)
	newModel := nm.(Model)

	if newModel.rates.downPulse <= 0 {
		t.Errorf("downPulse = %v after peak from zero; want > 0 (pulse triggered)", newModel.rates.downPulse)
	}
}

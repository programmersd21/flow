package ui

import (
	"math"
	"strings"
	"testing"
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
	// Test NaN and Inf - should not crash
	_ = FormatBpsExt(math.NaN(), UnitAuto, false)
	_ = FormatBpsExt(math.Inf(1), UnitAuto, false)
	_ = FormatBpsExt(math.Inf(-1), UnitAuto, false)
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

package ui

import (
	"strings"
	"testing"
)

func TestSplitHeroNumber(t *testing.T) {
	cases := []struct {
		bps  float64
		bits bool
		num  string
		unit string
	}{
		{0, false, "0", "B/s"},
		{500, false, "500", "B/s"},
		{1024, false, "1.02", "KB/s"},
		{195_000, false, "195", "KB/s"},
		{2_410_000, false, "2.41", "MB/s"},
		{11_200, false, "11.2", "KB/s"},
		{1_000_000_000, false, "1", "GB/s"},
		{1024, true, "8.19", "Kb/s"},
		{-5, false, "0", "B/s"},
	}
	for _, tc := range cases {
		num, unit := splitHeroNumber(tc.bps, tc.bits)
		if num != tc.num || unit != tc.unit {
			t.Errorf("splitHeroNumber(%v, %v) = (%q, %q), want (%q, %q)",
				tc.bps, tc.bits, num, unit, tc.num, tc.unit)
		}
	}
}

func TestSplitHeroNumberNeverExceedsFourChars(t *testing.T) {
	// The hero number must never grow wider than the space budgeted for it.
	for _, bps := range []float64{1, 9.9, 99.9, 999, 12_345, 987_654, 5.5e9} {
		num, _ := splitHeroNumber(bps, false)
		if len(num) > 4 {
			t.Errorf("splitHeroNumber(%v) = %q, want at most 4 chars", bps, num)
		}
	}
}

// TestSplitHeroNumberNoTrailingZeros is a regression test: 4 B/s formatted as
// "4.00" and 10 B/s as "10.0", so the hero showed two extra 5x5 glyphs for
// values needing one or two digits. The trailing-zero strip keeps those out.
func TestSplitHeroNumberNoTrailingZeros(t *testing.T) {
	cases := []struct {
		bps  float64
		want string
	}{
		{0, "0"},
		{4, "4"},
		{40, "40"},
		{400, "400"},
		{4000, "4"},
		{10, "10"},
		{100, "100"},
		{9.99, "9.99"},
		{1_010, "1.01"},
		{11_200, "11.2"},
		{195_000, "195"},
		{88_000, "88"},
		{2_410_000, "2.41"},
	}
	for _, tc := range cases {
		if got, _ := splitHeroNumber(tc.bps, false); got != tc.want {
			t.Errorf("splitHeroNumber(%v) = %q, want %q", tc.bps, got, tc.want)
		}
	}
}

// A one-digit value must not occupy four glyph cells.
func TestSplitHeroNumberGlyphWidthMatchesValue(t *testing.T) {
	for _, bps := range []float64{0, 1, 4, 9, 40, 999} {
		num, _ := splitHeroNumber(bps, false)
		digits := 0
		for _, r := range num {
			if r >= '0' && r <= '9' {
				digits++
			}
		}
		want := 1
		if bps >= 100 {
			want = 3
		} else if bps >= 10 {
			want = 2
		}
		if digits != want {
			t.Errorf("splitHeroNumber(%v) = %q has %d digits, want %d",
				bps, num, digits, want)
		}
	}
}

func TestRenderBigDigitsShape(t *testing.T) {
	rows := renderBigDigits("0")
	if len(rows) != digitSize {
		t.Fatalf("expected %d rows, got %d", digitSize, len(rows))
	}
	// Every row of a digit is the same width (tabular).
	w := len([]rune(rows[0]))
	for i, r := range rows {
		if len([]rune(r)) != w {
			t.Errorf("row %d width = %d, want %d (tabular)", i, len([]rune(r)), w)
		}
	}
	// '0' has a hole in the middle row: two filled columns, gap, two more.
	if !strings.Contains(rows[2], "██ ██") {
		t.Errorf("row 2 of '0' should have a centre hole, got %q", rows[2])
	}
}

func TestRenderBigDigitsAllGlyphs(t *testing.T) {
	for d := '0'; d <= '9'; d++ {
		rows := renderBigDigits(string(d))
		// Every glyph must paint something on every row (no blank glyph).
		for i, r := range rows {
			if !strings.ContainsRune(r, '█') {
				t.Errorf("digit %q row %d is empty: %q", d, i, r)
			}
		}
	}
	// '1' is the narrow one: only the right stroke is filled.
	rows := renderBigDigits("1")
	if strings.ContainsRune(rows[0], '█') && strings.HasPrefix(rows[0], "█") {
		t.Errorf("digit '1' should not fill its left columns, got %q", rows[0])
	}
}

func TestRenderBigDigitsDecimalPoint(t *testing.T) {
	rows := renderBigDigits("2.41")
	// The dot must not create a leading gap on the row above the baseline.
	if strings.TrimSpace(rows[digitSize-2]) == "" {
		t.Errorf("row above baseline should hold digit strokes, got %q", rows[digitSize-2])
	}
	// Baseline row contains the dot block and more content.
	if !strings.Contains(rows[digitSize-1], "█") {
		t.Errorf("baseline row missing blocks: %q", rows[digitSize-1])
	}
}

func TestRenderBigDigitsUnknownCharIsBlank(t *testing.T) {
	rows := renderBigDigits("x")
	for i, r := range rows {
		if strings.TrimSpace(r) != "" {
			t.Errorf("unknown char row %d should be blank, got %q", i, r)
		}
	}
}

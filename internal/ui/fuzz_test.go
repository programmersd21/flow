package ui

import (
	"math"
	"strings"
	"testing"
	"unicode/utf8"
)

// FuzzHeroLayout targets the layout arithmetic, where a bad width or height
// could produce a negative slice bound and panic the renderer.
func FuzzHeroLayout(f *testing.F) {
	f.Add(80, 24, 248_000.0, 197.0)
	f.Add(1, 1, 0.0, 0.0)
	f.Add(400, 200, 9.7e9, 1e-3)
	f.Add(0, 0, -5.0, math.NaN())

	f.Fuzz(func(t *testing.T, w, h int, down, up float64) {
		if w > 4000 || h > 4000 || w < -4000 || h < -4000 {
			t.Skip("out of the interesting range")
		}
		// Never trust a fuzzed float as an index or dimension source.
		if math.IsNaN(down) || math.IsInf(down, 0) || math.IsNaN(up) || math.IsInf(up, 0) {
			return
		}
		m := goldenModel(maxInt(w, 1), maxInt(h, 1))
		m.rates.animDown = math.Abs(down)
		m.rates.animUp = math.Abs(up)

		for _, mode := range []ViewMode{ViewHero, ViewCompact, ViewMini, ViewTiny} {
			// Must not panic for any input, valid or not.
			_, _ = pickViewModeAndContent(m)
			_ = dashboardContentLines(m, mode)
		}
		_ = renderTiny(m)
		_ = renderHelp(m)
		_ = renderThemes(m)
		_ = renderIfaceDetails(m)
		_ = renderProcesses(m)
	})
}

// FuzzSplitHeroNumber checks the number formatter never panics and never emits
// a string the digit renderer cannot handle.
func FuzzSplitHeroNumber(f *testing.F) {
	f.Add(0.0, false)
	f.Add(1e18, true)
	f.Add(-1.0, false)
	f.Add(0.004, false)

	f.Fuzz(func(t *testing.T, bps float64, bits bool) {
		if math.IsNaN(bps) || math.IsInf(bps, 0) {
			return
		}
		num, unit := splitHeroNumber(bps, bits)
		if num == "" {
			t.Fatal("splitHeroNumber returned an empty number")
		}
		if unit == "" {
			t.Fatal("splitHeroNumber returned an empty unit")
		}
		if len(num) > 4 {
			t.Errorf("number %q is %d chars, the layout budgets for at most 4", num, len(num))
		}
		// The digit renderer must handle every character it is given.
		rows := renderBigDigits(num)
		if len(rows) != digitSize {
			t.Fatalf("expected %d rows, got %d", digitSize, len(rows))
		}
		for i, c := range rows {
			for _, r := range c {
				if !strings.ContainsRune(" ▀▄█", r) {
					t.Errorf("row %d contains unexpected rune %q", i, r)
				}
			}
		}
	})
}

// FuzzFormatBytes covers the unit formatter used by both the UI and --format.
func FuzzFormatBytes(f *testing.F) {
	f.Add(0.0)
	f.Add(1024.0)
	f.Add(1e15)
	f.Add(-5.0)

	f.Fuzz(func(t *testing.T, b float64) {
		if math.IsNaN(b) || math.IsInf(b, 0) {
			return
		}
		s := formatBytes(b)
		if s == "" {
			t.Fatal("formatBytes returned an empty string")
		}
	})
}

// FuzzTruncate checks the shared string truncator cannot slice mid-rune.
func FuzzTruncate(f *testing.F) {
	f.Add("hello", 3)
	f.Add("", 0)
	f.Add("héllo wörld", 5)
	f.Add("日本語テキスト", 4)

	f.Fuzz(func(t *testing.T, s string, n int) {
		// Deliberately no lower bound on n: a non-positive budget used to
		// panic with a slice-bounds error.
		if n > 4096 {
			t.Skip("out of range")
		}
		got := truncate(s, n)
		if n >= 1 && len([]rune(got)) > n {
			t.Errorf("truncate(%q,%d) = %q exceeds the budget", s, n, got)
		}
		// Valid input must stay valid: truncate must never split a rune.
		if isValidUTF8(s) && !isValidUTF8(got) {
			t.Errorf("truncate split a rune: %q -> %q", s, got)
		}
	})
}

// isValidUTF8 reports whether s decodes cleanly and every rune survives a
// round trip, which is what "did not slice mid-rune" means in practice.
func isValidUTF8(s string) bool {
	return utf8.ValidString(s) && strings.ToValidUTF8(s, "�") == s
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

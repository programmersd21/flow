package ui

import (
	"fmt"
	"math"
	"strings"
)

// Big block digits: 5x5 bitmap font rendered with full blocks.
//
// Technique adapted from timr-tui (MIT, github.com/sectore/timr-tui): each
// digit is a 5x5 grid of on/off pixels, and every "on" pixel is one full
// terminal cell. Two-pixel-thick strokes plus solid fills is what makes this
// legible at a glance — thin 3-row zigzag fonts blur into each other, while
// chunky 5-row glyphs read instantly.
//
// Layout per glyph: 5 columns + 1 column of spacing (tabular, never jitters).
// The decimal point is a narrow 2-wide block on the bottom row.
const digitSize = 5

// digitPatterns maps 0-9 to 25 on/off pixels, row by row.
var digitPatterns = map[rune][digitSize * digitSize]uint8{
	'0': {
		1, 1, 1, 1, 1,
		1, 1, 0, 1, 1,
		1, 1, 0, 1, 1,
		1, 1, 0, 1, 1,
		1, 1, 1, 1, 1,
	},
	'1': {
		0, 0, 0, 1, 1,
		0, 0, 0, 1, 1,
		0, 0, 0, 1, 1,
		0, 0, 0, 1, 1,
		0, 0, 0, 1, 1,
	},
	'2': {
		1, 1, 1, 1, 1,
		0, 0, 0, 1, 1,
		1, 1, 1, 1, 1,
		1, 1, 0, 0, 0,
		1, 1, 1, 1, 1,
	},
	'3': {
		1, 1, 1, 1, 1,
		0, 0, 0, 1, 1,
		1, 1, 1, 1, 1,
		0, 0, 0, 1, 1,
		1, 1, 1, 1, 1,
	},
	'4': {
		1, 1, 0, 1, 1,
		1, 1, 0, 1, 1,
		1, 1, 1, 1, 1,
		0, 0, 0, 1, 1,
		0, 0, 0, 1, 1,
	},
	'5': {
		1, 1, 1, 1, 1,
		1, 1, 0, 0, 0,
		1, 1, 1, 1, 1,
		0, 0, 0, 1, 1,
		1, 1, 1, 1, 1,
	},
	'6': {
		1, 1, 1, 1, 1,
		1, 1, 0, 0, 0,
		1, 1, 1, 1, 1,
		1, 1, 0, 1, 1,
		1, 1, 1, 1, 1,
	},
	'7': {
		1, 1, 1, 1, 1,
		0, 0, 0, 1, 1,
		0, 0, 0, 1, 1,
		0, 0, 0, 1, 1,
		0, 0, 0, 1, 1,
	},
	'8': {
		1, 1, 1, 1, 1,
		1, 1, 0, 1, 1,
		1, 1, 1, 1, 1,
		1, 1, 0, 1, 1,
		1, 1, 1, 1, 1,
	},
	'9': {
		1, 1, 1, 1, 1,
		1, 1, 0, 1, 1,
		1, 1, 1, 1, 1,
		0, 0, 0, 1, 1,
		1, 1, 1, 1, 1,
	},
	' ': {
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
	},
}

// Note: the hero view calls splitHeroNumber + renderBigDigits directly (rather
// than a combined helper) so it can style the arrow, digits, unit, and caption
// in their own tiers. There is a single digit implementation in this file.

// splitHeroNumber formats a rate for big-digit display: 3 significant digits,
// at most 4 characters of number ("2.41", "195", "11.2", "0"), plus the unit.
func splitHeroNumber(val float64, bits bool) (string, string) {
	if val < 0 || math.IsNaN(val) || math.IsInf(val, 0) {
		val = 0
	}
	if bits {
		val *= 8
	}

	units := []string{"B/s", "KB/s", "MB/s", "GB/s", "TB/s", "PB/s"}
	if bits {
		units = []string{"b/s", "Kb/s", "Mb/s", "Gb/s", "Tb/s", "Pb/s"}
	}

	ui := 0
	for val >= 1000 && ui < len(units)-1 {
		val /= 1000
		ui++
	}
	// Saturate at the largest unit. Without this, an implausibly large rate
	// (or a fuzzed one) produced a ten-character number that broke the four
	// character budget the hero layout is built around, pushing the digits out
	// of their column.
	if val > 9999 {
		val = 9999
	}

	// Precision follows magnitude, then trailing zeros are stripped. Without
	// the strip, 4 B/s became "4.00" and 10 B/s became "10.0" — four
	// characters of glyphs for a value that needs one or two, which read as
	// extra broken blocks next to the digits.
	var num string
	switch {
	case val >= 100:
		num = fmt.Sprintf("%.0f", val)
	case val >= 10:
		num = fmt.Sprintf("%.1f", val)
	case val > 0.05:
		num = fmt.Sprintf("%.2f", val)
	default:
		num = "0"
	}
	if num != "0" {
		if strings.Contains(num, ".") {
			num = strings.TrimRight(num, "0")
			num = strings.TrimRight(num, ".")
		}
		if num == "" {
			num = "0"
		}
	}
	if len(num) > 4 {
		num = fmt.Sprintf("%.0f", val)
	}
	return num, units[ui]
}

// renderBigDigits renders a pre-formatted number into 5 plain rows.
// "On" pixels are full blocks, "off" pixels are spaces. The decimal point is
// a 2-wide block hugging the baseline — narrow enough to read as a dot, wide
// enough to see at 5-row scale.
func renderBigDigits(num string) [digitSize]string {
	var rows [digitSize]strings.Builder
	for i, ch := range num {
		if i > 0 {
			for r := 0; r < digitSize; r++ {
				rows[r].WriteByte(' ')
			}
		}
		if ch == '.' {
			for r := 0; r < digitSize-1; r++ {
				rows[r].WriteString("  ")
			}
			rows[digitSize-1].WriteString("██")
			continue
		}
		pattern, ok := digitPatterns[ch]
		if !ok {
			pattern = digitPatterns[' ']
		}
		for r := 0; r < digitSize; r++ {
			for c := 0; c < digitSize; c++ {
				if pattern[r*digitSize+c] == 1 {
					rows[r].WriteString("█")
				} else {
					rows[r].WriteByte(' ')
				}
			}
		}
	}
	return [digitSize]string{rows[0].String(), rows[1].String(), rows[2].String(), rows[3].String(), rows[4].String()}
}

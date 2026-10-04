package render

import (
	"math"
	"os"
	"strings"
)

// GlyphSet selects how the waveform is drawn.
type GlyphSet int

const (
	// GlyphBraille uses the U+2800 block: 2x4 dots per cell, the highest
	// resolution available in a terminal.
	GlyphBraille GlyphSet = iota
	// GlyphBlocks uses the block ramp for 8 vertical levels per cell.
	GlyphBlocks
	// GlyphAscii uses a text ramp for legacy consoles and fonts.
	GlyphAscii
)

// ParseGlyphSet maps a config value to a glyph set. Unknown values and "auto"
// resolve to braille; the caller decides the real fallback for dumb terms.
func ParseGlyphSet(s string) GlyphSet {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "blocks", "block":
		return GlyphBlocks
	case "ascii":
		return GlyphAscii
	default:
		return GlyphBraille
	}
}

var blockRamp = []rune{' ', '▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}
var asciiRamp = []rune{' ', '.', ':', '-', '=', '#', '@'}

// RenderWaveform draws normalized samples into width x height cells using the
// requested glyph set, returning plain (uncolored) rows. This is the
// degradation path for terminals or fonts without braille support: each cell
// holds one glyph chosen by the sample height within that row's band.
func RenderWaveform(samples []float64, width, height int, glyphs GlyphSet, inverted bool) []string {
	if width <= 0 || height <= 0 || len(samples) == 0 {
		return nil
	}
	rows := make([]string, height)

	for r := 0; r < height; r++ {
		var sb strings.Builder
		sb.Grow(width)
		var lo, hi float64
		if !inverted {
			hi = 1 - float64(r)/float64(height) // download fills up
			lo = 1 - float64(r+1)/float64(height)
		} else {
			lo = float64(r) / float64(height) // upload fills down
			hi = float64(r+1) / float64(height)
		}
		if lo < 0 {
			lo = 0
		}
		if hi > 1 {
			hi = 1
		}

		for c := 0; c < width; c++ {
			v := CatmullRomInterpolate(samples, sampleIndex(c, width, len(samples)))
			sb.WriteRune(cellGlyph(v, lo, hi, glyphs))
		}
		rows[r] = sb.String()
	}
	return rows
}

func sampleIndex(col, width, n int) float64 {
	if width <= 1 {
		return 0
	}
	return float64(col) / float64(width-1) * float64(n-1)
}

// cellGlyph picks the ramp glyph for a value inside a row band.
func cellGlyph(v, lo, hi float64, glyphs GlyphSet) rune {
	if v < lo {
		return ' '
	}
	mid := lo + (hi-lo)/2 // band midpoint: stable choice, no flicker
	switch glyphs {
	case GlyphAscii:
		idx := int(mid * float64(len(asciiRamp)-1))
		if idx < 1 {
			idx = 1
		}
		if idx >= len(asciiRamp) {
			idx = len(asciiRamp) - 1
		}
		return asciiRamp[idx]
	case GlyphBlocks:
		idx := int(math.Round(mid * float64(len(blockRamp)-1)))
		if idx < 1 {
			idx = 1
		}
		if idx >= len(blockRamp) {
			idx = len(blockRamp) - 1
		}
		return blockRamp[idx]
	default:
		if v >= hi-1e-9 {
			return '█'
		}
		if v >= mid {
			return '▄'
		}
		return '▀'
	}
}

// ColorCap describes what the terminal can display, so gradients degrade
// instead of emitting escapes the terminal will mangle.
type ColorCap int

const (
	CapNone ColorCap = iota // NO_COLOR: no escape codes at all
	Cap16                   // 16 colors: quantized
	Cap256                  // 256 colors: quantized
	CapTrue                 // 24-bit truecolor
)

// DetectColorCap inspects the environment. NO_COLOR wins over everything,
// then COLORTERM=truecolor/24bit, then a 256color TERM.
func DetectColorCap(noColorFlag bool) ColorCap {
	if noColorFlag {
		return CapNone
	}
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		return CapNone
	}
	switch strings.ToLower(os.Getenv("COLORTERM")) {
	case "truecolor", "24bit":
		return CapTrue
	}
	if strings.Contains(strings.ToLower(os.Getenv("TERM")), "256color") {
		return Cap256
	}
	if os.Getenv("TERM") == "" {
		return CapNone
	}
	return Cap16
}

// GradientSteps is how many distinct gradient steps a capability can show.
func GradientSteps(c ColorCap) int {
	switch c {
	case CapTrue:
		return 32
	case Cap256:
		return 8
	default:
		return 1
	}
}

// AllowsGradient reports whether gradients are meaningful at all.
func AllowsGradient(c ColorCap) bool {
	return GradientSteps(c) > 1
}

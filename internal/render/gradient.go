package render

import (
	"fmt"
	"math"
)

// RGB is an 8-bit color triplet.
type RGB struct {
	R, G, B uint8
}

// Hex returns the #rrggbb representation.
func (c RGB) Hex() string {
	return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
}

// LUT is a precomputed gradient lookup table.
//
// Exactly one of Steps and Palette is used. Steps holds interpolated RGB
// values. Palette holds literal color strings — terminal palette indices like
// "4" for ANSI themes — which must reach the canvas untouched: converting an
// index to RGB first destroys it (there is no RGB value to convert), which is
// what painted the whole waveform black on the `ansi` theme.
type LUT struct {
	Steps   []RGB
	Palette []string
}

// NewGradientLUT builds an N-step table interpolating start -> end in linear
// RGB. N is clamped to a minimum of 2.
func NewGradientLUT(start, end RGB, steps int) *LUT {
	if steps < 2 {
		steps = 2
	}
	table := make([]RGB, steps)
	for i := range table {
		t := float64(i) / float64(steps-1)
		// Promote to float64 before subtracting: end.R-start.R is a uint8
		// subtraction that wraps when end < start, which snapped a gradient
		// from a bright stop back to a dark one at the midpoint.
		table[i] = RGB{
			R: uint8(math.Round(float64(start.R) + (float64(end.R)-float64(start.R))*t)),
			G: uint8(math.Round(float64(start.G) + (float64(end.G)-float64(start.G))*t)),
			B: uint8(math.Round(float64(start.B) + (float64(end.B)-float64(start.B))*t)),
		}
	}
	return &LUT{Steps: table}
}

// Empty reports whether the LUT holds no colors at all.
func (lut *LUT) Empty() bool {
	return len(lut.Steps) == 0 && len(lut.Palette) == 0
}

// Sample returns the table color at fraction t in [0,1].
func (lut *LUT) Sample(t float64) RGB {
	if len(lut.Steps) == 0 {
		return RGB{}
	}
	return lut.Steps[lut.index(t)]
}

// SampleColor returns the color at fraction t in [0,1] in the form the canvas
// emitters accept: the literal palette index for a palette LUT, otherwise a
// "#rrggbb" string. Callers on the render path must use this rather than
// Sample(...).Hex(), which is meaningless for palette entries.
func (lut *LUT) SampleColor(t float64) string {
	if len(lut.Palette) > 0 {
		return lut.Palette[lut.index(t)]
	}
	return lut.Sample(t).Hex()
}

// index maps t in [0,1] to a step, clamping NaN and out-of-range values.
func (lut *LUT) index(t float64) int {
	n := len(lut.Steps)
	if len(lut.Palette) > 0 {
		n = len(lut.Palette)
	}
	if n == 0 {
		return 0
	}
	if t <= 0 || math.IsNaN(t) {
		return 0
	}
	if t >= 1 {
		return n - 1
	}
	idx := int(t * float64(n-1))
	if idx >= n {
		idx = n - 1
	}
	return idx
}

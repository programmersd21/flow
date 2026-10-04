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
type LUT struct {
	Steps []RGB
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
		table[i] = RGB{
			R: uint8(math.Round(float64(start.R) + float64(end.R-start.R)*t)),
			G: uint8(math.Round(float64(start.G) + float64(end.G-start.G)*t)),
			B: uint8(math.Round(float64(start.B) + float64(end.B-start.B)*t)),
		}
	}
	return &LUT{Steps: table}
}

// Sample returns the table color at fraction t in [0,1].
func (lut *LUT) Sample(t float64) RGB {
	if len(lut.Steps) == 0 {
		return RGB{}
	}
	if t <= 0 || math.IsNaN(t) {
		return lut.Steps[0]
	}
	if t >= 1 {
		return lut.Steps[len(lut.Steps)-1]
	}
	idx := int(t * float64(len(lut.Steps)-1))
	if idx >= len(lut.Steps) {
		idx = len(lut.Steps) - 1
	}
	return lut.Steps[idx]
}

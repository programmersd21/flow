package render

import (
	"math"
)

// CatmullRomInterpolate interpolates samples at fractional index t in
// [0, len(samples)-1]. The result is clamped to [0,1] so the crest never
// overshoots below the baseline.
func CatmullRomInterpolate(samples []float64, t float64) float64 {
	n := len(samples)
	if n == 0 {
		return 0
	}
	if n == 1 || t <= 0 {
		return samples[0]
	}
	if t >= float64(n-1) {
		return samples[n-1]
	}

	i := int(math.Floor(t))
	u := t - float64(i)

	p0 := samples[max(0, i-1)]
	p1 := samples[i]
	p2 := samples[min(n-1, i+1)]
	p3 := samples[min(n-1, i+2)]

	val := 0.5 * ((2 * p1) +
		(-p0+p2)*u +
		(2*p0-5*p1+4*p2-p3)*u*u +
		(-p0+3*p1-3*p2+p3)*u*u*u)

	if val < 0 || math.IsNaN(val) {
		return 0
	}
	if val > 1 {
		return 1
	}
	return val
}

// DrawAreaFill paints a solid filled waveform into the canvas (dots only, no
// colors; call ColorRows afterwards). samples are normalized to [0,1].
// If inverted is false (download half): baseline is the bottom dot row, fills
// upward. If inverted is true (upload half): baseline is the top dot row,
// fills downward.
func DrawAreaFill(c *Canvas, samples []float64, inverted bool) {
	if c.W <= 0 || c.H <= 0 || len(samples) == 0 {
		return
	}
	totalCols := 2 * c.W
	totalRows := 4 * c.H

	for col := 0; col < totalCols; col++ {
		t := float64(col) / float64(totalCols-1) * float64(len(samples)-1)
		v := CatmullRomInterpolate(samples, t)
		fillDots := int(math.Round(v * float64(totalRows)))
		if fillDots > totalRows {
			fillDots = totalRows
		}

		for row := 0; row < fillDots; row++ {
			var y int
			if !inverted {
				y = (totalRows - 1) - row
			} else {
				y = row
			}
			c.Set(col, y)
		}
	}
}

// ColorRows assigns per-row band colors: a uniform horizontal gradient from
// the axis (dim) to the outer edge (bright). Every filled cell in the same
// row shares one color, which keeps the waveform smooth — per-cell coloring
// produced vertical striping artifacts.
func ColorRows(c *Canvas, lut *LUT, inverted bool) {
	if lut == nil || len(lut.Steps) == 0 || c.H <= 0 {
		return
	}
	denom := c.H - 1
	if denom < 1 {
		denom = 1
	}
	for cy := 0; cy < c.H; cy++ {
		var frac float64
		if !inverted {
			frac = float64(c.H-1-cy) / float64(denom) // axis at bottom
		} else {
			frac = float64(cy) / float64(denom) // axis at top
		}
		// Keep the axis-adjacent band readable, not invisible.
		col := lut.Sample(0.2 + 0.8*frac).Hex()
		for cx := 0; cx < c.W; cx++ {
			if c.Cell(cx, cy) != ' ' {
				c.SetColor(cx, cy, col)
			}
		}
	}
}

// Smooth applies an exponential moving average for display only.
// factor in [0,1]: higher = smoother/laggier. Raw data is untouched.
func Smooth(samples []float64, factor float64) []float64 {
	if len(samples) == 0 || factor <= 0 {
		return samples
	}
	if factor > 1 {
		factor = 1
	}
	out := make([]float64, len(samples))
	prev := samples[0]
	for i, v := range samples {
		prev = prev*factor + v*(1-factor)
		out[i] = prev
	}
	return out
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

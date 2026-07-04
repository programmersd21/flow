// internal/sparkline/sparkline.go — smooth, high-resolution Braille & block sparkline rendering.
package sparkline

import (
	"math"
)

// Legacy block set (using space U+2003 or U+2581 ' ' instead of broken U+2581 typo)
var blocks = []rune{' ', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

// Render converts samples to a smooth block sparkline of exactly `width` chars.
func Render(samples []float64, width int, smoothing float64) string {
	if width <= 0 {
		return ""
	}

	src := samples
	if len(src) > width {
		src = src[len(src)-width:]
	}

	if len(src) == 0 {
		return string(repeatRune(blocks[0], width))
	}

	// Find maximum for normalization
	max := 0.0
	for _, v := range src {
		if v > max {
			max = v
		}
	}
	if max < 1 {
		max = 1
	}

	runes := make([]rune, width)
	pad := width - len(src)

	// Left-pad with lowest block (space)
	for i := 0; i < pad; i++ {
		runes[i] = blocks[0]
	}

	// Render samples with optional smoothing
	if smoothing > 0 && len(src) > 3 {
		smoothed := make([]float64, len(src))
		smoothed[0] = src[0]
		for i := 1; i < len(src)-1; i++ {
			smoothed[i] = (src[i-1] + src[i]*2 + src[i+1]) / 4.0
		}
		smoothed[len(src)-1] = src[len(src)-1]
		src = smoothed
	}

	for i, v := range src {
		ratio := v / max
		idx := valueToBlock(ratio)
		runes[pad+i] = blocks[idx]
	}

	return string(runes)
}

// RenderBraille renders a multi-row high-resolution Braille area graph.
// It supports sub-pixel horizontal scrolling via `frac` (0.0 to 1.0).
func RenderBraille(samples []float64, width int, height int, maxVal float64, frac float64) []string {
	if width <= 0 || height <= 0 {
		return nil
	}

	numDotsX := width * 2
	numDotsY := height * 4

	// Prepare grid of braille cells
	grid := make([][]rune, height)
	for r := 0; r < height; r++ {
		grid[r] = make([]rune, width)
		for c := 0; c < width; c++ {
			grid[r][c] = 0x2800 // Empty braille cell
		}
	}

	if len(samples) == 0 {
		lines := make([]string, height)
		for r := 0; r < height; r++ {
			lines[r] = string(grid[r])
		}
		return lines
	}

	// Find dynamic max if not supplied
	if maxVal <= 0 {
		for _, v := range samples {
			if v > maxVal {
				maxVal = v
			}
		}
	}
	if maxVal < 1 {
		maxVal = 1
	}

	// Clamp frac to valid range to prevent infinite/NaN values
	if math.IsNaN(frac) || math.IsInf(frac, 0) {
		frac = 0
	}
	if frac < 0 {
		frac = 0
	}
	if frac > 1 {
		frac = 1
	}

	// For each dot column, sample the history with fractional scroll offset
	dotsY := make([]float64, numDotsX)
	for x := 0; x < numDotsX; x++ {
		// As time passes (frac goes 0->1), the wave should scroll left.
		// Therefore, we shift the lookup index forward by `frac`.
		idx := float64(len(samples)-(numDotsX)) + float64(x) + frac
		dotsY[x] = sampleAt(samples, idx)
	}

	// Apply light smoothing to the dots for water-like flow
	smoothed := make([]float64, numDotsX)
	for i := 0; i < numDotsX; i++ {
		sum := 0.0
		weightSum := 0.0
		for offset := -2; offset <= 2; offset++ {
			idx := i + offset
			if idx >= 0 && idx < numDotsX {
				w := 1.0
				switch offset {
				case 0:
					w = 3.0
				case -1, 1:
					w = 2.0
				}
				sum += dotsY[idx] * w
				weightSum += w
			}
		}
		// Prevent division by zero
		if weightSum > 0 {
			smoothed[i] = sum / weightSum
		} else {
			smoothed[i] = 0
		}
	}
	dotsY = smoothed

	// Map each vertical value to the dot grid
	for x := 0; x < numDotsX; x++ {
		val := dotsY[x]
		ratio := val / maxVal
		if ratio < 0 {
			ratio = 0
		}
		if ratio > 1 {
			ratio = 1
		}

		// Apply soft quadratic easing for elegant peak shape
		ratio = easeOutQuad(ratio)

		hDots := ratio * float64(numDotsY)
		col := x / 2
		dx := x % 2

		// Validate column index
		if col < 0 || col >= width {
			continue
		}

		for yDot := 0; yDot < numDotsY; yDot++ {
			if float64(yDot) < hDots {
				row := height - 1 - (yDot / 4)
				dy := 3 - (yDot % 4)

				// Validate row index
				if row < 0 || row >= height {
					continue
				}

				var mask int
				if dx == 0 {
					switch dy {
					case 0:
						mask = 1 << 0
					case 1:
						mask = 1 << 1
					case 2:
						mask = 1 << 2
					case 3:
						mask = 1 << 6
					}
				} else {
					switch dy {
					case 0:
						mask = 1 << 3
					case 1:
						mask = 1 << 4
					case 2:
						mask = 1 << 5
					case 3:
						mask = 1 << 7
					}
				}

				grid[row][col] = rune(int(grid[row][col]) | mask)
			}
		}
	}

	lines := make([]string, height)
	for r := 0; r < height; r++ {
		lines[r] = string(grid[r])
	}
	return lines
}

func sampleAt(samples []float64, idx float64) float64 {
	n := len(samples)
	if n == 0 {
		return 0
	}
	// Clamp idx to prevent NaN/Inf propagation
	if math.IsNaN(idx) {
		idx = 0
	}
	if math.IsInf(idx, 1) { // positive infinity
		return samples[n-1]
	}
	if math.IsInf(idx, -1) { // negative infinity
		return 0
	}
	// Return 0 for positions before available data (prevents graph fill during startup)
	if idx < 0 {
		return 0
	}
	if idx == 0 {
		return samples[0]
	}
	if idx >= float64(n-1) {
		return samples[n-1]
	}
	i := int(math.Floor(idx))
	// Additional bounds check to prevent panic
	if i < 0 || i >= n-1 {
		if i >= n-1 {
			return samples[n-1]
		}
		return samples[0]
	}
	f := idx - float64(i)
	return samples[i]*(1.0-f) + samples[i+1]*f
}

func valueToBlock(ratio float64) int {
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}

	ratio = easeOutQuad(ratio)

	idx := int(math.Round(ratio * float64(len(blocks)-1)))
	if idx < 0 {
		idx = 0
	}
	if idx >= len(blocks) {
		idx = len(blocks) - 1
	}
	return idx
}

func easeOutQuad(t float64) float64 {
	return t * (2.0 - t)
}

func Slope(samples []float64, n int) float64 {
	if n < 2 || len(samples) < n {
		return 0
	}
	window := samples[len(samples)-n:]
	var sumX, sumY, sumXY, sumX2 float64
	for i, v := range window {
		x := float64(i)
		sumX += x
		sumY += v
		sumXY += x * v
		sumX2 += x * x
	}
	k := float64(n)
	denom := k*sumX2 - sumX*sumX
	if math.Abs(denom) < 1e-9 {
		return 0
	}
	return (k*sumXY - sumX*sumY) / denom
}

func VelocityGlyph(samples []float64, n int) string {
	if len(samples) < 2 {
		return "→"
	}
	s := Slope(samples, n)
	cur := samples[len(samples)-1]
	if cur < 1 {
		return "→"
	}
	threshold := cur * 0.05
	switch {
	case s > threshold:
		return "↗"
	case s < -threshold:
		return "↘"
	default:
		return "→"
	}
}

func repeatRune(r rune, count int) []rune {
	result := make([]rune, count)
	for i := range result {
		result[i] = r
	}
	return result
}

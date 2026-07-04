package sparkline

import (
	"math"
	"testing"
)

func TestRenderBraille_EmptySamples(t *testing.T) {
	lines := RenderBraille([]float64{}, 10, 4, 0, 0)
	if len(lines) != 4 {
		t.Errorf("Expected 4 lines, got %d", len(lines))
	}
	for _, line := range lines {
		runeCount := len([]rune(line))
		if runeCount != 10 {
			t.Errorf("Expected line width of 10 runes, got %d", runeCount)
		}
	}
}

func TestRenderBraille_InvalidFrac(t *testing.T) {
	samples := []float64{1.0, 2.0, 3.0, 4.0, 5.0}

	// Test with NaN frac - should not panic
	lines := RenderBraille(samples, 10, 4, 10, math.NaN())
	if len(lines) != 4 {
		t.Errorf("Expected 4 lines with NaN frac, got %d", len(lines))
	}

	// Test with infinite frac - should not panic
	lines = RenderBraille(samples, 10, 4, 10, math.Inf(1))
	if len(lines) != 4 {
		t.Errorf("Expected 4 lines with Inf frac, got %d", len(lines))
	}

	// Test with negative frac - should not panic
	lines = RenderBraille(samples, 10, 4, 10, -5.0)
	if len(lines) != 4 {
		t.Errorf("Expected 4 lines with negative frac, got %d", len(lines))
	}

	// Test with large frac - should not panic
	lines = RenderBraille(samples, 10, 4, 10, 100.0)
	if len(lines) != 4 {
		t.Errorf("Expected 4 lines with large frac, got %d", len(lines))
	}
}

func TestRenderBraille_SmallWidth(t *testing.T) {
	samples := []float64{1.0, 2.0, 3.0}

	// Test with width=1
	lines := RenderBraille(samples, 1, 4, 10, 0.5)
	if len(lines) != 4 {
		t.Errorf("Expected 4 lines, got %d", len(lines))
	}

	// Test with width=0 - should return nil
	lines = RenderBraille(samples, 0, 4, 10, 0.5)
	if lines != nil {
		t.Errorf("Expected nil for width=0, got %v", lines)
	}

	// Test with height=0 - should return nil
	lines = RenderBraille(samples, 10, 0, 10, 0.5)
	if lines != nil {
		t.Errorf("Expected nil for height=0, got %v", lines)
	}
}

func TestSampleAt_EdgeCases(t *testing.T) {
	samples := []float64{1.0, 2.0, 3.0, 4.0, 5.0}

	// Test negative index
	val := sampleAt(samples, -1.0)
	if val != 0 {
		t.Errorf("Expected 0 for negative index, got %f", val)
	}

	// Test beyond end
	val = sampleAt(samples, 10.0)
	if val != 5.0 {
		t.Errorf("Expected 5.0 for index beyond end, got %f", val)
	}

	// Test NaN index
	val = sampleAt(samples, math.NaN())
	if val != 1.0 {
		t.Errorf("Expected 1.0 for NaN index, got %f", val)
	}

	// Test Inf index
	val = sampleAt(samples, math.Inf(1))
	if val != 5.0 {
		t.Errorf("Expected 5.0 for Inf index, got %f", val)
	}

	// Test empty samples
	val = sampleAt([]float64{}, 0.0)
	if val != 0 {
		t.Errorf("Expected 0 for empty samples, got %f", val)
	}
}

func TestSampleAt_Interpolation(t *testing.T) {
	samples := []float64{0.0, 10.0, 20.0}

	// Test interpolation at midpoint
	val := sampleAt(samples, 0.5)
	if val != 5.0 {
		t.Errorf("Expected 5.0 for interpolation at 0.5, got %f", val)
	}

	// Test interpolation at 1.5
	val = sampleAt(samples, 1.5)
	if val != 15.0 {
		t.Errorf("Expected 15.0 for interpolation at 1.5, got %f", val)
	}
}

func TestRenderBraille_NormalCase(t *testing.T) {
	samples := make([]float64, 50)
	for i := range samples {
		samples[i] = float64(i)
	}

	lines := RenderBraille(samples, 20, 4, 50, 0.5)
	if len(lines) != 4 {
		t.Errorf("Expected 4 lines, got %d", len(lines))
	}

	for i, line := range lines {
		runeCount := len([]rune(line))
		if runeCount != 20 {
			t.Errorf("Line %d: expected width 20 runes, got %d", i, runeCount)
		}
	}
}

func TestSlope_EdgeCases(t *testing.T) {
	// Test with insufficient samples
	slope := Slope([]float64{1.0}, 2)
	if slope != 0 {
		t.Errorf("Expected 0 slope for single sample, got %f", slope)
	}

	// Test with empty samples
	slope = Slope([]float64{}, 5)
	if slope != 0 {
		t.Errorf("Expected 0 slope for empty samples, got %f", slope)
	}

	// Test with flat line
	slope = Slope([]float64{5.0, 5.0, 5.0, 5.0}, 4)
	if math.Abs(slope) > 0.0001 {
		t.Errorf("Expected near-zero slope for flat line, got %f", slope)
	}
}

func TestVelocityGlyph(t *testing.T) {
	// Test rising
	rising := []float64{1.0, 2.0, 3.0, 4.0, 5.0}
	glyph := VelocityGlyph(rising, 5)
	if glyph != "↗" {
		t.Errorf("Expected ↗ for rising values, got %s", glyph)
	}

	// Test falling
	falling := []float64{5.0, 4.0, 3.0, 2.0, 1.0}
	glyph = VelocityGlyph(falling, 5)
	if glyph != "↘" {
		t.Errorf("Expected ↘ for falling values, got %s", glyph)
	}

	// Test flat
	flat := []float64{3.0, 3.0, 3.0, 3.0}
	glyph = VelocityGlyph(flat, 4)
	if glyph != "→" {
		t.Errorf("Expected → for flat values, got %s", glyph)
	}

	// Test insufficient samples
	glyph = VelocityGlyph([]float64{1.0}, 5)
	if glyph != "→" {
		t.Errorf("Expected → for insufficient samples, got %s", glyph)
	}
}

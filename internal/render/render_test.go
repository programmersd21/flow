package render

import (
	"strings"
	"testing"
)

func TestCanvasBrailleEncoding(t *testing.T) {
	c := NewCanvas(2, 1)
	// dot1 (0,0) -> 0x01, dot4 (1,0) -> 0x08, dot8 (1,3) -> 0x80
	c.Set(0, 0)
	c.Set(1, 0)
	c.Set(3, 3)
	if got := c.Cell(0, 0); got != rune(0x2800+0x01+0x08) {
		t.Errorf("Cell(0,0) = %U, want U+2809", got)
	}
	if got := c.Cell(1, 0); got != rune(0x2800+0x80) {
		t.Errorf("Cell(1,0) = %U, want U+2880", got)
	}
	if got := c.Cell(5, 5); got != ' ' {
		t.Errorf("out-of-range Cell = %q, want space", got)
	}
	// out-of-range Set must not panic
	c.Set(-1, 0)
	c.Set(99, 99)
}

func TestCatmullRomInterpolate(t *testing.T) {
	if len([]float64{}) != 0 {
		t.Fatal("unreachable")
	}
	samples := []float64{0.0, 1.0, 0.0}
	if v := CatmullRomInterpolate(samples, 0); v != 0.0 {
		t.Errorf("at 0 = %v, want 0", v)
	}
	if v := CatmullRomInterpolate(samples, 1); v != 1.0 {
		t.Errorf("at 1 = %v, want 1", v)
	}
	// midpoint must be smooth and bounded
	v := CatmullRomInterpolate(samples, 0.5)
	if v < 0.4 || v > 1.0 {
		t.Errorf("at 0.5 = %v, want in [0.4, 1.0]", v)
	}
	// clamping: never below 0 or above 1
	spiky := []float64{1.0, 0.0, 1.0}
	for _, tt := range []float64{-1, 0.5, 1, 2, 99} {
		if v := CatmullRomInterpolate(spiky, tt); v < 0 || v > 1 {
			t.Errorf("overshoot at %v: %v", tt, v)
		}
	}
}

func TestAreaFill(t *testing.T) {
	c := NewCanvas(10, 4)
	lut := NewGradientLUT(RGB{10, 20, 30}, RGB{100, 200, 255}, 16)
	samples := []float64{0.2, 0.5, 0.8, 1.0, 0.6, 0.3}

	DrawAreaFill(c, samples, false)
	ColorRows(c, lut, false)
	lines := c.RenderLines()
	if len(lines) != 4 {
		t.Fatalf("RenderLines count = %d, want 4", len(lines))
	}
	// bottom row must be filled (baseline), top row partial
	if strings.TrimSpace(stripSeq(lines[3])) == "" {
		t.Error("bottom row should be filled")
	}
}

func TestSmooth(t *testing.T) {
	in := []float64{0, 1, 0, 1, 0}
	out := Smooth(in, 0.5)
	if len(out) != len(in) {
		t.Fatalf("len = %d", len(out))
	}
	// smoothing must damp the spikes
	if out[1] >= 1.0 || out[1] <= 0.4 {
		t.Errorf("smoothed spike = %v, want damped", out[1])
	}
	if got := Smooth(nil, 0.5); got != nil {
		t.Error("Smooth(nil) should be nil")
	}
}

func TestGradientLUT(t *testing.T) {
	lut := NewGradientLUT(RGB{0, 0, 0}, RGB{255, 255, 255}, 32)
	if len(lut.Steps) != 32 {
		t.Fatalf("steps = %d", len(lut.Steps))
	}
	if lut.Sample(0) != (RGB{0, 0, 0}) {
		t.Errorf("t=0 -> %v", lut.Sample(0))
	}
	if lut.Sample(1) != (RGB{255, 255, 255}) {
		t.Errorf("t=1 -> %v", lut.Sample(1))
	}
	mid := lut.Sample(0.5)
	if mid.R < 100 || mid.R > 160 {
		t.Errorf("midpoint = %v, want ~128", mid)
	}
}

func TestRenderWaveformGlyphs(t *testing.T) {
	samples := []float64{0.1, 0.5, 0.9, 0.5, 0.1}
	for _, g := range []GlyphSet{GlyphBraille, GlyphBlocks, GlyphAscii} {
		rows := RenderWaveform(samples, 20, 4, g, false)
		if len(rows) != 4 {
			t.Fatalf("glyphs=%d rows = %d", g, len(rows))
		}
		if rows[3] == strings.Repeat(" ", 20) {
			t.Errorf("glyphs=%d: bottom row empty, want fill", g)
		}
		up := RenderWaveform(samples, 20, 4, g, true)
		if up[0] == strings.Repeat(" ", 20) {
			t.Errorf("glyphs=%d inverted: top row empty, want fill", g)
		}
	}
	if ParseGlyphSet("blocks") != GlyphBlocks {
		t.Error("blocks not parsed")
	}
	if ParseGlyphSet("ascii") != GlyphAscii {
		t.Error("ascii not parsed")
	}
	if ParseGlyphSet("braille") != GlyphBraille || ParseGlyphSet("nope") != GlyphBraille {
		t.Error("default should be braille")
	}
}

func TestQuantize256(t *testing.T) {
	if got := quantize256(0, 0, 0); got != 16 {
		t.Errorf("black = %d, want 16", got)
	}
	if got := quantize256(255, 255, 255); got != 231 {
		t.Errorf("white = %d, want 231", got)
	}
	if got := quantize256(255, 0, 0); got != 196 {
		t.Errorf("red = %d, want 196", got)
	}
}

func TestRenderPlainHasNoEscapes(t *testing.T) {
	c := NewCanvas(8, 2)
	DrawAreaFill(c, []float64{0.5, 0.8, 0.3}, false)
	c.SetColor(0, 0, "#ff0000")
	for _, line := range c.RenderPlain() {
		if strings.Contains(line, "\x1b") {
			t.Errorf("plain render contains escapes: %q", line)
		}
	}
	lines := c.RenderLinesCap(CapNone)
	for _, line := range lines {
		if strings.Contains(line, "\x1b") {
			t.Errorf("CapNone contains escapes: %q", line)
		}
	}
}

func stripSeq(s string) string {
	var sb strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == 0x1b {
			j := i + 1
			for j < len(s) && (s[j] < 0x40 || s[j] > 0x7e) {
				j++
			}
			i = j
			continue
		}
		sb.WriteByte(s[i])
	}
	return sb.String()
}

func BenchmarkCanvasRenderLines(b *testing.B) {
	c := NewCanvas(100, 12)
	lut := NewGradientLUT(RGB{0, 100, 200}, RGB{200, 240, 255}, 32)
	samples := make([]float64, 40)
	for i := range samples {
		samples[i] = 0.5
	}
	DrawAreaFill(c, samples, false)
	ColorRows(c, lut, false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = c.RenderLines()
	}
}

func BenchmarkCanvasDrawAreaFill(b *testing.B) {
	c := NewCanvas(100, 12)
	samples := make([]float64, 40)
	for i := range samples {
		samples[i] = float64(i) / 40.0
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Reset(100, 12)
		DrawAreaFill(c, samples, false)
	}
}

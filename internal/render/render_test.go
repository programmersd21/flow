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

// A bright-to-dark ramp used to wrap in the uint8 subtraction (end-start) and
// snap back to bright at the midpoint. The values here are the exact output a
// correct linear interpolation produces.
func TestGradientLUTDescendingDoesNotWrap(t *testing.T) {
	lut := NewGradientLUT(RGB{255, 255, 255}, RGB{0, 0, 0}, 5)
	want := []RGB{
		{255, 255, 255}, {191, 191, 191}, {128, 128, 128}, {64, 64, 64}, {0, 0, 0},
	}
	for i, w := range want {
		if got := lut.Steps[i]; got != w {
			t.Errorf("step %d = %v, want %v", i, got, w)
		}
	}
}

// Palette LUTs carry terminal palette indices all the way to the emitted
// escape. They must use classic 16-color SGR (30-37/90-97) so the color is the
// terminal's own, in every color profile.
func TestPaletteLUTEmitsTrueANSIColors(t *testing.T) {
	paint := func(palette string) *Canvas {
		c := NewCanvas(2, 1)
		DrawAreaFill(c, []float64{1.0}, false)
		ColorRows(c, &LUT{Palette: []string{palette, palette}}, false)
		return c
	}
	for _, cap := range []ColorCap{CapTrue, Cap256, Cap16} {
		if got := paint("4").RenderLinesCap(cap)[0]; !strings.Contains(got, "\x1b[34m") {
			t.Errorf("cap %v: palette 4 = %q, want SGR 34", cap, got)
		}
		if got := paint("12").RenderLinesCap(cap)[0]; !strings.Contains(got, "\x1b[94m") {
			t.Errorf("cap %v: palette 12 = %q, want SGR 94", cap, got)
		}
		if got := paint("8").RenderLinesCap(cap)[0]; !strings.Contains(got, "\x1b[90m") {
			t.Errorf("cap %v: palette 8 = %q, want SGR 90", cap, got)
		}
	}
}

// Under a 16-color cap a truecolor theme must be quantized to a 16-color SGR
// code, not emitted as a 256-color escape the terminal cannot render.
func TestCap16QuantizesHexToSixteenColors(t *testing.T) {
	c := NewCanvas(2, 1)
	DrawAreaFill(c, []float64{1.0}, false)
	ColorRows(c, &LUT{Steps: []RGB{{255, 0, 0}, {255, 0, 0}}}, false)
	if got := c.RenderLinesCap(Cap16)[0]; !strings.Contains(got, "\x1b[91m") {
		t.Errorf("red under Cap16 = %q, want SGR 91", got)
	}
	if got := c.RenderLinesCap(CapTrue)[0]; !strings.Contains(got, "\x1b[38;2;255;0;0m") {
		t.Errorf("red under CapTrue = %q, want truecolor escape", got)
	}
}

func TestQuantize16(t *testing.T) {
	cases := []struct {
		r, g, b uint8
		want    int
	}{
		{255, 0, 0, 9}, {0, 0, 0, 0}, {255, 255, 255, 15},
		{0, 0, 238, 4}, {0, 205, 0, 2}, {229, 229, 229, 7},
	}
	for _, tc := range cases {
		if got := quantize16(tc.r, tc.g, tc.b); got != tc.want {
			t.Errorf("quantize16(%d,%d,%d) = %d, want %d", tc.r, tc.g, tc.b, got, tc.want)
		}
	}
}

func TestQuantize256GrayRampEndpoints(t *testing.T) {
	if got := quantize256(8, 8, 8); got != 232 {
		t.Errorf("gray 8 = %d, want 232", got)
	}
	if got := quantize256(238, 238, 238); got != 255 {
		t.Errorf("gray 238 = %d, want 255", got)
	}
}

func TestDetectColorCapDumbTerminal(t *testing.T) {
	t.Setenv("TERM", "dumb")
	t.Setenv("COLORTERM", "")
	if got := DetectColorCap(false); got != CapNone {
		t.Errorf("TERM=dumb -> %v, want CapNone", got)
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

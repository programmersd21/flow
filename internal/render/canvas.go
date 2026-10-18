package render

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Canvas is a terminal-cell canvas where each cell holds 2x4 braille dots.
// W and H are in terminal cells. Dot coordinates: x in [0,2W), y in [0,4H).
//
// Braille dot-to-bit mapping:
//
//	dot1 (0,0) = 0x01     dot4 (1,0) = 0x08
//	dot2 (0,1) = 0x02     dot5 (1,1) = 0x10
//	dot3 (0,2) = 0x04     dot6 (1,2) = 0x20
//	dot7 (0,3) = 0x40     dot8 (1,3) = 0x80
type Canvas struct {
	W, H   int
	cells  []uint8
	colors []string
}

// NewCanvas allocates a canvas. Dimensions below 1 are clamped to 1.
func NewCanvas(w, h int) *Canvas {
	if w <= 0 {
		w = 1
	}
	if h <= 0 {
		h = 1
	}
	size := w * h
	return &Canvas{W: w, H: h, cells: make([]uint8, size), colors: make([]string, size)}
}

// Reset clears the canvas, reusing the buffers when they are large enough.
// This keeps the steady-state render path allocation-free.
func (c *Canvas) Reset(w, h int) {
	if w <= 0 {
		w = 1
	}
	if h <= 0 {
		h = 1
	}
	size := w * h
	c.W, c.H = w, h
	if cap(c.cells) < size {
		c.cells = make([]uint8, size)
		c.colors = make([]string, size)
		return
	}
	c.cells = c.cells[:size]
	c.colors = c.colors[:size]
	for i := range c.cells {
		c.cells[i] = 0
		c.colors[i] = ""
	}
}

// Set lights one braille dot. Out-of-range coordinates are ignored.
func (c *Canvas) Set(x, y int) {
	if x < 0 || x >= 2*c.W || y < 0 || y >= 4*c.H {
		return
	}
	cx, cy := x/2, y/4
	dx, dy := x%2, y%4

	var mask uint8
	switch {
	case dx == 0 && dy == 0:
		mask = 0x01
	case dx == 0 && dy == 1:
		mask = 0x02
	case dx == 0 && dy == 2:
		mask = 0x04
	case dx == 0 && dy == 3:
		mask = 0x40
	case dx == 1 && dy == 0:
		mask = 0x08
	case dx == 1 && dy == 1:
		mask = 0x10
	case dx == 1 && dy == 2:
		mask = 0x20
	default:
		mask = 0x80
	}
	c.cells[cy*c.W+cx] |= mask
}

// SetColor assigns a color to one cell. Out-of-range cells are ignored.
func (c *Canvas) SetColor(cx, cy int, col string) {
	if cx < 0 || cx >= c.W || cy < 0 || cy >= c.H {
		return
	}
	c.colors[cy*c.W+cx] = col
}

// Cell returns the braille rune at (cx, cy), or ' ' when empty/out of range.
func (c *Canvas) Cell(cx, cy int) rune {
	if cx < 0 || cx >= c.W || cy < 0 || cy >= c.H {
		return ' '
	}
	if v := c.cells[cy*c.W+cx]; v != 0 {
		return rune(0x2800 + int(v))
	}
	return ' '
}

// RenderLines writes the canvas rows into lines (truecolor escapes).
func (c *Canvas) RenderLines() []string {
	return c.renderLines(emitTruecolor)
}

// RenderPlain emits bare braille rows with no escape codes. Used for
// NO_COLOR and dumb terminals: glyph density carries the shape.
func (c *Canvas) RenderPlain() []string {
	lines := make([]string, c.H)
	var sb strings.Builder
	for cy := 0; cy < c.H; cy++ {
		sb.Reset()
		for cx := 0; cx < c.W; cx++ {
			if v := c.cells[cy*c.W+cx]; v == 0 {
				sb.WriteByte(' ')
			} else {
				sb.WriteRune(rune(0x2800 + int(v)))
			}
		}
		lines[cy] = sb.String()
	}
	return lines
}

type colorEmitter func(sb *strings.Builder, col string) bool

// emitPalette16 writes the classic SGR code for a terminal palette index
// ("0"-"15"). This is the only escape form that is guaranteed to resolve to
// the user's own palette; 38;5;N enters the 256-color space, which some
// terminals keep fixed to xterm's defaults instead of the configured colors.
// Reports false when idx is outside 0-15.
func emitPalette16(sb *strings.Builder, idx int) bool {
	switch {
	case idx >= 0 && idx < 8:
		fmt.Fprintf(sb, "\x1b[%dm", 30+idx)
		return true
	case idx >= 8 && idx < 16:
		fmt.Fprintf(sb, "\x1b[%dm", 90+idx-8)
		return true
	}
	return false
}

// emitIndexed handles a bare palette index: 0-15 via real 16-color SGR, and
// anything above as a 256-color index. Reports false for non-numeric strings.
func emitIndexed(sb *strings.Builder, col string) bool {
	n, err := strconv.Atoi(col)
	if err != nil {
		return false
	}
	if emitPalette16(sb, n) {
		return true
	}
	fmt.Fprintf(sb, "\x1b[38;5;%dm", n)
	return true
}

func emitTruecolor(sb *strings.Builder, col string) bool {
	if col == "" {
		return false
	}
	if strings.HasPrefix(col, "#") && len(col) == 7 {
		var r, g, b uint8
		_, _ = fmt.Sscanf(col, "#%02x%02x%02x", &r, &g, &b)
		fmt.Fprintf(sb, "\x1b[38;2;%d;%d;%dm", r, g, b)
		return true
	}
	return emitIndexed(sb, col)
}

// paletteRGB holds xterm's default values for the 16 ANSI colors. They are
// only used to pick the nearest index when a truecolor value must be
// downsampled; the emitted escape references the terminal's palette, not the
// literal value.
var paletteRGB = [16][3]uint8{
	{0, 0, 0}, {205, 0, 0}, {0, 205, 0}, {205, 205, 0},
	{0, 0, 238}, {205, 0, 205}, {0, 205, 205}, {229, 229, 229},
	{127, 127, 127}, {255, 0, 0}, {0, 255, 0}, {255, 255, 0},
	{92, 92, 255}, {255, 0, 255}, {0, 255, 255}, {255, 255, 255},
}

// quantize16 maps an RGB triplet to the nearest of the 16 ANSI palette colors.
func quantize16(r, g, b uint8) int {
	best, bestDist := 0, 1<<62
	for i, c := range paletteRGB {
		dr := int(r) - int(c[0])
		dg := int(g) - int(c[1])
		db := int(b) - int(c[2])
		if d := dr*dr + dg*dg + db*db; d < bestDist {
			best, bestDist = i, d
		}
	}
	return best
}

// quantize256 maps an RGB triplet to the closest xterm-256 color index.
func quantize256(r, g, b uint8) int {
	if absDiff(r, g) < 8 && absDiff(g, b) < 8 {
		if r < 8 {
			return 16
		}
		if r > 238 {
			return 231
		}
		// The 24-step gray ramp runs 8,18,...,238 (232-255). Integer division
		// over a rounded ratio must land on 23 for 238; the previous formula
		// stopped one short and mapped the brightest gray to a darker step.
		step := int(math.Round(float64(int(r)-8) / 10.0))
		if step > 23 {
			step = 23
		}
		return 232 + step
	}
	ri := int(float64(r)/255*5 + 0.5)
	gi := int(float64(g)/255*5 + 0.5)
	bi := int(float64(b)/255*5 + 0.5)
	return 16 + 36*ri + 6*gi + bi
}

func absDiff(a, b uint8) uint8 {
	if a > b {
		return a - b
	}
	return b - a
}

func emit256(sb *strings.Builder, col string) bool {
	if col == "" {
		return false
	}
	if strings.HasPrefix(col, "#") && len(col) == 7 {
		var r, g, b uint8
		_, _ = fmt.Sscanf(col, "#%02x%02x%02x", &r, &g, &b)
		fmt.Fprintf(sb, "\x1b[38;5;%dm", quantize256(r, g, b))
		return true
	}
	return emitIndexed(sb, col)
}

// emit16 is the 16-color degradation path: truecolor values are snapped to the
// nearest palette entry and emitted as classic SGR. Emitting 38;5;N here (as
// this used to) is a 256-color escape on a terminal that cannot show it.
func emit16(sb *strings.Builder, col string) bool {
	if col == "" {
		return false
	}
	if strings.HasPrefix(col, "#") && len(col) == 7 {
		var r, g, b uint8
		_, _ = fmt.Sscanf(col, "#%02x%02x%02x", &r, &g, &b)
		emitPalette16(sb, quantize16(r, g, b))
		return true
	}
	return emitIndexed(sb, col)
}

// RenderLinesCap renders honoring a terminal color capability.
func (c *Canvas) RenderLinesCap(cap ColorCap) []string {
	switch cap {
	case CapNone:
		return c.RenderPlain()
	case Cap16:
		return c.renderLines(emit16)
	case Cap256:
		return c.renderLines(emit256)
	default:
		return c.renderLines(emitTruecolor)
	}
}

func (c *Canvas) renderLines(emit colorEmitter) []string {
	lines := make([]string, c.H)
	var sb strings.Builder
	for cy := 0; cy < c.H; cy++ {
		sb.Reset()
		lastColor := ""
		for cx := 0; cx < c.W; cx++ {
			idx := cy*c.W + cx
			charVal := c.cells[idx]
			col := c.colors[idx]

			if charVal == 0 {
				if lastColor != "" {
					sb.WriteString("\x1b[0m")
					lastColor = ""
				}
				sb.WriteByte(' ')
				continue
			}

			if col != lastColor {
				if !emit(&sb, col) {
					sb.WriteString("\x1b[0m")
				}
				lastColor = col
			}
			sb.WriteRune(rune(0x2800 + int(charVal)))
		}
		if lastColor != "" {
			sb.WriteString("\x1b[0m")
		}
		lines[cy] = sb.String()
	}
	return lines
}

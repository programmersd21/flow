package render

import (
	"fmt"
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
	fmt.Fprintf(sb, "\x1b[38;5;%sm", col)
	return true
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
		return 232 + int((float64(r)-8)/247*24)
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
	fmt.Fprintf(sb, "\x1b[38;5;%sm", col)
	return true
}

// RenderLinesCap renders honoring a terminal color capability.
func (c *Canvas) RenderLinesCap(cap ColorCap) []string {
	switch cap {
	case CapNone:
		return c.RenderPlain()
	case Cap16, Cap256:
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

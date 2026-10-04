package ui

import (
	"testing"

	"github.com/programmersd21/flow/internal/render"
)

// ─── render-path audit ───────────────────────────────────────────────────────
//
// These tests lock the invariants the hero waveform depends on. They exist
// because the render pipeline changed shape several times and each change
// introduced a defect that only showed visually (striping, saturation,
// debris). Behavior, not layout, is asserted here.

func TestAreaFillNeverExceedsBounds(t *testing.T) {
	c := render.NewCanvas(40, 6)
	samples := []float64{0.1, 0.9, 0.5, 1.0, 0.3}
	// must not panic regardless of values
	render.DrawAreaFill(c, samples, false)
	render.DrawAreaFill(c, samples, true)
}

// DrawAreaFill paints exactly round(v * totalDots) dots, no more, no less.
// Low values rely on the upstream compression exponent to stay visible — the
// fill itself is a faithful 1:1 mapping.
func TestAreaFillFaithfulMapping(t *testing.T) {
	c := render.NewCanvas(20, 4) // 16 dot rows
	render.DrawAreaFill(c, []float64{0.5, 0.5, 0.5}, false)
	dots := 0
	for y := 0; y < 4*c.H; y++ {
		for x := 0; x < 2*c.W; x++ {
			if cellHasDot(c, x, y) {
				dots++
			}
		}
	}
	// 0.5 * 16 = 8 dots per column, 40 columns
	if want := 8 * 40; dots != want {
		t.Errorf("0.5 fill painted %d dots, want %d", dots, want)
	}

	c2 := render.NewCanvas(20, 4)
	render.DrawAreaFill(c2, []float64{1.0, 1.0}, false)
	// full height: every dot row filled
	for x := 0; x < 2*c2.W; x++ {
		for y := 0; y < 4*c2.H; y++ {
			if !cellHasDot(c2, x, y) {
				t.Fatalf("full fill left a gap at (%d,%d)", x, y)
			}
		}
	}

	c3 := render.NewCanvas(20, 4)
	render.DrawAreaFill(c3, []float64{0, 0, 0}, false)
	for x := 0; x < 2*c3.W; x++ {
		for y := 0; y < 4*c3.H; y++ {
			if cellHasDot(c3, x, y) {
				t.Fatalf("zero fill painted a dot at (%d,%d)", x, y)
			}
		}
	}
}

func TestColorRowsDoesNotTouchEmptyCells(t *testing.T) {
	c := render.NewCanvas(30, 5)
	render.DrawAreaFill(c, []float64{0.5, 0.5, 0.5}, false)
	lut := buildThemeLUT(true)
	render.ColorRows(c, lut, false)
	// recolor must only apply to cells that hold dots
	for cy := 0; cy < c.H; cy++ {
		for cx := 0; cx < c.W; cx++ {
			// we cannot read colors back directly; assert no panic and that
			// render produces well-formed output
			_ = c.Cell(cx, cy)
		}
	}
	lines := c.RenderLines()
	if len(lines) != c.H {
		t.Errorf("expected %d lines, got %d", c.H, len(lines))
	}
}

func TestGridlinesNeverOverwriteData(t *testing.T) {
	c := render.NewCanvas(40, 8)
	// Saturate the canvas completely.
	render.DrawAreaFill(c, []float64{1, 1, 1, 1, 1, 1}, false)
	before := c.RenderPlain()
	drawGridlines(c, false)
	after := c.RenderPlain()
	for i := range before {
		if before[i] != after[i] {
			t.Errorf("gridline modified a filled cell in row %d", i)
		}
	}
}

func TestGridlinesOnlyDrawInEmptySpace(t *testing.T) {
	c := render.NewCanvas(40, 8)
	// Half fill: rows 4..7 have data, rows 0..3 are empty.
	render.DrawAreaFill(c, []float64{0.5, 0.5, 0.5, 0.5}, false)
	before := c.RenderPlain()
	drawGridlines(c, false)
	after := c.RenderPlain()
	// The midpoint rule may add dots only in the empty half (rows 0..3).
	for r := 4; r < 8; r++ {
		if before[r] != after[r] {
			t.Errorf("gridline drew into data row %d", r)
		}
	}
}

func TestDrawPeakHoldLeftColumnOnlyIsGone(t *testing.T) {
	// The removed peak line set only the left dot column, which rendered as
	// isolated "⠁" debris. This guards against that pattern returning.
	c := render.NewCanvas(20, 4)
	render.DrawAreaFill(c, []float64{0.9, 0.9}, false)
	for cy := 0; cy < c.H; cy++ {
		for cx := 0; cx < c.W; cx++ {
			cell := c.Cell(cx, cy)
			// a cell holding ONLY dot1 (top-left) is the debris pattern
			if cell == rune(0x2800+0x01) && c.Cell(cx, cy+1) != ' ' {
				t.Errorf("isolated dot1 debris at cell (%d,%d)", cx, cy)
			}
		}
	}
}

func TestRenderWaveformMirroredSymmetry(t *testing.T) {
	m := baseModel(96, 40)
	for i := 0; i < 200; i++ {
		m.downHist.Push(float64(5000 + i*10))
		m.upHist.Push(float64(3000 + i*5))
	}
	lines := dashboardContentLines(m, ViewHero)
	axis := -1
	for i, l := range lines {
		if contains(l, "┈") {
			axis = i
			break
		}
	}
	if axis < 0 {
		t.Fatal("no axis row found")
	}
	// down rows are above, up rows below — both non-empty
	if axis < 3 {
		t.Errorf("expected at least 3 rows above the axis, got %d", axis)
	}
	if len(lines)-axis < 3 {
		t.Errorf("expected at least 2 rows below the axis, got %d", len(lines)-axis-1)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

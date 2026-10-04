package ui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/programmersd21/flow/internal/render"
	"github.com/programmersd21/flow/internal/theme"
)

// ─── hero view ───────────────────────────────────────────────────────────────
//
// Layout (top to bottom):
//   1. header:  flow v0.3.2  ............  ● iface · latency
//   2. hero numbers: big digits download (left) / upload (right)
//   3. mirrored braille waveform (down above axis, up below)
//   4. time labels:  -60s ... now ▏
//   5. stats row:  today totals, ping, top talker
//   6. iface · status line + FROZEN footer (byte-identical to v0.3.1)

// heroHalfRows is the number of terminal rows per waveform half (down or up).
func heroHalfRows(termH int) int {
	switch {
	case termH >= 52:
		return 9
	case termH >= 46:
		return 8
	case termH >= 40:
		return 7
	case termH >= 34:
		return 5
	case termH >= 28:
		return 4
	case termH >= 24:
		return 3
	default:
		return 2
	}
}

// renderHeroDigits renders the big-digit block for one direction.
//
// It returns exactly digitSize rows of pure glyphs, all the same width. The
// arrow and unit deliberately do NOT go on the baseline row: that made the last
// row wider than the four above it, so the block stopped being tabular and a
// fragment of the number read as detached debris — clearest on the upload side,
// where the block is right-aligned.
func renderHeroDigits(bps float64, bits bool, download bool) []string {
	num, _ := splitHeroNumber(bps, bits)
	rows := renderBigDigits(num)

	// Vivid accent hue, not a gradient intensity: gradient crests fade toward
	// near-white pastels by design, which reads as beige on 5-row glyphs.
	style := theme.DownStyle()
	if !download {
		style = theme.UpStyle()
	}

	out := make([]string, 0, digitSize)
	for _, r := range rows {
		out = append(out, style.Render(r))
	}
	return out
}

// renderHeroCaption is the row directly beneath a digit block. The arrow and
// unit sit on the LEFT, followed by the stat, so nothing hangs off the digits
// and the whole line stays tabular:
//
//	↓ KB/s · peak 4.5 MB/s
func renderHeroCaption(bps float64, bits bool, download bool, stat string) string {
	_, unit := splitHeroNumber(bps, bits)
	arrow := "↑"
	if download {
		arrow = "↓"
	}
	style := theme.DownStyle()
	if !download {
		style = theme.UpStyle()
	}
	head := style.Render(arrow) + " " + theme.Muted().Render(unit)
	if stat == "" {
		return head
	}
	return head + theme.TextDim().Render(" · ") + stat
}

// joinCentered places the two hero blocks side by side and centers the pair.
//
// The invariant, stated plainly: each hero block has a fixed width, every row
// is centered inside that width, and the two blocks are centered as one group.
//
// An earlier version computed a pixel axis per block and then positioned each
// row with `rpad - lipgloss.Width(row)`, mixing an absolute terminal position
// with an already-padded row width — subtracting the left block twice and
// breaking the centering it claimed to maintain.
func joinCentered(left, right []string, totalWidth int) []string {
	n := max(len(left), len(right))
	for len(left) < n {
		left = append(left, "")
	}
	for len(right) < n {
		right = append(right, "")
	}

	gap := 8

	leftW := widestRow(left)
	rightW := widestRow(right)

	if leftW+gap+rightW > totalWidth {
		gap = max(1, totalWidth-leftW-rightW)
	}

	groupW := leftW + gap + rightW
	groupStart := max(0, (totalWidth-groupW)/2)

	out := make([]string, n)

	for i := range out {
		l := lipgloss.NewStyle().
			Width(leftW).
			Align(lipgloss.Center).
			Render(left[i])

		r := lipgloss.NewStyle().
			Width(rightW).
			Align(lipgloss.Center).
			Render(right[i])

		out[i] = strings.Repeat(" ", groupStart) +
			l +
			strings.Repeat(" ", gap) +
			r
	}

	return out
}

// widestRow is the render width of the widest row in a block.
func widestRow(rows []string) int {
	w := 0
	for _, r := range rows {
		if rw := lipgloss.Width(r); rw > w {
			w = rw
		}
	}
	return w
}

// ─── vertical scaling ───────────────────────────────────────────────────────

// Vertical-axis compression exponents per scale mode (`S` cycles them).
//
// Real traffic spans a huge dynamic range — a 70 KB/s idle floor next to an
// 8 MB/s burst is 120x — and a purely linear axis crushes everything below
// the peak into an invisible sliver. The default "auto" uses a gentle curve
// so the idle floor and a steady plateau are both legible while bursts still
// reach the top of the half.
const (
	expoAuto   = 0.65
	expoLinear = 1.0
	expoSqrt   = 0.45
)

func scaleExponent(mode int) float64 {
	switch mode {
	case 1:
		return expoLinear
	case 2:
		return expoSqrt
	default:
		return expoAuto
	}
}

func scaleModeName(mode int) string {
	switch mode {
	case 1:
		return "linear"
	case 2:
		return "sqrt"
	default:
		return "auto"
	}
}

// graphScale picks the y-axis ceiling for one waveform half.
//
// The ceiling tracks the peak of the *visible window*, not a long-lived
// rolling max. Two earlier versions were wrong in opposite directions:
//
//   - Ceiling from a rolling max: a burst that had already scrolled off
//     squashed everything still on screen into an invisible sliver.
//   - Ceiling pinned to peak*0.6 "to avoid magnifying a lone spike": when
//     traffic is steady, typical values sat ABOVE that ceiling, so the whole
//     waveform clipped to solid blocks and lost all shape. That looked like a
//     rendering bug and was one.
//
// Peak*1.05 never clips visible data, and the sqrt compression in normVal
// already gives a lone spike the visual dominance it deserves without hiding
// the rest of the traffic.
func graphScale(samples []float64, _ float64) float64 {
	const absFloor = 10 * 1000 // 10 KB/s: noise must not fill the screen
	peak := 0.0
	for _, v := range samples {
		if v > peak {
			peak = v
		}
	}
	ymax := peak * 1.05
	if ymax < absFloor {
		return absFloor
	}
	return ymax
}

// normVal maps one bps value into [0,1] with the ceiling and compression.
func normVal(v, ceil, exponent float64) float64 {
	if ceil <= 0 {
		return 0
	}
	x := v / ceil
	if x < 0 || math.IsNaN(x) {
		x = 0
	}
	if x > 1 {
		x = 1
	}
	if exponent == 1 {
		return x
	}
	return math.Pow(x, exponent)
}

// normalizeSamples scales raw bps samples into [0,1] with the ceiling and
// the vertical compression curve.
func normalizeSamples(samples []float64, maxVal, exponent float64) []float64 {
	if len(samples) == 0 || maxVal <= 0 {
		return nil
	}
	out := make([]float64, len(samples))
	for i, s := range samples {
		out[i] = normVal(s, maxVal, exponent)
	}
	return out
}

// scaleSamples multiplies all samples by f (launch fill-from-axis effect).
func scaleSamples(s []float64, f float64) []float64 {
	if f >= 1.0 {
		return s
	}
	out := make([]float64, len(s))
	for i, v := range s {
		out[i] = v * f
	}
	return out
}

// easeOutCubic: 1-(1-t)^3, the standard launch/transition easing.
func easeOutCubic(t float64) float64 {
	u := 1 - t
	return 1 - u*u*u
}

func isIdle(norm []float64) bool {
	if len(norm) == 0 {
		return true
	}
	for _, v := range norm {
		if v > 0.02 {
			return false
		}
	}
	return true
}

// idlePulseBrightness returns a slow breathing factor in [0.92, 1.08].
func idlePulseBrightness(now time.Time) float64 {
	phase := float64(now.UnixMilli()%4000) / 4000.0 * 2 * math.Pi
	return 1.0 + 0.08*math.Sin(phase)
}

// buildThemeLUT builds a 32-step gradient LUT from the active theme.
func buildThemeLUT(download bool) *render.LUT {
	steps := make([]render.RGB, 32)
	for i := range steps {
		t := float64(i) / 31.0
		steps[i] = hexToRGBRender(theme.GradientHex(download, t))
	}
	return &render.LUT{Steps: steps}
}

func hexToRGBRender(hex string) render.RGB {
	var r, g, b uint8
	if len(hex) == 7 && hex[0] == '#' {
		_, _ = fmt.Sscanf(hex, "#%02x%02x%02x", &r, &g, &b)
	}
	return render.RGB{R: r, G: g, B: b}
}

// flatLUT builds a single-color LUT for the no-gradient path.
func flatLUT(style lipgloss.Style) *render.LUT {
	hex := theme.SubtleHex() // fallback, not a design color
	if v, ok := style.GetForeground().(lipgloss.Color); ok {
		hex = string(v)
	}
	c := hexToRGBRender(hex)
	return &render.LUT{Steps: []render.RGB{c}}
}

// dimHexStr scales a theme color toward black for fixed chrome tones
// (gridline, peak line) so they track the active theme instead of being
// hardcoded literals. Delegates to theme.DimColor, which is a no-op for ANSI
// palette indices rather than mis-parsing them as hex into black.
func dimHexStr(col string, f float64) string {
	return theme.DimColor(col, f)
}

// tintRows colors uncolored fallback-glyph rows with a single mid tone.
func tintRows(rows []string, download bool) []string {
	style := theme.DownloadColor(0.6)
	if !download {
		style = theme.UploadColor(0.6)
	}
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = style.Render(r)
	}
	return out
}

// ─── waveform ───────────────────────────────────────────────────────────────

// renderOpts bundles waveform options so the signature stays small.
type renderOpts struct {
	now                  time.Time
	filter               DisplayFilter
	rippleDown, rippleUp time.Time
	noAnim               bool
	scaleMode            int
	grid                 bool
	glyphs               render.GlyphSet
	colorCap             render.ColorCap
	gradient             bool
}

// renderMirroredWave renders the mirrored braille waveform: download above
// the center axis, upload below it. Samples must already be normalized to
// [0,1] by the caller (single normalization point, shared with peak-hold).
func renderMirroredWave(normDown, normUp []float64, width, halfH int, o renderOpts) []string {
	if width < 8 {
		width = 8
	}
	if halfH < 1 {
		halfH = 1
	}

	downLUT := buildThemeLUT(true)
	upLUT := buildThemeLUT(false)

	lines := make([]string, 0, 2*halfH+1)

	idle := isIdle(normDown) && isIdle(normUp)
	pulse := idlePulseBrightness(o.now)

	showDown := o.filter != DisplayUpOnly
	showUp := o.filter != DisplayDownOnly
	mirrored := o.filter == DisplayBoth

	downH, upH := halfH, halfH
	if !mirrored {
		// A lone direction gets the full height.
		if showDown {
			downH = halfH*2 + 1
		} else {
			upH = halfH*2 + 1
		}
	}

	if showDown {
		if o.glyphs != render.GlyphBraille {
			lines = append(lines, tintRows(render.RenderWaveform(normDown, width, downH, o.glyphs, false), true)...)
		} else {
			c := render.NewCanvas(width, downH)
			if idle {
				drawIdleLine(c, pulse, false)
			} else {
				render.DrawAreaFill(c, normDown, false)
				if o.gradient {
					render.ColorRows(c, downLUT, false)
				} else {
					render.ColorRows(c, flatLUT(theme.DownloadColor(0.85)), false)
				}
				if o.grid {
					drawGridlines(c, false)
				}
				brightenNowColumn(c, downLUT)
				if !o.noAnim && !o.rippleDown.IsZero() {
					applyRipple(c, o.now.Sub(o.rippleDown), downLUT, false)
				}
			}
			lines = append(lines, c.RenderLinesCap(o.colorCap)...)
		}
	}

	if mirrored {
		lines = append(lines, renderAxisRow(width))
	}

	if showUp {
		if o.glyphs != render.GlyphBraille {
			lines = append(lines, tintRows(render.RenderWaveform(normUp, width, upH, o.glyphs, true), false)...)
		} else {
			c := render.NewCanvas(width, upH)
			if idle {
				drawIdleLine(c, pulse, true)
			} else {
				render.DrawAreaFill(c, normUp, true)
				if o.gradient {
					render.ColorRows(c, upLUT, true)
				} else {
					render.ColorRows(c, flatLUT(theme.UploadColor(0.85)), true)
				}
				if o.grid {
					drawGridlines(c, true)
				}
				brightenNowColumn(c, upLUT)
				if !o.noAnim && !o.rippleUp.IsZero() {
					applyRipple(c, o.now.Sub(o.rippleUp), upLUT, true)
				}
			}
			lines = append(lines, c.RenderLinesCap(o.colorCap)...)
		}
	}

	return lines
}

// drawIdleLine renders a flat dotted line along the baseline with a subtle pulse.
func drawIdleLine(c *render.Canvas, brightness float64, inverted bool) {
	baseY := 4*c.H - 1 // download: baseline at bottom
	if inverted {
		baseY = 0 // upload: baseline at top
	}
	col := theme.SubtleHex()
	if brightness > 1.0 {
		col = theme.MutedHex()
	}
	for x := 0; x < 2*c.W; x += 2 {
		c.Set(x, baseY)
		c.SetColor(x/2, baseY/4, col)
	}
}

// brightenNowColumn tints the rightmost 2 columns toward the crest tone so the
// eye sees live data entering.
func brightenNowColumn(c *render.Canvas, lut *render.LUT) {
	if lut == nil || len(lut.Steps) == 0 {
		return
	}
	for cx := c.W - 2; cx < c.W; cx++ {
		if cx < 0 {
			continue
		}
		for cy := 0; cy < c.H; cy++ {
			if c.Cell(cx, cy) != ' ' {
				c.SetColor(cx, cy, lut.Sample(0.85).Hex())
			}
		}
	}
}

// drawGridlines adds a single faint dotted rule at the midpoint of a half,
// but ONLY through cells that hold no data.
//
// Braille color is per-cell (2x4 dots), so a rule cannot be drawn *over* a
// filled cell without tinting that cell's other dots and producing a chunky
// band. The honest consequence is that the rule is visible only in the empty
// region above/below the fill — which is exactly where a scale reference is
// useful, and invisible when the waveform is saturated (nothing to reference).
// Because of that it defaults to OFF; `G` turns it on for sparse traffic.
func drawGridlines(c *render.Canvas, inverted bool) {
	const frac = 0.5
	dim := dimHexStr(theme.SubtleHex(), 0.5)
	totalRows := 4 * c.H
	var y int
	if !inverted {
		y = int(float64(totalRows-1) * (1 - frac))
	} else {
		y = int(float64(totalRows) * frac)
	}
	if y < 0 || y >= totalRows {
		return
	}
	cy := y / 4
	for cx := 0; cx < c.W; cx++ {
		if c.Cell(cx, cy) != ' ' {
			continue // never overwrite data
		}
		c.Set(cx*2, y) // left dot only: a dotted rule, not a solid bar
		c.SetColor(cx, cy, dim)
	}
}

// cellHasDot checks the exact dot (not just any dot in the cell).
func cellHasDot(c *render.Canvas, x, y int) bool {
	r := c.Cell(x/2, y/4)
	if r == ' ' {
		return false
	}
	val := int(r) - 0x2800
	dx, dy := x%2, y%4
	var mask int
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
	return val&mask != 0
}

// applyRipple overlays a gaussian brightness bump traveling leftward for ~400ms.
func applyRipple(c *render.Canvas, elapsed time.Duration, lut *render.LUT, inverted bool) {
	if elapsed < 0 || elapsed > 400*time.Millisecond || lut == nil || len(lut.Steps) == 0 {
		return
	}
	progress := elapsed.Seconds() / 0.4
	centerX := float64(c.W) - progress*float64(c.W)*0.5
	const sigma = 3.0
	peak := lut.Sample(1.0).Hex()
	for cx := 0; cx < c.W; cx++ {
		d := float64(cx) - centerX
		if gain := math.Exp(-d * d / (2 * sigma * sigma)); gain > 0.6 {
			for cy := 0; cy < c.H; cy++ {
				hasDot := false
				for dy := 0; dy < 4 && !hasDot; dy++ {
					for dx := 0; dx < 2 && !hasDot; dx++ {
						if cellHasDot(c, cx*2+dx, cy*4+dy) {
							hasDot = true
						}
					}
				}
				if hasDot {
					c.SetColor(cx, cy, peak)
				}
			}
		}
	}
	_ = inverted
}

// renderAxisRow renders the shared centre axis. It is intentionally the
// quietest element on screen: everything else reads against it, so it stays
// in the dim tier rather than competing for attention.
func renderAxisRow(width int) string {
	if width < 1 {
		width = 1
	}
	return theme.TextDim().Render(strings.Repeat("┈", width))
}

// renderTimeLabels renders the window hints under the waveform. The right edge
// gets an accent caret so "now" reads as live rather than static.
func renderTimeLabels(width int, windowSecs int) string {
	left := theme.TextDim().Render(fmt.Sprintf("−%ds", windowSecs))
	right := theme.TextDim().Render("now") + theme.DownloadColor(0.7).Render(" ▎")
	lw := lipgloss.Width(left)
	rw := lipgloss.Width(right)
	gap := width - lw - rw
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}

// Latency bands, in one place so the header, the stats row, and the status
// line can never disagree. Bands come from the theme's Good/Warn/Bad status
// tokens rather than ad-hoc hues, which is what keeps ping color meaningful
// across every theme.
const (
	latencyWarnMs = 60.0
	latencyBadMs  = 150.0
)

func latencyStyle(ms float64) lipgloss.Style {
	switch {
	case ms >= latencyBadMs:
		return theme.BadStyle()
	case ms >= latencyWarnMs:
		return theme.WarnStyle()
	default:
		return theme.GoodStyle()
	}
}

// renderHeroStats renders the stats row: today totals, ping, top talker.
func renderHeroStats(m Model, width int) string {
	sep := theme.TextDim().Render("  ·  ")
	var parts []string
	if m.displayFilter != DisplayUpOnly {
		parts = append(parts,
			theme.DownloadColor(0.6).Render("↓")+" "+
				theme.Soft().Bold(true).Render(formatBytes(m.tracker.TodayDown)))
	}
	if m.displayFilter != DisplayDownOnly {
		parts = append(parts,
			theme.UploadColor(0.6).Render("↑")+" "+
				theme.Soft().Bold(true).Render(formatBytes(m.tracker.TodayUp)))
	}
	line := theme.TextDim().Render("today") + sep + strings.Join(parts, sep)

	if m.pingLatency > 0 {
		ms := m.pingLatency.Seconds() * 1000
		line += sep + theme.TextDim().Render("ping") + " " +
			latencyStyle(ms).Render(fmt.Sprintf("%.0fms", ms))
	}

	if len(m.procs) > 0 && width >= 72 {
		top := m.procs[0]
		name := top.Name
		if p := strings.LastIndexByte(name, '/'); p >= 0 && p+1 < len(name) {
			name = name[p+1:] // show the binary, not the full path
		}
		line += sep + theme.TextDim().Render("top") + " " + theme.Muted().Render(name) +
			" " + theme.Soft().Render(fmt.Sprintf("%d", top.Connections)) +
			theme.TextDim().Render(" conns")
	}
	return line
}

// renderHeroHeader renders the header row with a status dot + iface + paused badge.
// renderHeroHeader renders one centered identity line:
//
//	flow v0.3.2 · wlan0 · 46ms
//
// Everything that identifies the session sits on one centered row joined by
// dots. Splitting it into a left/right spread meant the title and the interface
// drifted to opposite edges, which on a wide terminal put a large empty gulf
// between them.
func renderHeroHeader(m Model, width int) string {
	sep := theme.TextDim().Render(" · ")

	parts := []string{TitleRow(m.samplePulse)}

	// Link dot pulses with live traffic: dim at idle, accent when moving.
	dotStyle := theme.TextDim()
	if m.animDown > 1000 || m.animUp > 1000 {
		dotStyle = theme.DownloadColor(0.5 + 0.5*m.samplePulse)
	}
	parts = append(parts, dotStyle.Render("●")+theme.TextDim().Render(" ")+theme.Muted().Render(m.ifaceName))

	// Surface the display filter next to the interface. Without this, hiding a
	// direction (the `d` key) made half the graph vanish with no explanation
	// anywhere near the data.
	switch m.displayFilter {
	case DisplayDownOnly:
		parts = append(parts, theme.DownloadColor(0.7).Render("↓ only"))
	case DisplayUpOnly:
		parts = append(parts, theme.UploadColor(0.7).Render("↑ only"))
	}

	if m.paused {
		// A pill reads instantly; plain text gets lost in the chrome.
		parts = append(parts, theme.UploadColor(0.8).Render("▮ ")+
			theme.UploadColor(0.45).Render("paused"))
	}

	if m.pingLatency > 0 {
		ms := m.pingLatency.Seconds() * 1000
		parts = append(parts, latencyStyle(ms).Render(fmt.Sprintf("%.0fms", ms)))
	}

	line := strings.Join(parts, sep)

	// Center it, dropping the trailing part if the terminal is too narrow.
	if lipgloss.Width(line) > width {
		line = strings.Join(parts[:len(parts)-1], sep)
	}
	return centerInline(line, width)
}

// heroContentLines builds the full hero view content.
// The final lines are the FROZEN footer rows — byte-identical to v0.3.1.
func heroContentLines(m Model, contentW, termW, termH int) []string {
	halfH := heroHalfRows(termH)

	// Budget: header(1) + gap(1) + digits(6) + gap(1) + wave(2*halfH+1)
	//       + labels(1) + gap(1) + stats(1) + gap(1) + status(1) + gap(1)
	//       + footer(2)
	fixed := 1 + 1 + 6 + 1 + 1 + 1 + 1 + 1 + 1 + 1 + 2
	for termH-fixed-2*halfH < 0 && halfH > 1 {
		halfH--
	}

	narrow := contentW < 60

	// Launch animation progress in [0,1]; numbers count up, wave fills out.
	launchT := 1.0
	if !m.launchDone && !m.noAnim {
		elapsed := time.Since(m.launchStart).Seconds()
		launchT = easeOutCubic(math.Min(elapsed/0.6, 1.0))
	}

	lines := make([]string, 0, termH)

	// ── 1. Header ──
	lines = append(lines, renderHeroHeader(m, contentW))
	lines = append(lines, GapRow)

	// ── 2. Hero numbers ──
	// Captions use the same hue as their number, dimmed. Coloring both
	// number and caption ties them together and adds the color the screen
	// needs; leaving the caption grey made it read as unrelated chrome.
	downCap := theme.DownloadColor(0.45).Render("peak ") +
		theme.DownloadColor(0.7).Bold(true).Render(m.FormatBps(maxf(m.tracker.PeakDown, m.animDown)))
	upCap := theme.UploadColor(0.45).Render("session ") +
		theme.UploadColor(0.7).Bold(true).Render(formatBytes(m.tracker.TodayUp))

	downBlock := renderHeroDigits(m.animDown*launchT, m.bitsMode, true)
	upBlock := renderHeroDigits(m.animUp*launchT, m.bitsMode, false)
	// The unit + arrow live on the caption row, so every digit row keeps the
	// same width and the two blocks stay aligned.
	downBlock = append(downBlock, renderHeroCaption(m.animDown*launchT, m.bitsMode, true, downCap))
	upBlock = append(upBlock, renderHeroCaption(m.animUp*launchT, m.bitsMode, false, upCap))

	showDown := m.displayFilter != DisplayUpOnly
	showUp := m.displayFilter != DisplayDownOnly

	switch {
	case showDown && showUp && !narrow:
		lines = append(lines, joinCentered(downBlock, upBlock, contentW)...)
	case showDown && showUp && narrow:
		lines = append(lines, downBlock...)
		lines = append(lines, GapRow)
		lines = append(lines, upBlock...)
	case showDown:
		lines = append(lines, downBlock...)
	default:
		lines = append(lines, upBlock...)
	}
	lines = append(lines, GapRow)

	// ── 3. Mirrored waveform ──
	downSamples := m.windowedSamples(m.downHist)
	upSamples := m.windowedSamples(m.upHist)
	// Display-side EMA smoothing (raw samples stay raw for peaks/JSON/totals).
	downSamples = render.Smooth(downSamples, m.cfg.Graph.Smoothing)
	upSamples = render.Smooth(upSamples, m.cfg.Graph.Smoothing)
	downCeil := graphScale(downSamples, maxf(m.rollingMaxDown, m.animDown))
	upCeil := graphScale(upSamples, maxf(m.rollingMaxUp, m.animUp))
	expo := scaleExponent(m.scaleMode)
	if launchT < 1.0 {
		downSamples = scaleSamples(downSamples, launchT)
		upSamples = scaleSamples(upSamples, launchT)
	}
	now := time.Now()
	if !m.nowOverride.IsZero() {
		now = m.nowOverride
	}
	// Normalize against the ceiling here so the peak-hold line and the fill
	// share the exact same mapping.
	normDown := normalizeSamples(downSamples, downCeil, expo)
	normUp := normalizeSamples(upSamples, upCeil, expo)
	waveLines := renderMirroredWave(
		normDown, normUp, contentW, halfH,
		renderOpts{
			now: now, filter: m.displayFilter,
			rippleDown: m.rippleDownAt, rippleUp: m.rippleUpAt, noAnim: m.noAnim,
			scaleMode: m.scaleMode, grid: m.showGrid && m.cfg.GridlinesEnabled(),
			glyphs: m.glyphSet, colorCap: m.colorCap, gradient: m.cfg.GradientEnabled(),
		},
	)
	lines = append(lines, waveLines...)
	windowSecs := m.windowSecs
	if windowSecs <= 0 {
		windowSecs = 60
	}
	lines = append(lines, renderTimeLabels(contentW, windowSecs))
	lines = append(lines, GapRow)

	// ── 4. Separator, then every summary stat below it ──
	// A single hairline divides the live graph from the readouts, so the eye
	// stops treating the numbers as part of the waveform.
	lines = append(lines, theme.TextDim().Render(strings.Repeat("─", contentW)))

	// ── 5. Stats row ──
	lines = append(lines, renderHeroStats(m, contentW))
	lines = append(lines, GapRow)

	// ── 6. Status line ──
	// The header already shows interface and latency, so this row reports only
	// deviations from the defaults. Emitting "wlan0" here repeated the header
	// verbatim; emitting an empty row wasted a line.
	var statusParts []string
	if m.paused {
		statusParts = append(statusParts, theme.UploadColor(0.7).Render("paused"))
	}
	if m.bitsMode {
		statusParts = append(statusParts, theme.TextDim().Render("bits"))
	}
	switch m.displayFilter {
	case DisplayDownOnly:
		statusParts = append(statusParts, theme.TextDim().Render("↓ only"))
	case DisplayUpOnly:
		statusParts = append(statusParts, theme.TextDim().Render("↑ only"))
	}
	if m.refreshInterval != 100*time.Millisecond {
		statusParts = append(statusParts, theme.TextDim().Render("⟳ "+formatInterval(m.refreshInterval)))
	}
	if m.scaleMode != 0 {
		statusParts = append(statusParts, theme.TextDim().Render(scaleModeName(m.scaleMode)))
	}
	if m.toast != "" {
		// A transient toast borrows this row so the footer never moves.
		statusParts = []string{theme.Soft().Render(m.toast)}
	}
	if len(statusParts) > 0 {
		footerStyle := lipgloss.NewStyle().Width(contentW).Align(lipgloss.Center)
		lines = append(lines, footerStyle.Render(strings.Join(statusParts, " · ")))
		lines = append(lines, GapRow)
	}

	// ── 7. FROZEN FOOTER — byte-identical to v0.3.1, do not modify ──
	footerStyle := lipgloss.NewStyle().Width(contentW).Align(lipgloss.Center)
	renderKey := func(k, desc string) string {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.GetTextSoftColor())).
			Bold(true).
			Render(k) + " " + theme.Dim().Render(desc)
	}
	appHints := []string{
		renderKey("q", "quit"),
		renderKey("m", "mode"),
		renderKey("d", "filter"),
		renderKey("t", "theme"),
		renderKey("?", "help"),
	}
	linkHints := []string{
		renderKey("g", "github"),
		renderKey("u", "issues"),
		renderKey("x", "discuss"),
		renderKey("$", "sponsor"),
	}
	lines = append(lines, footerStyle.Render(strings.Join(appHints, " · ")))
	lines = append(lines, footerStyle.Render(strings.Join(linkHints, " · ")))

	_ = termW
	return lines
}

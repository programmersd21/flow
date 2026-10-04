package ui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/programmersd21/flow/internal/sparkline"
	"github.com/programmersd21/flow/internal/theme"
)

// ─── layout constants ────────────────────────────────────────────────────────

const (
	graphWindow      = 64
	heroMaxWidth     = 96
	compactMaxWidth  = 70
	miniMaxWidth     = 60
	HorizontalMargin = 4
	GapRow           = ""
)

// ─── small utilities ─────────────────────────────────────────────────────────

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func maxf(a, b float64) float64 {
	if math.IsNaN(a) {
		if math.IsNaN(b) {
			return 0
		}
		return b
	}
	if math.IsNaN(b) {
		return a
	}
	if a > b {
		return a
	}
	return b
}

func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-1]) + "…"
}

func formatInterval(d time.Duration) string {
	if d >= time.Second {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	return fmt.Sprintf("%dms", d.Milliseconds())
}

func formatBytes(b float64) string {
	if b < 0 || math.IsNaN(b) || math.IsInf(b, 0) {
		b = 0
	}
	const (
		KB = 1024.0
		MB = 1024 * KB
		GB = 1024 * MB
		TB = 1024 * GB
	)
	switch {
	case b >= TB:
		return fmt.Sprintf("%.2f TB", b/TB)
	case b >= GB:
		return fmt.Sprintf("%.1f GB", b/GB)
	case b >= MB:
		return fmt.Sprintf("%.1f MB", b/MB)
	case b >= KB:
		return fmt.Sprintf("%.0f KB", b/KB)
	default:
		return fmt.Sprintf("%.0f B", math.Max(0, b))
	}
}

func lineCount(lines []string) int {
	return strings.Count(strings.Join(lines, "\n"), "\n") + 1
}

// ─── layout helpers ───────────────────────────────────────────────────────────

// centerFrame vertically and horizontally centers `content` inside a terminal
// of the given width/height, padding with blank lines top and bottom.
func centerFrame(content string, width, height int) string {
	contentLines := strings.Split(content, "\n")
	frameH := len(contentLines)
	top := (height - frameH) / 2
	if top < 0 {
		top = 0
	}
	out := make([]string, 0, height)
	for i := 0; i < top; i++ {
		out = append(out, "")
	}
	for _, line := range contentLines {
		out = append(out, centerInline(line, width))
	}
	for len(out) < height {
		out = append(out, "")
	}
	if height > 0 && len(out) > height {
		out = out[:height]
	}
	return strings.Join(out, "\n")
}

// centerInline horizontally centers a single rendered string (respects ANSI).
func centerInline(s string, width int) string {
	if width <= 0 || s == "" {
		return s
	}
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return strings.Repeat(" ", (width-w)/2) + s
}

func spread(left, right string, width int) string {
	lw := lipgloss.Width(left)
	rw := lipgloss.Width(right)
	gap := width - lw - rw
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}

// TitleRow is used by compact/mini/tiny modes.
func TitleRow(_ float64) string {
	name := theme.Title().Render("flow")
	version := theme.Dim().Render("v" + strings.TrimPrefix(appVersion, "v"))
	return name + "  " + version
}

// ─── graph rendering ─────────────────────────────────────────────────────────

// renderColoredGraph renders a Braille area graph with a top-to-bottom intensity
// gradient, brightening toward the top.
func renderColoredGraph(samples []float64, width, height int, maxVal float64, frac float64, download bool) string {
	if len(samples) == 0 || width <= 0 || height <= 0 {
		return ""
	}

	lines := sparkline.RenderBraille(samples, width, height, maxVal, frac)
	if len(lines) == 0 {
		return ""
	}

	colored := make([]string, len(lines))
	speedRatio := 0.0
	if len(samples) > 0 && maxVal > 0 {
		speedRatio = theme.SpeedRatio(samples[len(samples)-1], maxVal)
	}
	for i, line := range lines {
		rowPos := float64(i) / float64(len(lines))
		intensity := (0.55 + 0.45*speedRatio) * (1.0 - rowPos*0.45)
		var style lipgloss.Style
		if download {
			style = theme.DownloadColor(intensity)
		} else {
			style = theme.UploadColor(intensity)
		}
		colored[i] = style.Render(line)
	}
	return strings.Join(colored, "\n")
}

// ─── panel: the core visual unit ─────────────────────────────────────────────

// renderPanel builds a single download or upload panel with a btop-style
// embedded header top-border and rounded frame.
func renderPanel(label, valueStr, peak string, peakPulse float64, graph string, width int, borderColor lipgloss.Color, mode ViewMode) string {
	boxWidth := width
	if boxWidth < 12 {
		boxWidth = 12
	}
	innerWidth := boxWidth - 2 // space between ╭ and ╮

	borderStyle := lipgloss.NewStyle().Foreground(borderColor)

	leftHdr := " " + label + "  " + valueStr + " "
	rightHdr := " peak " + peak + " "

	leftLen := lipgloss.Width(leftHdr)
	rightLen := lipgloss.Width(rightHdr)

	fillLen := innerWidth - leftLen - rightLen
	if fillLen < 1 {
		fillLen = 1
	}

	topLine := borderStyle.Render("╭") + leftHdr + borderStyle.Render(strings.Repeat("─", fillLen)) + rightHdr + borderStyle.Render("╮")
	bottomLine := borderStyle.Render("╰" + strings.Repeat("─", innerWidth) + "╯")
	pipe := borderStyle.Render("│")

	var panelLines []string
	panelLines = append(panelLines, topLine)

	switch mode {
	case ViewMini:
		for _, gl := range strings.Split(graph, "\n") {
			panelLines = append(panelLines, pipe+" "+gl+" "+pipe)
		}
	case ViewHero:
		for _, gl := range strings.Split(graph, "\n") {
			panelLines = append(panelLines, pipe+" "+gl+" "+pipe)
		}
	}

	panelLines = append(panelLines, bottomLine)
	return strings.Join(panelLines, "\n")
}

func renderMetricValue(value, trend string, ratio float64, download bool) string {
	valueStyle := theme.ValuePrimary(ratio, download)
	return valueStyle.Render(theme.DirArrow(download)+" "+value) + " " + theme.Dim().Render(trend)
}

func renderCompactRow(label, value, trend, peak string, peakPulse, ratio float64, width int, railColor lipgloss.Color, download bool) string {
	if width < 1 {
		width = 1
	}

	labelText := label
	labelWidth := 8
	if width < 48 {
		switch label {
		case "download":
			labelText = "down"
			labelWidth = 4
		case "upload":
			labelText = "up"
			labelWidth = 2
		}
	}

	available := width - (12 + labelWidth + lipgloss.Width(trend))
	if available < 2 {
		available = 2
	}
	valueWidth := available / 2
	peakWidth := available - valueWidth
	if valueWidth < 1 {
		valueWidth = 1
	}
	if peakWidth < 1 {
		peakWidth = 1
	}
	value = truncate(value, valueWidth)
	peak = truncate(peak, peakWidth)

	valueStyle := theme.ValuePrimary(ratio, download)
	rail := lipgloss.NewStyle().Foreground(railColor).Bold(true).Render("▌")
	left := rail + " " + theme.Label().Render(fmt.Sprintf("%-*s", labelWidth, labelText)) + " " +
		valueStyle.Render(theme.DirArrow(download)+" "+value) + " " + theme.Dim().Render(trend)
	right := theme.Dim().Render("peak ") + theme.PeakColor(peakPulse).Bold(true).Render(peak)
	row := spread(left, right, width)
	return lipgloss.NewStyle().MaxWidth(width).Render(row)
}

// ─── tiny (single-line) mode ─────────────────────────────────────────────────

func renderTiny(m Model) string {
	downRatio := theme.SpeedRatio(m.animDown, m.rollingMaxDown)
	upRatio := theme.SpeedRatio(m.animUp, m.rollingMaxUp)
	downStr := theme.DownloadColor(downRatio).Render("↓") + " " +
		theme.ValuePrimary(downRatio, true).Render(m.FormatBps(m.animDown))
	upStr := theme.UploadColor(upRatio).Render("↑") + " " +
		theme.ValuePrimary(upRatio, false).Render(m.FormatBps(m.animUp))
	w, h := m.width, m.height
	if w <= 0 {
		w = 80
	}
	if h <= 0 {
		h = 24
	}
	var out string
	switch m.displayFilter {
	case DisplayDownOnly:
		out = downStr
	case DisplayUpOnly:
		out = upStr
	default:
		out = downStr + "   " + upStr
	}
	return centerFrame(out, w, h)
}

// ─── stats line (ping) ────────────────────────────────────────────────────────

func renderStatsLine(m Model) string {
	if m.pingLatency <= 0 {
		return ""
	}
	ms := m.pingLatency.Seconds() * 1000
	return theme.Dim().Render("ping ") + latencyStyle(ms).Render(fmt.Sprintf("%.0fms", ms))
}

// ─── overlays ────────────────────────────────────────────────────────────────

func renderHelp(m Model) string {
	type item struct{ key, desc string }
	type group struct {
		title string
		items []item
	}

	groups := []group{
		{"navigation", []item{
			{"q", "quit"},
			{"m", "cycle view mode"},
			{"esc", "close overlay"},
			{"?", "toggle help"},
		}},
		{"display", []item{
			{"d", "filter both/down/up"},
			{"t", "theme picker"},
			{"S", "scale auto/linear/sqrt"},
			{"G", "toggle gridlines"},
			{"c", "cycle unit scale"},
			{"b", "bits / bytes"},
		}},
		{"data", []item{
			{"i", "cycle interface"},
			{"I", "interface details"},
			{"n", "network processes"},
			{"w", "cycle time window"},
			{"+ / -", "refresh rate"},
			{"p", "pause / resume"},
			{"r", "reset peaks (twice)"},
		}},
		{"export", []item{
			{"s", "snapshot .ansi/.txt"},
		}},
		{"links", []item{
			{"g", "github repo"},
			{"u", "issues"},
			{"x", "discussions"},
			{"$ / v", "sponsor / donate"},
		}},
	}

	w, h := m.width, m.height
	if w <= 0 {
		w = 80
	}
	if h <= 0 {
		h = 24
	}

	// Box budget: border (2) + horizontal padding (2 each side). The content
	// area is capped so the box never touches the terminal edges and never
	// exceeds the terminal width.
	boxW := w - 4
	if boxW > 96 {
		boxW = 96
	}
	if boxW < 24 {
		boxW = w - 2
		if boxW < 20 {
			boxW = 20
		}
	}
	contentW := boxW - 2 - 4 // border + padding
	if contentW < 20 {
		contentW = 20
	}

	// Columns: two only when both blocks fit with full descriptions.
	// One column needs ~34 cells (indent + key + longest desc); two need
	// ~71 including the gap. Below that a single scrollable column wins
	// over truncated two-column text.
	const needTwo = 34*2 + 3
	cols := 1
	if contentW >= needTwo {
		cols = 2
	}
	colWidth := contentW
	if cols == 2 {
		colWidth = (contentW - 3) / 2
	}

	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.GetAccentColor())).Bold(true)

	// renderRow builds one binding line, truncating the description (never
	// the key) so the line is exactly <= colWidth.
	renderRow := func(key, desc string) string {
		k := keyStyle.Render(fmt.Sprintf("%-7s", key))
		// 2 indent + 7 key + 1 space = 10 cells of chrome.
		avail := colWidth - 10
		if avail < 6 {
			avail = 6
		}
		d := desc
		if lipgloss.Width(d) > avail {
			d = truncate(d, avail)
		}
		line := "  " + k + " " + theme.Muted().Render(d)
		// Hard guarantee: never exceed colWidth even with wide runes.
		for lipgloss.Width(line) > colWidth && len(d) > 0 {
			d = truncate(d, len([]rune(d))-1)
			line = "  " + k + " " + theme.Muted().Render(d)
		}
		return line
	}

	// Build individual group blocks so columns can be perfectly balanced.
	type block struct {
		title string
		lines []string
	}
	var blocks []block
	for _, g := range groups {
		var blines []string
		blines = append(blines, theme.Dim().Render(g.title))
		for _, it := range g.items {
			blines = append(blines, renderRow(it.key, it.desc))
		}
		blocks = append(blocks, block{title: g.title, lines: blines})
	}

	// Distribute blocks into left and right columns aiming for equal height.
	var leftBlocks, rightBlocks []block
	leftLines, rightLines := 0, 0
	for _, b := range blocks {
		if leftLines <= rightLines {
			leftBlocks = append(leftBlocks, b)
			leftLines += len(b.lines) + 1
		} else {
			rightBlocks = append(rightBlocks, b)
			rightLines += len(b.lines) + 1
		}
	}

	flattenBlocks := func(bs []block) []string {
		var out []string
		for bi, b := range bs {
			out = append(out, b.lines...)
			if bi < len(bs)-1 {
				out = append(out, "")
			}
		}
		return out
	}

	left := flattenBlocks(leftBlocks)
	var right []string
	if cols == 2 {
		right = flattenBlocks(rightBlocks)
	}

	// Zip columns into full-width body lines with equal height.
	var body []string
	if cols == 1 {
		body = left
	} else {
		n := max(len(left), len(right))
		for i := 0; i < n; i++ {
			l, r := "", ""
			if i < len(left) {
				l = left[i]
			}
			if i < len(right) {
				r = right[i]
			}
			if r == "" {
				body = append(body, l)
				continue
			}
			pad := colWidth - lipgloss.Width(l)
			if pad < 3 {
				pad = 3
			}
			body = append(body, l+strings.Repeat(" ", pad)+r)
		}
	}

	title := theme.Title().Bold(true).Render("flow") + "  " + theme.Dim().Render("keyboard shortcuts")

	// Scroll window: title(1) + blank(1) + body + blank(1) + footer(1)
	// inside a bordered box => chrome ≈ 2 border + 4 inner = 6.
	visible := h - 6 - 4
	if visible < 3 {
		visible = 3
	}
	if visible > len(body) {
		visible = len(body)
	}
	start := m.helpScroll
	if start < 0 {
		start = 0
	}
	maxStart := len(body) - visible
	if maxStart < 0 {
		maxStart = 0
	}
	if start > maxStart {
		start = maxStart
	}
	window := body
	if len(body) > visible {
		window = body[start : start+visible]
	}

	moreUp := start > 0
	moreDown := start+visible < len(body)

	footer := "  " + theme.Dim().Render("j/k scroll · esc close")
	if moreUp && moreDown {
		footer = "  " + theme.Dim().Render("↑ more · ↓ more · j/k scroll · esc close")
	} else if moreUp {
		footer = "  " + theme.Dim().Render("↑ more · j/k scroll · esc close")
	} else if moreDown {
		footer = "  " + theme.Dim().Render("↓ more · j/k scroll · esc close")
	}
	if w < 52 {
		footer = "  " + theme.Dim().Render("j/k · esc")
	}

	rows := make([]string, 0, visible+5)
	rows = append(rows, "  "+title, "")
	rows = append(rows, window...)
	rows = append(rows, "", footer)

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(theme.GetBorderColor())).
		Padding(0, 3).
		MaxWidth(boxW).
		Render(strings.Join(rows, "\n"))
	return centerFrame(box, w, h)
}

func renderIfaceDetails(m Model) string {
	if m.ifaceDetails == nil {
		return renderHelp(m)
	}
	d := m.ifaceDetails
	title := theme.Title().Bold(true).Render(d.Name) + "  " + theme.Dim().Render("interface")
	var rows []string
	rows = append(rows, "", "  "+title, "")

	if d.HardwareAddr != "" && d.HardwareAddr != "00:00:00:00:00:00" {
		rows = append(rows, "  "+theme.Dim().Render("mac  ")+theme.Soft().Render(d.HardwareAddr))
	}
	for _, addr := range d.Addrs {
		rows = append(rows, "  "+theme.Dim().Render("addr ")+theme.Soft().Render(addr))
	}
	// Semantic Good/Bad for link state
	linkStyle := theme.GoodStyle()
	linkText := "up"
	if !d.IsUp {
		linkStyle = theme.BadStyle()
		linkText = "down"
	}
	rows = append(rows, "  "+theme.Dim().Render("link ")+linkStyle.Render(linkText))
	if d.Mtu > 0 {
		rows = append(rows, "  "+theme.Dim().Render("mtu  ")+theme.Soft().Render(fmt.Sprintf("%d", d.Mtu)))
	}
	rows = append(rows, "", "  "+theme.Dim().Render("esc  close"), "")

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(theme.GetBorderColor())).
		Padding(0, 2).
		Render(strings.Join(rows, "\n"))
	w, h := m.width, m.height
	if w <= 0 {
		w = 80
	}
	if h <= 0 {
		h = 24
	}
	return centerFrame(box, w, h)
}

func renderProcesses(m Model) string {
	w, h := m.width, m.height
	if w <= 0 {
		w = 80
	}
	if h <= 0 {
		h = 24
	}
	contentW := min(w-4, heroMaxWidth)
	if contentW < 40 {
		contentW = max(w-2, 40)
	}

	title := theme.Title().Bold(true).Render("network") + "  " + theme.Dim().Render("active processes")
	var rows []string
	rows = append(rows, "", "  "+title, "")

	if len(m.procs) == 0 {
		rows = append(rows, "  "+theme.Muted().Render("no network processes detected"))
	} else {
		maxRows := h - 10
		if maxRows < 5 {
			maxRows = 5
		}
		list := m.procs
		if len(list) > maxRows {
			list = list[:maxRows]
		}
		innerW := contentW - 8
		if innerW < 30 {
			innerW = 30
		}
		pidW := 6
		connW := 6
		nameW := innerW - pidW - connW - 4
		if nameW < 10 {
			nameW = 10
		}

		hdr := fmt.Sprintf("  %s  %s  %s",
			theme.Dim().Render(fmt.Sprintf("%-*s", pidW, "pid")),
			theme.Dim().Render(fmt.Sprintf("%-*s", nameW, "process")),
			theme.Dim().Render(fmt.Sprintf("%*s", connW, "conns")))
		sep := theme.Dim().Render(strings.Repeat("─", innerW+4))
		rows = append(rows, "  "+sep, hdr, "  "+sep)

		for _, p := range list {
			rows = append(rows, fmt.Sprintf("  %s  %s  %s",
				theme.Muted().Render(fmt.Sprintf("%-*d", pidW, p.PID)),
				theme.Soft().Render(fmt.Sprintf("%-*s", nameW, truncate(p.Name, nameW))),
				theme.Accent().Bold(true).Render(fmt.Sprintf("%*d", connW, p.Connections))))
		}
		rows = append(rows, "  "+sep)
	}
	rows = append(rows, "", "  "+theme.Dim().Render("esc  close"), "")

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(theme.GetBorderColor())).
		Padding(0, 2).
		Render(strings.Join(rows, "\n"))
	return centerFrame(box, w, h)
}

func renderThemes(m Model) string {
	w, h := m.width, m.height
	if w <= 0 {
		w = 80
	}
	if h <= 0 {
		h = 24
	}

	title := theme.Title().Bold(true).Render("themes") + "  " + theme.Dim().Render("pick a palette")
	themes := theme.ListThemes()

	// The box is what constrains a line, not the terminal: box = border(2) +
	// padding(6), and it is capped at 80 columns.
	boxW := min(w-4, 80)
	if boxW < 24 {
		boxW = max(w-2, 24)
	}
	innerW := boxW - 8

	// Row budget inside the bordered box:
	//   2 border + 2 title block + 2 footer block + 2 indicators = 8 rows of chrome.
	const chromeRows = 9
	budget := h - chromeRows
	if budget < 3 {
		budget = 3
	}
	visibleCount := min(budget, len(themes))
	if visibleCount < 1 {
		visibleCount = 1
	}

	showIndicators := h >= chromeRows+visibleCount
	startIdx := m.themeSelectionIdx - visibleCount/2
	if startIdx < 0 {
		startIdx = 0
	}
	if startIdx+visibleCount > len(themes) {
		startIdx = len(themes) - visibleCount
	}
	if startIdx < 0 {
		startIdx = 0
	}
	endIdx := startIdx + visibleCount

	accentColor := lipgloss.Color(theme.GetAccentColor())

	// One common name column width for the whole list, so descriptions form a
	// single straight column.
	nameColW := 0
	for _, t := range themes {
		if w := lipgloss.Width(t.Name); w > nameColW {
			nameColW = w
		}
	}
	if nameColW > 14 {
		nameColW = 14
	}

	var rows []string
	rows = append(rows, "  "+title, "")
	if showIndicators && startIdx > 0 {
		rows = append(rows, "  "+theme.Dim().Render("↑ more"))
	}
	for i := startIdx; i < endIdx; i++ {
		t := themes[i]
		selected := i == m.themeSelectionIdx

		// Color swatches: two ● dots showing the theme's download + upload hues.
		downHex, upHex := theme.ThemeSwatches(t.Name)
		var swatch string
		if downHex != "" && upHex != "" {
			dotDown := lipgloss.NewStyle().Foreground(lipgloss.Color(downHex)).Render("●")
			dotUp := lipgloss.NewStyle().Foreground(lipgloss.Color(upHex)).Render("●")
			swatch = dotDown + " " + dotUp
		} else {
			// Fallback for missing swatch info
			swatch = "   "
		}

		var cursor, nameStr string
		if selected {
			cursor = lipgloss.NewStyle().Foreground(accentColor).Bold(true).Render("›")
			nameStr = theme.Soft().Bold(true).Render(fmt.Sprintf("%-*s", nameColW, t.Name))
		} else {
			cursor = " "
			nameStr = theme.Muted().Render(fmt.Sprintf("%-*s", nameColW, t.Name))
		}

		// Layout: "  cursor swatch  name"
		line := "  " + cursor + " " + swatch + "  " + nameStr
		// Only append the description when it provably fits.
		if t.Description != "" {
			prefixW := 2 + 1 + 1 + 3 + 2 + nameColW + 2
			if avail := innerW - prefixW; avail >= 10 {
				desc := t.Description
				if lipgloss.Width(desc) > avail {
					desc = truncate(desc, avail)
				}
				line += "  " + descStyleFor(t, selected).Render(desc)
			}
		}
		rows = append(rows, line)
	}
	if showIndicators && endIdx < len(themes) {
		rows = append(rows, "  "+theme.Dim().Render("↓ more"))
	}
	// The hint must fit the box on its own line.
	hint := "j/k  navigate   enter  apply   esc  cancel"
	if innerW < lipgloss.Width(hint)+2 {
		hint = "esc  cancel"
	}
	rows = append(rows, "", "  "+theme.Dim().Render(hint))

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(theme.GetBorderColor())).
		Padding(0, 3).
		MaxWidth(boxW).
		Render(strings.Join(rows, "\n"))
	return centerFrame(box, w, h)
}

// descStyleFor picks the description style for a theme row: dim normally,
// mid when selected.
func descStyleFor(_ theme.ThemeInfo, selected bool) lipgloss.Style {
	if selected {
		return theme.Muted()
	}
	return theme.Dim()
}

// ─── main dashboard content ───────────────────────────────────────────────────

func pickViewModeAndContent(m Model) (ViewMode, []string) {
	if m.viewMode == ViewTiny {
		return ViewTiny, nil
	}
	if m.viewMode == ViewMini {
		return ViewMini, dashboardContentLines(m, ViewMini)
	}
	if m.viewMode == ViewCompact {
		return ViewCompact, dashboardContentLines(m, ViewCompact)
	}
	if m.width > 0 && m.width < 40 {
		return ViewTiny, nil
	}
	if m.height > 0 && m.height < 6 {
		return ViewTiny, nil
	}
	candidates := []ViewMode{ViewHero, ViewCompact, ViewMini}
	if m.width > 0 && m.width < 60 {
		candidates = []ViewMode{ViewCompact, ViewMini}
	}
	if m.height > 0 {
		for _, mode := range candidates {
			lines := dashboardContentLines(m, mode)
			if lineCount(lines) <= m.height {
				return mode, lines
			}
		}
		return ViewTiny, nil
	}
	return candidates[0], dashboardContentLines(m, candidates[0])
}

func dashboardContentLines(m Model, mode ViewMode) []string {
	termW := m.width
	if termW <= 0 {
		termW = 80
	}
	termH := m.height
	if termH <= 0 {
		termH = 24
	}

	// Content width: hero gets more space than compact/mini
	var contentW int
	switch mode {
	case ViewCompact, ViewMini:
		contentW = min(termW-HorizontalMargin, compactMaxWidth)
	default:
		contentW = min(termW-HorizontalMargin, heroMaxWidth)
	}
	if contentW < 40 {
		contentW = max(termW-2, 40)
	}

	// Hero uses big digits + mirrored waveform; mini uses the same mirrored
	// waveform with one-line numbers. Both share one renderer so the modes
	// read as one app instead of two designs.
	if mode == ViewHero {
		return heroContentLines(m, contentW, termW, termH)
	}
	if mode == ViewMini {
		return miniContentLines(m, contentW, termH)
	}

	// Graph inner width (panel border + padding eats 4 chars)
	graphW := contentW - 4
	if graphW < 10 {
		graphW = 10
	}

	downRatio := theme.SpeedRatio(m.animDown, maxf(m.rollingMaxDown, m.animDown))
	upRatio := theme.SpeedRatio(m.animUp, maxf(m.rollingMaxUp, m.animUp))
	downSamples := m.downHist.Slice()
	upSamples := m.upHist.Slice()
	downTrend := sparkline.VelocityGlyph(downSamples, slopeWindow)
	upTrend := sparkline.VelocityGlyph(upSamples, slopeWindow)

	// Fractional scroll offset for smooth graph animation
	frac := 0.0
	if !m.paused && m.refreshInterval > 0 {
		elapsed := time.Since(m.lastSampleTime).Seconds()
		interval := m.refreshInterval.Seconds()
		if interval > 0 {
			frac = elapsed / interval
		}
		if frac < 0 || math.IsNaN(frac) || math.IsInf(frac, 0) {
			frac = 0
		}
		if frac > 1 {
			frac = 1
		}
	}

	graphHeight := 5
	if mode == ViewMini {
		graphHeight = 3
	}

	var downGraph, upGraph string
	if mode != ViewCompact {
		downGraph = renderColoredGraph(downSamples, graphW, graphHeight, maxf(m.rollingMaxDown, m.animDown), frac, true)
		upGraph = renderColoredGraph(upSamples, graphW, graphHeight, maxf(m.rollingMaxUp, m.animUp), frac, false)
	}

	downBorderColor := theme.DownloadBorderColor(downRatio)
	upBorderColor := theme.UploadBorderColor(upRatio)

	lines := make([]string, 0, 28)

	// ── Header ────────────────────────────────────────────────────────────────

	lines = append(lines, TitleRow(m.samplePulse))
	lines = append(lines, GapRow)

	// ── Metrics ───────────────────────────────────────────────────────────────

	downSpeed := m.FormatBps(m.animDown)
	upSpeed := m.FormatBps(m.animUp)
	peakDown := m.FormatBps(m.tracker.PeakDown)
	peakUp := m.FormatBps(m.tracker.PeakUp)

	renderMetric := func(label, value, trend, peak string, pulse, ratio float64, graph string, color lipgloss.Color, download bool) string {
		if mode == ViewCompact {
			return renderCompactRow(label, value, trend, peak, pulse, ratio, contentW, color, download)
		}
		return renderPanel(label, renderMetricValue(value, trend, ratio, download), peak, pulse, graph, contentW, color, mode)
	}

	switch m.displayFilter {
	case DisplayBoth:
		lines = append(lines, renderMetric("download", downSpeed, downTrend, peakDown, m.downPulse, downRatio, downGraph, downBorderColor, true))
		lines = append(lines, GapRow)
		lines = append(lines, renderMetric("upload", upSpeed, upTrend, peakUp, m.upPulse, upRatio, upGraph, upBorderColor, false))
	case DisplayDownOnly:
		lines = append(lines, renderMetric("download", downSpeed, downTrend, peakDown, m.downPulse, downRatio, downGraph, downBorderColor, true))
	case DisplayUpOnly:
		lines = append(lines, renderMetric("upload", upSpeed, upTrend, peakUp, m.upPulse, upRatio, upGraph, upBorderColor, false))
	}

	// ── Footer (hero + compact only) ──────────────────────────────────────────

	if mode == ViewHero || mode == ViewCompact {

		// Session totals
		if (m.tracker.TodayDown > 0 || m.tracker.TodayUp > 0) && termH >= 20 {
			lines = append(lines, GapRow)
			var todayParts []string
			if m.displayFilter != DisplayUpOnly {
				todayParts = append(todayParts,
					theme.DownloadColor(0.6).Render("↓")+" "+
						theme.Soft().Bold(true).Render(formatBytes(m.tracker.TodayDown)))
			}
			if m.displayFilter != DisplayDownOnly {
				todayParts = append(todayParts,
					theme.UploadColor(0.6).Render("↑")+" "+
						theme.Soft().Bold(true).Render(formatBytes(m.tracker.TodayUp)))
			}
			todayLine := theme.Dim().Render("today  ") + strings.Join(todayParts, "   ")
			lines = append(lines, todayLine)
		}

		// Status line
		lines = append(lines, GapRow)
		statusParts := []string{theme.Muted().Render(m.ifaceName)}
		if m.paused {
			statusParts = append(statusParts, theme.Dim().Render("paused"))
		}
		if m.bitsMode {
			statusParts = append(statusParts, theme.Dim().Render("bits"))
		}
		switch m.displayFilter {
		case DisplayDownOnly:
			statusParts = append(statusParts, theme.Dim().Render("↓ only"))
		case DisplayUpOnly:
			statusParts = append(statusParts, theme.Dim().Render("↑ only"))
		}
		if m.refreshInterval != 100*time.Millisecond {
			statusParts = append(statusParts, theme.Dim().Render(formatInterval(m.refreshInterval)))
		}
		if sl := renderStatsLine(m); sl != "" {
			statusParts = append(statusParts, sl)
		}

		footerStyle := lipgloss.NewStyle().Width(contentW).Align(lipgloss.Center)
		lines = append(lines, footerStyle.Render(strings.Join(statusParts, " · ")))

		lines = append(lines, GapRow)
		renderKey := func(k, desc string) string {
			return lipgloss.NewStyle().
				Foreground(lipgloss.Color(theme.GetTextSoftColor())).
				Bold(true).
				Render(k) + " " + theme.Dim().Render(desc)
		}
		if mode == ViewCompact {
			lines = append(lines, footerStyle.Render(strings.Join([]string{
				renderKey("q", "quit"),
				renderKey("m", "mode"),
				renderKey("d", "filter"),
				renderKey("?", "help"),
			}, " · ")))
		} else {
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
		}
	}

	return lines
}

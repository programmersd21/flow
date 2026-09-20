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
	heroMaxWidth     = 86
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

// spread places left and right strings with a gap to fill exactly `width` chars.
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
	case ViewCompact:
		content := spread(leftHdr, rightHdr, innerWidth-2)
		panelLines = append(panelLines, pipe+" "+content+" "+pipe)
	default: // ViewHero
		for _, gl := range strings.Split(graph, "\n") {
			panelLines = append(panelLines, pipe+" "+gl+" "+pipe)
		}
	}

	panelLines = append(panelLines, bottomLine)
	return strings.Join(panelLines, "\n")
}

// ─── tiny (single-line) mode ─────────────────────────────────────────────────

func renderTiny(m Model) string {
	downRatio := theme.SpeedRatio(m.animDown, m.rollingMaxDown)
	upRatio := theme.SpeedRatio(m.animUp, m.rollingMaxUp)
	left := theme.DownloadColor(downRatio).Render("↓") + " " +
		theme.ValuePrimary(downRatio, true).Render(m.FormatBps(m.animDown))
	right := theme.UploadColor(upRatio).Render("↑") + " " +
		theme.ValuePrimary(upRatio, false).Render(m.FormatBps(m.animUp))
	w, h := m.width, m.height
	if w <= 0 {
		w = 80
	}
	if h <= 0 {
		h = 24
	}
	return centerFrame(left+"   "+right, w, h)
}

// ─── stats line (ping) ────────────────────────────────────────────────────────

func renderStatsLine(m Model) string {
	if m.pingLatency <= 0 {
		return ""
	}
	ms := m.pingLatency.Seconds() * 1000
	style := theme.Soft()
	if ms >= 100 {
		style = theme.Accent()
	}
	return theme.Dim().Render("ping ") + style.Bold(true).Render(fmt.Sprintf("%.0fms", ms))
}

// ─── overlays ────────────────────────────────────────────────────────────────

func renderHelp(m Model) string {
	type item struct{ key, desc string }
	items := []item{
		{"q", "quit"},
		{"m", "cycle view mode"},
		{"d", "cycle display filter"},
		{"n", "network processes"},
		{"t", "choose theme"},
		{"r", "reset peaks  (press twice)"},
		{"i", "cycle interface"},
		{"I", "interface details"},
		{"c", "cycle unit scale"},
		{"b", "toggle bits / bytes"},
		{"+ / -", "adjust refresh rate"},
		{"p", "pause / resume"},
		{"g", "open github repo"},
		{"u", "open issues"},
		{"x", "open discussions"},
		{"$ / v", "sponsor / donate"},
		{"?", "toggle help"},
	}

	title := theme.Title().Bold(true).Render("flow") + "  " + theme.Dim().Render("keyboard shortcuts")
	var rows []string
	rows = append(rows, "", "  "+title, "")
	for _, it := range items {
		k := lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.GetAccentColor())).
			Bold(true).
			Render(fmt.Sprintf("%-7s", it.key))
		d := theme.Muted().Render(it.desc)
		rows = append(rows, "  "+k+"  "+d)
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
	linkStyle := theme.Soft()
	linkText := "up"
	if !d.IsUp {
		linkStyle = theme.Accent()
		linkText = "down"
	}
	rows = append(rows, "  "+theme.Dim().Render("link ")+linkStyle.Bold(true).Render(linkText))
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

	// Scrollable window
	visibleCount := h - 10
	if visibleCount < 3 {
		visibleCount = 3
	}
	if visibleCount > len(themes) {
		visibleCount = len(themes)
	}
	startIdx := m.themeSelectionIdx - visibleCount/2
	if startIdx < 0 {
		startIdx = 0
	}
	if startIdx+visibleCount > len(themes) {
		startIdx = len(themes) - visibleCount
	}
	endIdx := startIdx + visibleCount

	var rows []string
	rows = append(rows, "", "  "+title, "")
	if startIdx > 0 {
		rows = append(rows, "  "+theme.Dim().Render("↑ more"))
	}
	for i := startIdx; i < endIdx; i++ {
		t := themes[i]
		cursor := "  "
		nameStyle := theme.Muted()
		descStyle := theme.Dim()
		if i == m.themeSelectionIdx {
			cursor = lipgloss.NewStyle().
				Foreground(lipgloss.Color(theme.GetAccentColor())).
				Bold(true).
				Render("› ")
			nameStyle = theme.Soft().Bold(true)
			descStyle = theme.Muted()
		}
		line := "  " + cursor + nameStyle.Render(t.Name)
		if t.Description != "" {
			line += "  " + descStyle.Render(t.Description)
		}
		rows = append(rows, line)
	}
	if endIdx < len(themes) {
		rows = append(rows, "  "+theme.Dim().Render("↓ more"))
	}
	rows = append(rows, "", "  "+theme.Dim().Render("j/k  navigate   enter  confirm   esc  cancel"), "")

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(theme.GetBorderColor())).
		Padding(0, 2).
		Render(strings.Join(rows, "\n"))
	return centerFrame(box, w, h)
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

	downVal := theme.DownloadColor(downRatio).Bold(true).Render(theme.DirArrow(true)) + " " +
		theme.ValuePrimary(downRatio, true).Render(m.FormatBps(m.animDown)) + " " +
		theme.Dim().Render(downTrend)
	upVal := theme.UploadColor(upRatio).Bold(true).Render(theme.DirArrow(false)) + " " +
		theme.ValuePrimary(upRatio, false).Render(m.FormatBps(m.animUp)) + " " +
		theme.Dim().Render(upTrend)
	peakDown := m.FormatBps(m.tracker.PeakDown)
	peakUp := m.FormatBps(m.tracker.PeakUp)

	switch m.displayFilter {
	case DisplayBoth:
		lines = append(lines, renderPanel("download", downVal, peakDown, m.downPulse, downGraph, contentW, downBorderColor, mode))
		lines = append(lines, GapRow)
		lines = append(lines, renderPanel("upload", upVal, peakUp, m.upPulse, upGraph, contentW, upBorderColor, mode))
	case DisplayDownOnly:
		lines = append(lines, renderPanel("download", downVal, peakDown, m.downPulse, downGraph, contentW, downBorderColor, mode))
	case DisplayUpOnly:
		lines = append(lines, renderPanel("upload", upVal, peakUp, m.upPulse, upGraph, contentW, upBorderColor, mode))
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

		// Key hints (split into 2 un-wrapped centered lines)
		lines = append(lines, GapRow)
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
	}

	return lines
}

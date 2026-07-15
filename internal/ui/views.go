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

const (
	graphWindow       = 64
	heroInnerMaxWidth = 80
	compactInnerMax   = 68
	PanelBorderWidth  = 2
	PanelPaddingX     = 2
	PanelExtraWidth   = PanelBorderWidth + 2*PanelPaddingX // 6
	HorizontalMargin  = 4
	GapRow            = ""
)

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func maxf(a, b float64) float64 {
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

func centerFrame(content string, width, height int) string {
	contentLines := strings.Split(content, "\n")
	if len(contentLines) == 0 {
		return content
	}
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

func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-1]) + "…"
}

func formatInterval(d time.Duration) string {
	if d >= time.Second {
		return fmt.Sprintf("every %ds", int(d.Seconds()))
	}
	return fmt.Sprintf("every %dms", d.Milliseconds())
}

func formatBytes(b float64) string {
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

func TitleRow(pulse float64) string {
	return fmt.Sprintf("%s  %s",
		theme.LogoDotColor(pulse).Render("●"),
		theme.Title().Render("flow"))
}

func renderTiny(m Model) string {
	downRatio := theme.SpeedRatio(m.animDown, m.rollingMaxDown)
	upRatio := theme.SpeedRatio(m.animUp, m.rollingMaxUp)
	left := theme.Label().Render("↓ download") + " " + theme.ValuePrimary(downRatio, true).Render(m.FormatBps(m.animDown))
	right := theme.Label().Render("↑ upload") + " " + theme.ValuePrimary(upRatio, false).Render(m.FormatBps(m.animUp))
	w := m.width
	if w <= 0 {
		w = 80
	}
	h := m.height
	if h <= 0 {
		h = 24
	}
	return centerFrame(left+"   "+right, w, h)
}

func renderStatsLine(m Model) string {
	if m.pingLatency > 0 {
		ms := m.pingLatency.Seconds() * 1000
		var color string
		switch {
		case ms < 30:
			color = "#10b981"
		case ms < 100:
			color = "#f59e0b"
		default:
			color = "#ef4444"
		}
		pingIcon := lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render("↔")
		pingVal := lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Bold(true).Render(fmt.Sprintf("%.0fms", ms))
		return theme.Muted().Render("ping ") + pingIcon + " " + pingVal
	}
	return ""
}

func renderPanel(title, value, peak string, peakPulse float64, graph string, width int, borderColor lipgloss.Color, mode ViewMode) string {
	innerW := width - PanelExtraWidth
	if innerW < 10 {
		innerW = 10
	}

	var panelLines []string
	panelLines = append(panelLines, theme.Label().Bold(true).Render(title))
	switch mode {
	case ViewMini:
		panelLines = append(panelLines, strings.Split(graph, "\n")...)
	case ViewCompact:
		gap := innerW - lipgloss.Width(value) - lipgloss.Width(fmt.Sprintf("peak: %s", peak))
		if gap < 2 {
			gap = 2
		}
		headerLine := value + strings.Repeat(" ", gap) + theme.Muted().Render("peak: ") + theme.PeakColor(peakPulse).Render(peak)
		panelLines = append(panelLines, headerLine)
	default:
		gap := innerW - lipgloss.Width(value) - lipgloss.Width(fmt.Sprintf("peak: %s", peak))
		if gap < 2 {
			gap = 2
		}
		headerLine := value + strings.Repeat(" ", gap) + theme.Muted().Render("peak: ") + theme.PeakColor(peakPulse).Render(peak)
		panelLines = append(panelLines, headerLine)
		panelLines = append(panelLines, GapRow)
		panelLines = append(panelLines, strings.Split(graph, "\n")...)
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, PanelPaddingX).
		Width(width).
		Render(strings.Join(panelLines, "\n"))
}

func renderColoredGraph(samples []float64, width, height int, maxVal float64, frac float64, download bool) string {
	lines := sparkline.RenderBraille(samples, width, height, maxVal, frac)
	coloredLines := make([]string, len(lines))
	for i, line := range lines {
		intensity := 1.0 - (float64(i) / float64(height))
		var speedRatio float64
		if len(samples) > 0 {
			speedRatio = theme.SpeedRatio(samples[len(samples)-1], maxVal)
		}
		intensity = (0.6 + 0.4*speedRatio) * intensity
		var style lipgloss.Style
		if download {
			style = theme.DownloadColor(intensity)
		} else {
			style = theme.UploadColor(intensity)
		}
		coloredLines[i] = style.Render(line)
	}
	return strings.Join(coloredLines, "\n")
}

func renderHelp(m Model) string {
	type item struct{ key, desc string }
	items := []item{
		{"q", "quit"},
		{"m", "cycle view mode"},
		{"n", "network processes"},
		{"t", "choose theme"},
		{"r", "reset peaks (press twice)"},
		{"i", "cycle interface"},
		{"I", "interface info"},
		{"c", "cycle units"},
		{"b", "toggle bits/bytes"},
		{"+ / -", "adjust refresh rate"},
		{"p", "pause / resume"},
		{"?", "open help"},
	}
	var lines []string
	for _, it := range items {
		k := theme.Soft().Bold(true).Render(it.key)
		lines = append(lines, "  "+k+"     "+theme.Muted().Render(it.desc))
	}
	title := theme.Label().Bold(true).Render("flow controls")
	var block []string
	block = append(block, "", "  "+title, "")
	block = append(block, lines...)
	block = append(block, "", "  "+theme.Dim().Render("press esc to return"), "")
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(theme.GetBorderColor())).
		Padding(0, 2).
		Render(strings.Join(block, "\n"))
	return centerFrame(box, m.width, m.height)
}

func renderIfaceDetails(m Model) string {
	if m.ifaceDetails == nil {
		return renderHelp(m)
	}
	d := m.ifaceDetails
	titleStyle := theme.Label().Bold(true)
	var block []string
	block = append(block, "", "  "+titleStyle.Render("interface: "+d.Name), "")
	if d.HardwareAddr != "" && d.HardwareAddr != "00:00:00:00:00:00" {
		block = append(block, "  "+theme.Muted().Render("mac  ")+theme.Soft().Render(d.HardwareAddr))
	}
	for _, addr := range d.Addrs {
		block = append(block, "  "+theme.Muted().Render("ip   ")+theme.Soft().Render(addr))
	}
	statusColor := "#10b981"
	statusText := "up"
	if !d.IsUp {
		statusColor = "#ef4444"
		statusText = "down"
	}
	block = append(block, "  "+theme.Muted().Render("link ")+lipgloss.NewStyle().Foreground(lipgloss.Color(statusColor)).Bold(true).Render(statusText))
	if d.Mtu > 0 {
		block = append(block, "  "+theme.Muted().Render("mtu  ")+theme.Soft().Render(fmt.Sprintf("%d", d.Mtu)))
	}
	block = append(block, "", "  "+theme.Dim().Render("press esc to return"), "")
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(theme.GetBorderColor())).
		Padding(0, 2).
		Render(strings.Join(block, "\n"))
	return centerFrame(box, m.width, m.height)
}

func renderProcesses(m Model) string {
	termW := m.width
	if termW <= 0 {
		termW = 80
	}
	termH := m.height
	if termH <= 0 {
		termH = 24
	}
	contentW := min(termW-4, heroInnerMaxWidth)
	if contentW < 40 {
		contentW = max(termW-2, 40)
	}
	titleStyle := theme.Label().Bold(true)
	var block []string
	block = append(block, "", "  "+titleStyle.Render("network processes"), "")
	if len(m.procs) == 0 {
		block = append(block, "  "+theme.Muted().Render("no active network processes detected"))
	} else {
		maxRows := termH - 10
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
		pidW := 7
		nameW := innerW - pidW - 10
		if nameW < 10 {
			nameW = 10
		}
		keyStyle := theme.Soft().Bold(true)
		muted := theme.Muted()
		header := fmt.Sprintf("  %s  %s  %s",
			keyStyle.Render(fmt.Sprintf("%-*s", pidW, "PID")),
			keyStyle.Render(fmt.Sprintf("%-*s", nameW, "Process")),
			keyStyle.Render(fmt.Sprintf("%*s", 7, "Conns")))
		sep := muted.Render(strings.Repeat("─", innerW+6))
		block = append(block, "  "+sep, header, "  "+sep)
		for _, p := range list {
			block = append(block, fmt.Sprintf("  %s  %s  %s",
				theme.Dim().Render(fmt.Sprintf("%-*d", pidW, p.PID)),
				theme.Soft().Render(fmt.Sprintf("%-*s", nameW, truncate(p.Name, nameW))),
				theme.Soft().Bold(true).Render(fmt.Sprintf("%*d", 7, p.Connections))))
		}
		block = append(block, "  "+sep)
	}
	block = append(block, "", "  "+theme.Dim().Render("press esc to return"), "")
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(theme.GetBorderColor())).
		Padding(0, 2).
		Render(strings.Join(block, "\n"))
	return centerFrame(box, termW, termH)
}

func renderThemes(m Model) string {
	termW := m.width
	if termW <= 0 {
		termW = 80
	}
	termH := m.height
	if termH <= 0 {
		termH = 24
	}
	titleStyle := theme.Label().Bold(true)
	themes := theme.ListThemes()
	var block []string
	block = append(block, "", "  "+titleStyle.Render("choose theme"), "")
	for i, t := range themes {
		cursor := "  "
		nameStyle := theme.Muted()
		descStyle := theme.Dim()
		if i == m.themeSelectionIdx {
			cursor = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.GetAccentColor())).Bold(true).Render("> ")
			nameStyle = theme.Soft().Bold(true)
			descStyle = theme.Soft()
		}
		block = append(block, "  "+cursor+nameStyle.Render(t.Name)+"  "+descStyle.Render(t.Description))
	}
	block = append(block, "", "  "+theme.Dim().Render("j/k navigate  enter confirm  esc cancel"), "")
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(theme.GetBorderColor())).
		Padding(0, 2).
		Render(strings.Join(block, "\n"))
	return centerFrame(box, termW, termH)
}

func dashboardLineCount(m Model, mode ViewMode) int {
	return lineCount(dashboardContentLines(m, mode))
}

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

	contentW := min(termW-HorizontalMargin, heroInnerMaxWidth)
	if mode == ViewCompact || mode == ViewMini {
		contentW = min(termW-HorizontalMargin, compactInnerMax)
	}
	if contentW < 40 {
		contentW = max(termW-2, 40)
	}

	innerW := contentW - PanelExtraWidth
	if innerW < 10 {
		innerW = 10
	}
	graphW := innerW

	downRatio := theme.SpeedRatio(m.animDown, maxf(m.rollingMaxDown, m.animDown))
	upRatio := theme.SpeedRatio(m.animUp, maxf(m.rollingMaxUp, m.animUp))
	downSamples := m.downHist.Slice()
	upSamples := m.upHist.Slice()
	downTrend := sparkline.VelocityGlyph(downSamples, slopeWindow)
	upTrend := sparkline.VelocityGlyph(upSamples, slopeWindow)

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

	graphHeight := 4
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
	lines := make([]string, 0, 24)

	switch mode {
	case ViewHero:
		if termH >= 28 {
			logo := theme.LogoColored(contentW)
			lines = append(lines, logo...)
			lines = append(lines, GapRow)
			lines = append(lines, theme.LogoSubtitle(contentW))
			lines = append(lines, GapRow)
		} else {
			lines = append(lines, TitleRow(m.samplePulse))
			lines = append(lines, GapRow)
		}
	case ViewCompact:
		lines = append(lines, TitleRow(m.samplePulse))
		lines = append(lines, GapRow)
	}

	downVal := theme.Muted().Render(theme.DirArrow(true)) + " " + theme.ValuePrimary(downRatio, true).Render(m.FormatBps(m.animDown)) + " " + theme.Muted().Render(downTrend)
	upVal := theme.Muted().Render(theme.DirArrow(false)) + " " + theme.ValuePrimary(upRatio, false).Render(m.FormatBps(m.animUp)) + " " + theme.Muted().Render(upTrend)
	peakDownVal := m.FormatBps(m.tracker.PeakDown)
	peakUpVal := m.FormatBps(m.tracker.PeakUp)

	lines = append(lines, renderPanel("download", downVal, peakDownVal, m.downPulse, downGraph, contentW, downBorderColor, mode))
	lines = append(lines, GapRow)
	lines = append(lines, renderPanel("upload", upVal, peakUpVal, m.upPulse, upGraph, contentW, upBorderColor, mode))

	if mode == ViewHero || mode == ViewCompact {
		if (m.tracker.TodayDown > 0 || m.tracker.TodayUp > 0) && termH >= 20 {
			lines = append(lines, GapRow)
			lines = append(lines, fmt.Sprintf("today  %s %s  %s %s",
				theme.DownloadColor(0.5).Render("↓"),
				theme.Soft().Render(formatBytes(m.tracker.TodayDown)),
				theme.UploadColor(0.5).Render("↑"),
				theme.Soft().Render(formatBytes(m.tracker.TodayUp))))
		}

		lines = append(lines, GapRow)
		dotColor := "#10b981"
		if m.paused {
			dotColor = "#ef4444"
		}
		ifaceStr := fmt.Sprintf("%s %s", lipgloss.NewStyle().Foreground(lipgloss.Color(dotColor)).Render("●"), theme.Muted().Render(m.ifaceName))
		if m.paused {
			ifaceStr += theme.Dim().Render("  paused")
		}
		if m.bitsMode {
			ifaceStr += theme.Dim().Render("  [bits]")
		}
		if m.refreshInterval != 100*time.Millisecond {
			ifaceStr += theme.Dim().Render("  " + formatInterval(m.refreshInterval))
		}

		footerStyle := lipgloss.NewStyle().Width(contentW).Align(lipgloss.Center)
		lines = append(lines, footerStyle.Render(ifaceStr))

		statsLine := renderStatsLine(m)
		if statsLine != "" && contentW >= 42 {
			lines = append(lines, GapRow)
			lines = append(lines, footerStyle.Render(statsLine))
		}

		lines = append(lines, GapRow)
		renderKey := func(k, desc string) string {
			return lipgloss.NewStyle().Foreground(lipgloss.Color(theme.GetTextSoftColor())).Bold(true).Render(k) + " " + theme.Dim().Render(desc)
		}
		hints := []string{
			renderKey("q", "quit"),
			renderKey("m", "mode"),
			renderKey("?", "help"),
		}
		lines = append(lines, footerStyle.Render(strings.Join(hints, " · ")))
	}
	return lines
}

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
)

func renderHero(m Model) string    { return renderDashboard(m, ViewHero) }
func renderCompact(m Model) string { return renderDashboard(m, ViewCompact) }
func renderMini(m Model) string    { return renderDashboard(m, ViewMini) }

func renderTiny(m Model) string {
	downRatio := theme.SpeedRatio(m.dispDown, m.rollingMaxDown)
	upRatio := theme.SpeedRatio(m.dispUp, m.rollingMaxUp)
	lbl := theme.Label()
	left := lbl.Render("↓ download") + " " + theme.ValuePrimary(downRatio, true).Render(m.FormatBps(m.dispDown))
	right := lbl.Render("↑ upload") + " " + theme.ValuePrimary(upRatio, false).Render(m.FormatBps(m.dispUp))
	line := left + "   " + right
	w := m.width
	if w <= 0 {
		w = 80
	}
	h := m.height
	if h <= 0 {
		h = 24
	}
	return centerFrame(line, w, h)
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

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.GetTextBrightColor())).
		Bold(true)

	title := titleStyle.Render("network processes")

	var block []string
	block = append(block, "")
	block = append(block, "  "+title)
	block = append(block, "")

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

		keyStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.GetAccentColor())).
			Bold(true)
		muted := theme.Muted()

		pidH := keyStyle.Render(fmt.Sprintf("%-*s", pidW, "PID"))
		nameH := keyStyle.Render(fmt.Sprintf("%-*s", nameW, "Process"))
		connH := keyStyle.Render(fmt.Sprintf("%*s", 7, "Conns"))
		header := fmt.Sprintf("  %s  %s  %s", pidH, nameH, connH)

		sep := muted.Render(strings.Repeat("─", innerW+6))

		block = append(block, "  "+sep)
		block = append(block, header)
		block = append(block, "  "+sep)

		for _, p := range list {
			pidS := muted.Render(fmt.Sprintf("%-*d", pidW, p.PID))
			nameS := theme.Accent().Render(fmt.Sprintf("%-*s", nameW, truncate(p.Name, nameW)))
			connS := keyStyle.Render(fmt.Sprintf("%*d", 7, p.Connections))
			block = append(block, fmt.Sprintf("  %s  %s  %s", pidS, nameS, connS))
		}
		block = append(block, "  "+sep)
	}

	block = append(block, "")
	block = append(block, "  "+theme.Dim().Render("press esc to return"))
	block = append(block, "")

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(theme.GetBorderColor())).
		Padding(0, 2).
		Render(strings.Join(block, "\n"))

	return centerFrame(box, termW, termH)
}

func truncate(s string, max int) string {
	w := lipgloss.Width(s)
	if w <= max {
		return s
	}
	runes := []rune(s)
	result := string(runes[:max-1]) + "…"
	return result
}

func renderHelp(m Model) string {
	type binding struct {
		key  string
		desc string
	}
	bindings := []binding{
		{"q", "quit"},
		{"m", "cycle view mode"},
		{"n", "network processes"},
		{"t", "choose theme"},
		{"r", "reset peaks"},
		{"i", "cycle interface"},
		{"c", "cycle units"},
		{"b", "toggle bits/bytes"},
		{"+ / -", "adjust refresh rate"},
		{"p", "pause / resume"},
		{"?", "toggle help"},
	}

	var keyLines []string
	for _, b := range bindings {
		kStr := lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.GetTextSoftColor())).
			Bold(true).
			Render(b.key)

		descStr := theme.Muted().Render(b.desc)
		keyLines = append(keyLines, "  "+kStr+"     "+descStr)
	}

	// Calculate maximum width
	maxKeyW := 0
	for _, line := range keyLines {
		if w := lipgloss.Width(line); w > maxKeyW {
			maxKeyW = w
		}
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.GetTextBrightColor())).
		Bold(true)

	title := titleStyle.Render("flow controls")

	var block []string
	block = append(block, "")
	block = append(block, "  "+title)
	block = append(block, "")
	block = append(block, keyLines...)
	block = append(block, "")
	block = append(block, "  "+theme.Dim().Render("press esc to return"))
	block = append(block, "")

	helpBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(theme.GetBorderColor())). // Modern Indigo border
		Padding(0, 2).
		Render(strings.Join(block, "\n"))

	return centerFrame(helpBox, m.width, m.height)
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

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.GetTextBrightColor())).
		Bold(true)

	title := titleStyle.Render("choose theme")

	themes := theme.ListThemes()

	var block []string
	block = append(block, "")
	block = append(block, "  "+title)
	block = append(block, "")

	for i, t := range themes {
		cursor := "  "
		nameStyle := theme.Muted()
		descStyle := theme.Dim()
		if i == m.themeSelectionIdx {
			cursor = lipgloss.NewStyle().
				Foreground(lipgloss.Color(theme.GetAccentColor())).
				Bold(true).
				Render("> ")
			nameStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(theme.GetTextBrightColor())).
				Bold(true)
			descStyle = theme.Soft()
		}
		line := "  " + cursor + nameStyle.Render(t.Name) + "  " + descStyle.Render(t.Description)
		block = append(block, line)
	}

	block = append(block, "")
	block = append(block, "  "+theme.Dim().Render("j/k navigate  enter confirm  esc cancel"))
	block = append(block, "")

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(theme.GetBorderColor())).
		Padding(0, 2).
		Render(strings.Join(block, "\n"))

	return centerFrame(box, termW, termH)
}

func TitleRow(breathe float64) string {
	dot := theme.LogoDotColor(breathe).Render("●")
	title := theme.Title().Render("flow")
	desc := theme.Muted().Render("bandwidth monitor")
	return fmt.Sprintf("%s  %s   %s", dot, title, desc)
}

func renderDashboard(m Model, mode ViewMode) string {
	termW := m.width
	if termW <= 0 {
		termW = 80
	}
	termH := m.height
	if termH <= 0 {
		termH = 24
	}

	contentW := min(termW-4, heroInnerMaxWidth)
	if mode == ViewCompact || mode == ViewMini {
		contentW = min(termW-4, compactInnerMax)
	}
	if contentW < 40 {
		contentW = max(termW-2, 40)
	}

	downRatio := theme.SpeedRatio(m.dispDown, maxf(m.rollingMaxDown, m.dispDown))
	upRatio := theme.SpeedRatio(m.dispUp, maxf(m.rollingMaxUp, m.dispUp))
	downPulse := m.downPulse
	upPulse := m.upPulse

	downSamples := m.downHist.Slice()
	upSamples := m.upHist.Slice()
	downTrend := sparkline.VelocityGlyph(downSamples, slopeWindow)
	upTrend := sparkline.VelocityGlyph(upSamples, slopeWindow)

	// Subtract 4 to account for border frame + internal padding
	// graphW must fit inside the panel's inner width (contentW - 4)
	graphW := min(contentW-4, graphWindow-4)
	if graphW < 10 {
		graphW = 10
	}

	// Compute fractional offset for smooth scrolling (30 FPS)
	frac := 0.0
	if !m.paused && m.refreshInterval > 0 {
		elapsed := time.Since(m.lastSampleTime).Seconds()
		interval := m.refreshInterval.Seconds()
		if interval > 0 {
			frac = elapsed / interval
		}
		// Clamp frac to valid range [0, 1] to prevent animation glitches
		if frac < 0 || math.IsNaN(frac) || math.IsInf(frac, 0) {
			frac = 0
		}
		if frac > 1 {
			frac = 1
		}
	}

	// Shorter graph for mini mode
	graphHeight := 4
	if mode == ViewMini {
		graphHeight = 3
	}

	// Render high-resolution Braille graphs with vertical gradients
	downGraph := renderColoredGraph(downSamples, graphW, graphHeight, maxf(m.rollingMaxDown, m.dispDown), frac, true)
	upGraph := renderColoredGraph(upSamples, graphW, graphHeight, maxf(m.rollingMaxUp, m.dispUp), frac, false)

	// Responsive TUI glowing borders
	downBorderColor := theme.DownloadBorderColor(downRatio)
	upBorderColor := theme.UploadBorderColor(upRatio)

	lines := make([]string, 0, 24)
	switch mode {
	case ViewHero:
		if termH >= 30 {
			logo := theme.LogoColored(contentW)
			lines = append(lines, logo...)
			lines = append(lines, "")
			sub := theme.LogoSubtitle(contentW)
			lines = append(lines, sub)
			lines = append(lines, "")
		} else {
			lines = append(lines, TitleRow(m.breathe))
			lines = append(lines, "")
		}
	case ViewCompact:
		lines = append(lines, TitleRow(m.breathe))
		lines = append(lines, "")
	}

	// Format download and upload speed values
	downVal := theme.Dim().Render(theme.DirArrow(true)) + " " + theme.ValuePrimary(downRatio, true).Render(m.FormatBps(m.dispDown)) + " " + theme.Muted().Render(downTrend)
	upVal := theme.Dim().Render(theme.DirArrow(false)) + " " + theme.ValuePrimary(upRatio, false).Render(m.FormatBps(m.dispUp)) + " " + theme.Muted().Render(upTrend)

	peakDownVal := m.FormatBps(m.tracker.PeakDown)
	peakUpVal := m.FormatBps(m.tracker.PeakUp)

	// Render Download Panel with border
	downPanel := renderPanel("download", downVal, peakDownVal, downPulse, downGraph, contentW, downBorderColor, mode == ViewMini)
	lines = append(lines, downPanel)
	lines = append(lines, "")

	// Render Upload Panel with border
	upPanel := renderPanel("upload", upVal, peakUpVal, upPulse, upGraph, contentW, upBorderColor, mode == ViewMini)
	lines = append(lines, upPanel)

	// Footer with interface status and keybinding hints (only for non-mini modes)
	if mode == ViewHero || mode == ViewCompact {
		hasTraffic := m.tracker.TodayDown > 0 || m.tracker.TodayUp > 0
		if hasTraffic && termH >= 24 {
			lines = append(lines, "")
			todayLine := fmt.Sprintf(
				"today  %s %s  %s %s",
				theme.DownloadColor(0.5).Render("↓"),
				theme.Accent().Render(formatBytes(m.tracker.TodayDown)),
				theme.UploadColor(0.5).Render("↑"),
				theme.Accent().Render(formatBytes(m.tracker.TodayUp)),
			)
			lines = append(lines, todayLine)
		}
		lines = append(lines, "")
		lines = append(lines, "")

		dotColor := "#10b981"
		if m.paused {
			dotColor = "#ef4444"
		}
		statusDot := lipgloss.NewStyle().Foreground(lipgloss.Color(dotColor)).Render("●")

		ifaceStr := fmt.Sprintf("%s %s", statusDot, theme.Accent().Render(m.ifaceName))
		if m.paused {
			ifaceStr += theme.Muted().Render("  paused")
		}

		renderKey := func(k, desc string) string {
			return lipgloss.NewStyle().Foreground(lipgloss.Color(theme.GetAccentColor())).Bold(true).Render(k) + " " + theme.Muted().Render(desc)
		}
		type keyHint struct{ key, desc string }
		var hints []keyHint
		switch {
		case contentW >= 78:
			hints = []keyHint{
				{"q", "quit"}, {"m", "mode"}, {"n", "proc"}, {"t", "theme"},
				{"r", "reset"}, {"i", "iface"}, {"c", "units"}, {"b", "bits"},
				{"+", "fast"}, {"-", "slow"}, {"p", "pause"}, {"?", "help"},
			}
		case contentW >= 55:
			hints = []keyHint{
				{"q", "quit"}, {"m", "mode"}, {"n", "proc"}, {"t", "theme"},
				{"p", "pause"}, {"?", "help"},
			}
		case contentW >= 42:
			hints = []keyHint{
				{"q", "quit"}, {"?", "help"},
			}
		}

		// Footer container style for precise centering
		footerStyle := lipgloss.NewStyle().Width(contentW).Align(lipgloss.Center)

		// 1. Interface
		lines = append(lines, footerStyle.Render(ifaceStr))

		// 2. Minimal stats (middle) — ping + bandwidth
		if contentW >= 42 {
			lines = append(lines, footerStyle.Render(renderStatsLine(m)))
		}

		if len(hints) > 0 {
			lines = append(lines, "")
			lines = append(lines, "")
			var hintParts []string
			for _, h := range hints {
				hintParts = append(hintParts, renderKey(h.key, h.desc))
			}
			hintStr := strings.Join(hintParts, " · ")
			lines = append(lines, footerStyle.Render(hintStr))
		}
	}

	content := strings.Join(lines, "\n")
	return centerFrame(content, termW, termH)
}

func renderPanel(title string, value string, peak string, peakPulse float64, graph string, width int, borderColor lipgloss.Color, isMini bool) string {
	innerW := width - 4 // border takes 2, padding takes 2
	if innerW < 20 {
		innerW = 20
	}

	leftPart := value

	// Typographic peak highlight using smooth color interpolation (no glitches or sparkle symbol)
	rightPart := fmt.Sprintf("peak: %s", theme.PeakColor(peakPulse).Render(peak))

	gap := innerW - lipgloss.Width(leftPart) - lipgloss.Width(rightPart)
	if gap < 2 {
		gap = 2
	}
	headerLine := leftPart + strings.Repeat(" ", gap) + rightPart

	// Muted section title
	titleLine := theme.Muted().Bold(true).Render(title)

	var panelLines []string
	panelLines = append(panelLines, titleLine)
	panelLines = append(panelLines, headerLine)
	if !isMini {
		panelLines = append(panelLines, "")
	}

	panelLines = append(panelLines, strings.Split(graph, "\n")...)

	panelContent := strings.Join(panelLines, "\n")

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1).
		Width(width)

	return borderStyle.Render(panelContent)
}

func (m Model) FormatBandwidthMeter() string {
	if m.err != nil {
		errStr := m.err.Error()
		if strings.Contains(errStr, "permission") || strings.Contains(errStr, "denied") {
			return lipgloss.JoinHorizontal(lipgloss.Left,
				theme.Dim().Render("↓"), " ", theme.Muted().Render("  no perm"), "   ",
				theme.Dim().Render("↑"), " ", theme.Muted().Render("  no perm"),
			)
		}
		if strings.Contains(errStr, "not found") || strings.Contains(errStr, "no usable") || strings.Contains(errStr, "interface") {
			return lipgloss.JoinHorizontal(lipgloss.Left,
				theme.Dim().Render("↓"), " ", theme.Muted().Render("   no dev"), "   ",
				theme.Dim().Render("↑"), " ", theme.Muted().Render("   no dev"),
			)
		}
		return lipgloss.JoinHorizontal(lipgloss.Left,
			theme.Dim().Render("↓"), " ", theme.Muted().Render("    error"), "   ",
			theme.Dim().Render("↑"), " ", theme.Muted().Render("    error"),
		)
	}

	downVal := FormatBpsFixedWidth(m.dispDown, m.unitMode, m.bitsMode)
	upVal := FormatBpsFixedWidth(m.dispUp, m.unitMode, m.bitsMode)

	downRatio := theme.SpeedRatio(m.dispDown, maxf(m.rollingMaxDown, m.dispDown))
	upRatio := theme.SpeedRatio(m.dispUp, maxf(m.rollingMaxUp, m.dispUp))

	downStyle := theme.ValuePrimary(downRatio, true)
	upStyle := theme.ValuePrimary(upRatio, false)

	return lipgloss.JoinHorizontal(lipgloss.Left,
		theme.Dim().Render("↓"), " ", downStyle.Render(downVal), "   ",
		theme.Dim().Render("↑"), " ", upStyle.Render(upVal),
	)
}

func renderStatsLine(m Model) string {
	pingStr := ""
	if m.pingLatency > 0 {
		ms := m.pingLatency.Seconds() * 1000
		var color string
		switch {
		case ms < 30:
			color = "#10b981" // green
		case ms < 100:
			color = "#f59e0b" // amber
		default:
			color = "#ef4444" // red
		}
		pingIcon := lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render("↔")
		pingVal := lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Bold(true).Render(fmt.Sprintf("%.0fms", ms))
		pingStr = pingIcon + " " + pingVal
	}

	downRatio := theme.SpeedRatio(m.dispDown, maxf(m.rollingMaxDown, m.dispDown))
	upRatio := theme.SpeedRatio(m.dispUp, maxf(m.rollingMaxUp, m.dispUp))
	downVal := theme.ValuePrimary(downRatio, true).Render(m.FormatBps(m.dispDown))
	upVal := theme.ValuePrimary(upRatio, false).Render(m.FormatBps(m.dispUp))

	down := theme.Dim().Render("↓") + " " + downVal
	up := theme.Dim().Render("↑") + " " + upVal

	if pingStr == "" {
		return down + "   " + up
	}
	return pingStr + "   " + down + "   " + up
}

func renderColoredGraph(samples []float64, width, height int, maxVal float64, frac float64, download bool) string {
	lines := sparkline.RenderBraille(samples, width, height, maxVal, frac)
	coloredLines := make([]string, len(lines))
	for i, line := range lines {
		// Row gradient: top row gets full color intensity, bottom gets base/idle intensity
		intensity := 1.0 - (float64(i) / float64(height))

		// Ensure graph remains colorful and bright even when idle, scaling up on high throughput
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

	// Center each line horizontally
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

	return strings.Join(out, "\n")
}

func centerInline(s string, width int) string {
	if width <= 0 {
		return s
	}
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return strings.Repeat(" ", (width-w)/2) + s
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
	if a > b {
		return a
	}
	return b
}

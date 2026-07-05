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
	return left + "   " + right
}

func renderHelp(m Model) string {
	type binding struct {
		key  string
		desc string
	}
	bindings := []binding{
		{"q", "quit"},
		{"m", "cycle view mode"},
		{"r", "reset peaks"},
		{"i", "cycle interface"},
		{"c", "cycle units"},
		{"p", "pause / resume"},
		{"?", "toggle help"},
	}

	var keyLines []string
	for _, b := range bindings {
		kStr := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#cbd5e1")). // slate 300
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
		Foreground(lipgloss.Color("#f8fafc")).
		Bold(true)

	title := titleStyle.Render("flow controls")

	var block []string
	block = append(block, "")
	block = append(block, "  "+title)
	block = append(block, "")
	block = append(block, keyLines...)
	block = append(block, "")
	block = append(block, "  "+theme.Dim().Render("press any key to return"))
	block = append(block, "")

	helpBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#6366f1")). // Modern Indigo border
		Padding(0, 2).
		Render(strings.Join(block, "\n"))

	return centerFrame(helpBox, m.width, m.height)
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

	// Footer stats and controls (only for non-mini modes)
	if mode == ViewHero || mode == ViewCompact {
		lines = append(lines, "")

		// Minimalist, high-end unicode today stats line
		todayLine := fmt.Sprintf(
			"today  %s  %s    today  %s  %s",
			lipgloss.NewStyle().Foreground(lipgloss.Color("#3b82f6")).Render("↓"),
			theme.Accent().Render(formatBytes(m.tracker.TodayDown)),
			lipgloss.NewStyle().Foreground(lipgloss.Color("#10b981")).Render("↑"),
			theme.Accent().Render(formatBytes(m.tracker.TodayUp)),
		)
		lines = append(lines, todayLine)
		lines = append(lines, "")

		// Minimalist navigation status line
		dotColor := "#10b981" // Active Green
		if m.paused {
			dotColor = "#ef4444" // Paused Red
		}
		statusDot := lipgloss.NewStyle().Foreground(lipgloss.Color(dotColor)).Render("●")

		ifaceStr := fmt.Sprintf("%s %s", statusDot, theme.Accent().Render(m.ifaceName))
		if m.paused {
			ifaceStr += theme.Muted().Render(" (paused)")
		}

		renderKey := func(k, desc string) string {
			return lipgloss.NewStyle().Foreground(lipgloss.Color("#a5b4fc")).Bold(true).Render(k) + " " + theme.Muted().Render(desc)
		}

		dotSep := theme.Dim().Render(" · ")

		var keys []string
		if contentW >= 75 {
			keys = []string{
				renderKey("q", "quit"),
				renderKey("m", "mode"),
				renderKey("r", "reset"),
				renderKey("i", "iface"),
				renderKey("c", "unit"),
				renderKey("p", "pause"),
				renderKey("?", "help"),
			}
		} else if contentW >= 55 {
			keys = []string{
				renderKey("q", "quit"),
				renderKey("m", "mode"),
				renderKey("r", "reset"),
				renderKey("p", "pause"),
				renderKey("?", "help"),
			}
		} else {
			keys = []string{
				renderKey("q", "quit"),
				renderKey("p", "pause"),
				renderKey("?", "help"),
			}
		}
		rightBar := strings.Join(keys, dotSep)

		leftBarW := lipgloss.Width(ifaceStr)
		rightBarW := lipgloss.Width(rightBar)
		gap := contentW - leftBarW - rightBarW

		var footerLine string
		if gap > 0 {
			footerLine = ifaceStr + strings.Repeat(" ", gap) + rightBar
		} else {
			footerLine = ifaceStr + "   " + rightBar
		}

		lines = append(lines, footerLine)
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

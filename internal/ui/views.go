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

func renderHero(m Model) string    { return renderDashboard(m, true) }
func renderCompact(m Model) string { return renderDashboard(m, false) }

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
		{"r", "reset peaks"},
		{"i", "cycle interface"},
		{"c", "cycle units"},
		{"p", "pause / resume"},
		{"?", "toggle help"},
	}

	// Build key binding lines — all padded to the same visual width
	var keyLines []string
	for _, b := range bindings {
		keyLines = append(keyLines, theme.Accent().Bold(true).Render(b.key)+"     "+theme.Muted().Render(b.desc))
	}
	maxKeyW := 0
	for _, line := range keyLines {
		if w := lipgloss.Width(line); w > maxKeyW {
			maxKeyW = w
		}
	}
	for i, line := range keyLines {
		if w := lipgloss.Width(line); w < maxKeyW {
			keyLines[i] = line + strings.Repeat(" ", maxKeyW-w)
		}
	}

	// Separator matches the key binding width for visual consistency
	sep := theme.Dim().Render(strings.Repeat("─", maxKeyW))

	// Assemble raw block — every line padded to center within maxKeyW
	titleFlow := theme.Title().Render("flow")
	titleCtrl := theme.Muted().Render("controls")
	titleFooter := theme.Soft().Render("press any key to return")

	flowPad := max(0, (maxKeyW-lipgloss.Width(titleFlow))/2)
	ctrlPad := max(0, (maxKeyW-lipgloss.Width(titleCtrl))/2)
	footPad := max(0, (maxKeyW-lipgloss.Width(titleFooter))/2)

	var block []string
	block = append(block, strings.Repeat(" ", flowPad)+titleFlow+strings.Repeat(" ", max(0, maxKeyW-flowPad-lipgloss.Width(titleFlow))))
	block = append(block, strings.Repeat(" ", ctrlPad)+titleCtrl+strings.Repeat(" ", max(0, maxKeyW-ctrlPad-lipgloss.Width(titleCtrl))))
	block = append(block, sep)
	block = append(block, "")
	block = append(block, keyLines...)
	block = append(block, "")
	block = append(block, sep)
	block = append(block, strings.Repeat(" ", footPad)+titleFooter+strings.Repeat(" ", max(0, maxKeyW-footPad-lipgloss.Width(titleFooter))))

	return centerFrame(strings.Join(block, "\n"), m.width, m.height)
}

func TitleRow(breathe float64) string {
	dot := theme.LogoDotColor(breathe).Render("●")
	title := theme.Title().Render("flow")
	desc := theme.Muted().Render("bandwidth monitor")
	return fmt.Sprintf("%s  %s   %s", dot, title, desc)
}

func renderDashboard(m Model, hero bool) string {
	termW := m.width
	if termW <= 0 {
		termW = 80
	}
	termH := m.height
	if termH <= 0 {
		termH = 24
	}

	contentW := min(termW-4, heroInnerMaxWidth)
	if !hero {
		contentW = min(termW-4, compactInnerMax)
	}
	if contentW < 40 {
		contentW = max(termW-2, 40)
	}

	downRatio := theme.SpeedRatio(m.dispDown, maxf(m.rollingMaxDown, m.dispDown))
	upRatio := theme.SpeedRatio(m.dispUp, maxf(m.rollingMaxUp, m.dispUp))
	downPulse := 0.0
	upPulse := 0.0

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

	// Render high-resolution Braille graphs with vertical gradients
	downGraph := renderColoredGraph(downSamples, graphW, 4, maxf(m.rollingMaxDown, m.dispDown), frac, true)
	upGraph := renderColoredGraph(upSamples, graphW, 4, maxf(m.rollingMaxUp, m.dispUp), frac, false)

	// Responsive TUI glowing borders
	downBorderColor := theme.DownloadBorderColor(downRatio)
	upBorderColor := theme.UploadBorderColor(upRatio)

	lines := make([]string, 0, 24)
	if hero && termH >= 30 {
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

	// Format download and upload speed values
	downVal := theme.Dim().Render(theme.DirArrow(true)) + " " + theme.ValuePrimary(downRatio, true).Render(m.FormatBps(m.dispDown)) + " " + theme.Muted().Render(downTrend)
	upVal := theme.Dim().Render(theme.DirArrow(false)) + " " + theme.ValuePrimary(upRatio, false).Render(m.FormatBps(m.dispUp)) + " " + theme.Muted().Render(upTrend)

	peakDownVal := m.FormatBps(m.tracker.PeakDown)
	peakUpVal := m.FormatBps(m.tracker.PeakUp)

	// Render Download Panel with border
	downPanel := renderPanel("download", downVal, peakDownVal, downPulse, downGraph, contentW, downBorderColor)
	lines = append(lines, downPanel)
	lines = append(lines, "")

	// Render Upload Panel with border
	upPanel := renderPanel("upload", upVal, peakUpVal, upPulse, upGraph, contentW, upBorderColor)
	lines = append(lines, upPanel)
	lines = append(lines, "")

	// Footer stats and controls (using restored clean arrows and separators)
	todayLine := fmt.Sprintf(
		"today %s  %s   today %s  %s",
		theme.Dim().Render("↓"), theme.Muted().Render(formatBytes(m.tracker.TodayDown)),
		theme.Dim().Render("↑"), theme.Muted().Render(formatBytes(m.tracker.TodayUp)),
	)
	lines = append(lines, todayLine)
	lines = append(lines, "")

	iface := m.ifaceName
	if m.paused {
		iface += "  [paused]"
	}
	lines = append(lines, theme.Muted().Render(iface))
	lines = append(lines, theme.Muted().Render("q quit · r reset · i interface · c units · p pause · ? help"))

	content := strings.Join(lines, "\n")
	return centerFrame(content, termW, termH)
}

func renderPanel(title string, value string, peak string, peakPulse float64, graph string, width int, borderColor lipgloss.Color) string {
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
	panelLines = append(panelLines, "")

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

package theme

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/programmersd21/flow/internal/animate"
)

// High-contrast, premium dark mode typography colors
const (
	textDimHex    = "#64748b" // Cool Slate Gray
	textMutedHex  = "#94a3b8" // Slate 400
	textSoftHex   = "#cbd5e1" // Slate 300
	textBaseHex   = "#e2e8f0" // Slate 200
	textBrightHex = "#f8fafc" // Slate 50
	textPureHex   = "#ffffff" // Pure White
)

// Vibrant color stops (Electric Blue/Indigo/Cyan for download, Emerald/Lime for upload)
var downloadStops = [5][3]uint8{
	{0x3b, 0x82, 0xf6}, // Vibrant Blue (#3B82F6)
	{0x63, 0x66, 0xf1}, // Indigo (#6366F1)
	{0x06, 0xb6, 0xd4}, // Bright Cyan (#06B6D4)
	{0x00, 0xf5, 0xd4}, // Glowing Mint-Cyan
	{0xff, 0xff, 0xff}, // Pure White
}

var uploadStops = [5][3]uint8{
	{0x10, 0xb9, 0x81}, // Emerald (#10B981)
	{0x22, 0xc5, 0x5e}, // Green (#22C55E)
	{0x84, 0xcc, 0x16}, // Lime (#84CC16)
	{0xa3, 0xe6, 0x35}, // Bright Lime
	{0xff, 0xff, 0xff}, // Pure White
}

var logoStops = [4][3]uint8{
	{0xd9, 0x46, 0xef}, // Fuchsia (#D946EF)
	{0x8b, 0x5c, 0xf6}, // Purple (#8B5C96)
	{0x3b, 0x82, 0xf6}, // Blue (#3B82F6)
	{0x06, 0xb6, 0xd4}, // Cyan (#06B6D4)
}

func Title() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(textPureHex)).
		Bold(true)
}

func Muted() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(textMutedHex))
}

func Soft() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(textSoftHex))
}

func Dim() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(textDimHex))
}

func LogoDotColor(breathe float64) lipgloss.Style {
	breathe = animate.Clamp01(breathe)
	r, g, b := animate.ColorLerp(0x64, 0x74, 0x8b, 0xf8, 0xfa, 0xfc, breathe)
	return lipgloss.NewStyle().Foreground(lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, b)))
}

func Accent() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(textBrightHex))
}

func Label() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(textSoftHex))
}

func TextDim() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(textDimHex))
}

func TextPrimary() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(textBrightHex))
}

// Dynamic colors

func DownloadColor(intensity float64) lipgloss.Style {
	intensity = animate.Clamp01(intensity)
	r, g, b := fiveStopGradient(downloadStops, intensity)
	return lipgloss.NewStyle().Foreground(lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, b)))
}

func UploadColor(intensity float64) lipgloss.Style {
	intensity = animate.Clamp01(intensity)
	r, g, b := fiveStopGradient(uploadStops, intensity)
	return lipgloss.NewStyle().Foreground(lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, b)))
}

func DownloadBorderColor(intensity float64) lipgloss.Color {
	intensity = animate.Clamp01(intensity)
	// Interpolates between deep indigo-blue (#2563eb) and electric cyan (#06b6d4)
	r, g, b := animate.ColorLerp(0x25, 0x63, 0xeb, 0x06, 0xb6, 0xd4, intensity)
	return lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, b))
}

func UploadBorderColor(intensity float64) lipgloss.Color {
	intensity = animate.Clamp01(intensity)
	// Interpolates between deep forest-green (#059669) and glowing lime (#84cc16)
	r, g, b := animate.ColorLerp(0x05, 0x96, 0x69, 0x84, 0xcc, 0x16, intensity)
	return lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, b))
}

func PeakColor(pulse float64) lipgloss.Style {
	pulse = animate.Clamp01(pulse)
	// Smoothly fades from bold white to muted slate gray
	r, g, b := animate.ColorLerp(0x94, 0xa3, 0xb8, 0xff, 0xff, 0xff, pulse)
	style := lipgloss.NewStyle().Foreground(lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, b)))
	if pulse > 0.4 {
		style = style.Bold(true)
	}
	return style
}

func fiveStopGradient(stops [5][3]uint8, t float64) (uint8, uint8, uint8) {
	t = animate.Clamp01(t)
	segment := t * 4.0
	idx := int(segment)
	if idx >= 4 {
		idx = 3
		segment = 4.0
	}
	localT := segment - float64(idx)
	r1, g1, b1 := stops[idx][0], stops[idx][1], stops[idx][2]
	r2, g2, b2 := stops[idx+1][0], stops[idx+1][1], stops[idx+1][2]
	return animate.ColorLerp(r1, g1, b1, r2, g2, b2, localT)
}

func SpeedRatio(current, rollingMax float64) float64 {
	if rollingMax <= 0 {
		return 0
	}
	return animate.Clamp01(current / rollingMax)
}

func ValuePrimary(intensity float64, download bool) lipgloss.Style {
	st := DownloadColor(intensity)
	if !download {
		st = UploadColor(intensity)
	}
	return st.Bold(true)
}

func ValueSecondary(download bool) lipgloss.Style {
	if download {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#3b82f6"))
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#10b981"))
}

func GraphColor(intensity float64, download bool) lipgloss.Style {
	if download {
		return DownloadColor(intensity)
	}
	return UploadColor(intensity)
}

func PulseColor(download bool, pulse float64) lipgloss.Style {
	intensity := 0.5 + pulse*0.5
	base := DownloadColor(intensity)
	if !download {
		base = UploadColor(intensity)
	}
	return base.Bold(true)
}

func Highlight(download bool, pulse float64) lipgloss.Style {
	if pulse <= 0 {
		return Soft()
	}
	return PulseColor(download, pulse)
}

// Logo with ANSI Shadow font - elegant, distinctive, iconic
var logoSrc = []string{
	"███████╗██╗      ██████╗ ██╗    ██╗",
	"██╔════╝██║     ██╔═══██╗██║    ██║",
	"█████╗  ██║     ██║   ██║██║ █╗ ██║",
	"██╔══╝  ██║     ██║   ██║██║███╗██║",
	"██║     ███████╗╚██████╔╝╚███╔███╔╝",
	"╚═╝     ╚══════╝ ╚═════╝  ╚══╝╚══╝ ",
}

// LogoColored returns the FLOW logo with vibrant vertical gradient.
func LogoColored(width int) []string {
	const logoW = 38
	if width < logoW {
		return nil
	}

	pad := (width - logoW) / 2
	if pad < 0 {
		pad = 0
	}
	left := strings.Repeat(" ", pad)
	right := strings.Repeat(" ", width-logoW-pad)

	lines := make([]string, len(logoSrc))
	for i, line := range logoSrc {
		rowT := float64(i) / float64(len(logoSrc)-1)
		r, g, b := fourStopLogoGradient(rowT, 1.0)
		color := lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, b))
		lines[i] = left + lipgloss.NewStyle().Foreground(color).Bold(true).Render(line) + right
	}
	return lines
}

// fourStopLogoGradient creates smooth Fuchsia→Purple→Blue→Cyan progression
func fourStopLogoGradient(position, brightness float64) (uint8, uint8, uint8) {
	position = animate.Clamp01(position)
	brightness = animate.Clamp01(brightness)

	var r, g, b uint8
	segment := position * 3.0
	idx := int(segment)
	if idx >= 3 {
		idx = 2
		segment = 3.0
	}
	t := segment - float64(idx)

	r, g, b = animate.ColorLerp(
		logoStops[idx][0], logoStops[idx][1], logoStops[idx][2],
		logoStops[idx+1][0], logoStops[idx+1][1], logoStops[idx+1][2],
		t,
	)

	// Apply brightness
	r = uint8(math.Min(255, float64(r)*brightness))
	g = uint8(math.Min(255, float64(g)*brightness))
	b = uint8(math.Min(255, float64(b)*brightness))

	return r, g, b
}

func DirArrow(download bool) string {
	if download {
		return "↓"
	}
	return "↑"
}

const logoSubtitle = "Calm your network. See it breathe."

// LogoSubtitle returns the centered hero subtitle, padded to width.
func LogoSubtitle(width int) string {
	subtitleW := len(logoSubtitle)
	color := lipgloss.Color(textMutedHex)
	styled := lipgloss.NewStyle().Foreground(color).Render(logoSubtitle)

	pad := (width - subtitleW) / 2
	if pad < 0 {
		pad = 0
	}
	rightPad := width - subtitleW - pad
	if rightPad < 0 {
		rightPad = 0
	}
	return strings.Repeat(" ", pad) + styled + strings.Repeat(" ", rightPad)
}

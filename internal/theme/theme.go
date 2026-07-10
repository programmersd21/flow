package theme

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/programmersd21/flow/internal/animate"
)

type Theme struct {
	Name                string
	TextDim             string
	TextMuted           string
	TextSoft            string
	TextBase            string
	TextBright          string
	TextPure            string
	Border              string
	Accent              string
	DownloadStops       [5][3]uint8
	UploadStops         [5][3]uint8
	DownloadBorderStart [3]uint8
	DownloadBorderEnd   [3]uint8
	UploadBorderStart   [3]uint8
	UploadBorderEnd     [3]uint8
	LogoStops           [4][3]uint8
}

type ThemeInfo struct {
	Name        string
	Description string
}

var themes = []Theme{
	{
		Name:       "default",
		TextDim:    "#64748b",
		TextMuted:  "#94a3b8",
		TextSoft:   "#cbd5e1",
		TextBase:   "#e2e8f0",
		TextBright: "#f8fafc",
		TextPure:   "#ffffff",
		Border:     "#334155", // Neutral slate-700
		Accent:     "#6366f1", // Indigo
		DownloadStops: [5][3]uint8{
			{0x3b, 0x82, 0xf6}, // Vibrant Blue (#3B82F6)
			{0x63, 0x66, 0xf1}, // Indigo (#6366F1)
			{0x06, 0xb6, 0xd4}, // Bright Cyan (#06B6D4)
			{0x00, 0xf5, 0xd4}, // Glowing Mint-Cyan
			{0xff, 0xff, 0xff}, // Pure White
		},
		UploadStops: [5][3]uint8{
			{0x10, 0xb9, 0x81}, // Emerald (#10B981)
			{0x22, 0xc5, 0x5e}, // Green (#22C55E)
			{0x84, 0xcc, 0x16}, // Lime (#84CC16)
			{0xa3, 0xe6, 0x35}, // Bright Lime
			{0xff, 0xff, 0xff}, // Pure White
		},
		DownloadBorderStart: [3]uint8{0x25, 0x63, 0xeb},
		DownloadBorderEnd:   [3]uint8{0x06, 0xb6, 0xd4},
		UploadBorderStart:   [3]uint8{0x05, 0x96, 0x69},
		UploadBorderEnd:     [3]uint8{0x84, 0xcc, 0x16},
		LogoStops: [4][3]uint8{
			{0xd9, 0x46, 0xef}, // Fuchsia
			{0x8b, 0x5c, 0xf6}, // Purple
			{0x3b, 0x82, 0xf6}, // Blue
			{0x06, 0xb6, 0xd4}, // Cyan
		},
	},
	{
		Name:       "nord",
		TextDim:    "#4c566a", // Polar Night Gray
		TextMuted:  "#434c5e", // Darker Gray
		TextSoft:   "#d8dee9", // Snow Storm Soft
		TextBase:   "#e5e9f0", // Snow Storm Base
		TextBright: "#eceff4", // Snow Storm Bright
		TextPure:   "#ffffff",
		Border:     "#3b4252", // Neutral Polar Night gray
		Accent:     "#88c0d0", // Frost Cyan
		DownloadStops: [5][3]uint8{
			{0x5e, 0x81, 0xac}, // Frost Blue (Dark)
			{0x81, 0xa1, 0xc1}, // Frost Blue (Medium)
			{0x88, 0xc0, 0xd0}, // Frost Cyan
			{0x8f, 0xbc, 0xbb}, // Frost Teal
			{0xff, 0xff, 0xff},
		},
		UploadStops: [5][3]uint8{
			{0xa3, 0xbe, 0x8c}, // Aurora Green
			{0xeb, 0xcb, 0x8b}, // Aurora Yellow
			{0xd0, 0x87, 0x70}, // Aurora Orange
			{0xbf, 0x61, 0x6a}, // Aurora Red
			{0xff, 0xff, 0xff},
		},
		DownloadBorderStart: [3]uint8{0x5e, 0x81, 0xac},
		DownloadBorderEnd:   [3]uint8{0x88, 0xc0, 0xd0},
		UploadBorderStart:   [3]uint8{0xa3, 0xbe, 0x8c},
		UploadBorderEnd:     [3]uint8{0xeb, 0xcb, 0x8b},
		LogoStops: [4][3]uint8{
			{0xb4, 0x8e, 0xad}, // Aurora Purple
			{0x88, 0xc0, 0xd0}, // Frost Cyan
			{0x81, 0xa1, 0xc1}, // Frost Blue
			{0xa3, 0xbe, 0x8c}, // Aurora Green
		},
	},
	{
		Name:       "dracula",
		TextDim:    "#6272a4", // Comment Purple
		TextMuted:  "#6272a4",
		TextSoft:   "#f8f8f2", // Foreground
		TextBase:   "#f8f8f2",
		TextBright: "#f1fa8c", // Yellow Accent
		TextPure:   "#ffffff",
		Border:     "#44475a", // Neutral comment gray
		Accent:     "#ff79c6", // Pink Accent
		DownloadStops: [5][3]uint8{
			{0xbd, 0x93, 0xf9}, // Purple
			{0x8b, 0xe9, 0xfd}, // Cyan
			{0x50, 0xfa, 0x7b}, // Green
			{0xf1, 0xfa, 0x8c}, // Yellow
			{0xff, 0xff, 0xff},
		},
		UploadStops: [5][3]uint8{
			{0xff, 0x79, 0xc6}, // Pink
			{0xff, 0xb8, 0x6c}, // Orange
			{0xff, 0x55, 0x55}, // Red
			{0xbd, 0x93, 0xf9}, // Purple
			{0xff, 0xff, 0xff},
		},
		DownloadBorderStart: [3]uint8{0xbd, 0x93, 0xf9},
		DownloadBorderEnd:   [3]uint8{0x8b, 0xe9, 0xfd},
		UploadBorderStart:   [3]uint8{0xff, 0x79, 0xc6},
		UploadBorderEnd:     [3]uint8{0xff, 0xb8, 0x6c},
		LogoStops: [4][3]uint8{
			{0xbd, 0x93, 0xf9},
			{0xff, 0x79, 0xc6},
			{0x8b, 0xe9, 0xfd},
			{0x50, 0xfa, 0x7b},
		},
	},
	{
		Name:       "gruvbox",
		TextDim:    "#7c6f64", // Gray
		TextMuted:  "#a89984", // Light Gray
		TextSoft:   "#ebdbb2", // Cream
		TextBase:   "#ebdbb2",
		TextBright: "#fabd2f", // Yellow
		TextPure:   "#ffffff",
		Border:     "#3c3836", // Neutral dark gray
		Accent:     "#fe8019", // Orange Accent
		DownloadStops: [5][3]uint8{
			{0x45, 0x85, 0x88}, // Blue
			{0x83, 0xa5, 0x98}, // Light Blue
			{0x8e, 0xc0, 0x7c}, // Aqua
			{0xeb, 0xdb, 0xb2}, // Cream
			{0xff, 0xff, 0xff},
		},
		UploadStops: [5][3]uint8{
			{0x98, 0x97, 0x1a}, // Green
			{0xb8, 0xbb, 0x26}, // Light Green
			{0xd7, 0x99, 0x21}, // Yellow
			{0xfe, 0x80, 0x19}, // Orange
			{0xff, 0xff, 0xff},
		},
		DownloadBorderStart: [3]uint8{0x45, 0x85, 0x88},
		DownloadBorderEnd:   [3]uint8{0x83, 0xa5, 0x98},
		UploadBorderStart:   [3]uint8{0x98, 0x97, 0x1a},
		UploadBorderEnd:     [3]uint8{0xb8, 0xbb, 0x26},
		LogoStops: [4][3]uint8{
			{0xd9, 0x46, 0xef},
			{0xfe, 0x80, 0x19},
			{0xb8, 0xbb, 0x26},
			{0x8c, 0xc0, 0x7c},
		},
	},
	{
		Name:       "forest",
		TextDim:    "#3f6212", // Dark Moss Green
		TextMuted:  "#4ade80", // Light Green
		TextSoft:   "#86efac", // Soft Green
		TextBase:   "#dcfce7", // Very Soft Green
		TextBright: "#f0fdf4", // Bright leaf
		TextPure:   "#ffffff",
		Border:     "#27272a", // Neutral dark gray
		Accent:     "#4ade80", // Highlight Green
		DownloadStops: [5][3]uint8{
			{0x0d, 0x94, 0x88}, // Ocean Teal
			{0x14, 0xb8, 0xa6}, // Teal
			{0x2d, 0xd4, 0xbf}, // Light Teal
			{0xa7, 0xf3, 0xd0}, // Emerald Soft
			{0xff, 0xff, 0xff},
		},
		UploadStops: [5][3]uint8{
			{0x16, 0xa3, 0x4a}, // Forest Green
			{0x22, 0xc5, 0x5e}, // Leaf Green
			{0x4a, 0xde, 0x80}, // Light Green
			{0x86, 0xef, 0xac}, // Soft Green
			{0xff, 0xff, 0xff},
		},
		DownloadBorderStart: [3]uint8{0x0d, 0x94, 0x88},
		DownloadBorderEnd:   [3]uint8{0x2d, 0xd4, 0xbf},
		UploadBorderStart:   [3]uint8{0x16, 0xa3, 0x4a},
		UploadBorderEnd:     [3]uint8{0x4a, 0xde, 0x80},
		LogoStops: [4][3]uint8{
			{0x15, 0x80, 0x3d},
			{0x22, 0xc5, 0x5e},
			{0x2d, 0xd4, 0xbf},
			{0xa7, 0xf3, 0xd0},
		},
	},
	{
		Name:       "monochrome",
		TextDim:    "#525252", // Neutral Gray
		TextMuted:  "#737373", // Lighter Gray
		TextSoft:   "#a3a3a3", // Soft Gray
		TextBase:   "#e5e5e5", // Off-white
		TextBright: "#ffffff", // Pure White
		TextPure:   "#ffffff",
		Border:     "#404040", // Neutral gray border
		Accent:     "#a3a3a3", // Neutral Accent
		DownloadStops: [5][3]uint8{
			{0xff, 0xff, 0xff},
			{0xe5, 0xe5, 0xe5},
			{0xd4, 0xd4, 0xd4},
			{0xa3, 0xa3, 0xa3},
			{0xff, 0xff, 0xff},
		},
		UploadStops: [5][3]uint8{
			{0x73, 0x73, 0x73},
			{0x8a, 0x8a, 0x8a},
			{0xa3, 0xa3, 0xa3},
			{0xd4, 0xd4, 0xd4},
			{0xff, 0xff, 0xff},
		},
		DownloadBorderStart: [3]uint8{0xff, 0xff, 0xff},
		DownloadBorderEnd:   [3]uint8{0xa3, 0xa3, 0xa3},
		UploadBorderStart:   [3]uint8{0x73, 0x73, 0x73},
		UploadBorderEnd:     [3]uint8{0xd4, 0xd4, 0xd4},
		LogoStops: [4][3]uint8{
			{0xff, 0xff, 0xff},
			{0xe5, 0xe5, 0xe5},
			{0xa3, 0xa3, 0xa3},
			{0x73, 0x73, 0x73},
		},
	},
	{
		Name:       "catppuccin",
		TextDim:    "#585b70",
		TextMuted:  "#7f849c",
		TextSoft:   "#a6adc8",
		TextBase:   "#cdd6f4",
		TextBright: "#f5e0dc",
		TextPure:   "#ffffff",
		Border:     "#313244", // Neutral surface0
		Accent:     "#cba6f7", // Mauve
		DownloadStops: [5][3]uint8{
			{0x89, 0xb4, 0xfa}, // Blue
			{0x74, 0xc7, 0xec}, // Sapphire
			{0x89, 0xdc, 0xeb}, // Sky
			{0x94, 0xe2, 0xd5}, // Teal
			{0xff, 0xff, 0xff},
		},
		UploadStops: [5][3]uint8{
			{0xa6, 0xe3, 0xa1}, // Green
			{0x94, 0xe2, 0xd5}, // Teal
			{0xf9, 0xe2, 0xaf}, // Yellow
			{0xfa, 0xb3, 0x87}, // Peach
			{0xff, 0xff, 0xff},
		},
		DownloadBorderStart: [3]uint8{0x89, 0xb4, 0xfa},
		DownloadBorderEnd:   [3]uint8{0x89, 0xdc, 0xeb},
		UploadBorderStart:   [3]uint8{0xa6, 0xe3, 0xa1},
		UploadBorderEnd:     [3]uint8{0xf9, 0xe2, 0xaf},
		LogoStops: [4][3]uint8{
			{0xcb, 0xa6, 0xf7},
			{0xf5, 0xc2, 0xe7},
			{0x89, 0xb4, 0xfa},
			{0x89, 0xdc, 0xeb},
		},
	},
	{
		Name:       "tokyo-night",
		TextDim:    "#565f89",
		TextMuted:  "#737aa2",
		TextSoft:   "#9aa5ce",
		TextBase:   "#a9b1d6",
		TextBright: "#c0caf5",
		TextPure:   "#ffffff",
		Border:     "#1f2335", // Neutral border/surface
		Accent:     "#bb9af7", // Purple
		DownloadStops: [5][3]uint8{
			{0x3b, 0x82, 0xf6},
			{0x7a, 0xa2, 0xf7},
			{0x7d, 0xcf, 0xff},
			{0xbb, 0x9a, 0xf7},
			{0xff, 0xff, 0xff},
		},
		UploadStops: [5][3]uint8{
			{0x9e, 0xce, 0x6a},
			{0xe0, 0xaf, 0x68},
			{0xff, 0x9e, 0x64},
			{0xf7, 0x76, 0x8e},
			{0xff, 0xff, 0xff},
		},
		DownloadBorderStart: [3]uint8{0x3b, 0x82, 0xf6},
		DownloadBorderEnd:   [3]uint8{0x7d, 0xcf, 0xff},
		UploadBorderStart:   [3]uint8{0x9e, 0xce, 0x6a},
		UploadBorderEnd:     [3]uint8{0xe0, 0xaf, 0x68},
		LogoStops: [4][3]uint8{
			{0xbb, 0x9a, 0xf7},
			{0x7a, 0xa2, 0xf7},
			{0x7d, 0xcf, 0xff},
			{0xf7, 0x76, 0x8e},
		},
	},
}

var activeTheme = &themes[0]

func SetTheme(name string) {
	for i := range themes {
		if themes[i].Name == name {
			activeTheme = &themes[i]
			return
		}
	}
	activeTheme = &themes[0]
}

func ListThemes() []ThemeInfo {
	return []ThemeInfo{
		{Name: "default", Description: "The standard flow palette with electric blue and emerald green"},
		{Name: "nord", Description: "A cool-toned, low-saturation polar palette with frost blues"},
		{Name: "dracula", Description: "A vibrant gothic theme with lavender, pinks, and cyans"},
		{Name: "gruvbox", Description: "A warm retro-minimal theme inspired by sand, rust, and moss"},
		{Name: "forest", Description: "A leafy natural palette with moss greens and ocean teals"},
		{Name: "monochrome", Description: "A high-contrast clean theme with neutral grays and silver"},
		{Name: "catppuccin", Description: "A soothing pastel palette with mauves, blues, and teals"},
		{Name: "tokyo-night", Description: "A deep dark theme with vibrant blues, cyans, and purples"},
	}
}

func GetBorderColor() string {
	return activeTheme.Border
}

func GetAccentColor() string {
	return activeTheme.Accent
}

func GetTextBrightColor() string {
	return activeTheme.TextBright
}

func GetTextMutedColor() string {
	return activeTheme.TextMuted
}

func GetTextSoftColor() string {
	return activeTheme.TextSoft
}

func GetTextDimColor() string {
	return activeTheme.TextDim
}

func Title() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(activeTheme.TextSoft)).
		Bold(true)
}

func Muted() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(activeTheme.TextMuted))
}

func Soft() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(activeTheme.TextSoft))
}

func Dim() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(activeTheme.TextDim))
}

func LogoDotColor(pulse float64) lipgloss.Style {
	pulse = animate.Clamp01(pulse)
	r1, g1, b1 := hexToRGB(activeTheme.TextDim)
	r2, g2, b2 := hexToRGB(activeTheme.Accent)
	r, g, b := animate.ColorLerp(r1, g1, b1, r2, g2, b2, pulse)
	return lipgloss.NewStyle().Foreground(lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, b)))
}

func Accent() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(activeTheme.Accent))
}

func Label() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(activeTheme.TextSoft))
}

func TextDim() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(activeTheme.TextDim))
}

func DownloadColor(intensity float64) lipgloss.Style {
	intensity = animate.Clamp01(intensity)
	r, g, b := fiveStopGradient(activeTheme.DownloadStops, intensity)
	return lipgloss.NewStyle().Foreground(lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, b)))
}

func UploadColor(intensity float64) lipgloss.Style {
	intensity = animate.Clamp01(intensity)
	r, g, b := fiveStopGradient(activeTheme.UploadStops, intensity)
	return lipgloss.NewStyle().Foreground(lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, b)))
}

func DownloadBorderColor(intensity float64) lipgloss.Color {
	intensity = animate.Clamp01(intensity)
	r, g, b := animate.ColorLerp(
		activeTheme.DownloadBorderStart[0], activeTheme.DownloadBorderStart[1], activeTheme.DownloadBorderStart[2],
		activeTheme.DownloadBorderEnd[0], activeTheme.DownloadBorderEnd[1], activeTheme.DownloadBorderEnd[2],
		intensity,
	)
	return lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, b))
}

func UploadBorderColor(intensity float64) lipgloss.Color {
	intensity = animate.Clamp01(intensity)
	r, g, b := animate.ColorLerp(
		activeTheme.UploadBorderStart[0], activeTheme.UploadBorderStart[1], activeTheme.UploadBorderStart[2],
		activeTheme.UploadBorderEnd[0], activeTheme.UploadBorderEnd[1], activeTheme.UploadBorderEnd[2],
		intensity,
	)
	return lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, b))
}

func PeakColor(pulse float64) lipgloss.Style {
	pulse = animate.Clamp01(pulse)
	r1, g1, b1 := hexToRGB(activeTheme.TextMuted)
	r2, g2, b2 := hexToRGB(activeTheme.Accent)
	r, g, b := animate.ColorLerp(r1, g1, b1, r2, g2, b2, pulse)
	return lipgloss.NewStyle().Foreground(lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, b)))
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

var logoSrc = []string{
	"███████╗██╗      ██████╗ ██╗    ██╗",
	"██╔════╝██║     ██╔═══██╗██║    ██║",
	"█████╗  ██║     ██║   ██║██║ █╗ ██║",
	"██╔══╝  ██║     ██║   ██║██║███╗██║",
	"██║     ███████╗╚██████╔╝╚███╔███╔╝",
	"╚═╝     ╚══════╝ ╚═════╝  ╚══╝╚══╝ ",
}

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
		activeTheme.LogoStops[idx][0], activeTheme.LogoStops[idx][1], activeTheme.LogoStops[idx][2],
		activeTheme.LogoStops[idx+1][0], activeTheme.LogoStops[idx+1][1], activeTheme.LogoStops[idx+1][2],
		t,
	)

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

func LogoSubtitle(width int) string {
	subtitleW := len(logoSubtitle)
	color := lipgloss.Color(activeTheme.TextMuted)
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

func hexToRGB(hex string) (uint8, uint8, uint8) {
	if len(hex) > 0 && hex[0] == '#' {
		hex = hex[1:]
	}
	if len(hex) != 6 {
		return 128, 128, 128
	}
	var r, g, b uint8
	_, _ = fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b)
	return r, g, b
}

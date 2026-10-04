package theme

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// TestGradientHexRespectsANSIPalette is a regression test: the `ansi` theme
// carries terminal palette *indices* and leaves its RGB stops zeroed.
// GradientHex used to read the zeroed stops, so every waveform cell came out
// #000000 and the whole graph rendered black on that theme.
func TestGradientHexRespectsANSIPalette(t *testing.T) {
	SetTheme("ansi")
	defer SetTheme("default")

	if !UsesANSI() {
		t.Fatal("ansi theme should report ANSI palette mode")
	}
	for _, download := range []bool{true, false} {
		for _, pos := range []float64{0, 0.25, 0.5, 0.75, 1} {
			got := GradientHex(download, pos)
			if got == "" {
				t.Fatalf("GradientHex(%v, %v) returned empty", download, pos)
			}
			if got == "#000000" {
				t.Errorf("GradientHex(%v, %v) = %q — black means the zeroed RGB "+
					"stops were used instead of the palette indices", download, pos, got)
			}
			if strings.HasPrefix(got, "#") {
				t.Errorf("GradientHex(%v, %v) = %q — ANSI themes must return a "+
					"palette index, not hex", download, pos, got)
			}
		}
	}
}

// TestAccentStylesHonourANSI: Down/Up accents must not resolve to black either.
func TestAccentStylesHonourANSI(t *testing.T) {
	SetTheme("ansi")
	defer SetTheme("default")
	for name, st := range map[string]lipgloss.TerminalColor{
		"down": DownStyle().GetForeground(),
		"up":   UpStyle().GetForeground(),
	} {
		if st == nil {
			t.Errorf("%s accent on ansi theme is unset", name)
			continue
		}
		if s := string(st.(lipgloss.Color)); s == "" || s == "#000000" {
			t.Errorf("%s accent on ansi theme = %q", name, s)
		}
	}
}

// TestGradientHexRGBForRGBThemes guards the normal path.
func TestGradientHexRGBForRGBThemes(t *testing.T) {
	SetTheme("default")
	got := GradientHex(true, 0.5)
	if !strings.HasPrefix(got, "#") || len(got) != 7 {
		t.Errorf("RGB theme should return #rrggbb, got %q", got)
	}
	if got == "#000000" {
		t.Error("default theme returned black")
	}
}

// TestDimColorLeavesPaletteIndicesAlone: palette indices are not hex, and
// parsing them as hex silently yielded black chrome (gridlines, peak line).
func TestDimColorLeavesPaletteIndicesAlone(t *testing.T) {
	for _, idx := range []string{"4", "12", "6"} {
		if got := DimColor(idx, 0.5); got != idx {
			t.Errorf("DimColor(%q, 0.5) = %q, want unchanged", idx, got)
		}
	}
	if got := DimColor("#ffffff", 0.5); got != "#7f7f7f" {
		t.Errorf("DimColor(#ffffff, 0.5) = %q, want #7f7f7f", got)
	}
	if got := DimColor("", 0.5); got != "" {
		t.Errorf("DimColor(\"\", 0.5) = %q", got)
	}
}

// Every built-in theme must produce non-black waveform colors.
func TestAllThemesProduceVisibleWaveform(t *testing.T) {
	defer SetTheme("default")
	names := []string{}
	for _, ti := range ListThemes() {
		names = append(names, ti.Name)
	}
	for _, name := range names {
		SetTheme(name)
		down := GradientHex(true, 0.5)
		up := GradientHex(false, 0.5)
		if down == "#000000" {
			t.Errorf("theme %q: download waveform is black", name)
		}
		if up == "#000000" {
			t.Errorf("theme %q: upload waveform is black", name)
		}
	}
}

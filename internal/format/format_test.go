package format

import (
	"math"
	"strings"
	"testing"
)

func TestRenderTemplate(t *testing.T) {
	d := Data{Iface: "wlan0", DownBps: 2_410_000, UpBps: 195_000,
		Down: "2.3 MB/s", Up: "190 KB/s", Time: "2026-10-04T12:00:00Z"}
	out, err := RenderTemplate(`{{.Iface}} {{.Down}} {{.Up}}`, d)
	if err != nil {
		t.Fatal(err)
	}
	if out != "wlan0 2.3 MB/s 190 KB/s" {
		t.Errorf("got %q", out)
	}
}

func TestRenderTemplateFuncs(t *testing.T) {
	d := Data{DownBps: 2_410_000}
	out, err := RenderTemplate(`{{.DownBps | rate}}`, d)
	if err != nil {
		t.Fatal(err)
	}
	if out != "2.4 MB/s" {
		t.Errorf("rate = %q", out)
	}
	out, err = RenderTemplate(`{{pad 6 "ab"}}|`, d)
	if err != nil {
		t.Fatal(err)
	}
	if out != "ab    |" {
		t.Errorf("pad = %q", out)
	}
	out, err = RenderTemplate(`{{spark .DownSamples}}`, Data{DownSamples: []float64{1, 2, 3}})
	if err != nil {
		t.Fatal(err)
	}
	if len([]rune(out)) != 3 {
		t.Errorf("spark = %q", out)
	}
}

func TestRenderTemplateError(t *testing.T) {
	if _, err := RenderTemplate(`{{.Nope`, Data{}); err == nil {
		t.Error("expected parse error")
	}
}

func TestColorRespectsNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	out, err := RenderTemplate(`{{color "red" "x"}}`, Data{})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "\x1b") {
		t.Errorf("NO_COLOR output has escapes: %q", out)
	}
}

// Negative and NaN samples used to index the block ramp out of range and
// panic; they must clamp to the lowest block instead.
func TestSparkClampsOutOfRangeSamples(t *testing.T) {
	for name, samples := range map[string][]float64{
		"negative": {10, -10, 5},
		"nan":      {10, math.NaN(), 5},
	} {
		out, err := RenderTemplate(`{{spark .DownSamples}}`, Data{DownSamples: samples})
		if err != nil {
			t.Fatalf("%s: spark errored: %v", name, err)
		}
		if got := len([]rune(out)); got != len(samples) {
			t.Errorf("%s: spark rendered %d runes, want %d", name, got, len(samples))
		}
	}
}

func TestPadNonPositiveWidthIsEmpty(t *testing.T) {
	for _, tmpl := range []string{`{{pad -1 "x"}}`, `{{pad 0 "x"}}`} {
		out, err := RenderTemplate(tmpl, Data{})
		if err != nil {
			t.Fatalf("%s: %v", tmpl, err)
		}
		if out != "" {
			t.Errorf("%s = %q, want empty", tmpl, out)
		}
	}
}

// Compile + Render exist so streaming modes parse once instead of per sample;
// they must produce the same output as the one-shot helper.
func TestCompileAndRenderMatchRenderTemplate(t *testing.T) {
	tmpl, err := Compile(`{{.Iface}} {{pad 6 .Down}}`)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	data := Data{Iface: "wlan0", Down: "2 MB/s"}
	got, err := Render(tmpl, data)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	want, err := RenderTemplate(`{{.Iface}} {{pad 6 .Down}}`, data)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("compiled render = %q, want %q", got, want)
	}
}

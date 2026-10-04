package format

import (
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

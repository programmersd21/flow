package ui

import (
	"flag"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/programmersd21/flow/internal/history"
	"github.com/programmersd21/flow/internal/theme"
)

var updateGolden = flag.Bool("update-golden", false, "rewrite golden frame files")

// goldenSamples is deterministic synthetic traffic: a slow oscillation with a
// burst near the end, so goldens exercise idle, plateau, and peak rendering.
func goldenSamples(n int, base, amp int, burstAt int, burstAmp float64) []float64 {
	out := make([]float64, n)
	for i := range out {
		t := float64(i) / 10.0
		v := float64(base) + float64(amp)*(0.5+0.5*math.Sin(t*0.5)) + float64(amp)*0.2*math.Sin(t*1.7+1.0)
		if i >= burstAt && i < burstAt+20 {
			v += burstAmp * (1 - float64(i-burstAt)/20.0)
		}
		if v < 0 {
			v = 0
		}
		out[i] = v
	}
	return out
}

// goldenModel builds a deterministic hero model from synthetic data.
func goldenModel(w, h int) Model {
	theme.SetTheme("default")
	dh := history.New(36000)
	uh := history.New(36000)
	var lastDown, lastUp, peakDown, peakUp float64
	down := goldenSamples(480, 1_200_000, 800_000, 380, 3_000_000)
	up := goldenSamples(480, 180_000, 120_000, 400, 500_000)
	for i := range down {
		dh.Push(down[i])
		uh.Push(up[i])
		lastDown, lastUp = down[i], up[i]
		if down[i] > peakDown {
			peakDown = down[i]
		}
		if up[i] > peakUp {
			peakUp = up[i]
		}
	}
	tr := history.NewTracker()
	tr.PeakDown = peakDown
	tr.PeakUp = peakUp
	tr.TodayDown = 16_900_000
	tr.TodayUp = 70_300_000
	return Model{
		width:  w,
		height: h,
		rates: rateState{
			lastSample:     time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC),
			pingLatency:    46 * time.Millisecond,
			animDown:       lastDown,
			animUp:         lastUp,
			rollingMaxDown: peakDown,
			rollingMaxUp:   peakUp,
		},
		downHist:        dh,
		upHist:          uh,
		tracker:         tr,
		refreshInterval: 100 * time.Millisecond,

		nowOverride: time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC),
		viewMode:    ViewHero,
		iface:       ifaceState{name: "wlan0"},

		windowSecs: 60,
		showGrid:   true,
		noAnim:     true, // static frames for goldens
		launchDone: true,
	}
}

// TestGoldenHeroFrames snapshots the hero screen at 4 fixed sizes.
// Regenerate with: go test ./internal/ui -run TestGoldenHeroFrames -update-golden
func TestGoldenHeroFrames(t *testing.T) {
	sizes := [][2]int{{60, 20}, {80, 24}, {100, 30}, {140, 40}}
	for _, sz := range sizes {
		w, h := sz[0], sz[1]
		name := filepath.Join("testdata", goldenName(w, h))
		m := goldenModel(w, h)
		lines := dashboardContentLines(m, ViewHero)
		frame := stripANSI(strings.Join(lines, "\n")) + "\n"

		if *updateGolden {
			if err := os.WriteFile(name, []byte(frame), 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		want, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("missing golden %s (run with -update-golden): %v", name, err)
		}
		if diff := compareFrames(frame, normalizeEOL(string(want))); diff != "" {
			t.Errorf("golden mismatch at %dx%d: %s\n--- got ---\n%s\n--- want ---\n%s",
				w, h, diff, frame, want)
		}
	}
}

// normalizeEOL strips carriage returns from a golden file.
//
// .gitattributes pins *.golden to LF, but a checkout with a different
// core.autocrlf setting, or an editor that rewrote the file, still delivers
// CRLF. That made every line differ from the generated frame by a trailing
// carriage return, which is why the golden test failed only on Windows.
// Normalising here keeps the comparison about content.
func normalizeEOL(s string) string {
	if !strings.ContainsRune(s, '\r') {
		return s
	}
	return strings.ReplaceAll(s, "\r\n", "\n")
}

// maxBrailleDiffs is the tolerance budget for braille-cell differences per
// frame.
//
// Go's math package explicitly does not guarantee bit-identical results across
// architectures, and the waveform ends in math.Pow. A single-ulp difference in
// a normalized value can flip one dot where a fill boundary lands on a
// threshold, so two correct renders of identical data can differ in a handful
// of cells depending on CPU and Go version.
//
// The budget is small on purpose. A real regression - changed layout, font,
// scale, or colors - alters hundreds of cells. Every non-braille rune is still
// compared exactly, so a regression in digits, labels, axes, or the frozen
// footer fails hard.
const maxBrailleDiffs = 12

// isBrailleCell reports whether r is a braille pattern dot character.
func isBrailleCell(r rune) bool {
	return r >= 0x2800 && r <= 0x28FF
}

// compareFrames reports why two frames differ, or "" when they match within
// tolerance. Line and rune counts must be identical; differing braille cells
// are counted against maxBrailleDiffs.
func compareFrames(got, want string) string {
	gLines := strings.Split(got, "\n")
	wLines := strings.Split(want, "\n")
	if len(gLines) != len(wLines) {
		return "line count differs"
	}
	diffs := 0
	for i := range gLines {
		gRunes := []rune(gLines[i])
		wRunes := []rune(wLines[i])
		if len(gRunes) != len(wRunes) {
			return "line width differs"
		}
		for j := range gRunes {
			if gRunes[j] == wRunes[j] {
				continue
			}
			if isBrailleCell(gRunes[j]) || isBrailleCell(wRunes[j]) {
				diffs++
				continue
			}
			return "non-braille content differs"
		}
	}
	if diffs > maxBrailleDiffs {
		return "too many braille cells differ"
	}
	return ""
}

func goldenName(w, h int) string {
	return "hero_" + itoa(w) + "x" + itoa(h) + ".golden"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [8]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

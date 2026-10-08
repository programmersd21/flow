package ui

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/programmersd21/flow/internal/history"
	"github.com/programmersd21/flow/internal/theme"
)

// TestFooterGoldenContract asserts the hard non-negotiable rule of flow:
// THE FOOTER IS FROZEN. The two footer lines of the hero home screen must render
// exactly as in v0.3.1: same text, same order, same separators.
// Row 1: q quit · m mode · d filter · t theme · ? help
// Row 2: g github · u issues · x discuss · $ sponsor
func TestFooterGoldenContract(t *testing.T) {
	theme.SetTheme("default")
	m := Model{
		width:           100,
		height:          30,
		tracker:         history.NewTracker(),
		downHist:        history.New(60),
		upHist:          history.New(60),
		refreshInterval: 100 * time.Millisecond,
		rates:           rateState{lastSample: time.Now()},
		viewMode:        ViewHero,
		iface:           ifaceState{name: "wlan0"},
	}

	lines := dashboardContentLines(m, ViewHero)
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines in hero mode, got %d", len(lines))
	}

	// The last two non-empty lines are the footer rows
	row1 := stripANSI(lines[len(lines)-2])
	row2 := stripANSI(lines[len(lines)-1])

	expectedRow1 := "q quit · m mode · d filter · t theme · ? help"
	expectedRow2 := "g github · u issues · x discuss · $ sponsor"

	if !strings.Contains(row1, expectedRow1) {
		t.Errorf("Footer Row 1 mismatch!\nGot:      %q\nExpected: %q", row1, expectedRow1)
	}

	if !strings.Contains(row2, expectedRow2) {
		t.Errorf("Footer Row 2 mismatch!\nGot:      %q\nExpected: %q", row2, expectedRow2)
	}
}

// TestTinyModeGoldenContract asserts --tiny output contract
func TestTinyModeGoldenContract(t *testing.T) {
	m := Model{
		width:    80,
		height:   24,
		viewMode: ViewTiny,
		rates:    rateState{animDown: 2048, animUp: 1024}, // 2 KB/s down, 1 KB/s up
	}
	out := renderTiny(m)
	clean := stripANSI(out)
	if !strings.Contains(clean, "↓") || !strings.Contains(clean, "↑") {
		t.Errorf("Tiny mode output missing arrows: %q", clean)
	}
}

var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func stripANSI(str string) string {
	return ansiRegex.ReplaceAllString(str, "")
}

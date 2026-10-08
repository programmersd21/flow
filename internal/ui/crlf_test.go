package ui

import (
	"strings"
	"testing"
)

// Regression guard for the Windows-only golden failure: a golden file
// delivered with CRLF endings must still compare equal to the generated frame,
// while an unnormalized comparison must still detect the difference.
func TestGoldenCompareHandlesCRLF(t *testing.T) {
	m := goldenModel(80, 24)
	frame := stripANSI(strings.Join(dashboardContentLines(m, ViewHero), "\n")) + "\n"
	crlf := strings.ReplaceAll(strings.TrimSuffix(frame, "\n"), "\n", "\r\n") + "\r\n"

	if strings.Contains(frame, "\r") {
		t.Fatal("generated frame must never contain a carriage return")
	}
	if diff := compareFrames(frame, normalizeEOL(crlf)); diff != "" {
		t.Errorf("CRLF golden should compare equal after normalizing, got %q", diff)
	}
	if diff := compareFrames(frame, crlf); diff == "" {
		t.Error("control: unnormalized CRLF must be detected as different")
	}
}

// Braille-cell drift within the tolerance must pass; beyond it must fail.
func TestGoldenCompareBrailleTolerance(t *testing.T) {
	m := goldenModel(80, 24)
	frame := stripANSI(strings.Join(dashboardContentLines(m, ViewHero), "\n")) + "\n"
	runes := []rune(frame)
	got := strings.Builder{}
	flipped := 0
	for _, r := range runes {
		if flipped < maxBrailleDiffs && isBrailleCell(r) {
			got.WriteRune(' ')
			flipped++
			continue
		}
		got.WriteRune(r)
	}
	if diff := compareFrames(frame, got.String()); diff != "" {
		t.Errorf("%d flipped braille cells should be tolerated, got %q", flipped, diff)
	}

	got.Reset()
	flipped = 0
	for _, r := range runes {
		if flipped < maxBrailleDiffs*10 && isBrailleCell(r) {
			got.WriteRune(' ')
			flipped++
			continue
		}
		got.WriteRune(r)
	}
	if diff := compareFrames(frame, got.String()); diff == "" {
		t.Errorf("%d flipped braille cells should exceed the tolerance", flipped)
	}
}

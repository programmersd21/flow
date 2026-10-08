package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// overlayBox draws a modal overlay and guarantees two things that every
// overlay depends on:
//
//   - it never draws wider than the terminal, and
//   - it never draws taller than the terminal.
//
// Both are load-bearing: an overlay wider than the terminal wraps and
// corrupts the surrounding display, and one taller than the terminal
// scrolls the dashboard out of view.
//
// On a terminal too small for a bordered box (fewer than about 6 columns), the
// rows are returned without a frame rather than drawing an overlay that cannot
// fit.
func overlayBox(rows []string, border lipgloss.Color, termW, termH int) string {
	if termW <= 0 {
		termW = 80
	}
	if termH <= 0 {
		termH = 24
	}

	// Fit to height first: the frame costs 2 rows.
	inner := termH - 2
	if inner > 0 && len(rows) > inner {
		rows = rows[:inner]
	}
	if len(rows) == 0 {
		return ""
	}

	if termW < 8 || termH < 4 {
		// Too small to frame. Return plain rows, clipped to the terminal.
		out := make([]string, 0, len(rows))
		for _, r := range rows {
			out = append(out, clipCols(r, termW))
		}
		return strings.Join(out, "\n")
	}

	// Box width: fill the terminal, minus a small margin on wide screens.
	boxW := termW
	if termW > 40 {
		boxW -= 4
	}
	if boxW > 100 {
		boxW = 100
	}
	if boxW < 8 {
		boxW = 8
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Padding(0, 2).
		MaxWidth(boxW).
		MaxHeight(termH).
		Render(strings.Join(rows, "\n"))

	return clipBlock(box, termW, termH)
}

// clipBlock hard-trims a rendered block to the terminal, line by line. This is a
// backstop for content that is too wide for the frame: lipgloss wraps rather
// than clips, and wrapping is what corrupts a TUI.
func clipBlock(s string, termW, termH int) string {
	lines := strings.Split(s, "\n")
	if len(lines) > termH {
		lines = lines[:termH]
	}
	for i, l := range lines {
		lines[i] = clipCols(l, termW)
	}
	return strings.Join(lines, "\n")
}

// clipCols cuts a line to termW visible columns, preserving escape sequences
// and never splitting a multi-byte rune.
func clipCols(line string, termW int) string {
	if termW <= 0 || lipgloss.Width(line) <= termW {
		return line
	}
	var b strings.Builder
	w := 0
	inEscape := false
	for _, r := range line {
		if inEscape {
			b.WriteRune(r)
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEscape = false
			}
			continue
		}
		if r == 0x1b {
			inEscape = true
			b.WriteRune(r)
			continue
		}
		rw := lipgloss.Width(string(r))
		if w+rw > termW {
			break
		}
		b.WriteRune(r)
		w += rw
	}
	// Reset any style left open by the cut.
	b.WriteString("\x1b[0m")
	return b.String()
}

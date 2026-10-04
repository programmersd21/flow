package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// onOff renders a boolean as a short label for toasts.
func onOff(b bool) string {
	if b {
		return "on"
	}
	return "off"
}

// stripANSISeq removes ANSI escape sequences for plain-text export.
func stripANSISeq(s string) string {
	var sb strings.Builder
	sb.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == 0x1b {
			j := i + 1
			for j < len(s) && (s[j] < 0x40 || s[j] > 0x7e) {
				j++
			}
			i = j
			continue
		}
		sb.WriteByte(s[i])
	}
	return sb.String()
}

// exportSnapshot writes the current frame as .ansi (raw) and .txt (plain)
// to the export dir, XDG pictures, or CWD. Never overwrites: a counter is
// appended when the name is taken.
func (m Model) exportSnapshot() (string, error) {
	frame := m.View()
	stamp := time.Now().Format("20060102-150405")

	dir := m.cfg.Export.Dir
	if dir == "" {
		dir = os.Getenv("XDG_PICTURES_DIR")
	}
	if dir == "" {
		if home, err := os.UserHomeDir(); err == nil {
			dir = filepath.Join(home, "Pictures")
		}
	}
	if _, err := os.Stat(dir); err != nil {
		dir = "."
	}

	base := filepath.Join(dir, "flow-"+stamp)
	ansiPath := uniquePath(base + ".ansi")
	if err := os.WriteFile(ansiPath, []byte(frame), 0o644); err != nil {
		return "", err
	}
	txtPath := uniquePath(base + ".txt")
	if err := os.WriteFile(txtPath, []byte(stripANSISeq(frame)), 0o644); err != nil {
		return "", err
	}
	return ansiPath, nil
}

func uniquePath(path string) string {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path
	}
	ext := filepath.Ext(path)
	stem := strings.TrimSuffix(path, ext)
	for i := 1; ; i++ {
		cand := fmt.Sprintf("%s-%d%s", stem, i, ext)
		if _, err := os.Stat(cand); os.IsNotExist(err) {
			return cand
		}
	}
}

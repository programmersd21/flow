package theme

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

type customTheme struct {
	Name       string   `toml:"name"`
	TextDim    string   `toml:"text_dim"`
	TextMuted  string   `toml:"text_muted"`
	TextSoft   string   `toml:"text_soft"`
	TextBase   string   `toml:"text_base"`
	TextBright string   `toml:"text_bright"`
	TextPure   string   `toml:"text_pure"`
	Border     string   `toml:"border"`
	Accent     string   `toml:"accent"`
	Download   []string `toml:"download"`
	Upload     []string `toml:"upload"`
}

func themesDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		home, err2 := os.UserHomeDir()
		if err2 != nil {
			return "", err2
		}
		base = filepath.Join(home, ".config")
	}
	dir := filepath.Join(base, "flow", "themes")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func LoadCustomThemes() []Theme {
	dir, err := themesDir()
	if err != nil {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var result []Theme
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".toml") {
			continue
		}
		t, err := parseThemeFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		result = append(result, t)
	}
	return result
}

func parseThemeFile(path string) (Theme, error) {
	var ct customTheme
	if _, err := toml.DecodeFile(path, &ct); err != nil {
		return Theme{}, err
	}
	if ct.Name == "" {
		return Theme{}, fmt.Errorf("theme missing name")
	}
	t := Theme{
		Name:       ct.Name,
		TextDim:    ct.TextDim,
		TextMuted:  ct.TextMuted,
		TextSoft:   ct.TextSoft,
		TextBase:   ct.TextBase,
		TextBright: ct.TextBright,
		TextPure:   ct.TextPure,
		Border:     ct.Border,
		Accent:     ct.Accent,
	}
	for i, s := range ct.Download {
		if i >= 5 {
			break
		}
		c, err := hexToRGBStrict(s)
		if err != nil {
			return Theme{}, err
		}
		t.DownloadStops[i] = c
	}
	for i, s := range ct.Upload {
		if i >= 5 {
			break
		}
		c, err := hexToRGBStrict(s)
		if err != nil {
			return Theme{}, err
		}
		t.UploadStops[i] = c
	}
	t.DownloadBorderStart = t.DownloadStops[0]
	t.DownloadBorderEnd = t.DownloadStops[3]
	t.UploadBorderStart = t.UploadStops[0]
	t.UploadBorderEnd = t.UploadStops[3]
	t.LogoStops = [4][3]uint8{
		t.DownloadStops[0],
		t.DownloadStops[1],
		t.UploadStops[0],
		t.UploadStops[1],
	}
	return t, nil
}

func hexToRGBStrict(s string) ([3]uint8, error) {
	s = strings.TrimPrefix(s, "#")
	if len(s) != 6 {
		return [3]uint8{}, fmt.Errorf("invalid hex: %s", s)
	}
	var c [3]uint8
	_, err := fmt.Sscanf(s, "%02x%02x%02x", &c[0], &c[1], &c[2])
	return c, err
}

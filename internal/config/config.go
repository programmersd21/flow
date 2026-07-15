// internal/config/config.go — TOML config loader with XDG_CONFIG_HOME support.

package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Refresh    duration `toml:"refresh"`
	History    int      `toml:"history"`
	Theme      string   `toml:"theme"`
	Unit       string   `toml:"unit"`
	Interface  string   `toml:"interface"`
	NoColor    bool     `toml:"no_color"`
	Bits       bool     `toml:"bits"`
	PingTarget string   `toml:"ping_target"`
}

func Defaults() Config {
	return Config{
		Refresh:    duration{100 * time.Millisecond},
		History:    60,
		Theme:      "default",
		Unit:       "auto",
		Interface:  "auto",
		NoColor:    false,
		Bits:       false,
		PingTarget: "1.1.1.1",
	}
}

// Load reads the config file, creating it with defaults if missing.
// CLI-flag overrides are applied by the caller after this returns.
func Load() (Config, error) {
	cfg := Defaults()

	path, err := configPath()
	if err != nil {
		return cfg, fmt.Errorf("config: resolve path: %w", err)
	}

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		if err2 := writeDefaults(path, cfg); err2 != nil {
			// Non-fatal: just return defaults if we can't write.
			fmt.Fprintf(os.Stderr, "flow: could not create config file: %v\n", err2)
		}
		return cfg, nil
	}

	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return cfg, fmt.Errorf("config: parse %s: %w", path, err)
	}
	return cfg, nil
}

// Save writes the current config structure to the user's config file.
func Save(cfg Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close() //nolint:errcheck

	_, err = fmt.Fprintf(f, defaultTOML,
		cfg.Refresh.String(),
		cfg.History,
		cfg.Theme,
		cfg.Unit,
		cfg.Interface,
		boolStr(cfg.NoColor),
		boolStr(cfg.Bits),
		cfg.PingTarget,
	)
	return err
}

// configPath resolves the config file location using platform-specific paths.
func configPath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		// Fallback to XDG-style path if os.UserConfigDir fails.
		base = os.Getenv("XDG_CONFIG_HOME")
		if base == "" {
			home, err2 := os.UserHomeDir()
			if err2 != nil {
				return "", err2
			}
			base = filepath.Join(home, ".config")
		}
	}
	return filepath.Join(base, "flow", "config.toml"), nil
}

func writeDefaults(path string, cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close() //nolint:errcheck

	_, err = fmt.Fprintf(f, defaultTOML,
		cfg.Refresh.String(),
		cfg.History,
		cfg.Theme,
		cfg.Unit,
		cfg.Interface,
		boolStr(cfg.NoColor),
		boolStr(cfg.Bits),
		cfg.PingTarget,
	)
	return err
}

const defaultTOML = `# flow configuration
# https://github.com/programmersd21/flow

refresh     = "%s"     # sampling interval (e.g. "100ms", "250ms", "1s")
history     = %d       # seconds of sparkline history
theme       = "%s"
unit        = "%s"     # auto | kb | mb | gb
interface   = "%s"     # auto or interface name (e.g. "eth0", "wlan0")
no_color    = %s
bits        = %s       # display in bits/sec instead of bytes/sec
ping_target = "%s"     # host for latency measurement
`

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// duration wraps time.Duration for TOML unmarshal of strings like "250ms".
type duration struct{ time.Duration }

func (d *duration) UnmarshalText(text []byte) error {
	v, err := time.ParseDuration(string(text))
	if err != nil {
		return err
	}
	d.Duration = v
	return nil
}

func (d duration) MarshalText() ([]byte, error) {
	return []byte(d.String()), nil
}

// RefreshDuration returns the configured interval; falls back to 100ms.
func (c Config) RefreshDuration() time.Duration {
	if c.Refresh.Duration <= 0 {
		return 100 * time.Millisecond
	}
	return c.Refresh.Duration
}

// NewDuration wraps time.Duration for TOML-compatible flag parsing.
func NewDuration(d time.Duration) duration {
	return duration{d}
}

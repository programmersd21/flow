// internal/config/config.go — TOML config loader with XDG/FLOW_CONFIG support.
//
// Existing flat keys (refresh, theme, unit, ...) keep working unchanged; the
// [ui], [graph], [latency], [history], [processes], [geoip] and [export]
// sections are additive.

package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

type UIConfig struct {
	View       string `toml:"view"`
	FPS        int    `toml:"fps"`
	Animations *bool  `toml:"animations"`
	LaunchAnim *bool  `toml:"launch_animation"`
	Glyphs     string `toml:"glyphs"`
	Gradient   *bool  `toml:"gradient"`
	Glow       *bool  `toml:"glow"`
	Mirrored   *bool  `toml:"mirrored"`
	IEC        bool   `toml:"iec"`
	MaxWidth   int    `toml:"max_width"`
}

type GraphConfig struct {
	Window      string  `toml:"window"`
	Scale       string  `toml:"scale"`
	FixedMax    string  `toml:"fixed_max"`
	SharedScale bool    `toml:"shared_scale"`
	Smoothing   float64 `toml:"smoothing"`
	PeakHold    *bool   `toml:"peak_hold"`
	Gridlines   *bool   `toml:"gridlines"`
	Floor       string  `toml:"floor"`
}

type LatencyConfig struct {
	Targets []string `toml:"targets"`
	Method  string   `toml:"method"`
}

// StorageConfig holds persistent-history settings.
//
// It is exposed as the [storage] TOML section rather than [history] because
// the flat key `history = <int seconds>` already exists and sets sparkline
// depth. A table and a scalar cannot share the name in TOML, and regressing
// the existing key is not an option.
type StorageConfig struct {
	Enabled       *bool `toml:"enabled"`
	RetentionDays int   `toml:"retention_days"`
}

type ProcessesConfig struct {
	ResolveDNS bool `toml:"resolve_dns"`
}

type GeoIPConfig struct {
	DBPath string `toml:"db_path"`
}

type ExportConfig struct {
	Dir string `toml:"dir"`
}

type Config struct {
	// Flat, long-standing keys.
	Refresh    duration `toml:"refresh"`
	History    int      `toml:"history"`
	Theme      string   `toml:"theme"`
	Unit       string   `toml:"unit"`
	Interface  string   `toml:"interface"`
	NoColor    bool     `toml:"no_color"`
	Bits       bool     `toml:"bits"`
	PingTarget string   `toml:"ping_target"`
	NoAnim     bool     `toml:"no_anim"`
	View       string   `toml:"view"`
	WindowSecs int      `toml:"window_secs"`

	UI        UIConfig        `toml:"ui"`
	Graph     GraphConfig     `toml:"graph"`
	Latency   LatencyConfig   `toml:"latency"`
	Storage   StorageConfig   `toml:"storage"`
	Processes ProcessesConfig `toml:"processes"`
	GeoIP     GeoIPConfig     `toml:"geoip"`
	Export    ExportConfig    `toml:"export"`

	// path is where this config was loaded from ("" if defaults were used).
	path string
	// warnings collected during load (unknown keys, invalid values).
	warnings []string
}

func Defaults() Config {
	yes := true
	return Config{
		Refresh:    duration{100 * time.Millisecond},
		History:    60,
		Theme:      "default",
		Unit:       "auto",
		Interface:  "auto",
		PingTarget: "1.1.1.1",
		UI: UIConfig{
			View:       "hero",
			FPS:        60,
			Animations: &yes,
			LaunchAnim: &yes,
			Glyphs:     "auto",
			Gradient:   &yes,
			Glow:       &yes,
			Mirrored:   &yes,
			MaxWidth:   100,
		},
		Graph: GraphConfig{
			Window:   "60s",
			Scale:    "auto",
			FixedMax: "100MB/s",
			// A faint midpoint rule, visible only where the waveform has no
			// fill. This matches the reference demo.png look.
			Gridlines: &yes,
			Smoothing: 0.35,
			PeakHold:  &yes,
			Floor:     "10KB/s",
		},
		Latency: LatencyConfig{
			Targets: []string{"1.1.1.1"},
			Method:  "auto",
		},
		Storage:   StorageConfig{Enabled: &yes, RetentionDays: 90},
		Processes: ProcessesConfig{},
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
	cfg.path = path

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		if err2 := writeFile(path, cfg); err2 != nil {
			// Non-fatal: just return defaults if we can't write.
			fmt.Fprintf(os.Stderr, "flow: could not create config file: %v\n", err2)
		}
		return cfg, nil
	}

	md, err := toml.DecodeFile(path, &cfg)
	if err != nil {
		return cfg, fmt.Errorf("config: parse %s: %w", path, err)
	}
	cfg.path = path

	// Unknown keys are a warning, never a crash.
	for _, k := range md.Undecoded() {
		cfg.warnings = append(cfg.warnings, "unknown config key "+k.String())
	}
	if err := cfg.validate(); err != nil {
		cfg.warnings = append(cfg.warnings, err.Error())
	}
	// Keep the flat keys and the section keys in sync.
	cfg.syncLegacy()
	return cfg, nil
}

// Warnings returns load-time warnings (unknown keys, invalid values).
func (c Config) Warnings() []string { return c.warnings }

// Path returns the config file path in use.
func (c Config) Path() string { return c.path }

// syncLegacy reconciles the flat legacy keys with the newer sections so a
// value set either way ends up in the same place.
func (c *Config) syncLegacy() {
	if c.View != "" {
		c.UI.View = c.View
	}
	c.View = ""
	if c.WindowSecs > 0 {
		c.Graph.Window = fmt.Sprintf("%ds", c.WindowSecs)
		c.WindowSecs = 0
	}
	if c.NoAnim && c.UI.Animations != nil && *c.UI.Animations {
		f := false
		c.UI.Animations = &f
	}
}

var validViews = map[string]bool{"hero": true, "compact": true, "mini": true, "tiny": true}
var validUnits = map[string]bool{"auto": true, "kb": true, "mb": true, "gb": true}
var validGlyphs = map[string]bool{"auto": true, "braille": true, "blocks": true, "ascii": true}
var validScales = map[string]bool{"auto": true, "log": true, "sqrt": true, "fixed": true}
var validMethods = map[string]bool{"auto": true, "icmp": true, "tcp": true}

// validate clamps every invalid value back to its default and reports all
// problems as a single comma-separated warning. Every value is checked, so one
// bad key never leaves the rest of the file unclamped.
func (c *Config) validate() error {
	var problems []string
	warn := func(format string, args ...any) {
		problems = append(problems, fmt.Sprintf(format, args...))
	}

	switch {
	case c.Refresh.Duration <= 0:
		c.Refresh = duration{100 * time.Millisecond}
		warn("refresh invalid; using 100ms")
	case c.Refresh.Duration < 10*time.Millisecond:
		c.Refresh = duration{10 * time.Millisecond}
		warn("refresh too small; using 10ms")
	case c.Refresh.Duration > time.Minute:
		c.Refresh = duration{time.Minute}
		warn("refresh too large; using 1m")
	}
	if !validUnits[strings.ToLower(c.Unit)] {
		c.Unit = "auto"
		warn("unit %q invalid; using auto", c.Unit)
	}
	if !validViews[strings.ToLower(c.UI.View)] {
		c.UI.View = "hero"
		warn("ui.view %q invalid; using hero", c.UI.View)
	}
	if !validGlyphs[strings.ToLower(c.UI.Glyphs)] {
		c.UI.Glyphs = "auto"
		warn("ui.glyphs %q invalid; using auto", c.UI.Glyphs)
	}
	if !validScales[strings.ToLower(c.Graph.Scale)] {
		c.Graph.Scale = "auto"
		warn("graph.scale %q invalid; using auto", c.Graph.Scale)
	}
	if !validMethods[strings.ToLower(c.Latency.Method)] {
		c.Latency.Method = "auto"
		warn("latency.method %q invalid; using auto", c.Latency.Method)
	}
	if c.UI.FPS < 15 {
		c.UI.FPS = 15
		warn("ui.fps %d below 15; using 15", c.UI.FPS)
	}
	if c.UI.FPS > 120 {
		c.UI.FPS = 120
		warn("ui.fps %d above 120; using 120", c.UI.FPS)
	}
	if c.UI.MaxWidth < 40 {
		c.UI.MaxWidth = 40
		warn("ui.max_width %d below 40; using 40", c.UI.MaxWidth)
	}
	if c.UI.MaxWidth > 200 {
		c.UI.MaxWidth = 200
		warn("ui.max_width %d above 200; using 200", c.UI.MaxWidth)
	}
	if c.Graph.Smoothing < 0 || c.Graph.Smoothing > 1 {
		c.Graph.Smoothing = 0.35
		warn("graph.smoothing outside 0..1; using 0.35")
	}
	if c.Storage.RetentionDays < 1 {
		c.Storage.RetentionDays = 90
		warn("storage.retention_days %d invalid; using 90", c.Storage.RetentionDays)
	}
	if len(c.Latency.Targets) == 0 {
		c.Latency.Targets = []string{c.PingTarget}
		if c.Latency.Targets[0] == "" {
			c.Latency.Targets = []string{"1.1.1.1"}
		}
	}
	if strings.ToLower(c.Graph.Scale) == "fixed" && strings.TrimSpace(c.Graph.FixedMax) == "" {
		c.Graph.FixedMax = "100MB/s"
		warn("graph.fixed_max empty while scale=fixed; using 100MB/s")
	}

	if len(problems) == 0 {
		return nil
	}
	return fmt.Errorf("config: %s", strings.Join(problems, "; "))
}

// ─── typed accessors used by the UI ─────────────────────────────────────────

func (c Config) RefreshDuration() time.Duration {
	if c.Refresh.Duration <= 0 {
		return 100 * time.Millisecond
	}
	return c.Refresh.Duration
}

func (c Config) AnimationsEnabled() bool {
	if c.NoAnim {
		return false
	}
	if c.UI.Animations == nil {
		return true
	}
	return *c.UI.Animations
}

func (c Config) LaunchAnimationEnabled() bool {
	if !c.AnimationsEnabled() {
		return false
	}
	if c.UI.LaunchAnim == nil {
		return true
	}
	return *c.UI.LaunchAnim
}

func (c Config) GradientEnabled() bool {
	return c.UI.Gradient == nil || *c.UI.Gradient
}

func (c Config) GlowEnabled() bool {
	return c.UI.Glow == nil || *c.UI.Glow
}

func (c Config) MirroredEnabled() bool {
	return c.UI.Mirrored == nil || *c.UI.Mirrored
}

// PeakHoldEnabled is retained for config compatibility. The full-width peak
// line it controlled was removed: it drew a horizontal rule at the height of
// the window's transient max, which mixed with the waveform and read as
// broken glyphs. The peak value is already reported numerically in the hero
// caption ("peak 4.5 MB/s").
func (c Config) PeakHoldEnabled() bool {
	return c.Graph.PeakHold == nil || *c.Graph.PeakHold
}

func (c Config) GridlinesEnabled() bool {
	return c.Graph.Gridlines == nil || *c.Graph.Gridlines
}

func (c Config) HistoryEnabled() bool {
	if c.Storage.Enabled == nil {
		return true
	}
	return *c.Storage.Enabled
}

// WindowSeconds converts graph.window ("60s", "5m", "1h", "24h") to seconds.
func (c Config) WindowSeconds() int {
	switch strings.ToLower(strings.TrimSpace(c.Graph.Window)) {
	case "1m":
		return 60
	case "5m":
		return 300
	case "15m":
		return 900
	case "1h":
		return 3600
	case "24h":
		return 86400
	default:
		if secs, err := strconv.Atoi(strings.TrimSuffix(c.Graph.Window, "s")); err == nil && secs > 0 {
			return secs
		}
		return 60
	}
}

// LatencyTargets returns the ping targets, falling back to ping_target.
func (c Config) LatencyTargets() []string {
	if len(c.Latency.Targets) > 0 {
		return c.Latency.Targets
	}
	if c.PingTarget != "" {
		return []string{c.PingTarget}
	}
	return []string{"1.1.1.1"}
}

// NeedsRestart reports whether a reload changed something that cannot be
// applied live. Theme and UI settings hot-reload; everything else needs a
// restart.
func (c Config) NeedsRestart(other Config) bool {
	return c.Refresh != other.Refresh ||
		c.Interface != other.Interface ||
		c.Unit != other.Unit ||
		c.Bits != other.Bits ||
		c.PingTarget != other.PingTarget ||
		c.NoColor != other.NoColor ||
		c.Processes != other.Processes ||
		strings.Join(c.Latency.Targets, ",") != strings.Join(other.Latency.Targets, ",") ||
		c.Latency.Method != other.Latency.Method ||
		c.Export.Dir != other.Export.Dir ||
		c.GeoIP.DBPath != other.GeoIP.DBPath
}

// ─── paths, save, reload ────────────────────────────────────────────────────

// configPath resolves the config file location. FLOW_CONFIG overrides
// everything, then XDG_CONFIG_HOME via os.UserConfigDir, then ~/.config.
func configPath() (string, error) {
	if p := os.Getenv("FLOW_CONFIG"); p != "" {
		return p, nil
	}
	base, err := os.UserConfigDir()
	if err != nil || base == "" {
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

// Save writes the full config atomically (temp file + rename) so a crash or a
// concurrent read never sees a half-written file.
func Save(cfg Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	return writeFile(path, cfg)
}

func writeFile(path string, cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data := []byte(render(cfg))
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Reload re-reads the config from disk, returning defaults-with-warnings if
// the file became unreadable. Used by the hot-reload poller.
func (c Config) Reload() (Config, error) {
	path := c.path
	if path == "" {
		var err error
		if path, err = configPath(); err != nil {
			return c, err
		}
	}
	if _, err := os.Stat(path); err != nil {
		return c, err
	}
	var next Config
	md, err := toml.DecodeFile(path, &next)
	if err != nil {
		return c, fmt.Errorf("config: parse %s: %w", path, err)
	}
	next.path = path
	for _, k := range md.Undecoded() {
		next.warnings = append(next.warnings, "unknown config key "+k.String())
	}
	if err := next.validate(); err != nil {
		next.warnings = append(next.warnings, err.Error())
	}
	next.syncLegacy()
	return next, nil
}

// Watch polls path for modification-time changes. It returns a channel of
// Config values whenever the file changes. Polling is used instead of fsnotify
// to avoid a new dependency; the interval is coarse and the cost is one stat.
func Watch(cfg Config, interval time.Duration) <-chan Config {
	out := make(chan Config, 1)
	path := cfg.path
	if path == "" {
		close(out)
		return out
	}
	last := modTime(path)
	go func() {
		defer close(out)
		t := time.NewTicker(interval)
		defer t.Stop()
		for range t.C {
			cur := modTime(path)
			if cur.Equal(last) {
				continue
			}
			last = cur
			if next, err := cfg.Reload(); err == nil {
				select {
				case out <- next:
				default:
				}
			}
		}
	}()
	return out
}

func modTime(path string) time.Time {
	fi, err := os.Stat(path)
	if err != nil {
		var zero time.Time
		if errors.Is(err, fs.ErrNotExist) {
			return zero
		}
		return zero
	}
	return fi.ModTime()
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func boolPtrStr(p *bool, def bool) string {
	if p == nil {
		return boolStr(def)
	}
	return boolStr(*p)
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

// NewDuration wraps time.Duration for TOML-compatible flag parsing.
func NewDuration(d time.Duration) duration { return duration{d} }

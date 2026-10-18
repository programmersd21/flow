package config

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// writeCfg points FLOW_CONFIG at a temp file with the given body and loads it.
func loadFrom(t *testing.T, body string) Config {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FLOW_CONFIG", p)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return cfg
}

func TestDefaultsAreValid(t *testing.T) {
	c := Defaults()
	if err := c.validate(); err != nil {
		t.Fatalf("defaults should validate cleanly, got %v", err)
	}
	if !c.AnimationsEnabled() || !c.LaunchAnimationEnabled() {
		t.Error("animations should default to enabled")
	}
	if !c.GradientEnabled() || !c.MirroredEnabled() {
		t.Error("gradient and mirrored should default to enabled")
	}
	if !c.GridlinesEnabled() {
		t.Error("gridlines should default to enabled (reference demo look)")
	}
	if c.WindowSeconds() != 60 {
		t.Errorf("default window = %d, want 60", c.WindowSeconds())
	}
}

func TestLegacyFlatConfigStillLoads(t *testing.T) {
	cfg := loadFrom(t, `
refresh     = "250ms"
theme       = "nord"
unit        = "mb"
interface   = "wlan0"
no_color    = true
bits        = true
ping_target = "8.8.8.8"
`)
	if cfg.RefreshDuration() != 250*time.Millisecond {
		t.Errorf("refresh = %v, want 250ms", cfg.RefreshDuration())
	}
	if cfg.Theme != "nord" || cfg.Unit != "mb" {
		t.Errorf("theme/unit = %q/%q", cfg.Theme, cfg.Unit)
	}
	if !cfg.NoColor || !cfg.Bits {
		t.Error("no_color/bits not loaded")
	}
	if len(cfg.Warnings()) != 0 {
		t.Errorf("legacy config should load without warnings, got %v", cfg.Warnings())
	}
}

func TestSectionedConfig(t *testing.T) {
	cfg := loadFrom(t, `
refresh = "150ms"

[ui]
view   = "mini"
fps    = 30
glyphs = "blocks"
max_width = 120

[graph]
window     = "5m"
scale      = "sqrt"
smoothing  = 0.5

[latency]
targets = ["1.1.1.1", "8.8.8.8"]
method  = "tcp"

[storage]
enabled        = false
retention_days = 30

[processes]
resolve_dns = true
`)
	if cfg.UI.View != "mini" || cfg.UI.FPS != 30 {
		t.Errorf("ui section = %+v", cfg.UI)
	}
	if cfg.UI.Glyphs != "blocks" || cfg.UI.MaxWidth != 120 {
		t.Errorf("ui glyphs/width = %q/%d", cfg.UI.Glyphs, cfg.UI.MaxWidth)
	}
	if cfg.WindowSeconds() != 300 {
		t.Errorf("graph.window 5m -> %d, want 300", cfg.WindowSeconds())
	}
	if cfg.Graph.Scale != "sqrt" {
		t.Errorf("graph.scale = %q", cfg.Graph.Scale)
	}
	if cfg.LatencyTargets()[1] != "8.8.8.8" {
		t.Errorf("latency.targets = %v", cfg.LatencyTargets())
	}
	if cfg.HistoryEnabled() {
		t.Error("storage.enabled=false was ignored")
	}
	if !cfg.Processes.ResolveDNS {
		t.Error("processes.resolve_dns ignored")
	}
}

func TestInvalidValuesFallBackWithWarning(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{"unit", "unit = \"furlongs\"\n", "unit"},
		{"ui.view", "[ui]\nview = \"hologram\"\n", "ui.view"},
		{"ui.glyphs", "[ui]\nglyphs = \"emoji\"\n", "ui.glyphs"},
		{"graph.scale", "[graph]\nscale = \"cubic\"\n", "graph.scale"},
		{"fps_low", "[ui]\nfps = 2\n", "ui.fps"},
		{"fps_high", "[ui]\nfps = 500\n", "ui.fps"},
		{"smoothing", "[graph]\nsmoothing = 4.0\n", "graph.smoothing"},
		{"method", "[latency]\nmethod = \"carrier-pigeon\"\n", "latency.method"},
		{"retention", "[storage]\nretention_days = 0\n", "storage.retention_days"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := loadFrom(t, tc.body)
			if len(cfg.Warnings()) == 0 {
				t.Fatalf("expected a warning for %s", tc.name)
			}
			found := false
			for _, w := range cfg.Warnings() {
				if strings.Contains(w, tc.want) {
					found = true
				}
			}
			if !found {
				t.Errorf("warnings %v do not mention %q", cfg.Warnings(), tc.want)
			}
			if err := cfg.validate(); err != nil {
				t.Errorf("post-clamp validate failed: %v", err)
			}
		})
	}
}

func TestEveryInvalidValueIsClampedNotJustTheFirst(t *testing.T) {
	cfg := loadFrom(t, `
unit = "furlongs"
[ui]
view = "hologram"
fps  = 900
[graph]
smoothing = -1
`)
	if cfg.Unit != "auto" {
		t.Errorf("unit not clamped: %q", cfg.Unit)
	}
	if cfg.UI.View != "hero" {
		t.Errorf("ui.view not clamped: %q", cfg.UI.View)
	}
	if cfg.UI.FPS != 120 {
		t.Errorf("ui.fps not clamped: %d", cfg.UI.FPS)
	}
	if cfg.Graph.Smoothing != 0.35 {
		t.Errorf("smoothing not clamped: %v", cfg.Graph.Smoothing)
	}
	if len(cfg.Warnings()) != 1 {
		t.Errorf("expected one combined warning, got %v", cfg.Warnings())
	}
}

func TestLegacyHistoryIntStillWorks(t *testing.T) {
	// `history = 60` is the long-standing sparkline-depth key. It must keep
	// parsing even though a [storage] section now exists.
	cfg := loadFrom(t, "history = 60\n\n[storage]\nretention_days = 7\n")
	if cfg.History != 60 {
		t.Errorf("legacy history = %d, want 60", cfg.History)
	}
	if cfg.Storage.RetentionDays != 7 {
		t.Errorf("storage.retention_days = %d, want 7", cfg.Storage.RetentionDays)
	}
}

func TestUnknownKeysWarnButDoNotCrash(t *testing.T) {
	cfg := loadFrom(t, "theme = \"nord\"\ntotally_unknown = 42\n\n[ui]\nnot_a_key = true\n")
	if len(cfg.Warnings()) == 0 {
		t.Fatal("expected unknown-key warnings")
	}
	if !strings.Contains(strings.Join(cfg.Warnings(), ";"), "totally_unknown") {
		t.Errorf("warnings should name the key, got %v", cfg.Warnings())
	}
	if cfg.Theme != "nord" {
		t.Error("valid keys should still apply alongside unknown ones")
	}
}

func TestRefreshClamped(t *testing.T) {
	if cfg := loadFrom(t, `refresh = "1ms"`); cfg.RefreshDuration() != 10*time.Millisecond {
		t.Errorf("refresh 1ms -> %v, want 10ms", cfg.RefreshDuration())
	}
	if cfg := loadFrom(t, `refresh = "90s"`); cfg.RefreshDuration() != time.Minute {
		t.Errorf("refresh 90s -> %v, want 1m", cfg.RefreshDuration())
	}
	if cfg := loadFrom(t, `refresh = "10s"`); cfg.RefreshDuration() != 10*time.Second {
		t.Errorf("refresh 10s should be kept, got %v", cfg.RefreshDuration())
	}
}

func TestSaveRoundTripIsAtomicAndParsable(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.toml")
	t.Setenv("FLOW_CONFIG", p)

	cfg := Defaults()
	cfg.Theme = "gruvbox"
	cfg.UI.View = "compact"
	cfg.Graph.Window = "15m"
	if err := Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := os.Stat(p + ".tmp"); !os.IsNotExist(err) {
		t.Error("Save left a .tmp file behind")
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if loaded.Theme != "gruvbox" || loaded.UI.View != "compact" {
		t.Errorf("round trip lost values: %+v", loaded)
	}
	if loaded.WindowSeconds() != 900 {
		t.Errorf("graph.window round trip = %d", loaded.WindowSeconds())
	}
	if len(loaded.Warnings()) != 0 {
		t.Errorf("our own file should not warn, got %v", loaded.Warnings())
	}
}

func TestNoAnimImpliesAnimationsDisabled(t *testing.T) {
	cfg := loadFrom(t, "no_anim = true\n")
	if cfg.AnimationsEnabled() || cfg.LaunchAnimationEnabled() {
		t.Error("no_anim should disable animations and launch")
	}
}

func TestRenderedFileContainsEverySection(t *testing.T) {
	out := render(Defaults())
	for _, want := range []string{"[ui]", "[graph]", "[latency]", "[storage]", "[processes]", "[geoip]", "[export]"} {
		if !strings.Contains(out, want) {
			t.Errorf("rendered config missing %s", want)
		}
	}
}

// Save is called at runtime (theme picker), so it must round-trip every
// documented key. `history` used to be dropped, silently deleting the user's
// setting the first time they picked a theme.
func TestSaveKeepsLegacyHistoryKey(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.toml")
	t.Setenv("FLOW_CONFIG", p)

	cfg := Defaults()
	cfg.History = 120
	if err := Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}
	body, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "history") {
		t.Error("rendered config omits the history key")
	}
	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.History != 120 {
		t.Errorf("history lost across Save/Load: %d, want 120", loaded.History)
	}
}

// Reload must fall back to defaults for keys absent from the file, not to Go
// zero values. Decoding over a zero Config returned fps=0, max_width=0, etc.
func TestReloadFallsBackToDefaults(t *testing.T) {
	cfg := loadFrom(t, "theme = \"nord\"\n")
	next, err := cfg.Reload()
	if err != nil {
		t.Fatalf("Reload: %v", err)
	}
	if next.Theme != "nord" {
		t.Errorf("reload lost the theme: %q", next.Theme)
	}
	if next.UI.FPS != 60 || next.UI.MaxWidth != 100 {
		t.Errorf("reload zeroed ui defaults: fps=%d max_width=%d", next.UI.FPS, next.UI.MaxWidth)
	}
	if next.Interface != "auto" || next.Graph.Smoothing != 0.35 {
		t.Errorf("reload zeroed core defaults: iface=%q smoothing=%v",
			next.Interface, next.Graph.Smoothing)
	}
}

// NaN slips past both range comparisons, so it needs an explicit check.
func TestNaNSmoothingIsClamped(t *testing.T) {
	cfg := loadFrom(t, "[graph]\nsmoothing = nan\n")
	if math.IsNaN(cfg.Graph.Smoothing) {
		t.Fatal("NaN smoothing survived validation")
	}
	if cfg.Graph.Smoothing != 0.35 {
		t.Errorf("smoothing = %v, want 0.35", cfg.Graph.Smoothing)
	}
	if len(cfg.Warnings()) == 0 {
		t.Error("expected a warning for NaN smoothing")
	}
}

// The flat `view` key is copied into [ui].view by syncLegacy, which must run
// before validation so an invalid value is clamped like any other.
func TestFlatViewKeyIsValidated(t *testing.T) {
	cfg := loadFrom(t, "view = \"hologram\"\n")
	if cfg.UI.View != "hero" {
		t.Errorf("ui.view = %q, want hero", cfg.UI.View)
	}
	joined := strings.Join(cfg.Warnings(), ";")
	if !strings.Contains(joined, "ui.view") {
		t.Errorf("warnings do not mention ui.view: %v", cfg.Warnings())
	}
}

// Warnings must name the offending value, not the replacement: a warning like
// `unit "auto" invalid; using auto` tells the user nothing to fix.
func TestWarningsNameTheOffendingValue(t *testing.T) {
	cfg := loadFrom(t, "unit = \"furlongs\"\n")
	joined := strings.Join(cfg.Warnings(), ";")
	if !strings.Contains(joined, "furlongs") {
		t.Errorf("warning should name the bad value: %v", cfg.Warnings())
	}
}

func TestWindowParsing(t *testing.T) {
	cases := map[string]int{
		"30s": 30, "10m": 600, "2h": 7200,
		"60s": 60, "1m": 60, "5m": 300, "15m": 900, "1h": 3600, "24h": 86400,
	}
	for in, want := range cases {
		cfg := loadFrom(t, fmt.Sprintf("[graph]\nwindow = %q\n", in))
		if got := cfg.WindowSeconds(); got != want {
			t.Errorf("window %q = %d, want %d", in, got, want)
		}
		if len(cfg.Warnings()) != 0 {
			t.Errorf("window %q produced warnings: %v", in, cfg.Warnings())
		}
	}

	bad := loadFrom(t, "[graph]\nwindow = \"banana\"\n")
	if bad.WindowSeconds() != 60 {
		t.Errorf("bad window -> %d, want 60", bad.WindowSeconds())
	}
	if len(bad.Warnings()) == 0 {
		t.Error("an unrecognized window should warn, not silently become 60s")
	}
}

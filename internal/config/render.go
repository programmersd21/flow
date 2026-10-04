package config

import (
	"fmt"
	"strings"
)

// render produces the annotated config file written on first run and on save.
// Every option is documented inline; unknown-key warnings in user files are
// ignored by the TOML decoder, so older files keep loading unchanged.
func render(c Config) string {
	var b strings.Builder
	p := func(format string, args ...any) {
		fmt.Fprintf(&b, format+"\n", args...)
	}

	b.WriteString(`# flow configuration
# https://github.com/programmersd21/flow
#
# Every key below is optional. Values you delete fall back to the default shown.
# Unknown keys produce a warning on stderr, never a crash.

`)
	p(`# ─── core ───────────────────────────────────────────────────────────────`)
	p(`refresh     = %q      # sampling interval (e.g. "100ms", "250ms", "1s")`, c.Refresh.String())
	p(`theme       = %q      # default | nord | dracula | gruvbox | forest | monochrome |`, c.Theme)
	p(`                       # catppuccin | tokyo-night | rose-pine | kanagawa | ansi`)
	p(`unit        = %q      # auto | kb | mb | gb`, c.Unit)
	p(`interface   = %q      # auto or an interface name (e.g. "eth0", "wlan0")`, c.Interface)
	p(`no_color    = %s     # disable all ANSI color`, boolStr(c.NoColor))
	p(`bits        = %s     # show bits/sec instead of bytes/sec`, boolStr(c.Bits))
	p(`no_anim     = %s     # reduced motion: no launch sequence, pulse, or ripple`, boolStr(c.NoAnim))
	p(`ping_target = %q     # latency target (also used when [latency].targets is empty)`, c.PingTarget)

	b.WriteString(`
# ─── ui ────────────────────────────────────────────────────────────────────
`)
	p(`[ui]`)
	p(`view      = %q      # hero | compact | mini | tiny`, c.UI.View)
	p(`fps       = %d       # animation clock (15..120); hot-reloaded`, c.UI.FPS)
	p(`animations = %s     # false disables every transition`, boolPtrStr(c.UI.Animations, true))
	p(`launch_animation = %s`, boolPtrStr(c.UI.LaunchAnim, true))
	p(`glyphs    = %q      # auto | braille | blocks | ascii`, c.UI.Glyphs)
	p(`gradient  = %s     # false renders flat bands`, boolPtrStr(c.UI.Gradient, true))
	p(`glow      = %s     # crest highlight`, boolPtrStr(c.UI.Glow, true))
	p(`mirrored  = %s     # download above the axis, upload below`, boolPtrStr(c.UI.Mirrored, true))
	p(`iec       = %s     # true uses 1024-based units (MiB) instead of 1000-based (MB)`, boolStr(c.UI.IEC))
	p(`max_width = %d      # content column cap; centered when wider`, c.UI.MaxWidth)

	b.WriteString(`
# ─── graph ─────────────────────────────────────────────────────────────────
`)
	p(`[graph]`)
	p(`window        = %q     # 60s | 5m | 15m | 1h | 24h   (also the w key)`, c.Graph.Window)
	p(`scale         = %q     # auto | log | sqrt | fixed    (also the S key)`, c.Graph.Scale)
	p(`fixed_max     = %q     # ceiling when scale = "fixed"`, c.Graph.FixedMax)
	p(`shared_scale  = %s    # true scales both halves together`, boolStr(c.Graph.SharedScale))
	p(`smoothing     = %.2f    # display EMA 0..1; raw data still feeds peaks/JSON`, c.Graph.Smoothing)
	p(`peak_hold     = %s    # accepted for compatibility; the peak value is shown in the hero caption`, boolPtrStr(c.Graph.PeakHold, true))
	p(`gridlines     = %s    # faint midpoint rule, visible where the wave has no fill (G key)`, boolPtrStr(c.Graph.Gridlines, true))
	p(`floor         = %q   # minimum y-max so noise does not fill the screen`, c.Graph.Floor)

	b.WriteString(`
# ─── latency ───────────────────────────────────────────────────────────────
`)
	p(`[latency]`)
	p(`targets = [%s]   # one line per target`, quoteList(c.Latency.Targets))
	p(`method  = %q      # auto | icmp | tcp  (auto falls back to TCP :443)`, c.Latency.Method)

	b.WriteString(`
# ─── storage ───────────────────────────────────────────────────────────────
# Named "storage" rather than "history" because the flat key
#   history = <int seconds>
# already exists and must keep working (it sets sparkline depth).
`)
	p(`[storage]`)
	p(`enabled        = %s    # false stops all writes`, boolPtrStr(c.Storage.Enabled, true))
	p(`retention_days = %d     # older daily totals are pruned`, c.Storage.RetentionDays)

	b.WriteString(`
# ─── processes ─────────────────────────────────────────────────────────────
`)
	p(`[processes]`)
	p(`resolve_dns = %s    # reverse DNS is off by default`, boolStr(c.Processes.ResolveDNS))

	b.WriteString(`
# ─── geoip (optional, local only) ──────────────────────────────────────────
`)
	p(`[geoip]`)
	p(`db_path = %q    # path to a local MaxMind DB; empty disables labels`, c.GeoIP.DBPath)

	b.WriteString(`
# ─── export ────────────────────────────────────────────────────────────────
`)
	p(`[export]`)
	p(`dir = %q      # snapshot directory; empty uses XDG pictures or CWD`, c.Export.Dir)
	return b.String()
}

func quoteList(items []string) string {
	if len(items) == 0 {
		return ""
	}
	q := make([]string, len(items))
	for i, s := range items {
		q[i] = fmt.Sprintf("%q", s)
	}
	return strings.Join(q, ", ")
}

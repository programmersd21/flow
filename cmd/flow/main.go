package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"regexp"
	"runtime/debug"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/programmersd21/flow/internal/collector"
	"github.com/programmersd21/flow/internal/config"
	"github.com/programmersd21/flow/internal/format"
	"github.com/programmersd21/flow/internal/history"
	"github.com/programmersd21/flow/internal/sampler"
	"github.com/programmersd21/flow/internal/theme"
	"github.com/programmersd21/flow/internal/ui"
)

// version is injected at build time with
//
//	-ldflags "-X main.version=$(cat VERSION)"
//
// which is what Makefile and GoReleaser do. Other builds (notably
// "go install ...@latest", which applies no ldflags) fall back to the module
// version recorded in the binary, so --version cannot silently report a stale
// number that was hardcoded here.
var version = "dev"

// cleanSemver matches plain release versions, not pseudo-versions.
var cleanSemver = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

// buildVersion returns the version to display: the injected value when present,
// otherwise the module version from build info, otherwise "dev".
func buildVersion() string {
	if v := strings.TrimSpace(version); v != "" && v != "dev" {
		return v
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		// Only a real release tag is meaningful. Pseudo-versions from a local
		// build (v0.3.3-0.2026...+dirty) would be noise in a status bar.
		if v := strings.TrimPrefix(info.Main.Version, "v"); cleanSemver.MatchString(v) {
			return v
		}
	}
	return "dev"
}

func main() {
	flagTiny := flag.Bool("tiny", false, "single-line mode for tmux/status bars")
	flagMini := flag.Bool("mini", false, "graphs-only mini mode")
	flagCompact := flag.Bool("compact", false, "numbers-only compact mode")
	flagJSON := flag.Bool("json", false, "one-shot JSON snapshot, then exit")
	flagJSONStream := flag.Bool("json-stream", false, "continuous JSON lines output")
	flagOnce := flag.Bool("once", false, "one-shot plain-text snapshot, then exit")
	flagIface := flag.String("interface", "", "force a specific network interface")
	flagRefresh := flag.Duration("refresh", 0, "sampling interval (e.g. 250ms)")
	flagNoColor := flag.Bool("no-color", false, "disable ANSI color output")
	flagBits := flag.Bool("bits", false, "display throughput in bits/sec instead of bytes/sec")
	flagFormat := flag.String("format", "", "template string for custom formatting")
	flagTheme := flag.String("theme", "", "override active theme")
	flagNoAnim := flag.Bool("no-anim", false, "disable animations (reduced motion)")
	flagView := flag.String("view", "", "start view: hero|compact|mini|tiny")
	flagWindow := flag.String("window", "", "initial time window: 1m|5m|15m|1h|24h")
	flagPing := flag.String("ping", "", "ping target host (default: 1.1.1.1)")
	flagWidth := flag.Int("width", 0, "fixed output width for --tiny/--format (status bars)")
	flagResetHistory := flag.Bool("reset-history", false, "clear persisted history, then exit")
	flagVersion := flag.Bool("version", false, "print version and exit")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "flow - See your network breathe.\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n  flow [flags]\n\nFlags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *flagVersion {
		fmt.Println("flow", buildVersion())
		return
	}

	if *flagResetHistory {
		if err := history.Reset(); err != nil {
			fmt.Fprintf(os.Stderr, "flow: reset history: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("flow: history cleared")
		return
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "flow: config error: %v\n", err)
		// Non-fatal: continue with defaults already populated.
	}

	if *flagIface != "" {
		cfg.Interface = *flagIface
	}
	if *flagRefresh != 0 {
		cfg.Refresh = config.NewDuration(*flagRefresh)
	}
	if *flagNoColor {
		cfg.NoColor = true
	}
	if cfg.NoColor {
		_ = os.Setenv("NO_COLOR", "1") // honoured by Lip Gloss automatically
	}
	if *flagBits {
		cfg.Bits = true
	}
	if *flagPing != "" {
		cfg.PingTarget = *flagPing
	}
	if *flagNoAnim {
		cfg.NoAnim = true
	}

	col := collector.New(cfg.Interface)

	refresh := cfg.RefreshDuration()
	smp := sampler.New(col, refresh)

	if *flagTheme != "" {
		cfg.Theme = *flagTheme
		theme.SetTheme(*flagTheme)
	}

	if *flagFormat != "" {
		runFormat(smp, *flagFormat, *flagJSONStream, *flagWidth, refresh)
		return
	}

	if *flagJSONStream {
		runJSONStream(smp, refresh, cfg.Bits)
		return
	}

	if *flagJSON || *flagOnce {
		runOnce(smp, *flagJSON, cfg.Bits, refresh)
		return
	}

	if *flagTiny {
		runTiny(smp, cfg.Bits)
		return
	}

	// Non-TTY stdout (pipes, scripts): default to a single tiny line.
	if !isTTY(os.Stdout) {
		runTiny(smp, cfg.Bits)
		return
	}

	ifaces, err := collector.Interfaces()
	if err != nil {
		ifaces = []string{cfg.Interface}
	}

	ctx, cancel := context.WithCancel(context.Background())
	go smp.Run(ctx)

	var forced ui.ViewMode
	switch {
	case *flagMini:
		forced = ui.ViewMini
	case *flagCompact:
		forced = ui.ViewCompact
	default:
		forced = ui.ViewHero
	}
	switch *flagView {
	case "hero":
		forced = ui.ViewHero
	case "compact":
		forced = ui.ViewCompact
	case "mini":
		forced = ui.ViewMini
	case "tiny":
		runTiny(smp, cfg.Bits)
		return
	}

	initialIface := cfg.Interface
	if initialIface == "auto" || initialIface == "" {
		initialIface = "auto"
	}

	ui.SetVersion(version)

	model := ui.New(cfg, smp, ifaces, initialIface, cancel, forced)
	if *flagWindow != "" {
		model = model.WithWindow(*flagWindow)
	}

	opts := []tea.ProgramOption{tea.WithAltScreen()}
	if *flagCompact || *flagMini {
		opts = []tea.ProgramOption{}
	}

	p := tea.NewProgram(model, opts...)
	m, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "flow: %v\n", err)
		os.Exit(1)
	}
	if finalModel, ok := m.(ui.Model); ok && finalModel.Err() != nil {
		fmt.Fprintf(os.Stderr, "flow: %v\n", finalModel.Err())
		os.Exit(1)
	}
}

func runTiny(smp *sampler.Sampler, bits bool) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go smp.Run(ctx)

	s1 := <-smp.Out
	if s1.Err != nil {
		fmt.Fprintf(os.Stderr, "flow: %v\n", s1.Err)
		os.Exit(1)
	}
	s := <-smp.Out
	if s.Err != nil {
		fmt.Fprintf(os.Stderr, "flow: %v\n", s.Err)
		os.Exit(1)
	}
	cancel()

	down := ui.FormatBpsExt(s.DownBps, ui.UnitAuto, bits)
	up := ui.FormatBpsExt(s.UpBps, ui.UnitAuto, bits)

	fmt.Printf("↓ %s · ↑ %s\n", down, up)
}

func runOnce(smp *sampler.Sampler, asJSON bool, bits bool, refresh time.Duration) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go smp.Run(ctx)

	s1 := <-smp.Out
	if s1.Err != nil {
		fmt.Fprintf(os.Stderr, "flow: %v\n", s1.Err)
		os.Exit(1)
	}
	s := <-smp.Out
	if s.Err != nil {
		fmt.Fprintf(os.Stderr, "flow: %v\n", s.Err)
		os.Exit(1)
	}
	cancel()

	// Peaks are tracked over the samples this process actually observed.
	tr := history.NewTracker()
	tr.Record(s1.DownBps, s1.UpBps, refresh.Seconds())
	tr.Record(s.DownBps, s.UpBps, refresh.Seconds())

	if asJSON {
		down := ui.FormatBpsExt(s.DownBps, ui.UnitAuto, bits)
		up := ui.FormatBpsExt(s.UpBps, ui.UnitAuto, bits)
		out := map[string]interface{}{
			"status":         "ok",
			"timestamp":      s.At.UTC().Format(time.RFC3339Nano),
			"interface":      s.Interface,
			"download_bps":   s.DownBps,
			"upload_bps":     s.UpBps,
			"download_human": down,
			"upload_human":   up,
			"peak_down_bps":  tr.PeakDown,
			"peak_up_bps":    tr.PeakUp,
			"unit_display":   autoUnitExt(s.DownBps, bits),
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(out)
		return
	}

	fmt.Printf("%s %s\n",
		ui.FormatBpsExt(s.DownBps, ui.UnitAuto, bits),
		ui.FormatBpsExt(s.UpBps, ui.UnitAuto, bits),
	)
}

func runJSONStream(smp *sampler.Sampler, refresh time.Duration, bits bool) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go smp.Run(ctx)

	enc := json.NewEncoder(os.Stdout)
	s1 := <-smp.Out
	if s1.Err != nil {
		fmt.Fprintf(os.Stderr, "flow: %v\n", s1.Err)
		os.Exit(1)
	}
	// Peaks are the highest rates observed since the stream started.
	tr := history.NewTracker()
	tr.Record(s1.DownBps, s1.UpBps, refresh.Seconds())
	for s := range smp.Out {
		if s.Err != nil {
			fmt.Fprintf(os.Stderr, "flow: %v\n", s.Err)
			os.Exit(1)
		}
		tr.Record(s.DownBps, s.UpBps, refresh.Seconds())
		_ = enc.Encode(map[string]interface{}{
			"status":         "ok",
			"timestamp":      s.At.UTC().Format(time.RFC3339Nano),
			"interface":      s.Interface,
			"download_bps":   s.DownBps,
			"upload_bps":     s.UpBps,
			"download_human": ui.FormatBpsExt(s.DownBps, ui.UnitAuto, bits),
			"upload_human":   ui.FormatBpsExt(s.UpBps, ui.UnitAuto, bits),
			"peak_down_bps":  tr.PeakDown,
			"peak_up_bps":    tr.PeakUp,
			"unit_display":   autoUnitExt(s.DownBps, bits),
		})
	}
}

// isTTY reports whether the file is a character device (terminal).
func isTTY(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// todayTotals loads persisted today totals. Missing file means zeros, never an error.
func todayTotals() (float64, float64) {
	tr := history.NewTracker()
	_ = tr.Load() // fresh day or missing file: zeros are correct
	return tr.TodayDown, tr.TodayUp
}

// padOrTruncate makes output exactly width visible chars for status bars.
func padOrTruncate(s string, width int) string {
	runes := []rune(s)
	if len(runes) > width {
		return string(runes[:width])
	}
	for len(runes) < width {
		runes = append(runes, ' ')
	}
	return string(runes)
}

func autoUnitExt(bps float64, bits bool) string {
	if bits {
		bps = bps * 8
		switch {
		case bps >= 1_073_741_824:
			return "Gb/s"
		case bps >= 1_048_576:
			return "Mb/s"
		case bps >= 1024:
			return "Kb/s"
		default:
			return "b/s"
		}
	}
	switch {
	case bps >= 1_073_741_824:
		return "GB/s"
	case bps >= 1_048_576:
		return "MB/s"
	case bps >= 1024:
		return "KB/s"
	default:
		return "B/s"
	}
}

func runFormat(smp *sampler.Sampler, tmplStr string, stream bool, width int, refresh time.Duration) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go smp.Run(ctx)

	s1 := <-smp.Out
	if s1.Err != nil {
		fmt.Fprintf(os.Stderr, "flow: %v\n", s1.Err)
		os.Exit(1)
	}
	tr := history.NewTracker()
	tr.Record(s1.DownBps, s1.UpBps, refresh.Seconds())

	for s := range smp.Out {
		if s.Err != nil {
			fmt.Fprintf(os.Stderr, "flow: %v\n", s.Err)
			os.Exit(1)
		}
		tr.Record(s.DownBps, s.UpBps, refresh.Seconds())
		todayDown, todayUp := todayTotals()
		data := format.Data{
			Iface:       s.Interface,
			DownBps:     s.DownBps,
			UpBps:       s.UpBps,
			Down:        ui.FormatBpsExt(s.DownBps, ui.UnitAuto, false),
			Up:          ui.FormatBpsExt(s.UpBps, ui.UnitAuto, false),
			PeakDownBps: tr.PeakDown,
			PeakUpBps:   tr.PeakUp,
			TodayDown:   format.Bytes(todayDown),
			TodayUp:     format.Bytes(todayUp),
			Time:        s.At.UTC().Format(time.RFC3339Nano),
		}
		out, err := format.RenderTemplate(tmplStr, data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "flow: template error: %v\n", err)
			os.Exit(2)
		}
		if width > 0 {
			out = padOrTruncate(out, width)
		}
		fmt.Println(out)
		if !stream {
			return
		}
	}
}

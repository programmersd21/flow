// cmd/flow/main.go
//
// Entry point: flag parsing, configuration wiring, TUI or non-interactive mode.

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/programmersd21/flow/internal/collector"
	"github.com/programmersd21/flow/internal/config"
	"github.com/programmersd21/flow/internal/sampler"
	"github.com/programmersd21/flow/internal/ui"
)

var version = "dev"

func main() {
	flagTiny := flag.Bool("tiny", false, "single-line mode for tmux/status bars")
	flagCompact := flag.Bool("compact", false, "numbers-only compact mode")
	flagJSON := flag.Bool("json", false, "one-shot JSON snapshot, then exit")
	flagOnce := flag.Bool("once", false, "one-shot plain-text snapshot, then exit")
	flagIface := flag.String("interface", "", "force a specific network interface")
	flagRefresh := flag.Duration("refresh", 0, "sampling interval (e.g. 250ms)")
	flagNoColor := flag.Bool("no-color", false, "disable ANSI color output")
	flagVersion := flag.Bool("version", false, "print version and exit")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "flow — See your network breathe.\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n  flow [flags]\n\nFlags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *flagVersion {
		fmt.Println("flow", version)
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
		_ = os.Setenv("NO_COLOR", "1") // honoured by Lip Gloss automatically
	}

	col := collector.New(cfg.Interface)

	refresh := cfg.RefreshDuration()
	smp := sampler.New(col, refresh)

	if *flagJSON || *flagOnce {
		runOnce(col, smp, refresh, *flagJSON)
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
	case *flagTiny:
		forced = ui.ViewTiny
	case *flagCompact:
		forced = ui.ViewCompact
	default:
		forced = ui.ViewHero
	}

	// Determine initial interface name (first sample may correct it).
	initialIface := cfg.Interface
	if initialIface == "auto" || initialIface == "" {
		initialIface = "auto"
	}

	model := ui.New(cfg, smp, ifaces, initialIface, cancel, forced)

	opts := []tea.ProgramOption{tea.WithAltScreen()}
	if *flagTiny || *flagCompact {
		opts = []tea.ProgramOption{}
	}

	p := tea.NewProgram(model, opts...)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "flow: %v\n", err)
		os.Exit(1)
	}
}

// runOnce takes exactly one sample and either prints JSON or plain text, then
// exits. Does not start the TUI.
func runOnce(col *collector.Collector, smp *sampler.Sampler, refresh time.Duration, asJSON bool) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go smp.Run(ctx)

	// Wait for two samples so the first diff is valid.
	<-smp.Out
	s := <-smp.Out
	cancel()

	if asJSON {
		out := map[string]interface{}{
			"download_bps":  s.DownBps,
			"upload_bps":    s.UpBps,
			"peak_down_bps": s.DownBps, // peak = current in one-shot mode
			"peak_up_bps":   s.UpBps,
			"interface":     s.Interface,
			"unit_display":  autoUnit(s.DownBps),
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(out)
		return
	}

	fmt.Printf("%s %s\n",
		ui.FormatBps(s.DownBps, ui.UnitAuto),
		ui.FormatBps(s.UpBps, ui.UnitAuto),
	)
}

func autoUnit(bps float64) string {
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

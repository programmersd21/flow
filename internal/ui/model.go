package ui

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/programmersd21/flow/internal/animate"
	"github.com/programmersd21/flow/internal/collector"
	"github.com/programmersd21/flow/internal/config"
	"github.com/programmersd21/flow/internal/history"
	"github.com/programmersd21/flow/internal/ping"
	"github.com/programmersd21/flow/internal/processes"
	"github.com/programmersd21/flow/internal/sampler"
	"github.com/programmersd21/flow/internal/theme"
)

type tickMsg struct{}

type sampleMsg sampler.Sample

type processesMsg []processes.Info

type pingMsg time.Duration

type ifaceDetailMsg struct {
	detail *collector.InterfaceDetail
	err    error
}

type DisplayFilter int

const (
	DisplayBoth DisplayFilter = iota
	DisplayDownOnly
	DisplayUpOnly
)

const (
	slopeWindow         = 6
	resetConfirmTimeout = 2 * time.Second
)

type ViewMode int

const (
	ViewHero ViewMode = iota
	ViewCompact
	ViewMini
	ViewTiny
)

type UnitMode int

const (
	UnitAuto UnitMode = iota
	UnitKB
	UnitMB
	UnitGB
)

type Model struct {
	keys KeyMap
	cfg  config.Config

	smp        *sampler.Sampler
	samplerCtx context.CancelFunc

	ifaces    []string
	ifaceIdx  int
	ifaceName string

	dispDown, dispUp       float64
	animDown, animUp       float64
	animDownVel, animUpVel float64

	rollingMaxDown, rollingMaxUp float64

	downHist *history.Ring
	upHist   *history.Ring
	tracker  *history.Tracker

	paused                 bool
	showHelp               bool
	showProcesses          bool
	showThemes             bool
	themeSelectionIdx      int
	themeSelectionOriginal string
	unitMode               UnitMode
	viewMode               ViewMode
	bitsMode               bool
	displayFilter          DisplayFilter
	procs                  []processes.Info

	width, height   int
	refreshInterval time.Duration
	lastSampleTime  time.Time

	samplePulse     float64
	downPulse       float64
	upPulse         float64
	pingLatency     time.Duration
	ifaceDetails    *collector.InterfaceDetail
	showIfaceDetail bool
	resetConfirm    bool
	resetConfirmAt  time.Time
	err             error
}

func New(
	cfg config.Config,
	smp *sampler.Sampler,
	ifaces []string,
	initialIface string,
	cancelFn context.CancelFunc,
	forced ViewMode,
) Model {
	histCap := cfg.History * 4
	if histCap < 60 {
		histCap = 60
	}

	var unitMode UnitMode
	switch strings.ToLower(cfg.Unit) {
	case "kb":
		unitMode = UnitKB
	case "mb":
		unitMode = UnitMB
	case "gb":
		unitMode = UnitGB
	default:
		unitMode = UnitAuto
	}

	theme.SetTheme(cfg.Theme)

	return Model{
		keys:            DefaultKeyMap(),
		cfg:             cfg,
		smp:             smp,
		samplerCtx:      cancelFn,
		ifaces:          ifaces,
		ifaceName:       initialIface,
		downHist:        history.New(histCap),
		upHist:          history.New(histCap),
		tracker:         loadTracker(),
		unitMode:        unitMode,
		viewMode:        forced,
		bitsMode:        cfg.Bits,
		refreshInterval: cfg.RefreshDuration(),
		lastSampleTime:  time.Now(),
		samplePulse:     1.0,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(waitForSample(m.smp.Out), tick(), refreshProcesses(), m.quickPing(), m.pingTick())
}

func (m Model) quickPing() tea.Cmd {
	target := m.cfg.PingTarget
	if target == "" {
		target = "1.1.1.1"
	}
	return func() tea.Msg {
		latency, err := ping.Measure(target, 2*time.Second)
		if err != nil {
			return pingMsg(0)
		}
		return pingMsg(latency)
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		return m.handleKey(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		m.samplePulse = math.Max(0, m.samplePulse-0.15)
		m.downPulse = math.Max(0, m.downPulse-0.1)
		m.upPulse = math.Max(0, m.upPulse-0.1)
		m.animDown = animate.Spring(m.animDown, m.dispDown, &m.animDownVel, 0.13)
		m.animUp = animate.Spring(m.animUp, m.dispUp, &m.animUpVel, 0.13)
		if m.resetConfirm && time.Since(m.resetConfirmAt) > resetConfirmTimeout {
			m.resetConfirm = false
		}
		return m, tick()

	case ifaceDetailMsg:
		if msg.err != nil {
			m.showIfaceDetail = false
		} else {
			m.ifaceDetails = msg.detail
		}
		return m, nil

	case processesMsg:
		m.procs = msg
		return m, nil

	case pingMsg:
		m.pingLatency = time.Duration(msg)
		return m, nil

	case sampleMsg:
		if msg.Err != nil {
			m.err = msg.Err
			m.samplerCtx()
			return m, tea.Quit
		}
		if !m.paused {
			m.samplePulse = 1.0
			m.lastSampleTime = time.Now()
			if msg.DownBps > m.tracker.PeakDown && m.tracker.PeakDown > 0 {
				m.downPulse = 1.0
			}
			if msg.UpBps > m.tracker.PeakUp && m.tracker.PeakUp > 0 {
				m.upPulse = 1.0
			}

			m.dispDown = msg.DownBps
			m.dispUp = msg.UpBps
			m.tracker.Record(msg.DownBps, msg.UpBps, m.refreshInterval.Seconds())
			m.downHist.Push(msg.DownBps)
			m.upHist.Push(msg.UpBps)
			m.updateRollingMax(msg.DownBps, msg.UpBps)
			m.ifaceName = msg.Interface
		}
		return m, waitForSample(m.smp.Out)
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.showThemes {
		switch msg.String() {
		case "q", "ctrl+c":
			m.samplerCtx()
			_ = m.tracker.Save()
			return m, tea.Quit
		case "esc":
			theme.SetTheme(m.themeSelectionOriginal)
			m.showThemes = false
			return m, nil
		case "up", "k":
			themes := theme.ListThemes()
			m.themeSelectionIdx = (m.themeSelectionIdx - 1 + len(themes)) % len(themes)
			theme.SetTheme(themes[m.themeSelectionIdx].Name)
			return m, nil
		case "down", "j":
			themes := theme.ListThemes()
			m.themeSelectionIdx = (m.themeSelectionIdx + 1) % len(themes)
			theme.SetTheme(themes[m.themeSelectionIdx].Name)
			return m, nil
		case "enter":
			themes := theme.ListThemes()
			selectedTheme := themes[m.themeSelectionIdx].Name
			m.cfg.Theme = selectedTheme
			_ = config.Save(m.cfg)
			m.showThemes = false
			return m, nil
		}
		return m, nil
	}

	if key.Matches(msg, m.keys.Esc) {
		if m.showIfaceDetail {
			m.showIfaceDetail = false
			return m, nil
		}
		if m.showHelp {
			m.showHelp = false
			return m, nil
		}
		if m.showProcesses {
			m.showProcesses = false
			return m, nil
		}
		if m.resetConfirm {
			m.resetConfirm = false
			return m, nil
		}
	}

	switch {
	case key.Matches(msg, m.keys.Quit):
		m.samplerCtx()
		_ = m.tracker.Save()
		return m, tea.Quit

	case key.Matches(msg, m.keys.Help):
		if !m.showHelp {
			m.showHelp = true
			m.showProcesses = false
		}

	case key.Matches(msg, m.keys.Mode):
		m.viewMode = (m.viewMode + 1) % 4

	case key.Matches(msg, m.keys.Processes):
		if !m.showProcesses {
			m.showProcesses = true
			m.showHelp = false
			return m, refreshProcesses()
		}

	case key.Matches(msg, m.keys.Pause):
		m.paused = !m.paused

	case key.Matches(msg, m.keys.Reset):
		if m.resetConfirm {
			m.tracker.ResetPeaks()
			m.tracker.TodayDown = 0
			m.tracker.TodayUp = 0
			m.rollingMaxDown = 0
			m.rollingMaxUp = 0
			m.dispDown = 0
			m.dispUp = 0
			m.downHist.Reset()
			m.upHist.Reset()
			m.downPulse = 0
			m.upPulse = 0
			m.resetConfirm = false
		} else {
			m.resetConfirm = true
			m.resetConfirmAt = time.Now()
		}

	case key.Matches(msg, m.keys.Unit):
		m.unitMode = (m.unitMode + 1) % 4

	case key.Matches(msg, m.keys.InterfaceInfo):
		m.showIfaceDetail = true
		return m, refreshIfaceDetails(m.ifaceName)

	case key.Matches(msg, m.keys.Interface):
		if len(m.ifaces) <= 1 {
			m.showIfaceDetail = true
			return m, refreshIfaceDetails(m.ifaceName)
		}
		m.ifaceIdx = (m.ifaceIdx + 1) % len(m.ifaces)
		newIface := m.ifaces[m.ifaceIdx]
		m.ifaceName = newIface
		m.dispDown = 0
		m.dispUp = 0
		m.rollingMaxDown = 0
		m.rollingMaxUp = 0
		m.downHist = history.New(m.downHist.Cap())
		m.upHist = history.New(m.upHist.Cap())
		m.samplerCtx()
		ctx, cancel := context.WithCancel(context.Background())
		m.samplerCtx = cancel
		col := collector.New(newIface)
		m.smp = sampler.New(col, m.refreshInterval)
		go m.smp.Run(ctx)
		return m, waitForSample(m.smp.Out)

	case key.Matches(msg, m.keys.Bits):
		m.bitsMode = !m.bitsMode

	case key.Matches(msg, m.keys.Display):
		m.displayFilter = (m.displayFilter + 1) % 3

	case key.Matches(msg, m.keys.Faster):
		m.adjustRefreshInterval(true)
		return m, waitForSample(m.smp.Out)

	case key.Matches(msg, m.keys.Slower):
		m.adjustRefreshInterval(false)
		return m, waitForSample(m.smp.Out)

	case key.Matches(msg, m.keys.Themes):
		m.showThemes = true
		m.themeSelectionOriginal = m.cfg.Theme
		themes := theme.ListThemes()
		m.themeSelectionIdx = 0
		for i, t := range themes {
			if t.Name == m.cfg.Theme {
				m.themeSelectionIdx = i
				break
			}
		}
		m.showHelp = false
		m.showProcesses = false

	default:
	}

	return m, nil
}

func (m *Model) adjustRefreshInterval(faster bool) {
	intervals := []time.Duration{
		50 * time.Millisecond,
		100 * time.Millisecond,
		250 * time.Millisecond,
		500 * time.Millisecond,
		1 * time.Second,
		2 * time.Second,
		3 * time.Second,
		5 * time.Second,
		10 * time.Second,
		30 * time.Second,
		60 * time.Second,
		300 * time.Second,
	}

	idx := -1
	for i, d := range intervals {
		if m.refreshInterval == d {
			idx = i
			break
		}
	}
	if idx == -1 {
		idx = 1
	}

	if faster {
		if idx > 0 {
			idx--
		}
	} else {
		if idx < len(intervals)-1 {
			idx++
		}
	}

	newInterval := intervals[idx]
	if newInterval != m.refreshInterval {
		m.refreshInterval = newInterval
		m.samplerCtx()
		ctx, cancel := context.WithCancel(context.Background())
		m.samplerCtx = cancel
		col := collector.New(m.ifaceName)
		m.smp = sampler.New(col, m.refreshInterval)
		go m.smp.Run(ctx)
	}
}

func (m Model) View() string {
	if m.showThemes {
		return renderThemes(m)
	}
	if m.showHelp {
		return renderHelp(m)
	}
	if m.showProcesses {
		return renderProcesses(m)
	}
	if m.showIfaceDetail {
		return renderIfaceDetails(m)
	}
	mode, lines := pickViewModeAndContent(m)
	if mode == ViewTiny {
		return renderTiny(m)
	}
	termW := m.width
	termH := m.height
	if termH <= 0 {
		termH = 24
	}
	return centerFrame(strings.Join(lines, "\n"), termW, termH)
}

func (m Model) effectiveViewMode() ViewMode {
	mode, _ := pickViewModeAndContent(m)
	return mode
}

func (m *Model) updateRollingMax(down, up float64) {
	const decay = 0.995
	m.rollingMaxDown *= decay
	m.rollingMaxUp *= decay
	if down > m.rollingMaxDown {
		m.rollingMaxDown = down
	}
	if up > m.rollingMaxUp {
		m.rollingMaxUp = up
	}
}

func (m Model) FormatBps(bps float64) string {
	return FormatBpsExt(bps, m.unitMode, m.bitsMode)
}

func FormatBps(bps float64, unit UnitMode) string {
	return FormatBpsExt(bps, unit, false)
}

func FormatBpsExt(bps float64, unit UnitMode, bits bool) string {
	if bps < 0 {
		bps = 0
	}
	if bits {
		bps = bps * 8
	}
	suffix := "B/s"
	if bits {
		suffix = "b/s"
	}

	switch unit {
	case UnitKB:
		unitName := "KB/s"
		if bits {
			unitName = "Kb/s"
		}
		return fmt.Sprintf("%.1f %s", bps/1024, unitName)
	case UnitMB:
		unitName := "MB/s"
		if bits {
			unitName = "Mb/s"
		}
		return fmt.Sprintf("%.1f %s", bps/1_048_576, unitName)
	case UnitGB:
		unitName := "GB/s"
		if bits {
			unitName = "Gb/s"
		}
		return fmt.Sprintf("%.3f %s", bps/1_073_741_824, unitName)
	default:
		switch {
		case bps >= 1_073_741_824:
			unitName := "GB/s"
			if bits {
				unitName = "Gb/s"
			}
			return fmt.Sprintf("%.2f %s", bps/1_073_741_824, unitName)
		case bps >= 1_048_576:
			unitName := "MB/s"
			if bits {
				unitName = "Mb/s"
			}
			return fmt.Sprintf("%.1f %s", bps/1_048_576, unitName)
		case bps >= 1024:
			unitName := "KB/s"
			if bits {
				unitName = "Kb/s"
			}
			return fmt.Sprintf("%.0f %s", bps/1024, unitName)
		default:
			return fmt.Sprintf("%.0f %s", bps, suffix)
		}
	}
}

func FormatBpsFixedWidth(bps float64, unit UnitMode, bits bool) string {
	valStr := FormatBpsExt(bps, unit, bits)
	return fmt.Sprintf("%10s", valStr)
}

func waitForSample(ch <-chan sampler.Sample) tea.Cmd {
	return func() tea.Msg { return sampleMsg(<-ch) }
}

func tick() tea.Cmd {
	return tea.Tick(130*time.Millisecond, func(time.Time) tea.Msg {
		return tickMsg{}
	})
}

func (m Model) pingTick() tea.Cmd {
	target := m.cfg.PingTarget
	if target == "" {
		target = "1.1.1.1"
	}
	return tea.Tick(5*time.Second, func(time.Time) tea.Msg {
		latency, err := ping.Measure(target, 2*time.Second)
		if err != nil {
			return pingMsg(0)
		}
		return pingMsg(latency)
	})
}

func (m Model) Err() error {
	return m.err
}

func refreshProcesses() tea.Cmd {
	return func() tea.Msg {
		list, err := processes.List()
		if err != nil {
			return processesMsg(nil)
		}
		if list == nil {
			return processesMsg(nil)
		}
		return processesMsg(list)
	}
}

func refreshIfaceDetails(ifaceName string) tea.Cmd {
	return func() tea.Msg {
		detail, err := collector.InterfaceDetails(ifaceName)
		if err != nil {
			return ifaceDetailMsg{err: err}
		}
		return ifaceDetailMsg{detail: detail}
	}
}

func loadTracker() *history.Tracker {
	t := history.NewTracker()
	_ = t.Load()
	return t
}

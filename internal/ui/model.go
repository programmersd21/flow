package ui

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/programmersd21/flow/internal/collector"
	"github.com/programmersd21/flow/internal/config"
	"github.com/programmersd21/flow/internal/history"
	"github.com/programmersd21/flow/internal/sampler"
)

type tickMsg struct{}

type sampleMsg sampler.Sample

const (
	slopeWindow = 6
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

	dispDown, dispUp float64

	rollingMaxDown, rollingMaxUp float64

	downHist *history.Ring
	upHist   *history.Ring
	tracker  *history.Tracker

	paused   bool
	showHelp bool
	unitMode UnitMode
	viewMode ViewMode

	width, height   int
	refreshInterval time.Duration
	lastSampleTime  time.Time

	breathe   float64
	downPulse float64
	upPulse   float64
	err       error
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

	return Model{
		keys:            DefaultKeyMap(),
		cfg:             cfg,
		smp:             smp,
		samplerCtx:      cancelFn,
		ifaces:          ifaces,
		ifaceName:       initialIface,
		downHist:        history.New(histCap),
		upHist:          history.New(histCap),
		tracker:         history.NewTracker(),
		unitMode:        unitMode,
		viewMode:        forced,
		refreshInterval: cfg.RefreshDuration(),
		lastSampleTime:  time.Now(),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(waitForSample(m.smp.Out), tick())
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
		m.breathe = 0.5 + 0.5*math.Sin(float64(time.Now().UnixMilli())/500)
		m.downPulse = math.Max(0, m.downPulse-0.08)
		m.upPulse = math.Max(0, m.upPulse-0.08)
		return m, tick()

	case sampleMsg:
		if msg.Err != nil {
			m.err = msg.Err
			m.samplerCtx()
			return m, tea.Quit
		}
		if !m.paused {
			m.lastSampleTime = time.Now()

			// Trigger pulse on new peaks
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
	switch {
	case key.Matches(msg, m.keys.Quit):
		m.samplerCtx()
		return m, tea.Quit

	case key.Matches(msg, m.keys.Help):
		m.showHelp = !m.showHelp

	case key.Matches(msg, m.keys.Mode):
		m.viewMode = (m.viewMode + 1) % 4

	case key.Matches(msg, m.keys.Pause):
		m.paused = !m.paused

	case key.Matches(msg, m.keys.Reset):
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

	case key.Matches(msg, m.keys.Unit):
		m.unitMode = (m.unitMode + 1) % 4

	case key.Matches(msg, m.keys.Interface):
		if len(m.ifaces) > 1 {
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
		}

	default:
		if m.showHelp {
			m.showHelp = false
		}
	}

	return m, nil
}

func (m Model) View() string {
	if m.showHelp {
		return renderHelp(m)
	}
	switch m.effectiveViewMode() {
	case ViewTiny:
		return renderTiny(m)
	case ViewMini:
		return renderMini(m)
	case ViewCompact:
		return renderCompact(m)
	default:
		return renderHero(m)
	}
}

func (m Model) effectiveViewMode() ViewMode {
	if m.viewMode == ViewTiny {
		return ViewTiny
	}
	if m.viewMode == ViewMini {
		return ViewMini
	}
	if m.viewMode == ViewCompact {
		return ViewCompact
	}

	if m.width > 0 && m.width < 40 {
		return ViewTiny
	}
	if m.height > 0 && m.height < 6 {
		return ViewTiny
	}
	if m.width > 0 && m.width < 60 {
		return ViewCompact
	}
	if m.height > 0 && m.height < 16 {
		return ViewMini
	}
	if m.height > 0 && m.height < 22 {
		return ViewCompact
	}
	return ViewHero
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
	return FormatBps(bps, m.unitMode)
}

func FormatBps(bps float64, unit UnitMode) string {
	if bps < 0 {
		bps = 0
	}

	switch unit {
	case UnitKB:
		return fmt.Sprintf("%.1f KB/s", bps/1024)
	case UnitMB:
		return fmt.Sprintf("%.1f MB/s", bps/1_048_576)
	case UnitGB:
		return fmt.Sprintf("%.3f GB/s", bps/1_073_741_824)
	default:
		switch {
		case bps >= 1_073_741_824:
			return fmt.Sprintf("%.2f GB/s", bps/1_073_741_824)
		case bps >= 1_048_576:
			return fmt.Sprintf("%.1f MB/s", bps/1_048_576)
		case bps >= 1024:
			return fmt.Sprintf("%.0f KB/s", bps/1024)
		default:
			return fmt.Sprintf("%.0f B/s", bps)
		}
	}
}

func waitForSample(ch <-chan sampler.Sample) tea.Cmd {
	return func() tea.Msg { return sampleMsg(<-ch) }
}

func tick() tea.Cmd {
	return tea.Tick(130*time.Millisecond, func(time.Time) tea.Msg {
		return tickMsg{}
	})
}

func (m Model) Err() error {
	return m.err
}

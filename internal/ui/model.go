package ui

import (
	"context"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/programmersd21/flow/internal/animate"
	"github.com/programmersd21/flow/internal/collector"
	"github.com/programmersd21/flow/internal/config"
	"github.com/programmersd21/flow/internal/history"
	"github.com/programmersd21/flow/internal/ping"
	"github.com/programmersd21/flow/internal/processes"
	"github.com/programmersd21/flow/internal/render"
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

// appVersion is set from the main package via SetVersion so the UI can
// surface it in the title. Defaults to "dev" for local builds.
var appVersion = "dev"

// SetVersion records the build version for display inside the UI.
func SetVersion(v string) {
	if v != "" {
		appVersion = v
	}
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
	crossfadeSeconds    = 0.18
)

// timeWindows are the graph time windows in seconds, cycled by the w key.
var timeWindows = []int{60, 300, 900, 3600, 86400}

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

// rateState is the pipeline that turns raw samples into what the screen shows:
// the sampled value, its spring animation, a decaying ceiling for scaling, and
// the short-lived pulses used for emphasis. Keeping it together makes the value
// pipeline legible in one place.
type rateState struct {
	// disp is the latest sampled rate.
	dispDown, dispUp float64
	// anim is the spring-animated value actually rendered.
	animDown, animUp float64
	// spring velocity, carried between ticks.
	velDown, velUp float64
	// rollingMax decays so the graph ceiling follows recent peaks rather than
	// a single spike seen hours ago.
	rollingMaxDown, rollingMaxUp float64
	// pulses decay after a peak and are used for emphasis.
	samplePulse, downPulse, upPulse float64
	// lastSample is when the newest sample arrived, for scroll and staleness.
	lastSample  time.Time
	pingLatency time.Duration
}

// overlayState is whatever is drawn on top of the dashboard. It answers exactly
// one question — what does the user see right now — and nothing else.
type overlayState struct {
	help        bool
	helpScroll  int
	processes   bool
	themes      bool
	themeIndex  int
	themeBefore string
	iface       bool
	ifaceDetail *collector.InterfaceDetail
	toast       string
	toastAt     time.Time
}

// ifaceState tracks which interface is being displayed and what else exists.
type ifaceState struct {
	all  []string
	idx  int
	name string
}

// Model is the dashboard state and the single owner of UI state. It receives
// samples and key presses from Bubble Tea and renders a view; it owns no
// sampling of its own — the sampler goroutine owns cadence and the collector
// owns OS access.
type Model struct {
	keys KeyMap
	cfg  config.Config

	// sampling: the sampler goroutine produces into smp.Out.
	smp        *sampler.Sampler
	samplerCtx context.CancelFunc

	rates rateState
	over  overlayState
	iface ifaceState

	downHist *history.Ring
	upHist   *history.Ring
	tracker  *history.Tracker
	procs    []processes.Info

	viewMode      ViewMode
	unitMode      UnitMode
	displayFilter DisplayFilter
	bitsMode      bool
	paused        bool

	width, height   int
	refreshInterval time.Duration

	resetConfirm   bool
	resetConfirmAt time.Time
	err            error

	// graph presentation
	windowSecs  int
	windowIdx   int
	scaleMode   int
	showGrid    bool
	noAnim      bool
	glyphSet    render.GlyphSet
	colorCap    render.ColorCap
	launchStart time.Time
	launchDone  bool
	themeFadeAt time.Time
	themeFading bool

	// rippleDownAt and rippleUpAt mark the last burst on each direction.
	rippleDownAt, rippleUpAt time.Time
	// nowOverride fixes the clock in tests so frames are deterministic.
	nowOverride time.Time
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
	// Ensure the ring can hold the longest practical window at 100ms refresh
	// (1h = 36000 samples = ~576 KB for two rings). 24h windows show available data.
	if histCap < 36000 {
		histCap = 36000
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

	noAnim := !cfg.AnimationsEnabled() || os.Getenv("FLOW_REDUCE_MOTION") == "1"

	glyphSet := render.ParseGlyphSet(cfg.UI.Glyphs)
	if os.Getenv("FLOW_GLYPHS") != "" {
		glyphSet = render.ParseGlyphSet(os.Getenv("FLOW_GLYPHS"))
	}
	colorCap := render.DetectColorCap(cfg.NoColor)

	now := time.Now()
	return Model{
		keys:            DefaultKeyMap(),
		glyphSet:        glyphSet,
		colorCap:        colorCap,
		cfg:             cfg,
		smp:             smp,
		samplerCtx:      cancelFn,
		iface:           ifaceState{all: ifaces, name: initialIface},
		downHist:        history.New(histCap),
		upHist:          history.New(histCap),
		tracker:         loadTracker(),
		unitMode:        unitMode,
		viewMode:        forced,
		bitsMode:        cfg.Bits,
		refreshInterval: cfg.RefreshDuration(),
		rates:           rateState{lastSample: now, samplePulse: 1.0},
		windowSecs:      cfg.WindowSeconds(),
		showGrid:        cfg.GridlinesEnabled(),
		noAnim:          noAnim,
		launchStart:     now,
		launchDone:      noAnim || !cfg.LaunchAnimationEnabled(),
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
		m.rates.samplePulse = math.Max(0, m.rates.samplePulse-0.15)
		m.rates.downPulse = math.Max(0, m.rates.downPulse-0.1)
		m.rates.upPulse = math.Max(0, m.rates.upPulse-0.1)
		m.rates.animDown = animate.Spring(m.rates.animDown, m.rates.dispDown, &m.rates.velDown, 0.13)
		m.rates.animUp = animate.Spring(m.rates.animUp, m.rates.dispUp, &m.rates.velUp, 0.13)
		if m.resetConfirm && time.Since(m.resetConfirmAt) > resetConfirmTimeout {
			m.resetConfirm = false
		}
		if !m.launchDone && time.Since(m.launchStart) > 700*time.Millisecond {
			m.launchDone = true
		}
		if m.themeFading && time.Since(m.themeFadeAt) > 180*time.Millisecond {
			m.themeFading = false
			theme.SetCrossfade(false)
		}
		if m.over.toast != "" && time.Since(m.over.toastAt) > 1500*time.Millisecond {
			m.over.toast = ""
		}
		return m, tick()

	case ifaceDetailMsg:
		if msg.err != nil {
			m.over.iface = false
		} else {
			m.over.ifaceDetail = msg.detail
		}
		return m, nil

	case processesMsg:
		m.procs = msg
		return m, nil

	case pingMsg:
		m.rates.pingLatency = time.Duration(msg)
		return m, m.pingTick()

	case sampleMsg:
		if msg.Err != nil {
			m.err = msg.Err
			m.samplerCtx()
			return m, tea.Quit
		}
		if !m.paused {
			m.rates.samplePulse = 1.0
			m.rates.lastSample = time.Now()
			// Compare against the peaks before recording, so a new peak can
			// pulse. No "> 0" guard: after `r` resets the peaks to zero, the
			// first sample is a genuine new peak and must register as one.
			if msg.DownBps > m.tracker.PeakDown {
				m.rates.downPulse = 1.0
			}
			if msg.UpBps > m.tracker.PeakUp {
				m.rates.upPulse = 1.0
			}

			// Burst detection: sample > 3x trailing median and above 50 KB/s floor.
			if !m.noAnim {
				const burstFloor = 50 * 1000
				if msg.DownBps > burstFloor && msg.DownBps > 3*trailingMedian(m.downHist) &&
					time.Since(m.rippleDownAt) > 500*time.Millisecond {
					m.rippleDownAt = time.Now()
				}
				if msg.UpBps > burstFloor && msg.UpBps > 3*trailingMedian(m.upHist) &&
					time.Since(m.rippleUpAt) > 500*time.Millisecond {
					m.rippleUpAt = time.Now()
				}
			}

			m.rates.dispDown = msg.DownBps
			m.rates.dispUp = msg.UpBps
			// The sampler reports the seconds its rates actually cover; use
			// that, not the configured refresh, so daily totals stay right when
			// ticks are late.
			interval := msg.Interval
			if interval <= 0 {
				interval = m.refreshInterval.Seconds()
			}
			m.tracker.Record(msg.DownBps, msg.UpBps, interval)
			m.downHist.Push(msg.DownBps)
			m.upHist.Push(msg.UpBps)
			m.updateRollingMax(msg.DownBps, msg.UpBps)
			m.iface.name = msg.Interface
		}
		return m, waitForSample(m.smp.Out)
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Any key skips the launch animation.
	if !m.launchDone {
		m.launchDone = true
	}

	if m.over.themes {
		switch msg.String() {
		case "q", "ctrl+c":
			m.samplerCtx()
			_ = m.tracker.Save()
			return m, tea.Quit
		case "esc":
			theme.SetTheme(m.over.themeBefore)
			m.over.themes = false
			return m, nil
		case "up", "k":
			themes := theme.ListThemes()
			m.over.themeIndex = (m.over.themeIndex - 1 + len(themes)) % len(themes)
			theme.SetTheme(themes[m.over.themeIndex].Name)
			return m, nil
		case "down", "j":
			themes := theme.ListThemes()
			m.over.themeIndex = (m.over.themeIndex + 1) % len(themes)
			theme.SetTheme(themes[m.over.themeIndex].Name)
			return m, nil
		case "enter":
			themes := theme.ListThemes()
			selectedTheme := themes[m.over.themeIndex].Name
			m.cfg.Theme = selectedTheme
			_ = config.Save(m.cfg)
			m.over.themes = false
			if !m.noAnim && selectedTheme != m.over.themeBefore {
				m.themeFading = true
				m.themeFadeAt = time.Now()
				theme.SetCrossfade(true)
			}
			return m, nil
		}
		return m, nil
	}

	if m.over.help {
		switch msg.String() {
		case "esc", "?", "q":
			m.over.help = false
			m.over.helpScroll = 0
			return m, nil
		case "up", "k":
			if m.over.helpScroll > 0 {
				m.over.helpScroll--
			}
			return m, nil
		case "down", "j":
			m.over.helpScroll++
			return m, nil
		case "pgup":
			m.over.helpScroll -= 5
			if m.over.helpScroll < 0 {
				m.over.helpScroll = 0
			}
			return m, nil
		case "pgdown", " ":
			m.over.helpScroll += 5
			return m, nil
		case "home":
			m.over.helpScroll = 0
			return m, nil
		case "end":
			m.over.helpScroll = 1 << 30 // clamped in render
			return m, nil
		}
		// Swallow all other keys so view-mode / quit / etc. don't fire
		// behind the overlay. Esc/?/q above are the way out.
		return m, nil
	}

	if key.Matches(msg, m.keys.Esc) {
		if m.over.iface {
			m.over.iface = false
			return m, nil
		}
		if m.over.processes {
			m.over.processes = false
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
		if !m.over.help {
			m.over.help = true
			m.over.helpScroll = 0
			m.over.processes = false
		} else {
			m.over.help = false
			m.over.helpScroll = 0
		}

	case key.Matches(msg, m.keys.Mode):
		m.viewMode = (m.viewMode + 1) % 4

	case key.Matches(msg, m.keys.Processes):
		if !m.over.processes {
			m.over.processes = true
			m.over.help = false
			return m, refreshProcesses()
		}

	case key.Matches(msg, m.keys.Pause):
		m.paused = !m.paused

	case key.Matches(msg, m.keys.Reset):
		if m.over.toast != "" && time.Since(m.over.toastAt) > resetConfirmTimeout {
			// a stale toast must not mask the two-press flow
			m.over.toast = ""
		}
		if m.resetConfirm {
			m.tracker.ResetPeaks()
			m.tracker.TodayDown = 0
			m.tracker.TodayUp = 0
			m.rates.rollingMaxDown = 0
			m.rates.rollingMaxUp = 0
			m.rates.dispDown = 0
			m.rates.dispUp = 0
			m.downHist.Reset()
			m.upHist.Reset()
			m.rates.downPulse = 0
			m.rates.upPulse = 0
			m.resetConfirm = false
		} else {
			m.resetConfirm = true
			m.resetConfirmAt = time.Now()
		}

	case key.Matches(msg, m.keys.Unit):
		m.unitMode = (m.unitMode + 1) % 4

	case key.Matches(msg, m.keys.InterfaceInfo):
		m.over.iface = true
		return m, refreshIfaceDetails(m.iface.name)

	case key.Matches(msg, m.keys.Interface):
		if len(m.iface.all) <= 1 {
			m.over.iface = true
			return m, refreshIfaceDetails(m.iface.name)
		}
		m.iface.idx = (m.iface.idx + 1) % len(m.iface.all)
		newIface := m.iface.all[m.iface.idx]
		m.iface.name = newIface
		m.rates.dispDown = 0
		m.rates.dispUp = 0
		m.rates.rollingMaxDown = 0
		m.rates.rollingMaxUp = 0
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

	case key.Matches(msg, m.keys.Window):
		m.windowIdx = (m.windowIdx + 1) % len(timeWindows)
		m.windowSecs = timeWindows[m.windowIdx]

	case key.Matches(msg, m.keys.Snapshot):
		if path, err := m.exportSnapshot(); err == nil {
			m.over.toast = "saved " + path
			m.over.toastAt = time.Now()
		} else {
			m.over.toast = "export failed"
			m.over.toastAt = time.Now()
		}

	case key.Matches(msg, m.keys.Scale):
		m.scaleMode = (m.scaleMode + 1) % 3
		m.over.toast = "scale: " + scaleModeName(m.scaleMode)
		m.over.toastAt = time.Now()

	case key.Matches(msg, m.keys.Grid):
		m.showGrid = !m.showGrid
		m.over.toast = "gridlines: " + onOff(m.showGrid)
		m.over.toastAt = time.Now()

	case key.Matches(msg, m.keys.Faster), msg.String() == "+", msg.String() == "=", msg.String() == "kp+":
		m.adjustRefreshInterval(true)
		return m, waitForSample(m.smp.Out)

	case key.Matches(msg, m.keys.Slower), msg.String() == "-", msg.String() == "_", msg.String() == "kp-":
		m.adjustRefreshInterval(false)
		return m, waitForSample(m.smp.Out)

	case key.Matches(msg, m.keys.Themes):
		m.over.themes = true
		m.over.themeBefore = m.cfg.Theme
		themes := theme.ListThemes()
		m.over.themeIndex = 0
		for i, t := range themes {
			if t.Name == m.cfg.Theme {
				m.over.themeIndex = i
				break
			}
		}
	case key.Matches(msg, m.keys.Repo):
		openURL("https://github.com/programmersd21/flow")

	case key.Matches(msg, m.keys.Issues):
		openURL("https://github.com/programmersd21/flow/issues")

	case key.Matches(msg, m.keys.Discussions):
		openURL("https://github.com/programmersd21/flow/discussions")

	case key.Matches(msg, m.keys.Donate):
		openURL("https://github.com/sponsors/programmersd21")

	default:
	}

	return m, nil
}

func (m *Model) adjustRefreshInterval(faster bool) {
	intervals := []time.Duration{
		25 * time.Millisecond,
		50 * time.Millisecond,
		100 * time.Millisecond,
		150 * time.Millisecond,
		250 * time.Millisecond,
		500 * time.Millisecond,
		750 * time.Millisecond,
		1 * time.Second,
		2 * time.Second,
		3 * time.Second,
		5 * time.Second,
		10 * time.Second,
	}

	idx := -1
	minDiff := time.Duration(1<<63 - 1)
	for i, d := range intervals {
		diff := m.refreshInterval - d
		if diff < 0 {
			diff = -diff
		}
		if diff < minDiff {
			minDiff = diff
			idx = i
		}
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
		col := collector.New(m.iface.name)
		m.smp = sampler.New(col, m.refreshInterval)
		go m.smp.Run(ctx)
	}
}

func (m Model) View() string {
	// Overlays take precedence
	if m.over.themes {
		return renderThemes(m)
	}
	if m.over.help {
		return renderHelp(m)
	}
	if m.over.processes {
		return renderProcesses(m)
	}
	if m.over.iface {
		return renderIfaceDetails(m)
	}

	// Main view
	mode, lines := pickViewModeAndContent(m)
	if mode == ViewTiny {
		return renderTiny(m)
	}

	content := strings.Join(lines, "\n")

	termW := m.width
	if termW <= 0 {
		termW = 80
	}
	termH := m.height
	if termH <= 0 {
		termH = 24
	}

	framed := centerFrame(content, termW, termH)

	// Theme crossfade: dip the whole frame to faint and bring it back over
	// ~180ms so a theme switch reads as a soft dissolve, not a hard cut.
	// Faint (SGR 2) is used rather than a background wash because it works on
	// any terminal background, including transparent ones.
	if m.themeFading && !m.noAnim {
		elapsed := time.Since(m.themeFadeAt).Seconds()
		if t := elapsed / crossfadeSeconds; t >= 0 && t < 1 {
			faint := lipgloss.NewStyle().Faint(t < 1)
			return faint.Render(framed)
		}
	}
	return framed
}

// trailingMedian returns the median of the ring's current samples (0 if empty).
func trailingMedian(r *history.Ring) float64 {
	if r == nil {
		return 0
	}
	s := r.Slice()
	if len(s) == 0 {
		return 0
	}
	cp := make([]float64, len(s))
	copy(cp, s)
	sort.Float64s(cp)
	return cp[len(cp)/2]
}

func (m *Model) updateRollingMax(down, up float64) {
	const decay = 0.995
	m.rates.rollingMaxDown *= decay
	m.rates.rollingMaxUp *= decay
	if down > m.rates.rollingMaxDown {
		m.rates.rollingMaxDown = down
	}
	if up > m.rates.rollingMaxUp {
		m.rates.rollingMaxUp = up
	}
}

func (m Model) FormatBps(bps float64) string {
	return FormatBpsExt(bps, m.unitMode, m.bitsMode)
}

func FormatBps(bps float64, unit UnitMode) string {
	return FormatBpsExt(bps, unit, false)
}

func FormatBpsExt(bps float64, unit UnitMode, bits bool) string {
	if bps < 0 || math.IsNaN(bps) || math.IsInf(bps, 0) {
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

// windowedSamples returns the ring samples that fit the current time window,
// downsampled to at most 2*histDisplayCap points for rendering.
func (m Model) windowedSamples(r *history.Ring) []float64 {
	if r == nil {
		return nil
	}
	s := r.Slice()
	if len(s) == 0 {
		return s
	}
	// How many samples fit in the window at the current refresh rate?
	perSec := float64(time.Second) / float64(m.refreshInterval)
	if perSec < 1 {
		perSec = 1
	}
	want := int(float64(m.windowSecs) * perSec)
	if want > 0 && len(s) > want {
		s = s[len(s)-want:]
	}
	return s
}

// WithWindow sets the initial graph time window from a CLI string like "5m".
func (m Model) WithWindow(w string) Model {
	secs := map[string]int{"1m": 60, "5m": 300, "15m": 900, "1h": 3600, "24h": 86400}
	if s, ok := secs[w]; ok {
		m.windowSecs = s
		for i, tw := range timeWindows {
			if tw == s {
				m.windowIdx = i
				break
			}
		}
	}
	return m
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

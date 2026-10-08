## [0.3.3] - 2026-10-04

Quality and reliability release. No new user-facing features.

### Fixed
- **Auto interface selection followed lifetime byte totals.** With the default
  `interface = "auto"`, flow ranked interfaces by cumulative bytes since boot. On
  a machine with a long-lived high-traffic NIC that later sits idle, flow
  watched the wrong interface and reported its background traffic, so a
  throughput test on another interface appeared to read a few KB/s. Selection is
  now based on current rate, with hysteresis (a challenger must be 3x busier and
  above an 8 KiB/s floor) so the display does not flicker between interfaces.
- **`truncate` panicked on a non-positive width.** Any caller that computed a
  narrow column budget could crash the UI with a slice-bounds error.
- **`renderHelp` could loop forever on a narrow terminal.** Row shrinking
  delegated to `truncate`, which floors at one rune, so a terminal too narrow to
  hold the key column never terminated the loop.
- **Overlays drew wider than the terminal.** The help, processes, interface and
  theme overlays forced a minimum box width, so opening one on a narrow terminal
  wrapped and corrupted the display. All overlays now share a width- and
  height-clamped box helper.
- **Content wider than the terminal was not clipped.** `centerInline` returned
  overflowing lines untouched instead of clipping them.
- **Absurdly large rates produced a ten-character number**, breaking the four
  character budget the hero layout is built around. Units now extend to PB/s and
  the value saturates.
- **`--interface` with an unknown name** now reports the available interfaces
  instead of a bare "not found".
- **`FLOW_DATA`** relocates the persisted daily totals, matching `FLOW_CONFIG`,
  so tests and sandboxed setups no longer touch real user data.

### Changed
- `ui.Model` no longer carries 65 independent fields. Sampling values, spring
  animation, scaling ceilings and pulses are grouped into `rateState`; overlay
  visibility into `overlayState`; interface tracking into `ifaceState`. `Model`
  coordinates them rather than owning each concern directly.
- `sampler.Sampler` depends on a `collector.Reader` interface rather than a
  concrete collector, which is what makes counter behaviour testable.
- `make check` is now a single gate: gofmt check, go vet, golangci-lint, tests,
  the race detector, and a build. `make fuzz` runs the fuzz targets.
- Golden frames tolerate a small number of braille-cell differences. Go's `math`
  package does not guarantee bit-identical results across architectures, and the
  waveform ends in `math.Pow`; a single-ulp difference can flip one dot. Every
  non-braille rune is still compared exactly, so a real regression fails hard.

### Added
- CLI contract tests that build the binary and exercise every flag, the invalid
  input paths, and the exit codes, with stdout and stderr assertions.
- Sampler tests covering rate arithmetic, idle links, counter resets, uint64
  wraparound, read errors, context cancellation, and that a stalled consumer
  cannot block the sampler.
- Collector tests for the rate-based selection policy, hysteresis, vanished
  interfaces, and virtual-interface exclusion.
- Fuzz targets for hero layout, number formatting, byte formatting, and
  truncation.
- Terminal robustness tests across 13 widths by 10 heights for every mode, plus
  hostile state (NaN, infinities, huge and tiny rates, long and Unicode names).
- Overlay tests asserting no overlay exceeds its terminal.
- History tests for the `FLOW_DATA` override.

### Versioning
- `--version` no longer depends on a hardcoded string. It uses the injected
  build value, falling back to the module version recorded in the binary, so
  `go install` (which applies no ldflags) reports the real version instead of a
  stale one.

## [0.3.2] - 2026-10-04

### Added
- **Rewritten home screen**: 5x5 bitmap block digits for download/upload
  (technique adapted from [timr-tui](https://github.com/sectore/timr-tui),
  MIT) with a mirrored braille waveform — download above the centre axis,
  upload below — plus per-half auto-scaling, a dotted centre axis, time
  labels, and a stats row (`today`, `ping`, top talker).
- `internal/render`: braille canvas (2x4 dots per cell), solid area-fill
  renderer with Catmull-Rom smooth edges, per-row gradient band colouring, EMA
  display smoothing, and blocks/ascii fallback glyphs with
  truecolor/256/16/NO_COLOR degradation.
- `internal/format`: `--format` template engine (`rate`, `bytes`, `bits`,
  `pad`, `color`, `spark`) with fixed-width `--width` output for status bars.
- **New keys**: `w` cycle time window (1m/5m/15m/1h/24h), `s` export snapshot
  (`.ansi` + `.txt`), `S` cycle graph scale (auto/linear/sqrt), `G` toggle
  gridlines. Toasts confirm each one without moving the footer.
- **New flags**: `--view`, `--window`, `--no-anim`, `--theme`, `--format`,
  `--width`, `--reset-history`.
- VU-meter peak-hold line, burst ripple, launch sequence, and idle breathing
  pulse — all disabled by `--no-anim`, `FLOW_REDUCE_MOTION=1`, or config.
- Golden-frame tests at 60x20, 80x24, 100x30, and 140x40; footer contract test
  locks the frozen footer bytes.
- Semantic theme tokens (`Fg`, `Muted`, `Subtle`, `Down/Up` + `Lo/Hi`
  gradient endpoints, `Good/Warn/Bad`) with automatic derivation for existing
  themes, so old custom themes keep working.
- Config sections (`[ui]`, `[graph]`, `[latency]`, `[storage]`, `[processes]`,
  `[geoip]`, `[export]`) with validation that clamps bad values and warns
  instead of crashing. Flat keys keep working. `FLOW_CONFIG` overrides the
  path; the `[storage]` name avoids colliding with the existing
  `history = <int>` key.
- Docs: `docs/themes.md`, `docs/json.md`, `docs/integrations.md`,
  `docs/architecture.md`.

### Changed
- Hero numbers are centered as a **single group**: each block gets a fixed
  width, every row is centered inside its width, and a group offset centers
  the pair in the content width (dropping that offset pinned everything to
  the left edge — covered by the centering assertion in
  `TestJoinCenteredStableAcrossValues`). An earlier version mixed an absolute axis position with an
  already-padded row width, subtracting the left block twice and breaking the
  centering it claimed. Covered by `TestJoinCenteredStableAcrossValues`.
- Theme picker swatches use the accent tokens the hero digits actually render
  in, instead of a paler gradient stop. Centering each block inside its own half made two blocks of very
  different widths sit at two different centers, which read as uneven. Because
  the group is centered, a block's origin depends only on the terminal width —
  the numbers no longer drift left or right as the rate changes.

- Theme picker lays names out in one padded column so descriptions form a
  single straight line.
- Help overlay packs groups into two columns by rendered height rather than
  item count, which had left the right column visibly shorter.
- Hero header is centered by the frame, not by itself. Pre-padding the
  header and then centering the frame shifted it right of true center —
  the same double-centering bug the digit blocks had.
- Hero layout reorganized for clarity:
  - The header is a single **centered** identity line joined by dots —
    `flow v0.3.2 · ● wlan0 · 46ms`. It was a left/right spread, which on a
    wide terminal put the title and the interface at opposite edges with an
    empty gulf between them.
  - The arrow and unit moved **off the baseline digit row onto the caption
    row** (`↓ KB/s · peak 4.5 MB/s`). Sharing the baseline made the last digit
    row wider than the four above it, so the block stopped being tabular and a
    fragment of the number read as detached debris — most visible on the
    upload side, where the block is right-aligned. The digits are now five
    rows of identical width.
  - A hairline rule separates the live graph from the readouts, and every
    summary stat (`today`, `ping`, top talker) sits below it.
  - The status row now reports only deviations from defaults and disappears
    when there are none; it previously repeated the interface name that the
    header already shows.
- Home screen no longer uses boxed panels; depth comes from gradients,
  dim/bright tiers, and whitespace.
- Hero digits use the vivid accent token (`Down`/`Up`) instead of a high
  gradient intensity. Gradient crests fade toward near-white pastels by
  design, which read as washed-out beige on 5-row glyphs.
- Captions (`peak`, `session`) are tinted with their number's hue, the stats
  row uses dot separators with latency band coloring, the link dot pulses with
  live traffic, `paused` renders as a pill, and the axis uses the dim tier so
  everything reads against it.
- Theme switches crossfade over ~180ms by dipping the frame to faint and back,
  which works on any terminal background including transparent ones.
- Hero layout adapts: side-by-side numbers on wide terminals, stacked on
  narrow (<60 cols); waveform height scales with terminal height.
- Non-TTY stdout defaults to a single `--tiny`-style line, so pipes and
  scripts never receive ANSI garbage.
- Vertical axis uses a gentle compression curve by default so a 70 KB/s idle
  floor and an 8 MB/s burst are both legible; `S` cycles to linear or sqrt.
- Waveform values are EMA-smoothed for display only; raw samples still feed
  peaks, `today` totals, and JSON.
- Help overlay groups every binding (navigation, display, data, export,
  links), fits small terminals, and swallows keys while open.

### Removed
- Cut the command palette, latency/heatmap/particle views, demo mode, and the
  unused `internal/data` and `internal/motion` packages. `flow` does one
  thing — show live throughput beautifully — and every remaining key serves
  that. The ping readout stays in the hero stats row where it belongs.
- **The peak-hold line.** It drew a rule across the full width at the height
  of the window's transient max, and it set only the left dot column of every
  other cell — producing a full-width row of isolated `⠁` glyphs that read as
  broken ASCII sitting on top of the download waveform. The peak value is
  already reported numerically in the hero caption (`peak 4.5 MB/s`), so the
  line carried no information the screen did not already show.
  `graph.peak_hold` is still accepted so existing configs load unchanged.

### Fixed
- **ANSI theme rendered the whole graph black.** `ansi` carries terminal
  palette *indices* and leaves its RGB stops zeroed; `GradientHex` read the
  zeroed stops, so every waveform cell came out `#000000`. It now returns a
  palette index for ANSI themes (the render engine already emits both forms),
  and `DimColor` no longer mis-parses an index like `"8"` as hex into black
  chrome. Covered by `TestGradientHexRespectsANSIPalette` and
  `TestAllThemesProduceVisibleWaveform`.
- **Latency color now follows the theme status tokens.** The hero stats row
  used ad-hoc hues while the status line used `Good`/`Warn`/`Bad`, so the two
  ping readouts could disagree. All call sites share one `latencyStyle` helper
  (good < 60ms, warn < 150ms, bad above).
- **Theme picker was clipped.** Its row budget ignored the `↑/↓ more`
  indicators, description widths were measured against the terminal rather
  than the box (so they wrapped and ate rows), and the hint line itself was
  wider than narrow boxes. It now fits from 30x10 up, dropping to a borderless
  list on very short terminals. Covered by `TestThemeMenuNeverOverflows`.
- **The `d` filter is now visible.** Hiding a direction made half the graph
  vanish with no explanation near the data; the filter state now sits next to
  the interface in the header.
- **Waveform saturation**: the y-axis ceiling was pinned to `peak * 0.6` to
  "avoid magnifying a lone spike". With steady traffic that ceiling sat below
  the typical value, so the whole graph clamped to solid blocks and lost all
  shape. The ceiling now tracks the visible window peak, which never clips.
  Covered by `TestGraphScaleNeverClipsSteadyTraffic`.
- **Gridlines defaulted on but were invisible.** Braille color is per-cell
  (2x4 dots), so a rule cannot cross a filled cell without tinting that cell's
  other dots. The rule is therefore only visible in empty space — on a busy
  graph that made the feature look broken. It now defaults to OFF, and `G`
  turns it on where it is actually useful.
- Removed a cryptic always-on `grid` chip from the status line; the `G` toast
  already confirms the toggle.
- Hero digits: removed a duplicate badge line that printed the unit twice and
  inflated the block height.
- Waveform colour striping: cells are coloured in horizontal bands by height
  instead of per column, which removed vertical artefacts.
- Upload half idle line now renders on the correct baseline (was drawn at the
  canvas bottom).
- Dense isolated "glow" dots beyond the fill edge are gone, so the waveform
  reads as solid areas.
- Decimal point in the big-digit font renders as a visible dot on the baseline
  instead of a 3-column gap.
- Hero status line keeps the interface name prefix (`iface · refresh · ping`).
- Double normalization bug that flattened the waveform to the baseline when
  the render path normalized already-normalized samples.

### Fixed (merged from UX polish pass)
- **Duplicate ping** — the hero status line no longer repeats the latency value
  already shown in the centered header (`flow v0.3.2 · ● wlan0 · 46ms`).
- **Latency coloring** — ping values now use semantic Good / Warn / Bad hues
  (green < 60 ms, amber 60–150 ms, red ≥ 150 ms) everywhere: hero header,
  hero stats row, compact/mini status line. Previously the upload gradient
  color (yellow/amber) was used unconditionally, which looked confusing on
  themes where upload is green.
- **Interface details overlay** — link state "up" uses `GoodStyle` (green),
  "down" uses `BadStyle` (red), matching the semantic palette instead of the
  theme accent which could be any hue.
- **Tiny mode** — respects the `d` display filter: shows only download, only
  upload, or both, instead of always showing both.
- **Status line** — non-default refresh rate is prefixed with `⟳` so
  `250ms` reads as `⟳ 250ms` and isn't mistaken for a latency value.
- **Paused indicator** — uses a `▮ paused` pill in the upload hue, making
  it stand out as a state rather than a label.

### Added
- **Theme picker color swatches** — each theme row now shows two `●` dots
  (download hue · upload hue) so you can see the palette before switching,
  without needing to commit. ANSI theme falls back to terminal colors 4 and 2.
- `theme.GoodStyle()`, `theme.WarnStyle()`, `theme.BadStyle()` — new semantic
  style helpers backed by the existing `Good/Warn/Bad` tokens on every built-in
  theme; custom themes inherit sensible defaults automatically.
- `theme.ThemeSwatches(name)` — returns the accent download/upload colors for
  any theme by name without switching the active theme, matching what the hero
  digits actually render in.

## [0.3.1] - 2026-09-24

### Fixed
- Removed duplicate download and upload rows in Compact mode; each direction now renders one metric row.

### Changed
- Replaced Compact mode's bordered panels with minimal color-rail metric rows and a single-line hint footer.
- VERSION bumped to 0.3.1.

## [0.3.0] - 2026-09-20

### Added
- Direct browser teleportation shortcuts: open GitHub repository (`g`), Issues (`u`), Discussions (`x`), and Sponsor/Donate (`$`/`v`) instantly in default web browser.
- Two new built-in themes: `rose-pine` (serene palette with pine, foam, iris, and love) and `kanagawa` (earthy Japanese watercolor tones inspired by Hokusai).
- Adaptive scroll window with dynamic indicator in theme picker modal (`t`), rendering cleanly on compact terminal windows.
- Ultra-smooth 20 FPS UI animation engine driven by fine-tuned spring dynamics (`dt = 0.05`).
- Noise floor filtering (< 64 B/s) in Braille sparkline renderer to eliminate background keepalive chatter.

### Changed
- Complete btop-inspired UI/UX redesign featuring embedded header top-border frames (`┌─ download ... peak ─┐`).
- Restored full-width Braille sparkline graphs for maximum visual impact.
- Smoothed default color palette with pastel sapphire, ice cyan, mint, and amber gold gradient stops.
- Single-pass horizontal viewport centering prevents metadata footer drift across terminal widths.
- Refined theme selector cursor indicator to a sleek typographic arrow (`›`).
- README and VERSION updated for 0.3.0.

## [0.2.5] - 2026-08-28

### Added
- New `ansi` theme — uses your terminal's own 16 standard colors, so `flow` matches whatever palette it's running in. Pick it with `t`.
- README updated with the new theme.

### Changed
- The demo video now paces itself naturally: each theme lingers on screen long enough to appreciate it, and mode/filter/interface transitions pause a beat so viewers can register each change.
- VERSION bumped to 0.2.5.

## [0.2.4] - 2026-08-27

### Added
- In-app version display — the running version (e.g. `v0.2.4`) now appears as a small accent-colored pill, centered under the hero `flow` logo and next to the title in compact/mini modes (and via `flow --version`).
- The active-interface dot now glows: it flashes bright on each sample in compact and mini modes.

### Changed
- Much more vibrant, cheerful colors — the default theme's download/upload gradients, borders, and graph colors were brightened and saturated for a lively look.
- All text is now prominent against any background (black or transparent): the dim/muted text ramp was brightened across every built-in theme so secondary labels, flags, and hints always stay clearly readable.
- The hero mode now combines the rainbow ASCII `flow` logo with a centered version pill and the `"Calm your network. See it breathe."` tagline.
- Status and latency indicators now use theme-consistent colors instead of hardcoded green/amber/red, so they always match the active theme. The accent color highlights elevated latency or a down link.
- Removed the green interface-status dot from the footer for a cleaner, less cluttered look. The interface name and state (paused / down only / up only / refresh interval) remain.
- VERSION bumped to 0.2.4.

## [0.2.3] - 2026-08-19

### Fixed
- Spring animation and views hardened against non-finite (NaN/Inf) values across `animate.Spring`, `FormatBpsExt`, `maxf`, and `formatBytes` to prevent propagation of invalid numbers.
- Ping measurements rescheduled in the Bubble Tea update loop to prevent blocking, isolating `TestLoadMissing` for reliability.

### Changed
- VERSION bumped to 0.2.3.

## [0.2.2] - 2026-08-18

### Fixed
- Current download/upload speed always displaying `0 B/s` — the spring animation in `animate.Spring` was numerically unstable at the UI tick interval (130ms), since `1 - damping*dt` evaluated to a n[...]

### Changed
- VERSION bumped to 0.2.2.

## [0.2.1] - 2026-07-23

### Added
- Display filter — Press `d` to cycle through download-only, upload-only, or both panels, allowing full focus on a single direction.
- Footer indicator shows `[down only]` or `[up only]` when a filter is active.

### Fixed
- Panic when sampling interval was set to 300s — `Clamp01` now guards against NaN values, preventing `fiveStopGradient` from computing an out-of-range index with `int(NaN)`.
- Added NaN/Inf guards in `SpeedRatio` and `fiveStopGradient` for defense-in-depth against floating-point edge cases.

### Changed
- VERSION bumped to 0.2.1.
- Removed ROADMAP.md; future plans will be communicated through GitHub Issues and Releases.

## [0.2.0] - 2026-07-15

### Added
- Spring-smoothed value animation — Throughput values now glide smoothly toward their targets using spring physics instead of snapping, matching the architecture documented in README.
- Configurable ping target — Customize the latency check host via `ping_target` in config or `--ping` CLI flag (default: 1.1.1.1).
- Streaming JSON output (`--json-stream`) — Continuous JSON Lines output to stdout for piping into other tools, one sample per line.
- Daily totals persistence — Today's traffic totals are saved to `~/.config/flow/stats.json` on quit and restored on next launch, so totals survive process restarts.
- Custom theme files — Place `.toml` files in `~/.config/flow/themes/` to define user themes with custom colors and gradients, automatically picked up at startup.
- ROADWAY.md — Published project roadmap covering v0.2.x through v0.5.x.
- Expanded test coverage — New test suites for animate package (spring physics, easing) and history persistence (save/load round-trip).

### Changed
- VERSION bumped to 0.2.0.
- Makefile — Added `make check` target for format, vet, lint, and test in one command.

## [0.1.7] - 2026-07-10

### Added
- Sample Heartbeat Dot — The logo dot now flashes in the active theme's accent color upon receiving a network sample and decays smoothly, replacing the continuous decorative breathing animation with[...]

### Improved
- Typography Hierarchy — Established a strict 3-tier typographic hierarchy: Tier 1 (throughput values, bold/brightest), Tier 2 (labels, peaks, secondary stats, medium weight/muted), and Tier 3 (foot[...]
- Color restraint — Overhauled all 8 color themes to use neutral slate/gray borders. Accent colors are reserved strictly for interactive focus and peak/sampling pulses, and colors are reserved stric[...]
- Onboarding & Footer Clarity — Replaced the long list of shortcuts in the footer with a clean, low-clutter, three-item group (`q quit · m mode · ? help`) to simplify the interface on startup.
- Spacing System — Applied a consistent character-grid layout with a standard 1-row vertical gap between all layout sections. Re-derived panel and graph width calculations to align waveforms precise[...]
- Duplicate data removal — Removed redundant download/upload stats from the footer status line, displaying only ping latency if active.

### Changed
- Strictly numbers-only Compact mode — Refined `ViewCompact` (`--compact`) to display only throughput numbers, peaks, and trends (graphs are now omitted as defined).
- Strictly graphs-only Mini mode — Refined `ViewMini` (`--mini`) to show only the waveforms and titles, omitting throughput numbers and peaks to maintain graphs-only utility.
- VERSION bumped to 0.1.7.

### Fixed
- Color config option handling — Fixed a bug where configuring `no_color = true` in the TOML file did not set the `NO_COLOR` environment variable or disable Lip Gloss color formatting on startup.
- Redeclarations and duplicate code — Cleaned up several duplicate functions and unused helpers in theme and animate packages.

## [0.1.6] - 2026-07-10

### Added

- Interface Details Overlay — Press `I` (capital i) to view IP addresses, MAC address, link status, and MTU for the current network interface. Uses both gopsutil and the Go standard library for cros[...]
- Reset Confirmation — Pressing `r` now requires a second press within 2 seconds to confirm, preventing accidental data loss. Press `esc` to cancel.
- Interface Info Keybinding (`I`) — Documented in the help overlay alongside all other bindings.
- Expanded test coverage — New test suites for layout utilities (centerInline, formatBytes, formatInterval, truncate), overlay rendering (help, processes, themes), collector edge cases (loopback det[...]
- ROADMAP.md — Published a public roadmap covering v0.2.x through v0.5.x.

### Improved

- Codebase modularity — views.go (729 lines) split into four focused files: views.go (dashboard layout), panels.go (download/upload panel rendering), overlays.go (help, processes, themes, interface [...]
- Help overlay — Now documents the new `I` interface info keybinding and the reset confirmation (press twice) behavior.
- Collector testability — Added `collector_test.go` with tests for `isLoopback`, `pickBest`, constructor, and `InterfaceDetails`.
- centerInline now handles empty strings gracefully (returns the input unchanged instead of adding whitespace padding).

### Changed

- Major internal restructuring — views.go decomposed into panels.go, overlays.go, and layout.go with zero behavioural changes.
- VERSION bumped to 0.1.6 (significant architecture improvements and new features).

### Fixed

- `runTiny` in main.go had identical branches in the `noColor` if/else block (dead code). Removed the conditional — both branches did the same thing.

## [0.1.5] - 2026-07-08

### Added

- Enriched JSON snapshot output -- `--once --json` now includes `status`, `timestamp` (RFC3339), `download_human`, and `upload_human` fields for self-contained script consumption.
- Refresh interval permanently displayed in footer when non-default (e.g. "every 5s").
- Extended sampling range up to 5 minutes (30s, 60s, 300s) for long-term overnight monitoring.
- Bits indicator -- footer shows `[bits]` label when bits mode is active for clear differentiation from bytes mode.

## [0.1.4] - 2026-07-07

### Fixed

- Dashboard Overflow on Short Terminals - Replaced fixed height thresholds for view mode selection with a measurement-based approach that renders each candidate mode and picks the largest one whose ac[...]

## [0.1.3] - 2026-07-06

### Added

- Theme Selector - Press `t` to open an interactive theme browser with j/k navigation, enter to confirm, and esc to cancel. Includes 8 themes: default, nord, dracula, gruvbox, forest, monochrome, catp[...]
- Bits/sec Display Mode - Press `b` to toggle display between bytes per second and bits per second. Persisted via `bits` config option.
- Command Line Flag - `--bits` flag to start flow in bits/sec display mode directly.
- Interactive Refresh Scaling - Press `+` / `-` keys to speed up or slow down the sampling rate dynamically (50ms–2s range).
- Config Option - `bits = false` TOML option to persist bits/sec preference.
- In-TUI Tiny Mode - `m` key cycles to a centered single-line output inside the TUI, matching the standalone `--tiny` behavior.
- Live Latency (Ping) - A minimal ping indicator measures TCP latency to 1.1.1.1 every 5 seconds and displays it color-coded (green <30ms, amber <100ms, red >=100ms) with a ↔ unicode glyph. First pi[...]

### Changed

- Footer Restructure - Three clean centered rows using `lipgloss.Align(Center)` for mathematically precise centering: interface status (top), minimal stats line with ping + bandwidth (middle, no gap b[...]
- Overlay Dismiss - All overlays (help, processes, theme selector) now use only `esc` to dismiss. `?` opens help, `n` opens processes, `t` opens theme selector — but none of these toggle them closed[...]
- Processes Panel - Redesigned with a rounded indigo border matching the help menu aesthetic, consistent padding, muted separators, and centered layout. Displays "no active network processes detected"[...]
- Today Stats - Deduplicated "today" label (was showing "today" twice), cleaner formatting.
- Makefile - Cross-platform support for Linux, macOS, and Windows (automatic binary extension, platform-agnostic directory creation and cleanup).

## [0.1.2] - 2026-07-05

### Added

- Network Processes panel - press `n` to view active network processes sorted by connection count
- Per-process connection count tracking via gopsutil (cross-platform)
- Graceful fallback on platforms without per-process bandwidth APIs
- Friendly message when no active network processes are detected

### Improved

- In-TUI Tiny Mode (`m` key) now renders centered in the terminal viewport instead of appearing at the top-left corner
- Footer key hints updated to include the new `n` shortcut

## [0.1.1] - 2026-07-05

### Added

- Graphs-only "mini" mode (`--mini`) showing just download/upload panels and waveforms, omitting global title, today's summary, active interface, and key help hints.
- Key binding `m` to interactively cycle through view modes (`hero` -> `compact` -> `mini` -> `tiny` -> `hero`).
- Responsive vertical layout resizing, automatically scaling down to mini mode when the screen height is too small for compact/hero dashboards.
- Premium dev-tool theme styling inspired by Stripe, Spotify, and Apple aesthetics, featuring high-contrast vertical gradients.
- Sleek, modern rounded borders for a clean and unified desktop-TUI look.
- Minimalist, high-end unicode today statistics using colored down/upload arrows (`↓` / `↑`) and clean accent-colored values.
- Refined dot-separated (`·`) status and navigation footer containing real-time active/paused interface dot status (`●`) and highlighted key binds.
- Live peak pulsing white-flash animations when a new session throughput record is reached.
- Clean modal help overlay with modern rounded border styling and highlighted keys.
- `--tiny` mode is now fully independent of Bubble Tea, works reliably in tmux `#(...)`, cron, pipes, and redirected stdout
- Platform-specific config paths: Linux (`~/.config/flow/config.toml`), macOS (`~/Library/Application Support/flow/config.toml`), Windows (`%APPDATA%/flow/config.toml`)

### Fixed

- Daily traffic totals failing to reset when the calendar month/year changes (now compares full date: year, month, and day).
- TUI and one-shot modes hanging indefinitely on network counter read errors (now propagates errors through sampler and exits gracefully with a message).
- Config file not being created on macOS and Windows due to non-standard path resolution
- `--tiny` no longer initializes Bubble Tea, Lip Gloss, termenv, or terminal queries - zero TTY dependency
- `--tiny --no-color` emits clean plain text with no ANSI sequences

## [0.1.0] - 2026-07-04

### Added

- Real-time download (↓) and upload (↑) throughput display
- Block-element sparklines for live graphs
- Velocity glyphs (↗ ↘ →) next to throughput values
- Direction arrows (↓ ↑) on all labels
- Light/dark terminal background detection with adaptive colors
- Three view modes: hero, compact, tiny
- Session peak and daily traffic tracking
- Graceful resize: hero -> compact -> tiny automatically
- Tiny mode (`--tiny`) for tmux status-right
- Auto-scaling units (B/s, KB/s, MB/s, GB/s)
- Speed-based color gradients
- Zero-configuration TOML config with auto-creation
- Non-interactive modes: `--json` and `--once`
- Cross-platform: Linux, macOS, Windows
- GitHub Actions CI and release workflows
- Issue templates and dependabot configuration
- Multi-row high-resolution Braille-grid waveforms for download and upload history
- Sub-pixel horizontal scrolling at 30 FPS for smooth wave movement
- Speed-reactive, glowing rounded borders wrapping download and upload panels
- Typographic peak highlights (bright white flashes) on new records
- Clean breathing TitleRow with dynamic color-shifting bullet dot next to logo title
- Refreshed theme stops with highly vibrant blue/indigo/cyan and emerald/lime gradients

### Changed

- Simplified the hero branding to a plain FLOW title for a calmer, more iconic terminal identity.
- Reworked the terminal UI into a calmer, premium dashboard with a larger title hierarchy and more whitespace.
- Replaced abrupt value easing with spring-driven interpolation for smoother motion in the render loop.
- Added brief pulse and shimmer feedback for peaks and live traffic.
- Refreshed the theme system with a restrained blue, cyan, emerald, and near-white palette that degrades cleanly in low-color terminals.
- Updated history and graph presentation to better emphasize flowing movement over static dashboard chrome.
- Theme system supports light and dark palettes simultaneously
- Sparkline engine rewritten for block-element output
- Default sampling interval decreased from 250ms to 100ms
- Layout tightened - dividers replace blank rows
- Smoothed animations via ease-out interpolation
- Transitioned layouts to stacked, clean dashboard panels with optimized vertical spacing
- Replaced block-element sparklines with the new high-resolution Braille grid
- Standardized layout centering and restored clean minimalist unicode symbols (arrows, dots)

### Fixed

- Reset key fully clears peak tracking, daily totals, rolling maxima, display values, and history ring buffers
- Interface cycling no longer stalls updates
- Interface cycling resets all display state and ring buffers
- Config unit field is case-insensitive
- Various lint and typecheck violations resolved
- Sampler uses a sliding-window average to eliminate zero reads from coarse OS counter granularity

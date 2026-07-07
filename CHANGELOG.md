## [0.1.4] - 2026-07-07

### Fixed

- Dashboard Overflow on Short Terminals - Replaced fixed height thresholds for view mode selection with a measurement-based approach that renders each candidate mode and picks the largest one whose actual line count fits the terminal. Added a safety clamp in `centerFrame` so output never exceeds the terminal height (trims footer instead of clipping the top). This prevents the TUI title and graphs from being scrolled out of view on small windows.

## [0.1.3] - 2026-07-06

### Added

- Theme Selector - Press `t` to open an interactive theme browser with j/k navigation, enter to confirm, and esc to cancel. Includes 8 themes: default, nord, dracula, gruvbox, forest, monochrome, catppuccin, tokyo-night.
- Bits/sec Display Mode - Press `b` to toggle display between bytes per second and bits per second. Persisted via `bits` config option.
- Command Line Flag - `--bits` flag to start flow in bits/sec display mode directly.
- Interactive Refresh Scaling - Press `+` / `-` keys to speed up or slow down the sampling rate dynamically (50ms–2s range).
- Config Option - `bits = false` TOML option to persist bits/sec preference.
- In-TUI Tiny Mode - `m` key cycles to a centered single-line output inside the TUI, matching the standalone `--tiny` behavior.
- Live Latency (Ping) - A minimal ping indicator measures TCP latency to 1.1.1.1 every 5 seconds and displays it color-coded (green <30ms, amber <100ms, red >=100ms) with a ↔ unicode glyph. First ping fires immediately on launch via a separate goroutine, followed by periodic ticks.

### Changed

- Footer Restructure - Three clean centered rows using `lipgloss.Align(Center)` for mathematically precise centering: interface status (top), minimal stats line with ping + bandwidth (middle, no gap between wifi and stats), keybinding hints (bottom). Uniform 2-line gaps between footer sections. Tighter intentional spacing throughout.
- Overlay Dismiss - All overlays (help, processes, theme selector) now use only `esc` to dismiss. `?` opens help, `n` opens processes, `t` opens theme selector — but none of these toggle them closed. Only `esc` returns to the dashboard.
- Processes Panel - Redesigned with a rounded indigo border matching the help menu aesthetic, consistent padding, muted separators, and centered layout. Displays "no active network processes detected" when empty.
- Today Stats - Deduplicated "today" label (was showing "today" twice), cleaner formatting.
- Makefile - Cross-platform support for Linux, macOS, and Windows (automatic binary extension, platform-agnostic directory creation and cleanup).
- Processes Panel - Redesigned with a rounded indigo border matching the help menu aesthetic, consistent padding, muted separators, and centered layout. Displays "no active network processes detected" when empty.
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

- Simplified the hero branding to a plain FLOW title for a calmer,
  more iconic terminal identity.
- Reworked the terminal UI into a calmer, premium dashboard with a
  larger title hierarchy and more whitespace.
- Replaced abrupt value easing with spring-driven interpolation for
  smoother motion in the render loop.
- Added brief pulse and shimmer feedback for peaks and live traffic.
- Refreshed the theme system with a restrained blue, cyan, emerald, and
  near-white palette that degrades cleanly in low-color terminals.
- Updated history and graph presentation to better emphasize flowing
  movement over static dashboard chrome.
- Theme system supports light and dark palettes simultaneously
- Sparkline engine rewritten for block-element output
- Default sampling interval decreased from 250ms to 100ms
- Layout tightened - dividers replace blank rows
- Smoothed animations via ease-out interpolation
- Transitioned layouts to stacked, clean dashboard panels with optimized vertical spacing
- Replaced block-element sparklines with the new high-resolution Braille grid
- Standardized layout centering and restored clean minimalist unicode symbols (arrows, dots)

### Fixed

- Reset key fully clears peak tracking, daily totals, rolling maxima,
  display values, and history ring buffers
- Interface cycling no longer stalls updates
- Interface cycling resets all display state and ring buffers
- Config unit field is case-insensitive
- Various lint and typecheck violations resolved
- Sampler uses a sliding-window average to eliminate zero reads from
  coarse OS counter granularity

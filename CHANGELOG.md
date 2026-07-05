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
- `--tiny` no longer initializes Bubble Tea, Lip Gloss, termenv, or terminal queries — zero TTY dependency
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

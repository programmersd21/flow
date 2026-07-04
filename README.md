# flow

See your network breathe.

<img src="./docs/demo.gif" alt="flow demo" width="100%">

<p align="center">
  <img src="https://img.shields.io/badge/build-passing-00d4aa?style=for-the-badge&labelColor=000000" alt="build">
  <img src="https://img.shields.io/github/go-mod/go-version/programmersd21/flow?style=for-the-badge&labelColor=000000&color=4488ff" alt="go version">
  <img src="https://img.shields.io/github/license/programmersd21/flow?style=for-the-badge&labelColor=000000&color=88cc44" alt="license">
  <img src="https://img.shields.io/github/v/release/programmersd21/flow?style=for-the-badge&labelColor=000000&color=ff6644" alt="release">
  <img src="https://img.shields.io/github/downloads/programmersd21/flow/total?style=for-the-badge&labelColor=000000&color=bb44ff" alt="downloads">
  <img src="https://img.shields.io/github/stars/programmersd21/flow?style=for-the-badge&labelColor=000000&color=ffcc00" alt="stars">
</p>

## Modes

flow adapts to your terminal width with three distinct views.

| hero | compact | tiny |
|:---:|:---:|:---:|
| <img src="./docs/normal_mode.png" alt="hero mode"> | <img src="./docs/compact_mode.png" alt="compact mode"> | <img src="./docs/tiny_mode.png" alt="tiny mode"> |
| full dashboard with sparklines, peaks, and daily totals | numbers-only view for narrow terminals | single-line output for tmux status bars |

## Anatomy

The UI is intentionally minimal:

- Large centered breathing title row (`● FLOW`) that scales to a multi-line ASCII art logo on larger terminals.
- Clean stacked panels separating download and upload statistics, wrapped in custom rounded borders.
- Speed-reactive glowing borders (transitions from dark slate/forest to bright cyan/emerald under load).
- High-resolution U+2800 Braille-grid waveforms scrolling horizontally with sub-pixel precision.
- Velocity glyphs (↗ ↘ →) next to values indicating trend directions.
- Session peaks (flashes white on updates) and daily totals in a clean info bar.
- Quiet footer with interface and keybindings.

The layout is centered in both X and Y and designed to feel premium before it feels technical.

## Install

```sh
go install github.com/programmersd21/flow/cmd/flow@latest
```

Or build from source:

```sh
git clone https://github.com/programmersd21/flow
cd flow
make install
```

Pre-built binaries for Linux, macOS, Windows (amd64/arm64) are on the
[releases page](https://github.com/programmersd21/flow/releases).

## Philosophy

> Does this help someone understand their network in under one second?
> If no — cut it.

flow stays deliberately small. No CPU panels, no packet counters, no
multi-pane layouts. Download and upload throughput, in real time,
nothing else.

The interface is built to feel calm and expensive: large typography,
soft gradients, spring-driven motion, and restrained decoration.

## Features

- Real-time download (↓) and upload (↑) throughput
- Spring-driven interpolation for display values
- Pulse and shimmer microinteractions tied to live traffic
- High-resolution Braille-grid waveforms with 30 FPS horizontal smooth scrolling
- Speed-reactive, glowing rounded borders
- Typographic peak highlights (bold white flash on breaching records)
- Velocity glyphs (↗ ↘ →) show traffic direction trend
- Auto-scaling units: B/s, KB/s, MB/s, GB/s (cycle with `c`)
- Saturated color gradients that brighten with activity
- Session peak tracking and daily traffic totals
- Three view modes: hero, compact, tiny
- Graceful auto-resize between modes
- Zero configuration — optional TOML config at
  `~/.config/flow/config.toml`
- Non-interactive modes: `--json` and `--once` for scripts
- Cross-platform: Linux, macOS, Windows

## Keybindings

![keybinds](docs/keybinds.png)

| Key         | Action                      |
|-------------|-----------------------------|
| `q` / `^C`  | quit                        |
| `r`         | reset session peaks         |
| `i`         | cycle interfaces            |
| `c`         | cycle units (auto/KB/MB/GB) |
| `p`         | pause / resume              |
| `?`         | toggle help                 |

## Usage

![help](docs/cli_help.png)

```sh
flow                        # hero view, auto interface
flow --tiny                 # single-line tmux mode
flow --compact              # numbers only
flow --json                 # one-shot JSON, then exit
flow --once                 # one-shot plain text, then exit
flow --interface wlan0      # pin interface
flow --refresh 500ms        # slower sampling (default 100ms)
flow --no-color
flow --version
flow --help
```

### --json output

```json
{
  "download_bps": 124300000,
  "upload_bps": 18400000,
  "peak_down_bps": 320000000,
  "peak_up_bps": 48000000,
  "interface": "wlan0",
  "unit_display": "MB/s"
}
```

### tmux

```sh
# ~/.tmux.conf
set -g status-right "#(flow --tiny --no-color)"
set -g status-interval 1
```

## Configuration

`~/.config/flow/config.toml` is auto-created on first run. It respects
`XDG_CONFIG_HOME`.

```toml
refresh   = "100ms"   # sampling interval
history   = 60        # seconds of sparkline history retained
theme     = "default"
unit      = "auto"    # auto | kb | mb | gb (case-insensitive)
interface = "auto"    # auto or name (e.g. eth0, wlan0)
no_color  = false
```

## Architecture

Two decoupled loops:

- **Sampling loop** (~10 Hz): reads OS network counters, computes a
  sliding-window average, emits a sample on a channel.
- **Render loop** (~30 fps): springs display values toward the latest
  sample, decays brief pulses, and renders the dashboard.

The model keeps rendering separate from collection so the UI can stay
smooth without adding work to the sampler.

Idle CPU stays well under 1%.

## Platform notes

- Linux: reads `/proc/net/dev` via gopsutil.
- macOS: sysctl / getifaddrs.
- Windows: `GetIfTable2`. No elevated privileges.

## Star History

<a href="https://www.star-history.com/?repos=programmersd21%2Fflow&type=date&legend=top-left">
 <picture>
   <source media="(prefers-color-scheme: dark)" srcset="https://api.star-history.com/chart?repos=programmersd21/flow&type=date&theme=dark&legend=top-left" />
   <source media="(prefers-color-scheme: light)" srcset="https://api.star-history.com/chart?repos=programmersd21/flow&type=date&legend=top-left" />
   <img alt="Star History Chart" src="https://api.star-history.com/chart?repos=programmersd21/flow&type=date&legend=top-left" />
 </picture>
</a>

## Development

```sh
make check       # fmt-check + vet + lint + test — run before every PR
make build       # build ./bin/flow
make test        # go test ./... -race -cover
make release-dry # goreleaser snapshot (no publish)
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for the full guide.

## License

MIT — see [LICENSE](LICENSE).

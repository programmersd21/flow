<div align="center">

### flow

*A terminal dashboard for real-time network throughput.*

</div>

<img src="./docs/demo.gif" alt="flow demo" width="100%">

<p align="center">
  <img src="https://img.shields.io/badge/build-passing-00d4aa?style=for-the-badge&labelColor=000000" alt="build">
  <img src="https://img.shields.io/github/go-mod/go-version/programmersd21/flow?style=for-the-badge&labelColor=000000&color=4488ff" alt="go version">
  <img src="https://img.shields.io/github/license/programmersd21/flow?style=for-the-badge&labelColor=000000&color=88cc44" alt="license">
  <img src="https://img.shields.io/github/v/release/programmersd21/flow?style=for-the-badge&labelColor=000000&color=ff6644" alt="release">
  <img src="https://img.shields.io/github/downloads/programmersd21/flow/total?style=for-the-badge&labelColor=000000&color=bb44ff" alt="downloads">
  <img src="https://img.shields.io/github/stars/programmersd21/flow?style=for-the-badge&labelColor=000000&color=ffcc00" alt="stars">
</p>

## Contents

- [Install](#install)
- [Rationale](#rationale)
- [Philosophy](#philosophy)
- [Modes](#modes)
- [Features](#features)
- [Usage](#usage)
- [Configuration](#configuration)
- [Architecture](#architecture)
- [Development](#development)
- [Star History](#star-history)
- [License](#license)

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

Pre-built binaries for Linux, macOS, and Windows (amd64 and arm64) are available on the [releases page](https://github.com/programmersd21/flow/releases).

## Rationale

Most network monitors display CPU usage, per-process breakdowns, packet counts, and connection tables. flow displays throughput only.

Every feature decision is evaluated against a single question: does this help the user understand their network within one second. If not, it is not included.

The result is a small, deliberately scoped tool. There are no additional panels, no required configuration, and no unnecessary complexity in either the interface or the underlying implementation.

## Philosophy

```mermaid
flowchart LR
    Input["Network throughput"] --> Q{"Understood within<br/>one second?"}
    Q -->|Yes| Keep["Retain feature"]
    Q -->|No| Cut["Remove feature"]
    Keep --> Result["Calm, minimal interface"]
    Cut --> Result
```

Every feature is evaluated against one question: does this help a user understand their network within one second. If not, it is removed.

flow does not include CPU panels, packet counters, or multi-pane layouts. It reports download and upload throughput, in real time, and nothing else.

The interface is built for restraint rather than density: large typography, controlled color, and spring-based motion in place of decoration.

## Modes

flow adjusts its display according to terminal width.

| hero | compact | tiny |
|:---:|:---:|:---:|
| <img src="./docs/normal_mode.png" alt="hero mode"> | <img src="./docs/compact_mode.png" alt="compact mode"> | <img src="./docs/tiny_mode.png" alt="tiny mode"> |
| Full dashboard with sparklines, peaks, and daily totals | Numeric values only, for narrow terminals | Single-line output, intended for status bars |

## Features

- Real-time download and upload throughput
- Interpolated display values using spring-based animation
- Braille-grid waveform rendering at 30 frames per second
- Border color reflects current transfer speed
- Visual indication when a new session peak is recorded
- Directional indicators for traffic trend
- Automatic unit scaling from B/s to GB/s
- Session peak tracking and daily traffic totals
- Three display modes with automatic switching on resize
- No required configuration; optional TOML configuration file
- Non-interactive output modes for use in scripts
- Supported on Linux, macOS, and Windows

## Usage

![help](docs/cli_help.png)

```sh
flow                        # hero view, auto interface
flow --tiny                 # single-line mode for status bars
flow --compact               # numeric values only
flow --json                  # single JSON output, then exit
flow --once                  # single plain-text output, then exit
flow --interface wlan0       # specify network interface
flow --refresh 500ms         # adjust sampling interval (default 100ms)
flow --no-color
flow --version
flow --help
```

### Keybindings

![keybinds](docs/keybinds.png)

| Key         | Action                      |
|-------------|-----------------------------|
| `q` / `^C`  | Quit                         |
| `r`         | Reset session peaks          |
| `i`         | Cycle network interfaces     |
| `c`         | Cycle display units          |
| `p`         | Pause or resume              |
| `?`         | Toggle help                  |

### JSON output

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

### tmux integration

```sh
# ~/.tmux.conf
set -g status-right "#(flow --tiny --no-color)"
set -g status-interval 1
```

## Configuration

A configuration file is created automatically at `~/.config/flow/config.toml` on first run. The `XDG_CONFIG_HOME` environment variable is respected if set.

```toml
refresh   = "100ms"   # sampling interval
history   = 60        # seconds of retained sparkline history
theme     = "default"
unit      = "auto"    # auto, kb, mb, or gb
interface = "auto"    # auto, or a specific interface name (e.g. eth0, wlan0)
no_color  = false
```

<details>
<summary>Interface layout</summary>

```mermaid
flowchart TD
    Title["flow"]
    Download["DOWNLOAD panel: current speed, peak, waveform"]
    Upload["UPLOAD panel: current speed, peak, waveform"]
    Info["Daily totals and active interface"]
    Footer["Keybinding reference"]

    Title --> Download
    Download --> Upload
    Upload --> Info
    Info --> Footer
```

All elements are centered on both axes. Panel border color changes according to current transfer speed.

</details>

## Architecture

flow runs two independent loops connected by a channel.

```mermaid
flowchart LR
    subgraph Sampling["Sampling loop, approximately 10 Hz"]
        OS["Network counters via gopsutil"] --> SlidingWindow["Sliding-window average"]
        SlidingWindow --> Channel["Sample channel"]
    end

    subgraph Rendering["Render loop, approximately 30 fps"]
        Channel --> Spring["Spring interpolation"]
        Spring --> Display["Dashboard render (Bubble Tea)"]
        Theme["Theme configuration"] -.-> Display
    end
```

- The sampling loop reads network counters from the operating system, computes a sliding-window average, and emits a sample on a channel.
- The render loop interpolates display values toward the latest sample and renders the dashboard.

Separating collection from rendering keeps the interface responsive without adding load to the sampler. Idle CPU usage remains below one percent.

**Platform notes:** Linux reads `/proc/net/dev` via gopsutil. macOS uses sysctl and getifaddrs. Windows uses `GetIfTable2`. No elevated privileges are required on any platform.

## Development

```sh
make check       # format check, vet, lint, and test
make build       # build ./bin/flow
make test        # go test ./... -race -cover
make release-dry # goreleaser snapshot build, no publish
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for contribution guidelines.

## Star History

<a href="https://www.star-history.com/?repos=programmersd21%2Fflow&type=date&legend=top-left">
 <picture>
   <source media="(prefers-color-scheme: dark)" srcset="https://api.star-history.com/chart?repos=programmersd21/flow&type=date&theme=dark&legend=top-left" />
   <source media="(prefers-color-scheme: light)" srcset="https://api.star-history.com/chart?repos=programmersd21/flow&type=date&legend=top-left" />
   <img alt="Star History Chart" src="https://api.star-history.com/chart?repos=programmersd21/flow&type=date&legend=top-left" />
 </picture>
</a>

## License

MIT. See [LICENSE](LICENSE).

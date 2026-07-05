<div align="center">

### flow

*A terminal dashboard for real-time network throughput.*

<img src="./docs/demo.gif" alt="flow demo" width="100%">

<h3 align="center">
  Your Internet Stats, TUIfied.
</h3>

<p align="center">
  <i>Fast • Beautiful • Cross-Platform • Open Source</i>
</p>

</div>

<p align="center">
  <a href="https://github.com/programmersd21/flow/actions">
    <img src="https://img.shields.io/github/actions/workflow/status/programmersd21/flow/release.yml?style=for-the-badge&logo=githubactions&logoColor=white&label=Build&labelColor=111111&color=00c853" alt="Build">
  </a>
  <a href="https://github.com/programmersd21/flow/releases">
    <img src="https://img.shields.io/github/v/release/programmersd21/flow?style=for-the-badge&logo=github&logoColor=white&label=Release&labelColor=111111&color=ff6d00" alt="Release">
  </a>
  <a href="https://github.com/programmersd21/flow/releases">
    <img src="https://img.shields.io/github/downloads/programmersd21/flow/total?style=for-the-badge&logo=github&logoColor=white&label=Downloads&labelColor=111111&color=8e24aa" alt="Downloads">
  </a>
  <a href="https://github.com/programmersd21/flow/stargazers">
    <img src="https://img.shields.io/github/stars/programmersd21/flow?style=for-the-badge&logo=github&logoColor=white&label=Stars&labelColor=111111&color=fbc02d" alt="Stars">
  </a>
</p>

<p align="center">
  <a href="https://github.com/programmersd21/flow">
    <img src="https://img.shields.io/github/go-mod/go-version/programmersd21/flow?style=for-the-badge&logo=go&logoColor=white&label=Go&labelColor=111111&color=2196f3" alt="Go Version">
  </a>
  <a href="https://github.com/programmersd21/flow/blob/main/LICENSE">
    <img src="https://img.shields.io/github/license/programmersd21/flow?style=for-the-badge&label=License&labelColor=111111&color=43a047" alt="License">
  </a>
  <a href="https://github.com/programmersd21/homebrew-flow">
    <img src="https://img.shields.io/badge/Homebrew-brew%20install%20programmersd21%2Fflow%2Fflow?style=for-the-badge&logo=homebrew&logoColor=white&labelColor=111111&color=fbb040" alt="Homebrew">
  </a>
  <a href="https://aur.archlinux.org/packages/flow-network-monitor-bin">
    <img src="https://img.shields.io/aur/version/flow-network-monitor-bin?style=for-the-badge&logo=archlinux&logoColor=white&label=AUR&labelColor=111111&color=1793d1" alt="AUR Version">
  </a>
  <a href="https://aur.archlinux.org/packages/flow-network-monitor-bin">
    <img src="https://img.shields.io/aur/popularity/flow-network-monitor-bin?style=for-the-badge&logo=archlinux&logoColor=white&label=Popularity&labelColor=111111&color=1976d2" alt="AUR Popularity">
  </a>
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

1. AUR:

```sh
yay -S flow-network-monitor-bin
```

**💝 Thanks to [@Dominiquini](https://github.com/Dominiquini) for assisting in AUR deployment!**

2. Homebrew:

```sh
brew install programmersd21/flow/flow
```

3. Go:

```sh
go install github.com/programmersd21/flow/cmd/flow@latest
```

4. From source:

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

flow adjusts its display according to terminal width and height.

| hero | compact | mini | tiny |
|:---:|:---:|:---:|:---:|
| <img src="./docs/normal_mode.png" alt="hero mode"> | <img src="./docs/compact_mode.png" alt="compact mode"> | | <img src="./docs/tiny_mode.png" alt="tiny mode"> |
| Full dashboard with logo branding, waveforms, peaks, and daily totals | Cleaner layout with title row, waveforms, peaks, and daily totals | Graphs-only layout, hiding header logo, today's summary, and help footers | Single-line output, intended for status bars |

## Features

- Real-time download and upload throughput
- Interpolated display values using spring-based animation
- Braille-grid waveform rendering at 30 frames per second
- Border color reflects current transfer speed with sleek, modern rounded outlines
- Live peak pulsing white-flash animations when a new session peak is reached
- Minimalist, high-end unicode today statistics and navigation footer
- Directional indicators for traffic trend
- Automatic unit scaling from B/s to GB/s
- Session peak tracking and daily traffic totals
- Four display modes with automatic responsive switching on both width and height resize
- No required configuration; optional TOML configuration file
- Non-interactive output modes for use in scripts
- Supported on Linux, macOS, and Windows

## Usage

![help](docs/cli_help.png)

```sh
flow                        # hero view, auto interface
flow --tiny                 # single-line mode for status bars
flow --mini                 # graphs-only mode, no headers/footers
flow --compact              # compact layout with title row and waveforms
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
| `m`         | Cycle display/view modes    |
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

A configuration file is created automatically on first run:

| Platform | Path |
|----------|------|
| Linux | `~/.config/flow/config.toml` |
| macOS | `~/Library/Application Support/flow/config.toml` |
| Windows | `%APPDATA%\flow\config.toml` |

The `XDG_CONFIG_HOME` environment variable is respected on Linux if set.

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

## 💙 Community & Support

Thank you to everyone who has supported **flow**. Whether you've starred the repository, reported an issue, submitted a pull request, shared the project, or simply use it every day—your support is what keeps the project growing.

### ⭐ Stargazers

<p align="center">
  <a href="https://github.com/programmersd21/flow/stargazers">
    <img src="https://readme-contribs.as93.net/stargazers/programmersd21/flow" alt="Stargazers of flow">
  </a>
</p>

### ❤️ Sponsor

If **flow** has made your terminal a little better, please consider sponsoring its development on GitHub.

Your sponsorship helps support:

- 🚀 New features
- 🐛 Bug fixes
- ⚡ Performance improvements
- 📖 Better documentation
- 🛠️ Long-term maintenance

If you're unable to sponsor, you can still make a huge difference by:

- ⭐ Starring the repository
- 📢 Sharing **flow** with others
- 💬 Recommending it to friends and colleagues
- 🐞 Reporting bugs or suggesting improvements
- 🤝 Contributing code or documentation

Every contribution, no matter how small, helps **flow** continue to grow.

**Thank you for supporting open source. ❤️**

## License

MIT. See [LICENSE](LICENSE).

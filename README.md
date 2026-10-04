<div align="center">

# flow

**Real-time network throughput in your terminal.**

<img src="./assets/demo.png" alt="flow demo" width="100%">

[![build](https://img.shields.io/github/actions/workflow/status/programmersd21/flow/release.yml?style=for-the-badge&label=build&labelColor=282828&color=b8bb26&logo=githubactions&logoColor=fbf1c7&logoSize=auto)](https://github.com/programmersd21/flow/actions)
[![release](https://img.shields.io/github/v/release/programmersd21/flow?style=for-the-badge&label=release&labelColor=282828&color=fabd2f&logo=git&logoColor=fbf1c7&logoSize=auto)](https://github.com/programmersd21/flow/releases)
[![stars](https://img.shields.io/github/stars/programmersd21/flow?style=for-the-badge&label=stars&labelColor=282828&color=d79921&logo=starship&logoColor=fbf1c7&logoSize=auto)](https://github.com/programmersd21/flow)
[![license](https://img.shields.io/github/license/programmersd21/flow?style=for-the-badge&label=license&labelColor=282828&color=83a598&logo=opensourceinitiative&logoColor=fbf1c7&logoSize=auto)](LICENSE)

</div>

## install

**arch linux**

```sh
yay -S flow-network-monitor-bin
```

**homebrew**

```sh
brew install programmersd21/flow/flow
```

**go**

```sh
go install github.com/programmersd21/flow/cmd/flow@latest
```

or download a binary from [releases](https://github.com/programmersd21/flow/releases).

## usage

```sh
flow                              # hero view (default)
flow --tiny                       # single line, for status bars
flow --mini                       # graphs only
flow --compact                    # condensed numbers
flow --json                       # one JSON snapshot, then exit
flow --json-stream                # newline-delimited JSON, continuous
flow --theme nord --window 5m     # override theme and time window
flow --no-anim                    # disable animations (reduced motion)
flow --format '↓ {{.Down}} ↑ {{.Up}}'   # templated text output
```

When stdout is not a TTY (a pipe, a script, a status bar) `flow` prints a
single plain line instead of starting the TUI, so it is safe to pipe.

### tmux

```sh
set -g status-right "#(flow --tiny --no-color)"
```

### waybar / polybar / sketchybar / i3blocks

All of them can consume `flow --json-stream` or a fixed-width `--format` line.
See [docs/integrations.md](docs/integrations.md) for copy-paste snippets.

## features

* **Mirrored Braille Waveform**: Download above the axis, upload below, rendered on a 2x4-dot braille canvas with a smooth interpolated crest
* **Big Block Digits**: 5x5 bitmap hero numbers (timr-tui technique) — single-stroke, tabular, never jitter
* **Calm Motion**: Spring-driven value easing, an idle breathing pulse, a burst ripple, a pulsing link dot, a 180ms theme crossfade, and a skippable launch sequence — all disabled by `--no-anim`
* **Time Windows**: `w` cycles 1m / 5m / 15m / 1h / 24h
* **Scale Modes**: `S` cycles auto / linear / sqrt; `G` toggles the midpoint gridline (off by default — see below)
* **Snapshot Export**: `s` writes the current frame as `.ansi` and `.txt`
* **11 Built-in Themes**: `default`, `nord`, `dracula`, `gruvbox`, `forest`, `monochrome`, `catppuccin`, `tokyo-night`, `rose-pine`, `kanagawa`, `ansi`
* **Custom TOML Themes**: Load custom color schemes from your config directory (see [docs/themes.md](docs/themes.md))
* **Text Streams First**: `--json`, `--json-stream`, and `--format` (see [docs/json.md](docs/json.md))
* **Zero Overhead**: No telemetry, no network calls except the ping target, no cgo in the core path
* **Cross-Platform**: Linux, macOS, and Windows support

## keys

| key | action |
| :-: | ------ |
| `q` | quit |
| `m` | cycle view mode |
| `d` | cycle display filter (both / down / up) |
| `t` | theme selector |
| `n` | network processes inspector |
| `i` | cycle network interface |
| `I` | interface details |
| `c` | cycle unit scale (auto / KB/s / MB/s / GB/s) |
| `b` | toggle bits/sec vs bytes/sec |
| `+` / `-` | adjust sampling interval |
| `p` | pause / resume sampling |
| `r` | reset peak counters (press twice) |
| `g` | open GitHub repository in browser |
| `u` | open GitHub issues in browser |
| `x` | open GitHub discussions in browser |
| `$` / `v` | sponsor / donate on GitHub |
| `?` | help menu |
| `w` | cycle time window (1m / 5m / 15m / 1h / 24h) |
| `s` | export snapshot (`.ansi` + `.txt`) |
| `S` | cycle graph scale (auto / linear / sqrt) |
| `G` | toggle gridlines |

## configuration

Created automatically on first run:

```text
Linux    ~/.config/flow/config.toml
macOS    ~/Library/Application Support/flow/config.toml
Windows  %APPDATA%\flow\config.toml
```

```toml
refresh     = "100ms"
theme       = "default"
unit        = "auto"
interface   = "auto"
bits        = false
ping_target = "1.1.1.1"
no_anim     = false   # disable animations
```

Environment variables: `NO_COLOR`, `FLOW_REDUCE_MOTION=1` (reduced motion),
`FLOW_CONFIG` (override the config file path), `FLOW_GLYPHS` (force
`braille` / `blocks` / `ascii` waveform glyphs).

## development

```sh
make check
make test
make build
```

## license

MIT

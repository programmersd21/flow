# text output

Every number shown on screen is also available as text. This is deliberate:
`flow` is a Unix tool first.

## --json

One JSON object, then exit.

```sh
flow --json
```

```json
{
  "status": "ok",
  "timestamp": "2026-10-04T12:00:00Z",
  "interface": "wlan0",
  "download_bps": 2410000,
  "upload_bps": 195000,
  "download_human": "2.3 MB/s",
  "upload_human": "190 KB/s",
  "peak_down_bps": 2410000,
  "peak_up_bps": 2410000,
  "unit_display": "MB/s"
}
```

Field names are stable across releases.

## --json-stream

Newline-delimited JSON, one object per sample. Stops on SIGINT.

```sh
flow --json-stream | jq -r '"\(.download_human) down"'
```

## --format

Templated output for status bars and one-liners.

```sh
flow --format '↓ {{.Down}} ↑ {{.Up}}'
flow --format '{{.Iface}} {{.DownBps}} {{.UpBps}}' --width 40
flow --tiny --format '{{.Down}} / {{.Up}}'
```

### fields

| field | meaning |
| ----- | ------- |
| `.Iface` | interface name |
| `.DownBps`, `.UpBps` | raw bytes/sec (numbers) |
| `.Down`, `.Up` | human-readable strings |
| `.PeakDownBps`, `.PeakUpBps` | peaks for this session |
| `.TodayDown`, `.TodayUp` | totals since local midnight (human strings) |
| `.Time` | RFC3339 timestamp |

### functions

| function | meaning |
| -------- | ------- |
| `rate` | format a bytes/sec rate for humans |
| `bytes` | format a byte count for humans |
| `bits` | format a bits/sec rate for humans |
| `pad N` | right-pad a string to N columns |
| `color name` | apply a color (respects `NO_COLOR`) |
| `spark` | tiny sparkline from a sample slice |

A bad template prints a message to stderr and exits with code **2**.

## --width

Pads or truncates output to a fixed column count so status bars do not jitter.

```sh
flow --tiny --format '{{.Down}} / {{.Up}}' --width 24
```

## pipes and exit codes

If stdout is not a TTY and no output flag was given, `flow` prints a single
plain line and exits rather than starting the TUI. You will never get ANSI
escape codes in a pipe by accident.

| code | meaning |
| ---- | ------- |
| `0` | success |
| `1` | runtime error |
| `2` | usage, template, or config error |
| `130` | interrupted (SIGINT) in non-TUI modes |

## `--no-color`

With `--tiny` the output is plain arrow text, suitable for tmux, polybar,
waybar, sketchybar, and i3blocks.

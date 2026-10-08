# JSON output

`flow` exposes two machine-readable modes. Both are stable: field names, types,
and units do not change within a major version.

## `--once --json`

Emits a single JSON object on stdout, then exits.

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

### Contract

| field | type | units | notes |
| ----- | ---- | ----- | ----- |
| `status` | string | — | `"ok"` on success |
| `timestamp` | string | RFC 3339, UTC | when the sample was taken |
| `interface` | string | — | interface the rates were read from; empty only if unavailable |
| `download_bps` | number | **bytes per second** | never negative |
| `upload_bps` | number | **bytes per second** | never negative |
| `download_human` | string | — | preformatted for display, with `--bits` applied |
| `upload_human` | string | — | preformatted for display |
| `peak_down_bps` | number | bytes per second | session peak |
| `peak_up_bps` | number | bytes per second | session peak |
| `unit_display` | string | — | unit of `download_human`/`upload_human` |

**Rates are always bytes per second** unless `--bits` is passed, in which case
`download_bps`/`upload_bps` remain bytes per second and only the `*_human`
strings switch to bit units (`Kb/s`, `Mb/s`, …). Consumers that want one
unambiguous unit should read the numeric fields.

All fields are always present. Values are never negative and never `null`.

### Errors

A failure writes a message to **stderr** and exits non-zero. stdout stays empty,
so a consumer never has to distinguish "no output" from "valid empty document".

## `--json-stream`

Emits one JSON object per line (JSON Lines), continuously, until interrupted.

```jsonl
{"status":"ok","timestamp":"...","interface":"wlan0","download_bps":2410000, ...}
{"status":"ok","timestamp":"...","interface":"wlan0","download_bps":2388000, ...}
```

Each line is a complete JSON object with the same schema as `--once --json`.
Lines are written as samples arrive, so `--json-stream` is meant to be piped;
it does not exit on its own. Use `--once --json` for a snapshot.

## Usage

```sh
# current rate, as a number
flow --once --json | jq -r '.download_bps'

# bits instead of bytes in the human strings
flow --once --json --bits | jq -r '.download_human'

# follow the stream
flow --json-stream | jq -r '"\(.timestamp) \(.download_bps)"'
```

## Compatibility

Fields are added, never removed or repurposed. A consumer can ignore fields it
does not know. `--tiny`, `--format` and the TUI are unaffected by these modes.
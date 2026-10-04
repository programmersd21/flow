# status bar integrations

`flow` is designed to live in a status bar. Three approaches work:

1. `--tiny` — one pre-coloured line, cheapest option.
2. `--format` — your own layout, optionally fixed width with `--width`.
3. `--json-stream` — full data for tools that do their own rendering.

In all three, adding `--no-color` makes the output plain ASCII, which is what
panels that re-render the string expect.

## tmux

```sh
set -g status-right "#(flow --tiny --no-color)"
```

Fixed width, so the status bar never reflows:

```sh
set -g status-right '#(flow --format "{{.Down}} / {{.Up}}" --width 28 --no-color)'
```

## polybar

```ini
modules-right += flow
flow = flow --format '{{.Down}} ↓  {{.Up}} ↑' --no-color
```

One shot per poll. (`--json-stream` never exits, so it is for pipes, not
poll-based modules.)

## waybar

```json
{
  "modules-right": {
    "custom/flow": {
      "exec": "flow --format '{{.Down}} ↓ {{.Up}} ↑' --no-color",
      "interval": 1
    }
  }
}
```

## sketchybar

```bash
sketchybar --add item flow right \
  --set flow script="flow --format '↓ {{.Down}}  ↑ {{.Up}}' --no-color" \
               update_freq=2
```

## i3blocks

```ini
[flow]
command=flow --format 'net ↓{{.Down}} ↑{{.Up}}' --no-color
interval=2
```

## Notes

- `--refresh` controls how often values update; the default is `100ms`.
- `--interface <name>` pins a specific interface, which avoids surprises on
  machines with many virtual adapters (docker, vboxnet, tailscale).

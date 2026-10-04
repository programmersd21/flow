# architecture

## package layout

```text
cmd/flow/            flag parsing and process wiring
internal/
  animate/           spring dynamics and easing for value animation
  collector/         per-OS interface counters
  sampler/           counter deltas -> rates
  history/           ring buffers, today totals, persistence
  ping/              latency measurement with timeouts
  processes/         per-process socket enumeration
  theme/             semantic tokens, 11 built-ins, custom TOML themes
  render/            braille canvas, area fill, gradients, glyph fallbacks
  sparkline/         velocity glyphs and compact braille graphs
  format/            --format templates
  ui/                Bubble Tea models and all rendering (incl. big digits)
```

`ui` composes `render` and `theme`. `render` knows nothing about Bubble Tea or
lipgloss, which is what makes it unit-testable and benchmarkable in isolation.

## data flow

```text
collector ──► sampler ──► Sample{DownBps, UpBps} ──► ui.Model
                                                        │
                          ┌─────────────────────────────┼──────────────┐
                          ▼                             ▼              ▼
                   history.Ring (raw)              rollingMax      tracker
                          │                       (scale)      (peaks/today)
                          ▼                             │              │
                   render.Smooth ───────────────────►   ▼              ▼
                   (display only)               hero waveform     stats row
```

Rates come from monotonic counter deltas over a sliding window, so a single
dropped read does not spike the graph. Display smoothing is applied at render
time only; peaks, `today` totals, and JSON output stay honest.

## rendering

The hero view is drawn onto a **braille canvas**: each terminal cell holds 2x4
dots (`U+2800` plus a dot mask). That gives 2x the horizontal and 4x the
vertical resolution of the character grid, which is why the waveform has a
smooth interpolated crest instead of a staircase.

`DrawAreaFill` walks each of the 2*W dot columns, interpolates the sample at
that position with a Catmull-Rom spline (clamped so it never overshoots below
the baseline), and fills from the baseline outward. `ColorRows` then assigns a
color per **row band** rather than per cell: per-cell coloring produced
vertical striping artifacts, while horizontal bands read as a smooth gradient.

The mirrored view renders two half-canvases sharing a centre axis — download
fills upward from the axis, upload fills downward.

### no chrome over the data

Gridlines and the peak-hold rule were both removed after they rendered as
visual noise on a filled waveform. Braille color is per-cell (2x4 dots), so a
rule can only be drawn through *empty* cells — meaning a reference line behind
the data is either invisible or a row of stray dots. Scale references are
better served by the numeric caption than by overlay geometry, so the hero
view keeps the waveform area clean.

### hero digits

Hero numbers are a 5x5 bitmap font (one full block per "on" pixel, technique
adapted from timr-tui). Two-pixel-thick strokes plus solid fills is what makes
them legible at a glance; the earlier 3-row zigzag font blurred into itself
at large sizes. Glyphs
are tabular (5 columns + 1 column of spacing) so a changing value never shifts
the layout, and the decimal point is a narrow 2-wide block on the baseline.

They are colored from the theme's accent token rather than a high gradient
intensity: gradient crests fade toward near-white pastels by design, which
reads as washed-out beige on large glyphs.

Two rules keep the block visually stable:

- **The five digit rows must be identical width.** The arrow and unit
  therefore live on the caption row below (`↓ KB/s · peak 4.5 MB/s`), not on
  the baseline. Putting them on the baseline made the last row wider, so the
  block stopped being tabular and part of the number read as detached debris.
- **Trailing zeros are stripped.** `4` B/s must render as `4`, not `4.00`;
  otherwise a one-digit value silently costs four glyphs.
- **The pair is centered as a group.** Each hero block has a fixed width
  (its widest row), every row is centered inside that width with
  `lipgloss.Width/Align(Center)`, and the two blocks plus a fixed gap are
  centered together. A block's width therefore depends only on the value's
  magnitude class, so nothing moves while the value stays in its unit.

### vertical scaling

A pure linear axis is wrong for real traffic: a 70 KB/s idle floor next to an
8 MB/s burst is a 120x range, and linear crushes everything below the peak
into an invisible sliver. The default `auto` mode applies a gentle compression
curve (`v^0.65`). `S` cycles to linear and sqrt for people who want them. Each
half scales independently by default so a small upload next to a large
download stays visible.

## testing

```sh
make check      # gofmt, go vet, golangci-lint, go test
make test       # go test ./...
make build      # build ./bin/flow

go test -race ./...
go test ./internal/ui -run Golden -update-golden
go test -bench=. ./internal/render
```

Invariants of the render path are asserted in
`internal/ui/render_audit_test.go` (fill is a faithful 1:1 dot mapping,
gridlines never overwrite data, mirrored halves are symmetric) because each
past regression in this pipeline was visual rather than a crash.

Contracts enforced by tests and should not be broken:

- **`TestFooterGoldenContract`** locks the exact footer bytes of the home screen.
- **`TestGoldenHeroFrames`** snapshots the hero view at 60x20, 80x24, 100x30,
  and 140x40 from deterministic synthetic data.

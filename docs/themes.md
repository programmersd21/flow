# themes

`flow` ships with 11 built-in themes and loads custom ones from your config
directory.

## built-in themes

`default`, `nord`, `dracula`, `gruvbox`, `forest`, `monochrome`, `catppuccin`,
`tokyo-night`, `rose-pine`, `kanagawa`, `ansi`.

`ansi` steps through terminal palette indexes instead of RGB, so `flow` blends
into whatever colorscheme your terminal defines. It is the safest choice on
terminals without truecolor.

## semantic tokens

Every view reads color only through semantic tokens. No hex literals in view
code, which is what makes theme switching and contrast tuning tractable.

```text
Fg      bright tier   live values, hero numbers
Muted   mid tier      labels, units, captions
Subtle  dim tier      chrome, axes, gridlines, footer

Down    download accent          Up    upload accent
DownHi  download gradient crest  UpHi  upload gradient crest
DownLo  download gradient base   UpLo  upload gradient base

Good / Warn / Bad    link state, latency readouts, errors
```

Rules that keep a theme coherent:

- **Three text tiers only.** Live value = `Fg`, label = `Muted`, chrome = `Subtle`.
- **A hue never means two things.** Download has one hue, upload has one hue.
- Gradients interpolate between `*Lo` and `*Hi`, precomputed into a 32-step
  lookup table at theme load.
- Contrast targets: `Fg` >= 4.5:1, `Muted` >= 3:1 against the expected
  background. `Subtle` is intentionally lower but must stay legible.

## custom themes

Drop a TOML file into the `themes/` subdirectory of your config dir:

```text
Linux    ~/.config/flow/themes/
macOS    ~/Library/Application Support/flow/themes/
Windows  %APPDATA%\flow\themes\
```

Each file defines the text tiers plus two 5-stop gradients (crest first).
`flow` picks the file up on the next theme-picker open or restart.

```toml
name = "midnight"

text_dim     = "#3f4a63"   # Subtle: chrome, axes, footer
text_muted   = "#7c88a8"   # Muted: labels, units
text_soft    = "#aab4cc"
text_base    = "#c8d0e4"
text_bright  = "#e6ecf8"   # Fg: live values
text_pure    = "#ffffff"
border       = "#232a3d"
accent       = "#ffcc66"

# 5 gradient stops each, crest first. Fewer than 5 is fine;
# missing stops stay black, so provide at least 2-3.
download = ["#7fd4ff", "#4aa8f0", "#2b6fd6", "#1d4ed8", "#1e3a8a"]
upload   = ["#ffd98a", "#f0b45a", "#d6892b", "#b45309", "#7c2d12"]
```

The `Down`/`Up` accent tokens and their `Lo`/`Hi` gradient endpoints are
derived from these stops automatically, so a custom file keeps working as new
tokens are added.

## degradation

| capability | behaviour |
| ---------- | --------- |
| truecolor | full 32-step gradients |
| 256 color | gradients quantized to the nearest xterm color (8 steps) |
| 16 color | gradients quantized to the terminal palette; bold for emphasis |
| `NO_COLOR` / `--no-color` | monochrome, glyph density carries the shape |

`NO_COLOR` in the environment and `--no-color` on the command line both disable
color entirely. `COLORTERM=truecolor` and a `TERM` containing `256color` select
the gradient quality.

## switching

Press `t` to open the theme picker (the preview recolors the UI as you move,
and `esc` reverts). `--theme <name>` overrides the config for one run.

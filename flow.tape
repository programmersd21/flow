Output docs/demo.gif

Require make
Require curl

Set Shell "bash"
Set FontFamily "JetBrains Mono"
Set FontSize 16
Set Theme {
  "name": "Flow",
  "black": "#45475a",
  "red": "#f38ba8",
  "green": "#a6e3a1",
  "yellow": "#f9e2af",
  "blue": "#89b4fa",
  "magenta": "#f5c2e7",
  "cyan": "#94e2d5",
  "white": "#bac2de",
  "brightBlack": "#585b70",
  "brightRed": "#f38ba8",
  "brightGreen": "#a6e3a1",
  "brightYellow": "#f9e2af",
  "brightBlue": "#89b4fa",
  "brightMagenta": "#f5c2e7",
  "brightCyan": "#94e2d5",
  "brightWhite": "#cdd6f4",
  "background": "#000000",
  "foreground": "#cdd6f4",
  "selection": "#585b70",
  "cursor": "#f5e0dc"
}
Set Width 1400
Set Height 920
Set Padding 16
Set MarginFill "#000000"
Set Margin 8
Set WindowBar "Colorful"
Set PlaybackSpeed 1.0
Set Framerate 12

Hide

# Build
Type "make build"
Enter
Sleep 2s
Ctrl+L
Sleep 200ms

# Background traffic
Type "curl -L --limit-rate 5M --max-time 60 -o /dev/null https://speed.hetzner.de/100MB.bin &>/dev/null &"
Enter
Sleep 300ms
Type "curl -L --limit-rate 3M --max-time 60 -o /dev/null https://proof.ovh.net/files/100Mb.dat &>/dev/null &"
Enter
Sleep 300ms
Ctrl+L
Sleep 200ms

# Launch flow
Type@50ms "./bin/flow --refresh 100ms"
Sleep 300ms
Enter
Sleep 3s

# Reveal — hero mode
Show
Sleep 4s

# --- All 8 themes ---

# Default → nord
Type "t"
Sleep 800ms
Type "j"
Sleep 600ms
Enter
Sleep 2.5s

# Nord → dracula
Type "t"
Sleep 600ms
Type "j"
Sleep 500ms
Enter
Sleep 2.5s

# Dracula → gruvbox
Type "t"
Sleep 600ms
Type "j"
Sleep 400ms
Enter
Sleep 2.5s

# Gruvbox → forest
Type "t"
Sleep 600ms
Type "j"
Sleep 400ms
Enter
Sleep 2.5s

# Forest → monochrome
Type "t"
Sleep 600ms
Type "j"
Sleep 400ms
Enter
Sleep 2.5s

# Monochrome → catppuccin
Type "t"
Sleep 600ms
Type "j"
Sleep 400ms
Enter
Sleep 2.5s

# Catppuccin → tokyo-night
Type "t"
Sleep 600ms
Type "j"
Sleep 400ms
Enter
Sleep 2.5s

Sleep 2s

# Reset
Type "r"
Sleep 2s

# Interface cycle
Type "i"
Sleep 1.5s
Type "i"
Sleep 1.5s

# View modes
Type "m"
Sleep 1.5s
Type "m"
Sleep 1.5s

# Bits + units
Type "b"
Sleep 1.5s
Type "c"
Sleep 1.2s
Type "c"
Sleep 1.2s

# Pause / resume
Type "p"
Sleep 3s
Type "p"
Sleep 1.5s

# Processes
Type "n"
Sleep 3s
Escape
Sleep 1.5s

# Help
Type "?"
Sleep 3s
Escape
Sleep 1.5s

Sleep 3s
Hide
Type "q"
Sleep 1s

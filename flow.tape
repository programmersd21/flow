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
Set Width 1200
Set Height 680
Set Padding 16
Set MarginFill "#000000"
Set Margin 8
Set WindowBar "Colorful"
Set PlaybackSpeed 1.0

Hide

# Build
Type "make build"
Enter
Sleep 3s

# Clear for a pristine start
Ctrl+L
Sleep 300ms

# Steady medium-contention traffic — long-running throttled downloads
Type "curl -L --limit-rate 4M --max-time 45 -o /dev/null https://speed.hetzner.de/100MB.bin &>/dev/null &"
Enter
Sleep 400ms
Type "curl -L --limit-rate 3M --max-time 40 -o /dev/null https://proof.ovh.net/files/100Mb.dat &>/dev/null &"
Enter
Sleep 400ms
Type "curl -L --limit-rate 5M --max-time 42 -o /dev/null https://speed.hetzner.de/100MB.bin &>/dev/null &"
Enter
Sleep 400ms
Type "curl -L --limit-rate 2M --max-time 35 -o /dev/null https://proof.ovh.net/files/100Mb.dat &>/dev/null &"
Enter
Sleep 400ms
Type "curl -L --limit-rate 3M --max-time 48 -o /dev/null https://speed.hetzner.de/100MB.bin &>/dev/null &"
Enter
Sleep 400ms
Type "curl -X POST --limit-rate 3M --max-time 38 -o /dev/null https://httpbin.org/post &>/dev/null &"
Enter
Sleep 400ms

# Small delayed bumps for visual texture on the steady baseline
Type "(sleep 6; curl -L --limit-rate 4M --max-time 25 -o /dev/null https://speed.hetzner.de/100MB.bin &>/dev/null) &"
Enter
Sleep 400ms
Type "(sleep 12; curl -L --limit-rate 3M --max-time 20 -o /dev/null https://proof.ovh.net/files/100Mb.dat &>/dev/null) &"
Enter
Sleep 400ms

# Clean slate before launch
Ctrl+L
Sleep 200ms

# Type the launch command character by character — builds anticipation
Type@50ms "./bin/flow --refresh 100ms"
Sleep 400ms
Enter
Sleep 3s

# Reveal — hero logo, breathing dot, subtitle, all 12 downloads raging
Show
Sleep 6s

# Reset peaks — clean slate mid-storm
Type "r"
Sleep 3s

# Cycle through all unit modes at a deliberate pace
Type "c"
Sleep 2s
Type "c"
Sleep 2s
Type "c"
Sleep 2s
Type "c"
Sleep 2s

# Freeze the storm
Type "p"
Sleep 2s
Type "p"
Sleep 2s

# Mid-demo burst fires here — graphs spike again

# Show the help overlay
Type "?"
Sleep 4s
Type "?"
Sleep 2s

# Let the fresh burst animate through
Sleep 4s

# Hide before quitting so the GIF ends on the clean TUI screen
Hide
Type "q"
Sleep 2s

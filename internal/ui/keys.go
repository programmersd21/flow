package ui

import "github.com/charmbracelet/bubbles/key"

type KeyMap struct {
	Quit      key.Binding
	Reset     key.Binding
	Interface key.Binding
	Unit      key.Binding
	Pause     key.Binding
	Help      key.Binding
	Mode      key.Binding
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Reset: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "reset peaks"),
		),
		Interface: key.NewBinding(
			key.WithKeys("i"),
			key.WithHelp("i", "cycle interface"),
		),
		Unit: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "cycle units"),
		),
		Pause: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "pause/resume"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "toggle help"),
		),
		Mode: key.NewBinding(
			key.WithKeys("m"),
			key.WithHelp("m", "cycle view mode"),
		),
	}
}

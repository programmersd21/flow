package ui

import "github.com/charmbracelet/bubbles/key"

type KeyMap struct {
	Quit          key.Binding
	Esc           key.Binding
	Reset         key.Binding
	Interface     key.Binding
	InterfaceInfo key.Binding
	Unit          key.Binding
	Pause         key.Binding
	Help          key.Binding
	Mode          key.Binding
	Processes     key.Binding
	Bits          key.Binding
	Faster        key.Binding
	Slower        key.Binding
	Themes        key.Binding
	History       key.Binding
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Esc: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
		Reset: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "reset peaks"),
		),
		Interface: key.NewBinding(
			key.WithKeys("i"),
			key.WithHelp("i", "cycle interface"),
		),
		InterfaceInfo: key.NewBinding(
			key.WithKeys("I"),
			key.WithHelp("I", "interface info"),
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
			key.WithHelp("?", "open help"),
		),
		Mode: key.NewBinding(
			key.WithKeys("m"),
			key.WithHelp("m", "cycle view mode"),
		),
		Processes: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "network processes"),
		),
		Bits: key.NewBinding(
			key.WithKeys("b"),
			key.WithHelp("b", "toggle bits/bytes"),
		),
		Faster: key.NewBinding(
			key.WithKeys("+", "="),
			key.WithHelp("+", "faster refresh"),
		),
		Slower: key.NewBinding(
			key.WithKeys("-"),
			key.WithHelp("-", "slower refresh"),
		),
		Themes: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "choose theme"),
		),
	}
}

package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Up     key.Binding
	Down   key.Binding
	Tab    key.Binding
	Top    key.Binding
	Bottom key.Binding
	Append   key.Binding
	Edit     key.Binding
	Copy     key.Binding
	Projects key.Binding
	Help   key.Binding
	Quit   key.Binding
}

func defaultKeyMap() keyMap {
	return keyMap{
		Up: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("↓/j", "down"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "switch focus"),
		),
		Top: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("g", "top"),
		),
		Bottom: key.NewBinding(
			key.WithKeys("G"),
			key.WithHelp("G", "bottom"),
		),
		Append: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "append"),
		),
		Edit: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "edit in $EDITOR"),
		),
		Copy: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "copy log"),
		),
		Projects: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "switch project"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "esc", "ctrl+c"),
			key.WithHelp("q/esc", "quit"),
		),
	}
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Append, k.Edit, k.Copy, k.Projects, k.Help, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Tab},
		{k.Append, k.Edit, k.Copy},
		{k.Projects, k.Top, k.Bottom},
		{k.Help, k.Quit},
	}
}

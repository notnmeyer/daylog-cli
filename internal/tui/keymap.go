package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Older    key.Binding
	Newer    key.Binding
	JumpDay  key.Binding
	Search   key.Binding
	Scroll   key.Binding
	Top      key.Binding
	Bottom   key.Binding
	Append   key.Binding
	Edit     key.Binding
	Copy     key.Binding
	Projects key.Binding
	Todos    key.Binding
	Help     key.Binding
	Quit     key.Binding
}

func defaultKeyMap() keyMap {
	return keyMap{
		Older: key.NewBinding(
			key.WithKeys("h", "left"),
			key.WithHelp("←/h", "older day"),
		),
		Newer: key.NewBinding(
			key.WithKeys("l", "right"),
			key.WithHelp("→/l", "newer day"),
		),
		JumpDay: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d/esc", "day list"),
		),
		// "/" opens the ledger filter (which searches dates AND log content);
		// it no longer exists in the reading view
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "filter"),
		),
		Scroll: key.NewBinding(
			key.WithKeys("j", "k", "up", "down"),
			key.WithHelp("↑↓/jk", "scroll"),
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
		Todos: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "todos"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		// esc is "cancel modal" everywhere; binding it to quit too means a
		// second esc after closing a modal would drop the whole session
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}

// ShortHelp leads with help and quit so the two most important discovery
// hints survive when the footer truncates on a narrow terminal
// help is shown in the reading view, which no longer has search/filter (that
// lives on the ledger), so k.Search is omitted here
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit, k.Older, k.Newer, k.JumpDay, k.Append, k.Edit, k.Copy, k.Todos}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Older, k.Newer, k.JumpDay},
		{k.Append, k.Edit, k.Copy},
		{k.Todos, k.Projects},
		{k.Scroll, k.Top, k.Bottom},
		{k.Help, k.Quit},
	}
}

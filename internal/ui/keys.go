package ui

import "github.com/charmbracelet/bubbles/key"

type KeyMap struct {
	Up           key.Binding
	Down         key.Binding
	Enter        key.Binding
	Search       key.Binding
	Escape       key.Binding
	Tab          key.Binding
	Copy         key.Binding
	Quit         key.Binding
	PageUp       key.Binding
	PageDown     key.Binding
	Edit         key.Binding
	EditExternal key.Binding
	Send         key.Binding
}

var Keys = KeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	Search: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "search"),
	),
	Escape: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back"),
	),
	Tab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "switch pane"),
	),
	Copy: key.NewBinding(
		key.WithKeys("y"),
		key.WithHelp("y", "copy schema"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	PageUp: key.NewBinding(
		key.WithKeys("pgup", "ctrl+u"),
		key.WithHelp("pgup", "page up"),
	),
	PageDown: key.NewBinding(
		key.WithKeys("pgdown", "ctrl+d"),
		key.WithHelp("pgdn", "page down"),
	),
	Edit: key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "edit"),
	),
	EditExternal: key.NewBinding(
		key.WithKeys("E"),
		key.WithHelp("E", "$EDITOR"),
	),
	Send: key.NewBinding(
		key.WithKeys("ctrl+enter", "ctrl+s"),
		key.WithHelp("ctrl+enter", "send"),
	),
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Enter, k.Edit, k.Send, k.Copy, k.Quit}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Enter},
		{k.Search, k.Escape, k.Tab},
		{k.Edit, k.EditExternal, k.Send},
		{k.Copy, k.Quit},
	}
}

package ui

import "github.com/charmbracelet/bubbles/key"

// keyMap holds every binding used across screens. Bindings are context-sensitive:
// View() and Update() only act on the ones relevant to the current screen, but the
// help component can render them.
type keyMap struct {
	Up      key.Binding
	Down    key.Binding
	Enter   key.Binding
	Back    key.Binding
	Quit    key.Binding
	Help    key.Binding
	Refresh key.Binding
	Filter  key.Binding

	ScrollLeft  key.Binding
	ScrollRight key.Binding

	// Clipboard
	Yank     key.Binding
	YankMenu key.Binding

	// Menu / actions
	EC2        key.Binding
	ASG        key.Binding
	ELB        key.Binding
	Regions    key.Binding
	Profiles   key.Binding
	Logout     key.Binding
	Shell      key.Binding
	PortFwd    key.Binding
	StopFwd    key.Binding
	StopAllFwd key.Binding
	RestartFwd key.Binding
	ClearFwd   key.Binding
	Tunnels    key.Binding
	ToggleHost key.Binding

	MetricWindow key.Binding
}

func defaultKeyMap() keyMap {
	return keyMap{
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
			key.WithHelp("enter", "confirm"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		Filter: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "filter"),
		),
		ScrollLeft: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "scroll left"),
		),
		ScrollRight: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "scroll right"),
		),
		Yank: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "copy"),
		),
		YankMenu: key.NewBinding(
			key.WithKeys("Y"),
			key.WithHelp("Y", "copy (menu)"),
		),
		EC2: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "EC2"),
		),
		ASG: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "Auto Scaling Groups"),
		),
		ELB: key.NewBinding(
			key.WithKeys("b"),
			key.WithHelp("b", "Load Balancers"),
		),
		Regions: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("g", "region"),
		),
		Profiles: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "profiles"),
		),
		Logout: key.NewBinding(
			key.WithKeys("L"),
			key.WithHelp("L", "logout"),
		),
		Shell: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "shell SSM"),
		),
		PortFwd: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "port-forward"),
		),
		StopFwd: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "stop"),
		),
		StopAllFwd: key.NewBinding(
			key.WithKeys("X"),
			key.WithHelp("X", "stop all"),
		),
		RestartFwd: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "restart"),
		),
		ClearFwd: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "clear stopped"),
		),
		Tunnels: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "tunnels"),
		),
		ToggleHost: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "remote host"),
		),
		MetricWindow: key.NewBinding(
			key.WithKeys("m"),
			key.WithHelp("m", "metrics window"),
		),
	}
}

// ShortHelp implements help.KeyMap (a generic fallback line).
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Back, k.Quit}
}

// FullHelp implements help.KeyMap.
func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Enter, k.Back},
		{k.Profiles, k.Regions, k.Logout},
		{k.EC2, k.Tunnels},
		{k.ASG, k.ELB},
		{k.Shell, k.PortFwd, k.Filter, k.Refresh},
		{k.Yank, k.YankMenu},
		{k.StopFwd, k.StopAllFwd, k.RestartFwd, k.ClearFwd},
		{k.Help, k.Quit},
	}
}

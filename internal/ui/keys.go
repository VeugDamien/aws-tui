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
	Tunnels    key.Binding
	ToggleHost key.Binding

	MetricWindow key.Binding
}

func defaultKeyMap() keyMap {
	return keyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "haut"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "bas"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "valider"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "retour"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quitter"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "aide"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "rafraîchir"),
		),
		Filter: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "filtrer"),
		),
		ScrollLeft: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "défiler gauche"),
		),
		ScrollRight: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "défiler droite"),
		),
		Yank: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "copier"),
		),
		YankMenu: key.NewBinding(
			key.WithKeys("Y"),
			key.WithHelp("Y", "copier (menu)"),
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
			key.WithHelp("g", "région"),
		),
		Profiles: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "profils"),
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
			key.WithHelp("x", "arrêter"),
		),
		StopAllFwd: key.NewBinding(
			key.WithKeys("X"),
			key.WithHelp("X", "tout arrêter"),
		),
		Tunnels: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "tunnels"),
		),
		ToggleHost: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "hôte distant"),
		),
		MetricWindow: key.NewBinding(
			key.WithKeys("m"),
			key.WithHelp("m", "fenêtre métriques"),
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
		{k.StopFwd, k.StopAllFwd},
		{k.Help, k.Quit},
	}
}

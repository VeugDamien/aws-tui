package ui

import "github.com/charmbracelet/lipgloss"

var (
	colorAccent  = lipgloss.Color("63")  // purple
	colorMuted   = lipgloss.Color("241") // gray
	colorError   = lipgloss.Color("203") // red
	colorOK      = lipgloss.Color("78")  // green
	colorWarn    = lipgloss.Color("214") // orange
	colorSubtle  = lipgloss.Color("250")
	colorInverse = lipgloss.Color("230")
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorInverse).
			Background(colorAccent).
			Padding(0, 1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	statusBarStyle = lipgloss.NewStyle().
			Foreground(colorSubtle)

	statusRuleStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	statusLineStyle = lipgloss.NewStyle().
			Foreground(colorInverse).
			Background(colorAccent).
			Bold(true).
			Padding(0, 1)

	identityStyle = lipgloss.NewStyle().
			Foreground(colorOK)

	errorStyle = lipgloss.NewStyle().
			Foreground(colorError).
			Bold(true)

	errorBoxStyle = lipgloss.NewStyle().
			Foreground(colorError).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorError).
			Padding(0, 1)

	warnStyle = lipgloss.NewStyle().
			Foreground(colorWarn)

	helpStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	menuItemStyle = lipgloss.NewStyle().
			Padding(0, 2)

	menuKeyStyle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	menuSelStyle = lipgloss.NewStyle().
			Foreground(colorInverse).
			Background(lipgloss.Color("57")).
			Bold(true).
			Padding(0, 1)

	configBarStyle = lipgloss.NewStyle().
			Foreground(colorSubtle).
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(colorMuted).
			Padding(0, 1).
			MarginTop(1)

	configKeyStyle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	configValStyle = lipgloss.NewStyle().
			Foreground(colorInverse).
			Bold(true)

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorAccent).
			Padding(0, 1)

	labelStyle = lipgloss.NewStyle().
			Foreground(colorSubtle).
			Width(14)
)

package ui

import "github.com/charmbracelet/lipgloss"

// Palette: a vivid cyan/blue theme that adapts to light and dark terminals via
// lipgloss.AdaptiveColor (the Light variant is used on light backgrounds, the Dark
// variant on dark ones — detected automatically).
var (
	colorAccent  = lipgloss.AdaptiveColor{Light: "31", Dark: "44"}   // cyan / teal (primary)
	colorAccent2 = lipgloss.AdaptiveColor{Light: "25", Dark: "75"}   // blue (secondary)
	colorMuted   = lipgloss.AdaptiveColor{Light: "244", Dark: "244"} // mid gray
	colorError   = lipgloss.AdaptiveColor{Light: "160", Dark: "203"} // red / coral
	colorOK      = lipgloss.AdaptiveColor{Light: "28", Dark: "42"}   // green
	colorWarn    = lipgloss.AdaptiveColor{Light: "130", Dark: "214"} // amber
	colorSubtle  = lipgloss.AdaptiveColor{Light: "60", Dark: "245"}  // dim text
	colorFg      = lipgloss.AdaptiveColor{Light: "236", Dark: "252"} // primary text
)

var (
	// titleStyle: the app name, bold accent (no filled bar).
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	// sectionTitleStyle: bold accent heading used for screen/section titles.
	sectionTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorAccent)

	statusBarStyle = lipgloss.NewStyle().
			Foreground(colorSubtle)

	statusRuleStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	// statusLineStyle: the transient status message, accent text with a leading
	// marker (no filled bar).
	statusLineStyle = lipgloss.NewStyle().
			Foreground(colorAccent2).
			Bold(true)

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

	updateHintStyle = lipgloss.NewStyle().
			Foreground(colorOK).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	menuItemStyle = lipgloss.NewStyle()

	menuKeyStyle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	// menuSelStyle: the highlighted menu entry. The selection is marked with an
	// accent bar in the view; here we just brighten and bold the text.
	menuSelStyle = lipgloss.NewStyle().
			Foreground(colorFg).
			Bold(true)

	// selBarStyle renders the accent bar that marks a selected row/entry.
	selBarStyle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	configBarStyle = lipgloss.NewStyle().
			Foreground(colorSubtle).
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(colorMuted).
			Padding(0, 1).
			MarginTop(1)

	configKeyStyle = lipgloss.NewStyle().
			Foreground(colorAccent2).
			Bold(true)

	configValStyle = lipgloss.NewStyle().
			Foreground(colorFg).
			Bold(true)

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorAccent).
			Padding(0, 1)

	labelStyle = lipgloss.NewStyle().
			Foreground(colorSubtle).
			Width(14)
)

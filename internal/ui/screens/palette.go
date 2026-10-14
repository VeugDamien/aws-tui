package screens

import "github.com/charmbracelet/lipgloss"

// Shared adaptive palette for the screen sub-package, mirroring the root ui theme
// (vivid cyan/blue, adapting to light/dark terminals). Keeping it here avoids a
// dependency on the parent ui package while staying visually consistent.
var (
	paletteAccent  = lipgloss.AdaptiveColor{Light: "31", Dark: "44"}   // cyan / teal
	paletteAccent2 = lipgloss.AdaptiveColor{Light: "25", Dark: "75"}   // blue
	paletteMuted   = lipgloss.AdaptiveColor{Light: "244", Dark: "244"} // mid gray
	paletteSubtle  = lipgloss.AdaptiveColor{Light: "60", Dark: "245"}  // dim text
	paletteFg      = lipgloss.AdaptiveColor{Light: "236", Dark: "252"} // primary text
	paletteOK      = lipgloss.AdaptiveColor{Light: "28", Dark: "42"}   // green
	paletteWarn    = lipgloss.AdaptiveColor{Light: "130", Dark: "214"} // amber
	paletteError   = lipgloss.AdaptiveColor{Light: "160", Dark: "203"} // red / coral
	paletteBorder  = lipgloss.AdaptiveColor{Light: "250", Dark: "240"} // subtle border
)

package screens

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/VeugDamien/aws-tui/internal/awsconfig"
)

// Styles for the grouped profile list. Kept here (screens package) so the widget
// is self-contained and reusable independently of the root ui styles.
var (
	plColorAccent = lipgloss.Color("63")  // purple
	plColorMuted  = lipgloss.Color("241") // gray
	plColorSubtle = lipgloss.Color("250")
	plColorText   = lipgloss.Color("252")
	plColorSSO    = lipgloss.Color("78")  // green
	plColorDep    = lipgloss.Color("214") // orange
	plColorStand  = lipgloss.Color("111") // blue
	plColorSelBg  = lipgloss.Color("57")  // selection background
	plColorSelFg  = lipgloss.Color("230") // selection foreground

	plTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("230")).
			Background(plColorAccent).
			Padding(0, 1)

	plHeaderSSO = lipgloss.NewStyle().
			Bold(true).
			Foreground(plColorSSO)

	plHeaderDep = lipgloss.NewStyle().
			Bold(true).
			Foreground(plColorDep)

	plHeaderStand = lipgloss.NewStyle().
			Bold(true).
			Foreground(plColorStand)

	plHeaderMeta = lipgloss.NewStyle().
			Foreground(plColorMuted)

	plTreeStyle = lipgloss.NewStyle().
			Foreground(plColorMuted)

	plNameStyle = lipgloss.NewStyle().
			Foreground(plColorText)

	plNameSelStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(plColorSelFg).
			Background(plColorSelBg)

	plKindStyle = lipgloss.NewStyle().
			Foreground(plColorSubtle)

	plRegionStyle = lipgloss.NewStyle().
			Foreground(plColorMuted).
			Italic(true)

	plFilterStyle = lipgloss.NewStyle().
			Foreground(plColorDep)

	plEmptyStyle = lipgloss.NewStyle().
			Foreground(plColorMuted).
			Italic(true)
)

// rowKind distinguishes the two kinds of rendered rows.
type rowKind int

const (
	rowHeader rowKind = iota
	rowProfile
)

// row is a single rendered line: either a group header (not selectable) or a
// profile entry (selectable).
type row struct {
	kind    rowKind
	group   awsconfig.Group // set for headers
	profile awsconfig.Profile
	depth   int
	last    bool // last child at its depth (for └─ vs ├─)
}

// ProfileList is a grouped, keyboard-navigable profile picker. Group headers are
// rendered but skipped during navigation; only profile rows are selectable. It
// supports a live name filter and vertical scrolling.
type ProfileList struct {
	groups []awsconfig.Group

	rows     []row // all rows (headers + profiles), pre-flattened
	selable  []int // indices into rows that are selectable (profile rows)
	selected int   // index into selable

	filter    string
	filtering bool

	width, height int
	offset        int // first visible row (scroll)
}

// NewProfileList builds a grouped profile list from the flat profile slice.
func NewProfileList(profiles []awsconfig.Profile, width, height int) ProfileList {
	p := ProfileList{
		groups: awsconfig.GroupProfiles(profiles),
		width:  width,
		height: height,
	}
	p.rebuild()
	return p
}

// SetSize updates the widget dimensions.
func (p *ProfileList) SetSize(width, height int) {
	p.width = width
	p.height = height
	p.ensureVisible()
}

// Filtering reports whether the widget is currently capturing filter text.
func (p ProfileList) Filtering() bool { return p.filtering }

// Filter returns the current filter text.
func (p ProfileList) Filter() string { return p.filter }

// SelectedProfile returns the currently highlighted profile, if any.
func (p ProfileList) SelectedProfile() (awsconfig.Profile, bool) {
	if p.selected < 0 || p.selected >= len(p.selable) {
		return awsconfig.Profile{}, false
	}
	return p.rows[p.selable[p.selected]].profile, true
}

// rebuild recomputes the flattened rows from the groups and current filter, then
// clamps the selection.
func (p *ProfileList) rebuild() {
	p.rows = p.rows[:0]
	p.selable = p.selable[:0]

	needle := strings.ToLower(strings.TrimSpace(p.filter))

	for _, g := range p.groups {
		// Filter members first so empty groups are hidden entirely.
		var members []awsconfig.GroupedProfile
		for _, m := range g.Profiles {
			if needle == "" || strings.Contains(strings.ToLower(m.Profile.Name), needle) {
				members = append(members, m)
			}
		}
		if len(members) == 0 {
			continue
		}

		p.rows = append(p.rows, row{kind: rowHeader, group: g})

		for i, m := range members {
			// "last" is true when no later member has the same-or-deeper chain at
			// this depth; for our simple depths, last-at-depth is enough for a
			// clean tree. We compute it per contiguous depth run.
			last := isLastAtDepth(members, i)
			p.rows = append(p.rows, row{
				kind:    rowProfile,
				profile: m.Profile,
				depth:   m.Depth,
				last:    last,
			})
			p.selable = append(p.selable, len(p.rows)-1)
		}
	}

	if p.selected >= len(p.selable) {
		p.selected = len(p.selable) - 1
	}
	if p.selected < 0 {
		p.selected = 0
	}
	p.ensureVisible()
}

// isLastAtDepth reports whether member i is the last one at its own depth within
// its subtree (used to pick └─ over ├─).
func isLastAtDepth(members []awsconfig.GroupedProfile, i int) bool {
	d := members[i].Depth
	for j := i + 1; j < len(members); j++ {
		if members[j].Depth < d {
			return true // left the subtree
		}
		if members[j].Depth == d {
			return false // another sibling follows
		}
	}
	return true
}

// SetFilter replaces the filter text and rebuilds.
func (p *ProfileList) SetFilter(s string) {
	p.filter = s
	p.rebuild()
}

// StartFilter enters filter-capture mode.
func (p *ProfileList) StartFilter() { p.filtering = true }

// StopFilter leaves filter-capture mode, keeping the current text.
func (p *ProfileList) StopFilter() { p.filtering = false }

// ClearFilter leaves filter-capture mode and clears the text.
func (p *ProfileList) ClearFilter() {
	p.filtering = false
	p.SetFilter("")
}

// HandleFilterKey feeds a keystroke to the active filter. Returns true when the
// key was consumed by the filter.
func (p *ProfileList) HandleFilterKey(key string, runes []rune) bool {
	if !p.filtering {
		return false
	}
	switch key {
	case "backspace":
		if n := len(p.filter); n > 0 {
			p.SetFilter(p.filter[:n-1])
		}
	default:
		if len(runes) == 1 {
			p.SetFilter(p.filter + string(runes))
		}
	}
	return true
}

// MoveUp / MoveDown move the selection among selectable rows.
func (p *ProfileList) MoveUp() {
	if p.selected > 0 {
		p.selected--
		p.ensureVisible()
	}
}

func (p *ProfileList) MoveDown() {
	if p.selected < len(p.selable)-1 {
		p.selected++
		p.ensureVisible()
	}
}

// visibleRows is the number of rows the body can display (excluding the title
// line and the filter line).
func (p ProfileList) visibleRows() int {
	h := p.height - 1 // title
	if p.filtering || p.filter != "" {
		h--
	}
	if h < 1 {
		h = 1
	}
	return h
}

// ensureVisible scrolls so the selected row stays within the viewport.
func (p *ProfileList) ensureVisible() {
	if len(p.selable) == 0 {
		p.offset = 0
		return
	}
	sel := p.selable[p.selected]
	vis := p.visibleRows()
	if sel < p.offset {
		p.offset = sel
	}
	if sel >= p.offset+vis {
		p.offset = sel - vis + 1
	}
	// Prefer showing a group header when the first visible row is a profile whose
	// header sits just above and would otherwise be clipped.
	if p.offset > 0 && p.rows[p.offset].kind == rowProfile {
		if p.rows[p.offset-1].kind == rowHeader {
			p.offset--
		}
	}
	if p.offset < 0 {
		p.offset = 0
	}
}

// View renders the full widget.
func (p ProfileList) View() string {
	var b strings.Builder

	b.WriteString(plTitleStyle.Render("AWS profiles"))
	if n := len(p.selable); n > 0 {
		b.WriteString(plHeaderMeta.Render(fmt.Sprintf("  %d profile(s)", n)))
	}
	b.WriteByte('\n')

	if p.filtering || p.filter != "" {
		cursor := ""
		if p.filtering {
			cursor = "█"
		}
		b.WriteString(plFilterStyle.Render("/ "+p.filter+cursor) + "\n")
	}

	if len(p.selable) == 0 {
		b.WriteString(plEmptyStyle.Render("No profile matches the filter."))
		return b.String()
	}

	vis := p.visibleRows()
	end := p.offset + vis
	if end > len(p.rows) {
		end = len(p.rows)
	}

	for i := p.offset; i < end; i++ {
		b.WriteString(p.renderRow(i))
		if i < end-1 {
			b.WriteByte('\n')
		}
	}

	return b.String()
}

func (p ProfileList) renderRow(i int) string {
	r := p.rows[i]
	if r.kind == rowHeader {
		return p.renderHeader(r.group)
	}
	return p.renderProfile(i, r)
}

func (p ProfileList) renderHeader(g awsconfig.Group) string {
	switch g.Kind {
	case awsconfig.GroupSSO:
		return plHeaderSSO.Render("▼ SSO session") + "  " +
			plHeaderSSO.Render(g.Label) +
			plHeaderMeta.Render(fmt.Sprintf("  (%d)", len(g.Profiles)))
	case awsconfig.GroupDependency:
		return plHeaderDep.Render("▼ derived from") + "  " +
			plHeaderDep.Render(g.Label) +
			plHeaderMeta.Render(fmt.Sprintf("  (%d)", len(g.Profiles)))
	default:
		return plHeaderStand.Render("▼ standalone") +
			plHeaderMeta.Render(fmt.Sprintf("  (%d)", len(g.Profiles)))
	}
}

func (p ProfileList) renderProfile(rowIdx int, r row) string {
	// Is this the selected row?
	selected := p.selected >= 0 && p.selected < len(p.selable) && p.selable[p.selected] == rowIdx

	// Tree prefix based on depth.
	var tree string
	if r.depth == 0 {
		tree = "  "
	} else {
		branch := "├─ "
		if r.last {
			branch = "└─ "
		}
		tree = strings.Repeat("   ", r.depth-1) + "  " + branch
	}

	pointer := "  "
	if selected {
		pointer = plNameSelStyle.Render("▶ ")
	}

	name := r.profile.Name
	nameRendered := plNameStyle.Render(name)
	if selected {
		nameRendered = plNameSelStyle.Render(name)
	}

	// Trailing meta: auth kind + region, dimmed.
	meta := plKindStyle.Render(r.profile.AuthKind.String())
	if reg := r.profile.Region; reg != "" {
		meta += "  " + plRegionStyle.Render(reg)
	} else {
		meta += "  " + plRegionStyle.Render("(default region)")
	}

	left := pointer + plTreeStyle.Render(tree) + nameRendered
	return padRight(left, p.nameColWidth()) + "  " + meta
}

// nameColWidth is the column at which meta (kind/region) is aligned.
func (p ProfileList) nameColWidth() int {
	w := p.width / 2
	if w < 28 {
		w = 28
	}
	if w > 48 {
		w = 48
	}
	return w
}

// padRight pads s with spaces to the given visible width (accounting for ANSI).
func padRight(s string, width int) string {
	gap := width - lipgloss.Width(s)
	if gap <= 0 {
		return s
	}
	return s + strings.Repeat(" ", gap)
}

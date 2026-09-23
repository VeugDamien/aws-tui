package screens

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/VeugDamien/aws-tui/internal/awsclient"
)

// Column content widths (generous; the table scrolls horizontally if it does not
// fit the terminal). These are the widths of the text cells, excluding the single
// space gutter rendered between columns.
const (
	colNameW       = 40
	colInstanceIDW = 19
	colStateW      = 10
	colTypeW       = 12
	colPrivateIPW  = 15
	colAZW         = 12
	colGutter      = 1 // space between columns
)

var ec2Headers = []struct {
	title string
	width int
}{
	{"Name", colNameW},
	{"Instance ID", colInstanceIDW},
	{"State", colStateW},
	{"Type", colTypeW},
	{"Private IP", colPrivateIPW},
	{"AZ", colAZW},
}

var (
	ec2HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("63"))

	ec2SelectedStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("230")).
				Background(lipgloss.Color("63"))

	ec2ScrollHintStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("241"))
)

// EC2Table is a custom table over the instance list supporting text filtering,
// vertical scrolling and horizontal scrolling (the bubbles table cannot scroll
// horizontally). Rows are rendered as plain strings and cropped to the horizontal
// window before any ANSI styling is applied, avoiding ANSI-slicing issues.
type EC2Table struct {
	all      []awsclient.Instance
	filtered []awsclient.Instance

	Filter    string
	Filtering bool

	cursor  int // selected row in filtered
	top     int // first visible row (vertical scroll)
	hScroll int // horizontal scroll offset in columns

	width  int // available render width
	height int // available render height (rows incl. header)
}

// NewEC2Table builds an empty table sized to width/height.
func NewEC2Table(width, height int) EC2Table {
	return EC2Table{width: width, height: height}
}

// Focus is a no-op kept for API compatibility (this widget is always "focused"
// while its screen is active).
func (e *EC2Table) Focus() {}

// SetInstances replaces the dataset and refreshes the filtered view.
func (e *EC2Table) SetInstances(instances []awsclient.Instance) {
	e.all = instances
	e.apply()
}

// SetFilter updates the text filter and refreshes the filtered view.
func (e *EC2Table) SetFilter(f string) {
	e.Filter = f
	e.apply()
}

// Resize updates the available render area.
func (e *EC2Table) Resize(width, height int) {
	e.width = width
	e.height = height
	e.clampHScroll()
}

// MoveUp moves the selection up by n rows.
func (e *EC2Table) MoveUp(n int) {
	e.cursor -= n
	if e.cursor < 0 {
		e.cursor = 0
	}
	e.ensureVisible()
}

// MoveDown moves the selection down by n rows.
func (e *EC2Table) MoveDown(n int) {
	e.cursor += n
	if e.cursor > len(e.filtered)-1 {
		e.cursor = len(e.filtered) - 1
	}
	if e.cursor < 0 {
		e.cursor = 0
	}
	e.ensureVisible()
}

// ScrollLeft scrolls the columns left by n.
func (e *EC2Table) ScrollLeft(n int) {
	e.hScroll -= n
	if e.hScroll < 0 {
		e.hScroll = 0
	}
}

// ScrollRight scrolls the columns right by n.
func (e *EC2Table) ScrollRight(n int) {
	e.hScroll += n
	e.clampHScroll()
}

// SelectedInstance returns the highlighted instance, if any.
func (e *EC2Table) SelectedInstance() (awsclient.Instance, bool) {
	if e.cursor < 0 || e.cursor >= len(e.filtered) {
		return awsclient.Instance{}, false
	}
	return e.filtered[e.cursor], true
}

// Count returns the number of visible (filtered) instances.
func (e *EC2Table) Count() int { return len(e.filtered) }

// TotalCount returns the number of instances before filtering.
func (e *EC2Table) TotalCount() int { return len(e.all) }

// CanScrollHorizontally reports whether the full table is wider than the view.
func (e *EC2Table) CanScrollHorizontally() bool {
	return e.naturalWidth() > e.width
}

// naturalWidth is the full rendered width of the table at full column widths.
func (e *EC2Table) naturalWidth() int {
	w := 0
	for i, h := range ec2Headers {
		w += h.width
		if i < len(ec2Headers)-1 {
			w += colGutter
		}
	}
	return w
}

func (e *EC2Table) clampHScroll() {
	maxScroll := e.naturalWidth() - e.width
	if maxScroll < 0 {
		maxScroll = 0
	}
	if e.hScroll > maxScroll {
		e.hScroll = maxScroll
	}
	if e.hScroll < 0 {
		e.hScroll = 0
	}
}

// bodyRows returns how many data rows fit (height minus header line).
func (e *EC2Table) bodyRows() int {
	r := e.height - 1
	if r < 1 {
		r = 1
	}
	return r
}

func (e *EC2Table) ensureVisible() {
	rows := e.bodyRows()
	if e.cursor < e.top {
		e.top = e.cursor
	}
	if e.cursor >= e.top+rows {
		e.top = e.cursor - rows + 1
	}
	if e.top < 0 {
		e.top = 0
	}
}

func (e *EC2Table) apply() {
	q := strings.ToLower(strings.TrimSpace(e.Filter))
	e.filtered = e.filtered[:0]
	for _, inst := range e.all {
		if q != "" {
			hay := strings.ToLower(inst.Name + " " + inst.InstanceID + " " + inst.PrivateIP + " " + inst.State)
			if !strings.Contains(hay, q) {
				continue
			}
		}
		e.filtered = append(e.filtered, inst)
	}
	if e.cursor > len(e.filtered)-1 {
		e.cursor = len(e.filtered) - 1
	}
	if e.cursor < 0 {
		e.cursor = 0
	}
	e.ensureVisible()
}

// View renders the table (header + visible rows), cropped to the horizontal window.
func (e *EC2Table) View() string {
	var b strings.Builder

	// Header (plain, then cropped, then styled).
	header := e.crop(e.formatHeader())
	b.WriteString(ec2HeaderStyle.Render(header))
	b.WriteString("\n")

	rows := e.bodyRows()
	end := e.top + rows
	if end > len(e.filtered) {
		end = len(e.filtered)
	}

	for i := e.top; i < end; i++ {
		line := e.crop(e.formatRow(e.filtered[i]))
		if i == e.cursor {
			line = ec2SelectedStyle.Render(line)
		}
		b.WriteString(line)
		if i < end-1 {
			b.WriteString("\n")
		}
	}

	// Fill remaining rows with blank lines to keep a stable height.
	for i := end - e.top; i < rows; i++ {
		b.WriteString("\n")
	}

	if e.CanScrollHorizontally() {
		b.WriteString("\n" + ec2ScrollHintStyle.Render("← / → défiler horizontalement"))
	}
	return b.String()
}

// formatHeader builds the plain header line.
func (e *EC2Table) formatHeader() string {
	cells := make([]string, len(ec2Headers))
	for i, h := range ec2Headers {
		cells[i] = pad(h.title, h.width)
	}
	return strings.Join(cells, strings.Repeat(" ", colGutter))
}

// formatRow builds a plain data line for an instance.
func (e *EC2Table) formatRow(inst awsclient.Instance) string {
	name := inst.Name
	if name == "" {
		name = "-"
	}
	cells := []string{
		pad(name, colNameW),
		pad(inst.InstanceID, colInstanceIDW),
		pad(inst.State, colStateW),
		pad(inst.Type, colTypeW),
		pad(dash(inst.PrivateIP), colPrivateIPW),
		pad(inst.AZ, colAZW),
	}
	return strings.Join(cells, strings.Repeat(" ", colGutter))
}

// crop returns the horizontal window [hScroll, hScroll+width] of a plain string,
// padded to the view width so every line has a uniform length.
func (e *EC2Table) crop(s string) string {
	r := []rune(s)
	start := e.hScroll
	if start > len(r) {
		start = len(r)
	}
	end := start + e.width
	if end > len(r) {
		end = len(r)
	}
	window := string(r[start:end])
	return pad(window, e.width)
}

// pad truncates or right-pads s to exactly w runes.
func pad(s string, w int) string {
	r := []rune(s)
	if len(r) > w {
		if w <= 1 {
			return string(r[:w])
		}
		return string(r[:w-1]) + "…"
	}
	return s + strings.Repeat(" ", w-len(r))
}

func dash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

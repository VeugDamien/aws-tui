package screens

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/VeugDamien/aws-tui/internal/awsclient"
)

// ASG table column widths.
const (
	colASGNameW    = 44
	colASGCapW     = 16 // "min/des/max"
	colASGInstW    = 12 // "healthy/total"
	colASGLaunchW  = 22
	colASGStatusW  = 12
	colASGGutter   = 2
)

var asgHeaders = []struct {
	title string
	width int
}{
	{"Name", colASGNameW},
	{"min/des/max", colASGCapW},
	{"Instances", colASGInstW},
	{"Launch", colASGLaunchW},
	{"Status", colASGStatusW},
}

var (
	asgHeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("63"))

	asgSelectedStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("230")).
				Background(lipgloss.Color("63"))
)

// ASGTable is a filterable, scrollable table over Auto Scaling groups.
type ASGTable struct {
	all      []awsclient.AutoScalingGroup
	filtered []awsclient.AutoScalingGroup

	Filter    string
	Filtering bool

	cursor int
	top    int

	width  int
	height int
}

// NewASGTable builds an empty table sized to width/height.
func NewASGTable(width, height int) ASGTable {
	return ASGTable{width: width, height: height}
}

// SetGroups replaces the dataset and refreshes the filtered view.
func (t *ASGTable) SetGroups(groups []awsclient.AutoScalingGroup) {
	t.all = groups
	t.apply()
}

// SetFilter updates the text filter and refreshes.
func (t *ASGTable) SetFilter(f string) {
	t.Filter = f
	t.apply()
}

// Resize updates the render area.
func (t *ASGTable) Resize(width, height int) {
	t.width = width
	t.height = height
}

// MoveUp / MoveDown move the selection.
func (t *ASGTable) MoveUp(n int) {
	t.cursor -= n
	if t.cursor < 0 {
		t.cursor = 0
	}
	t.ensureVisible()
}

func (t *ASGTable) MoveDown(n int) {
	t.cursor += n
	if t.cursor > len(t.filtered)-1 {
		t.cursor = len(t.filtered) - 1
	}
	if t.cursor < 0 {
		t.cursor = 0
	}
	t.ensureVisible()
}

// SelectedGroup returns the highlighted group, if any.
func (t *ASGTable) SelectedGroup() (awsclient.AutoScalingGroup, bool) {
	if t.cursor < 0 || t.cursor >= len(t.filtered) {
		return awsclient.AutoScalingGroup{}, false
	}
	return t.filtered[t.cursor], true
}

// Count / TotalCount report visible and total group counts.
func (t *ASGTable) Count() int      { return len(t.filtered) }
func (t *ASGTable) TotalCount() int { return len(t.all) }

func (t *ASGTable) bodyRows() int {
	r := t.height - 1
	if r < 1 {
		r = 1
	}
	return r
}

func (t *ASGTable) ensureVisible() {
	rows := t.bodyRows()
	if t.cursor < t.top {
		t.top = t.cursor
	}
	if t.cursor >= t.top+rows {
		t.top = t.cursor - rows + 1
	}
	if t.top < 0 {
		t.top = 0
	}
}

func (t *ASGTable) apply() {
	q := strings.ToLower(strings.TrimSpace(t.Filter))
	t.filtered = t.filtered[:0]
	for _, g := range t.all {
		if q != "" {
			hay := strings.ToLower(g.Name + " " + g.LaunchName + " " + strings.Join(g.AZs, " "))
			if !strings.Contains(hay, q) {
				continue
			}
		}
		t.filtered = append(t.filtered, g)
	}
	if t.cursor > len(t.filtered)-1 {
		t.cursor = len(t.filtered) - 1
	}
	if t.cursor < 0 {
		t.cursor = 0
	}
	t.ensureVisible()
}

// View renders the table (header + visible rows).
func (t *ASGTable) View() string {
	var b strings.Builder

	b.WriteString(asgHeaderStyle.Render(t.formatHeader()))
	b.WriteString("\n")

	rows := t.bodyRows()
	end := t.top + rows
	if end > len(t.filtered) {
		end = len(t.filtered)
	}

	for i := t.top; i < end; i++ {
		line := t.formatRow(t.filtered[i])
		if i == t.cursor {
			line = asgSelectedStyle.Render(line)
		}
		b.WriteString(line)
		if i < end-1 {
			b.WriteString("\n")
		}
	}
	for i := end - t.top; i < rows; i++ {
		b.WriteString("\n")
	}
	return b.String()
}

func (t *ASGTable) formatHeader() string {
	cells := make([]string, len(asgHeaders))
	for i, h := range asgHeaders {
		cells[i] = pad(h.title, h.width)
	}
	return strings.Join(cells, strings.Repeat(" ", colASGGutter))
}

func (t *ASGTable) formatRow(g awsclient.AutoScalingGroup) string {
	cap := fmt.Sprintf("%d/%d/%d", g.MinSize, g.DesiredCapacity, g.MaxSize)
	inst := fmt.Sprintf("%d/%d", g.HealthyCount(), g.InstanceCount())
	launch := g.LaunchName
	if launch == "" {
		launch = g.LaunchType
	}
	status := g.Status
	if status == "" {
		status = "active"
	}
	cells := []string{
		pad(g.Name, colASGNameW),
		pad(cap, colASGCapW),
		pad(inst, colASGInstW),
		pad(dash(launch), colASGLaunchW),
		pad(status, colASGStatusW),
	}
	return strings.Join(cells, strings.Repeat(" ", colASGGutter))
}

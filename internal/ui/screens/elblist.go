package screens

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/VeugDamien/aws-tui/internal/awsclient"
)

// ELB table column widths.
const (
	colELBNameW   = 32
	colELBTypeW   = 12
	colELBSchemeW = 16
	colELBStateW  = 12
	colELBDNSW    = 48
	colELBGutter  = 2
)

var elbHeaders = []struct {
	title string
	width int
}{
	{"Nom", colELBNameW},
	{"Type", colELBTypeW},
	{"Schéma", colELBSchemeW},
	{"État", colELBStateW},
	{"DNS", colELBDNSW},
}

var (
	elbHeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("63"))

	elbSelectedStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("230")).
				Background(lipgloss.Color("63"))

	elbScrollHintStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("241"))
)

// ELBTable is a filterable table over ELBv2 load balancers with horizontal
// scrolling (the DNS column is wide).
type ELBTable struct {
	all      []awsclient.LoadBalancer
	filtered []awsclient.LoadBalancer

	Filter    string
	Filtering bool

	cursor  int
	top     int
	hScroll int

	width  int
	height int
}

// NewELBTable builds an empty table sized to width/height.
func NewELBTable(width, height int) ELBTable {
	return ELBTable{width: width, height: height}
}

// SetLoadBalancers replaces the dataset and refreshes the filtered view.
func (t *ELBTable) SetLoadBalancers(lbs []awsclient.LoadBalancer) {
	t.all = lbs
	t.apply()
}

// SetFilter updates the text filter and refreshes.
func (t *ELBTable) SetFilter(f string) {
	t.Filter = f
	t.apply()
}

// Resize updates the render area.
func (t *ELBTable) Resize(width, height int) {
	t.width = width
	t.height = height
	t.clampHScroll()
}

// MoveUp / MoveDown move the selection.
func (t *ELBTable) MoveUp(n int) {
	t.cursor -= n
	if t.cursor < 0 {
		t.cursor = 0
	}
	t.ensureVisible()
}

func (t *ELBTable) MoveDown(n int) {
	t.cursor += n
	if t.cursor > len(t.filtered)-1 {
		t.cursor = len(t.filtered) - 1
	}
	if t.cursor < 0 {
		t.cursor = 0
	}
	t.ensureVisible()
}

// ScrollLeft / ScrollRight scroll the columns horizontally.
func (t *ELBTable) ScrollLeft(n int) {
	t.hScroll -= n
	if t.hScroll < 0 {
		t.hScroll = 0
	}
}

func (t *ELBTable) ScrollRight(n int) {
	t.hScroll += n
	t.clampHScroll()
}

// SelectedLoadBalancer returns the highlighted load balancer, if any.
func (t *ELBTable) SelectedLoadBalancer() (awsclient.LoadBalancer, bool) {
	if t.cursor < 0 || t.cursor >= len(t.filtered) {
		return awsclient.LoadBalancer{}, false
	}
	return t.filtered[t.cursor], true
}

// Count / TotalCount report visible and total counts.
func (t *ELBTable) Count() int      { return len(t.filtered) }
func (t *ELBTable) TotalCount() int { return len(t.all) }

// CanScrollHorizontally reports whether the table is wider than the view.
func (t *ELBTable) CanScrollHorizontally() bool { return t.naturalWidth() > t.width }

func (t *ELBTable) naturalWidth() int {
	w := 0
	for i, h := range elbHeaders {
		w += h.width
		if i < len(elbHeaders)-1 {
			w += colELBGutter
		}
	}
	return w
}

func (t *ELBTable) clampHScroll() {
	maxScroll := t.naturalWidth() - t.width
	if maxScroll < 0 {
		maxScroll = 0
	}
	if t.hScroll > maxScroll {
		t.hScroll = maxScroll
	}
	if t.hScroll < 0 {
		t.hScroll = 0
	}
}

func (t *ELBTable) bodyRows() int {
	r := t.height - 1
	if r < 1 {
		r = 1
	}
	return r
}

func (t *ELBTable) ensureVisible() {
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

func (t *ELBTable) apply() {
	q := strings.ToLower(strings.TrimSpace(t.Filter))
	t.filtered = t.filtered[:0]
	for _, lb := range t.all {
		if q != "" {
			hay := strings.ToLower(lb.Name + " " + lb.Type + " " + lb.DNSName + " " + lb.State)
			if !strings.Contains(hay, q) {
				continue
			}
		}
		t.filtered = append(t.filtered, lb)
	}
	if t.cursor > len(t.filtered)-1 {
		t.cursor = len(t.filtered) - 1
	}
	if t.cursor < 0 {
		t.cursor = 0
	}
	t.ensureVisible()
}

// View renders the table (header + visible rows), cropped to the horizontal window.
func (t *ELBTable) View() string {
	var b strings.Builder

	b.WriteString(elbHeaderStyle.Render(t.crop(t.formatHeader())))
	b.WriteString("\n")

	rows := t.bodyRows()
	end := t.top + rows
	if end > len(t.filtered) {
		end = len(t.filtered)
	}

	for i := t.top; i < end; i++ {
		line := t.crop(t.formatRow(t.filtered[i]))
		if i == t.cursor {
			line = elbSelectedStyle.Render(line)
		}
		b.WriteString(line)
		if i < end-1 {
			b.WriteString("\n")
		}
	}
	for i := end - t.top; i < rows; i++ {
		b.WriteString("\n")
	}

	if t.CanScrollHorizontally() {
		b.WriteString("\n" + elbScrollHintStyle.Render("← / → défiler horizontalement"))
	}
	return b.String()
}

func (t *ELBTable) formatHeader() string {
	cells := make([]string, len(elbHeaders))
	for i, h := range elbHeaders {
		cells[i] = pad(h.title, h.width)
	}
	return strings.Join(cells, strings.Repeat(" ", colELBGutter))
}

func (t *ELBTable) formatRow(lb awsclient.LoadBalancer) string {
	cells := []string{
		pad(lb.Name, colELBNameW),
		pad(dash(lb.Type), colELBTypeW),
		pad(dash(lb.Scheme), colELBSchemeW),
		pad(dash(lb.State), colELBStateW),
		pad(dash(lb.DNSName), colELBDNSW),
	}
	return strings.Join(cells, strings.Repeat(" ", colELBGutter))
}

// crop returns the horizontal window of a plain string, padded to the view width.
func (t *ELBTable) crop(s string) string {
	r := []rune(s)
	start := t.hScroll
	if start > len(r) {
		start = len(r)
	}
	end := start + t.width
	if end > len(r) {
		end = len(r)
	}
	return pad(string(r[start:end]), t.width)
}

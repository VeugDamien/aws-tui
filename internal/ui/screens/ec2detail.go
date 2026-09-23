package screens

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/VeugDamien/aws-tui/internal/awsclient"
)

// blockState tracks async loading of a detail block.
type blockState int

const (
	BlockLoading blockState = iota
	BlockLoaded
	BlockError
)

// EC2Detail holds the state of the instance detail page.
type EC2Detail struct {
	Instance awsclient.Instance

	SGState blockState
	SGErr   error
	SGs     []awsclient.SecurityGroup

	MetricsState blockState
	MetricsErr   error
	Metrics      awsclient.InstanceMetrics

	// MetricWindowIdx indexes MetricWindows for the selected time window.
	MetricWindowIdx int

	// Copy menu overlay state.
	copyOpen   bool
	copyItems  []CopyField
	copyCursor int

	scroll int
	width  int
	height int
}

// CopyField is one copyable value offered by the detail page's copy menu.
type CopyField struct {
	Label string
	Value string
}

// MetricWindows are the selectable time windows (in minutes) for the metrics block.
var MetricWindows = []int{60, 180, 720, 1440}

// MetricWindowLabel renders a window (minutes) as a short label.
func MetricWindowLabel(min int) string {
	if min%60 == 0 {
		return fmt.Sprintf("%dh", min/60)
	}
	return fmt.Sprintf("%dmin", min)
}

// MetricWindow returns the currently selected window in minutes.
func (d *EC2Detail) MetricWindow() int {
	if d.MetricWindowIdx < 0 || d.MetricWindowIdx >= len(MetricWindows) {
		return MetricWindows[0]
	}
	return MetricWindows[d.MetricWindowIdx]
}

// CycleMetricWindow advances to the next time window and returns the new value.
func (d *EC2Detail) CycleMetricWindow() int {
	d.MetricWindowIdx = (d.MetricWindowIdx + 1) % len(MetricWindows)
	return d.MetricWindow()
}

// NewEC2Detail builds the detail page for an instance with both async blocks
// initially in the loading state.
func NewEC2Detail(inst awsclient.Instance, width, height int) EC2Detail {
	return EC2Detail{
		Instance:     inst,
		SGState:      BlockLoading,
		MetricsState: BlockLoading,
		width:        width,
		height:       height,
	}
}

// Resize updates the render area.
func (d *EC2Detail) Resize(width, height int) {
	d.width = width
	d.height = height
}

// CopyMenuOpen reports whether the copy overlay is currently shown.
func (d *EC2Detail) CopyMenuOpen() bool { return d.copyOpen }

// OpenCopyMenu builds the list of copyable fields (skipping empty ones) and shows
// the overlay. region/account are used to build the instance ARN when possible.
func (d *EC2Detail) OpenCopyMenu(region, account string) {
	i := d.Instance
	candidates := []CopyField{
		{"Instance ID", i.InstanceID},
		{"Name", i.Name},
		{"Private IP", i.PrivateIP},
		{"Public IP", i.PublicIP},
		{"Type", i.Type},
		{"AZ", i.AZ},
		{"VPC", i.VPCID},
		{"Subnet", i.SubnetID},
		{"AMI", i.ImageID},
		{"ARN", instanceARN(region, account, i.InstanceID)},
	}
	d.copyItems = d.copyItems[:0]
	for _, c := range candidates {
		if c.Value != "" {
			d.copyItems = append(d.copyItems, c)
		}
	}
	d.copyCursor = 0
	d.copyOpen = true
}

// CloseCopyMenu hides the overlay.
func (d *EC2Detail) CloseCopyMenu() { d.copyOpen = false }

// CopyMenuUp / CopyMenuDown move the overlay cursor.
func (d *EC2Detail) CopyMenuUp() {
	if d.copyCursor > 0 {
		d.copyCursor--
	}
}

func (d *EC2Detail) CopyMenuDown() {
	if d.copyCursor < len(d.copyItems)-1 {
		d.copyCursor++
	}
}

// SelectedCopyField returns the highlighted field of the copy menu.
func (d *EC2Detail) SelectedCopyField() (CopyField, bool) {
	if d.copyCursor < 0 || d.copyCursor >= len(d.copyItems) {
		return CopyField{}, false
	}
	return d.copyItems[d.copyCursor], true
}

// instanceARN builds an EC2 instance ARN when region and account are known.
func instanceARN(region, account, instanceID string) string {
	if region == "" || account == "" || instanceID == "" {
		return ""
	}
	return fmt.Sprintf("arn:aws:ec2:%s:%s:instance/%s", region, account, instanceID)
}

// ScrollUp / ScrollDown move the viewport.
func (d *EC2Detail) ScrollUp(n int) {
	d.scroll -= n
	if d.scroll < 0 {
		d.scroll = 0
	}
}

func (d *EC2Detail) ScrollDown(n int) {
	d.scroll += n
	// clamped against content in View.
}

var (
	detailTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("230")).
				Background(lipgloss.Color("63")).
				Padding(0, 1)

	detailSectionStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("63")).
				Border(lipgloss.NormalBorder(), false, false, true, false).
				BorderForeground(lipgloss.Color("240"))

	detailLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("250")).
				Width(18)

	detailMutedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	detailErrStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	detailOKStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("78"))

	detailMetricsBoxStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("63")).
				Padding(0, 1)

	detailMetricsTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("63"))

	detailMetricLabelStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("250"))

	detailSparkStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("78"))

	detailAxisStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	detailWindowActiveStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("230")).
				Background(lipgloss.Color("63")).
				Padding(0, 1)

	detailTagStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("250"))

	detailCopyBoxStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("63")).
				Padding(0, 1)

	detailCopyTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("230")).
				Background(lipgloss.Color("63")).
				Padding(0, 1)

	detailCopySelStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("230")).
				Background(lipgloss.Color("57"))

	detailCopyLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("250"))

	detailCopyValStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252"))
)

// View renders the detail page as two columns: a scrollable left column
// (general + security) and a bordered metrics box on the right.
func (d *EC2Detail) View(spinner string) string {
	name := d.Instance.Name
	if name == "" {
		name = d.Instance.InstanceID
	}
	title := detailTitleStyle.Render(" "+name+" ") + "  " + detailMutedStyle.Render(d.Instance.InstanceID)

	leftW, rightW := d.columnWidths()

	// Left column content (general + security), vertically scrollable.
	var left strings.Builder
	left.WriteString(d.renderGeneral(leftW))
	left.WriteString("\n")
	left.WriteString(d.renderSecurity(spinner, leftW))
	leftStr, total, shown := d.applyScroll(left.String())

	// Right column: metrics box stacked above a tags box, both bordered.
	metricsBox := d.renderMetricsBox(spinner, rightW)
	// Tags budget = remaining height after title (2 lines) and the metrics box,
	// minus the newline separator (1) and the tags box chrome (border 2 + title 2).
	tagsBudget := d.height - 2 - lipgloss.Height(metricsBox) - 1 - 4
	right := metricsBox + "\n" + d.renderTagsBox(rightW, tagsBudget)

	columns := lipgloss.JoinHorizontal(
		lipgloss.Top,
		lipgloss.NewStyle().Width(leftW).Render(leftStr),
		"  ",
		right,
	)

	out := title + "\n\n" + columns
	if total > shown {
		out += "\n" + detailMutedStyle.Render(fmt.Sprintf("↑/↓ scroll (%d/%d)", d.scroll+1, total))
	}
	if d.copyOpen {
		return title + "\n\n" + d.renderCopyMenu()
	}
	return out
}

// renderCopyMenu renders the copy overlay: a bordered list of copyable fields.
func (d *EC2Detail) renderCopyMenu() string {
	var b strings.Builder
	b.WriteString(detailCopyTitleStyle.Render(" Copy ") + "\n\n")

	for i, f := range d.copyItems {
		label := pad(f.Label, 13)
		val := truncateRunes(f.Value, 48)
		row := label + " " + val
		if i == d.copyCursor {
			b.WriteString("▶ " + detailCopySelStyle.Render(pad(row, 62)) + "\n")
		} else {
			b.WriteString("  " + detailCopyLabelStyle.Render(label) + " " + detailCopyValStyle.Render(val) + "\n")
		}
	}

	b.WriteString("\n" + detailMutedStyle.Render("↑/↓ select · enter copy · esc close"))
	return detailCopyBoxStyle.Render(b.String())
}

// truncateRunes shortens s to at most n runes with an ellipsis when cut.
func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n <= 1 {
		return string(r[:n])
	}
	return string(r[:n-1]) + "…"
}

// columnWidths splits the available width between the left content and the right
// metrics box. The metrics box is widened to fit a sparkline plus a time axis.
func (d *EC2Detail) columnWidths() (left, right int) {
	w := d.width
	if w < 60 {
		w = 60
	}
	// Metrics box target width (room for a wide sparkline + axis labels).
	right = 52
	if right > w/2 {
		right = w / 2
	}
	if right < 34 {
		right = 34
	}
	left = w - right - 2 // gap
	if left < 30 {
		left = 30
	}
	return left, right
}

// applyScroll returns the visible window of s plus total/shown line counts.
func (d *EC2Detail) applyScroll(s string) (string, int, int) {
	lines := strings.Split(s, "\n")
	view := d.height
	if view < 3 {
		view = 3
	}
	if d.scroll > len(lines)-1 {
		d.scroll = len(lines) - 1
	}
	if d.scroll < 0 {
		d.scroll = 0
	}
	end := d.scroll + view
	if end > len(lines) {
		end = len(lines)
	}
	return strings.Join(lines[d.scroll:end], "\n"), len(lines), view
}

func (d *EC2Detail) section(title string, width int) string {
	return detailSectionStyle.Width(width).Render(title) + "\n"
}

func kv(label, value string) string {
	if value == "" {
		value = "-"
	}
	return detailLabelStyle.Render(label) + value + "\n"
}

func (d *EC2Detail) renderGeneral(width int) string {
	i := d.Instance
	var b strings.Builder
	b.WriteString(d.section("General information", width))
	b.WriteString(kv("State", i.State))
	b.WriteString(kv("Type", i.Type))
	b.WriteString(kv("AZ", i.AZ))
	b.WriteString(kv("Private IP", i.PrivateIP))
	b.WriteString(kv("Public IP", i.PublicIP))
	b.WriteString(kv("VPC", i.VPCID))
	b.WriteString(kv("Subnet", i.SubnetID))
	b.WriteString(kv("AMI", i.ImageID))
	b.WriteString(kv("Architecture", i.Architecture))
	b.WriteString(kv("Platform", i.Platform))
	b.WriteString(kv("SSH key", i.KeyName))
	b.WriteString(kv("IAM profile", shortARN(i.IAMProfile)))
	b.WriteString(kv("Monitoring", i.Monitoring))
	b.WriteString(kv("Launched at", i.LaunchTime))
	return b.String()
}

func (d *EC2Detail) renderSecurity(spinner string, width int) string {
	var b strings.Builder
	b.WriteString(d.section("Security / network", width))

	switch d.SGState {
	case BlockLoading:
		b.WriteString(detailMutedStyle.Render(spinner+" Loading security groups…") + "\n")
	case BlockError:
		b.WriteString(detailErrStyle.Render("Error: "+errText(d.SGErr)) + "\n")
	case BlockLoaded:
		if len(d.SGs) == 0 {
			b.WriteString(detailMutedStyle.Render("No security group.") + "\n")
		}
		for _, sg := range d.SGs {
			b.WriteString(detailOKStyle.Render("● "+sg.Name) + " " + detailMutedStyle.Render("("+sg.ID+")") + "\n")
			b.WriteString(detailMutedStyle.Render("  Inbound:") + "\n")
			b.WriteString(renderRules(sg.Inbound))
			b.WriteString(detailMutedStyle.Render("  Outbound:") + "\n")
			b.WriteString(renderRules(sg.Outbound))
		}
	}
	return b.String()
}

func renderRules(rules []awsclient.SGRule) string {
	if len(rules) == 0 {
		return "    " + detailMutedStyle.Render("(no rule)") + "\n"
	}
	var b strings.Builder
	for _, r := range rules {
		b.WriteString(fmt.Sprintf("    %-5s  %-10s  %s\n", r.Protocol, r.Ports, r.Source))
	}
	return b.String()
}

// renderMetricsBox renders the metrics inside a bordered box of the given width.
func (d *EC2Detail) renderMetricsBox(spinner string, width int) string {
	// Inner width = box width minus borders (2) and padding (2).
	inner := width - 4
	if inner < 16 {
		inner = 16
	}

	var b strings.Builder
	b.WriteString(detailMetricsTitleStyle.Render("Metrics") + "  " + d.windowSelector() + "\n")
	b.WriteString(detailMutedStyle.Render(d.metricsPeriodLabel()) + "\n\n")

	switch d.MetricsState {
	case BlockLoading:
		b.WriteString(detailMutedStyle.Render(spinner + " Loading…"))
	case BlockError:
		b.WriteString(detailErrStyle.Render("Error:") + "\n" + detailErrStyle.Render(errText(d.MetricsErr)))
	case BlockLoaded:
		win := d.Metrics.WindowMinutes
		b.WriteString(renderMetric(d.Metrics.CPU, false, inner, win))
		b.WriteString("\n")
		b.WriteString(renderMetric(d.Metrics.NetworkIn, true, inner, win))
		b.WriteString("\n")
		b.WriteString(renderMetric(d.Metrics.NetworkOut, true, inner, win))
	}

	return detailMetricsBoxStyle.Width(width - 2).Render(b.String())
}

// minTagsShown is the floor of tag lines we show even on a short terminal.
const minTagsShown = 3

// renderTagsBox renders the instance tags inside a bordered box, showing as many as
// fit in maxLines tag rows (computed from the terminal height); the rest are
// summarised as "+N autres".
func (d *EC2Detail) renderTagsBox(width, maxLines int) string {
	inner := width - 4
	if inner < 16 {
		inner = 16
	}

	tags := d.Instance.Tags
	var b strings.Builder
	b.WriteString(detailMetricsTitleStyle.Render(fmt.Sprintf("Tags (%d)", len(tags))) + "\n\n")

	if len(tags) == 0 {
		b.WriteString(detailMutedStyle.Render("(no tag)"))
		return detailMetricsBoxStyle.Width(width - 2).Render(b.String())
	}

	// Number of tag rows we can display, clamped to a sensible floor.
	capacity := maxLines
	if capacity < minTagsShown {
		capacity = minTagsShown
	}

	shown := tags
	overflow := false
	if len(tags) > capacity {
		// Reserve one row for the "+N autres" summary.
		keep := capacity - 1
		if keep < 1 {
			keep = 1
		}
		shown = tags[:keep]
		overflow = true
	}

	for _, t := range shown {
		line := t.Key + " = " + t.Value
		b.WriteString(detailTagStyle.Render(pad(line, inner)) + "\n")
	}
	if overflow {
		b.WriteString(detailMutedStyle.Render(fmt.Sprintf("+%d more", len(tags)-len(shown))))
	}

	return detailMetricsBoxStyle.Width(width - 2).Render(strings.TrimRight(b.String(), "\n"))
}

// windowSelector renders the selectable windows with the active one highlighted.
func (d *EC2Detail) windowSelector() string {
	parts := make([]string, len(MetricWindows))
	for i, w := range MetricWindows {
		label := MetricWindowLabel(w)
		if i == d.MetricWindowIdx {
			parts[i] = detailWindowActiveStyle.Render(label)
		} else {
			parts[i] = detailMutedStyle.Render(label)
		}
	}
	return strings.Join(parts, detailMutedStyle.Render("/"))
}

// metricsPeriodLabel renders e.g. "Window 1h · 1 pt/5min" for the selected window.
func (d *EC2Detail) metricsPeriodLabel() string {
	win := d.MetricWindow()
	// Period reflects the loaded data when available, else estimate from window.
	per := d.Metrics.PeriodMinutes
	if per == 0 {
		switch {
		case win <= 180:
			per = 5
		case win <= 720:
			per = 15
		default:
			per = 60
		}
	}
	return fmt.Sprintf("Window %s · 1 pt/%dmin · [m] change", MetricWindowLabel(win), per)
}

// renderMetric renders one metric: label, a wide sparkline, a time axis and stats.
func renderMetric(s awsclient.MetricSeries, bytesUnit bool, inner, windowMin int) string {
	if s.Label == "" {
		return detailMutedStyle.Render("(unavailable)")
	}

	sparkW := inner
	if sparkW < 12 {
		sparkW = 12
	}
	spark := SparklineFixed(s.Values, sparkW)
	axis := timeAxis(windowMin, sparkW)

	var latest, avg, max string
	if bytesUnit {
		latest, avg, max = humanBytes(s.Latest), humanBytes(s.Avg), humanBytes(s.Max)
	} else {
		latest = fmt.Sprintf("%.1f%%", s.Latest)
		avg = fmt.Sprintf("%.1f%%", s.Avg)
		max = fmt.Sprintf("%.1f%%", s.Max)
	}

	var b strings.Builder
	b.WriteString(detailMetricLabelStyle.Render(s.Label) + "\n")
	b.WriteString(detailSparkStyle.Render(spark) + "\n")
	b.WriteString(detailAxisStyle.Render(axis) + "\n")
	b.WriteString(detailMutedStyle.Render(fmt.Sprintf("cur %s · avg %s · max %s", latest, avg, max)))
	return b.String()
}

// timeAxis renders a left-aligned "-Xm"/"-Xh" start label and a right-aligned
// "now" label spanning exactly width characters.
func timeAxis(windowMin, width int) string {
	left := fmt.Sprintf("-%dm", windowMin)
	if windowMin%60 == 0 {
		left = fmt.Sprintf("-%dh", windowMin/60)
	}
	right := "now"
	gap := width - len([]rune(left)) - len([]rune(right))
	if gap < 1 {
		// Too narrow: just show the start label truncated to width.
		r := []rune(left)
		if len(r) > width {
			return string(r[:width])
		}
		return left
	}
	return left + strings.Repeat(" ", gap) + right
}

// helpers

func shortARN(arn string) string {
	if i := strings.LastIndex(arn, "/"); i >= 0 {
		return arn[i+1:]
	}
	return arn
}

func errText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

package screens

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/VeugDamien/aws-tui/internal/awsclient"
)

// ASGDetail holds the state of the Auto Scaling group detail page.
type ASGDetail struct {
	Group awsclient.AutoScalingGroup

	// Target groups (async: fetched with health).
	TGState blockState
	TGErr   error
	TGs     []awsclient.TargetGroup

	// Recent scaling activities (async).
	ActState blockState
	ActErr   error
	Acts     []awsclient.ScalingActivity

	scroll int
	width  int
	height int
}

var (
	asgHealthyStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("78"))
	asgUnhealthyStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	asgPendingStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
)

// NewASGDetail builds the detail page with both async blocks loading.
func NewASGDetail(g awsclient.AutoScalingGroup, width, height int) ASGDetail {
	return ASGDetail{
		Group:    g,
		TGState:  BlockLoading,
		ActState: BlockLoading,
		width:    width,
		height:   height,
	}
}

// Resize updates the render area.
func (d *ASGDetail) Resize(width, height int) {
	d.width = width
	d.height = height
}

// ScrollUp / ScrollDown move the viewport.
func (d *ASGDetail) ScrollUp(n int) {
	d.scroll -= n
	if d.scroll < 0 {
		d.scroll = 0
	}
}

func (d *ASGDetail) ScrollDown(n int) { d.scroll += n }

// View renders the ASG detail page as a single scrollable column.
func (d *ASGDetail) View(spinner string) string {
	g := d.Group
	title := detailTitleStyle.Render(" "+g.Name+" ") + "  " +
		detailMutedStyle.Render(fmt.Sprintf("%d/%d/%d (min/dés/max)", g.MinSize, g.DesiredCapacity, g.MaxSize))

	var b strings.Builder
	width := d.width
	if width < 40 {
		width = 40
	}

	b.WriteString(d.renderCapacities(width))
	b.WriteString("\n")
	b.WriteString(d.renderInstances(width))
	b.WriteString("\n")
	b.WriteString(d.renderTargetGroups(spinner, width))
	b.WriteString("\n")
	b.WriteString(d.renderActivities(spinner, width))

	body, total, shown := d.applyScroll(b.String())
	out := title + "\n\n" + body
	if total > shown {
		out += "\n" + detailMutedStyle.Render(fmt.Sprintf("↑/↓ défiler (%d/%d)", d.scroll+1, total))
	}
	return out
}

func (d *ASGDetail) applyScroll(s string) (string, int, int) {
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

func (d *ASGDetail) renderCapacities(width int) string {
	g := d.Group
	var b strings.Builder
	b.WriteString(detailSectionStyle.Width(width).Render("Capacités & configuration") + "\n")
	b.WriteString(kv("Désirée", fmt.Sprintf("%d", g.DesiredCapacity)))
	b.WriteString(kv("Min / Max", fmt.Sprintf("%d / %d", g.MinSize, g.MaxSize)))
	b.WriteString(kv("Health check", g.HealthCheckType))
	b.WriteString(kv("Lancement", launchLabel(g)))
	b.WriteString(kv("Zones", strings.Join(g.AZs, ", ")))
	b.WriteString(kv("Subnets", g.VPCZoneID))
	if g.Status != "" {
		b.WriteString(kv("Statut", g.Status))
	}
	b.WriteString(kv("Créé le", g.CreatedTime))
	return b.String()
}

func launchLabel(g awsclient.AutoScalingGroup) string {
	if g.LaunchName == "" {
		return g.LaunchType
	}
	return fmt.Sprintf("%s (%s)", g.LaunchName, g.LaunchType)
}

func (d *ASGDetail) renderInstances(width int) string {
	g := d.Group
	var b strings.Builder
	title := fmt.Sprintf("Instances (%d/%d saines)", g.HealthyCount(), g.InstanceCount())
	b.WriteString(detailSectionStyle.Width(width).Render(title) + "\n")

	if len(g.Instances) == 0 {
		b.WriteString(detailMutedStyle.Render("Aucune instance.") + "\n")
		return b.String()
	}
	for _, inst := range g.Instances {
		health := healthDot(inst.HealthStatus)
		line := fmt.Sprintf("%s  %-19s  %-12s  %s",
			health, inst.InstanceID, inst.AZ, inst.LifecycleState)
		b.WriteString(line + "\n")
	}
	return b.String()
}

// healthDot renders a coloured status indicator for an ASG instance health string.
func healthDot(status string) string {
	switch status {
	case "Healthy":
		return asgHealthyStyle.Render("●")
	case "Unhealthy":
		return asgUnhealthyStyle.Render("●")
	default:
		return asgPendingStyle.Render("◌")
	}
}

func (d *ASGDetail) renderTargetGroups(spinner string, width int) string {
	var b strings.Builder
	b.WriteString(detailSectionStyle.Width(width).Render("Target groups") + "\n")

	switch d.TGState {
	case BlockLoading:
		b.WriteString(detailMutedStyle.Render(spinner+" Chargement des target groups…") + "\n")
	case BlockError:
		b.WriteString(detailErrStyle.Render("Erreur: "+errText(d.TGErr)) + "\n")
	case BlockLoaded:
		if len(d.TGs) == 0 {
			b.WriteString(detailMutedStyle.Render("Aucun target group.") + "\n")
		}
		for _, tg := range d.TGs {
			b.WriteString(renderTGSummary(tg) + "\n")
		}
	}
	return b.String()
}

// renderTGSummary renders one target group line with a health summary.
func renderTGSummary(tg awsclient.TargetGroup) string {
	healthy, total := tg.HealthyCount(), tg.TotalCount()
	summary := fmt.Sprintf("%d/%d saines", healthy, total)
	style := asgHealthyStyle
	if healthy < total {
		style = asgUnhealthyStyle
	}
	if total == 0 {
		style = asgPendingStyle
	}
	proto := fmt.Sprintf("%s:%d", tg.Protocol, tg.Port)
	return fmt.Sprintf("%s %s  %s  %s",
		style.Render("●"),
		pad(tg.Name, 30),
		detailMutedStyle.Render(pad(proto, 12)),
		style.Render(summary))
}

func (d *ASGDetail) renderActivities(spinner string, width int) string {
	var b strings.Builder
	b.WriteString(detailSectionStyle.Width(width).Render("Activités récentes") + "\n")

	switch d.ActState {
	case BlockLoading:
		b.WriteString(detailMutedStyle.Render(spinner+" Chargement des activités…") + "\n")
	case BlockError:
		b.WriteString(detailErrStyle.Render("Erreur: "+errText(d.ActErr)) + "\n")
	case BlockLoaded:
		if len(d.Acts) == 0 {
			b.WriteString(detailMutedStyle.Render("Aucune activité récente.") + "\n")
		}
		for _, a := range d.Acts {
			status := a.StatusCode
			st := detailMutedStyle
			switch status {
			case "Successful":
				st = asgHealthyStyle
			case "Failed", "Cancelled":
				st = asgUnhealthyStyle
			case "InProgress", "PreInService", "WaitingForInstanceId":
				st = asgPendingStyle
			}
			b.WriteString(st.Render("• "+status) + " " +
				detailMutedStyle.Render(shortTime(a.StartTime)) + "  " +
				truncateRunes(a.Description, width-24) + "\n")
		}
	}
	return b.String()
}

// shortTime trims an RFC3339 timestamp to "MM-DD HH:MM" for compact display.
func shortTime(ts string) string {
	// 2006-01-02T15:04:05Z07:00 -> keep "01-02 15:04"
	if len(ts) < 16 {
		return ts
	}
	return ts[5:10] + " " + ts[11:16]
}

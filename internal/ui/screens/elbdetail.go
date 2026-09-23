package screens

import (
	"fmt"
	"strings"

	"github.com/VeugDamien/aws-tui/internal/awsclient"
)

// ELBDetail holds the state of the load balancer detail page.
type ELBDetail struct {
	LB awsclient.LoadBalancer

	// Listeners (async).
	ListenersState blockState
	ListenersErr   error
	Listeners      []awsclient.Listener

	// Target groups with health (async).
	TGState blockState
	TGErr   error
	TGs     []awsclient.TargetGroup

	scroll int
	width  int
	height int
}

// NewELBDetail builds the detail page with both async blocks loading.
func NewELBDetail(lb awsclient.LoadBalancer, width, height int) ELBDetail {
	return ELBDetail{
		LB:             lb,
		ListenersState: BlockLoading,
		TGState:        BlockLoading,
		width:          width,
		height:         height,
	}
}

// Resize updates the render area.
func (d *ELBDetail) Resize(width, height int) {
	d.width = width
	d.height = height
}

// ScrollUp / ScrollDown move the viewport.
func (d *ELBDetail) ScrollUp(n int) {
	d.scroll -= n
	if d.scroll < 0 {
		d.scroll = 0
	}
}

func (d *ELBDetail) ScrollDown(n int) { d.scroll += n }

// View renders the load balancer detail page as a single scrollable column.
func (d *ELBDetail) View(spinner string) string {
	lb := d.LB
	title := detailTitleStyle.Render(" "+lb.Name+" ") + "  " +
		detailMutedStyle.Render(fmt.Sprintf("%s · %s", lb.Type, lb.Scheme))

	width := d.width
	if width < 40 {
		width = 40
	}

	var b strings.Builder
	b.WriteString(d.renderGeneral(width))
	b.WriteString("\n")
	b.WriteString(d.renderListeners(spinner, width))
	b.WriteString("\n")
	b.WriteString(d.renderTargetGroups(spinner, width))

	body, total, shown := d.applyScroll(b.String())
	out := title + "\n\n" + body
	if total > shown {
		out += "\n" + detailMutedStyle.Render(fmt.Sprintf("↑/↓ défiler (%d/%d)", d.scroll+1, total))
	}
	return out
}

func (d *ELBDetail) applyScroll(s string) (string, int, int) {
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

func (d *ELBDetail) renderGeneral(width int) string {
	lb := d.LB
	var b strings.Builder
	b.WriteString(detailSectionStyle.Width(width).Render("Informations générales") + "\n")
	b.WriteString(kv("Type", lb.Type))
	b.WriteString(kv("Schéma", lb.Scheme))
	b.WriteString(kv("État", lb.State))
	b.WriteString(kv("DNS", lb.DNSName))
	b.WriteString(kv("VPC", lb.VPCID))
	b.WriteString(kv("Zones", strings.Join(lb.AZs, ", ")))
	b.WriteString(kv("Créé le", lb.CreatedTime))
	return b.String()
}

func (d *ELBDetail) renderListeners(spinner string, width int) string {
	var b strings.Builder
	b.WriteString(detailSectionStyle.Width(width).Render("Listeners") + "\n")

	switch d.ListenersState {
	case BlockLoading:
		b.WriteString(detailMutedStyle.Render(spinner+" Chargement des listeners…") + "\n")
	case BlockError:
		b.WriteString(detailErrStyle.Render("Erreur: "+errText(d.ListenersErr)) + "\n")
	case BlockLoaded:
		if len(d.Listeners) == 0 {
			b.WriteString(detailMutedStyle.Render("Aucun listener.") + "\n")
		}
		for _, l := range d.Listeners {
			b.WriteString(detailOKStyle.Render(fmt.Sprintf("● %s:%d", l.Protocol, l.Port)) + "\n")
		}
	}
	return b.String()
}

func (d *ELBDetail) renderTargetGroups(spinner string, width int) string {
	var b strings.Builder
	b.WriteString(detailSectionStyle.Width(width).Render("Target groups & santé des cibles") + "\n")

	switch d.TGState {
	case BlockLoading:
		b.WriteString(detailMutedStyle.Render(spinner+" Chargement des cibles…") + "\n")
	case BlockError:
		b.WriteString(detailErrStyle.Render("Erreur: "+errText(d.TGErr)) + "\n")
	case BlockLoaded:
		if len(d.TGs) == 0 {
			b.WriteString(detailMutedStyle.Render("Aucun target group.") + "\n")
		}
		for _, tg := range d.TGs {
			b.WriteString(renderTGSummary(tg) + "\n")
			for _, t := range tg.Targets {
				b.WriteString("    " + renderTargetLine(t) + "\n")
			}
		}
	}
	return b.String()
}

// renderTargetLine renders one target's health inside a target group.
func renderTargetLine(t awsclient.TargetHealth) string {
	var style = asgPendingStyle
	switch t.State {
	case "healthy":
		style = asgHealthyStyle
	case "unhealthy", "unused":
		style = asgUnhealthyStyle
	}
	id := t.ID
	if t.Port > 0 {
		id = fmt.Sprintf("%s:%d", t.ID, t.Port)
	}
	line := style.Render("○ ") + pad(id, 28) + " " + style.Render(pad(t.State, 12))
	if t.Reason != "" {
		line += " " + detailMutedStyle.Render(t.Reason)
	}
	return line
}

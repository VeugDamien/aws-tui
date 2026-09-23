package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// View renders the current screen, filling the whole terminal with the footer
// and status bar anchored to the bottom.
func (m AppModel) View() string {
	if m.width == 0 || m.height == 0 {
		return "Initialisation…"
	}

	var body string
	switch m.screen {
	case ScreenProfiles:
		body = m.viewProfiles()
	case ScreenActions:
		body = m.viewActions()
	case ScreenRegions:
		body = m.viewRegions()
	case ScreenEC2:
		body = m.viewEC2()
	case ScreenEC2Detail:
		body = m.ec2Detail.View(m.spinner.View())
	case ScreenASG:
		body = m.viewASG()
	case ScreenASGDetail:
		body = m.asgDetail.View(m.spinner.View())
	case ScreenELB:
		body = m.viewELB()
	case ScreenELBDetail:
		body = m.elbDetail.View(m.spinner.View())
	case ScreenPortForwardForm:
		body = m.viewPFForm()
	case ScreenTunnels:
		body = m.viewTunnels()
	}

	// Top block: header + optional config bar + body.
	top := []string{m.header()}
	if m.isActionScreen() {
		top = append(top, m.configBar())
	}
	top = append(top, "", body)
	if m.loading {
		top = append(top, "", fmt.Sprintf("%s %s", m.spinner.View(), m.loadingMsg))
	}
	if m.err != nil {
		top = append(top, "", errorBoxStyle.Render("Erreur: "+m.err.Error()))
	}

	// Bottom block: status bar (if any) + footer, anchored to the bottom edge.
	var bottom []string
	if m.statusMsg != "" {
		bottom = append(bottom, m.statusBar())
	}
	bottom = append(bottom, m.footer())

	// Inner area available after the doc box margins (1 row, 2 cols each side).
	const marginV, marginH = 1, 2
	innerW := m.width - 2*marginH
	innerH := m.height - 2*marginV
	if innerW < 1 {
		innerW = 1
	}
	if innerH < 1 {
		innerH = 1
	}

	topStr := strings.Join(top, "\n")
	bottomStr := strings.Join(bottom, "\n")

	// Spacer fills the gap so the bottom block sits on the last line.
	gap := innerH - lipgloss.Height(topStr) - lipgloss.Height(bottomStr)
	if gap < 1 {
		gap = 1
	}
	spacer := strings.Repeat("\n", gap)

	content := topStr + spacer + bottomStr

	// Force the content to exactly fill the inner area.
	box := lipgloss.NewStyle().
		Width(innerW).
		Height(innerH).
		MaxHeight(innerH).
		Margin(marginV, marginH).
		Render(content)

	return box
}

// statusBar renders the transient status message in a visually isolated bar,
// separated from the content above by a horizontal rule.
func (m AppModel) statusBar() string {
	width := m.contentWidth()
	rule := statusRuleStyle.Render(strings.Repeat("─", width))
	msg := statusLineStyle.Render("• " + m.statusMsg)
	return rule + "\n" + msg
}

// header renders the title line plus the active-tunnel counter.
func (m AppModel) header() string {
	title := titleStyle.Render(" aws-tui ")
	right := ""
	if n := m.tunnels.activeCount(); n > 0 {
		right = "  " + warnStyle.Render(fmt.Sprintf("⇄ %d tunnel(s)", n))
	}
	return lipgloss.JoinHorizontal(lipgloss.Center, title, right)
}

// configBar renders the persistent configuration bar (profile, region, account).
// It is shown on action screens so the active context is always visible and the
// p/g/L shortcuts make sense.
func (m AppModel) configBar() string {
	profile := m.activeProfile
	if profile == "" {
		profile = "—"
	}
	account := warnStyle.Render("non connecté")
	if m.identity != nil {
		if m.accountAlias != "" {
			// Friendly alias first, account number in parentheses.
			account = identityStyle.Render(m.accountAlias) +
				subtitleStyle.Render(" ("+m.identity.Account+")")
		} else {
			account = identityStyle.Render(m.identity.Account)
		}
	}

	segs := []string{
		configKeyStyle.Render("[p]") + " profil: " + configValStyle.Render(profile),
		configKeyStyle.Render("[g]") + " région: " + configValStyle.Render(m.activeRegion),
		"compte: " + account,
	}
	bar := strings.Join(segs, "   │   ")
	return configBarStyle.Render(bar)
}

func (m AppModel) footer() string {
	if m.help.ShowAll {
		return m.help.View(m.keys)
	}

	var hints string
	switch m.screen {
	case ScreenProfiles:
		hints = "↑/↓ naviguer · / filtrer · y copier nom · enter sélectionner · esc annuler · q quitter"
	case ScreenActions:
		hints = "↑/↓ naviguer · enter valider   │   p profil · g région · L logout   │   ? aide · q quitter"
	case ScreenRegions:
		hints = "↑/↓ naviguer · / filtrer · enter choisir · esc annuler"
	case ScreenEC2:
		hints = "↑/↓ · ←/→ défiler · enter détails · / filtrer · y copier ID · r rafraîchir · s shell · f port-forward · t tunnels · esc retour"
	case ScreenASG:
		hints = "↑/↓ naviguer · enter détails · / filtrer · y copier nom · r rafraîchir · esc retour"
	case ScreenASGDetail:
		hints = "↑/↓ défiler · y copier nom · r rafraîchir · esc retour"
	case ScreenELB:
		hints = "↑/↓ · ←/→ défiler · enter détails · / filtrer · y copier DNS · r rafraîchir · esc retour"
	case ScreenELBDetail:
		hints = "↑/↓ défiler · y copier DNS · r rafraîchir · esc retour"
	case ScreenEC2Detail:
		if m.ec2Detail.CopyMenuOpen() {
			hints = "↑/↓ choisir · enter copier · esc fermer"
		} else {
			hints = "↑/↓ défiler · y copier ID · Y copier… · s session · f port-forward · m métriques · r rafraîchir · esc retour"
		}
	case ScreenPortForwardForm:
		hints = "↑/↓ champ · tab hôte distant · enter lancer · esc annuler"
	case ScreenTunnels:
		hints = "↑/↓ naviguer · x arrêter · X tout arrêter · esc retour"
	}
	return helpStyle.Render(hints)
}

func (m AppModel) viewProfiles() string {
	if len(m.profiles) == 0 && !m.loading {
		return warnStyle.Render("Aucun profil trouvé dans ~/.aws/config.")
	}
	return m.profileList.View()
}

// viewActions renders the actions hub: only functional actions. Configuration
// (profile/region/account) lives in the persistent config bar above.
func (m AppModel) viewActions() string {
	var b strings.Builder

	b.WriteString(subtitleStyle.Render("Actions") + "\n\n")

	for i, it := range m.actionItems() {
		pointer := "  "
		label := menuKeyStyle.Render(it.key) + "  " + it.label
		if i == m.actionCursor {
			pointer = menuKeyStyle.Render("▶ ")
			label = menuSelStyle.Render(it.key+"  "+it.label)
		}
		b.WriteString(pointer + menuItemStyle.Render(label) + "\n")
	}

	if m.identity != nil {
		b.WriteString("\n" + labelStyle.Render("Identité") + subtitleStyle.Render(m.identity.Arn))
	}
	return b.String()
}

func (m AppModel) viewRegions() string {
	return m.regionList.View()
}

func (m AppModel) viewEC2() string {
	var b strings.Builder

	header := fmt.Sprintf("Instances EC2 — %d/%d", m.ec2.Count(), m.ec2.TotalCount())
	b.WriteString(subtitleStyle.Render(header) + "\n")

	if m.ec2.Filtering || m.ec2.Filter != "" {
		cursor := ""
		if m.ec2.Filtering {
			cursor = "█"
		}
		b.WriteString(fmt.Sprintf("Filtre: %s%s\n", m.ec2.Filter, cursor))
	}

	b.WriteString(m.ec2.View())

	if m.ec2.TotalCount() == 0 && !m.loading {
		b.WriteString("\n" + warnStyle.Render("Aucune instance (ou accès refusé). 'r' pour réessayer."))
	}
	return b.String()
}

func (m AppModel) viewASG() string {
	var b strings.Builder

	header := fmt.Sprintf("Auto Scaling Groups — %d/%d", m.asg.Count(), m.asg.TotalCount())
	b.WriteString(subtitleStyle.Render(header) + "\n")

	if m.asg.Filtering || m.asg.Filter != "" {
		cursor := ""
		if m.asg.Filtering {
			cursor = "█"
		}
		b.WriteString(fmt.Sprintf("Filtre: %s%s\n", m.asg.Filter, cursor))
	}

	b.WriteString(m.asg.View())

	if m.asg.TotalCount() == 0 && !m.loading {
		b.WriteString("\n" + warnStyle.Render("Aucun Auto Scaling Group (ou accès refusé). 'r' pour réessayer."))
	}
	return b.String()
}

func (m AppModel) viewELB() string {
	var b strings.Builder

	header := fmt.Sprintf("Load Balancers — %d/%d", m.elb.Count(), m.elb.TotalCount())
	b.WriteString(subtitleStyle.Render(header) + "\n")

	if m.elb.Filtering || m.elb.Filter != "" {
		cursor := ""
		if m.elb.Filtering {
			cursor = "█"
		}
		b.WriteString(fmt.Sprintf("Filtre: %s%s\n", m.elb.Filter, cursor))
	}

	b.WriteString(m.elb.View())

	if m.elb.TotalCount() == 0 && !m.loading {
		b.WriteString("\n" + warnStyle.Render("Aucun Load Balancer (ou accès refusé). 'r' pour réessayer."))
	}
	return b.String()
}

func (m AppModel) viewPFForm() string {
	var b strings.Builder

	mode := "vers un port de l'instance"
	if m.pfForm.RemoteHost {
		mode = "vers un hôte distant (bastion)"
	}
	tgt := m.pfForm.Target
	if m.pfForm.TargetName != "" {
		tgt = fmt.Sprintf("%s (%s)", m.pfForm.TargetName, m.pfForm.Target)
	}

	b.WriteString(subtitleStyle.Render("Port-forward "+mode) + "\n")
	b.WriteString(labelStyle.Render("Cible") + tgt + "\n\n")
	b.WriteString(m.pfForm.View())
	return panelStyle.Render(b.String())
}

func (m AppModel) viewTunnels() string {
	tunnels := m.tunnels.list()
	if len(tunnels) == 0 {
		return warnStyle.Render("Aucun tunnel. Depuis la liste EC2, sélectionnez une instance et appuyez sur 'f'.")
	}

	var b strings.Builder
	b.WriteString(subtitleStyle.Render(fmt.Sprintf("Tunnels (%d)", len(tunnels))) + "\n\n")

	for i, t := range tunnels {
		cursor := "  "
		if i == m.tunnelCursor {
			cursor = menuKeyStyle.Render("▶ ")
		}

		var state string
		switch t.State {
		case tunnelActive:
			state = identityStyle.Render("● actif   ")
		case tunnelStarting:
			state = subtitleStyle.Render("◌ démarrage")
		case tunnelStopped:
			state = subtitleStyle.Render("○ arrêté  ")
		case tunnelFailed:
			state = errorStyle.Render("✕ erreur  ")
		}

		line := fmt.Sprintf("%s#%-2d %s  %s", cursor, t.ID, state, t.Summary())
		b.WriteString(line + "\n")

		if t.State == tunnelFailed && t.Err != nil {
			b.WriteString("      " + errorStyle.Render(t.Err.Error()) + "\n")
		} else if i == m.tunnelCursor {
			if out := strings.TrimSpace(t.Output()); out != "" {
				b.WriteString("      " + statusBarStyle.Render(lastLine(out)) + "\n")
			}
		}
	}

	return panelStyle.Render(b.String())
}

// lastLine returns the last non-empty line of s (for compact status display).
func lastLine(s string) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.TrimSpace(lines[i]) != "" {
			return strings.TrimSpace(lines[i])
		}
	}
	return ""
}

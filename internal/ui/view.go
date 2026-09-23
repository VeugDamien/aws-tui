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
		return "Initializing…"
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
		top = append(top, "", errorBoxStyle.Render("Error: "+m.err.Error()))
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
	account := warnStyle.Render("not connected")
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
		configKeyStyle.Render("[p]") + " profile: " + configValStyle.Render(profile),
		configKeyStyle.Render("[g]") + " region: " + configValStyle.Render(m.activeRegion),
		"account: " + account,
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
		hints = "↑/↓ navigate · / filter · y copy name · enter select · esc cancel · q quit"
	case ScreenActions:
		hints = "↑/↓ navigate · enter confirm   │   p profile · g region · L logout   │   ? help · q quit"
	case ScreenRegions:
		hints = "↑/↓ navigate · / filter · enter select · esc cancel"
	case ScreenEC2:
		hints = "↑/↓ · ←/→ scroll · enter details · / filter · y copy ID · r refresh · s shell · f port-forward · t tunnels · esc back"
	case ScreenASG:
		hints = "↑/↓ navigate · enter details · / filter · y copy name · r refresh · esc back"
	case ScreenASGDetail:
		hints = "↑/↓ scroll · y copy name · r refresh · esc back"
	case ScreenELB:
		hints = "↑/↓ · ←/→ scroll · enter details · / filter · y copy DNS · r refresh · esc back"
	case ScreenELBDetail:
		hints = "↑/↓ scroll · y copy DNS · r refresh · esc back"
	case ScreenEC2Detail:
		if m.ec2Detail.CopyMenuOpen() {
			hints = "↑/↓ select · enter copy · esc close"
		} else {
			hints = "↑/↓ scroll · y copy ID · Y copy… · s session · f port-forward · m metrics · r refresh · esc back"
		}
	case ScreenPortForwardForm:
		hints = "↑/↓ field · tab remote host · enter start · esc cancel"
	case ScreenTunnels:
		hints = "↑/↓ navigate · x stop · X stop all · r restart · c clear stopped · esc back"
	}
	return helpStyle.Render(hints)
}

func (m AppModel) viewProfiles() string {
	if len(m.profiles) == 0 && !m.loading {
		return warnStyle.Render("No profile found in ~/.aws/config.")
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
		b.WriteString("\n" + labelStyle.Render("Identity") + subtitleStyle.Render(m.identity.Arn))
	}
	return b.String()
}

func (m AppModel) viewRegions() string {
	return m.regionList.View()
}

func (m AppModel) viewEC2() string {
	var b strings.Builder

	header := fmt.Sprintf("EC2 instances — %d/%d", m.ec2.Count(), m.ec2.TotalCount())
	b.WriteString(subtitleStyle.Render(header) + "\n")

	if m.ec2.Filtering || m.ec2.Filter != "" {
		cursor := ""
		if m.ec2.Filtering {
			cursor = "█"
		}
		b.WriteString(fmt.Sprintf("Filter: %s%s\n", m.ec2.Filter, cursor))
	}

	b.WriteString(m.ec2.View())

	if m.ec2.TotalCount() == 0 && !m.loading {
		b.WriteString("\n" + warnStyle.Render("No instance (or access denied). Press 'r' to retry."))
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
		b.WriteString(fmt.Sprintf("Filter: %s%s\n", m.asg.Filter, cursor))
	}

	b.WriteString(m.asg.View())

	if m.asg.TotalCount() == 0 && !m.loading {
		b.WriteString("\n" + warnStyle.Render("No Auto Scaling Group (or access denied). Press 'r' to retry."))
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
		b.WriteString(fmt.Sprintf("Filter: %s%s\n", m.elb.Filter, cursor))
	}

	b.WriteString(m.elb.View())

	if m.elb.TotalCount() == 0 && !m.loading {
		b.WriteString("\n" + warnStyle.Render("No Load Balancer (or access denied). Press 'r' to retry."))
	}
	return b.String()
}

func (m AppModel) viewPFForm() string {
	var b strings.Builder

	mode := "to an instance port"
	if m.pfForm.RemoteHost {
		mode = "to a remote host (bastion)"
	}
	tgt := m.pfForm.Target
	if m.pfForm.TargetName != "" {
		tgt = fmt.Sprintf("%s (%s)", m.pfForm.TargetName, m.pfForm.Target)
	}

	b.WriteString(subtitleStyle.Render("Port-forward "+mode) + "\n")
	b.WriteString(labelStyle.Render("Target") + tgt + "\n\n")
	b.WriteString(m.pfForm.View())
	return panelStyle.Render(b.String())
}

func (m AppModel) viewTunnels() string {
	tunnels := m.tunnels.list()
	if len(tunnels) == 0 {
		return warnStyle.Render("No tunnel. From the EC2 list, select an instance and press 'f'.")
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
			state = identityStyle.Render("● active  ")
		case tunnelStarting:
			state = subtitleStyle.Render("◌ starting ")
		case tunnelStopped:
			state = subtitleStyle.Render("○ stopped ")
		case tunnelFailed:
			state = errorStyle.Render("✕ error   ")
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

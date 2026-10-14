package ui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/VeugDamien/aws-tui/internal/auth"
	"github.com/VeugDamien/aws-tui/internal/awsclient"
	"github.com/VeugDamien/aws-tui/internal/awsconfig"
	"github.com/VeugDamien/aws-tui/internal/state"
	"github.com/VeugDamien/aws-tui/internal/ui/screens"
)

// statusTimeout is how long a transient status message stays on screen.
const statusTimeout = 10 * time.Second

// clearStatusMsg requests clearing the status line if it hasn't changed since.
type clearStatusMsg struct{ seq int }

// setStatus sets a transient status message and returns a command that clears it
// after statusTimeout (unless a newer status replaces it first). It mutates the
// receiver, so call it on a pointer or reassign the returned model.
func (m *AppModel) setStatus(text string) tea.Cmd {
	m.statusMsg = text
	m.statusSeq++
	seq := m.statusSeq
	return tea.Tick(statusTimeout, func(time.Time) tea.Msg {
		return clearStatusMsg{seq: seq}
	})
}

// Update is the main message router.
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.handleResize(msg)

	case profilesLoadedMsg:
		m.loading = false
		m.loadingMsg = ""
		m.profiles = msg.profiles
		m.profileList = screens.NewProfileList(msg.profiles, m.contentWidth(), m.contentHeight())
		m.err = nil
		// On startup, restore the last used profile if it still exists.
		if !m.startupDone {
			m.startupDone = true
			if p, ok := m.lastProfile(); ok {
				// Show the loading over the Actions context, not the profile list.
				m.screen = ScreenActions
				// Automatic restore: a required login will be confirmed first.
				return m.connectToProfile(p, false)
			}
			// Nothing memorised (or profile gone): land on Actions, not connected.
			m.screen = ScreenActions
		}
		return m, nil

	case whoamiMsg:
		m.loading = false
		m.loadingMsg = ""
		m.identity = msg.identity
		m.accountAlias = "" // reset; resolved asynchronously below
		m.awsCfg = msg.cfg
		m.err = nil
		clearCmd := m.setStatus("Connected.")
		// Resolve the account alias in the background (non-fatal if denied).
		aliasCmd := accountAliasCmd(msg.cfg, msg.identity.Account)
		// Persist the now-connected profile as the last used one.
		saveCmd := saveProfileCmd(m.activeProfile)
		// Land on the action we came from (EC2/Tunnels) or the actions hub.
		dest := m.returnOrActions()
		if dest == ScreenEC2 {
			// Rebuild a fresh table and reload instances for the new context.
			m.ec2 = screens.NewEC2Table(m.contentWidth(), m.ec2ContentHeight())
			m.screen = ScreenEC2
			m.loading = true
			m.loadingMsg = "Loading EC2 instances…"
			return m, tea.Batch(listInstancesCmd(&m), aliasCmd, saveCmd, clearCmd, m.spinner.Tick)
		}
		m.screen = dest
		return m, tea.Batch(aliasCmd, saveCmd, clearCmd)

	case accountAliasMsg:
		// Ignore stale responses from a previously selected profile/account.
		if m.identity != nil && msg.account == m.identity.Account {
			m.accountAlias = msg.alias
		}
		return m, nil

	case authRequiredMsg:
		// Resolve which profile actually has an interactive login: the selected
		// one, or — for a credential_process/derived profile — its loggable
		// parent. If nothing in the chain can be logged in (e.g. standalone static
		// keys), surface the underlying error instead of attempting a login.
		target, ok := auth.LoginTarget(msg.profile, m.profileByName)
		if !ok {
			m.loading = false
			m.loadingMsg = ""
			m.err = fmt.Errorf("profile %q cannot be authenticated interactively (no SSO/login in its chain)", msg.profile.Name)
			return m, nil
		}
		// A manual profile selection logs in immediately; any other context
		// (startup restore, token expiry during an action) first shows a
		// confirmation screen naming the profile so the user knows what they are
		// authenticating.
		if m.autoLogin {
			m.loading = true
			m.loadingMsg = fmt.Sprintf("Authenticating profile %q…", target.Name)
			return m, loginCmd(target)
		}
		m.loading = false
		m.loadingMsg = ""
		m.loginProfile = target
		m.loginConfirmReturn = m.screen
		m.screen = ScreenLoginConfirm
		return m, nil

	case loginDoneMsg:
		if msg.err != nil {
			m.loading = false
			m.err = fmt.Errorf("login failed: %w", msg.err)
			return m, nil
		}
		// Retry identity after a successful login/logout.
		m.loading = true
		m.loadingMsg = "Verifying identity…"
		return m, whoamiCmd(m.currentProfile(), m.activeRegion)

	case instancesLoadedMsg:
		m.loading = false
		m.loadingMsg = ""
		m.instances = msg.instances
		m.ec2.SetInstances(msg.instances)
		m.err = nil
		m.statusMsg = fmt.Sprintf("%d instance(s).", len(msg.instances))
		return m, nil

	case asgsLoadedMsg:
		m.loading = false
		m.loadingMsg = ""
		m.asgs = msg.groups
		m.asg.SetGroups(msg.groups)
		m.err = nil
		m.statusMsg = fmt.Sprintf("%d Auto Scaling Group(s).", len(msg.groups))
		return m, nil

	case lbsLoadedMsg:
		m.loading = false
		m.loadingMsg = ""
		m.lbs = msg.lbs
		m.elb.SetLoadBalancers(msg.lbs)
		m.err = nil
		m.statusMsg = fmt.Sprintf("%d Load Balancer(s).", len(msg.lbs))
		return m, nil

	case asgDetailMsg:
		// Ignore if the detail page moved to another group.
		if m.screen == ScreenASGDetail && msg.groupName == m.asgDetail.Group.Name {
			if msg.targetsErr != nil {
				m.asgDetail.TGState = screens.BlockError
				m.asgDetail.TGErr = msg.targetsErr
			} else {
				m.asgDetail.TGState = screens.BlockLoaded
				m.asgDetail.TGs = msg.targets
			}
			if msg.actErr != nil {
				m.asgDetail.ActState = screens.BlockError
				m.asgDetail.ActErr = msg.actErr
			} else {
				m.asgDetail.ActState = screens.BlockLoaded
				m.asgDetail.Acts = msg.activities
			}
		}
		return m, nil

	case elbListenersMsg:
		if m.screen == ScreenELBDetail && msg.lbARN == m.elbDetail.LB.ARN {
			if msg.err != nil {
				m.elbDetail.ListenersState = screens.BlockError
				m.elbDetail.ListenersErr = msg.err
			} else {
				m.elbDetail.ListenersState = screens.BlockLoaded
				m.elbDetail.Listeners = msg.listeners
			}
		}
		return m, nil

	case elbTargetsMsg:
		if m.screen == ScreenELBDetail && msg.lbARN == m.elbDetail.LB.ARN {
			if msg.err != nil {
				m.elbDetail.TGState = screens.BlockError
				m.elbDetail.TGErr = msg.err
			} else {
				m.elbDetail.TGState = screens.BlockLoaded
				m.elbDetail.TGs = msg.tgs
			}
		}
		return m, nil

	case sgLoadedMsg:
		// Ignore if the detail page moved to another instance.
		if m.screen == ScreenEC2Detail && msg.instanceID == m.ec2Detail.Instance.InstanceID {
			if msg.err != nil {
				m.ec2Detail.SGState = screens.BlockError
				m.ec2Detail.SGErr = msg.err
			} else {
				m.ec2Detail.SGState = screens.BlockLoaded
				m.ec2Detail.SGs = msg.groups
			}
		}
		return m, nil

	case metricsLoadedMsg:
		if m.screen == ScreenEC2Detail && msg.instanceID == m.ec2Detail.Instance.InstanceID {
			if msg.err != nil {
				m.ec2Detail.MetricsState = screens.BlockError
				m.ec2Detail.MetricsErr = msg.err
			} else {
				m.ec2Detail.MetricsState = screens.BlockLoaded
				m.ec2Detail.Metrics = msg.metrics
			}
		}
		return m, nil

	case shellFinishedMsg:
		if msg.err != nil {
			m.err = fmt.Errorf("SSM session ended with error: %w", msg.err)
			return m, nil
		}
		return m, m.setStatus("SSM session ended.")

	case tunnelStartedMsg:
		m.loading = false
		m.loadingMsg = ""
		m = m.returnFromPF()
		clearCmd := m.setStatus(fmt.Sprintf("Tunnel #%d started: %s", msg.id, msg.summary))
		// Watch this tunnel for exit without blocking the UI.
		return m, tea.Batch(waitTunnelCmd(m.tunnels, msg.id), clearCmd)

	case tunnelExitedMsg:
		var cmd tea.Cmd
		if msg.err != nil && !m.tunnels.wasStopRequested(msg.id) {
			// Unexpected exit (process died on its own): mark as failed.
			m.tunnels.setState(msg.id, tunnelFailed, msg.err)
			cmd = m.setStatus(fmt.Sprintf("Tunnel #%d failed.", msg.id))
		} else {
			// Clean exit or user-requested stop (signal: terminated).
			m.tunnels.setState(msg.id, tunnelStopped, nil)
			cmd = m.setStatus(fmt.Sprintf("Tunnel #%d stopped.", msg.id))
		}
		m.clampTunnelCursor()
		return m, cmd

	case clearStatusMsg:
		// Only clear if no newer status has replaced this one.
		if msg.seq == m.statusSeq {
			m.statusMsg = ""
		}
		return m, nil

	case updateAvailableMsg:
		// Record the newer version so the header can show a discreet hint. This is
		// purely informational and never interrupts the user.
		m.updateVersion = msg.latest
		return m, nil

	case clipboardMsg:
		if msg.err != nil {
			m.err = fmt.Errorf("copy failed: %w", msg.err)
			return m, nil
		}
		m.err = nil
		return m, m.setStatus(fmt.Sprintf("Copied (%s): %s", msg.label, truncate(msg.value, 60)))

	case errMsg:
		m.loading = false
		m.loadingMsg = ""
		m.err = msg.err
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	// Forward to spinner while loading (global) or while the detail page has a
	// pending async block.
	if m.loading || m.detailLoading() {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

// detailLoading reports whether the detail page is open with a block still loading.
func (m AppModel) detailLoading() bool {
	switch m.screen {
	case ScreenEC2Detail:
		return m.ec2Detail.SGState == screens.BlockLoading || m.ec2Detail.MetricsState == screens.BlockLoading
	case ScreenASGDetail:
		return m.asgDetail.TGState == screens.BlockLoading || m.asgDetail.ActState == screens.BlockLoading
	case ScreenELBDetail:
		return m.elbDetail.ListenersState == screens.BlockLoading || m.elbDetail.TGState == screens.BlockLoading
	}
	return false
}

func (m AppModel) handleResize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height
	m.profileList.SetSize(m.contentWidth(), m.contentHeight())
	if m.regionList.Items() != nil {
		m.regionList.SetSize(m.contentWidth(), m.contentHeight())
	}
	if m.screen == ScreenEC2 {
		m.ec2.Resize(m.contentWidth(), m.ec2ContentHeight())
		m.ec2.Focus()
	}
	if m.screen == ScreenEC2Detail {
		m.ec2Detail.Resize(m.contentWidth(), m.contentHeight()-2)
	}
	if m.screen == ScreenASG {
		m.asg.Resize(m.contentWidth(), m.ec2ContentHeight())
	}
	if m.screen == ScreenASGDetail {
		m.asgDetail.Resize(m.contentWidth(), m.contentHeight()-2)
	}
	if m.screen == ScreenELB {
		m.elb.Resize(m.contentWidth(), m.ec2ContentHeight())
	}
	if m.screen == ScreenELBDetail {
		m.elbDetail.Resize(m.contentWidth(), m.contentHeight()-2)
	}
	return m, nil
}

func (m AppModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Global quit, except while typing in a filter or form field.
	if key.Matches(msg, m.keys.Quit) && !m.isTyping() {
		m.tunnels.stopAll()
		return m, tea.Quit
	}
	if key.Matches(msg, m.keys.Help) && !m.isTyping() {
		m.help.ShowAll = !m.help.ShowAll
		return m, nil
	}

	// Config shortcuts available from any action screen (persistent config bar).
	if m.isActionScreen() && !m.isTyping() {
		switch {
		case key.Matches(msg, m.keys.Profiles):
			m.returnScreen = m.screen
			m.screen = ScreenProfiles
			return m, nil
		case key.Matches(msg, m.keys.Regions):
			m.returnScreen = m.screen
			m.regionList = screens.NewRegionList(awsclient.Regions(), m.contentWidth(), m.contentHeight())
			m.screen = ScreenRegions
			return m, nil
		case key.Matches(msg, m.keys.Logout):
			p := m.currentProfile()
			m.loading = true
			m.loadingMsg = "Logging out…"
			m.identity = nil
			m.accountAlias = ""
			m.instances = nil
			m.ec2.SetInstances(nil)
			m.asgs = nil
			m.lbs = nil
			m.screen = ScreenProfiles
			return m, tea.Batch(logoutCmd(p), m.spinner.Tick)
		}
	}

	switch m.screen {
	case ScreenProfiles:
		return m.updateProfiles(msg)
	case ScreenActions:
		return m.updateActions(msg)
	case ScreenRegions:
		return m.updateRegions(msg)
	case ScreenEC2:
		return m.updateEC2(msg)
	case ScreenEC2Detail:
		return m.updateEC2Detail(msg)
	case ScreenASG:
		return m.updateASG(msg)
	case ScreenASGDetail:
		return m.updateASGDetail(msg)
	case ScreenELB:
		return m.updateELB(msg)
	case ScreenELBDetail:
		return m.updateELBDetail(msg)
	case ScreenLoginConfirm:
		return m.updateLoginConfirm(msg)
	case ScreenPortForwardForm:
		return m.updatePFForm(msg)
	case ScreenTunnels:
		return m.updateTunnels(msg)
	}
	return m, nil
}

// isActionScreen reports whether the current screen is a functional action screen
// (as opposed to a config overlay), where the persistent config shortcuts apply.
func (m AppModel) isActionScreen() bool {
	switch m.screen {
	case ScreenActions, ScreenEC2, ScreenASG, ScreenELB, ScreenTunnels, ScreenPortForwardForm:
		return true
	}
	return false
}

// isTyping reports whether a text-entry widget currently has focus, so global
// single-letter shortcuts must be suppressed.
func (m AppModel) isTyping() bool {
	switch m.screen {
	case ScreenProfiles:
		return m.profileList.Filtering()
	case ScreenRegions:
		return m.regionList.FilterState() == list.Filtering
	case ScreenEC2:
		return m.ec2.Filtering
	case ScreenASG:
		return m.asg.Filtering
	case ScreenELB:
		return m.elb.Filtering
	case ScreenPortForwardForm:
		return true
	}
	return false
}

// updateLoginConfirm handles the login confirmation screen: enter launches the
// interactive login for the pending profile, esc cancels and returns.
func (m AppModel) updateLoginConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Enter):
		p := m.loginProfile
		m.loading = true
		m.loadingMsg = fmt.Sprintf("Authenticating profile %q…", p.Name)
		// Returning to the originating screen is handled by whoamiMsg via
		// returnOrActions once the login succeeds.
		m.screen = m.loginConfirmReturn
		return m, loginCmd(p)
	case key.Matches(msg, m.keys.Back):
		// Cancel: drop back to the originating screen without logging in.
		m.loading = false
		m.loadingMsg = ""
		dest := m.loginConfirmReturn
		if dest == ScreenLoginConfirm {
			dest = ScreenActions
		}
		m.screen = dest
		return m, nil
	}
	return m, nil
}

func (m AppModel) updateProfiles(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// While capturing filter text, route keys to the filter first.
	if m.profileList.Filtering() {
		switch msg.String() {
		case "enter":
			m.profileList.StopFilter()
			return m, nil
		case "esc":
			m.profileList.ClearFilter()
			return m, nil
		default:
			m.profileList.HandleFilterKey(msg.String(), msg.Runes)
			return m, nil
		}
	}

	switch {
	case key.Matches(msg, m.keys.Filter):
		m.profileList.StartFilter()
		return m, nil
	case key.Matches(msg, m.keys.Yank):
		if p, ok := m.profileList.SelectedProfile(); ok {
			return m, copyCmd("profile", p.Name)
		}
		return m, nil
	case key.Matches(msg, m.keys.Back):
		// Cancel: if we already have an identity, return to the action we came from.
		if m.profileList.Filter() != "" {
			m.profileList.ClearFilter()
			return m, nil
		}
		if m.identity != nil {
			m.screen = m.returnOrActions()
		}
		return m, nil
	case key.Matches(msg, m.keys.Up):
		m.profileList.MoveUp()
		return m, nil
	case key.Matches(msg, m.keys.Down):
		m.profileList.MoveDown()
		return m, nil
	case key.Matches(msg, m.keys.Enter):
		if p, ok := m.profileList.SelectedProfile(); ok {
			// Manual selection: the user just chose this profile, so an ensuing
			// login does not need a confirmation screen.
			return m.connectToProfile(p, true)
		}
		return m, nil
	}
	return m, nil
}

// connectToProfile sets the given profile active, clears cached state and starts
// identity verification (and login if needed). manual is true when the user
// explicitly selected the profile from the list, in which case a subsequent login
// skips the confirmation screen (they just chose it). Automatic connections (e.g.
// startup restore) leave manual=false so a required login is confirmed first.
func (m AppModel) connectToProfile(p awsconfig.Profile, manual bool) (tea.Model, tea.Cmd) {
	m.activeProfile = p.Name
	m.autoLogin = manual
	if p.Region != "" {
		m.activeRegion = p.Region
	} else {
		m.activeRegion = defaultRegion
	}
	m.identity = nil
	m.accountAlias = ""
	// Invalidate cached resources from any previous profile.
	m.instances = nil
	m.ec2.SetInstances(nil)
	m.asgs = nil
	m.lbs = nil
	m.loading = true
	m.loadingMsg = fmt.Sprintf("Connecting to profile %q…", p.Name)
	return m, tea.Batch(whoamiCmd(p, m.activeRegion), m.spinner.Tick)
}

// lastProfile returns the persisted last-used profile if it still exists in the
// current profile list.
func (m AppModel) lastProfile() (awsconfig.Profile, bool) {
	st, err := state.Load()
	if err != nil || st.LastProfile == "" {
		return awsconfig.Profile{}, false
	}
	for _, p := range m.profiles {
		if p.Name == st.LastProfile {
			return p, true
		}
	}
	return awsconfig.Profile{}, false
}

// returnOrActions returns the saved return screen, defaulting to the actions hub.
func (m AppModel) returnOrActions() Screen {
	switch m.returnScreen {
	case ScreenEC2, ScreenTunnels, ScreenActions:
		return m.returnScreen
	default:
		return ScreenActions
	}
}

// actionItem is one entry of the main actions menu. Keeping the list in a single
// place lets updateActions (navigation + shortcuts) and viewActions (rendering)
// stay in sync.
type actionItem struct {
	key      string
	label    string
	category string
	// run performs the action and returns the next model/command.
	run func(m AppModel) (tea.Model, tea.Cmd)
}

// actionItems returns the ordered service entries of the actions hub, grouped by
// category (the order here defines both the linear ↑/↓ order and the on-screen
// grouping). Tunnels is intentionally excluded: it's a transversal view shown as a
// footer shortcut, not an AWS service to browse.
func (m AppModel) actionItems() []actionItem {
	return []actionItem{
		{key: "e", label: "EC2 instances", category: "Compute", run: (AppModel).enterEC2},
		{key: "a", label: "Auto Scaling Groups", category: "Compute", run: (AppModel).enterASG},
		{key: "b", label: "Load Balancers (ALB/NLB)", category: "Network", run: (AppModel).enterELB},
	}
}

// actionCategories returns the ordered, de-duplicated list of categories as they
// appear in actionItems.
func (m AppModel) actionCategories() []string {
	var cats []string
	seen := map[string]bool{}
	for _, it := range m.actionItems() {
		if !seen[it.category] {
			seen[it.category] = true
			cats = append(cats, it.category)
		}
	}
	return cats
}

// nextCategoryCursor returns the index of the first item of the category following
// the one currently under the cursor (wrapping to the first category).
func (m AppModel) nextCategoryCursor() int {
	items := m.actionItems()
	if len(items) == 0 {
		return 0
	}
	cur := m.actionCursor
	if cur < 0 || cur >= len(items) {
		cur = 0
	}
	curCat := items[cur].category
	// Find the start of the next distinct category after the current one.
	for i := cur + 1; i < len(items); i++ {
		if items[i].category != curCat {
			return i
		}
	}
	// Wrap: jump to the very first item (first category).
	return 0
}

// updateActions handles the actions hub: cursor navigation (↑/↓ + enter) plus the
// direct single-key shortcuts. Profile/region/logout are handled globally via the
// persistent config bar.
func (m AppModel) updateActions(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	items := m.actionItems()

	switch {
	case key.Matches(msg, m.keys.Up):
		if m.actionCursor > 0 {
			m.actionCursor--
		}
		return m, nil
	case key.Matches(msg, m.keys.Down):
		if m.actionCursor < len(items)-1 {
			m.actionCursor++
		}
		return m, nil
	case key.Matches(msg, m.keys.ToggleHost): // tab: jump to the next category
		m.actionCursor = m.nextCategoryCursor()
		return m, nil
	case key.Matches(msg, m.keys.Enter):
		if m.actionCursor >= 0 && m.actionCursor < len(items) {
			return items[m.actionCursor].run(m)
		}
		return m, nil
	case key.Matches(msg, m.keys.EC2):
		return m.enterEC2()
	case key.Matches(msg, m.keys.ASG):
		return m.enterASG()
	case key.Matches(msg, m.keys.ELB):
		return m.enterELB()
	case key.Matches(msg, m.keys.Tunnels):
		return m.enterTunnels()
	}
	return m, nil
}

// enterEC2 opens the EC2 list screen, loading instances if none are cached.
func (m AppModel) enterEC2() (tea.Model, tea.Cmd) {
	if m.identity == nil {
		m.err = fmt.Errorf("not connected: choose a profile with 'p'")
		return m, nil
	}
	// Build a fresh, focused, styled table (preserving any loaded data).
	existing := m.instances
	m.ec2 = screens.NewEC2Table(m.contentWidth(), m.ec2ContentHeight())
	m.ec2.SetInstances(existing)
	m.screen = ScreenEC2
	if m.ec2.TotalCount() == 0 {
		m.loading = true
		m.loadingMsg = "Loading EC2 instances…"
		return m, tea.Batch(listInstancesCmd(&m), m.spinner.Tick)
	}
	return m, nil
}

// enterTunnels opens the background-tunnels screen.
func (m AppModel) enterTunnels() (tea.Model, tea.Cmd) {
	m.tunnelCursor = 0
	m.tunnelReturn = ScreenActions
	m.screen = ScreenTunnels
	return m, nil
}

// enterASG opens the Auto Scaling Groups list, loading them if none are cached.
func (m AppModel) enterASG() (tea.Model, tea.Cmd) {
	if m.identity == nil {
		m.err = fmt.Errorf("not connected: choose a profile with 'p'")
		return m, nil
	}
	existing := m.asgs
	m.asg = screens.NewASGTable(m.contentWidth(), m.ec2ContentHeight())
	m.asg.SetGroups(existing)
	m.screen = ScreenASG
	if m.asg.TotalCount() == 0 {
		m.loading = true
		m.loadingMsg = "Loading Auto Scaling Groups…"
		return m, tea.Batch(listASGCmd(&m), m.spinner.Tick)
	}
	return m, nil
}

// enterELB opens the Load Balancers list, loading them if none are cached.
func (m AppModel) enterELB() (tea.Model, tea.Cmd) {
	if m.identity == nil {
		m.err = fmt.Errorf("not connected: choose a profile with 'p'")
		return m, nil
	}
	existing := m.lbs
	m.elb = screens.NewELBTable(m.contentWidth(), m.ec2ContentHeight())
	m.elb.SetLoadBalancers(existing)
	m.screen = ScreenELB
	if m.elb.TotalCount() == 0 {
		m.loading = true
		m.loadingMsg = "Loading Load Balancers…"
		return m, tea.Batch(listELBCmd(&m), m.spinner.Tick)
	}
	return m, nil
}

func (m AppModel) updateRegions(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, m.keys.Back) && m.regionList.FilterState() != list.Filtering {
		m.screen = m.returnOrActions()
		return m, nil
	}
	if key.Matches(msg, m.keys.Enter) && m.regionList.FilterState() != list.Filtering {
		if it, ok := m.regionList.SelectedItem().(screens.RegionItem); ok {
			if it.Region == m.activeRegion {
				m.screen = m.returnOrActions()
				return m, nil
			}
			m.activeRegion = it.Region
			// Invalidate cached resources and rebuild config for the new region.
			m.ec2.SetInstances(nil)
			m.instances = nil
			m.asgs = nil
			m.lbs = nil
			clearCmd := m.setStatus(fmt.Sprintf("Region: %s.", it.Region))

			dest := m.returnOrActions()
			if dest == ScreenEC2 {
				m.ec2 = screens.NewEC2Table(m.contentWidth(), m.ec2ContentHeight())
				m.screen = ScreenEC2
				m.loading = true
				m.loadingMsg = "Applying region…"
				return m, tea.Batch(whoamiCmd(m.currentProfile(), m.activeRegion), clearCmd, m.spinner.Tick)
			}
			m.screen = dest
			m.loading = true
			m.loadingMsg = "Applying region…"
			return m, tea.Batch(whoamiCmd(m.currentProfile(), m.activeRegion), clearCmd, m.spinner.Tick)
		}
	}
	var cmd tea.Cmd
	m.regionList, cmd = m.regionList.Update(msg)
	return m, cmd
}

func (m AppModel) updateEC2(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Filtering mode: capture text until enter/esc.
	if m.ec2.Filtering {
		switch msg.String() {
		case "enter":
			m.ec2.Filtering = false
			return m, nil
		case "esc":
			m.ec2.Filtering = false
			m.ec2.SetFilter("")
			return m, nil
		case "backspace":
			if n := len(m.ec2.Filter); n > 0 {
				m.ec2.SetFilter(m.ec2.Filter[:n-1])
			}
			return m, nil
		default:
			if len(msg.Runes) == 1 {
				m.ec2.SetFilter(m.ec2.Filter + string(msg.Runes))
			}
			return m, nil
		}
	}

	switch {
	case key.Matches(msg, m.keys.Back):
		m.screen = ScreenActions
		return m, nil
	case key.Matches(msg, m.keys.Filter):
		m.ec2.Filtering = true
		return m, nil
	case key.Matches(msg, m.keys.Refresh):
		m.loading = true
		m.loadingMsg = "Refreshing…"
		return m, tea.Batch(listInstancesCmd(&m), m.spinner.Tick)
	case key.Matches(msg, m.keys.Yank):
		inst, ok := m.ec2.SelectedInstance()
		if !ok {
			m.err = fmt.Errorf("no instance selected")
			return m, nil
		}
		return m, copyCmd("instance ID", inst.InstanceID)
	case key.Matches(msg, m.keys.Shell):
		inst, ok := m.ec2.SelectedInstance()
		if !ok {
			m.err = fmt.Errorf("no instance selected")
			return m, nil
		}
		return m, shellCmd(m.activeProfile, m.activeRegion, inst.InstanceID)
	case key.Matches(msg, m.keys.PortFwd):
		inst, ok := m.ec2.SelectedInstance()
		if !ok {
			m.err = fmt.Errorf("no instance selected")
			return m, nil
		}
		m.pfForm = screens.NewPortForwardForm(inst.InstanceID, inst.Name)
		m.pfReturn = ScreenEC2
		m.screen = ScreenPortForwardForm
		return m, nil
	case key.Matches(msg, m.keys.Tunnels):
		m.tunnelCursor = 0
		m.tunnelReturn = ScreenEC2
		m.screen = ScreenTunnels
		return m, nil
	case key.Matches(msg, m.keys.Enter):
		inst, ok := m.ec2.SelectedInstance()
		if !ok {
			m.err = fmt.Errorf("no instance selected")
			return m, nil
		}
		m.ec2Detail = screens.NewEC2Detail(inst, m.contentWidth(), m.contentHeight()-2)
		m.screen = ScreenEC2Detail
		// Load SG and metrics in the background.
		return m, tea.Batch(
			loadSecurityGroupsCmd(m.awsCfg, inst.InstanceID, inst.SecurityGroupIDs),
			loadMetricsCmd(m.awsCfg, inst.InstanceID, m.ec2Detail.MetricWindow()),
			m.spinner.Tick,
		)
	case key.Matches(msg, m.keys.Up):
		m.ec2.MoveUp(1)
		return m, nil
	case key.Matches(msg, m.keys.Down):
		m.ec2.MoveDown(1)
		return m, nil
	case key.Matches(msg, m.keys.ScrollLeft):
		m.ec2.ScrollLeft(8)
		return m, nil
	case key.Matches(msg, m.keys.ScrollRight):
		m.ec2.ScrollRight(8)
		return m, nil
	}
	return m, nil
}

// updateEC2Detail handles the instance detail page: scroll and back.
func (m AppModel) updateEC2Detail(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// When the copy overlay is open, it captures all navigation keys.
	if m.ec2Detail.CopyMenuOpen() {
		switch {
		case key.Matches(msg, m.keys.Back):
			m.ec2Detail.CloseCopyMenu()
			return m, nil
		case key.Matches(msg, m.keys.Up):
			m.ec2Detail.CopyMenuUp()
			return m, nil
		case key.Matches(msg, m.keys.Down):
			m.ec2Detail.CopyMenuDown()
			return m, nil
		case key.Matches(msg, m.keys.Enter):
			if f, ok := m.ec2Detail.SelectedCopyField(); ok {
				m.ec2Detail.CloseCopyMenu()
				return m, copyCmd(f.Label, f.Value)
			}
			return m, nil
		}
		return m, nil
	}

	switch {
	case key.Matches(msg, m.keys.Back):
		m.screen = ScreenEC2
		m.ec2.Focus()
		return m, nil
	case key.Matches(msg, m.keys.Yank):
		return m, copyCmd("instance ID", m.ec2Detail.Instance.InstanceID)
	case key.Matches(msg, m.keys.YankMenu):
		account := ""
		if m.identity != nil {
			account = m.identity.Account
		}
		m.ec2Detail.OpenCopyMenu(m.activeRegion, account)
		return m, nil
	case key.Matches(msg, m.keys.Up):
		m.ec2Detail.ScrollUp(1)
		return m, nil
	case key.Matches(msg, m.keys.Down):
		m.ec2Detail.ScrollDown(1)
		return m, nil
	case key.Matches(msg, m.keys.Refresh):
		// Reload both async blocks.
		inst := m.ec2Detail.Instance
		m.ec2Detail.SGState = screens.BlockLoading
		m.ec2Detail.MetricsState = screens.BlockLoading
		return m, tea.Batch(
			loadSecurityGroupsCmd(m.awsCfg, inst.InstanceID, inst.SecurityGroupIDs),
			loadMetricsCmd(m.awsCfg, inst.InstanceID, m.ec2Detail.MetricWindow()),
			m.spinner.Tick,
		)
	case key.Matches(msg, m.keys.MetricWindow):
		// Cycle the metrics time window and reload only the metrics block.
		win := m.ec2Detail.CycleMetricWindow()
		m.ec2Detail.MetricsState = screens.BlockLoading
		return m, tea.Batch(
			loadMetricsCmd(m.awsCfg, m.ec2Detail.Instance.InstanceID, win),
			m.spinner.Tick,
		)
	case key.Matches(msg, m.keys.Shell):
		return m, shellCmd(m.activeProfile, m.activeRegion, m.ec2Detail.Instance.InstanceID)
	case key.Matches(msg, m.keys.PortFwd):
		inst := m.ec2Detail.Instance
		m.pfForm = screens.NewPortForwardForm(inst.InstanceID, inst.Name)
		m.pfReturn = ScreenEC2Detail
		m.screen = ScreenPortForwardForm
		return m, nil
	}
	return m, nil
}

// updateASG handles the Auto Scaling Groups list screen.
func (m AppModel) updateASG(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.asg.Filtering {
		switch msg.String() {
		case "enter":
			m.asg.Filtering = false
		case "esc":
			m.asg.Filtering = false
			m.asg.SetFilter("")
		case "backspace":
			if n := len(m.asg.Filter); n > 0 {
				m.asg.SetFilter(m.asg.Filter[:n-1])
			}
		default:
			if len(msg.Runes) == 1 {
				m.asg.SetFilter(m.asg.Filter + string(msg.Runes))
			}
		}
		return m, nil
	}

	switch {
	case key.Matches(msg, m.keys.Back):
		m.screen = ScreenActions
		return m, nil
	case key.Matches(msg, m.keys.Filter):
		m.asg.Filtering = true
		return m, nil
	case key.Matches(msg, m.keys.Refresh):
		m.loading = true
		m.loadingMsg = "Refreshing…"
		return m, tea.Batch(listASGCmd(&m), m.spinner.Tick)
	case key.Matches(msg, m.keys.Yank):
		if g, ok := m.asg.SelectedGroup(); ok {
			return m, copyCmd("ASG name", g.Name)
		}
		return m, nil
	case key.Matches(msg, m.keys.Enter):
		g, ok := m.asg.SelectedGroup()
		if !ok {
			return m, nil
		}
		m.asgDetail = screens.NewASGDetail(g, m.contentWidth(), m.contentHeight()-2)
		m.screen = ScreenASGDetail
		return m, tea.Batch(loadASGDetailCmd(m.awsCfg, g), m.spinner.Tick)
	case key.Matches(msg, m.keys.Up):
		m.asg.MoveUp(1)
		return m, nil
	case key.Matches(msg, m.keys.Down):
		m.asg.MoveDown(1)
		return m, nil
	}
	return m, nil
}

// updateASGDetail handles the ASG detail page: scroll, refresh, back.
func (m AppModel) updateASGDetail(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// When the copy overlay is open, it captures all navigation keys.
	if m.asgDetail.CopyMenuOpen() {
		switch {
		case key.Matches(msg, m.keys.Back):
			m.asgDetail.CloseCopyMenu()
			return m, nil
		case key.Matches(msg, m.keys.Up):
			m.asgDetail.CopyMenuUp()
			return m, nil
		case key.Matches(msg, m.keys.Down):
			m.asgDetail.CopyMenuDown()
			return m, nil
		case key.Matches(msg, m.keys.Enter):
			if f, ok := m.asgDetail.SelectedCopyField(); ok {
				m.asgDetail.CloseCopyMenu()
				return m, copyCmd(f.Label, f.Value)
			}
			return m, nil
		}
		return m, nil
	}

	switch {
	case key.Matches(msg, m.keys.Back):
		m.screen = ScreenASG
		return m, nil
	case key.Matches(msg, m.keys.Up):
		m.asgDetail.ScrollUp(1)
		return m, nil
	case key.Matches(msg, m.keys.Down):
		m.asgDetail.ScrollDown(1)
		return m, nil
	case key.Matches(msg, m.keys.Yank):
		return m, copyCmd("ASG name", m.asgDetail.Group.Name)
	case key.Matches(msg, m.keys.YankMenu):
		m.asgDetail.OpenCopyMenu()
		return m, nil
	case key.Matches(msg, m.keys.Refresh):
		m.asgDetail.TGState = screens.BlockLoading
		m.asgDetail.ActState = screens.BlockLoading
		return m, tea.Batch(loadASGDetailCmd(m.awsCfg, m.asgDetail.Group), m.spinner.Tick)
	}
	return m, nil
}

// updateELB handles the Load Balancers list screen.
func (m AppModel) updateELB(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.elb.Filtering {
		switch msg.String() {
		case "enter":
			m.elb.Filtering = false
		case "esc":
			m.elb.Filtering = false
			m.elb.SetFilter("")
		case "backspace":
			if n := len(m.elb.Filter); n > 0 {
				m.elb.SetFilter(m.elb.Filter[:n-1])
			}
		default:
			if len(msg.Runes) == 1 {
				m.elb.SetFilter(m.elb.Filter + string(msg.Runes))
			}
		}
		return m, nil
	}

	switch {
	case key.Matches(msg, m.keys.Back):
		m.screen = ScreenActions
		return m, nil
	case key.Matches(msg, m.keys.Filter):
		m.elb.Filtering = true
		return m, nil
	case key.Matches(msg, m.keys.Refresh):
		m.loading = true
		m.loadingMsg = "Refreshing…"
		return m, tea.Batch(listELBCmd(&m), m.spinner.Tick)
	case key.Matches(msg, m.keys.Yank):
		if lb, ok := m.elb.SelectedLoadBalancer(); ok {
			return m, copyCmd("DNS", lb.DNSName)
		}
		return m, nil
	case key.Matches(msg, m.keys.Enter):
		lb, ok := m.elb.SelectedLoadBalancer()
		if !ok {
			return m, nil
		}
		m.elbDetail = screens.NewELBDetail(lb, m.contentWidth(), m.contentHeight()-2)
		m.screen = ScreenELBDetail
		return m, tea.Batch(
			loadELBListenersCmd(m.awsCfg, lb.ARN),
			loadELBTargetsCmd(m.awsCfg, lb.ARN),
			m.spinner.Tick,
		)
	case key.Matches(msg, m.keys.Up):
		m.elb.MoveUp(1)
		return m, nil
	case key.Matches(msg, m.keys.Down):
		m.elb.MoveDown(1)
		return m, nil
	case key.Matches(msg, m.keys.ScrollLeft):
		m.elb.ScrollLeft(8)
		return m, nil
	case key.Matches(msg, m.keys.ScrollRight):
		m.elb.ScrollRight(8)
		return m, nil
	}
	return m, nil
}

// updateELBDetail handles the load balancer detail page: scroll, refresh, back.
func (m AppModel) updateELBDetail(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// When the copy overlay is open, it captures all navigation keys.
	if m.elbDetail.CopyMenuOpen() {
		switch {
		case key.Matches(msg, m.keys.Back):
			m.elbDetail.CloseCopyMenu()
			return m, nil
		case key.Matches(msg, m.keys.Up):
			m.elbDetail.CopyMenuUp()
			return m, nil
		case key.Matches(msg, m.keys.Down):
			m.elbDetail.CopyMenuDown()
			return m, nil
		case key.Matches(msg, m.keys.Enter):
			if f, ok := m.elbDetail.SelectedCopyField(); ok {
				m.elbDetail.CloseCopyMenu()
				return m, copyCmd(f.Label, f.Value)
			}
			return m, nil
		}
		return m, nil
	}

	switch {
	case key.Matches(msg, m.keys.Back):
		m.screen = ScreenELB
		return m, nil
	case key.Matches(msg, m.keys.Up):
		m.elbDetail.ScrollUp(1)
		return m, nil
	case key.Matches(msg, m.keys.Down):
		m.elbDetail.ScrollDown(1)
		return m, nil
	case key.Matches(msg, m.keys.Yank):
		return m, copyCmd("DNS", m.elbDetail.LB.DNSName)
	case key.Matches(msg, m.keys.YankMenu):
		m.elbDetail.OpenCopyMenu()
		return m, nil
	case key.Matches(msg, m.keys.Refresh):
		m.elbDetail.ListenersState = screens.BlockLoading
		m.elbDetail.TGState = screens.BlockLoading
		return m, tea.Batch(
			loadELBListenersCmd(m.awsCfg, m.elbDetail.LB.ARN),
			loadELBTargetsCmd(m.awsCfg, m.elbDetail.LB.ARN),
			m.spinner.Tick,
		)
	}
	return m, nil
}

// returnFromPF returns to the screen the port-forward form was opened from
// (EC2 list or instance detail), re-focusing the table when relevant.
func (m AppModel) returnFromPF() AppModel {
	switch m.pfReturn {
	case ScreenEC2Detail:
		m.screen = ScreenEC2Detail
	default:
		m.screen = ScreenEC2
		m.ec2.Focus()
	}
	return m
}

func (m AppModel) updatePFForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m = m.returnFromPF()
		return m, nil
	case "tab", "down":
		m.pfForm.Next()
		return m, nil
	case "shift+tab", "up":
		m.pfForm.Prev()
		return m, nil
	case "ctrl+t":
		m.pfForm.ToggleRemoteHost()
		return m, nil
	case "enter":
		host, remotePort, localPort, err := m.pfForm.Values()
		if err != nil {
			m.err = err
			return m, nil
		}
		m.err = nil
		m.loading = true
		m.loadingMsg = "Establishing tunnel…"
		return m, tea.Batch(
			startTunnelCmd(m.tunnels, m.activeProfile, m.activeRegion,
				m.pfForm.Target, m.pfForm.TargetName, host, remotePort, localPort),
			m.spinner.Tick,
		)
	}
	cmd := m.pfForm.Update(msg)
	return m, cmd
}

// updateTunnels handles the background-tunnels screen: navigate, stop one, stop all.
func (m AppModel) updateTunnels(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	tunnels := m.tunnels.list()
	switch {
	case key.Matches(msg, m.keys.Back):
		dest := m.tunnelReturn
		if dest != ScreenEC2 && dest != ScreenActions {
			dest = ScreenActions
		}
		if dest == ScreenEC2 {
			m.ec2.Focus()
		}
		m.screen = dest
		return m, nil
	case key.Matches(msg, m.keys.Up):
		if m.tunnelCursor > 0 {
			m.tunnelCursor--
		}
		return m, nil
	case key.Matches(msg, m.keys.Down):
		if m.tunnelCursor < len(tunnels)-1 {
			m.tunnelCursor++
		}
		return m, nil
	case key.Matches(msg, m.keys.StopFwd):
		if m.tunnelCursor >= 0 && m.tunnelCursor < len(tunnels) {
			id := tunnels[m.tunnelCursor].ID
			m.tunnels.stop(id)
			return m, m.setStatus(fmt.Sprintf("Stopping tunnel #%d…", id))
		}
		return m, nil
	case key.Matches(msg, m.keys.StopAllFwd): // 'X' stop all on this screen
		m.tunnels.stopAll()
		return m, m.setStatus("Stopping all tunnels…")
	case key.Matches(msg, m.keys.RestartFwd): // 'r' restart the selected stopped/failed tunnel
		if m.tunnelCursor >= 0 && m.tunnelCursor < len(tunnels) {
			t := tunnels[m.tunnelCursor]
			if t.State != tunnelStopped && t.State != tunnelFailed {
				return m, m.setStatus("Only a stopped tunnel can be restarted.")
			}
			return m, tea.Batch(
				restartTunnelCmd(m.tunnels, t.ID),
				m.setStatus(fmt.Sprintf("Restarting tunnel #%d…", t.ID)),
			)
		}
		return m, nil
	case key.Matches(msg, m.keys.ClearFwd): // 'c' remove stopped/failed tunnels from the list
		n := m.tunnels.clearStopped()
		m.clampTunnelCursor()
		if n == 0 {
			return m, m.setStatus("No stopped tunnel to clear.")
		}
		return m, m.setStatus(fmt.Sprintf("Cleared %d stopped tunnel(s).", n))
	}
	return m, nil
}

// clampTunnelCursor keeps the tunnels cursor within bounds after list changes.
func (m *AppModel) clampTunnelCursor() {
	n := len(m.tunnels.list())
	if m.tunnelCursor >= n {
		m.tunnelCursor = n - 1
	}
	if m.tunnelCursor < 0 {
		m.tunnelCursor = 0
	}
}

// truncate shortens s to at most n runes, appending an ellipsis when cut.
func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n <= 1 {
		return string(r[:n])
	}
	return string(r[:n-1]) + "…"
}

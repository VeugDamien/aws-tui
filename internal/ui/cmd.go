package ui

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"

	"github.com/aws/aws-sdk-go-v2/aws"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/VeugDamien/aws-tui/internal/auth"
	"github.com/VeugDamien/aws-tui/internal/awsclient"
	"github.com/VeugDamien/aws-tui/internal/awsconfig"
	"github.com/VeugDamien/aws-tui/internal/clipboard"
	"github.com/VeugDamien/aws-tui/internal/ssm"
	"github.com/VeugDamien/aws-tui/internal/state"
)

// loadProfilesCmd reads ~/.aws/config asynchronously.
func loadProfilesCmd() tea.Cmd {
	return func() tea.Msg {
		profiles, err := awsconfig.LoadProfiles()
		if err != nil {
			return errMsg{err}
		}
		return profilesLoadedMsg{profiles}
	}
}

// copyCmd copies value to the system clipboard, echoing back label/value so the
// UI can show a confirmation (or an error) in the status bar.
func copyCmd(label, value string) tea.Cmd {
	return func() tea.Msg {
		err := clipboard.Copy(value)
		return clipboardMsg{label: label, value: value, err: err}
	}
}

// whoamiCmd loads the SDK config for the profile/region and verifies identity.
// It distinguishes "needs login" from other errors.
func whoamiCmd(profile awsconfig.Profile, region string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		cfg, err := awsclient.LoadConfig(ctx, profile.Name, region)
		if err != nil {
			if auth.NeedsLogin(err) {
				return authRequiredMsg{profile}
			}
			return errMsg{err}
		}
		identity, err := auth.Whoami(ctx, cfg)
		if err != nil {
			if auth.NeedsLogin(err) {
				return authRequiredMsg{profile}
			}
			return errMsg{err}
		}
		return whoamiMsg{identity: identity, cfg: cfg}
	}
}

// loginCmd runs the interactive login subprocess for the profile, giving the
// terminal to the CLI (browser flow). On return, identity is re-checked by the
// caller via loginDoneMsg.
func loginCmd(profile awsconfig.Profile) tea.Cmd {
	name, args, ok := auth.LoginCommand(profile)
	if !ok {
		return func() tea.Msg {
			return errMsg{fmt.Errorf("no interactive login method for profile %q (%s)", profile.Name, profile.AuthKind)}
		}
	}
	c := exec.Command(name, args...)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return loginDoneMsg{err}
	})
}

// logoutCmd runs the logout subprocess for the profile.
func logoutCmd(profile awsconfig.Profile) tea.Cmd {
	name, args, ok := auth.LogoutCommand(profile)
	if !ok {
		return func() tea.Msg {
			return errMsg{fmt.Errorf("logout not applicable for profile %q (%s)", profile.Name, profile.AuthKind)}
		}
	}
	return func() tea.Msg {
		out, err := exec.Command(name, args...).CombinedOutput()
		if err != nil {
			return errMsg{fmt.Errorf("logout: %v: %s", err, bytes.TrimSpace(out))}
		}
		return loginDoneMsg{nil}
	}
}

// listInstancesCmd fetches EC2 instances for the active config.
func listInstancesCmd(m *AppModel) tea.Cmd {
	cfg := m.awsCfg
	return func() tea.Msg {
		instances, err := awsclient.ListInstances(context.Background(), cfg)
		if err != nil {
			if auth.NeedsLogin(err) {
				return authRequiredMsg{m.currentProfile()}
			}
			return errMsg{err}
		}
		return instancesLoadedMsg{instances}
	}
}

// listASGCmd fetches Auto Scaling groups for the active config.
func listASGCmd(m *AppModel) tea.Cmd {
	cfg := m.awsCfg
	profile := m.currentProfile()
	return func() tea.Msg {
		groups, err := awsclient.ListAutoScalingGroups(context.Background(), cfg)
		if err != nil {
			if auth.NeedsLogin(err) {
				return authRequiredMsg{profile}
			}
			return errMsg{err}
		}
		return asgsLoadedMsg{groups}
	}
}

// listELBCmd fetches ELBv2 load balancers for the active config.
func listELBCmd(m *AppModel) tea.Cmd {
	cfg := m.awsCfg
	profile := m.currentProfile()
	return func() tea.Msg {
		lbs, err := awsclient.ListLoadBalancers(context.Background(), cfg)
		if err != nil {
			if auth.NeedsLogin(err) {
				return authRequiredMsg{profile}
			}
			return errMsg{err}
		}
		return lbsLoadedMsg{lbs}
	}
}

// loadASGDetailCmd fetches the target groups (with health) and recent scaling
// activities for an ASG detail page. Each block carries its own error so one may
// fail (e.g. missing permission) without hiding the other.
func loadASGDetailCmd(cfg aws.Config, group awsclient.AutoScalingGroup) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		tgs, tgErr := awsclient.DescribeTargetGroupsByARNs(ctx, cfg, group.TargetGroupARNs)
		acts, actErr := awsclient.DescribeScalingActivities(ctx, cfg, group.Name, 8)
		return asgDetailMsg{
			groupName:  group.Name,
			targets:    tgs,
			targetsErr: tgErr,
			activities: acts,
			actErr:     actErr,
		}
	}
}

// loadELBListenersCmd fetches the listeners for a load balancer.
func loadELBListenersCmd(cfg aws.Config, lbARN string) tea.Cmd {
	return func() tea.Msg {
		listeners, err := awsclient.DescribeListeners(context.Background(), cfg, lbARN)
		return elbListenersMsg{lbARN: lbARN, listeners: listeners, err: err}
	}
}

// loadELBTargetsCmd fetches the target groups (with health) for a load balancer.
func loadELBTargetsCmd(cfg aws.Config, lbARN string) tea.Cmd {
	return func() tea.Msg {
		tgs, err := awsclient.DescribeTargetGroupsForLB(context.Background(), cfg, lbARN)
		return elbTargetsMsg{lbARN: lbARN, tgs: tgs, err: err}
	}
}

// accountAliasCmd resolves the IAM account alias. Failures (e.g. missing
// iam:ListAccountAliases permission) are swallowed: the UI keeps the account
// number. The account is echoed back so a stale response from a previous profile
// can be ignored.
func accountAliasCmd(cfg aws.Config, account string) tea.Cmd {
	return func() tea.Msg {
		alias, err := awsclient.AccountAlias(context.Background(), cfg)
		if err != nil {
			return accountAliasMsg{account: account, alias: ""}
		}
		return accountAliasMsg{account: account, alias: alias}
	}
}

// saveProfileCmd persists the last used profile. Persistence failures are silent
// (non-fatal): they must never disrupt the session.
func saveProfileCmd(profile string) tea.Cmd {
	return func() tea.Msg {
		_ = state.Save(state.State{LastProfile: profile})
		return nil
	}
}

// loadSecurityGroupsCmd fetches the security groups for the detail page.
func loadSecurityGroupsCmd(cfg aws.Config, instanceID string, ids []string) tea.Cmd {
	return func() tea.Msg {
		groups, err := awsclient.DescribeSecurityGroups(context.Background(), cfg, ids)
		return sgLoadedMsg{instanceID: instanceID, groups: groups, err: err}
	}
}

// loadMetricsCmd fetches CloudWatch metrics for the detail page over the given
// window (minutes).
func loadMetricsCmd(cfg aws.Config, instanceID string, windowMin int) tea.Cmd {
	return func() tea.Msg {
		m, err := awsclient.GetInstanceMetrics(context.Background(), cfg, instanceID, windowMin)
		return metricsLoadedMsg{instanceID: instanceID, metrics: m, err: err}
	}
}

// shellCmd opens an interactive SSM shell session, ceding the terminal.
func shellCmd(profile, region, target string) tea.Cmd {
	if !ssm.PluginAvailable() {
		return func() tea.Msg {
			return errMsg{fmt.Errorf("session-manager-plugin not found: install it to use SSM sessions")}
		}
	}
	c := ssm.ShellCommand(profile, region, target)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return shellFinishedMsg{err}
	})
}

// startTunnelCmd launches a background port-forward, registers it in the manager
// and returns immediately so the UI stays responsive. A companion waitTunnelCmd
// (batched by the caller) watches for the subprocess exit.
func startTunnelCmd(mgr *tunnelManager, profile, region, target, targetName, host, remotePort, localPort string) tea.Cmd {
	return func() tea.Msg {
		if !ssm.PluginAvailable() {
			return errMsg{fmt.Errorf("session-manager-plugin not found: install it to use port-forwarding")}
		}
		ctx, cancel := context.WithCancel(context.Background())

		var c *exec.Cmd
		if host != "" {
			c = ssm.PortForwardRemoteHostCommand(ctx, profile, region, target, host, remotePort, localPort)
		} else {
			c = ssm.PortForwardCommand(ctx, profile, region, target, remotePort, localPort)
		}
		// Run in its own process group so we can kill the child
		// session-manager-plugin along with the aws CLI on stop.
		setProcessGroup(c)

		buf := &safeBuffer{}
		t := &Tunnel{
			Target: target, TargetName: targetName, Host: host,
			RemotePort: remotePort, LocalPort: localPort,
			Profile: profile, Region: region,
			State:  tunnelStarting,
			cancel: cancel,
			proc:   c,
			buf:    buf,
			done:   make(chan error, 1),
		}
		mgr.add(t)

		if err := launchTunnel(c, cancel, buf); err != nil {
			mgr.setState(t.ID, tunnelFailed, err)
			return tunnelExitedMsg{id: t.ID, err: err}
		}
		mgr.setState(t.ID, tunnelActive, nil)

		// Watch for exit in a detached goroutine; the result is delivered to the
		// UI via waitTunnelCmd which reads this channel.
		go func() {
			err := c.Wait()
			t.done <- err
		}()

		return tunnelStartedMsg{id: t.ID, summary: t.Summary()}
	}
}

// restartTunnelCmd relaunches a previously stopped or failed tunnel, reusing its
// stored connection parameters. It re-arms the existing entry (keeping its ID) so
// the list stays stable, then behaves like startTunnelCmd. A companion
// waitTunnelCmd (batched by the caller) watches for the subprocess exit.
func restartTunnelCmd(mgr *tunnelManager, id int) tea.Cmd {
	return func() tea.Msg {
		if !ssm.PluginAvailable() {
			return errMsg{fmt.Errorf("session-manager-plugin not found: install it to use port-forwarding")}
		}
		t := mgr.get(id)
		if t == nil {
			return nil
		}

		ctx, cancel := context.WithCancel(context.Background())
		var c *exec.Cmd
		if t.Host != "" {
			c = ssm.PortForwardRemoteHostCommand(ctx, t.Profile, t.Region, t.Target, t.Host, t.RemotePort, t.LocalPort)
		} else {
			c = ssm.PortForwardCommand(ctx, t.Profile, t.Region, t.Target, t.RemotePort, t.LocalPort)
		}
		setProcessGroup(c)

		buf := &safeBuffer{}
		if !mgr.prepareRestart(id, c, cancel, buf) {
			// Tunnel is no longer in a restartable state (e.g. running); nothing to do.
			cancel()
			return nil
		}

		if err := launchTunnel(c, cancel, buf); err != nil {
			mgr.setState(id, tunnelFailed, err)
			return tunnelExitedMsg{id: id, err: err}
		}
		mgr.setState(id, tunnelActive, nil)

		done := mgr.get(id).done
		go func() {
			done <- c.Wait()
		}()

		return tunnelStartedMsg{id: id, summary: t.Summary()}
	}
}

// waitTunnelCmd blocks until the tunnel's subprocess exits, then emits
// tunnelExitedMsg. It is safe to run in a Bubble Tea command goroutine.
func waitTunnelCmd(mgr *tunnelManager, id int) tea.Cmd {
	return func() tea.Msg {
		t := mgr.get(id)
		if t == nil {
			return nil
		}
		err := <-t.done
		return tunnelExitedMsg{id: id, err: err}
	}
}

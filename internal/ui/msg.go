package ui

import (
	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/VeugDamien/aws-tui/internal/auth"
	"github.com/VeugDamien/aws-tui/internal/awsclient"
	"github.com/VeugDamien/aws-tui/internal/awsconfig"
)

// profilesLoadedMsg carries the profiles read from ~/.aws/config.
type profilesLoadedMsg struct{ profiles []awsconfig.Profile }

// whoamiMsg is emitted when GetCallerIdentity succeeds.
type whoamiMsg struct {
	identity *auth.CallerIdentity
	cfg      aws.Config
}

// authRequiredMsg is emitted when an AWS call failed due to missing/expired creds
// and an interactive login is applicable.
type authRequiredMsg struct{ profile awsconfig.Profile }

// loginDoneMsg is emitted after an interactive login/logout subprocess returns.
type loginDoneMsg struct{ err error }

// instancesLoadedMsg carries the EC2 instances for the active profile/region.
type instancesLoadedMsg struct{ instances []awsclient.Instance }

// asgsLoadedMsg carries the Auto Scaling groups for the active profile/region.
type asgsLoadedMsg struct{ groups []awsclient.AutoScalingGroup }

// lbsLoadedMsg carries the load balancers for the active profile/region.
type lbsLoadedMsg struct{ lbs []awsclient.LoadBalancer }

// asgDetailMsg carries the async blocks for the ASG detail page (target groups
// and recent scaling activities), each with its own error.
type asgDetailMsg struct {
	groupName  string
	targets    []awsclient.TargetGroup
	targetsErr error
	activities []awsclient.ScalingActivity
	actErr     error
}

// elbListenersMsg carries the listeners for the load balancer detail page.
type elbListenersMsg struct {
	lbARN     string
	listeners []awsclient.Listener
	err       error
}

// elbTargetsMsg carries the target groups with health for the LB detail page.
type elbTargetsMsg struct {
	lbARN string
	tgs   []awsclient.TargetGroup
	err   error
}

// sgLoadedMsg carries the security groups for the detail page (or an error).
type sgLoadedMsg struct {
	instanceID string
	groups     []awsclient.SecurityGroup
	err        error
}

// metricsLoadedMsg carries the CloudWatch metrics for the detail page (or error).
type metricsLoadedMsg struct {
	instanceID string
	metrics    awsclient.InstanceMetrics
	err        error
}

// accountAliasMsg carries the resolved IAM account alias (may be empty).
type accountAliasMsg struct {
	account string // the account this alias was resolved for
	alias   string
}

// tunnelStartedMsg is emitted when a background port-forward has been launched
// successfully (the subprocess started).
type tunnelStartedMsg struct {
	id      int
	summary string
}

// tunnelExitedMsg is emitted when a background port-forward subprocess exits.
type tunnelExitedMsg struct {
	id  int
	err error
}

// shellFinishedMsg is emitted when an interactive shell session returns.
type shellFinishedMsg struct{ err error }

// clipboardMsg is emitted after a copy-to-clipboard attempt. label describes what
// was copied (e.g. "instance ID") for the confirmation message.
type clipboardMsg struct {
	label string
	value string
	err   error
}

// errMsg carries a non-fatal error to display.
type errMsg struct{ err error }

func (e errMsg) Error() string { return e.err.Error() }

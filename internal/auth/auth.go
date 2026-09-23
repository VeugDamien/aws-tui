// Package auth handles identity verification and login/logout command building.
package auth

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	"github.com/VeugDamien/aws-tui/internal/awsconfig"
)

// whoamiTimeout bounds the GetCallerIdentity call so the UI never hangs.
const whoamiTimeout = 8 * time.Second

// CallerIdentity is the resolved identity for the active profile.
type CallerIdentity struct {
	Account string
	Arn     string
	UserID  string
}

// Whoami calls STS GetCallerIdentity with a short timeout.
func Whoami(ctx context.Context, cfg aws.Config) (*CallerIdentity, error) {
	ctx, cancel := context.WithTimeout(ctx, whoamiTimeout)
	defer cancel()

	client := sts.NewFromConfig(cfg)
	out, err := client.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return nil, err
	}
	return &CallerIdentity{
		Account: aws.ToString(out.Account),
		Arn:     aws.ToString(out.Arn),
		UserID:  aws.ToString(out.UserId),
	}, nil
}

// isSSH reports whether we appear to run inside an SSH session.
func isSSH() bool {
	return os.Getenv("SSH_TTY") != "" || os.Getenv("SSH_CONNECTION") != ""
}

// LoginCommand returns the argv to (re)authenticate the given profile.
// ok is false when no interactive login is applicable (credential_process / static).
// When running over SSH, the --remote flag is appended for SSO and login flows.
func LoginCommand(p awsconfig.Profile) (name string, args []string, ok bool) {
	switch p.AuthKind {
	case awsconfig.AuthSSO:
		args = []string{"sso", "login", "--sso-session", p.SSOSession}
	case awsconfig.AuthLogin:
		args = []string{"login", "--profile", p.Name}
	case awsconfig.AuthUnknown:
		// Best effort: try aws login for profiles without an explicit marker.
		args = []string{"login", "--profile", p.Name}
	default:
		// AuthCredentialProcess and AuthStatic: nothing to do interactively.
		return "", nil, false
	}

	if isSSH() {
		args = append(args, "--remote")
	}
	return "aws", args, true
}

// LogoutCommand returns the argv to clear cached credentials for the profile.
// ok is false when logout is not applicable.
func LogoutCommand(p awsconfig.Profile) (name string, args []string, ok bool) {
	switch p.AuthKind {
	case awsconfig.AuthSSO:
		return "aws", []string{"sso", "logout"}, true
	case awsconfig.AuthLogin:
		return "aws", []string{"logout", "--profile", p.Name}, true
	default:
		return "", nil, false
	}
}

// NeedsLogin inspects an error from Whoami (or any AWS call) and reports whether it
// is caused by missing/expired credentials, i.e. an interactive login may help.
func NeedsLogin(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	needles := []string{
		"sso",
		"token",
		"expired",
		"failed to refresh",
		"no valid credential",
		"failed to retrieve credentials",
		"get credentials",
		"unable to load credentials",
		"securitytokenservice: expiredtoken",
		"invalidclienttokenid",
		"could not find",
	}
	for _, n := range needles {
		if strings.Contains(msg, n) {
			return true
		}
	}
	return false
}

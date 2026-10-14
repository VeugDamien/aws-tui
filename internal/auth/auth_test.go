package auth

import (
	"reflect"
	"testing"

	"github.com/VeugDamien/aws-tui/internal/awsconfig"
)

func TestLoginCommand(t *testing.T) {
	// Ensure a clean (non-SSH) environment for the base cases.
	t.Setenv("SSH_TTY", "")
	t.Setenv("SSH_CONNECTION", "")

	tests := []struct {
		name     string
		profile  awsconfig.Profile
		wantName string
		wantArgs []string
		wantOK   bool
	}{
		{
			name:     "sso",
			profile:  awsconfig.Profile{Name: "p", AuthKind: awsconfig.AuthSSO, SSOSession: "eGF"},
			wantName: "aws",
			wantArgs: []string{"sso", "login", "--sso-session", "eGF"},
			wantOK:   true,
		},
		{
			name:     "login",
			profile:  awsconfig.Profile{Name: "egf", AuthKind: awsconfig.AuthLogin},
			wantName: "aws",
			wantArgs: []string{"login", "--profile", "egf"},
			wantOK:   true,
		},
		{
			name:     "unknown best effort",
			profile:  awsconfig.Profile{Name: "barep", AuthKind: awsconfig.AuthUnknown},
			wantName: "aws",
			wantArgs: []string{"login", "--profile", "barep"},
			wantOK:   true,
		},
		{
			name:    "credential process",
			profile: awsconfig.Profile{Name: "cp", AuthKind: awsconfig.AuthCredentialProcess},
			wantOK:  false,
		},
		{
			name:    "static",
			profile: awsconfig.Profile{Name: "st", AuthKind: awsconfig.AuthStatic},
			wantOK:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, args, ok := LoginCommand(tt.profile)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if name != tt.wantName {
				t.Errorf("name = %q, want %q", name, tt.wantName)
			}
			if !reflect.DeepEqual(args, tt.wantArgs) {
				t.Errorf("args = %v, want %v", args, tt.wantArgs)
			}
		})
	}
}

func TestLoginCommandSSHAppendsRemote(t *testing.T) {
	t.Setenv("SSH_TTY", "/dev/ttys001")

	_, args, ok := LoginCommand(awsconfig.Profile{Name: "egf", AuthKind: awsconfig.AuthLogin})
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got := args[len(args)-1]; got != "--remote" {
		t.Errorf("last arg = %q, want --remote (args=%v)", got, args)
	}
}

func TestLogoutCommand(t *testing.T) {
	tests := []struct {
		name     string
		profile  awsconfig.Profile
		wantArgs []string
		wantOK   bool
	}{
		{"sso", awsconfig.Profile{Name: "p", AuthKind: awsconfig.AuthSSO}, []string{"sso", "logout"}, true},
		{"login", awsconfig.Profile{Name: "egf", AuthKind: awsconfig.AuthLogin}, []string{"logout", "--profile", "egf"}, true},
		{"static", awsconfig.Profile{Name: "st", AuthKind: awsconfig.AuthStatic}, nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, args, ok := LogoutCommand(tt.profile)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && !reflect.DeepEqual(args, tt.wantArgs) {
				t.Errorf("args = %v, want %v", args, tt.wantArgs)
			}
		})
	}
}

func TestCanLogin(t *testing.T) {
	cases := []struct {
		kind awsconfig.AuthKind
		want bool
	}{
		{awsconfig.AuthSSO, true},
		{awsconfig.AuthLogin, true},
		{awsconfig.AuthUnknown, true},
		{awsconfig.AuthStatic, false},            // static keys: no interactive login
		{awsconfig.AuthCredentialProcess, false}, // derives its own credentials
	}
	for _, c := range cases {
		if got := CanLogin(awsconfig.Profile{Name: "p", AuthKind: c.kind}); got != c.want {
			t.Errorf("CanLogin(%v) = %v, want %v", c.kind, got, c.want)
		}
	}
}

func TestLoginTarget(t *testing.T) {
	// Profile registry for lookups.
	profiles := map[string]awsconfig.Profile{
		"sso-root":  {Name: "sso-root", AuthKind: awsconfig.AuthSSO, SSOSession: "s"},
		"static":    {Name: "static", AuthKind: awsconfig.AuthStatic},
		"tf-sso":    {Name: "tf-sso", AuthKind: awsconfig.AuthCredentialProcess, SourceProfile: "sso-root"},
		"tf-static": {Name: "tf-static", AuthKind: awsconfig.AuthCredentialProcess, SourceProfile: "static"},
		"tf-orphan": {Name: "tf-orphan", AuthKind: awsconfig.AuthCredentialProcess, SourceProfile: "missing"},
	}
	lookup := func(name string) (awsconfig.Profile, bool) {
		p, ok := profiles[name]
		return p, ok
	}

	cases := []struct {
		name       string
		in         awsconfig.Profile
		wantOK     bool
		wantTarget string
	}{
		{"direct sso", profiles["sso-root"], true, "sso-root"},
		{"cred-process derived from sso logs in the parent", profiles["tf-sso"], true, "sso-root"},
		{"cred-process derived from static has no login", profiles["tf-static"], false, ""},
		{"standalone static has no login", profiles["static"], false, ""},
		{"cred-process with missing parent has no login", profiles["tf-orphan"], false, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := LoginTarget(c.in, lookup)
			if ok != c.wantOK {
				t.Fatalf("ok = %v, want %v", ok, c.wantOK)
			}
			if ok && got.Name != c.wantTarget {
				t.Errorf("target = %q, want %q", got.Name, c.wantTarget)
			}
		})
	}
}

func TestLoginTargetBreaksCycles(t *testing.T) {
	// Two credential_process profiles pointing at each other must not loop.
	profiles := map[string]awsconfig.Profile{
		"a": {Name: "a", AuthKind: awsconfig.AuthCredentialProcess, SourceProfile: "b"},
		"b": {Name: "b", AuthKind: awsconfig.AuthCredentialProcess, SourceProfile: "a"},
	}
	lookup := func(name string) (awsconfig.Profile, bool) {
		p, ok := profiles[name]
		return p, ok
	}
	if _, ok := LoginTarget(profiles["a"], lookup); ok {
		t.Error("a cycle with no loggable profile should return ok=false")
	}
}

func TestNeedsLogin(t *testing.T) {
	if NeedsLogin(nil) {
		t.Error("nil error should not need login")
	}
	if !NeedsLogin(errString("the SSO session associated with this profile has expired")) {
		t.Error("expired SSO error should need login")
	}
	if NeedsLogin(errString("AccessDenied: not authorized to perform ec2:DescribeInstances")) {
		t.Error("access denied should NOT trigger login")
	}
}

type errString string

func (e errString) Error() string { return string(e) }

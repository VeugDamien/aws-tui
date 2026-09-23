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

package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/VeugDamien/aws-tui/internal/awsconfig"
)

// keyMsg builds a tea.KeyMsg for a single rune or named key.
func keyEnter() tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyEnter} }
func keyEsc() tea.KeyMsg   { return tea.KeyMsg{Type: tea.KeyEsc} }

func newModelReady() AppModel {
	m := New("v0.2.0")
	m.width, m.height = 100, 40
	return m
}

// TestAuthRequiredManualSkipsConfirmation verifies that a login triggered by a
// manual profile selection goes straight to the login command, no confirm screen.
func TestAuthRequiredManualSkipsConfirmation(t *testing.T) {
	m := newModelReady()
	m.autoLogin = true // manual selection

	p := awsconfig.Profile{Name: "prod", AuthKind: awsconfig.AuthSSO, SSOSession: "sso-prod"}
	updated, cmd := m.Update(authRequiredMsg{profile: p})
	m = updated.(AppModel)

	if m.screen == ScreenLoginConfirm {
		t.Fatal("manual login should not show the confirmation screen")
	}
	if cmd == nil {
		t.Fatal("expected a login command to be returned")
	}
	if !m.loading {
		t.Fatal("expected loading state while authenticating")
	}
}

// TestAuthRequiredAutoShowsConfirmation verifies that a non-manual login (startup
// restore or token expiry) shows the confirmation screen naming the profile.
func TestAuthRequiredAutoShowsConfirmation(t *testing.T) {
	m := newModelReady()
	m.autoLogin = false
	m.screen = ScreenActions

	p := awsconfig.Profile{Name: "staging", AuthKind: awsconfig.AuthSSO, SSOSession: "sso-stg"}
	updated, _ := m.Update(authRequiredMsg{profile: p})
	m = updated.(AppModel)

	if m.screen != ScreenLoginConfirm {
		t.Fatalf("expected ScreenLoginConfirm, got %v", m.screen)
	}
	if m.loginProfile.Name != "staging" {
		t.Fatalf("expected pending profile 'staging', got %q", m.loginProfile.Name)
	}
	if m.loginConfirmReturn != ScreenActions {
		t.Fatalf("expected return screen to be ScreenActions, got %v", m.loginConfirmReturn)
	}
	if m.loading {
		t.Fatal("should not be in loading state while awaiting confirmation")
	}

	// The confirmation view must name the profile.
	view := m.viewLoginConfirm()
	if !strings.Contains(view, "staging") {
		t.Fatalf("confirmation view should name the profile, got: %q", view)
	}
}

// TestLoginConfirmEnterLaunchesLogin verifies enter starts the login.
func TestLoginConfirmEnterLaunchesLogin(t *testing.T) {
	m := newModelReady()
	m.screen = ScreenLoginConfirm
	m.loginConfirmReturn = ScreenActions
	m.loginProfile = awsconfig.Profile{Name: "prod", AuthKind: awsconfig.AuthSSO, SSOSession: "s"}

	updated, cmd := m.updateLoginConfirm(keyEnter())
	m = updated.(AppModel)

	if cmd == nil {
		t.Fatal("expected a login command on enter")
	}
	if !m.loading {
		t.Fatal("expected loading after confirming login")
	}
}

// TestLoginConfirmEscCancels verifies esc returns without logging in.
func TestLoginConfirmEscCancels(t *testing.T) {
	m := newModelReady()
	m.screen = ScreenLoginConfirm
	m.loginConfirmReturn = ScreenActions
	m.loginProfile = awsconfig.Profile{Name: "prod"}

	updated, _ := m.updateLoginConfirm(keyEsc())
	m = updated.(AppModel)

	if m.screen != ScreenActions {
		t.Fatalf("expected to return to ScreenActions on cancel, got %v", m.screen)
	}
	if m.loading {
		t.Fatal("should not be loading after cancel")
	}
}

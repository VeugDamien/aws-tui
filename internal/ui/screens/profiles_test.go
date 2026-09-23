package screens

import (
	"strings"
	"testing"

	"github.com/VeugDamien/aws-tui/internal/awsconfig"
)

// sampleProfiles mirrors a realistic ~/.aws/config: an SSO session shared by
// several profiles, a Terraform-style credential_process profile deriving from a
// standalone one, plus a couple of unrelated profiles.
func sampleProfiles() []awsconfig.Profile {
	return []awsconfig.Profile{
		{Name: "default", AuthKind: awsconfig.AuthUnknown, Region: "eu-west-3"},
		{Name: "egf", AuthKind: awsconfig.AuthUnknown, Region: "eu-west-3"},
		{Name: "tf-egf", AuthKind: awsconfig.AuthCredentialProcess, SourceProfile: "egf",
			CredentialProcess: "aws configure export-credentials --profile egf"},
		{Name: "sso-egf-mgmt", AuthKind: awsconfig.AuthSSO, SSOSession: "eGF", Region: "eu-west-3"},
		{Name: "sso-egf-devops", AuthKind: awsconfig.AuthSSO, SSOSession: "eGF", Region: "eu-west-3"},
		{Name: "sso-egf-network", AuthKind: awsconfig.AuthSSO, SSOSession: "eGF", Region: "eu-west-3"},
		{Name: "egf-prd", AuthKind: awsconfig.AuthSSO, SSOSession: "eGF", Region: "eu-west-3"},
	}
}

func TestProfileListSelectionSkipsHeaders(t *testing.T) {
	pl := NewProfileList(sampleProfiles(), 80, 30)

	// First selectable must be a profile, never a header.
	first, ok := pl.SelectedProfile()
	if !ok {
		t.Fatal("no profile selected initially")
	}
	if first.Name == "" {
		t.Fatal("selected profile has empty name")
	}

	// Walk down through every selectable row; each must yield a profile.
	seen := map[string]bool{seenName(first): true}
	for i := 0; i < len(sampleProfiles())+5; i++ {
		pl.MoveDown()
		if p, ok := pl.SelectedProfile(); ok {
			seen[seenName(p)] = true
		}
	}
	// All 7 profiles must be reachable by navigation.
	if len(seen) != 7 {
		t.Errorf("reachable profiles = %d, want 7 (%v)", len(seen), seen)
	}
}

func seenName(p awsconfig.Profile) string { return p.Name }

func TestProfileListFilter(t *testing.T) {
	pl := NewProfileList(sampleProfiles(), 80, 30)
	pl.StartFilter()
	pl.HandleFilterKey("s", []rune{'s'})
	pl.HandleFilterKey("s", []rune{'s'})
	pl.HandleFilterKey("o", []rune{'o'})

	out := pl.View()
	// Only sso-* profiles contain "sso"; egf/default must be gone.
	if strings.Contains(out, "default") {
		t.Errorf("filtered view should not contain 'default':\n%s", out)
	}
	if !strings.Contains(out, "sso-egf-mgmt") {
		t.Errorf("filtered view should contain 'sso-egf-mgmt':\n%s", out)
	}
}

// TestProfileListRenderDump prints the rendered widget for manual inspection.
// Run with: go test ./internal/ui/screens -run RenderDump -v
func TestProfileListRenderDump(t *testing.T) {
	pl := NewProfileList(sampleProfiles(), 80, 30)
	t.Log("\n" + pl.View())
	pl.MoveDown()
	pl.MoveDown()
	pl.MoveDown()
	t.Log("\n--- after 3x MoveDown ---\n" + pl.View())
}

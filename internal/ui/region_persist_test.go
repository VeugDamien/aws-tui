package ui

import (
	"testing"

	"github.com/VeugDamien/aws-tui/internal/awsconfig"
)

// TestConnectToProfileRegionPrecedence verifies the region precedence rule:
// the profile's explicit region wins; otherwise an already-set region (e.g. the
// persisted last region) is kept; otherwise the default applies.
func TestConnectToProfileRegionPrecedence(t *testing.T) {
	t.Run("profile region wins", func(t *testing.T) {
		m := New("v0.3.0")
		m.activeRegion = "eu-west-1" // pretend a persisted/previous region
		updated, _ := m.connectToProfile(awsconfig.Profile{Name: "p", Region: "us-east-1"}, true)
		if got := updated.(AppModel).activeRegion; got != "us-east-1" {
			t.Fatalf("region = %q, want us-east-1 (profile region should win)", got)
		}
	})

	t.Run("keeps current region when profile has none", func(t *testing.T) {
		m := New("v0.3.0")
		m.activeRegion = "eu-west-1" // persisted region
		updated, _ := m.connectToProfile(awsconfig.Profile{Name: "p"}, true)
		if got := updated.(AppModel).activeRegion; got != "eu-west-1" {
			t.Fatalf("region = %q, want eu-west-1 (should keep current)", got)
		}
	})

	t.Run("falls back to default when nothing set", func(t *testing.T) {
		m := New("v0.3.0")
		m.activeRegion = "" // nothing persisted
		updated, _ := m.connectToProfile(awsconfig.Profile{Name: "p"}, true)
		if got := updated.(AppModel).activeRegion; got != defaultRegion {
			t.Fatalf("region = %q, want %q (default)", got, defaultRegion)
		}
	})
}

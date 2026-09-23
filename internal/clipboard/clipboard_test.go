package clipboard

import (
	"runtime"
	"testing"
)

func TestCandidatesPerOS(t *testing.T) {
	got := candidates()
	if len(got) == 0 {
		t.Fatal("candidates() returned none")
	}
	switch runtime.GOOS {
	case "darwin":
		if got[0].name != "pbcopy" {
			t.Errorf("darwin first tool = %q, want pbcopy", got[0].name)
		}
	case "windows":
		if got[0].name != "clip" {
			t.Errorf("windows first tool = %q, want clip", got[0].name)
		}
	default:
		// Linux/BSD: expect the Wayland tool first, X11 fallbacks after.
		names := make([]string, len(got))
		for i, c := range got {
			names[i] = c.name
		}
		if names[0] != "wl-copy" {
			t.Errorf("linux tool order = %v, want wl-copy first", names)
		}
	}
}

// TestCopyRoundTrip only runs when a clipboard tool is actually present; on macOS
// dev machines this verifies the real pbcopy path without asserting the paste
// side (which would require pbpaste and could clobber the user's clipboard).
func TestCopyDoesNotErrorWhenToolPresent(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("only exercised on darwin where pbcopy is guaranteed")
	}
	if err := Copy("aws-tui-clipboard-test"); err != nil {
		t.Fatalf("Copy: %v", err)
	}
}

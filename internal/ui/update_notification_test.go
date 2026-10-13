package ui

import (
	"strings"
	"testing"
)

// TestUpdateAvailableMsgSetsHeaderHint verifies that receiving an
// updateAvailableMsg records the version and surfaces a discreet header hint.
func TestUpdateAvailableMsgSetsHeaderHint(t *testing.T) {
	m := New("v0.1.0")
	m.width, m.height = 100, 40

	// No hint before the message.
	if strings.Contains(m.header(), "available") {
		t.Fatal("header should not show an update hint before the message")
	}

	updated, _ := m.Update(updateAvailableMsg{latest: "v0.2.0"})
	m = updated.(AppModel)

	if m.updateVersion != "v0.2.0" {
		t.Fatalf("expected updateVersion to be set, got %q", m.updateVersion)
	}
	h := m.header()
	if !strings.Contains(h, "v0.2.0") || !strings.Contains(h, "upgrade") {
		t.Fatalf("header should advertise the new version and the upgrade command, got: %q", h)
	}
}

// TestHeaderNoHintWhenUpToDate verifies the header stays clean when no newer
// version was reported.
func TestHeaderNoHintWhenUpToDate(t *testing.T) {
	m := New("v0.2.0")
	m.width, m.height = 100, 40
	if strings.Contains(m.header(), "available") {
		t.Fatal("header should not show an update hint when up to date")
	}
}

package ui

import (
	"context"
	"errors"
	"os/exec"
	"testing"
)

// newManagerWith seeds a manager with tunnels in the given states and returns it
// along with the assigned IDs (in insertion order).
func newManagerWith(states ...tunnelState) (*tunnelManager, []int) {
	mgr := newTunnelManager()
	ids := make([]int, 0, len(states))
	for _, st := range states {
		t := &Tunnel{State: st}
		mgr.add(t)
		ids = append(ids, t.ID)
	}
	return mgr, ids
}

func TestClearStoppedRemovesOnlyTerminalTunnels(t *testing.T) {
	mgr, ids := newManagerWith(tunnelActive, tunnelStopped, tunnelStarting, tunnelFailed)

	removed := mgr.clearStopped()
	if removed != 2 {
		t.Fatalf("expected 2 removed, got %d", removed)
	}

	remaining := mgr.list()
	if len(remaining) != 2 {
		t.Fatalf("expected 2 remaining, got %d", len(remaining))
	}
	for _, tu := range remaining {
		if tu.State == tunnelStopped || tu.State == tunnelFailed {
			t.Fatalf("terminal tunnel #%d was not cleared", tu.ID)
		}
	}
	// The active and starting tunnels (ids[0], ids[2]) must survive.
	if mgr.get(ids[0]) == nil || mgr.get(ids[2]) == nil {
		t.Fatal("running tunnels should be preserved")
	}
	if mgr.get(ids[1]) != nil || mgr.get(ids[3]) != nil {
		t.Fatal("stopped/failed tunnels should be gone")
	}
}

func TestClearStoppedNoop(t *testing.T) {
	mgr, _ := newManagerWith(tunnelActive, tunnelStarting)
	if removed := mgr.clearStopped(); removed != 0 {
		t.Fatalf("expected 0 removed, got %d", removed)
	}
	if len(mgr.list()) != 2 {
		t.Fatal("no tunnel should have been removed")
	}
}

func TestPrepareRestartReArmsStoppedTunnel(t *testing.T) {
	mgr, ids := newManagerWith(tunnelStopped)
	id := ids[0]

	// A stopped tunnel carries a previous error and no live process.
	mgr.setState(id, tunnelStopped, errors.New("boom"))

	proc := exec.Command("true")
	_, cancel := context.WithCancel(context.Background())
	defer cancel()
	buf := &safeBuffer{}

	if ok := mgr.prepareRestart(id, proc, cancel, buf); !ok {
		t.Fatal("prepareRestart should succeed for a stopped tunnel")
	}

	got := mgr.get(id)
	if got.State != tunnelStarting {
		t.Fatalf("expected starting state, got %v", got.State)
	}
	if got.Err != nil {
		t.Fatalf("expected error cleared, got %v", got.Err)
	}
	if got.stopRequested {
		t.Fatal("expected stopRequested reset to false")
	}
	if got.proc != proc || got.buf != buf || got.done == nil {
		t.Fatal("expected runtime fields to be re-armed")
	}
}

func TestPrepareRestartRejectsRunningTunnel(t *testing.T) {
	mgr, ids := newManagerWith(tunnelActive)
	proc := exec.Command("true")
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	if ok := mgr.prepareRestart(ids[0], proc, cancel, &safeBuffer{}); ok {
		t.Fatal("prepareRestart should reject a running tunnel")
	}
}

func TestPrepareRestartMissingTunnel(t *testing.T) {
	mgr := newTunnelManager()
	_, cancel := context.WithCancel(context.Background())
	defer cancel()
	if ok := mgr.prepareRestart(999, exec.Command("true"), cancel, &safeBuffer{}); ok {
		t.Fatal("prepareRestart should return false for an unknown id")
	}
}

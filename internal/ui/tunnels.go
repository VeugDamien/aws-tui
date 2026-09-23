package ui

import (
	"context"
	"os/exec"
	"sync"
	"time"
)

// tunnelState is the lifecycle state of a background port-forward.
type tunnelState int

const (
	tunnelStarting tunnelState = iota
	tunnelActive
	tunnelStopped
	tunnelFailed
)

func (s tunnelState) String() string {
	switch s {
	case tunnelStarting:
		return "starting"
	case tunnelActive:
		return "active"
	case tunnelStopped:
		return "stopped"
	case tunnelFailed:
		return "error"
	default:
		return "?"
	}
}

// Tunnel describes a single background SSM port-forward.
type Tunnel struct {
	ID         int
	Target     string // instance id
	TargetName string
	Host       string // empty for instance-port forward
	RemotePort string
	LocalPort  string
	Profile    string
	Region     string

	State   tunnelState
	Err     error
	Started time.Time

	// stopRequested is true when the user asked to stop this tunnel, so an exit
	// with a "signal: terminated" error is treated as a normal stop, not a failure.
	stopRequested bool

	cancel context.CancelFunc
	proc   *exec.Cmd
	buf    *safeBuffer
	done   chan error
}

// Summary renders a one-line description of the tunnel endpoint.
func (t *Tunnel) Summary() string {
	tgt := t.Target
	if t.TargetName != "" {
		tgt = t.TargetName
	}
	if t.Host != "" {
		return tgt + " → " + t.Host + ":" + t.RemotePort + "  (local :" + t.LocalPort + ")"
	}
	return tgt + ":" + t.RemotePort + "  (local :" + t.LocalPort + ")"
}

// Output returns the captured subprocess output so far.
func (t *Tunnel) Output() string {
	if t.buf == nil {
		return ""
	}
	return t.buf.String()
}

// tunnelManager is a goroutine-safe registry of background tunnels. A single
// instance is shared by value-copied AppModel snapshots via a pointer, so its
// state is preserved across Bubble Tea updates.
type tunnelManager struct {
	mu      sync.Mutex
	nextID  int
	tunnels []*Tunnel
}

func newTunnelManager() *tunnelManager {
	return &tunnelManager{}
}

// add registers a new tunnel in the starting state and returns it.
func (mgr *tunnelManager) add(t *Tunnel) {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()
	mgr.nextID++
	t.ID = mgr.nextID
	t.Started = time.Now()
	mgr.tunnels = append(mgr.tunnels, t)
}

// get returns the tunnel with the given id, or nil.
func (mgr *tunnelManager) get(id int) *Tunnel {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()
	for _, t := range mgr.tunnels {
		if t.ID == id {
			return t
		}
	}
	return nil
}

// setState updates the state (and optional error) of a tunnel.
func (mgr *tunnelManager) setState(id int, st tunnelState, err error) {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()
	for _, t := range mgr.tunnels {
		if t.ID == id {
			t.State = st
			if err != nil {
				t.Err = err
			}
			return
		}
	}
}

// list returns a snapshot copy of the tunnels for display.
func (mgr *tunnelManager) list() []Tunnel {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()
	out := make([]Tunnel, 0, len(mgr.tunnels))
	for _, t := range mgr.tunnels {
		out = append(out, *t)
	}
	return out
}

// activeCount returns the number of tunnels currently starting or active.
func (mgr *tunnelManager) activeCount() int {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()
	n := 0
	for _, t := range mgr.tunnels {
		if t.State == tunnelStarting || t.State == tunnelActive {
			n++
		}
	}
	return n
}

// stop cancels a single tunnel.
// stop terminates a single tunnel (and its child session-manager-plugin).
func (mgr *tunnelManager) stop(id int) {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()
	for _, t := range mgr.tunnels {
		if t.ID == id && (t.State == tunnelStarting || t.State == tunnelActive) {
			killTunnel(t)
			return
		}
	}
}

// stopAll terminates every running tunnel (used on quit).
func (mgr *tunnelManager) stopAll() {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()
	for _, t := range mgr.tunnels {
		if t.State == tunnelStarting || t.State == tunnelActive {
			killTunnel(t)
		}
	}
}

// killTunnel marks the tunnel as intentionally stopped, then terminates its whole
// process group so the child session-manager-plugin is killed too, cancelling the
// context as a fallback. Callers must hold the manager mutex.
func killTunnel(t *Tunnel) {
	t.stopRequested = true
	if t.proc != nil && t.proc.Process != nil {
		killProcessGroup(t.proc)
	}
	if t.cancel != nil {
		t.cancel()
	}
}

// wasStopRequested reports whether the given tunnel was stopped on purpose.
func (mgr *tunnelManager) wasStopRequested(id int) bool {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()
	for _, t := range mgr.tunnels {
		if t.ID == id {
			return t.stopRequested
		}
	}
	return false
}

// launch starts the subprocess for a tunnel and wires its lifecycle. It returns
// an error if the process could not be started.
func launchTunnel(c *exec.Cmd, cancel context.CancelFunc, buf *safeBuffer) error {
	c.Stdout = buf
	c.Stderr = buf
	if err := c.Start(); err != nil {
		cancel()
		return err
	}
	return nil
}

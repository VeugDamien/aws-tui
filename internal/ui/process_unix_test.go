//go:build !windows

package ui

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// TestKillProcessGroupKillsChild verifies that killing the process group also
// terminates a child process (mirroring aws CLI -> session-manager-plugin).
func TestKillProcessGroupKillsChild(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "child.pid")

	// Parent shell spawns a long-lived child that writes its PID, then waits.
	script := `child() { echo $$ > "` + marker + `"; sleep 60; }; child & wait`
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := exec.CommandContext(ctx, "sh", "-c", script)
	setProcessGroup(c)
	if err := c.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}

	// Wait for the child to report its PID.
	var childPID int
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		b, err := os.ReadFile(marker)
		if err == nil {
			if pid, convErr := strconv.Atoi(strings.TrimSpace(string(b))); convErr == nil && pid > 0 {
				childPID = pid
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	if childPID == 0 {
		t.Fatal("child PID never reported")
	}

	// Kill the whole group.
	killProcessGroup(c)
	_ = c.Wait()

	// The child must be gone shortly after.
	gone := false
	deadline = time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if !processAlive(childPID) {
			gone = true
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !gone {
		// Best-effort cleanup if the test failed.
		_ = syscall.Kill(childPID, syscall.SIGKILL)
		t.Fatalf("child process %d still alive after group kill", childPID)
	}
}

// processAlive reports whether a process with the given PID exists.
func processAlive(pid int) bool {
	// Signal 0 performs error checking without actually sending a signal.
	err := syscall.Kill(pid, 0)
	return err == nil
}

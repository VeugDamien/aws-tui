//go:build !windows

package ui

import (
	"os/exec"
	"syscall"
)

// setProcessGroup makes the command the leader of a new process group so that the
// whole group (including the child session-manager-plugin) can be signalled at once.
func setProcessGroup(c *exec.Cmd) {
	if c.SysProcAttr == nil {
		c.SysProcAttr = &syscall.SysProcAttr{}
	}
	c.SysProcAttr.Setpgid = true
}

// killProcessGroup terminates the entire process group led by c. It sends SIGTERM
// first, then SIGKILL, using the negative PID to target the group.
func killProcessGroup(c *exec.Cmd) {
	if c.Process == nil {
		return
	}
	pgid, err := syscall.Getpgid(c.Process.Pid)
	if err != nil {
		// Fall back to killing just the process.
		_ = c.Process.Kill()
		return
	}
	// Negative pgid signals the whole group.
	_ = syscall.Kill(-pgid, syscall.SIGTERM)
	_ = syscall.Kill(-pgid, syscall.SIGKILL)
}

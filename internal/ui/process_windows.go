//go:build windows

package ui

import (
	"os/exec"
	"strconv"
	"syscall"
)

// setProcessGroup creates a new process group for the command. On Windows this
// uses CREATE_NEW_PROCESS_GROUP so the spawned tree is isolated; the actual
// tree termination is handled by killProcessGroup via taskkill.
func setProcessGroup(c *exec.Cmd) {
	if c.SysProcAttr == nil {
		c.SysProcAttr = &syscall.SysProcAttr{}
	}
	c.SysProcAttr.CreationFlags |= syscall.CREATE_NEW_PROCESS_GROUP
}

// killProcessGroup terminates the whole process tree on Windows. exec.CommandContext
// (or Process.Kill) only kills the direct child (the aws CLI), leaving the
// session-manager-plugin running, so we use "taskkill /T /F /PID <pid>" which
// forcibly terminates the process and all of its descendants.
func killProcessGroup(c *exec.Cmd) {
	if c.Process == nil {
		return
	}
	pid := strconv.Itoa(c.Process.Pid)
	kill := exec.Command("taskkill", "/T", "/F", "/PID", pid)
	// Hide the taskkill console window.
	kill.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := kill.Run(); err != nil {
		// Fall back to killing just the direct process.
		_ = c.Process.Kill()
	}
}

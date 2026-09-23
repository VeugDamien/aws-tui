// Package ssm builds the AWS CLI commands used for SSM sessions and port-forwards.
// Interactive sessions are delegated to "aws ssm start-session", which invokes the
// session-manager-plugin; the SDK alone cannot manage the interactive tunnel.
package ssm

import (
	"context"
	"fmt"
	"os/exec"
)

// PluginAvailable reports whether the session-manager-plugin is on PATH.
func PluginAvailable() bool {
	_, err := exec.LookPath("session-manager-plugin")
	return err == nil
}

// CLIAvailable reports whether the aws CLI is on PATH.
func CLIAvailable() bool {
	_, err := exec.LookPath("aws")
	return err == nil
}

// baseArgs are the common start-session arguments.
func baseArgs(profile, region, target string) []string {
	return []string{
		"ssm", "start-session",
		"--target", target,
		"--profile", profile,
		"--region", region,
	}
}

// ShellArgs returns the argv for an interactive shell session.
func ShellArgs(profile, region, target string) []string {
	return baseArgs(profile, region, target)
}

// PortForwardArgs returns the argv for a local port-forward to a port on the target.
func PortForwardArgs(profile, region, target, remotePort, localPort string) []string {
	args := baseArgs(profile, region, target)
	args = append(args,
		"--document-name", "AWS-StartPortForwardingSession",
		"--parameters", fmt.Sprintf("portNumber=%s,localPortNumber=%s", remotePort, localPort),
	)
	return args
}

// PortForwardRemoteHostArgs returns the argv for a port-forward to a remote host
// reachable from the target (the target acts as a bastion).
func PortForwardRemoteHostArgs(profile, region, target, host, remotePort, localPort string) []string {
	args := baseArgs(profile, region, target)
	args = append(args,
		"--document-name", "AWS-StartPortForwardingSessionToRemoteHost",
		"--parameters", fmt.Sprintf("host=%s,portNumber=%s,localPortNumber=%s", host, remotePort, localPort),
	)
	return args
}

// ShellCommand builds an *exec.Cmd for an interactive shell session.
func ShellCommand(profile, region, target string) *exec.Cmd {
	return exec.Command("aws", ShellArgs(profile, region, target)...)
}

// PortForwardCommand builds a context-bound *exec.Cmd for a port-forward so it can
// be cancelled by cancelling the context.
func PortForwardCommand(ctx context.Context, profile, region, target, remotePort, localPort string) *exec.Cmd {
	return exec.CommandContext(ctx, "aws", PortForwardArgs(profile, region, target, remotePort, localPort)...)
}

// PortForwardRemoteHostCommand builds a context-bound *exec.Cmd for a remote-host
// port-forward.
func PortForwardRemoteHostCommand(ctx context.Context, profile, region, target, host, remotePort, localPort string) *exec.Cmd {
	return exec.CommandContext(ctx, "aws", PortForwardRemoteHostArgs(profile, region, target, host, remotePort, localPort)...)
}

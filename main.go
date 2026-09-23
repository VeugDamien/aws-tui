// Command aws-tui is a terminal UI to manage AWS connection profiles, list EC2
// instances and start SSM sessions or port-forwards.
package main

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/VeugDamien/aws-tui/internal/selfupdate"
	"github.com/VeugDamien/aws-tui/internal/ssm"
	"github.com/VeugDamien/aws-tui/internal/ui"
)

// Build information, injected at link time via -ldflags. Defaults are used for
// `go build`/`go install` without ldflags (development builds).
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// Sub-command / flag handling (no external dependency).
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v", "version":
			fmt.Printf("aws-tui %s (commit %s, built %s)\n", version, commit, date)
			return
		case "--help", "-h", "help":
			printUsage()
			return
		case "upgrade", "update", "self-update":
			os.Exit(runUpgrade(os.Args[2:]))
		case "check-update", "check-updates":
			os.Exit(runCheckUpdate())
		default:
			fmt.Fprintf(os.Stderr, "unknown command: %q\n\n", os.Args[1])
			printUsage()
			os.Exit(2)
		}
	}

	// Pre-flight: the AWS CLI is required for login flows and SSM sessions.
	if !ssm.CLIAvailable() {
		fmt.Fprintln(os.Stderr, "Error: the AWS CLI ('aws') was not found in PATH.")
		fmt.Fprintln(os.Stderr, "Install AWS CLI v2 and try again.")
		os.Exit(1)
	}

	p := tea.NewProgram(ui.New(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %v\n", err)
		os.Exit(1)
	}
}

// runCheckUpdate reports whether a newer release is available, without installing.
func runCheckUpdate() int {
	status, err := selfupdate.CheckUpdate(context.Background(), version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	if status.Available {
		fmt.Printf("A new version is available: %s (current: %s)\n", status.LatestVersion, version)
		fmt.Println("Run \"aws-tui upgrade\" to update.")
		return 0
	}
	fmt.Printf("aws-tui is up to date (%s).\n", version)
	return 0
}

// runUpgrade downloads and installs the latest release. Supports --force.
func runUpgrade(args []string) int {
	force := false
	for _, a := range args {
		switch a {
		case "--force", "-f":
			force = true
		case "--help", "-h":
			fmt.Println("Usage: aws-tui upgrade [--force]\n\n  --force  reinstall even if already up to date.")
			return 0
		default:
			fmt.Fprintf(os.Stderr, "unknown option: %q\n", a)
			return 2
		}
	}

	res, err := selfupdate.Upgrade(context.Background(), version, selfupdate.Options{
		Force: force,
		Logf:  func(f string, a ...any) { fmt.Printf("==> "+f+"\n", a...) },
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	if !res.Updated {
		return 0
	}
	fmt.Printf("aws-tui was updated to %s. Run the command again.\n", res.NewVersion)
	return 0
}

func printUsage() {
	fmt.Println(`aws-tui — terminal UI for AWS (profiles, EC2, ASG, Load Balancers, SSM).

Usage:
  aws-tui                Launch the interface.
  aws-tui upgrade        Update aws-tui to the latest version (--force to force).
  aws-tui check-update   Check whether a new version is available.
  aws-tui --version      Print the version.
  aws-tui --help         Show this help.

Requirements: AWS CLI v2 and session-manager-plugin in PATH.`)
}

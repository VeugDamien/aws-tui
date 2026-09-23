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
			fmt.Fprintf(os.Stderr, "commande inconnue: %q\n\n", os.Args[1])
			printUsage()
			os.Exit(2)
		}
	}

	// Pre-flight: the AWS CLI is required for login flows and SSM sessions.
	if !ssm.CLIAvailable() {
		fmt.Fprintln(os.Stderr, "Erreur: l'AWS CLI ('aws') est introuvable dans le PATH.")
		fmt.Fprintln(os.Stderr, "Installez AWS CLI v2 puis réessayez.")
		os.Exit(1)
	}

	p := tea.NewProgram(ui.New(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Erreur fatale: %v\n", err)
		os.Exit(1)
	}
}

// runCheckUpdate reports whether a newer release is available, without installing.
func runCheckUpdate() int {
	status, err := selfupdate.CheckUpdate(context.Background(), version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erreur: %v\n", err)
		return 1
	}
	if status.Available {
		fmt.Printf("Une nouvelle version est disponible : %s (actuelle : %s)\n", status.LatestVersion, version)
		fmt.Println("Lancez « aws-tui upgrade » pour mettre à jour.")
		return 0
	}
	fmt.Printf("aws-tui est à jour (%s).\n", version)
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
			fmt.Println("Usage: aws-tui upgrade [--force]\n\n  --force  réinstalle même si déjà à jour.")
			return 0
		default:
			fmt.Fprintf(os.Stderr, "option inconnue: %q\n", a)
			return 2
		}
	}

	res, err := selfupdate.Upgrade(context.Background(), version, selfupdate.Options{
		Force: force,
		Logf:  func(f string, a ...any) { fmt.Printf("==> "+f+"\n", a...) },
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erreur: %v\n", err)
		return 1
	}
	if !res.Updated {
		return 0
	}
	fmt.Printf("aws-tui a été mis à jour vers %s. Relancez la commande.\n", res.NewVersion)
	return 0
}

func printUsage() {
	fmt.Println(`aws-tui — interface terminal pour AWS (profils, EC2, ASG, Load Balancers, SSM).

Usage:
  aws-tui                Lance l'interface.
  aws-tui upgrade        Met à jour aws-tui vers la dernière version (--force pour forcer).
  aws-tui check-update   Vérifie si une nouvelle version est disponible.
  aws-tui --version      Affiche la version.
  aws-tui --help         Affiche cette aide.

Pré-requis: AWS CLI v2 et session-manager-plugin dans le PATH.`)
}

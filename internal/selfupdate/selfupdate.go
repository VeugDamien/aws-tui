package selfupdate

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// binaryName is the executable name inside archives.
const binaryName = "aws-tui"

// assetName returns the release archive filename for the current platform, given
// a version number without the leading "v" (e.g. "1.2.3").
func assetName(numVersion string) string {
	ext := "tar.gz"
	if runtime.GOOS == "windows" {
		ext = "zip"
	}
	return fmt.Sprintf("%s_%s_%s_%s.%s", binaryName, numVersion, runtime.GOOS, runtime.GOARCH, ext)
}

// Status is the result of a non-destructive update check.
type Status struct {
	CurrentVersion string
	LatestVersion  string
	Available      bool
	Release        *Release
}

// CheckUpdate queries the latest release and compares it with current, without
// downloading or installing anything.
func CheckUpdate(ctx context.Context, current string) (*Status, error) {
	rel, err := LatestRelease(ctx)
	if err != nil {
		return nil, err
	}
	return &Status{
		CurrentVersion: current,
		LatestVersion:  rel.TagName,
		Available:      IsNewer(current, rel.TagName),
		Release:        rel,
	}, nil
}

// Options tunes an Upgrade run.
type Options struct {
	// Force reinstalls even when the current version is already up to date.
	Force bool
	// Logf, when set, receives human-readable progress lines.
	Logf func(format string, args ...any)
}

func (o Options) logf(format string, args ...any) {
	if o.Logf != nil {
		o.Logf(format, args...)
	}
}

// Result describes the outcome of an Upgrade.
type Result struct {
	PreviousVersion string
	NewVersion      string
	Updated         bool // false when already up to date and not forced
}

// Upgrade downloads the latest release for the current platform, verifies its
// checksum and atomically replaces the running executable. It refuses to run when
// the binary appears to be managed by a package manager.
func Upgrade(ctx context.Context, current string, opts Options) (*Result, error) {
	exe, err := executablePath()
	if err != nil {
		return nil, err
	}

	// Guard: package-manager-managed installs must be updated by that manager.
	if mgr := managedBy(exe); mgr != "" {
		return nil, fmt.Errorf(
			"aws-tui appears to be installed via %s (%s).\n"+
				"Update it with that manager (e.g. \"%s\"),\n"+
				"or reinstall via the install script to enable auto-update.",
			mgr, exe, upgradeHint(mgr))
	}

	opts.logf("Looking up the latest version…")
	status, err := CheckUpdate(ctx, current)
	if err != nil {
		return nil, err
	}

	if !status.Available && !opts.Force {
		opts.logf("Already up to date (%s).", current)
		return &Result{PreviousVersion: current, NewVersion: current, Updated: false}, nil
	}

	num := strings.TrimPrefix(status.LatestVersion, "v")
	asset, err := status.Release.findAsset(assetName(num))
	if err != nil {
		return nil, err
	}

	opts.logf("Downloading %s…", asset.Name)
	archivePath, cleanup, err := downloadToTemp(ctx, asset)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	opts.logf("Verifying SHA-256 checksum…")
	if err := verifyChecksum(ctx, status.Release, asset.Name, archivePath); err != nil {
		return nil, err
	}

	opts.logf("Extracting…")
	newBin, err := extractBinary(archivePath)
	if err != nil {
		return nil, err
	}
	defer os.Remove(newBin)

	opts.logf("Installing %s → %s…", status.LatestVersion, exe)
	if err := replaceExecutable(exe, newBin); err != nil {
		return nil, err
	}

	opts.logf("Update complete: %s → %s", current, status.LatestVersion)
	return &Result{PreviousVersion: current, NewVersion: status.LatestVersion, Updated: true}, nil
}

// executablePath returns the absolute, symlink-resolved path of the running binary.
func executablePath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("executable path not found: %w", err)
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return exe, nil
}

// managedBy returns a non-empty label when the executable path looks like it is
// owned by a package manager, meaning self-update should be declined.
func managedBy(exe string) string {
	p := filepath.ToSlash(exe)
	switch {
	case strings.Contains(p, "/Cellar/"), strings.Contains(p, "/homebrew/"),
		strings.Contains(p, "/linuxbrew/"):
		return "Homebrew"
	case strings.HasPrefix(p, "/usr/bin/"), strings.HasPrefix(p, "/usr/local/Cellar/"):
		// /usr/bin is where the .deb/.rpm packages install.
		return "a system package (apt/dnf/rpm)"
	case strings.Contains(strings.ToLower(p), "/scoop/"):
		return "Scoop"
	case strings.Contains(p, "Program Files"):
		return "a Windows installer"
	}
	return ""
}

// upgradeHint returns the recommended command to update via a given manager.
func upgradeHint(mgr string) string {
	switch {
	case strings.Contains(mgr, "Homebrew"):
		return "brew upgrade aws-tui"
	case strings.Contains(mgr, "system package"):
		return "sudo apt upgrade aws-tui / sudo dnf upgrade aws-tui"
	case strings.Contains(mgr, "Scoop"):
		return "scoop update aws-tui"
	default:
		return "your package manager"
	}
}

//go:build windows

package selfupdate

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// replaceExecutable replaces dst with src on Windows. A running .exe cannot be
// overwritten, but it can be renamed: move the current binary aside to "<name>.old",
// then put the new binary in place. The stale ".old" is removed on a later run.
func replaceExecutable(dst, src string) error {
	dstDir := filepath.Dir(dst)

	// Stage the new binary next to the target (same volume → fast move).
	staged := filepath.Join(dstDir, ".aws-tui.new.exe")
	if err := copyFile(src, staged); err != nil {
		return fmt.Errorf("préparation du nouveau binaire (droits d'écriture sur %s ?): %w", dstDir, err)
	}
	defer os.Remove(staged)

	// Clean up any leftover .old from a previous upgrade (best effort).
	old := dst + ".old"
	_ = os.Remove(old)

	// Move the running executable aside; this is permitted while it runs.
	if err := os.Rename(dst, old); err != nil {
		return fmt.Errorf("déplacement de l'ancien binaire: %w", err)
	}

	// Put the new binary in place.
	if err := os.Rename(staged, dst); err != nil {
		// Roll back so the tool stays runnable.
		_ = os.Rename(old, dst)
		return fmt.Errorf("installation du nouveau binaire: %w", err)
	}

	// Try to delete the old binary; it stays locked while running, and will be
	// removed by the cleanup above on the next upgrade.
	_ = os.Remove(old)
	return nil
}

// copyFile copies src to dst, replacing dst if it exists.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(dst)
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(dst)
		return err
	}
	return nil
}

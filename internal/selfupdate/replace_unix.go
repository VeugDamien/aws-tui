//go:build !windows

package selfupdate

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// replaceExecutable replaces the file at dst with the freshly extracted binary at
// src, atomically when possible. On Unix, an in-use executable can be replaced via
// rename (the running process keeps the old inode). It preserves dst's permissions.
func replaceExecutable(dst, src string) error {
	dstDir := filepath.Dir(dst)

	// Preserve the existing file mode if we can stat it.
	mode := os.FileMode(0o755)
	if fi, err := os.Stat(dst); err == nil {
		mode = fi.Mode().Perm()
	}

	// Stage the new binary in the destination directory so rename is atomic
	// (same filesystem). Fall back to a copy if rename across devices fails.
	staged := filepath.Join(dstDir, ".aws-tui.new")
	if err := copyFile(src, staged, mode); err != nil {
		return fmt.Errorf("préparation du nouveau binaire (droits d'écriture sur %s ?): %w", dstDir, err)
	}
	defer os.Remove(staged) // no-op if the rename below consumed it

	if err := os.Rename(staged, dst); err != nil {
		return fmt.Errorf("remplacement du binaire %s: %w", dst, err)
	}
	return nil
}

// copyFile copies src to dst with the given mode, replacing dst if it exists.
func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
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
	return os.Chmod(dst, mode)
}

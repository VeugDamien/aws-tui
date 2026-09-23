package selfupdate

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path"
	"runtime"
	"strings"
)

// maxBinaryBytes caps a single extracted file to guard against decompression
// bombs (100 MiB is well above the real binary size).
const maxBinaryBytes = 100 << 20

// extractBinary extracts the aws-tui executable from the archive to a temporary
// file and returns its path. The caller is responsible for removing it.
func extractBinary(archivePath string) (string, error) {
	if runtime.GOOS == "windows" || strings.HasSuffix(archivePath, ".zip") {
		return extractFromZip(archivePath)
	}
	return extractFromTarGz(archivePath)
}

// wantBinary is the base name we look for inside the archive.
func wantBinary() string {
	if runtime.GOOS == "windows" {
		return binaryName + ".exe"
	}
	return binaryName
}

func extractFromTarGz(archivePath string) (string, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return "", fmt.Errorf("gzip: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	want := wantBinary()
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("lecture tar: %w", err)
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		if path.Base(hdr.Name) == want {
			return writeTempBinary(tr)
		}
	}
	return "", fmt.Errorf("binary %q not found in the archive", want)
}

func extractFromZip(archivePath string) (string, error) {
	zr, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", fmt.Errorf("zip: %w", err)
	}
	defer zr.Close()

	want := wantBinary()
	for _, zf := range zr.File {
		if path.Base(zf.Name) != want {
			continue
		}
		rc, err := zf.Open()
		if err != nil {
			return "", err
		}
		defer rc.Close()
		return writeTempBinary(rc)
	}
	return "", fmt.Errorf("binary %q not found in the archive", want)
}

// writeTempBinary copies r into a new temp file (in the same directory as the
// target executable when possible) and marks it executable.
func writeTempBinary(r io.Reader) (string, error) {
	out, err := os.CreateTemp("", "aws-tui-bin-*")
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(out, io.LimitReader(r, maxBinaryBytes)); err != nil {
		out.Close()
		os.Remove(out.Name())
		return "", fmt.Errorf("extracting the binary: %w", err)
	}
	if err := out.Close(); err != nil {
		os.Remove(out.Name())
		return "", err
	}
	if err := os.Chmod(out.Name(), 0o755); err != nil {
		os.Remove(out.Name())
		return "", err
	}
	return out.Name(), nil
}

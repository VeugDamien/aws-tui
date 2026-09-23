package selfupdate

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// maxDownloadBytes caps archive downloads defensively (200 MiB is well above any
// realistic binary archive).
const maxDownloadBytes = 200 << 20

// downloadToTemp streams an asset to a temporary file and returns its path plus a
// cleanup function to remove it.
func downloadToTemp(ctx context.Context, asset Asset) (path string, cleanup func(), err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, asset.BrowserDownloadURL, nil)
	if err != nil {
		return "", func() {}, err
	}
	req.Header.Set("User-Agent", "aws-tui-selfupdate")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", func() {}, fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", func() {}, fmt.Errorf("download: status %d", resp.StatusCode)
	}

	f, err := os.CreateTemp("", "aws-tui-update-*")
	if err != nil {
		return "", func() {}, err
	}
	cleanup = func() { f.Close(); os.Remove(f.Name()) }

	if _, err := io.Copy(f, io.LimitReader(resp.Body, maxDownloadBytes)); err != nil {
		cleanup()
		return "", func() {}, fmt.Errorf("writing the archive: %w", err)
	}
	if err := f.Close(); err != nil {
		cleanup()
		return "", func() {}, err
	}
	return f.Name(), cleanup, nil
}

// verifyChecksum downloads the release's checksums.txt and confirms the SHA-256 of
// archivePath matches the entry for assetName. When checksums.txt is absent, it
// returns an error (refusing to install unverified binaries).
func verifyChecksum(ctx context.Context, rel *Release, assetName, archivePath string) error {
	sums, err := rel.findAsset("checksums.txt")
	if err != nil {
		return fmt.Errorf("checksums.txt not found in the release: refusing to install an unverified binary")
	}

	expected, err := fetchExpectedChecksum(ctx, sums.BrowserDownloadURL, assetName)
	if err != nil {
		return err
	}

	actual, err := fileSHA256(archivePath)
	if err != nil {
		return err
	}
	if !strings.EqualFold(actual, expected) {
		return fmt.Errorf("invalid checksum for %s (expected %s, got %s)", assetName, expected, actual)
	}
	return nil
}

// fetchExpectedChecksum downloads a checksums.txt file and returns the hash listed
// for the given asset name.
func fetchExpectedChecksum(ctx context.Context, url, assetName string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "aws-tui-selfupdate")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("downloading checksums: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("checksums: statut %d", resp.StatusCode)
	}

	// Each line: "<sha256>  <filename>". The filename may be bare or prefixed.
	sc := bufio.NewScanner(io.LimitReader(resp.Body, 1<<20))
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) != 2 {
			continue
		}
		name := fields[1]
		name = strings.TrimPrefix(name, "./")
		name = strings.TrimPrefix(name, "*") // some tools mark binary mode with '*'
		if name == assetName {
			return fields[0], nil
		}
	}
	if err := sc.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("no checksum listed for %s", assetName)
}

// fileSHA256 returns the lowercase hex SHA-256 of a file.
func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

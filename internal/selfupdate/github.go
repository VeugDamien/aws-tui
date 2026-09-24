package selfupdate

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Repository coordinates for the GitHub releases API.
const (
	owner = "VeugDamien"
	repo  = "aws-tui"
)

// httpTimeout bounds every network call so the CLI never hangs.
const httpTimeout = 30 * time.Second

// Release is the subset of a GitHub release we care about.
type Release struct {
	TagName string  `json:"tag_name"`
	Name    string  `json:"name"`
	HTMLURL string  `json:"html_url"`
	Assets  []Asset `json:"assets"`
}

// Asset is a single downloadable file attached to a release.
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// allowedRedirectHosts are the domains the self-updater is willing to follow a
// redirect to. GitHub's release/API endpoints redirect asset downloads to its
// own CDN (and historically to signed S3 URLs), so we allow those and nothing
// else. This prevents a tampered or malicious redirect from pointing the updater
// at an attacker-controlled host.
var allowedRedirectHosts = []string{
	"github.com",
	"api.github.com",
	"codeload.github.com",
	"objects.githubusercontent.com",
	"release-assets.githubusercontent.com",
	"githubusercontent.com",
	"amazonaws.com", // GitHub asset downloads historically 302 to signed S3 URLs
}

// hostAllowed reports whether host (or a parent domain) is in the allow-list.
func hostAllowed(host string) bool {
	host = strings.ToLower(host)
	if i := strings.IndexByte(host, ':'); i >= 0 {
		host = host[:i] // strip any port
	}
	for _, allowed := range allowedRedirectHosts {
		if host == allowed || strings.HasSuffix(host, "."+allowed) {
			return true
		}
	}
	return false
}

// httpClient is the shared client with a sane timeout. CheckRedirect confines
// redirects to known GitHub/AWS hosts and caps the redirect chain length.
var httpClient = &http.Client{
	Timeout: httpTimeout,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return fmt.Errorf("stopped after 10 redirects")
		}
		if !hostAllowed(req.URL.Hostname()) {
			return fmt.Errorf("refusing redirect to untrusted host %q", req.URL.Hostname())
		}
		return nil
	},
}

// LatestRelease fetches the latest published (non-draft, non-prerelease) release.
func LatestRelease(ctx context.Context) (*Release, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "aws-tui-selfupdate")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("contacting the GitHub API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("no release published for %s/%s", owner, repo)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("GitHub API: status %d: %s", resp.StatusCode, string(body))
	}

	var rel Release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, fmt.Errorf("decoding GitHub response: %w", err)
	}
	if rel.TagName == "" {
		return nil, fmt.Errorf("GitHub response without tag_name")
	}
	return &rel, nil
}

// findAsset returns the asset whose name equals want, or an error listing the
// available asset names.
func (r *Release) findAsset(want string) (Asset, error) {
	for _, a := range r.Assets {
		if a.Name == want {
			return a, nil
		}
	}
	var names []string
	for _, a := range r.Assets {
		names = append(names, a.Name)
	}
	return Asset{}, fmt.Errorf("archive %q not found in the release (%d assets available: %v)", want, len(r.Assets), names)
}

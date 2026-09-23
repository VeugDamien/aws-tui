// Package selfupdate implements a manual, dependency-free self-update mechanism
// for aws-tui: it queries the GitHub Releases API, downloads the platform
// archive, verifies its SHA-256 checksum and atomically replaces the running
// binary. It refuses to act when the binary is managed by a package manager.
package selfupdate

import (
	"strconv"
	"strings"
)

// semver is a parsed semantic version (major.minor.patch), ignoring any
// pre-release/build metadata for the comparison.
type semver struct {
	major, minor, patch int
	pre                 string // pre-release label, e.g. "rc1" (empty for releases)
	ok                  bool   // whether parsing succeeded
}

// parseVersion parses strings like "v1.2.3", "1.2.3", "1.2.3-rc1" or "dev".
// Unparseable inputs (e.g. "dev") return ok=false.
func parseVersion(s string) semver {
	v := semver{}
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "v")
	if s == "" {
		return v
	}

	// Split off pre-release / build metadata.
	core := s
	if i := strings.IndexAny(s, "-+"); i >= 0 {
		core = s[:i]
		if s[i] == '-' {
			v.pre = strings.TrimSuffix(s[i+1:], "")
		}
	}

	parts := strings.Split(core, ".")
	if len(parts) == 0 || len(parts) > 3 {
		return v
	}
	nums := make([]int, 3)
	for i := 0; i < len(parts); i++ {
		n, err := strconv.Atoi(parts[i])
		if err != nil || n < 0 {
			return v
		}
		nums[i] = n
	}
	v.major, v.minor, v.patch = nums[0], nums[1], nums[2]
	v.ok = true
	return v
}

// compare returns -1 if a<b, 0 if equal, +1 if a>b (core version only). A version
// with a pre-release label is considered lower than the same core without one.
func compare(a, b semver) int {
	switch {
	case a.major != b.major:
		return sign(a.major - b.major)
	case a.minor != b.minor:
		return sign(a.minor - b.minor)
	case a.patch != b.patch:
		return sign(a.patch - b.patch)
	}
	// Same core: a release (no pre) outranks a pre-release.
	switch {
	case a.pre == "" && b.pre != "":
		return 1
	case a.pre != "" && b.pre == "":
		return -1
	case a.pre == b.pre:
		return 0
	case a.pre < b.pre:
		return -1
	default:
		return 1
	}
}

func sign(n int) int {
	switch {
	case n < 0:
		return -1
	case n > 0:
		return 1
	default:
		return 0
	}
}

// IsNewer reports whether the "latest" version string is strictly newer than the
// "current" one. If current is unparseable (e.g. a "dev" build), any parseable
// latest is considered newer so developers can still test the flow.
func IsNewer(current, latest string) bool {
	c := parseVersion(current)
	l := parseVersion(latest)
	if !l.ok {
		return false
	}
	if !c.ok {
		return true
	}
	return compare(l, c) > 0
}

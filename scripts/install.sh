#!/usr/bin/env sh
#
# install.sh — installs aws-tui on macOS or Linux from GitHub Releases.
#
# Usage (latest version):
#   curl -fsSL https://raw.githubusercontent.com/VeugDamien/aws-tui/main/scripts/install.sh | sh
#
# Options via environment variables:
#   VERSION=v1.2.3   install a specific version (default: latest release)
#   BINDIR=~/.local/bin   install directory (default: /usr/local/bin,
#                         or ~/.local/bin without root access)
#
# POSIX sh, no bashisms. Dependencies: curl or wget, tar, uname.

set -eu

OWNER="VeugDamien"
REPO="aws-tui"
BINARY="aws-tui"

info()  { printf '\033[1;34m==>\033[0m %s\n' "$1"; }
warn()  { printf '\033[1;33mwarning:\033[0m %s\n' "$1" >&2; }
error() { printf '\033[1;31merror:\033[0m %s\n' "$1" >&2; exit 1; }

# --- OS / architecture detection -------------------------------------------

os="$(uname -s)"
case "$os" in
	Darwin) os="darwin" ;;
	Linux)  os="linux" ;;
	*) error "unsupported OS: $os (this script handles macOS and Linux; on Windows, use install.ps1)." ;;
esac

arch="$(uname -m)"
case "$arch" in
	x86_64 | amd64) arch="amd64" ;;
	arm64 | aarch64) arch="arm64" ;;
	*) error "unsupported architecture: $arch" ;;
esac

# --- Download tool ----------------------------------------------------------

if command -v curl >/dev/null 2>&1; then
	dl() { curl -fsSL "$1"; }
	dl_out() { curl -fsSL "$1" -o "$2"; }
elif command -v wget >/dev/null 2>&1; then
	dl() { wget -qO- "$1"; }
	dl_out() { wget -qO "$2" "$1"; }
else
	error "curl or wget is required."
fi

# --- Version resolution -----------------------------------------------------

VERSION="${VERSION:-}"
if [ -z "$VERSION" ]; then
	info "Looking up the latest version…"
	api="https://api.github.com/repos/${OWNER}/${REPO}/releases/latest"
	# Extracts "tag_name": "vX.Y.Z" without depending on jq.
	VERSION="$(dl "$api" | grep '"tag_name"' | head -n1 | sed -E 's/.*"tag_name" *: *"([^"]+)".*/\1/')"
	[ -n "$VERSION" ] || error "unable to determine the latest version (no release published?)."
fi
# Number without the 'v' prefix for the archive name.
num="${VERSION#v}"

# --- Install directory ------------------------------------------------------

if [ -n "${BINDIR:-}" ]; then
	bindir="$BINDIR"
elif [ "$(id -u)" = "0" ]; then
	bindir="/usr/local/bin"
elif [ -w "/usr/local/bin" ] 2>/dev/null; then
	bindir="/usr/local/bin"
else
	bindir="$HOME/.local/bin"
fi

# --- Download + verification ------------------------------------------------

archive="${BINARY}_${num}_${os}_${arch}.tar.gz"
base="https://github.com/${OWNER}/${REPO}/releases/download/${VERSION}"
url="${base}/${archive}"

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

info "Downloading ${archive} (${VERSION})…"
dl_out "$url" "$tmp/$archive" || error "download failed: $url"

# Verify the checksum if the checksums.txt file is available.
if dl_out "${base}/checksums.txt" "$tmp/checksums.txt" 2>/dev/null; then
	info "Verifying SHA-256 checksum…"
	expected="$(grep " .*${archive}\$" "$tmp/checksums.txt" | awk '{print $1}' | head -n1)"
	if [ -n "$expected" ]; then
		if command -v sha256sum >/dev/null 2>&1; then
			actual="$(sha256sum "$tmp/$archive" | awk '{print $1}')"
		elif command -v shasum >/dev/null 2>&1; then
			actual="$(shasum -a 256 "$tmp/$archive" | awk '{print $1}')"
		else
			actual=""
			warn "no sha256 tool; verification skipped."
		fi
		if [ -n "$actual" ] && [ "$actual" != "$expected" ]; then
			error "invalid checksum (expected $expected, got $actual)."
		fi
	fi
else
	warn "checksums.txt unavailable; verification skipped."
fi

# --- Extraction + installation ---------------------------------------------

info "Extracting…"
tar -xzf "$tmp/$archive" -C "$tmp"
# The archive contains a folder ${BINARY}_${num}_${os}_${arch}/${BINARY}.
src="$tmp/${BINARY}_${num}_${os}_${arch}/${BINARY}"
[ -f "$src" ] || src="$(find "$tmp" -type f -name "$BINARY" | head -n1)"
[ -f "$src" ] || error "binary not found in the archive."

mkdir -p "$bindir"
if [ -w "$bindir" ]; then
	install -m 0755 "$src" "$bindir/$BINARY"
else
	info "Elevated privileges required to write to $bindir (sudo)…"
	sudo install -m 0755 "$src" "$bindir/$BINARY"
fi

info "Installed: $bindir/$BINARY"

# --- Post-install checks ----------------------------------------------------

case ":$PATH:" in
	*":$bindir:"*) : ;;
	*) warn "$bindir is not in your PATH. Add:  export PATH=\"$bindir:\$PATH\"" ;;
esac

command -v aws >/dev/null 2>&1 || warn "AWS CLI v2 ('aws') not found: required for logins and SSM sessions."
command -v session-manager-plugin >/dev/null 2>&1 || warn "session-manager-plugin not found: required for SSM sessions/port-forwards."

"$bindir/$BINARY" --version 2>/dev/null || true
info "Done. Run: $BINARY"

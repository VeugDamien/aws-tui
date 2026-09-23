#!/usr/bin/env sh
#
# uninstall.sh — removes aws-tui from a macOS/Linux machine.
#
# Looks for the binary in the usual install locations and removes it.
# Usage:
#   scripts/uninstall.sh
#   BINDIR=~/.local/bin scripts/uninstall.sh   # explicit location

set -eu

BINARY="aws-tui"

info()  { printf '\033[1;34m==>\033[0m %s\n' "$1"; }
warn()  { printf '\033[1;33mwarning:\033[0m %s\n' "$1" >&2; }

# Candidate locations (the provided one first).
candidates="${BINDIR:-} /usr/local/bin $HOME/.local/bin $HOME/bin"

found=""
for dir in $candidates; do
	[ -n "$dir" ] || continue
	if [ -f "$dir/$BINARY" ]; then
		found="$dir/$BINARY"
		if [ -w "$dir" ]; then
			rm -f "$found"
		else
			info "Elevated privileges required for $dir (sudo)…"
			sudo rm -f "$found"
		fi
		info "Removed: $found"
	fi
done

if [ -z "$found" ]; then
	warn "No '$BINARY' binary found in: $candidates"
	exit 1
fi

info "Uninstall complete."
info "Note: your AWS configuration (~/.aws) and aws-tui state are left untouched."

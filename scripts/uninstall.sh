#!/usr/bin/env sh
#
# uninstall.sh — retire aws-tui d'un poste macOS/Linux.
#
# Cherche le binaire dans les emplacements d'installation usuels et le supprime.
# Usage :
#   scripts/uninstall.sh
#   BINDIR=~/.local/bin scripts/uninstall.sh   # emplacement explicite

set -eu

BINARY="aws-tui"

info()  { printf '\033[1;34m==>\033[0m %s\n' "$1"; }
warn()  { printf '\033[1;33mattention:\033[0m %s\n' "$1" >&2; }

# Emplacements candidats (celui fourni en premier).
candidates="${BINDIR:-} /usr/local/bin $HOME/.local/bin $HOME/bin"

found=""
for dir in $candidates; do
	[ -n "$dir" ] || continue
	if [ -f "$dir/$BINARY" ]; then
		found="$dir/$BINARY"
		if [ -w "$dir" ]; then
			rm -f "$found"
		else
			info "Droits élevés requis pour $dir (sudo)…"
			sudo rm -f "$found"
		fi
		info "Supprimé : $found"
	fi
done

if [ -z "$found" ]; then
	warn "Aucun binaire '$BINARY' trouvé dans : $candidates"
	exit 1
fi

info "Désinstallation terminée."
info "Note : la configuration AWS (~/.aws) et l'état d'aws-tui ne sont pas touchés."

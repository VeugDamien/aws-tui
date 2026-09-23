#!/usr/bin/env sh
#
# install.sh — installe aws-tui sur macOS ou Linux depuis les Releases GitHub.
#
# Usage (dernière version) :
#   curl -fsSL https://raw.githubusercontent.com/VeugDamien/aws-tui/main/scripts/install.sh | sh
#
# Options via variables d'environnement :
#   VERSION=v1.2.3   installe une version précise (défaut : dernière release)
#   BINDIR=~/.local/bin   répertoire d'installation (défaut : /usr/local/bin,
#                         ou ~/.local/bin sans droits root)
#
# POSIX sh, sans bashisme. Dépendances : curl ou wget, tar, uname.

set -eu

OWNER="VeugDamien"
REPO="aws-tui"
BINARY="aws-tui"

info()  { printf '\033[1;34m==>\033[0m %s\n' "$1"; }
warn()  { printf '\033[1;33mattention:\033[0m %s\n' "$1" >&2; }
error() { printf '\033[1;31merreur:\033[0m %s\n' "$1" >&2; exit 1; }

# --- Détection OS / architecture -------------------------------------------

os="$(uname -s)"
case "$os" in
	Darwin) os="darwin" ;;
	Linux)  os="linux" ;;
	*) error "OS non supporté : $os (ce script gère macOS et Linux ; sous Windows, utilisez install.ps1)." ;;
esac

arch="$(uname -m)"
case "$arch" in
	x86_64 | amd64) arch="amd64" ;;
	arm64 | aarch64) arch="arm64" ;;
	*) error "architecture non supportée : $arch" ;;
esac

# --- Outil de téléchargement ------------------------------------------------

if command -v curl >/dev/null 2>&1; then
	dl() { curl -fsSL "$1"; }
	dl_out() { curl -fsSL "$1" -o "$2"; }
elif command -v wget >/dev/null 2>&1; then
	dl() { wget -qO- "$1"; }
	dl_out() { wget -qO "$2" "$1"; }
else
	error "curl ou wget est requis."
fi

# --- Résolution de la version ----------------------------------------------

VERSION="${VERSION:-}"
if [ -z "$VERSION" ]; then
	info "Recherche de la dernière version…"
	api="https://api.github.com/repos/${OWNER}/${REPO}/releases/latest"
	# Extrait "tag_name": "vX.Y.Z" sans dépendre de jq.
	VERSION="$(dl "$api" | grep '"tag_name"' | head -n1 | sed -E 's/.*"tag_name" *: *"([^"]+)".*/\1/')"
	[ -n "$VERSION" ] || error "impossible de déterminer la dernière version (aucune release publiée ?)."
fi
# Numéro sans le préfixe 'v' pour le nom d'archive.
num="${VERSION#v}"

# --- Répertoire d'installation ---------------------------------------------

if [ -n "${BINDIR:-}" ]; then
	bindir="$BINDIR"
elif [ "$(id -u)" = "0" ]; then
	bindir="/usr/local/bin"
elif [ -w "/usr/local/bin" ] 2>/dev/null; then
	bindir="/usr/local/bin"
else
	bindir="$HOME/.local/bin"
fi

# --- Téléchargement + vérification -----------------------------------------

archive="${BINARY}_${num}_${os}_${arch}.tar.gz"
base="https://github.com/${OWNER}/${REPO}/releases/download/${VERSION}"
url="${base}/${archive}"

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

info "Téléchargement de ${archive} (${VERSION})…"
dl_out "$url" "$tmp/$archive" || error "téléchargement échoué : $url"

# Vérification du checksum si le fichier checksums.txt est disponible.
if dl_out "${base}/checksums.txt" "$tmp/checksums.txt" 2>/dev/null; then
	info "Vérification du checksum SHA-256…"
	expected="$(grep " .*${archive}\$" "$tmp/checksums.txt" | awk '{print $1}' | head -n1)"
	if [ -n "$expected" ]; then
		if command -v sha256sum >/dev/null 2>&1; then
			actual="$(sha256sum "$tmp/$archive" | awk '{print $1}')"
		elif command -v shasum >/dev/null 2>&1; then
			actual="$(shasum -a 256 "$tmp/$archive" | awk '{print $1}')"
		else
			actual=""
			warn "aucun outil sha256 ; vérification ignorée."
		fi
		if [ -n "$actual" ] && [ "$actual" != "$expected" ]; then
			error "checksum invalide (attendu $expected, obtenu $actual)."
		fi
	fi
else
	warn "checksums.txt indisponible ; vérification ignorée."
fi

# --- Extraction + installation ---------------------------------------------

info "Extraction…"
tar -xzf "$tmp/$archive" -C "$tmp"
# L'archive contient un dossier ${BINARY}_${num}_${os}_${arch}/${BINARY}.
src="$tmp/${BINARY}_${num}_${os}_${arch}/${BINARY}"
[ -f "$src" ] || src="$(find "$tmp" -type f -name "$BINARY" | head -n1)"
[ -f "$src" ] || error "binaire introuvable dans l'archive."

mkdir -p "$bindir"
if [ -w "$bindir" ]; then
	install -m 0755 "$src" "$bindir/$BINARY"
else
	info "Droits élevés requis pour écrire dans $bindir (sudo)…"
	sudo install -m 0755 "$src" "$bindir/$BINARY"
fi

info "Installé : $bindir/$BINARY"

# --- Vérifications post-installation ----------------------------------------

case ":$PATH:" in
	*":$bindir:"*) : ;;
	*) warn "$bindir n'est pas dans votre PATH. Ajoutez :  export PATH=\"$bindir:\$PATH\"" ;;
esac

command -v aws >/dev/null 2>&1 || warn "AWS CLI v2 ('aws') introuvable : requis pour les connexions et sessions SSM."
command -v session-manager-plugin >/dev/null 2>&1 || warn "session-manager-plugin introuvable : requis pour les sessions/port-forwards SSM."

"$bindir/$BINARY" --version 2>/dev/null || true
info "Terminé. Lancez : $BINARY"

#!/usr/bin/env bash
#
# build.sh — cross-compile aws-tui pour toutes les plateformes, crée les archives
# de distribution (.tar.gz pour Unix, .zip pour Windows) et un fichier de
# checksums SHA-256, le tout dans ./dist.
#
# Usage :
#   scripts/build.sh                 # version dérivée de git
#   VERSION=1.2.3 scripts/build.sh   # version explicite
#
# N'a besoin que de Go (et zip pour les archives Windows, sinon un .zip est
# produit via un fallback tar). Aucune dépendance externe.

set -euo pipefail

# Racine du projet (dossier parent de scripts/).
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

BINARY="aws-tui"
DIST="dist"

VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}"
COMMIT="${COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo none)}"
DATE="${DATE:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}"

LDFLAGS="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}"

# os/arch ciblés.
PLATFORMS=(
	"darwin/amd64"
	"darwin/arm64"
	"linux/amd64"
	"linux/arm64"
	"windows/amd64"
)

echo "aws-tui build ${VERSION} (commit ${COMMIT})"
rm -rf "$DIST"
mkdir -p "$DIST"

for platform in "${PLATFORMS[@]}"; do
	os="${platform%/*}"
	arch="${platform#*/}"
	ext=""
	[ "$os" = "windows" ] && ext=".exe"

	name="${BINARY}_${VERSION}_${os}_${arch}"
	workdir="${DIST}/${name}"
	mkdir -p "$workdir"

	echo "→ ${os}/${arch}"
	CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" \
		go build -trimpath -ldflags "$LDFLAGS" -o "${workdir}/${BINARY}${ext}" .

	# Documents inclus dans l'archive (best effort).
	for doc in README.md LICENSE; do
		[ -f "$doc" ] && cp "$doc" "$workdir/" || true
	done

	# Archive : .zip pour Windows, .tar.gz sinon.
	if [ "$os" = "windows" ]; then
		( cd "$DIST" && zip -qr "${name}.zip" "$name" )
	else
		( cd "$DIST" && tar -czf "${name}.tar.gz" "$name" )
	fi
	rm -rf "$workdir"
done

# Checksums SHA-256 de toutes les archives.
echo "→ checksums"
(
	cd "$DIST"
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum ./*.tar.gz ./*.zip 2>/dev/null > checksums.txt || true
	elif command -v shasum >/dev/null 2>&1; then
		shasum -a 256 ./*.tar.gz ./*.zip 2>/dev/null > checksums.txt || true
	fi
)

echo
echo "Artefacts dans ${DIST}/ :"
ls -1 "$DIST"

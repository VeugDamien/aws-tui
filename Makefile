# aws-tui — Makefile
#
# Cibles principales :
#   make build        compile le binaire local dans ./bin
#   make install      compile et installe dans $(PREFIX)/bin (défaut /usr/local)
#   make uninstall    retire le binaire installé
#   make cross        compile pour toutes les plateformes dans ./dist
#   make test         lance les tests
#   make vet          go vet
#   make check        vet + test
#   make clean        supprime bin/ et dist/
#   make version      affiche la version détectée

BINARY      := aws-tui
PKG         := github.com/VeugDamien/aws-tui
MODULE_MAIN := .

# Version dérivée de git (tag le plus proche), sinon "dev".
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

# Injection des métadonnées dans le paquet main.
LDFLAGS := -s -w \
	-X main.version=$(VERSION) \
	-X main.commit=$(COMMIT) \
	-X main.date=$(DATE)

# Répertoire d'installation (surchargable : make install PREFIX=$HOME/.local).
PREFIX  ?= /usr/local
BINDIR  := $(PREFIX)/bin

# Plateformes ciblées par `make cross` (format os/arch).
PLATFORMS := \
	darwin/amd64 \
	darwin/arm64 \
	linux/amd64 \
	linux/arm64 \
	windows/amd64

.DEFAULT_GOAL := build

.PHONY: build
build:
	@mkdir -p bin
	CGO_ENABLED=0 go build -trimpath -ldflags '$(LDFLAGS)' -o bin/$(BINARY) $(MODULE_MAIN)
	@echo "Compilé : bin/$(BINARY) ($(VERSION))"

.PHONY: install
install: build
	@install -d "$(BINDIR)"
	@install -m 0755 bin/$(BINARY) "$(BINDIR)/$(BINARY)"
	@echo "Installé : $(BINDIR)/$(BINARY)"

.PHONY: uninstall
uninstall:
	@rm -f "$(BINDIR)/$(BINARY)"
	@echo "Désinstallé : $(BINDIR)/$(BINARY)"

.PHONY: cross
cross:
	@mkdir -p dist
	@for platform in $(PLATFORMS); do \
		os=$${platform%/*}; arch=$${platform#*/}; \
		ext=""; [ "$$os" = "windows" ] && ext=".exe"; \
		out="dist/$(BINARY)_$${os}_$${arch}$${ext}"; \
		echo "→ $$out"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch \
			go build -trimpath -ldflags '$(LDFLAGS)' -o "$$out" $(MODULE_MAIN) || exit 1; \
	done
	@echo "Binaires cross-compilés dans dist/"

.PHONY: test
test:
	go test ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: check
check: vet test

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: clean
clean:
	rm -rf bin dist
	@echo "Nettoyé : bin/ dist/"

.PHONY: version
version:
	@echo "version=$(VERSION) commit=$(COMMIT) date=$(DATE)"

# Vérifie la config GoReleaser (nécessite goreleaser installé).
.PHONY: release-check
release-check:
	goreleaser check
	goreleaser release --snapshot --clean --skip=publish

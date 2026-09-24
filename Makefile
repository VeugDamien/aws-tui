# aws-tui — Makefile
#
# Main targets:
#   make build        build the local binary into ./bin
#   make install      build and install into $(PREFIX)/bin (default /usr/local)
#   make uninstall    remove the installed binary
#   make cross        build for all platforms into ./dist
#   make test         run the tests
#   make vet          go vet
#   make lint         golangci-lint (requires golangci-lint installed)
#   make check        vet + test
#   make clean        remove bin/ and dist/
#   make version      print the detected version

BINARY      := aws-tui
PKG         := github.com/VeugDamien/aws-tui
MODULE_MAIN := .

# Version derived from git (nearest tag), otherwise "dev".
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

# Inject metadata into the main package.
LDFLAGS := -s -w \
	-X main.version=$(VERSION) \
	-X main.commit=$(COMMIT) \
	-X main.date=$(DATE)

# Install directory (overridable: make install PREFIX=$HOME/.local).
PREFIX  ?= /usr/local
BINDIR  := $(PREFIX)/bin

# Platforms targeted by `make cross` (os/arch format).
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
	@echo "Built: bin/$(BINARY) ($(VERSION))"

.PHONY: install
install: build
	@install -d "$(BINDIR)"
	@install -m 0755 bin/$(BINARY) "$(BINDIR)/$(BINARY)"
	@echo "Installed: $(BINDIR)/$(BINARY)"

.PHONY: uninstall
uninstall:
	@rm -f "$(BINDIR)/$(BINARY)"
	@echo "Uninstalled: $(BINDIR)/$(BINARY)"

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
	@echo "Cross-compiled binaries in dist/"

.PHONY: test
test:
	go test ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: lint
lint:
	golangci-lint run

.PHONY: check
check: vet test

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: clean
clean:
	rm -rf bin dist
	@echo "Cleaned: bin/ dist/"

.PHONY: version
version:
	@echo "version=$(VERSION) commit=$(COMMIT) date=$(DATE)"

# Check the GoReleaser config (requires goreleaser installed).
.PHONY: release-check
release-check:
	goreleaser check
	goreleaser release --snapshot --clean --skip=publish

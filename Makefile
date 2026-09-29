.PHONY: build run test lint clean install bootstrap deps

BINARY    := hikari
MODULE    := github.com/NubiMa/hikari-cli
CMD       := ./cmd/hikari
VERSION   := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT    := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE      := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS   := -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

# Minimum Go version required
GO_MIN_VERSION := 1.24.2
# Go toolchain to install if none found (must match go.mod)
GO_INSTALL_VERSION := 1.24.2
GO_INSTALL_DIR     := $(HOME)/.local/go
GO_INSTALL_BIN     := $(GO_INSTALL_DIR)/bin/go

# ---------------------------------------------------------------------------
# bootstrap: install Go locally if it is absent or too old
# ---------------------------------------------------------------------------

bootstrap:
	@./scripts/bootstrap_go.sh

# deps: ensure Go is available (run bootstrap if needed), then tidy modules
deps: bootstrap
	@go mod download

# ---------------------------------------------------------------------------
# Build
# ---------------------------------------------------------------------------

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) $(CMD)

run:
	go run -ldflags "$(LDFLAGS)" $(CMD)

# ---------------------------------------------------------------------------
# Test / Lint
# ---------------------------------------------------------------------------

test:
	go test ./... -v -race -timeout 60s

test-short:
	go test ./... -short

lint:
	@which golangci-lint > /dev/null || (echo "Install golangci-lint: https://golangci-lint.run/usage/install/" && exit 1)
	golangci-lint run ./...

# ---------------------------------------------------------------------------
# Install / Clean
# ---------------------------------------------------------------------------

install: build
	install -m 755 $(BINARY) /usr/local/bin/$(BINARY)
	@echo "Installed to /usr/local/bin/$(BINARY)"

clean:
	rm -f $(BINARY)

.DEFAULT_GOAL := build

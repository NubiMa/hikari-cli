.PHONY: build run test lint clean install

BINARY    := hikari
MODULE    := github.com/nubiv/hikari
CMD       := ./cmd/hikari
VERSION   := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT    := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE      := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS   := -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) $(CMD)

run:
	go run -ldflags "$(LDFLAGS)" $(CMD)

test:
	go test ./... -v -race -timeout 60s

test-short:
	go test ./... -short

lint:
	@which golangci-lint > /dev/null || (echo "Install golangci-lint: https://golangci-lint.run/usage/install/" && exit 1)
	golangci-lint run ./...

clean:
	rm -f $(BINARY)

install: build
	install -m 755 $(BINARY) /usr/local/bin/$(BINARY)
	@echo "Installed to /usr/local/bin/$(BINARY)"

.DEFAULT_GOAL := build

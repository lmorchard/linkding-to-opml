.PHONY: build test clean run lint format fmt setup

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "v0.0.1")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

build:
	@echo "Building linkding-to-opml for $(shell go env GOOS)/$(shell go env GOARCH)"
	@if [ "$(shell go env GOOS)" = "linux" ]; then \
		echo "Using static linking for Linux build"; \
		go build -ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.Date=$(DATE) -linkmode external -extldflags '-static'" -o linkding-to-opml main.go; \
	else \
		go build -ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.Date=$(DATE)" -o linkding-to-opml main.go; \
	fi

test:
	go test -v -race -coverprofile=coverage.out ./...

clean:
	rm -f linkding-to-opml
	rm -f linkding-to-opml.exe
	rm -f linkding-to-opml.gob
	rm -f feeds.opml
	rm -f coverage.out

run: build
	./linkding-to-opml

format fmt:
	@GOPATH=$$(go env GOPATH); \
	if [ ! -f "$$GOPATH/bin/gofumpt" ]; then \
		echo "gofumpt not found. Please install it: go install mvdan.cc/gofumpt@latest"; \
		exit 1; \
	fi
	go fmt ./...
	$$(go env GOPATH)/bin/gofumpt -w .

lint:
	@GOPATH=$$(go env GOPATH); \
	if [ ! -f "$$GOPATH/bin/golangci-lint" ]; then \
		echo "golangci-lint not found. Please install it: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		exit 1; \
	fi
	$$(go env GOPATH)/bin/golangci-lint run --timeout=5m

setup:
	@echo "Installing development tools..."
	go install mvdan.cc/gofumpt@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Tools installed successfully!"
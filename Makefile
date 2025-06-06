# portctl Makefile

# Variables
BINARY_NAME=portctl
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-s -w -X dagger/portctl/cmd.Version=$(VERSION)"
BUILD_DIR=build

# Default target
.PHONY: all
all: build

# Build for current platform
.PHONY: build
build:
	go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/portctl

# Install dependencies
.PHONY: deps
deps:
	go mod download
	go mod tidy

# Run tests (TDD)
.PHONY: test
test:
	go test -v ./...

# Run Ginkgo tests (advanced TDD)
.PHONY: test-ginkgo
test-ginkgo:
	go install github.com/onsi/ginkgo/v2/ginkgo@latest
	ginkgo -r --randomize-all --fail-on-pending --cover

# Run BDD tests (Godog)
.PHONY: test-bdd
test-bdd:
	go install github.com/cucumber/godog/cmd/godog@latest
	godog run ./features

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Security scan (gosec)
.PHONY: sec
gosec:
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	gosec ./...

# Vulnerability scan (govulncheck)
.PHONY: vuln
govulncheck:
	go install golang.org/x/vuln/cmd/govulncheck@latest
	govulncheck ./...

# Build for all platforms
.PHONY: build-all
build-all: clean
	mkdir -p $(BUILD_DIR)
	# macOS
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/portctl
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/portctl
	# Linux
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/portctl
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/portctl
	# Windows
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/portctl
	GOOS=windows GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-arm64.exe ./cmd/portctl

# Install locally
.PHONY: install
install: build
	sudo mv $(BINARY_NAME) /usr/local/bin/

# Uninstall
.PHONY: uninstall
uninstall:
	sudo rm -f /usr/local/bin/$(BINARY_NAME)

# Clean build artifacts
.PHONY: clean
clean:
	rm -f $(BINARY_NAME)
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

# Lint code
.PHONY: lint
lint:
	golangci-lint run

# Format code
.PHONY: fmt
fmt:
	go fmt ./...

# Vet code
.PHONY: vet
vet:
	go vet ./...

# Run all quality checks
.PHONY: check
check: fmt vet lint test

# Development setup
.PHONY: dev-setup
dev-setup:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Create release archives
.PHONY: release
release: build-all
	cd $(BUILD_DIR) && \
	tar -czf $(BINARY_NAME)-darwin-amd64.tar.gz $(BINARY_NAME)-darwin-amd64 && \
	tar -czf $(BINARY_NAME)-darwin-arm64.tar.gz $(BINARY_NAME)-darwin-arm64 && \
	tar -czf $(BINARY_NAME)-linux-amd64.tar.gz $(BINARY_NAME)-linux-amd64 && \
	tar -czf $(BINARY_NAME)-linux-arm64.tar.gz $(BINARY_NAME)-linux-arm64 && \
	zip $(BINARY_NAME)-windows-amd64.zip $(BINARY_NAME)-windows-amd64.exe && \
	zip $(BINARY_NAME)-windows-arm64.zip $(BINARY_NAME)-windows-arm64.exe

# Quick demo
.PHONY: demo
demo: build
	@echo "🚀 portctl Demo"
	@echo "==============="
	@echo "1. Listing all processes with open ports:"
	@./$(BINARY_NAME) list | head -10
	@echo "\n2. Getting help:"
	@./$(BINARY_NAME) --help

# Help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build         - Build for current platform"
	@echo "  build-all     - Build for all platforms"
	@echo "  test          - Run TDD (Go test)"
	@echo "  test-ginkgo   - Run advanced TDD (Ginkgo)"
	@echo "  test-bdd      - Run BDD (Godog)"
	@echo "  test-coverage - Run tests with coverage"
	@echo "  sec           - Run gosec security scan"
	@echo "  vuln          - Run govulncheck vulnerability scan"
	@echo "  docs          - Generate API docs (godoc)"
	@echo "  install       - Install locally to /usr/local/bin"
	@echo "  uninstall     - Remove from /usr/local/bin"
	@echo "  clean         - Clean build artifacts"
	@echo "  lint          - Run linter"
	@echo "  fmt           - Format code"
	@echo "  vet           - Vet code"
	@echo "  check         - Run all quality checks"
	@echo "  dev-setup     - Install development tools"
	@echo "  release       - Create release archives"
	@echo "  demo          - Quick demonstration"
	@echo "  help          - Show this help"

# Generate API docs
.PHONY: docs
docs:
	go install golang.org/x/tools/cmd/godoc@latest
	godoc -html > docs/api/index.html

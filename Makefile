# portctl Makefile
# Wraps Dagger pipeline for consistent execution

# Variables
BINARY_NAME=portctl
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

# Default target
.PHONY: all
all: lint test build

# --- Dagger Wrappers ---

.PHONY: lint
lint:
	dagger call lint --src=.

.PHONY: test
test:
	dagger call test --src=.

.PHONY: build
build:
	dagger call build --src=.

.PHONY: sec
sec:
	dagger call security-scan --src=.

.PHONY: docs
docs:
	dagger call docs --src=.

.PHONY: release
release:
	dagger call release --src=. --github-token=env:GITHUB_TOKEN --tap-github-token=env:TAP_GITHUB_TOKEN

.PHONY: well-known
well-known:
	dagger call well-known --src=.

.PHONY: manifest
manifest:
	dagger call generate-manifest --src=.

.PHONY: sbom
sbom:
	dagger call sbom --src=.

# --- Legacy/Local Dev Helpers (Optional) ---

.PHONY: install
install:
	go install ./cmd/portctl

.PHONY: clean
clean:
	rm -f $(BINARY_NAME)
	rm -rf build
	rm -rf artifacts
	rm -f coverage.out

.PHONY: help
help:
	@echo "Available targets (via Dagger):"
	@echo "  lint          - Run linter"
	@echo "  test          - Run tests"
	@echo "  build         - Build binary"
	@echo "  sec           - Run security scan"
	@echo "  docs          - Generate documentation"
	@echo "  release       - Run release pipeline"
	@echo "  well-known    - Validate .well-known metadata"
	@echo "  manifest      - Generate MCP manifest"
	@echo "  sbom          - Generate SBOM"
	@echo ""
	@echo "Local helpers:"
	@echo "  install       - Go install locally"
	@echo "  clean         - Clean artifacts"

